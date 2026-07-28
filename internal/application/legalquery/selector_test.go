package legalquery

import (
	"fmt"
	"slices"
	"sync"
	"testing"
)

func TestSelectorはselectionModeを実行判断より優先する(t *testing.T) {
	t.Parallel()

	candidate := mustSelectorTestCandidate(
		t,
		"candidate-clear",
		200,
		nil,
		1,
	)
	packState := mustSelectorTestPackState(t, nil, nil)
	auto := mustSelectTestPlan(
		t,
		mustSelectorTestProfileSetResult(
			t,
			[]LegalQueryCandidate{candidate},
			nil,
			QuerySelectionModeAutomatic,
			nil,
		),
		packState,
		DefaultLimitPerAttempt,
	)
	if auto.Decision() != PlanDecisionSingle {
		t.Fatalf("auto decision = %q", auto.Decision())
	}

	clarification := mustSelectTestPlan(
		t,
		mustSelectorTestProfileSetResult(
			t,
			[]LegalQueryCandidate{candidate},
			nil,
			QuerySelectionModeClarificationRequired,
			nil,
		),
		packState,
		DefaultLimitPerAttempt,
	)
	if clarification.Decision() != PlanDecisionNeedsClarification ||
		!slices.Equal(
			clarification.ReasonCodes(),
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		) {
		t.Fatalf(
			"SOT-ARCH-025: clarification_required plan = decision:%q reasons:%#v",
			clarification.Decision(),
			clarification.ReasonCodes(),
		)
	}
	if len(clarification.Budget().StepBudgets()) != 0 {
		t.Fatal("SOT-MODEL-023: clarification_required に実行予算を割り当てました")
	}
}

func TestSelectorは最低実行閾値の境界を守る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		score         int
		wantReason    ReasonCode
		wantSelection int
	}{
		{
			score:         79,
			wantReason:    ReasonCodeBelowExecutionThreshold,
			wantSelection: 0,
		},
		{
			score:         80,
			wantReason:    ReasonCodeAmbiguousCandidates,
			wantSelection: 1,
		},
		{
			score:         81,
			wantReason:    ReasonCodeAmbiguousCandidates,
			wantSelection: 1,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("score-%d", test.score), func(t *testing.T) {
			t.Parallel()
			candidate := mustSelectorTestCandidate(
				t,
				fmt.Sprintf("candidate-%d", test.score),
				test.score,
				nil,
				1,
			)
			plan := mustSelectTestPlan(
				t,
				mustSelectorTestProfileSetResult(
					t,
					[]LegalQueryCandidate{candidate},
					nil,
					QuerySelectionModeAutomatic,
					nil,
				),
				mustSelectorTestPackState(t, nil, nil),
				DefaultLimitPerAttempt,
			)
			if plan.Decision() != PlanDecisionNeedsClarification ||
				!slices.Equal(plan.ReasonCodes(), []ReasonCode{test.wantReason}) ||
				len(plan.Selected()) != test.wantSelection {
				t.Fatalf(
					"SOT-ARCH-023: score=%d plan = decision:%q reasons:%#v selected:%d",
					test.score,
					plan.Decision(),
					plan.ReasonCodes(),
					len(plan.Selected()),
				)
			}
		})
	}
}

func TestSelectorは単独閾値の境界を守る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		score        int
		wantDecision PlanDecision
	}{
		{score: 119, wantDecision: PlanDecisionNeedsClarification},
		{score: 120, wantDecision: PlanDecisionSingle},
		{score: 121, wantDecision: PlanDecisionSingle},
	}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("score-%d", test.score), func(t *testing.T) {
			t.Parallel()
			candidate := mustSelectorTestCandidate(
				t,
				fmt.Sprintf("candidate-%d", test.score),
				test.score,
				nil,
				1,
			)
			plan := mustSelectTestPlan(
				t,
				mustSelectorTestProfileSetResult(
					t,
					[]LegalQueryCandidate{candidate},
					nil,
					QuerySelectionModeAutomatic,
					nil,
				),
				mustSelectorTestPackState(t, nil, nil),
				DefaultLimitPerAttempt,
			)
			if plan.Decision() != test.wantDecision {
				t.Fatalf(
					"SOT-ARCH-023: score=%d decision=%q; want %q",
					test.score,
					plan.Decision(),
					test.wantDecision,
				)
			}
		})
	}
}

