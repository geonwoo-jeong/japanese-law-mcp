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
	assertMetricCounter(t, metrics.MeaningSignatureMetric(), 3, 3)
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

func TestAggregateSemanticCaseEvaluationsはduplicateCaseIdを拒否する(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	evaluation := mustPlanEvaluation(
		t,
		"holdout-duplicate-case",
		[]string{"intent-law-search"},
		mustExpectedMeaningWithID(
			t,
			"meaning-duplicate",
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			fixture,
		),
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-duplicate",
				900,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixture,
			),
		},
		[]string{"candidate-duplicate"},
	)

	if _, err := AggregateSemanticCaseEvaluations(
		[]SemanticCaseEvaluation{evaluation, evaluation},
	); err == nil {
		t.Fatal("SOT-ENG-024: duplicate caseId 集計を受理しました")
	}
}

func TestAggregateSemanticCaseEvaluationsは未初期化と混在状態を拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		evaluation SemanticCaseEvaluation
	}{
		{
			name:       "uninitialized",
			evaluation: SemanticCaseEvaluation{},
		},
		{
			name: "request_error_with_plan_fields",
			evaluation: SemanticCaseEvaluation{
				caseID:              "holdout-invalid-request",
				categoryIDs:         []string{"input-boundary"},
				coverageIDs:         []string{"input-query-empty"},
				expectedKind:        legalquerycorpus.SemanticExpectedKindRequestError,
				requestErrorMatched: true,
				planOutcomeMatched:  true,
				initialized:         true,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := AggregateSemanticCaseEvaluations(
				[]SemanticCaseEvaluation{test.evaluation},
			); err == nil {
				t.Fatal("SOT-ENG-024: 불량 evaluation 을 집계에 허용했습니다")
			}
		})
	}
}

func TestAggregateSemanticCaseEvaluationsはcategory順を決定的に返す(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	first := mustPlanEvaluation(
		t,
		"holdout-category-b",
		[]string{"name-official"},
		mustExpectedMeaningWithID(
			t,
			"meaning-category-b",
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			fixture,
		),
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-category-b",
				900,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixture,
			),
		},
		[]string{"candidate-category-b"},
	)
	second := mustPlanEvaluation(
		t,
		"holdout-category-a",
		[]string{"intent-law-search"},
		mustExpectedMeaningWithID(
			t,
			"meaning-category-a",
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			fixture,
		),
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-category-a",
				880,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixture,
			),
		},
		[]string{"candidate-category-a"},
	)

	metrics, err := AggregateSemanticCaseEvaluations(
		[]SemanticCaseEvaluation{first, second},
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024: category 順テスト集計 error = %v", err)
	}

	got := metrics.CategoryMetrics()
	if len(got) != 2 {
		t.Fatalf("SOT-ENG-024: category 件数 = %d、期待値 = 2", len(got))
	}
	if got[0].CategoryID() != "capability-intent" ||
		got[1].CategoryID() != "law-name-and-concept" {
		t.Fatalf(
			"SOT-ENG-024: category 順 = [%s, %s]",
			got[0].CategoryID(),
			got[1].CategoryID(),
		)
	}
}

