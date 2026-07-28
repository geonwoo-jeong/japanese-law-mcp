package legalquery

import (
	"context"
	"testing"
)

func TestExecutorDispatchesAllSevenInputKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		stepName     string
		stepID       string
		kind         LogicalInputKind
		resultKind   LegalQueryAttemptResultKind
		isCollection bool
	}{
		{
			name: "法令検索", stepName: "法令検索", stepID: "step-law-search",
			kind: InputKindLawSearch, resultKind: LegalQueryResultKindLawSearch,
			isCollection: true,
		},
		{
			name: "法令本文検索", stepName: "法令本文検索",
			stepID: "step-law-content", kind: InputKindLawContentSearch,
			resultKind: LegalQueryResultKindLawContentSearch, isCollection: true,
		},
		{
			name: "法令読取り", stepName: "法令読取り", stepID: "step-law-read",
			kind: InputKindLawRead, resultKind: LegalQueryResultKindLawDocument,
		},
		{
			name: "条文読取り", stepName: "条文読取り", stepID: "step-article-read",
			kind: InputKindLawArticleRead, resultKind: LegalQueryResultKindLawArticle,
		},
		{
			name: "更新一覧", stepName: "更新一覧", stepID: "step-law-updates",
			kind: InputKindLawUpdates, resultKind: LegalQueryResultKindLawUpdates,
			isCollection: true,
		},
		{
			name: "裁判例検索", stepName: "裁判例検索", stepID: "step-judicial-search",
			kind:       InputKindJudicialDecisionSearch,
			resultKind: LegalQueryResultKindJudicialSearch, isCollection: true,
		},
		{
			name: "裁判例読取り", stepName: "裁判例読取り",
			stepID: "step-judicial-read", kind: InputKindJudicialDecisionRead,
			resultKind: LegalQueryResultKindJudicialDecision,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			recorder, core, judicial := newExecutorTestFacades(t)
			executor, err := NewExecutor(core, judicial)
			if err != nil {
				t.Fatalf("Executor を作成できません: %v", err)
			}
			plan := mustExecutorSinglePlan(
				t,
				mustExecutorStep(t, test.stepName, test.stepID),
			)

			execution, err := executor.Execute(context.Background(), plan)
			if err != nil {
				t.Fatalf("SOT-ARCH-022: Execute error = %v", err)
			}
			attempts := execution.Attempts()
			if len(attempts) != 1 ||
				executorTestAttemptResultKind(attempts[0]) != test.resultKind {
				t.Fatalf(
					"SOT-MODEL-024: attempt variant = %#v",
					attempts,
				)
			}
			calls := recorder.callsSnapshot()
			if len(calls) != 1 || calls[0].kind != test.kind {
				t.Fatalf("SOT-ARCH-022: facade calls = %#v", calls)
			}
			limit, hasLimit := calls[0].budget.EffectiveLimit()
			if test.isCollection {
				if calls[0].budget.ReservedItems() != 0 ||
					!hasLimit ||
					limit != 10 {
					t.Fatalf(
						"SOT-MODEL-023: collection budget = (%d, %d, %t)",
						calls[0].budget.ReservedItems(),
						limit,
						hasLimit,
					)
				}
			} else if calls[0].budget.ReservedItems() != 1 || hasLimit {
				t.Fatalf(
					"SOT-MODEL-023: read budget = (%d, %d, %t)",
					calls[0].budget.ReservedItems(),
					limit,
					hasLimit,
				)
			}
		})
	}
}

func executorTestAttemptResultKind(
	attempt LegalQueryAttempt,
) LegalQueryAttemptResultKind {
	switch value := attempt.(type) {
	case LegalQueryLawSearchAttempt:
		return value.ResultKind()
	case LegalQueryLawContentSearchAttempt:
		return value.ResultKind()
	case LegalQueryLawDocumentAttempt:
		return value.ResultKind()
	case LegalQueryLawArticleAttempt:
		return value.ResultKind()
	case LegalQueryLawUpdatesAttempt:
		return value.ResultKind()
	case LegalQueryJudicialSearchAttempt:
		return value.ResultKind()
	case LegalQueryJudicialDecisionAttempt:
		return value.ResultKind()
	default:
		return ""
	}
}