func TestSelectorは単独marginの境界を守る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		margin       int
		wantDecision PlanDecision
	}{
		{margin: 24, wantDecision: PlanDecisionNeedsClarification},
		{margin: 25, wantDecision: PlanDecisionSingle},
		{margin: 26, wantDecision: PlanDecisionSingle},
	}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("margin-%d", test.margin), func(t *testing.T) {
			t.Parallel()
			first := mustSelectorTestCandidate(
				t,
				"candidate-first",
				130,
				nil,
				1,
			)
			second := mustSelectorTestCandidate(
				t,
				"candidate-second",
				130-test.margin,
				nil,
				1,
			)
			plan := mustSelectTestPlan(
				t,
				mustSelectorTestProfileSetResult(
					t,
					[]LegalQueryCandidate{first, second},
					nil,
					QuerySelectionModeAutomatic,
					nil,
				),
				mustSelectorTestPackState(t, nil, nil),
				DefaultLimitPerAttempt,
			)
			if plan.Decision() != test.wantDecision {
				t.Fatalf(
					"SOT-ARCH-023: margin=%d decision=%q; want %q",
					test.margin,
					plan.Decision(),
					test.wantDecision,
				)
			}
		})
	}
}

func TestSelectorはhedgeMarginの境界を守る(t *testing.T) {
	t.Parallel()

	tests := []struct {
		margin       int
		wantDecision PlanDecision
	}{
		{margin: 9, wantDecision: PlanDecisionHedged},
		{margin: 10, wantDecision: PlanDecisionHedged},
		{margin: 11, wantDecision: PlanDecisionNeedsClarification},
	}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("margin-%d", test.margin), func(t *testing.T) {
			t.Parallel()
			first := mustSelectorTestCandidate(
				t,
				"candidate-first",
				110,
				nil,
				1,
			)
			second := mustSelectorTestCandidate(
				t,
				"candidate-second",
				110-test.margin,
				nil,
				1,
			)
			pair := mustSelectorTestHedgePair(t, first, second)
			plan := mustSelectTestPlan(
				t,
				mustSelectorTestProfileSetResult(
					t,
					[]LegalQueryCandidate{first, second},
					nil,
					QuerySelectionModeAutomatic,
					[]CandidateHedgePair{pair},
				),
				mustSelectorTestPackState(t, nil, nil),
				DefaultLimitPerAttempt,
			)
			if plan.Decision() != test.wantDecision {
				t.Fatalf(
					"SOT-ARCH-023: hedge margin=%d decision=%q; want %q",
					test.margin,
					plan.Decision(),
					test.wantDecision,
				)
			}
		})
	}
}

func TestSelectorは明示された候補組だけをhedgeする(t *testing.T) {
	t.Parallel()

	first := mustSelectorTestCandidate(t, "candidate-first", 110, nil, 1)
	second := mustSelectorTestCandidate(t, "candidate-second", 105, nil, 1)
	plan := mustSelectTestPlan(
		t,
		mustSelectorTestProfileSetResult(
			t,
			[]LegalQueryCandidate{first, second},
			nil,
			QuerySelectionModeAutomatic,
			nil,
		),
		mustSelectorTestPackState(t, nil, nil),
		DefaultLimitPerAttempt,
	)
	if plan.Decision() != PlanDecisionNeedsClarification ||
		!slices.Equal(
			plan.ReasonCodes(),
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		) {
		t.Fatalf(
			"SOT-ARCH-023: 明示 hedge 組なしの plan = decision:%q reasons:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
		)
	}
}

