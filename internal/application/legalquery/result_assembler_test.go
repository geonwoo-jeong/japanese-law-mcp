package legalquery

import (
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestAssembleLegalQueryExecutionResultDerivesStatusAndKeepsConstructorOrder(
	t *testing.T,
) {
	t.Parallel()

	fixtures := newAttemptPayloadFixtures(t)
	search := planTestStep(t, "法令検索", "step-search")
	content := planTestStep(t, "法令本文検索", "step-content")
	twoStepPlan := mustExecutorSinglePlan(t, search, content)
	nonempty := resultTestLawSearchAttempt(
		t,
		search,
		[]model.SourcedResource[model.LawSummary]{fixtures.lawSummary},
		true,
	)
	emptySearch := resultTestLawSearchAttempt(t, search, nil, false)
	emptyContent := resultTestLawContentEmptyAttempt(t, content)
	failedSearch := resultTestFailedAttempt(t, search)
	failedContent := resultTestFailedAttempt(t, content)

	tests := []struct {
		name     string
		plan     LegalQueryPlan
		attempts []LegalQueryAttempt
		status   LegalQueryResultStatus
		notices  []string
	}{
		{
			name:     "非空成功があれば completed",
			plan:     twoStepPlan,
			attempts: []LegalQueryAttempt{nonempty, emptyContent},
			status:   LegalQueryResultStatusCompleted,
			notices: []string{
				LegalQueryFirstPageNotice,
				LegalQuerySeparateAttemptsNotice,
			},
		},
		{
			name: "全成功が空なら empty",
			plan: mustExecutorSinglePlan(t, search),
			attempts: []LegalQueryAttempt{
				emptySearch,
			},
			status:  LegalQueryResultStatusEmpty,
			notices: []string{},
		},
		{
			name:     "非空成功と失敗なら partial",
			plan:     twoStepPlan,
			attempts: []LegalQueryAttempt{nonempty, failedContent},
			status:   LegalQueryResultStatusPartial,
			notices: []string{
				LegalQueryFirstPageNotice,
				LegalQuerySeparateAttemptsNotice,
				LegalQueryPartialFailureNotice,
			},
		},
		{
			name:     "空成功と失敗でも partial",
			plan:     twoStepPlan,
			attempts: []LegalQueryAttempt{failedSearch, emptyContent},
			status:   LegalQueryResultStatusPartial,
			notices: []string{
				LegalQuerySeparateAttemptsNotice,
				LegalQueryPartialFailureNotice,
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			execution := mustAssemblerExecution(t, test.plan, test.attempts...)
			result, err := AssembleLegalQueryExecutionResult(test.plan, execution)
			if err != nil {
				t.Fatalf("SOT-MODEL-024: result を組み立てられません: %v", err)
			}
			if result.Status() != test.status {
				t.Fatalf("SOT-MODEL-024: status = %q, want %q", result.Status(), test.status)
			}
			if !reflect.DeepEqual(result.Notices(), test.notices) {
				t.Fatalf(
					"SOT-MODEL-024: notices = %#v, want %#v",
					result.Notices(),
					test.notices,
				)
			}
			gotAttempts := result.Attempts()
			if len(gotAttempts) != len(test.attempts) {
				t.Fatalf(
					"SOT-MODEL-024: attempts = %d, want %d",
					len(gotAttempts),
					len(test.attempts),
				)
			}
			for index := range gotAttempts {
				if gotAttempts[index].StepID() != test.attempts[index].StepID() ||
					gotAttempts[index].Outcome() != test.attempts[index].Outcome() {
					t.Fatalf(
						"SOT-ARCH-023: attempts[%d] = (%q, %q), want (%q, %q)",
						index,
						gotAttempts[index].StepID(),
						gotAttempts[index].Outcome(),
						test.attempts[index].StepID(),
						test.attempts[index].Outcome(),
					)
				}
			}
		})
	}
}

func TestAssembleLegalQueryExecutionResultAcceptsHedgedPlan(t *testing.T) {
	t.Parallel()

	fixtures := newAttemptPayloadFixtures(t)
	firstStep := planTestStep(t, "法令検索", "step-first")
	secondStep := planTestStep(t, "法令本文検索", "step-second")
	plan := mustExecutorHedgedPlan(
		t,
		[]LegalQueryCandidateStep{firstStep},
		[]LegalQueryCandidateStep{secondStep},
	)
	first := resultTestLawSearchAttempt(
		t,
		firstStep,
		[]model.SourcedResource[model.LawSummary]{fixtures.lawSummary},
		false,
	)
	second, err := NewLegalQueryLawContentSearchAttempt(
		LegalQueryLawContentSearchAttemptValues{
			InterpretationID: "interpretation-2",
			Step:             secondStep,
			Page:             resultTestUnknownPage(t, 0),
			Items:            []model.SourcedResource[model.LawContentMatch]{},
		},
	)
	if err != nil {
		t.Fatalf("試験用 attempt を作成できません: %v", err)
	}
	execution := mustAssemblerExecution(t, plan, first, second)

	result, err := AssembleLegalQueryExecutionResult(plan, execution)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: hedged result を組み立てられません: %v", err)
	}
	if result.Status() != LegalQueryResultStatusCompleted ||
		result.Decision() != LegalQueryResultDecisionHedged ||
		len(result.Interpretations()) != 2 {
		t.Fatalf("SOT-MODEL-024: hedged result = %#v", result)
	}
	if got := []string{
		result.Attempts()[0].StepID(),
		result.Attempts()[1].StepID(),
	}; !reflect.DeepEqual(got, []string{"step-first", "step-second"}) {
		t.Fatalf("SOT-ARCH-023: attempt 順 = %#v", got)
	}
}

