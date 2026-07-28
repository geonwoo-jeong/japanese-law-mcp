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

func TestEvaluateSemanticPlanCaseは非single系decisionの規則も評価する(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]

	t.Run("needs_clarification", func(t *testing.T) {
		t.Parallel()

		expectedMeaning := mustExpectedMeaningWithID(
			t,
			"meaning-clarify",
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			fixture,
		)
		semanticCase := mustSemanticPlanCase(
			t,
			"holdout-clarification",
			[]string{"intent-law-search"},
			"行政手続法か行政不服審査法か確認したい",
			nil,
			mustExpectedPlan(
				t,
				legalquery.PlanDecisionNeedsClarification,
				[]legalquery.ReasonCode{legalquery.ReasonCodeBelowExecutionThreshold},
				[]legalquerycorpus.ExpectedMeaning{expectedMeaning},
				[]string{"meaning-clarify"},
			),
		)
		actualPlan := mustPlan(
			t,
			"profile-default-v1",
			legalquery.PlanDecisionNeedsClarification,
			[]legalquery.LegalQueryCandidate{
				mustCandidate(
					t,
					"candidate-clarify",
					650,
					legalquery.ConfidenceHigh,
					[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
					nil,
					nil,
					fixture,
				),
			},
			[]string{"candidate-clarify"},
			[]legalquery.ReasonCode{legalquery.ReasonCodeBelowExecutionThreshold},
			10,
		)

		evaluation, err := EvaluateSemanticPlanCase(semanticCase, actualPlan)
		if err != nil {
			t.Fatalf("SOT-ENG-024/026: needs_clarification 評価 error = %v", err)
		}
		if !evaluation.PlanOutcomeMatched() ||
			!evaluation.PrimaryTop1Matched() ||
			!evaluation.PrimaryTop2Matched() {
			t.Fatal("SOT-ENG-024: needs_clarification の評価が崩れています")
		}
		if matched, applicable := evaluation.HighConfidencePrecision(); !matched || !applicable {
			t.Fatalf(
				"SOT-ENG-024: needs_clarification high-confidence = (%t, %t)",
				matched,
				applicable,
			)
		}
	})

	t.Run("capability_unavailable", func(t *testing.T) {
		t.Parallel()

		expectedMeaning := mustExpectedMeaningWithID(
			t,
			"meaning-pack-disabled",
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			[]string{"judicial-cases"},
			fixture,
		)
		safetyVariant := legalquerycorpus.SafetyVariantOrdinary
		semanticCase := mustSemanticPlanCaseWithSafety(
			t,
			"holdout-pack-disabled",
			[]string{"boundary-pack-disabled"},
			"判例も含めて確認",
			nil,
			&safetyVariant,
			mustExpectedPlan(
				t,
				legalquery.PlanDecisionCapabilityUnavailable,
				[]legalquery.ReasonCode{legalquery.ReasonCodeRequiredPackDisabled},
				[]legalquerycorpus.ExpectedMeaning{expectedMeaning},
				[]string{"meaning-pack-disabled"},
			),
		)
		actualPlan := mustPlan(
			t,
			"profile-default-v1",
			legalquery.PlanDecisionCapabilityUnavailable,
			[]legalquery.LegalQueryCandidate{
				mustCandidate(
					t,
					"candidate-pack-disabled",
					700,
					legalquery.ConfidenceHigh,
					[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
					nil,
					[]string{"judicial-cases"},
					fixture,
				),
			},
			[]string{"candidate-pack-disabled"},
			[]legalquery.ReasonCode{legalquery.ReasonCodeRequiredPackDisabled},
			10,
		)

		evaluation, err := EvaluateSemanticPlanCase(semanticCase, actualPlan)
		if err != nil {
			t.Fatalf("SOT-ENG-024/026: capability_unavailable 評価 error = %v", err)
		}
		if !evaluation.PlanOutcomeMatched() ||
			!evaluation.PrimaryTop1Matched() ||
			!evaluation.PrimaryTop2Matched() {
			t.Fatal("SOT-ENG-024: capability_unavailable の selection/ranking を誤判定しました")
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()

		expectedMeaning := mustExpectedMeaningWithID(
			t,
			"meaning-unsupported",
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			fixture,
		)
		semanticCase := mustSemanticPlanCase(
			t,
			"holdout-unsupported",
			[]string{"unsupported-legal-advice"},
			"永住許可の見込みを判断して",
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
					"candidate-unsupported",
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
		if !evaluation.PlanOutcomeMatched() {
			t.Fatal("SOT-ENG-024: unsupported 의 decision/reason/selection 비교가 실패했습니다")
		}
		if evaluation.PrimaryTop1Matched() || evaluation.PrimaryTop2Matched() {
			t.Fatal("SOT-ENG-024: unsupported 를 ranking 분모에 넣었습니다")
		}
		if matched, applicable := evaluation.HighConfidencePrecision(); matched || applicable {
			t.Fatalf(
				"SOT-ENG-024: unsupported high-confidence = (%t, %t)",
				matched,
				applicable,
			)
		}
	})
}

func TestEvaluateSemanticPlanCaseは有効pack付きavailable選択を受理する(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[5]
	expectedMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-pack-enabled",
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		[]string{"judicial-cases"},
		fixture,
	)
	semanticCase := mustSemanticPlanCase(
		t,
		"holdout-pack-enabled",
		[]string{"pack-judicial-enabled"},
		"判例付きで検索",
		[]string{"judicial-cases"},
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionSingle,
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			[]legalquerycorpus.ExpectedMeaning{expectedMeaning},
			[]string{"meaning-pack-enabled"},
		),
	)
	actualPlan := mustPlanWithSelectionAvailabilities(
		t,
		"profile-default-v1",
		legalquery.PlanDecisionSingle,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-pack-enabled",
				910,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				[]string{"judicial-cases"},
				fixture,
			),
		},
		[]string{"candidate-pack-enabled"},
		map[string]legalquery.SelectionAvailability{
			"candidate-pack-enabled": legalquery.SelectionAvailabilityAvailable,
		},
		[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
		10,
	)

	evaluation, err := EvaluateSemanticPlanCase(semanticCase, actualPlan)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: pack enabled selection 評価 error = %v", err)
	}
	if !evaluation.PlanOutcomeMatched() {
		t.Fatal("SOT-ENG-024/026: enabled pack 의 available selection 을 거부했습니다")
	}
}

