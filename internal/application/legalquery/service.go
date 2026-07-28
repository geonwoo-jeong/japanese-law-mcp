package legalquery

import (
	"context"
	"fmt"
	"time"
)

// Service は、一回の統合法情報照会を固定期限の単方向パイプラインとして調整する。
type Service struct {
	preprocessor   QueryPreprocessor
	profiles       QueryProfileSet
	packState      PackState
	executor       Executor
	requestTimeout time.Duration
	initialized    bool
}

// NewService は、前処理、profile、pack 状態、executor および一要求の期限を固定する。
func NewService(
	preprocessor QueryPreprocessor,
	profiles QueryProfileSet,
	packState PackState,
	executor Executor,
	requestTimeout time.Duration,
) (*Service, error) {
	service := &Service{
		preprocessor:   preprocessor,
		profiles:       profiles,
		packState:      packState,
		executor:       executor,
		requestTimeout: requestTimeout,
		initialized:    true,
	}
	if err := service.Validate(); err != nil {
		return nil, err
	}
	return service, nil
}

// Validate は、起動時に固定した依存と request timeout を再検証する。
func (s *Service) Validate() error {
	if s == nil {
		return fmt.Errorf("Service は必須です")
	}
	if !s.initialized {
		return fmt.Errorf("Service は NewService で作成しなければなりません")
	}
	if isNilInterfaceValue(s.preprocessor) {
		return fmt.Errorf("query preprocessor は必須です")
	}
	if err := s.profiles.Validate(); err != nil {
		return fmt.Errorf("query profile set が有効ではありません: %w", err)
	}
	if isNilInterfaceValue(s.packState) {
		return fmt.Errorf("pack state は必須です")
	}
	if err := s.executor.Validate(); err != nil {
		return fmt.Errorf("executor が有効ではありません: %w", err)
	}
	if s.requestTimeout <= 0 {
		return fmt.Errorf("request timeout は正でなければなりません")
	}
	return nil
}

// Query は、検証済み request を計画し、必要な場合だけ確定済み plan を実行する。
func (s *Service) Query(
	ctx context.Context,
	request Request,
) (LegalQueryResult, error) {
	if err := s.validateQueryInput(ctx, request); err != nil {
		return nil, err
	}
	requestContext, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	plan, err := s.plan(requestContext, request)
	if err != nil {
		return nil, err
	}
	result, err := s.resultForPlan(requestContext, plan)
	if err != nil {
		return nil, err
	}
	if err := requestContext.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) validateQueryInput(
	ctx context.Context,
	request Request,
) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if isNilInterfaceValue(ctx) {
		return fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := request.Validate(); err != nil {
		return fmt.Errorf("request が有効ではありません: %w", err)
	}
	return nil
}

func (s *Service) plan(
	ctx context.Context,
	request Request,
) (LegalQueryPlan, error) {
	preprocessed, err := s.preprocessor.Preprocess(ctx, request)
	if err != nil {
		return LegalQueryPlan{}, contextualStageError(
			ctx,
			err,
			"照会文を前処理できません",
		)
	}
	if err := ctx.Err(); err != nil {
		return LegalQueryPlan{}, err
	}
	if err := validatePreprocessedRequest(request, preprocessed); err != nil {
		return LegalQueryPlan{}, err
	}
	profileResult, err := s.profiles.Collect(preprocessed)
	if err != nil {
		return LegalQueryPlan{}, contextualStageError(
			ctx,
			err,
			"意味候補を生成できません",
		)
	}
	if err := ctx.Err(); err != nil {
		return LegalQueryPlan{}, err
	}
	plan, err := SelectLegalQueryPlan(SelectorInput{
		ProfileSetResult: profileResult,
		PackState:        s.packState,
		LimitPerAttempt:  request.LimitPerAttempt(),
	})
	if err != nil {
		return LegalQueryPlan{}, contextualStageError(
			ctx,
			err,
			"統合照会 plan を確定できません",
		)
	}
	if err := ctx.Err(); err != nil {
		return LegalQueryPlan{}, err
	}
	return plan, nil
}

func (s *Service) resultForPlan(
	ctx context.Context,
	plan LegalQueryPlan,
) (LegalQueryResult, error) {
	switch plan.Decision() {
	case PlanDecisionNeedsClarification,
		PlanDecisionCapabilityUnavailable,
		PlanDecisionUnsupported:
		result, err := AssembleLegalQueryNonExecutionResult(plan)
		if err != nil {
			return nil, contextualStageError(
				ctx,
				err,
				"非実行結果を組み立てられません",
			)
		}
		return result, nil
	case PlanDecisionSingle, PlanDecisionHedged:
		execution, err := s.executor.Execute(ctx, plan)
		if err != nil {
			return nil, contextualStageError(
				ctx,
				err,
				"統合照会 plan を実行できません",
			)
		}
		result, err := AssembleLegalQueryExecutionResult(plan, execution)
		if err != nil {
			return nil, contextualStageError(
				ctx,
				err,
				"実行結果を組み立てられません",
			)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("plan decision が定義されていません")
	}
}

func validatePreprocessedRequest(
	request Request,
	preprocessed PreprocessResult,
) error {
	if err := preprocessed.Validate(); err != nil {
		return fmt.Errorf("前処理結果が有効ではありません: %w", err)
	}
	if preprocessed.Query() != request.Query() {
		return fmt.Errorf("前処理結果の query が request と一致しません")
	}
	requestRef, requestHasRef := request.Ref()
	resultRef, resultHasRef := preprocessed.Ref()
	if requestHasRef != resultHasRef ||
		requestHasRef && requestRef != resultRef {
		return fmt.Errorf("前処理結果の ref が request と一致しません")
	}
	return nil
}

func contextualStageError(
	ctx context.Context,
	stageError error,
	message string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%s: %w", message, stageError)
}
