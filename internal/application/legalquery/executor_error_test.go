package legalquery

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestExecutorReturnsPartialExecutionForExecutedFailure(t *testing.T) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	sensitiveCause := errors.New("raw provider detail token=secret")
	recorder.errs[InputKindLawSearch] = mustExecutorExecutedError(
		t,
		sensitiveCause,
	)
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	plan := mustExecutorSinglePlan(
		t,
		mustExecutorStep(t, "法令検索", "step-failed"),
		mustExecutorStep(t, "法令本文検索", "step-success"),
	)

	execution, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("SOT-ARCH-023: partial Execute error = %v", err)
	}
	attempts := execution.Attempts()
	if len(attempts) != 2 ||
		attempts[0].Outcome() != LegalQueryAttemptOutcomeFailed ||
		attempts[1].Outcome() != LegalQueryAttemptOutcomeCompleted {
		t.Fatalf("SOT-MODEL-024: partial attempts = %#v", attempts)
	}
	failed, ok := attempts[0].(LegalQueryFailedAttempt)
	if !ok || failed.Error().Code() != model.ErrorCodeInternalError {
		t.Fatalf("SOT-IF-051: unknown port error の分類 = %#v", attempts[0])
	}
	if strings.Contains(failed.Error().Message(), sensitiveCause.Error()) {
		t.Fatal("SOT-IF-051: 公開 message に元の provider error を含めました")
	}
	if recorder.callCount() != 2 {
		t.Fatalf("SOT-ARCH-023: 部分失敗後の call count = %d", recorder.callCount())
	}
}

func TestExecutorReturnsPlanFirstPublicErrorWhenAllStepsFail(
	t *testing.T,
) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	recorder.errs[InputKindLawRead] = mustExecutorExecutedError(
		t,
		lawdocumentread.ErrNotFound,
	)
	recorder.errs[InputKindLawArticleRead] = mustExecutorExecutedError(
		t,
		lawarticleread.ErrAmbiguousLocation,
	)
	secondCompleted := make(chan struct{})
	recorder.hook = func(
		_ context.Context,
		kind LogicalInputKind,
		_ LegalQueryStepBudget,
	) error {
		if kind == InputKindLawRead {
			<-secondCompleted
		}
		if kind == InputKindLawArticleRead {
			close(secondCompleted)
		}
		return nil
	}
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	plan := mustExecutorHedgedPlan(
		t,
		[]LegalQueryCandidateStep{
			mustExecutorStep(t, "法令読取り", "step-plan-first"),
		},
		[]LegalQueryCandidateStep{
			mustExecutorStep(t, "条文読取り", "step-completed-first"),
		},
	)

	execution, err := executor.Execute(context.Background(), plan)
	if err == nil {
		t.Fatal("SOT-MODEL-024: 全 step 失敗を成功 execution として返しました")
	}
	if execution.Validate() == nil {
		t.Fatal("SOT-MODEL-024: 全失敗時に execution artifact を返しました")
	}
	var allFailed LegalQueryAllFailedError
	if !errors.As(err, &allFailed) {
		t.Fatalf("SOT-IF-051: typed all-failed error ではありません: %v", err)
	}
	if allFailed.ErrorResult().Code() != model.ErrorCodeNotFound {
		t.Fatalf(
			"SOT-IF-051: plan-first error = %q",
			allFailed.ErrorResult().Code(),
		)
	}
	if recorder.callCount() != 2 {
		t.Fatalf("SOT-ARCH-023: all-failed call count = %d", recorder.callCount())
	}
}

func TestExecutorKeepsPrePortFailureFatal(t *testing.T) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	cause := errors.New("request materialization contract failure")
	recorder.errs[InputKindLawSearch] = cause
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	plan := mustExecutorSinglePlan(
		t,
		mustExecutorStep(t, "法令検索", "step-fatal"),
		mustExecutorStep(t, "法令本文検索", "step-must-not-run"),
	)

	execution, err := executor.Execute(context.Background(), plan)
	if err == nil || !errors.Is(err, cause) {
		t.Fatalf("SOT-ARCH-026: pre-port cause を保持しませんでした: %v", err)
	}
	var allFailed LegalQueryAllFailedError
	if errors.As(err, &allFailed) {
		t.Fatal("SOT-ARCH-026: pre-port error を failed attempt に読み替えました")
	}
	if execution.Validate() == nil {
		t.Fatal("SOT-ARCH-026: fatal error と execution を同時に返しました")
	}
	if recorder.callCount() != 1 {
		t.Fatalf(
			"SOT-ARCH-026: fatal error 後の call count = %d",
			recorder.callCount(),
		)
	}
}