func TestEvaluateSemanticPlanCaseはhedgedを評価できる(t *testing.T) {
	t.Parallel()

	fixtures := comparisonStepFixtures(t)
	firstMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-first",
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixtures[0],
	)
	secondMeaning := mustExpectedMeaningWithID(
		t,
		"meaning-second",
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixtures[1],
	)
	semanticCase := mustSemanticPlanCase(
		t,
		"holdout-hedged-order",
		[]string{"name-official"},
		"行政手続法または行政不服審査法",
		nil,
		mustExpectedPlan(
			t,
			legalquery.PlanDecisionHedged,
			[]legalquery.ReasonCode{legalquery.ReasonCodeHedgedCloseCandidates},
			[]legalquerycorpus.ExpectedMeaning{firstMeaning, secondMeaning},
			[]string{"meaning-first", "meaning-second"},
		),
	)
	actualPlan := mustPlan(
		t,
		"profile-default-v1",
		legalquery.PlanDecisionHedged,
		[]legalquery.LegalQueryCandidate{
			mustCandidate(
				t,
				"candidate-first",
				900,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixtures[0],
			),
			mustCandidate(
				t,
				"candidate-second",
				890,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				fixtures[1],
			),
		},
		[]string{"candidate-first", "candidate-second"},
		[]legalquery.ReasonCode{legalquery.ReasonCodeHedgedCloseCandidates},
		10,
	)

	evaluation, err := EvaluateSemanticPlanCase(semanticCase, actualPlan)
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: hedged 評価 error = %v", err)
	}
	if !evaluation.PlanOutcomeMatched() {
		t.Fatal("SOT-ENG-024: hedged の decision/reason/selection 完全一致を失いました")
	}
	if !evaluation.PrimaryTop1Matched() || !evaluation.PrimaryTop2Matched() {
		t.Fatal("SOT-ENG-024: hedged の ranking 一致を失いました")
	}
	if matched, applicable := evaluation.HighConfidencePrecision(); !matched || !applicable {
		t.Fatalf(
			"SOT-ENG-024: hedged high-confidence = (%t, %t)",
			matched,
			applicable,
		)
	}
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
