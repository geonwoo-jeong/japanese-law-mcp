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
	planning legalqueryplanning.Dependencies
}

// New は、repository 内の組込み成果物だけを使う evaluator を構築する。
func New() (*Evaluator, error) {
	planning, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	return &Evaluator{planning: planning}, nil
}

// Evaluate は、一件の semantic case を製品 request、前処理器および selector で評価する。
func (e *Evaluator) Evaluate(
	ctx context.Context,
	semanticCase legalquerycorpus.SemanticCase,
) (legalqueryeval.SemanticCaseEvaluation, error) {
	if e == nil {
		return legalqueryeval.SemanticCaseEvaluation{}, fmt.Errorf(
			"default profile evaluator は nil にできません",
		)
	}
	if ctx == nil {
		return legalqueryeval.SemanticCaseEvaluation{}, fmt.Errorf(
			"context は nil にできません",
		)
	}
	if err := semanticCase.Validate(); err != nil {
		return legalqueryeval.SemanticCaseEvaluation{}, fmt.Errorf(
			"semantic case が有効ではありません: %w",
			err,
		)
	}

	request, err := productRequest(semanticCase.Request())
	if err != nil {
		return evaluateRequestError(semanticCase, err)
	}
	if semanticCase.Expected().Kind() ==
		legalquerycorpus.SemanticExpectedKindRequestError {
		return legalqueryeval.SemanticCaseEvaluation{}, fmt.Errorf(
			"request_error を期待する入力が製品 request に受理されました",
		)
	}

	plan, err := e.selectPlan(ctx, semanticCase, request)
	if err != nil {
		return legalqueryeval.SemanticCaseEvaluation{}, err
	}
	return legalqueryeval.EvaluateSemanticPlanCase(semanticCase, plan)
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
