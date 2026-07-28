package legalqueryeval

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestEvaluateSemanticPlanCaseは完全一致のplanを評価できる(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expectedMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-law-search",
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)
	semanticCase := mustSemanticPlanCase(
		t,
		"development-intent-law-search",
		[]string{"intent-law-search"},
		"行政手続法を検索",
		nil,
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionSingle,
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			[]legalquerycorpus.ExpectedMeaning{expectedMeaning},
			[]string{"meaning-law-search"},
		),
	)
	actualPlan := mustPlan(
		t,
		"profile-default-v1",
		legalquery.PlanDecisionSingle,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-law-search",
				900,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixture,
			),
		},
		[]string{"candidate-law-search"},
		[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
		10,
	)

	evaluation, err := EvaluateSemanticPlanCase(semanticCase, actualPlan)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: EvaluateSemanticPlanCase() error = %v", err)
	}
	if !evaluation.PlanOutcomeMatched() {
		t.Fatal("SOT-ENG-024: decision/reason/selection の完全一致を失敗として扱いました")
	}
	if !evaluation.PrimaryTop1Matched() || !evaluation.PrimaryTop2Matched() {
		t.Fatal("SOT-ENG-024: 主正解の ranking 一致を誤判定しました")
	}
	if matched, applicable := evaluation.HighConfidencePrecision(); !matched || !applicable {
		t.Fatalf(
			"SOT-ENG-024: high-confidence precision = (%t, %t)、期待値 = (true, true)",
			matched,
			applicable,
		)
	}
	meanings := evaluation.Meanings()
	if len(meanings) != 1 || meanings[0].MatchedCandidateRank() != 1 {
		t.Fatalf("SOT-ENG-026: meanings = %#v", meanings)
	}
	if !meanings[0].SignatureMatched() {
		t.Fatal("SOT-ENG-026: 主正解の意味署名一致を失いました")
	}
	assertMeaningAssertions(
		t,
		meanings[0],
		true,
		true,
		false,
		false,
	)
}

func TestEvaluateSemanticPlanCaseはtop2recallとhighConfidence非適用を分離する(t *testing.T) {
	t.Parallel()

	fixtures := comparisonStepFixtures(t)
	primary := mustExpectedMeaningWithID(
		t,
		"meaning-law-search",
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixtures[0],
	)
	secondary := mustExpectedMeaningWithID(
		t,
		"meaning-law-content-search",
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixtures[1],
	)
	semanticCase := mustSemanticPlanCase(
		t,
		"holdout-name-top2",
		[]string{"name-official"},
		"行政手続法を検索",
		nil,
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionSingle,
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			[]legalquerycorpus.ExpectedMeaning{primary, secondary},
			[]string{"meaning-law-search"},
		),
	)
	actualPlan := mustPlan(
		t,
		"profile-default-v1",
		legalquery.PlanDecisionSingle,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-wrong-top1",
				950,
				legalquery.ConfidenceMedium,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixtures[1],
			),
			mustCandidate(
				t,
				"candidate-correct-top2",
				900,
				legalquery.ConfidenceMedium,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixtures[0],
			),
		},
		[]string{"candidate-correct-top2"},
		[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
		10,
	)

	evaluation, err := EvaluateSemanticPlanCase(semanticCase, actualPlan)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: EvaluateSemanticPlanCase() error = %v", err)
	}
	if !evaluation.PlanOutcomeMatched() {
		t.Fatal("SOT-ENG-024: selection の意味一致を誤って失敗にしました")
	}
	if evaluation.PrimaryTop1Matched() {
		t.Fatal("SOT-ENG-024: 誤った top-1 を主正解と判定しました")
	}
	if !evaluation.PrimaryTop2Matched() {
		t.Fatal("SOT-ENG-024: top-2 recall を失いました")
	}
	if matched, applicable := evaluation.HighConfidencePrecision(); matched || applicable {
		t.Fatalf(
			"SOT-ENG-024: high-confidence precision = (%t, %t)、期待値 = (false, false)",
			matched,
			applicable,
		)
	}
}

func TestEvaluateSemanticPlanCaseはdecisionReasonSelectionとconceptAssertion差を分離する(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expectedMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-concept",
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
		},
		[]string{"concept-a"},
		nil,
		fixture,
	)
	semanticCase := mustSemanticPlanCase(
		t,
		"holdout-concept-single",
		[]string{"concept-single"},
		"永住許可を検索",
		nil,
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionSingle,
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			[]legalquerycorpus.ExpectedMeaning{expectedMeaning},
			[]string{"meaning-concept"},
		),
	)
	actualPlan := mustPlan(
		t,
		"profile-default-v1",
		legalquery.PlanDecisionSingle,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-concept-mismatch",
				920,
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
				fixture,
			),
		},
		[]string{"candidate-concept-mismatch"},
		[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
		10,
	)

	evaluation, err := EvaluateSemanticPlanCase(semanticCase, actualPlan)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: EvaluateSemanticPlanCase() error = %v", err)
	}
	if !evaluation.PlanOutcomeMatched() {
		t.Fatal("SOT-ENG-024: plan outcome の完全一致まで巻き込んで失敗させました")
	}
	meanings := evaluation.Meanings()
	if len(meanings) != 1 {
		t.Fatalf("SOT-ENG-026: meanings 件数 = %d", len(meanings))
	}
	assertMeaningAssertions(
		t,
		meanings[0],
		true,
		true,
		false,
		true,
	)
}

func TestEvaluateSemanticRequestErrorCaseはfieldとerrorCodeを比較する(t *testing.T) {
	t.Parallel()

	semanticCase := mustSemanticRequestErrorCase(
		t,
		"holdout-input-query-empty",
		[]string{"input-query-empty"},
		"",
		legalquerycorpus.RequestErrorFieldQuery,
	)
	argumentError := mustArgumentError(
		t,
		"query",
		"は一文字以上でなければなりません",
	)

	evaluation, err := EvaluateSemanticRequestErrorCase(semanticCase, argumentError)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: EvaluateSemanticRequestErrorCase() error = %v", err)
	}
	if !evaluation.RequestErrorMatched() {
		t.Fatal("SOT-ENG-026: request_error の field/code 完全一致を失いました")
	}
	if evaluation.PlanOutcomeMatched() || evaluation.PrimaryTop1Matched() || evaluation.PrimaryTop2Matched() {
		t.Fatal("SOT-ENG-024: request_error case 를 ranking/plan 집계에 섞었습니다")
	}
}

func assertMeaningAssertions(
	t *testing.T,
	evaluation MeaningEvaluation,
	wantEvidenceMatched bool,
	wantEvidenceApplicable bool,
	wantConceptMatched bool,
	wantConceptApplicable bool,
) {
	t.Helper()

	evidenceMatched, evidenceApplicable := evaluation.EvidenceAssertion()
	if evidenceMatched != wantEvidenceMatched || evidenceApplicable != wantEvidenceApplicable {
		t.Fatalf(
			"SOT-ENG-026: EvidenceAssertion() = (%t, %t)、期待値 = (%t, %t)",
			evidenceMatched,
			evidenceApplicable,
			wantEvidenceMatched,
			wantEvidenceApplicable,
		)
	}
	conceptMatched, conceptApplicable := evaluation.ConceptAssertion()
	if conceptMatched != wantConceptMatched || conceptApplicable != wantConceptApplicable {
		t.Fatalf(
			"SOT-ENG-026: ConceptAssertion() = (%t, %t)、期待値 = (%t, %t)",
			conceptMatched,
			conceptApplicable,
			wantConceptMatched,
			wantConceptApplicable,
		)
	}
}
