package legalqueryeval

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestSemanticReproducibilityV2はPlan有無不一致を評価失敗へ集計する(
	t *testing.T,
) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expectedMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-synthetic-reproducibility",
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)
	planCase := mustSemanticPlanCase(
		t,
		"holdout-synthetic-plan-presence",
		[]string{"intent-law-search"},
		"合成した計画有無入力",
		nil,
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionSingle,
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			[]legalquerycorpus.ExpectedMeaning{expectedMeaning},
			[]string{"meaning-synthetic-reproducibility"},
		),
	)
	requestErrorCase := mustSemanticRequestErrorCase(
		t,
		"holdout-synthetic-unexpected-plan",
		[]string{"input-query-empty"},
		"",
		legalquerycorpus.RequestErrorFieldQuery,
	)
	actualPlan := mustPlan(
		t,
		"profile-set-synthetic-v1",
		legalquery.PlanDecisionSingle,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-synthetic",
				100,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixture,
			),
		},
		[]string{"candidate-synthetic"},
		[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
		20,
	)
	evaluate := func(
		_ context.Context,
		semanticCase legalquerycorpus.SemanticCase,
	) (
		SemanticCaseEvaluation,
		legalquery.LegalQueryPlan,
		bool,
		error,
	) {
		if semanticCase.Expected().Kind() ==
			legalquerycorpus.SemanticExpectedKindPlan {
			evaluation, err := EvaluateSemanticPlanArgumentErrorCaseV2(
				semanticCase,
				mustArgumentError(t, "query", "は一文字以上でなければなりません"),
			)
			return evaluation, legalquery.LegalQueryPlan{}, false, err
		}
		evaluation, err := EvaluateSemanticAcceptedRequestErrorCaseV2(
			semanticCase,
		)
		return evaluation, actualPlan, true, err
	}

	if _, _, err := evaluateSemanticCasesReproducibility(
		context.Background(),
		[]legalquerycorpus.SemanticCase{planCase},
		evaluate,
		semanticReproducibilityStrict,
	); err == nil {
		t.Fatal("SOT-ENG-024: v1 strict が plan 有無不一致を受理しました")
	}

	semantic, reproducibility, err := evaluateSemanticCasesReproducibility(
		context.Background(),
		[]legalquerycorpus.SemanticCase{planCase, requestErrorCase},
		evaluate,
		semanticReproducibilityScoredMismatch,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/038/040: v2 境界不一致の集計 error = %v", err)
	}
	planMetric := semanticMetricForTest(
		t,
		semantic,
		SemanticMetricPlanOutcome,
	)
	requestMetric := semanticMetricForTest(
		t,
		semantic,
		SemanticMetricRequestError,
	)
	if semantic.CaseCount() != 2 ||
		planMetric.Matched() != 0 || planMetric.Total() != 1 ||
		requestMetric.Matched() != 0 || requestMetric.Total() != 1 {
		t.Fatalf("SOT-ENG-024: semantic report = %#v", semantic)
	}
	if reproducibility.Numerator() != 0 ||
		reproducibility.Denominator() != 1 ||
		len(reproducibility.FailedCaseIDs()) != 1 ||
		reproducibility.FailedCaseIDs()[0] != planCase.CaseID() {
		t.Fatalf("SOT-ENG-024: reproducibility = %#v", reproducibility)
	}
}

func semanticMetricForTest(
	t *testing.T,
	report SemanticReport,
	metricID SemanticMetricID,
) SemanticMetricReport {
	t.Helper()

	for _, metric := range report.Metrics() {
		if metric.MetricID() == metricID {
			return metric
		}
	}
	t.Fatalf("semantic metric %q がありません", metricID)
	return SemanticMetricReport{}
}