func TestLegalQueryResultAssemblersRejectWrongDirectionAndInvalidExecution(
	t *testing.T,
) {
	t.Parallel()

	step := planTestStep(t, "法令検索", "step-search")
	executingPlan := mustExecutorSinglePlan(t, step)
	execution := mustAssemblerExecution(
		t,
		executingPlan,
		resultTestLawSearchAttempt(t, step, nil, false),
	)

	for _, decision := range []PlanDecision{
		PlanDecisionSingle,
		PlanDecisionHedged,
	} {
		var plan LegalQueryPlan
		if decision == PlanDecisionSingle {
			plan = executingPlan
		} else {
			second := planTestStep(t, "法令本文検索", "step-second")
			plan = mustExecutorHedgedPlan(
				t,
				[]LegalQueryCandidateStep{step},
				[]LegalQueryCandidateStep{second},
			)
		}
		if _, err := AssembleLegalQueryNonExecutionResult(plan); err == nil {
			t.Fatalf("SOT-MODEL-024: %s plan を非実行 assembler が受理しました", decision)
		}
	}

	nonExecutingPlans := []LegalQueryPlan{
		assemblerClarificationPlan(t, nil, nil),
		assemblerCapabilityUnavailablePlan(t),
		assemblerUnsupportedPlan(t, ReasonCodeUnsupportedTaskOrResource),
	}
	for _, plan := range nonExecutingPlans {
		if _, err := AssembleLegalQueryExecutionResult(plan, execution); err == nil {
			t.Fatalf(
				"SOT-MODEL-024: %s plan を実行 assembler が受理しました",
				plan.Decision(),
			)
		}
	}

	if _, err := AssembleLegalQueryExecutionResult(
		executingPlan,
		LegalQueryExecution{},
	); err == nil {
		t.Fatal("SOT-ENG-025: zero-value execution を受理しました")
	}

	failed := resultTestFailedAttempt(t, step)
	allFailed := LegalQueryExecution{
		attempts:    []LegalQueryAttempt{failed},
		initialized: true,
	}
	if _, err := AssembleLegalQueryExecutionResult(
		executingPlan,
		allFailed,
	); err == nil {
		t.Fatal("SOT-MODEL-024: 全 failed execution artifact を受理しました")
	}
}

