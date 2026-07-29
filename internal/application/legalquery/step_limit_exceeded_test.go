package legalquery

import (
	"reflect"
	"testing"
)

func TestQueryCompositionConstraintは定義済み値だけを受理する(t *testing.T) {
	t.Parallel()

	for _, value := range []QueryCompositionConstraint{
		QueryCompositionConstraintNone,
		QueryCompositionConstraintIneligible,
		QueryCompositionConstraintStepLimitExceeded,
	} {
		if err := value.Validate(); err != nil {
			t.Fatalf("SOT-ARCH-027: constraint %q を拒否しました: %v", value, err)
		}
	}
	for _, value := range []QueryCompositionConstraint{
		"",
		"unknown",
	} {
		if err := value.Validate(); err == nil {
			t.Fatalf("SOT-ARCH-027: constraint %q を受理しました", value)
		}
	}
}

func TestCandidateGenerationはProfileSet内部だけの合成不適格制約を拒否する(
	t *testing.T,
) {
	t.Parallel()

	values := CandidateGenerationValues{
		ProfileID:             "core",
		ProfileVersion:        "core-v1",
		RankingVersion:        "ranking-v1",
		SelectionMode:         QuerySelectionModeAutomatic,
		CompositionConstraint: QueryCompositionConstraintIneligible,
	}
	if _, err := NewCandidateGeneration(values); err == nil {
		t.Fatal("SOT-MODEL-026: profile が内部合成不適格制約を出力できました")
	}
}

func TestSelectorは合成不適格を部分実行せず明確化する(t *testing.T) {
	t.Parallel()

	candidate := mustSelectorTestCandidate(
		t,
		"candidate-composition-ineligible",
		200,
		[]string{"judicial-cases"},
		1,
	)
	result := mustSelectorTestProfileSetResult(
		t,
		[]LegalQueryCandidate{candidate},
		nil,
		QuerySelectionModeAutomatic,
		nil,
	)
	result.compositionConstraint = QueryCompositionConstraintIneligible

	plan := mustSelectTestPlan(
		t,
		result,
		mustSelectorTestPackState(
			t,
			[]string{"judicial-cases"},
			nil,
		),
		DefaultLimitPerAttempt,
	)
	if plan.Decision() != PlanDecisionNeedsClarification ||
		len(plan.Selected()) != 0 ||
		!reflect.DeepEqual(
			plan.ReasonCodes(),
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		) ||
		len(plan.Budget().StepBudgets()) != 0 {
		t.Fatalf(
			"SOT-ARCH-027: 合成不適格 plan = decision:%q selected:%#v reasons:%#v budget:%#v",
			plan.Decision(),
			plan.Selected(),
			plan.ReasonCodes(),
			plan.Budget(),
		)
	}
}

func TestLegalQueryPlanは五step以上を単独理由で明確化する(t *testing.T) {
	t.Parallel()

	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionNeedsClarification,
		nil,
		nil,
		[]ReasonCode{ReasonCode("step_limit_exceeded")},
	))
	if err != nil {
		t.Fatalf("SOT-MODEL-023: step_limit_exceeded plan を作成できません: %v", err)
	}
	if plan.Decision() != PlanDecisionNeedsClarification ||
		len(plan.Selected()) != 0 ||
		!reflect.DeepEqual(
			plan.ReasonCodes(),
			[]ReasonCode{ReasonCode("step_limit_exceeded")},
		) ||
		len(plan.Budget().StepBudgets()) != 0 {
		t.Fatalf(
			"SOT-MODEL-023: step_limit_exceeded plan = decision:%q selected:%#v reasons:%#v budget:%#v",
			plan.Decision(),
			plan.Selected(),
			plan.ReasonCodes(),
			plan.Budget(),
		)
	}
}

func TestSelectorはstep上限制約をpack可否より先に非実行へ変換する(
	t *testing.T,
) {
	t.Parallel()

	candidate := mustSelectorTestCandidate(
		t,
		"candidate-step-limit-pack",
		200,
		[]string{"judicial-cases"},
		1,
	)
	result := mustSelectorTestProfileSetResult(
		t,
		[]LegalQueryCandidate{candidate},
		nil,
		QuerySelectionModeAutomatic,
		nil,
	)
	result.compositionConstraint =
		QueryCompositionConstraintStepLimitExceeded

	plan := mustSelectTestPlan(
		t,
		result,
		mustSelectorTestPackState(
			t,
			[]string{"judicial-cases"},
			nil,
		),
		DefaultLimitPerAttempt,
	)
	if plan.Decision() != PlanDecisionNeedsClarification ||
		len(plan.Selected()) != 0 ||
		!reflect.DeepEqual(
			plan.ReasonCodes(),
			[]ReasonCode{ReasonCodeStepLimitExceeded},
		) ||
		len(plan.Budget().StepBudgets()) != 0 {
		t.Fatalf(
			"SOT-ARCH-027: step 上限制約の plan = decision:%q selected:%#v reasons:%#v budget:%#v",
			plan.Decision(),
			plan.Selected(),
			plan.ReasonCodes(),
			plan.Budget(),
		)
	}
}

