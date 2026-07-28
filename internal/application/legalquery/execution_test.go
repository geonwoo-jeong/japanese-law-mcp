package legalquery

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLegalQueryExecutionIsImmutable(t *testing.T) {
	t.Parallel()

	step := mustExecutorStep(t, "法令検索", "step-search")
	payloads := newExecutorTestPayloads(t)
	attempt, err := NewLegalQueryLawSearchAttempt(
		LegalQueryLawSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             step,
			Page:             resultTestUnknownPage(t, 1),
			Items:            payloads.lawSearch.Items(),
		},
	)
	if err != nil {
		t.Fatalf("試験用 attempt を作成できません: %v", err)
	}
	if _, err := NewLegalQueryExecution(LegalQueryExecutionValues{
		Attempts: []LegalQueryAttempt{attempt},
	}); err == nil {
		t.Fatal("SOT-MODEL-024: plan のない execution を受理しました")
	}
	plan := mustExecutorSinglePlan(t, step)
	values := []LegalQueryAttempt{attempt}
	execution, err := NewLegalQueryExecution(
		LegalQueryExecutionValues{Plan: plan, Attempts: values},
	)
	if err != nil {
		t.Fatalf("LegalQueryExecution を作成できません: %v", err)
	}

	failed, err := NewLegalQueryFailedAttempt(LegalQueryFailedAttemptValues{
		InterpretationID: "interpretation-1",
		Step:             step,
		Error:            resultTestError(t, model.ErrorCodeInternalError),
	})
	if err != nil {
		t.Fatalf("試験用 failed attempt を作成できません: %v", err)
	}
	values[0] = failed
	first := execution.Attempts()
	first[0] = failed
	second := execution.Attempts()
	if len(second) != 1 ||
		second[0].Outcome() != LegalQueryAttemptOutcomeCompleted {
		t.Fatal("SOT-ENG-025: execution の attempts が外部変更されました")
	}
	if err := execution.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-024: execution が有効ではありません: %v", err)
	}
}

func TestLegalQueryExecutionFailsClosed(t *testing.T) {
	t.Parallel()

	var zero LegalQueryExecution
	if err := zero.Validate(); err == nil {
		t.Fatal("SOT-ENG-025: zero-value execution を有効と判定しました")
	}
	step := mustExecutorStep(t, "法令検索", "step-search")
	plan := mustExecutorSinglePlan(t, step)
	if _, err := NewLegalQueryExecution(
		LegalQueryExecutionValues{
			Plan:     plan,
			Attempts: []LegalQueryAttempt{},
		},
	); err == nil {
		t.Fatal("SOT-MODEL-024: 空の execution を受理しました")
	}

	var typedNil *LegalQueryLawSearchAttempt
	if _, err := NewLegalQueryExecution(LegalQueryExecutionValues{
		Plan:     plan,
		Attempts: []LegalQueryAttempt{typedNil},
	}); err == nil {
		t.Fatal("SOT-ENG-025: typed nil attempt を受理しました")
	}

	failed, err := NewLegalQueryFailedAttempt(LegalQueryFailedAttemptValues{
		InterpretationID: "interpretation-1",
		Step:             step,
		Error:            resultTestError(t, model.ErrorCodeInternalError),
	})
	if err != nil {
		t.Fatalf("試験用 failed attempt を作成できません: %v", err)
	}
	if _, err := NewLegalQueryExecution(LegalQueryExecutionValues{
		Plan:     plan,
		Attempts: []LegalQueryAttempt{failed},
	}); err == nil {
		t.Fatal("SOT-MODEL-024: 全失敗を execution artifact として受理しました")
	}
}

func TestLegalQueryExecutionRejectsReverseStepOrderWithinInterpretation(
	t *testing.T,
) {
	t.Parallel()

	firstStep := mustExecutorStep(t, "法令検索", "step-first")
	secondStep := mustExecutorStep(t, "法令本文検索", "step-second")
	plan := mustExecutorSinglePlan(t, firstStep, secondStep)
	if err := plan.Validate(); err != nil {
		t.Fatalf("試験用 plan が有効ではありません: %v", err)
	}
	payloads := newExecutorTestPayloads(t)
	firstAttempt, err := NewLegalQueryLawSearchAttempt(
		LegalQueryLawSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             firstStep,
			Page:             resultTestUnknownPage(t, 1),
			Items:            payloads.lawSearch.Items(),
		},
	)
	if err != nil {
		t.Fatalf("第一 attempt を作成できません: %v", err)
	}
	secondAttempt, err := NewLegalQueryLawContentSearchAttempt(
		LegalQueryLawContentSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             secondStep,
			Page:             resultTestUnknownPage(t, 1),
			Items:            payloads.lawContent.Items(),
		},
	)
	if err != nil {
		t.Fatalf("第二 attempt を作成できません: %v", err)
	}

	if _, err := NewLegalQueryExecution(LegalQueryExecutionValues{
		Plan:     plan,
		Attempts: []LegalQueryAttempt{secondAttempt, firstAttempt},
	}); err == nil {
		t.Fatal("SOT-MODEL-024: 同一 interpretation 内の逆順 attempt を受理しました")
	}
}