func TestAssembleLegalQueryExecutionResultRechecksExecutionAgainstSuppliedPlan(
	t *testing.T,
) {
	t.Parallel()

	originalStep := planTestStep(t, "法令検索", "step-search")
	originalPlan := mustExecutorSinglePlan(t, originalStep)
	execution := mustAssemblerExecution(
		t,
		originalPlan,
		resultTestLawSearchAttempt(t, originalStep, nil, false),
	)

	differentIDStep := planTestStep(t, "法令検索", "step-other")
	differentIDPlan := mustExecutorSinglePlan(t, differentIDStep)
	if _, err := AssembleLegalQueryExecutionResult(
		differentIDPlan,
		execution,
	); err == nil {
		t.Fatal("SOT-MODEL-024: stepId が異なる plan と execution を受理しました")
	}

	differentIntent, err := NewLawSearchIntentV1(
		LawSearchIntentV1Values{Query: "刑法"},
	)
	if err != nil {
		t.Fatalf("試験用 logical input を作成できません: %v", err)
	}
	differentInputValues := validStepFixtures(t)["法令検索"].values
	differentInputValues.StepID = originalStep.StepID()
	differentInputValues.LogicalInput = differentIntent
	differentInputStep, err := NewLegalQueryCandidateStep(differentInputValues)
	if err != nil {
		t.Fatalf("試験用 step を作成できません: %v", err)
	}
	differentInputPlan := mustExecutorSinglePlan(t, differentInputStep)
	if _, err := AssembleLegalQueryExecutionResult(
		differentInputPlan,
		execution,
	); err == nil {
		t.Fatal("SOT-ARCH-024: logical input が異なる plan と execution を受理しました")
	}
}

func TestAssembleLegalQueryNonExecutionResultMapsUnsupportedNotices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		reasons []ReasonCode
		notices []string
	}{
		{
			name:    "非日本語",
			reasons: []ReasonCode{ReasonCodeNonJapaneseQuery},
			notices: []string{LegalQueryNonJapaneseNotice},
		},
		{
			name:    "決定的な構造だけ",
			reasons: []ReasonCode{ReasonCodeStandaloneStructuredQuery},
			notices: []string{LegalQueryStandaloneStructuredNotice},
		},
		{
			name:    "対象外意図との混在",
			reasons: []ReasonCode{ReasonCodeMixedUnsupportedIntent},
			notices: []string{LegalQueryMixedUnsupportedNotice},
		},
		{
			name:    "未採用 task または resource",
			reasons: []ReasonCode{ReasonCodeUnsupportedTaskOrResource},
			notices: []string{LegalQueryUnsupportedScopeNotice},
		},
		{
			name: "三理由の規定順",
			reasons: []ReasonCode{
				ReasonCodeNonJapaneseQuery,
				ReasonCodeMixedUnsupportedIntent,
				ReasonCodeUnsupportedTaskOrResource,
			},
			notices: []string{
				LegalQueryNonJapaneseNotice,
				LegalQueryMixedUnsupportedNotice,
				LegalQueryUnsupportedScopeNotice,
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := AssembleLegalQueryNonExecutionResult(
				assemblerUnsupportedPlan(t, test.reasons...),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-024: unsupported result を組み立てられません: %v", err)
			}
			if result.Status() != LegalQueryResultStatusUnsupported ||
				!reflect.DeepEqual(result.Notices(), test.notices) ||
				len(result.Interpretations()) != 0 ||
				len(result.Attempts()) != 0 {
				t.Fatalf("SOT-MODEL-024: unsupported result = %#v", result)
			}
		})
	}
}

func TestAssembleLegalQueryNonExecutionResultMapsCapabilityUnavailable(
	t *testing.T,
) {
	t.Parallel()

	result, err := AssembleLegalQueryNonExecutionResult(
		assemblerCapabilityUnavailablePlan(t),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: capability unavailable を組み立てられません: %v", err)
	}
	interpretations := result.Interpretations()
	if result.Status() != LegalQueryResultStatusCapabilityUnavailable ||
		!reflect.DeepEqual(
			result.Notices(),
			[]string{LegalQueryPackDisabledNotice},
		) ||
		len(interpretations) != 1 ||
		interpretations[0].Availability() != SelectionAvailabilityPackDisabled ||
		!reflect.DeepEqual(
			interpretations[0].RequiredPacks(),
			[]string{"judicial-cases"},
		) ||
		len(result.Attempts()) != 0 {
		t.Fatalf("SOT-MODEL-024: capability unavailable result = %#v", result)
	}
}

