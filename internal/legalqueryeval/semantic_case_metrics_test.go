package legalqueryeval

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestAggregateSemanticCaseEvaluationsは主要指標とcategory別件数を集計する(t *testing.T) {
	t.Parallel()

	fixtures := comparisonStepFixtures(t)
	evaluationExact := mustPlanEvaluation(
		t,
		"holdout-intent-01",
		[]string{"intent-law-search"},
		mustExpectedMeaningWithID(
			t,
			"meaning-exact",
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			fixtures[0],
		),
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-exact",
				900,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixtures[0],
			),
		},
		[]string{"candidate-exact"},
	)
	evaluationTop2 := mustPlanEvaluation(
		t,
		"holdout-name-01",
		[]string{"name-official"},
		mustExpectedMeaningWithID(
			t,
			"meaning-top2",
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			fixtures[0],
		),
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-top1-wrong",
				950,
				legalquery.ConfidenceMedium,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixtures[1],
			),
			mustCandidate(
				t,
				"candidate-top2-correct",
				900,
				legalquery.ConfidenceMedium,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixtures[0],
			),
		},
		[]string{"candidate-top2-correct"},
	)
	evaluationConcept := mustPlanEvaluation(
		t,
		"holdout-name-02",
		[]string{"concept-single"},
		mustExpectedMeaningWithID(
			t,
			"meaning-concept",
			[]legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceLegalConcept,
			},
			[]string{"concept-a"},
			nil,
			fixtures[0],
		),
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-concept",
				910,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{
					legalquery.EvidenceExplicitTask,
					legalquery.EvidenceLegalConcept,
				},
				[]legalquery.LegalConceptSource{
					mustComparisonConceptSource(
						t,
						"concept-b",
						"資料乙",
						"https://example.go.jp/concept/b",
						"2026-07-28",
					),
				},
				nil,
				fixtures[0],
			),
		},
		[]string{"candidate-concept"},
	)
	requestErrorCase := mustSemanticRequestErrorCase(
		t,
		"holdout-input-01",
		[]string{"input-query-empty"},
		"",
		legalquerycorpus.RequestErrorFieldQuery,
	)
	requestErrorEval, err := EvaluateSemanticRequestErrorCase(
		requestErrorCase,
		mustArgumentError(t, "query", "は一文字以上でなければなりません"),
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: request_error 評価に失敗しました: %v", err)
	}

	metrics, err := AggregateSemanticCaseEvaluations(
		[]SemanticCaseEvaluation{
			evaluationExact,
			evaluationTop2,
			evaluationConcept,
			requestErrorEval,
		},
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024: AggregateSemanticCaseEvaluations() error = %v", err)
	}

	if metrics.CaseCount() != 4 {
		t.Fatalf("SOT-ENG-024: CaseCount() = %d、期待値 = 4", metrics.CaseCount())
	}
	assertMetricCounter(t, metrics.PlanOutcomeMetric(), 3, 3)
	assertMetricCounter(t, metrics.RequestErrorMetric(), 1, 1)
	assertMetricCounter(t, metrics.Top1Metric(), 2, 3)
	assertMetricCounter(t, metrics.Top2Metric(), 3, 3)
	assertMetricCounter(t, metrics.HighConfidenceMetric(), 2, 2)
	assertMetricCounter(t, metrics.EvidenceAssertionMetric(), 3, 3)
	assertMetricCounter(t, metrics.ConceptAssertionMetric(), 0, 1)

	categoryMetrics := metrics.CategoryMetrics()
	if len(categoryMetrics) != 3 {
		t.Fatalf("SOT-ENG-024: category metric 件数 = %d、期待値 = 3", len(categoryMetrics))
	}

	nameMetrics := findCategoryMetrics(t, categoryMetrics, "law-name-and-concept")
	assertMetricCounter(t, nameMetrics.Top1Metric(), 1, 2)
	assertMetricCounter(t, nameMetrics.Top2Metric(), 2, 2)
	assertMetricCounter(t, nameMetrics.ConceptAssertionMetric(), 0, 1)
}

func mustPlanEvaluation(
	t *testing.T,
	caseID string,
	coverageIDs []string,
	expectedMeaning legalquerycorpus.ExpectedMeaning,
	ranked []legalquery.LegalQueryCandidate,
	selectedIDs []string,
) SemanticCaseEvaluation {
	t.Helper()

	semanticCase := mustSemanticPlanCase(
		t,
		caseID,
		coverageIDs,
		"統合照会の評価",
		nil,
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionSingle,
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			[]legalquerycorpus.ExpectedMeaning{expectedMeaning},
			[]string{expectedMeaning.MeaningID()},
		),
	)
	evaluation, err := EvaluateSemanticPlanCase(
		semanticCase,
		mustPlan(
			t,
			"profile-default-v1",
			legalquery.PlanDecisionSingle,
			ranked,
			selectedIDs,
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			10,
		),
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: plan 評価に失敗しました: %v", err)
	}
	return evaluation
}

func assertMetricCounter(
	t *testing.T,
	counter MetricCounter,
	wantMatched int,
	wantTotal int,
) {
	t.Helper()

	if counter.Matched() != wantMatched || counter.Total() != wantTotal {
		t.Fatalf(
			"SOT-ENG-024: counter = (%d, %d)、期待値 = (%d, %d)",
			counter.Matched(),
			counter.Total(),
			wantMatched,
			wantTotal,
		)
	}
}

func findCategoryMetrics(
	t *testing.T,
	values []CategoryMetrics,
	categoryID string,
) CategoryMetrics {
	t.Helper()

	for _, value := range values {
		if value.CategoryID() == categoryID {
			return value
		}
	}
	t.Fatalf("SOT-ENG-024: category metric %q がありません", categoryID)
	return CategoryMetrics{}
}