func TestSelectorは未知の候補合成制約を拒否する(t *testing.T) {
	t.Parallel()

	result := mustSelectorTestProfileSetResult(
		t,
		nil,
		nil,
		QuerySelectionModeAutomatic,
		nil,
	)
	result.compositionConstraint = QueryCompositionConstraint("unknown")

	if _, err := SelectLegalQueryPlan(SelectorInput{
		ProfileSetResult: result,
		PackState:        mustSelectorTestPackState(t, nil, nil),
		LimitPerAttempt:  DefaultLimitPerAttempt,
	}); err == nil {
		t.Fatal("SOT-ARCH-027: selector が未知の候補合成制約を受理しました")
	}
}

func TestSelectorは対象外との混在をstep上限制約より先に拒否する(
	t *testing.T,
) {
	t.Parallel()

	candidate := mustSelectorTestCandidate(
		t,
		"candidate-step-limit-unsupported",
		200,
		nil,
		1,
	)
	result := mustSelectorTestProfileSetResult(
		t,
		[]LegalQueryCandidate{candidate},
		[]CandidateGenerationSignal{
			CandidateSignalUnsupportedLegalAdvice,
		},
		QuerySelectionModeAutomatic,
		nil,
	)
	result.compositionConstraint =
		QueryCompositionConstraintStepLimitExceeded

	plan := mustSelectTestPlan(
		t,
		result,
		mustSelectorTestPackState(t, nil, nil),
		DefaultLimitPerAttempt,
	)
	if plan.Decision() != PlanDecisionUnsupported ||
		len(plan.Selected()) != 0 ||
		!reflect.DeepEqual(
			plan.ReasonCodes(),
			[]ReasonCode{ReasonCodeMixedUnsupportedIntent},
		) {
		t.Fatalf(
			"SOT-MODEL-023: unsupported と step 上限制約の plan = decision:%q selected:%#v reasons:%#v",
			plan.Decision(),
			plan.Selected(),
			plan.ReasonCodes(),
		)
	}
}

func TestLegalQueryPlanはstep上限理由の部分候補と他理由を拒否する(
	t *testing.T,
) {
	t.Parallel()

	candidate := planTestCandidate(
		t,
		"candidate-step-limit",
		600,
		nil,
		planTestStep(t, "法令検索", "step-limit"),
	)
	tests := map[string]LegalQueryPlanValues{
		"部分候補": planTestValues(
			PlanDecisionNeedsClarification,
			[]LegalQueryCandidate{candidate},
			[]LegalQueryPlanSelection{
				planTestSelection(
					t,
					candidate,
					SelectionAvailabilityAvailable,
				),
			},
			[]ReasonCode{ReasonCode("step_limit_exceeded")},
		),
		"他理由との併存": planTestValues(
			PlanDecisionNeedsClarification,
			nil,
			nil,
			[]ReasonCode{
				ReasonCodeBelowExecutionThreshold,
				ReasonCode("step_limit_exceeded"),
			},
		),
	}
	for name, values := range tests {
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewLegalQueryPlan(values); err == nil {
				t.Fatal("SOT-MODEL-023: step_limit_exceeded の不正な plan を受理しました")
			}
		})
	}
}

func TestStep上限の明確化は専用質問だけを返す(t *testing.T) {
	t.Parallel()

	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionNeedsClarification,
		nil,
		nil,
		[]ReasonCode{ReasonCode("step_limit_exceeded")},
	))
	if err != nil {
		t.Fatalf("試験用 step_limit_exceeded plan を作成できません: %v", err)
	}
	result, err := AssembleLegalQueryNonExecutionResult(plan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: step 上限の明確化結果を作成できません: %v", err)
	}
	clarification, ok := result.(LegalQueryNeedsClarificationResult)
	if !ok {
		t.Fatalf("SOT-MODEL-024: result type = %T", result)
	}
	if got, want := clarification.Clarification().Questions(),
		[]LegalQueryQuestion{
			LegalQueryQuestion(
				"一度に取得する項目を4件以下に分けて指定してください。",
			),
		}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SOT-MODEL-024: questions = %#v, want %#v", got, want)
	}
	if len(result.Interpretations()) != 0 ||
		len(result.Attempts()) != 0 {
		t.Fatalf(
			"SOT-MODEL-024: step 上限で部分結果を公開しました: interpretations=%#v attempts=%#v",
			result.Interpretations(),
			result.Attempts(),
		)
	}
}

func TestStep上限の明確化は専用質問以外を拒否する(t *testing.T) {
	t.Parallel()

	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionNeedsClarification,
		nil,
		nil,
		[]ReasonCode{ReasonCode("step_limit_exceeded")},
	))
	if err != nil {
		t.Fatalf("試験用 step_limit_exceeded plan を作成できません: %v", err)
	}
	if _, err := NewLegalQueryClarification(
		plan,
		[]LegalQueryQuestion{LegalQueryQuestionTask},
	); err == nil {
		t.Fatal("SOT-MODEL-024: step 上限へ一般の明確化質問を結び付けました")
	}

	ordinaryPlan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionNeedsClarification,
		nil,
		nil,
		[]ReasonCode{ReasonCodeBelowExecutionThreshold},
	))
	if err != nil {
		t.Fatalf("試験用通常 clarification plan を作成できません: %v", err)
	}
	if _, err := NewLegalQueryClarification(
		ordinaryPlan,
		[]LegalQueryQuestion{LegalQueryQuestionStepLimitExceeded},
	); err == nil {
		t.Fatal("SOT-MODEL-024: 通常の明確化へ step 上限の専用質問を結び付けました")
	}
}
