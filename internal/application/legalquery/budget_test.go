package legalquery

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestLegalQueryBudgetKeepsFixedSOTLimits(t *testing.T) {
	t.Parallel()

	plan := budgetTestExecutingPlan(t, 0, 1, 20)
	budget := plan.Budget()
	if MaxRankedCandidates != 16 ||
		MaxSelectedCandidates != 2 ||
		MaxParallelCandidates != 2 ||
		MaxCapabilityCalls != 4 ||
		MaxItemsPerCollectionStep != 20 ||
		MaxReturnedItems != 40 ||
		!FirstPageOnly {
		t.Fatal("SOT-MODEL-023: package の固定予算が一致しません")
	}
	if budget.MaxRankedCandidates() != MaxRankedCandidates ||
		budget.MaxSelectedCandidates() != MaxSelectedCandidates ||
		budget.MaxParallelCandidates() != MaxParallelCandidates ||
		budget.MaxCapabilityCalls() != MaxCapabilityCalls ||
		budget.MaxItemsPerCollectionStep() != MaxItemsPerCollectionStep ||
		budget.MaxReturnedItems() != MaxReturnedItems ||
		budget.FirstPageOnly() != FirstPageOnly {
		t.Fatal("SOT-MODEL-023: plan の固定予算が一致しません")
	}
}

func TestLegalQueryBudgetAllocatesEveryReadCollectionCombination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		readSteps       int
		collectionSteps int
		effectiveLimit  int
	}{
		{0, 1, 20},
		{0, 2, 20},
		{0, 3, 13},
		{0, 4, 10},
		{1, 0, 0},
		{1, 1, 20},
		{1, 2, 19},
		{1, 3, 13},
		{2, 0, 0},
		{2, 1, 20},
		{2, 2, 19},
		{3, 0, 0},
		{3, 1, 20},
		{4, 0, 0},
	}
	for _, testCase := range tests {
		testCase := testCase
		name := fmt.Sprintf("R%d-C%d", testCase.readSteps, testCase.collectionSteps)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan := budgetTestExecutingPlan(
				t,
				testCase.readSteps,
				testCase.collectionSteps,
				MaxLimitPerAttempt,
			)
			budgetTestAssertAllocation(
				t,
				plan.Budget(),
				testCase.readSteps,
				testCase.collectionSteps,
				testCase.effectiveLimit,
			)
		})
	}
}

func TestLegalQueryBudgetCapsCollectionByRequestedLimit(t *testing.T) {
	t.Parallel()

	plan := budgetTestExecutingPlan(t, 0, 4, 7)
	budgetTestAssertAllocation(t, plan.Budget(), 0, 4, 7)
	if plan.Budget().LimitPerAttempt() != 7 {
		t.Fatalf("SOT-MODEL-023: limitPerAttempt = %d", plan.Budget().LimitPerAttempt())
	}
}

func TestLegalQueryBudgetPreservesSelectedCandidateAndStepOrder(t *testing.T) {
	t.Parallel()

	first := planTestCandidate(t, "candidate-first", 900, nil,
		planTestStep(t, "法令検索", "step-first-search"),
		planTestStep(t, "法令読取り", "step-first-read"))
	second := planTestCandidate(t, "candidate-second", 800, nil,
		planTestStep(t, "条文読取り", "step-second-read"),
		planTestStep(t, "更新一覧", "step-second-list"))
	plan, err := NewLegalQueryPlan(LegalQueryPlanValues{
		ProfileVersion:   "profile-budget-v1",
		Decision:         PlanDecisionHedged,
		RankedCandidates: []LegalQueryCandidate{first, second},
		Selected: []LegalQueryPlanSelection{
			planTestSelection(t, first, SelectionAvailabilityAvailable),
			planTestSelection(t, second, SelectionAvailabilityAvailable),
		},
		ReasonCodes:     []ReasonCode{ReasonCodeHedgedCloseCandidates},
		LimitPerAttempt: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := plan.Budget().StepBudgets()
	wantCandidates := []string{
		"candidate-first",
		"candidate-first",
		"candidate-second",
		"candidate-second",
	}
	wantSteps := []string{
		"step-first-search",
		"step-first-read",
		"step-second-read",
		"step-second-list",
	}
	for index := range got {
		if got[index].CandidateID() != wantCandidates[index] ||
			got[index].StepID() != wantSteps[index] {
			t.Fatalf("SOT-MODEL-023: stepBudgets[%d] = %#v", index, got[index])
		}
	}
	if plan.Budget().ReadStepCount() != 2 ||
		plan.Budget().CollectionStepCount() != 2 {
		t.Fatal("SOT-MODEL-023: hedged plan の R/C が一致しません")
	}
	wantReserved := []int{0, 1, 1, 0}
	for index, stepBudget := range got {
		limit, hasLimit := stepBudget.EffectiveLimit()
		if stepBudget.ReservedItems() != wantReserved[index] {
			t.Fatalf("stepBudgets[%d].reservedItems = %d", index, stepBudget.ReservedItems())
		}
		isCollection := wantReserved[index] == 0
		if hasLimit != isCollection || isCollection && limit != 19 {
			t.Fatalf("stepBudgets[%d].effectiveLimit = %d, %t", index, limit, hasLimit)
		}
	}
}

func TestLegalQueryBudgetIsEmptyForNonExecutingDecisions(t *testing.T) {
	t.Parallel()

	core := planTestCandidate(t, "candidate-core", 900, nil,
		planTestStep(t, "法令読取り", "step-core"))
	pack := planTestCandidate(t, "candidate-pack", 800, []string{"judicial-cases"},
		planTestStep(t, "裁判例検索", "step-pack"))
	tests := map[string]LegalQueryPlanValues{
		"needs_clarification": planTestValues(
			PlanDecisionNeedsClarification,
			[]LegalQueryCandidate{core},
			[]LegalQueryPlanSelection{
				planTestSelection(t, core, SelectionAvailabilityAvailable),
			},
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		),
		"capability_unavailable": planTestValues(
			PlanDecisionCapabilityUnavailable,
			[]LegalQueryCandidate{pack},
			[]LegalQueryPlanSelection{
				planTestSelection(t, pack, SelectionAvailabilityPackDisabled),
			},
			[]ReasonCode{ReasonCodeRequiredPackDisabled},
		),
		"unsupported": planTestValues(
			PlanDecisionUnsupported,
			[]LegalQueryCandidate{core},
			nil,
			[]ReasonCode{ReasonCodeUnsupportedTaskOrResource},
		),
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan, err := NewLegalQueryPlan(values)
			if err != nil {
				t.Fatal(err)
			}
			budget := plan.Budget()
			if budget.ReadStepCount() != 0 ||
				budget.CollectionStepCount() != 0 ||
				len(budget.StepBudgets()) != 0 ||
				budget.StepBudgets() == nil {
				t.Fatalf("SOT-MODEL-023: 非実行 budget = %#v", budget)
			}
			if budget.LimitPerAttempt() != values.LimitPerAttempt {
				t.Fatalf("limitPerAttempt = %d", budget.LimitPerAttempt())
			}
		})
	}
}