func TestAssembleLegalQueryNonExecutionResultDerivesClarificationQuestions(
	t *testing.T,
) {
	t.Parallel()

	lawSearch := assemblerCandidate(
		t,
		"candidate-law-search",
		900,
		planTestStep(t, "法令検索", "step-law-search"),
	)
	lawRead := assemblerCandidate(
		t,
		"candidate-law-read",
		800,
		planTestStep(t, "法令読取り", "step-law-read"),
	)
	provisionSearch := assemblerCandidate(
		t,
		"candidate-provision-search",
		700,
		planTestStep(t, "法令本文検索", "step-provision-search"),
	)
	judicialSearch := assemblerCandidate(
		t,
		"candidate-judicial-search",
		600,
		planTestStep(t, "裁判例検索", "step-judicial-search"),
	)

	tests := []struct {
		name      string
		ranked    []LegalQueryCandidate
		selected  []int
		questions []LegalQueryQuestion
	}{
		{
			name: "候補なし",
			questions: []LegalQueryQuestion{
				LegalQueryQuestionTask,
				LegalQueryQuestionResource,
			},
		},
		{
			name:     "task の曖昧さ",
			ranked:   []LegalQueryCandidate{lawSearch, lawRead},
			selected: []int{0, 1},
			questions: []LegalQueryQuestion{
				LegalQueryQuestionTask,
				LegalQueryQuestionLaw,
			},
		},
		{
			name:     "resource の曖昧さと法令系 alias",
			ranked:   []LegalQueryCandidate{lawSearch, provisionSearch},
			selected: []int{0, 1},
			questions: []LegalQueryQuestion{
				LegalQueryQuestionResource,
				LegalQueryQuestionLaw,
			},
		},
		{
			name:      "法令",
			ranked:    []LegalQueryCandidate{lawSearch},
			questions: []LegalQueryQuestion{LegalQueryQuestionLaw},
		},
		{
			name:      "条文も法令質問へまとめる",
			ranked:    []LegalQueryCandidate{provisionSearch},
			questions: []LegalQueryQuestion{LegalQueryQuestionLaw},
		},
		{
			name:     "裁判例",
			ranked:   []LegalQueryCandidate{judicialSearch},
			selected: []int{0},
			questions: []LegalQueryQuestion{
				LegalQueryQuestionJudicialDecision,
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := AssembleLegalQueryNonExecutionResult(
				assemblerClarificationPlan(t, test.ranked, test.selected),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-024: clarification を組み立てられません: %v", err)
			}
			clarification, ok := result.(LegalQueryNeedsClarificationResult)
			if !ok {
				t.Fatalf("SOT-MODEL-024: result type = %T", result)
			}
			if !reflect.DeepEqual(
				clarification.Clarification().Questions(),
				test.questions,
			) {
				t.Fatalf(
					"SOT-MODEL-024: questions = %#v, want %#v",
					clarification.Clarification().Questions(),
					test.questions,
				)
			}
		})
	}
}

func TestClarificationQuestionsUseSelectedPoolAndRemainOrderedImmutableAndBounded(
	t *testing.T,
) {
	t.Parallel()

	lawSearch := assemblerCandidate(
		t,
		"candidate-law-search",
		900,
		planTestStep(t, "法令検索", "step-law-search"),
	)
	lawRead := assemblerCandidate(
		t,
		"candidate-law-read",
		800,
		planTestStep(t, "法令読取り", "step-law-read"),
	)
	judicialRead := assemblerCandidate(
		t,
		"candidate-judicial-read",
		700,
		planTestStep(t, "裁判例読取り", "step-judicial-read"),
	)

	selectedPlan := assemblerClarificationPlan(
		t,
		[]LegalQueryCandidate{lawSearch, lawRead, judicialRead},
		[]int{0},
	)
	selectedResult, err := AssembleLegalQueryNonExecutionResult(selectedPlan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: selected pool result を組み立てられません: %v", err)
	}
	if got := mustAssemblerClarificationQuestions(t, selectedResult); !reflect.DeepEqual(
		got,
		[]LegalQueryQuestion{LegalQueryQuestionLaw},
	) {
		t.Fatalf("SOT-MODEL-024: selected pool questions = %#v", got)
	}

	rankedPlan := assemblerClarificationPlan(
		t,
		[]LegalQueryCandidate{lawSearch, lawRead, judicialRead},
		nil,
	)
	rankedResult, err := AssembleLegalQueryNonExecutionResult(rankedPlan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: ranked pool result を組み立てられません: %v", err)
	}
	rankedQuestions := mustAssemblerClarificationQuestions(t, rankedResult)
	if !reflect.DeepEqual(rankedQuestions, []LegalQueryQuestion{
		LegalQueryQuestionTask,
		LegalQueryQuestionLaw,
	}) {
		t.Fatalf("SOT-MODEL-024: ranked 上位二候補の questions = %#v", rankedQuestions)
	}

	mixedPlan := assemblerClarificationPlan(
		t,
		[]LegalQueryCandidate{lawSearch, judicialRead},
		[]int{0, 1},
	)
	mixedResult, err := AssembleLegalQueryNonExecutionResult(mixedPlan)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: mixed result を組み立てられません: %v", err)
	}
	mixedQuestions := mustAssemblerClarificationQuestions(t, mixedResult)
	if !reflect.DeepEqual(mixedQuestions, []LegalQueryQuestion{
		LegalQueryQuestionTask,
		LegalQueryQuestionResource,
	}) {
		t.Fatalf("SOT-MODEL-024: questions の順序または上限 = %#v", mixedQuestions)
	}
	mixedQuestions[0] = LegalQueryQuestionJudicialDecision
	if got := mustAssemblerClarificationQuestions(t, mixedResult); !reflect.DeepEqual(
		got,
		[]LegalQueryQuestion{
			LegalQueryQuestionTask,
			LegalQueryQuestionResource,
		},
	) {
		t.Fatalf("SOT-ENG-025: questions を getter から変更できました: %#v", got)
	}
}