func TestAggregateSemanticCaseEvaluationsは複数meaningのassertionをcase単位で集計する(
	t *testing.T,
) {
	t.Parallel()

	fixtures := comparisonStepFixtures(t)
	firstMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-concept-first",
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
		},
		[]string{"concept-a"},
		nil,
		fixtures[0],
	)
	secondMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-concept-second",
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
		},
		[]string{"concept-b"},
		nil,
		fixtures[1],
	)
	semanticCase := mustSemanticPlanCase(
		t,
		"holdout-concept-multiple",
		[]string{"concept-single"},
		"永住許可に関係する法令と条文を検索",
		nil,
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionHedged,
			[]legalquery.ReasonCode{legalquery.ReasonCodeHedgedCloseCandidates},
			[]legalquerycorpus.ExpectedMeaning{firstMeaning, secondMeaning},
			[]string{"meaning-concept-first", "meaning-concept-second"},
		),
	)
	actualPlan := mustPlan(
		t,
		"profile-default-v1",
		legalquery.PlanDecisionHedged,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-concept-first",
				900,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{
					legalquery.EvidenceExplicitTask,
					legalquery.EvidenceLegalConcept,
				},
				[]legalquery.LegalConceptSource{
					mustComparisonConceptSource(
						t,
						"concept-a",
						"資料甲",
						"https://example.go.jp/concept/a",
						"2026-07-28",
					),
				},
				nil,
				fixtures[0],
			),
			mustCandidate(
				t,
				"candidate-concept-second",
				890,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{
					legalquery.EvidenceExplicitTask,
					legalquery.EvidenceLegalConcept,
				},
				[]legalquery.LegalConceptSource{
					mustComparisonConceptSource(
						t,
						"concept-c",
						"資料丙",
						"https://example.go.jp/concept/c",
						"2026-07-28",
					),
				},
				nil,
				fixtures[1],
			),
		},
		[]string{"candidate-concept-first", "candidate-concept-second"},
		[]legalquery.ReasonCode{legalquery.ReasonCodeHedgedCloseCandidates},
		10,
	)
	evaluation, err := EvaluateSemanticPlanCase(semanticCase, actualPlan)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: 複数 meaning 評価 error = %v", err)
	}

	metrics, err := AggregateSemanticCaseEvaluations(
		[]SemanticCaseEvaluation{evaluation},
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024: 複数 meaning 集計 error = %v", err)
	}
	assertMetricCounter(t, metrics.PlanOutcomeMetric(), 1, 1)
	assertMetricCounter(t, metrics.MeaningSignatureMetric(), 1, 1)
	assertMetricCounter(t, metrics.EvidenceAssertionMetric(), 1, 1)
	assertMetricCounter(t, metrics.ConceptAssertionMetric(), 0, 1)

	category := findCategoryMetrics(
		t,
		metrics.CategoryMetrics(),
		"law-name-and-concept",
	)
	assertMetricCounter(t, category.MeaningSignatureMetric(), 1, 1)
	assertMetricCounter(t, category.EvidenceAssertionMetric(), 1, 1)
	assertMetricCounter(t, category.ConceptAssertionMetric(), 0, 1)
}

func TestAggregateSemanticCaseEvaluationsはunsupportedをranking母集団から除外する(
	t *testing.T,
) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expectedMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-unsupported-metric",
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)
	semanticCase := mustSemanticPlanCase(
		t,
		"holdout-unsupported-metric",
		[]string{"unsupported-legal-advice"},
		"具体的な勝訴見込みを判断して",
		nil,
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionUnsupported,
			[]legalquery.ReasonCode{legalquery.ReasonCodeUnsupportedTaskOrResource},
			[]legalquerycorpus.ExpectedMeaning{expectedMeaning},
			nil,
		),
	)
	actualPlan := mustPlan(
		t,
		"profile-default-v1",
		legalquery.PlanDecisionUnsupported,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-unsupported-metric",
				400,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixture,
			),
		},
		nil,
		[]legalquery.ReasonCode{legalquery.ReasonCodeUnsupportedTaskOrResource},
		10,
	)
	evaluation, err := EvaluateSemanticPlanCase(semanticCase, actualPlan)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: unsupported 評価 error = %v", err)
	}
	metrics, err := AggregateSemanticCaseEvaluations(
		[]SemanticCaseEvaluation{evaluation},
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024: unsupported 集計 error = %v", err)
	}

	assertMetricCounter(t, metrics.PlanOutcomeMetric(), 1, 1)
	assertMetricCounter(t, metrics.Top1Metric(), 0, 0)
	assertMetricCounter(t, metrics.Top2Metric(), 0, 0)
	assertMetricCounter(t, metrics.HighConfidenceMetric(), 0, 0)
	assertMetricCounter(t, metrics.MeaningSignatureMetric(), 1, 1)
	assertMetricCounter(t, metrics.EvidenceAssertionMetric(), 1, 1)
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
