// Package defaultprofile は、製品と同じ組込み profile set で semantic case を評価する。
package defaultprofile

import (
	"context"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Evaluator は、製品と同じ前処理器と default profile set を不変に保持する。
type Evaluator struct {
	planning       Planning
	boundaryPolicy requestBoundaryPolicy
}

type requestBoundaryPolicy uint8

const (
	requestBoundaryStrict requestBoundaryPolicy = iota + 1
	requestBoundaryScoredMismatch
)

// Planning は、標準または候補の固定 profile set を評価器へ渡す閉じた境界である。
type Planning interface {
	Preprocessor() legalquery.QueryPreprocessor
	Profiles() legalquery.QueryProfileSet
	ProfileMetadata() []legalquery.QueryProfileMetadata
}

// ScoresRequestBoundaryMismatch は、request 境界不一致を評価失敗へ変換するか返す。
func (e *Evaluator) ScoresRequestBoundaryMismatch() bool {
	return e != nil && e.boundaryPolicy == requestBoundaryScoredMismatch
}

// New は、repository 内の組込み成果物だけを使う evaluator を構築する。
func New() (*Evaluator, error) {
	planning, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	return NewWithPlanning(planning)
}

// NewV2 は、request 境界不一致を semantic 評価失敗へ変換する evaluator を構築する。
func NewV2() (*Evaluator, error) {
	planning, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	return NewWithPlanningV2(planning)
}

// NewWithPlanning は、CI 専用候補を含む明示的な固定 planning 依存を構成する。
func NewWithPlanning(planning Planning) (*Evaluator, error) {
	return newWithPlanning(planning, requestBoundaryStrict)
}

// NewWithPlanningV2 は、CI 専用候補の request 境界不一致を評価失敗へ
// 変換する evaluator を構築する。
func NewWithPlanningV2(planning Planning) (*Evaluator, error) {
	return newWithPlanning(planning, requestBoundaryScoredMismatch)
}

func newWithPlanning(
	planning Planning,
	policy requestBoundaryPolicy,
) (*Evaluator, error) {
	if planning == nil {
		return nil, fmt.Errorf("planning 依存は nil にできません")
	}
	if policy != requestBoundaryStrict &&
		policy != requestBoundaryScoredMismatch {
		return nil, fmt.Errorf("request boundary policy が定義されていません")
	}
	profiles := planning.Profiles()
	if err := profiles.Validate(); err != nil {
		return nil, fmt.Errorf("planning profile set が不正です: %w", err)
	}
	metadata := planning.ProfileMetadata()
	if len(metadata) == 0 {
		return nil, fmt.Errorf("planning profile metadata は一件以上必要です")
	}
	return &Evaluator{planning: planning, boundaryPolicy: policy}, nil
}

// Evaluate は、一件の semantic case を製品 request、前処理器および selector で評価する。
func (e *Evaluator) Evaluate(
	ctx context.Context,
	semanticCase legalquerycorpus.SemanticCase,
) (legalqueryeval.SemanticCaseEvaluation, error) {
	evaluation, _, _, err := e.EvaluateWithPlan(ctx, semanticCase)
	return evaluation, err
}

// EvaluateWithPlan は、plan case の評価値と再現性検査用 plan を同時に返す。
func (e *Evaluator) EvaluateWithPlan(
	ctx context.Context,
	semanticCase legalquerycorpus.SemanticCase,
) (
	legalqueryeval.SemanticCaseEvaluation,
	legalquery.LegalQueryPlan,
	bool,
	error,
) {
	if e == nil {
		return legalqueryeval.SemanticCaseEvaluation{},
			legalquery.LegalQueryPlan{},
			false,
			fmt.Errorf(
				"default profile evaluator は nil にできません",
			)
	}
	if ctx == nil {
		return legalqueryeval.SemanticCaseEvaluation{},
			legalquery.LegalQueryPlan{},
			false,
			fmt.Errorf(
				"context は nil にできません",
			)
	}
	if err := semanticCase.Validate(); err != nil {
		return legalqueryeval.SemanticCaseEvaluation{},
			legalquery.LegalQueryPlan{},
			false,
			fmt.Errorf(
				"semantic case が有効ではありません: %w",
				err,
			)
	}

	request, err := productRequest(semanticCase.Request())
	if err != nil {
		evaluation, evaluationErr := evaluateRequestError(
			semanticCase,
			err,
			e.boundaryPolicy,
		)
		return evaluation, legalquery.LegalQueryPlan{}, false, evaluationErr
	}
	if semanticCase.Expected().Kind() ==
		legalquerycorpus.SemanticExpectedKindRequestError {
		if e.boundaryPolicy == requestBoundaryStrict {
			_, boundaryErr := evaluateAcceptedRequestError(
				semanticCase,
				e.boundaryPolicy,
			)
			return legalqueryeval.SemanticCaseEvaluation{},
				legalquery.LegalQueryPlan{},
				false,
				boundaryErr
		}
		plan, selectErr := e.selectPlan(ctx, semanticCase, request)
		if selectErr != nil {
			return legalqueryeval.SemanticCaseEvaluation{},
				legalquery.LegalQueryPlan{}, false, selectErr
		}
		evaluation, evaluationErr := evaluateAcceptedRequestError(
			semanticCase,
			e.boundaryPolicy,
		)
		if evaluationErr != nil {
			return legalqueryeval.SemanticCaseEvaluation{},
				legalquery.LegalQueryPlan{}, false, evaluationErr
		}
		return evaluation, plan, true, nil
	}

	plan, err := e.selectPlan(ctx, semanticCase, request)
	if err != nil {
		return legalqueryeval.SemanticCaseEvaluation{},
			legalquery.LegalQueryPlan{},
			false,
			err
	}
	evaluation, err := legalqueryeval.EvaluateSemanticPlanCase(
		semanticCase,
		plan,
	)
	if err != nil {
		return legalqueryeval.SemanticCaseEvaluation{},
			legalquery.LegalQueryPlan{},
			false,
			err
	}
	return evaluation, plan, true, nil
}

func evaluateAcceptedRequestError(
	semanticCase legalquerycorpus.SemanticCase,
	policy requestBoundaryPolicy,
) (legalqueryeval.SemanticCaseEvaluation, error) {
	if policy == requestBoundaryStrict {
		return legalqueryeval.SemanticCaseEvaluation{}, fmt.Errorf(
			"request_error を期待する入力が製品 request に受理されました",
		)
	}
	return legalqueryeval.EvaluateSemanticAcceptedRequestErrorCaseV2(
		semanticCase,
	)
}

func (e *Evaluator) selectPlan(
	ctx context.Context,
	semanticCase legalquerycorpus.SemanticCase,
	request legalquery.Request,
) (legalquery.LegalQueryPlan, error) {
	preprocessed, err := e.planning.Preprocessor().Preprocess(ctx, request)
	if err != nil {
		return legalquery.LegalQueryPlan{}, fmt.Errorf(
			"統合照会の前処理に失敗しました: %w",
			err,
		)
	}
	profileSetResult, err := e.planning.Profiles().Collect(preprocessed)
	if err != nil {
		return legalquery.LegalQueryPlan{}, fmt.Errorf(
			"default profile の候補生成に失敗しました: %w",
			err,
		)
	}
	packState, err := legalquery.NewStaticPackState(
		[]string{legalqueryplanning.JudicialCasesPackID},
		semanticCase.EnabledPacks(),
	)
	if err != nil {
		return legalquery.LegalQueryPlan{}, fmt.Errorf(
			"semantic case の pack 状態が有効ではありません: %w",
			err,
		)
	}
	plan, err := legalquery.SelectLegalQueryPlan(legalquery.SelectorInput{
		ProfileSetResult: profileSetResult,
		PackState:        packState,
		LimitPerAttempt:  request.LimitPerAttempt(),
	})
	if err != nil {
		return legalquery.LegalQueryPlan{}, fmt.Errorf(
			"default profile の plan を選択できません: %w",
			err,
		)
	}
	return plan, nil
}

func evaluateRequestError(
	semanticCase legalquerycorpus.SemanticCase,
	requestErr error,
	policy requestBoundaryPolicy,
) (legalqueryeval.SemanticCaseEvaluation, error) {
	var argumentError legalquery.ArgumentError
	if !errors.As(requestErr, &argumentError) {
		return legalqueryeval.SemanticCaseEvaluation{}, fmt.Errorf(
			"製品 request を構築できません: %w",
			requestErr,
		)
	}
	if semanticCase.Expected().Kind() !=
		legalquerycorpus.SemanticExpectedKindRequestError {
		if policy == requestBoundaryScoredMismatch {
			return legalqueryeval.EvaluateSemanticPlanArgumentErrorCaseV2(
				semanticCase,
				argumentError,
			)
		}
		return legalqueryeval.SemanticCaseEvaluation{}, fmt.Errorf(
			"plan を期待する入力が request error になりました: %w",
			requestErr,
		)
	}
	return legalqueryeval.EvaluateSemanticRequestErrorCase(
		semanticCase,
		argumentError,
	)
}

func productRequest(
	raw legalquerycorpus.Request,
) (legalquery.Request, error) {
	values := legalquery.RequestValues{Query: raw.Query()}
	if limit, exists := raw.LimitPerAttempt(); exists {
		values.LimitPerAttempt = &limit
	}
	base, err := legalquery.NewRequest(values)
	if err != nil {
		return legalquery.Request{}, err
	}
	rawRef, exists := raw.Ref()
	if !exists {
		return base, nil
	}
	ref, err := productRef(rawRef)
	if err != nil {
		return legalquery.Request{}, err
	}
	values.Ref = &ref
	return legalquery.NewRequest(values)
}

func productRef(
	raw legalquerycorpus.RequestRef,
) (model.SourceResourceRef, error) {
	keyValues := model.SourceResourceKeyValues{
		SourceID:     raw.Key().SourceID(),
		ResourceType: raw.Key().ResourceType(),
		ResourceID:   raw.Key().ResourceID(),
	}
	if versionID, exists := raw.Key().VersionID(); exists {
		keyValues.VersionID = versionID
	}
	key, err := model.NewSourceResourceKey(keyValues)
	if err != nil {
		return model.SourceResourceRef{}, invalidRefError()
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: raw.ProviderID(),
		Key:        key,
	})
	if err != nil {
		return model.SourceResourceRef{}, invalidRefError()
	}
	return ref, nil
}

func invalidRefError() error {
	argumentError, err := legalquery.NewArgumentError(
		"ref",
		"は有効な SourceResourceRef でなければなりません",
	)
	if err != nil {
		return fmt.Errorf("ref の入力エラーを構築できません: %w", err)
	}
	return argumentError
}