func TestSelectorは第三の有効候補がある場合にhedgeしない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		thirdScore   int
		wantDecision PlanDecision
	}{
		{
			name:         "第三候補が閾値未満",
			thirdScore:   79,
			wantDecision: PlanDecisionHedged,
		},
		{
			name:         "第三候補が閾値と同じ",
			thirdScore:   80,
			wantDecision: PlanDecisionNeedsClarification,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			first := mustSelectorTestCandidate(t, "candidate-first", 110, nil, 1)
			second := mustSelectorTestCandidate(t, "candidate-second", 105, nil, 1)
			third := mustSelectorTestCandidate(
				t,
				"candidate-third",
				test.thirdScore,
				nil,
				1,
			)
			pair := mustSelectorTestHedgePair(t, first, second)
			plan := mustSelectTestPlan(
				t,
				mustSelectorTestProfileSetResult(
					t,
					[]LegalQueryCandidate{first, second, third},
					nil,
					QuerySelectionModeAutomatic,
					[]CandidateHedgePair{pair},
				),
				mustSelectorTestPackState(t, nil, nil),
				DefaultLimitPerAttempt,
			)
			if plan.Decision() != test.wantDecision {
				t.Fatalf(
					"SOT-ARCH-023: third score=%d decision=%q; want %q",
					test.thirdScore,
					plan.Decision(),
					test.wantDecision,
				)
			}
			if test.wantDecision == PlanDecisionNeedsClarification &&
				len(plan.Selected()) != MaxSelectedCandidates {
				t.Fatalf("明確化で示す候補数 = %d", len(plan.Selected()))
			}
		})
	}
}

func TestSelectorは複数stepの一候補をsingleとして選ぶ(t *testing.T) {
	t.Parallel()

	candidate := mustSelectorTestCandidate(
		t,
		"candidate-four-topics",
		120,
		nil,
		MaxCapabilityCalls,
	)
	plan := mustSelectTestPlan(
		t,
		mustSelectorTestProfileSetResult(
			t,
			[]LegalQueryCandidate{candidate},
			nil,
			QuerySelectionModeAutomatic,
			nil,
		),
		mustSelectorTestPackState(t, nil, nil),
		7,
	)
	if plan.Decision() != PlanDecisionSingle ||
		len(plan.Selected()) != 1 {
		t.Fatalf(
			"SOT-ARCH-025: 複数主題 plan = decision:%q selected:%d",
			plan.Decision(),
			len(plan.Selected()),
		)
	}
	budget := plan.Budget()
	if budget.CollectionStepCount() != MaxCapabilityCalls ||
		budget.ReadStepCount() != 0 ||
		len(budget.StepBudgets()) != MaxCapabilityCalls {
		t.Fatalf(
			"SOT-MODEL-023: 四 step の固定予算 = R:%d C:%d steps:%d",
			budget.ReadStepCount(),
			budget.CollectionStepCount(),
			len(budget.StepBudgets()),
		)
	}
	for index, stepBudget := range budget.StepBudgets() {
		limit, exists := stepBudget.EffectiveLimit()
		if stepBudget.CandidateID() != candidate.CandidateID() ||
			!exists ||
			limit != 7 {
			t.Fatalf("stepBudgets[%d] = %#v", index, stepBudget)
		}
	}
}

func TestSelectorは混在した対象外意図でも内部候補を保持する(t *testing.T) {
	t.Parallel()

	signals := []CandidateGenerationSignal{
		CandidateSignalUnsupportedLegalAdvice,
		CandidateSignalUnsupportedTranslation,
		CandidateSignalUnsupportedTaskOrResource,
	}
	for _, signal := range signals {
		signal := signal
		t.Run(string(signal), func(t *testing.T) {
			t.Parallel()
			candidate := mustSelectorTestCandidate(
				t,
				"candidate-supported-part",
				130,
				nil,
				1,
			)
			plan := mustSelectTestPlan(
				t,
				mustSelectorTestProfileSetResult(
					t,
					[]LegalQueryCandidate{candidate},
					[]CandidateGenerationSignal{signal},
					QuerySelectionModeAutomatic,
					nil,
				),
				mustSelectorTestPackState(t, nil, nil),
				DefaultLimitPerAttempt,
			)
			wantReasons := []ReasonCode{ReasonCodeMixedUnsupportedIntent}
			if signal == CandidateSignalUnsupportedTaskOrResource {
				wantReasons = append(
					wantReasons,
					ReasonCodeUnsupportedTaskOrResource,
				)
			}
			if plan.Decision() != PlanDecisionUnsupported ||
				!slices.Equal(plan.ReasonCodes(), wantReasons) {
				t.Fatalf(
					"SOT-MODEL-023: mixed plan = decision:%q reasons:%#v",
					plan.Decision(),
					plan.ReasonCodes(),
				)
			}
			ranked := plan.RankedCandidates()
			if len(ranked) != 1 ||
				ranked[0].CandidateID() != candidate.CandidateID() {
				t.Fatalf("SOT-MODEL-023: mixed rankedCandidates = %#v", ranked)
			}
			if len(plan.Selected()) != 0 ||
				len(plan.Budget().StepBudgets()) != 0 {
				t.Fatal("SOT-MODEL-023: mixed unsupported の候補を部分実行しました")
			}
		})
	}
}

