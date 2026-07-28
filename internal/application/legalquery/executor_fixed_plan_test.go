package legalquery

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
)

func TestExecutorRunsOnlySelectedStepsWithoutImplicitRead(t *testing.T) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	selectedCandidate := mustExecutorCandidate(
		t,
		"candidate-1",
		600,
		mustExecutorStep(t, "法令検索", "step-search"),
	)
	unselectedCandidate := mustExecutorCandidate(
		t,
		"candidate-2",
		590,
		mustExecutorStep(t, "法令読取り", "step-read"),
	)
	selection, err := NewLegalQueryPlanSelection(
		LegalQueryPlanSelectionValues{
			CandidateID:   selectedCandidate.CandidateID(),
			Availability:  SelectionAvailabilityAvailable,
			RequiredPacks: selectedCandidate.RequiredPacks(),
		},
	)
	if err != nil {
		t.Fatalf("試験用 selection を作成できません: %v", err)
	}
	plan, err := NewLegalQueryPlan(LegalQueryPlanValues{
		ProfileVersion: "executor-fixed-plan-v1",
		Decision:       PlanDecisionSingle,
		RankedCandidates: []LegalQueryCandidate{
			selectedCandidate,
			unselectedCandidate,
		},
		Selected:        []LegalQueryPlanSelection{selection},
		ReasonCodes:     []ReasonCode{ReasonCodeSingleClearCandidate},
		LimitPerAttempt: 10,
	})
	if err != nil {
		t.Fatalf("試験用 plan を作成できません: %v", err)
	}

	execution, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute に失敗しました: %v", err)
	}
	attempts := execution.Attempts()
	if len(attempts) != 1 || attempts[0].StepID() != "step-search" {
		t.Fatalf("SOT-ARCH-023: plan 外の attempt を返しました: %#v", attempts)
	}
	calls := recorder.callsSnapshot()
	if len(calls) != 1 || calls[0].kind != InputKindLawSearch {
		t.Fatalf("SOT-ARCH-023: plan 外または暗黙 read を実行しました: %#v", calls)
	}
}

func TestExecutorDoesNotRedistributeBudgetAfterEmptyAttempt(t *testing.T) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	emptyPage, err := lawsearch.NewPage(lawsearch.PageValues{
		Items: nil,
		Page:  mustExecutorTestSourcePage(t, 0),
	})
	if err != nil {
		t.Fatalf("試験用 empty page を作成できません: %v", err)
	}
	core.payloads.lawSearch = emptyPage
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	plan := mustExecutorSinglePlan(
		t,
		mustExecutorStep(t, "法令検索", "step-search"),
		mustExecutorStep(t, "更新一覧", "step-updates"),
	)

	execution, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute に失敗しました: %v", err)
	}
	attempts := execution.Attempts()
	if len(attempts) != 2 ||
		attempts[0].Outcome() != LegalQueryAttemptOutcomeEmpty ||
		attempts[1].Outcome() != LegalQueryAttemptOutcomeCompleted {
		t.Fatalf("空結果後の attempts = %#v", attempts)
	}
	calls := recorder.callsSnapshot()
	if len(calls) != 2 {
		t.Fatalf("call count = %d", len(calls))
	}
	for index, call := range calls {
		limit, exists := call.budget.EffectiveLimit()
		if !exists || limit != 10 {
			t.Fatalf(
				"SOT-ARCH-023: calls[%d] の固定予算 = (%d, %t)",
				index,
				limit,
				exists,
			)
		}
	}
}