func mustAssemblerExecution(
	t *testing.T,
	plan LegalQueryPlan,
	attempts ...LegalQueryAttempt,
) LegalQueryExecution {
	t.Helper()
	execution, err := NewLegalQueryExecution(LegalQueryExecutionValues{
		Plan:     plan,
		Attempts: attempts,
	})
	if err != nil {
		t.Fatalf("試験用 execution を作成できません: %v", err)
	}
	return execution
}

func assemblerCandidate(
	t *testing.T,
	candidateID string,
	score int,
	steps ...LegalQueryCandidateStep,
) LegalQueryCandidate {
	t.Helper()
	requiredPacks := []string{}
	for _, step := range steps {
		if isJudicialInputKind(step.InputKind()) {
			requiredPacks = []string{"judicial-cases"}
			break
		}
	}
	return planTestCandidate(t, candidateID, score, requiredPacks, steps...)
}

func assemblerClarificationPlan(
	t *testing.T,
	ranked []LegalQueryCandidate,
	selectedIndexes []int,
) LegalQueryPlan {
	t.Helper()
	selected := make([]LegalQueryPlanSelection, 0, len(selectedIndexes))
	for _, index := range selectedIndexes {
		selected = append(
			selected,
			planTestSelection(
				t,
				ranked[index],
				SelectionAvailabilityAvailable,
			),
		)
	}
	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionNeedsClarification,
		ranked,
		selected,
		[]ReasonCode{ReasonCodeAmbiguousCandidates},
	))
	if err != nil {
		t.Fatalf("試験用 clarification plan を作成できません: %v", err)
	}
	return plan
}

func assemblerCapabilityUnavailablePlan(t *testing.T) LegalQueryPlan {
	t.Helper()
	step := planTestStep(t, "裁判例検索", "step-unavailable")
	return resultTestPlan(
		t,
		PlanDecisionCapabilityUnavailable,
		[]string{"judicial-cases"},
		SelectionAvailabilityPackDisabled,
		ReasonCodeRequiredPackDisabled,
		step,
	)
}

func assemblerUnsupportedPlan(
	t *testing.T,
	reasons ...ReasonCode,
) LegalQueryPlan {
	t.Helper()
	if len(reasons) == 1 &&
		reasons[0] == ReasonCodeStandaloneStructuredQuery {
		plan, err := NewLegalQueryPlan(planTestValues(
			PlanDecisionUnsupported,
			nil,
			nil,
			reasons,
		))
		if err != nil {
			t.Fatalf("試験用 unsupported plan を作成できません: %v", err)
		}
		return plan
	}
	candidate := assemblerCandidate(
		t,
		"candidate-unsupported",
		100,
		planTestStep(t, "法令検索", "step-unsupported"),
	)
	plan, err := NewLegalQueryPlan(planTestValues(
		PlanDecisionUnsupported,
		[]LegalQueryCandidate{candidate},
		nil,
		reasons,
	))
	if err != nil {
		t.Fatalf("試験用 unsupported plan を作成できません: %v", err)
	}
	return plan
}

func mustAssemblerClarificationQuestions(
	t *testing.T,
	result LegalQueryResult,
) []LegalQueryQuestion {
	t.Helper()
	clarification, ok := result.(LegalQueryNeedsClarificationResult)
	if !ok {
		t.Fatalf("SOT-MODEL-024: result type = %T", result)
	}
	return clarification.Clarification().Questions()
}