func TestSelectorは対象外だけの照会を候補なしで返す(t *testing.T) {
	t.Parallel()

	plan := mustSelectTestPlan(
		t,
		mustSelectorTestProfileSetResult(
			t,
			nil,
			[]CandidateGenerationSignal{
				CandidateSignalUnsupportedTaskOrResource,
			},
			QuerySelectionModeAutomatic,
			nil,
		),
		mustSelectorTestPackState(t, nil, nil),
		DefaultLimitPerAttempt,
	)
	if plan.Decision() != PlanDecisionUnsupported ||
		!slices.Equal(
			plan.ReasonCodes(),
			[]ReasonCode{ReasonCodeUnsupportedTaskOrResource},
		) ||
		len(plan.RankedCandidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-023: 対象外 plan = decision:%q reasons:%#v ranked:%d",
			plan.Decision(),
			plan.ReasonCodes(),
			len(plan.RankedCandidates()),
		)
	}
}

func TestSelectorはpack状態を意味順位の後に適用する(t *testing.T) {
	t.Parallel()

	t.Run("core候補", func(t *testing.T) {
		t.Parallel()
		candidate := mustSelectorTestCandidate(
			t,
			"candidate-core",
			130,
			nil,
			1,
		)
		plan := mustSelectTestPlan(
			t,
			mustSelectorTestProfileSetResult(
				t,
				[]LegalQueryCandidate{candidate},
				nil,
				QuerySelectionModeAutomatic,
				nil,
			),
			mustSelectorTestPackState(t, nil, nil),
			DefaultLimitPerAttempt,
		)
		if plan.Decision() != PlanDecisionSingle ||
			plan.Selected()[0].Availability() != SelectionAvailabilityAvailable {
			t.Fatalf("core plan = decision:%q selected:%#v", plan.Decision(), plan.Selected())
		}
	})

	t.Run("有効な採用済みpack", func(t *testing.T) {
		t.Parallel()
		candidate := mustSelectorTestCandidate(
			t,
			"candidate-pack",
			130,
			[]string{"judicial-cases"},
			1,
		)
		plan := mustSelectTestPlan(
			t,
			mustSelectorTestProfileSetResult(
				t,
				[]LegalQueryCandidate{candidate},
				nil,
				QuerySelectionModeAutomatic,
				nil,
			),
			mustSelectorTestPackState(
				t,
				[]string{"judicial-cases"},
				[]string{"judicial-cases"},
			),
			DefaultLimitPerAttempt,
		)
		if plan.Decision() != PlanDecisionSingle ||
			plan.Selected()[0].Availability() != SelectionAvailabilityAvailable {
			t.Fatalf("enabled pack plan = decision:%q selected:%#v", plan.Decision(), plan.Selected())
		}
	})

	t.Run("無効な採用済みpack", func(t *testing.T) {
		t.Parallel()
		candidate := mustSelectorTestCandidate(
			t,
			"candidate-pack",
			130,
			[]string{"judicial-cases"},
			1,
		)
		plan := mustSelectTestPlan(
			t,
			mustSelectorTestProfileSetResult(
				t,
				[]LegalQueryCandidate{candidate},
				nil,
				QuerySelectionModeAutomatic,
				nil,
			),
			mustSelectorTestPackState(
				t,
				[]string{"judicial-cases"},
				nil,
			),
			DefaultLimitPerAttempt,
		)
		if plan.Decision() != PlanDecisionCapabilityUnavailable ||
			!slices.Equal(
				plan.ReasonCodes(),
				[]ReasonCode{ReasonCodeRequiredPackDisabled},
			) ||
			len(plan.Selected()) != 1 ||
			plan.Selected()[0].Availability() != SelectionAvailabilityPackDisabled ||
			len(plan.Budget().StepBudgets()) != 0 {
			t.Fatalf(
				"disabled pack plan = decision:%q reasons:%#v selected:%#v",
				plan.Decision(),
				plan.ReasonCodes(),
				plan.Selected(),
			)
		}
	})

	t.Run("未知のpack", func(t *testing.T) {
		t.Parallel()
		candidate := mustSelectorTestCandidate(
			t,
			"candidate-pack",
			130,
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
		if _, err := SelectLegalQueryPlan(SelectorInput{
			ProfileSetResult: result,
			PackState:        mustSelectorTestPackState(t, nil, nil),
			LimitPerAttempt:  DefaultLimitPerAttempt,
		}); err == nil {
			t.Fatal("SOT-ARCH-019: 未採用の required pack を受理しました")
		}
	})
}

func TestSelectorは無効pack候補を下位core候補へ置き換えない(t *testing.T) {
	t.Parallel()

	packCandidate := mustSelectorTestCandidate(
		t,
		"candidate-pack",
		130,
		[]string{"judicial-cases"},
		1,
	)
	coreCandidate := mustSelectorTestCandidate(
		t,
		"candidate-core",
		100,
		nil,
		1,
	)
	plan := mustSelectTestPlan(
		t,
		mustSelectorTestProfileSetResult(
			t,
			[]LegalQueryCandidate{packCandidate, coreCandidate},
			nil,
			QuerySelectionModeAutomatic,
			nil,
		),
		mustSelectorTestPackState(
			t,
			[]string{"judicial-cases"},
			nil,
		),
		DefaultLimitPerAttempt,
	)
	if plan.Decision() != PlanDecisionCapabilityUnavailable ||
		len(plan.Selected()) != 1 ||
		plan.Selected()[0].CandidateID() != packCandidate.CandidateID() {
		t.Fatalf(
			"SOT-ARCH-023: disabled 上位候補の plan = decision:%q selected:%#v",
			plan.Decision(),
			plan.Selected(),
		)
	}
}

func TestSelectorはhedge対象の一方が無効なら利用可能候補だけを実行しない(
	t *testing.T,
) {
	t.Parallel()

	coreCandidate := mustSelectorTestCandidate(
		t,
		"candidate-core",
		110,
		nil,
		1,
	)
	packCandidate := mustSelectorTestCandidate(
		t,
		"candidate-pack",
		105,
		[]string{"judicial-cases"},
		1,
	)
	pair := mustSelectorTestHedgePair(t, coreCandidate, packCandidate)
	plan := mustSelectTestPlan(
		t,
		mustSelectorTestProfileSetResult(
			t,
			[]LegalQueryCandidate{coreCandidate, packCandidate},
			nil,
			QuerySelectionModeAutomatic,
			[]CandidateHedgePair{pair},
		),
		mustSelectorTestPackState(
			t,
			[]string{"judicial-cases"},
			nil,
		),
		DefaultLimitPerAttempt,
	)
	if plan.Decision() != PlanDecisionCapabilityUnavailable ||
		len(plan.Selected()) != 1 ||
		plan.Selected()[0].CandidateID() != packCandidate.CandidateID() ||
		plan.Selected()[0].Availability() != SelectionAvailabilityPackDisabled ||
		len(plan.Budget().StepBudgets()) != 0 {
		t.Fatalf(
			"SOT-ARCH-023: 一方無効 hedge plan = decision:%q selected:%#v",
			plan.Decision(),
			plan.Selected(),
		)
	}
}

func TestSelectorは複数step候補の一部pack無効でも全体を実行しない(
	t *testing.T,
) {
	t.Parallel()

	candidate := mustSelectorTestCandidate(
		t,
		"candidate-pack-multi",
		130,
		[]string{"judicial-cases"},
		3,
	)
	plan := mustSelectTestPlan(
		t,
		mustSelectorTestProfileSetResult(
			t,
			[]LegalQueryCandidate{candidate},
			nil,
			QuerySelectionModeAutomatic,
			nil,
		),
		mustSelectorTestPackState(
			t,
			[]string{"judicial-cases"},
			nil,
		),
		DefaultLimitPerAttempt,
	)
	if plan.Decision() != PlanDecisionCapabilityUnavailable ||
		len(plan.Budget().StepBudgets()) != 0 {
		t.Fatalf(
			"SOT-MODEL-023: multi-step disabled plan = decision:%q budget:%#v",
			plan.Decision(),
			plan.Budget(),
		)
	}
}

func TestSelectorが作るplanは固定予算を変更しない(t *testing.T) {
	t.Parallel()

	candidate := mustSelectorTestCandidate(
		t,
		"candidate-budget",
		130,
		nil,
		1,
	)
	result := mustSelectorTestProfileSetResult(
		t,
		[]LegalQueryCandidate{candidate},
		nil,
		QuerySelectionModeAutomatic,
		nil,
	)
	plan := mustSelectTestPlan(
		t,
		result,
		mustSelectorTestPackState(t, nil, nil),
		6,
	)
	budget := plan.Budget()
	if plan.ProfileVersion() != result.ProfileVersion() ||
		plan.ProfileVersion() == result.RankingVersion() ||
		budget.MaxRankedCandidates() != 16 ||
		budget.MaxSelectedCandidates() != 2 ||
		budget.MaxParallelCandidates() != 2 ||
		budget.MaxCapabilityCalls() != 4 ||
		budget.MaxItemsPerCollectionStep() != 20 ||
		budget.MaxReturnedItems() != 40 ||
		!budget.FirstPageOnly() ||
		budget.LimitPerAttempt() != 6 {
		t.Fatalf(
			"SOT-MODEL-023: selector plan の profileVersion または固定予算が一致しません: %#v",
			budget,
		)
	}
}

func TestSelectorは不変なprofileSetから並行して同じplanを作る(
	t *testing.T,
) {
	t.Parallel()

	candidate := mustSelectorTestCandidate(
		t,
		"candidate-concurrent",
		130,
		nil,
		1,
	)
	profileSet, err := NewQueryProfileSet([]QueryProfile{
		selectorTestProfile{
			metadata: mustSelectorTestMetadata(
				t,
				"core",
				selectorTestProfileVersion,
				selectorTestRankingVersion,
			),
			candidates:    []LegalQueryCandidate{candidate},
			selectionMode: QuerySelectionModeAutomatic,
		},
	})
	if err != nil {
		t.Fatalf("並行試験用 profile set を作成できません: %v", err)
	}
	preprocessed := mustSelectorTestPreprocessResult(t)
	packState := mustSelectorTestPackState(t, nil, nil)
	const workerCount = 16
	errors := make(chan error, workerCount)
	var waitGroup sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			result, collectErr := profileSet.Collect(preprocessed)
			if collectErr != nil {
				errors <- collectErr
				return
			}
			plan, selectErr := SelectLegalQueryPlan(SelectorInput{
				ProfileSetResult: result,
				PackState:        packState,
				LimitPerAttempt:  DefaultLimitPerAttempt,
			})
			if selectErr != nil {
				errors <- selectErr
				return
			}
			if plan.Decision() != PlanDecisionSingle ||
				plan.ProfileVersion() != result.ProfileVersion() ||
				len(plan.Selected()) != 1 ||
				plan.Selected()[0].CandidateID() !=
					candidate.CandidateID() {
				errors <- fmt.Errorf("並行 selector の plan が一致しません")
			}
		}()
	}
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		t.Errorf("SOT-ENG-025: 並行 plan 作成に失敗しました: %v", err)
	}
}
