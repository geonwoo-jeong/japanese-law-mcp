package legalqueryeval

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestEvaluateSemanticHoldoutReproducibilityはPlan順序の非決定性を検出する(
	t *testing.T,
) {
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		semanticRunnerRepositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	step := comparisonStepFixtures(t)[0]
	firstPlan := mustPlan(
		t,
		"profile-set-v1",
		legalquery.PlanDecisionSingle,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-a",
				100,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				step,
			),
		},
		[]string{"candidate-a"},
		[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
		20,
	)
	secondPlan := mustPlan(
		t,
		"profile-set-v1",
		legalquery.PlanDecisionSingle,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-b",
				100,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				step,
			),
		},
		[]string{"candidate-b"},
		[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
		20,
	)

	callCounts := make(map[string]int)
	firstPlanCaseID := ""
	semantic, reproducibility, err := EvaluateSemanticHoldoutReproducibility(
		context.Background(),
		corpus,
		func(
			_ context.Context,
			semanticCase legalquerycorpus.SemanticCase,
		) (
			SemanticCaseEvaluation,
			legalquery.LegalQueryPlan,
			bool,
			error,
		) {
			evaluation := matchedSemanticEvaluation(semanticCase)
			if semanticCase.Expected().Kind() ==
				legalquerycorpus.SemanticExpectedKindRequestError {
				return evaluation, legalquery.LegalQueryPlan{}, false, nil
			}
			callCounts[semanticCase.CaseID()]++
			if firstPlanCaseID == "" {
				firstPlanCaseID = semanticCase.CaseID()
			}
			if semanticCase.CaseID() == firstPlanCaseID &&
				callCounts[semanticCase.CaseID()] == 2 {
				return evaluation, secondPlan, true, nil
			}
			return evaluation, firstPlan, true, nil
		},
	)
	if err != nil {
		t.Fatalf("EvaluateSemanticHoldoutReproducibility() error = %v", err)
	}
	if semantic.CaseCount() != len(corpus.Holdout()) {
		t.Fatalf(
			"semantic case count = %d, want %d",
			semantic.CaseCount(),
			len(corpus.Holdout()),
		)
	}
	if reproducibility.Denominator() == 0 ||
		reproducibility.Numerator() != reproducibility.Denominator()-1 ||
		len(reproducibility.FailedCaseIDs()) != 1 ||
		reproducibility.FailedCaseIDs()[0] != firstPlanCaseID {
		t.Fatalf(
			"SOT-ENG-024: plan 順序の非決定性を検出できません: %#v",
			reproducibility,
		)
	}
}