func TestExecutorMapsExecutedErrorsToSafePublicResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		step              LegalQueryCandidateStep
		cause             error
		code              model.ErrorCode
		assertNoCauseText bool
	}{
		{
			name:  "ambiguous location",
			step:  mustExecutorStep(t, "条文読取り", "step-ambiguous"),
			cause: lawarticleread.ErrAmbiguousLocation,
			code:  model.ErrorCodeAmbiguousLocation,
		},
		{
			name: "source error",
			step: mustExecutorStep(t, "法令検索", "step-source"),
			cause: mustExecutorSourceError(
				t,
				model.SourceErrorCodeSourceTimeout,
			),
			code: model.ErrorCodeSourceTimeout,
		},
		{
			name:  "execution timeout",
			step:  mustExecutorStep(t, "法令検索", "step-execution-timeout"),
			cause: context.DeadlineExceeded,
			code:  model.ErrorCodeSourceTimeout,
		},
		{
			name: "実行後の unsupported capability",
			step: mustExecutorStep(t, "法令検索", "step-unsupported-capability"),
			cause: mustExecutorSourceError(
				t,
				model.SourceErrorCodeUnsupportedCapability,
			),
			code: model.ErrorCodeInternalError,
		},
		{
			name: "実行後の configuration required",
			step: mustExecutorStep(t, "法令検索", "step-configuration-required"),
			cause: mustExecutorSourceError(
				t,
				model.SourceErrorCodeConfigurationRequired,
			),
			code: model.ErrorCodeInternalError,
		},
		{
			name:              "unknown actual port error",
			step:              mustExecutorStep(t, "法令検索", "step-internal"),
			cause:             errors.New("provider response contains secret=abc"),
			code:              model.ErrorCodeInternalError,
			assertNoCauseText: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			recorder, core, judicial := newExecutorTestFacades(t)
			recorder.errs[test.step.InputKind()] = mustExecutorExecutedError(
				t,
				test.cause,
			)
			executor, err := NewExecutor(core, judicial)
			if err != nil {
				t.Fatalf("Executor を作成できません: %v", err)
			}
			_, err = executor.Execute(
				context.Background(),
				mustExecutorSinglePlan(t, test.step),
			)
			var allFailed LegalQueryAllFailedError
			if !errors.As(err, &allFailed) {
				t.Fatalf("typed all-failed error ではありません: %v", err)
			}
			public := allFailed.ErrorResult()
			if public.Code() != test.code {
				t.Fatalf(
					"SOT-IF-051: public code = %q, want %q",
					public.Code(),
					test.code,
				)
			}
			if test.assertNoCauseText &&
				strings.Contains(public.Message(), test.cause.Error()) {
				t.Fatal("SOT-IF-051: 公開 message に元の cause を含めました")
			}
		})
	}
}

func mustExecutorExecutedError(t *testing.T, cause error) error {
	t.Helper()
	result, err := NewExecutedStepError(cause)
	if err != nil {
		t.Fatalf("試験用 ExecutedStepError を作成できません: %v", err)
	}
	return result
}

type executorTestSourceOperation string

const executorTestSearchOperation executorTestSourceOperation = "search"

func (executorTestSourceOperation) SourceOperationProviderID() string {
	return "executor-provider"
}

func (o executorTestSourceOperation) SourceOperationName() string {
	return string(o)
}

func (o executorTestSourceOperation) ValidateSourceOperation() error {
	if o != executorTestSearchOperation {
		return errors.New("operation が定義されていません")
	}
	return nil
}

func mustExecutorSourceError(
	t *testing.T,
	code model.SourceErrorCode,
) model.SourceError {
	t.Helper()
	const providerID = "executor-provider"
	capability, err := model.NewProviderCapability(
		model.ProviderCapabilityValues{
			ID:           lawsearch.CapabilityID,
			MajorVersion: lawsearch.MajorVersion,
			Level:        model.CapabilityLevelCore,
			Stability:    model.CapabilityStabilityStable,
		},
	)
	if err != nil {
		t.Fatalf("試験用 ProviderCapability を作成できません: %v", err)
	}
	capabilities := []model.ProviderCapability{capability}
	if code == model.SourceErrorCodeUnsupportedCapability {
		capabilities = []model.ProviderCapability{}
	}
	provider, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID: providerID,
		Source: resultTestInformationSource(
			t,
			"executor-source",
			"https://example.go.jp/",
		),
		AdapterContractVersion: "1.0.0",
		UpstreamSpecVersion:    "2026-07-28",
		VerifiedAt:             mustDate(t, "2026-07-28"),
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           capabilities,
	})
	if err != nil {
		t.Fatalf("試験用 ProviderDescriptor を作成できません: %v", err)
	}
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   provider,
		Capability: capability,
		Operation:  executorTestSearchOperation,
	})
	if err != nil {
		t.Fatalf("試験用 SourceError を作成できません: %v", err)
	}
	return sourceError
}