func TestLegalQueryBudgetRejectsInvalidLimitPerAttempt(t *testing.T) {
	t.Parallel()

	for _, limit := range []int{0, MaxLimitPerAttempt + 1} {
		values := planTestSingleValues(t)
		values.LimitPerAttempt = limit
		if _, err := NewLegalQueryPlan(values); err == nil {
			t.Fatalf("SOT-MODEL-023: limitPerAttempt=%d を受理しました", limit)
		}
	}
}

func TestLegalQueryBudgetIsImmutableAndRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	plan := budgetTestExecutingPlan(t, 1, 1, 20)
	stepBudgets := plan.Budget().StepBudgets()
	stepBudgets[0].candidateID = "changed"
	stepBudgets[0].stepID = "changed"
	if plan.Budget().StepBudgets()[0].CandidateID() != "candidate-budget" ||
		plan.Budget().StepBudgets()[0].StepID() != "read-a" {
		t.Fatal("SOT-MODEL-023: budget を getter から変更できました")
	}
	for _, target := range []any{
		&LegalQueryStepBudget{},
		&LegalQueryBudget{},
	} {
		if err := json.Unmarshal([]byte(`{}`), target); err == nil {
			t.Fatalf("SOT-MODEL-023: %T を JSON から直接復元できました", target)
		}
	}
}

func budgetTestExecutingPlan(
	t *testing.T,
	readSteps int,
	collectionSteps int,
	limitPerAttempt int,
) LegalQueryPlan {
	t.Helper()
	steps := budgetTestSteps(t, readSteps, collectionSteps)
	candidate := planTestCandidate(t, "candidate-budget", 900, nil, steps...)
	plan, err := NewLegalQueryPlan(LegalQueryPlanValues{
		ProfileVersion:   "profile-budget-v1",
		Decision:         PlanDecisionSingle,
		RankedCandidates: []LegalQueryCandidate{candidate},
		Selected: []LegalQueryPlanSelection{
			planTestSelection(t, candidate, SelectionAvailabilityAvailable),
		},
		ReasonCodes:     []ReasonCode{ReasonCodeSingleClearCandidate},
		LimitPerAttempt: limitPerAttempt,
	})
	if err != nil {
		t.Fatalf("試験用 plan を作成できません: %v", err)
	}
	return plan
}

func budgetTestSteps(
	t *testing.T,
	readSteps int,
	collectionSteps int,
) []LegalQueryCandidateStep {
	t.Helper()
	readFixtures := []string{"法令読取り", "条文読取り", "裁判例読取り", "法令読取り"}
	collectionFixtures := []string{"法令検索", "法令本文検索", "更新一覧", "裁判例検索"}
	steps := make([]LegalQueryCandidateStep, 0, readSteps+collectionSteps)
	for index := 0; index < readSteps; index++ {
		steps = append(steps, planTestStep(
			t,
			readFixtures[index],
			"read-"+planTestLetter(index),
		))
	}
	for index := 0; index < collectionSteps; index++ {
		steps = append(steps, planTestStep(
			t,
			collectionFixtures[index],
			"collection-"+planTestLetter(index),
		))
	}
	return steps
}

func budgetTestAssertAllocation(
	t *testing.T,
	budget LegalQueryBudget,
	readSteps int,
	collectionSteps int,
	effectiveLimit int,
) {
	t.Helper()
	if budget.ReadStepCount() != readSteps ||
		budget.CollectionStepCount() != collectionSteps ||
		len(budget.StepBudgets()) != readSteps+collectionSteps {
		t.Fatalf("SOT-MODEL-023: R/C と stepBudgets が一致しません")
	}
	for index, stepBudget := range budget.StepBudgets() {
		limit, hasLimit := stepBudget.EffectiveLimit()
		if index < readSteps {
			if stepBudget.ReservedItems() != 1 || hasLimit {
				t.Fatalf("read step budget = %#v", stepBudget)
			}
			continue
		}
		if stepBudget.ReservedItems() != 0 ||
			!hasLimit ||
			limit != effectiveLimit {
			t.Fatalf("collection step budget = %#v", stepBudget)
		}
	}
	if err := budget.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-023: budget が無効です: %v", err)
	}
}
