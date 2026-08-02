package legalqueryeval

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestSemanticBoundaryMismatchV2は評価失敗を不変値で表す(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expectedMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-synthetic-plan",
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)
	planCase := mustSemanticPlanCase(
		t,
		"holdout-synthetic-plan-boundary",
		[]string{"intent-law-search"},
		"合成した計画境界入力",
		nil,
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionSingle,
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			[]legalquerycorpus.ExpectedMeaning{expectedMeaning},
			[]string{"meaning-synthetic-plan"},
		),
	)
	planMismatch, err := EvaluateSemanticPlanArgumentErrorCaseV2(
		planCase,
		mustArgumentError(t, "query", "は一文字以上でなければなりません"),
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: plan 境界不一致を評価できません: %v", err)
	}
	if planMismatch.PlanOutcomeMatched() ||
		planMismatch.RequestErrorMatched() ||
		planMismatch.PrimaryTop1Matched() ||
		planMismatch.PrimaryTop2Matched() {
		t.Fatal("SOT-ENG-024: plan 境界不一致を成功として評価しました")
	}
	meanings := planMismatch.Meanings()
	if len(meanings) != 1 || meanings[0].MeaningID() != "meaning-synthetic-plan" ||
		meanings[0].SignatureMatched() {
		t.Fatalf("SOT-ENG-026: plan 境界不一致の meanings = %#v", meanings)
	}

	requestErrorCase := mustSemanticRequestErrorCase(
		t,
		"holdout-synthetic-request-boundary",
		[]string{"input-query-empty"},
		"",
		legalquerycorpus.RequestErrorFieldQuery,
	)
	acceptedMismatch, err := EvaluateSemanticAcceptedRequestErrorCaseV2(
		requestErrorCase,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: request_error 境界不一致を評価できません: %v", err)
	}
	if acceptedMismatch.ExpectedKind() !=
		legalquerycorpus.SemanticExpectedKindRequestError ||
		acceptedMismatch.RequestErrorMatched() ||
		acceptedMismatch.PlanOutcomeMatched() ||
		len(acceptedMismatch.Meanings()) != 0 {
		t.Fatalf("SOT-ENG-024: request_error 境界不一致 = %#v", acceptedMismatch)
	}

	metrics, err := AggregateSemanticCaseEvaluations(
		[]SemanticCaseEvaluation{planMismatch, acceptedMismatch},
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024: 境界不一致を集計できません: %v", err)
	}
	if metrics.PlanOutcomeMetric().Matched() != 0 ||
		metrics.PlanOutcomeMetric().Total() != 1 ||
		metrics.RequestErrorMetric().Matched() != 0 ||
		metrics.RequestErrorMetric().Total() != 1 ||
		metrics.MeaningSignatureMetric().Matched() != 0 ||
		metrics.MeaningSignatureMetric().Total() != 1 {
		t.Fatalf("SOT-ENG-024: 境界不一致の metrics = %#v", metrics)
	}
}
