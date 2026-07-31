package legalquerycorpus

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestExecutionCaseは局所的に整合するfixtureを保持する(t *testing.T) {
	t.Parallel()

	values := validExecutionCaseValuesForTest(t)
	executionCase, err := NewExecutionCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewExecutionCase() error = %v", err)
	}
	if executionCase.ArtifactKind() != ArtifactKindExecutionCase ||
		executionCase.SchemaVersion() != 1 ||
		executionCase.CaseID() != "execution-law-search" ||
		executionCase.SemanticCaseID() != "development-law-search" {
		t.Fatalf("SOT-ENG-026: execution case header = %#v", executionCase)
	}
	if got := executionCase.ScenarioIDs(); len(got) != 1 ||
		got[0] != "execution-nonempty" {
		t.Fatalf("SOT-ENG-026: scenarioIds = %#v", got)
	}
	if got := executionCase.Actions(); len(got) != 1 ||
		got[0].MeaningID() != "law-search" {
		t.Fatalf("SOT-ENG-026: actions = %#v", got)
	}
	if executionCase.Expected().Terminal() != ExecutionExpectedTerminalResult {
		t.Fatalf("SOT-ENG-026: expected = %#v", executionCase.Expected())
	}
}

func TestExecutionCaseはheaderとscenarioCatalogを検証する(t *testing.T) {
	t.Parallel()

	tests := map[string]func(ExecutionCaseValues) ExecutionCaseValues{
		"artifactKind": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.ArtifactKind = ArtifactKindSemanticCase
			return values
		},
		"schemaVersion": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.SchemaVersion = 3
			return values
		},
		"caseId prefix": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.CaseID = "development-law-search"
			return values
		},
		"caseId形式": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.CaseID = "execution_Law"
			return values
		},
		"semanticCaseId prefix": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.SemanticCaseID = "holdout-law-search"
			return values
		},
		"scenario空": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.ScenarioIDs = []string{}
			return values
		},
		"scenario未知": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.ScenarioIDs = []string{"execution-unknown"}
			return values
		},
		"scenario順序": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.ScenarioIDs = []string{
				"execution-timeout",
				"execution-all-failed",
			}
			return values
		},
		"scenario重複": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.ScenarioIDs = []string{
				"execution-nonempty",
				"execution-nonempty",
			}
			return values
		},
		"action空": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.Actions = []ExecutionAction{}
			return values
		},
		"action未初期化": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.Actions = []ExecutionAction{{}}
			return values
		},
		"expectedなし": func(values ExecutionCaseValues) ExecutionCaseValues {
			values.Expected = nil
			return values
		},
		"expected pointer": func(values ExecutionCaseValues) ExecutionCaseValues {
			expected := values.Expected.(ExecutionExpectedResult)
			values.Expected = &expected
			return values
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewExecutionCase(
				mutate(validExecutionCaseValuesForTest(t)),
			); err == nil {
				t.Fatal("SOT-ENG-026: 不正な execution case を受理した")
			}
		})
	}
	if err := (ExecutionCase{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: ExecutionCase の zero value を受理した")
	}
}

func TestExecutionCaseはactionの擬似Plan順を検証する(t *testing.T) {
	t.Parallel()

	read, _ := NewReadSuccessOutcome()
	tests := map[string][]ExecutionAction{
		"参照重複": {
			mustCaseAction(t, "meaning-one", 1, 1, read),
			mustCaseAction(t, "meaning-one", 1, 2, read),
		},
		"ordinalが1から始まらない": {
			mustCaseAction(t, "meaning-one", 2, 1, read),
		},
		"ordinalに欠番": {
			mustCaseAction(t, "meaning-one", 1, 1, read),
			mustCaseAction(t, "meaning-one", 3, 2, read),
		},
		"meaning blockが再出現": {
			mustCaseAction(t, "meaning-one", 1, 1, read),
			mustCaseAction(t, "meaning-two", 1, 2, read),
			mustCaseAction(t, "meaning-one", 2, 3, read),
		},
		"meaningが三件": {
			mustCaseAction(t, "meaning-one", 1, 1, read),
			mustCaseAction(t, "meaning-two", 1, 2, read),
			mustCaseAction(t, "meaning-three", 1, 3, read),
		},
	}
	for name, actions := range tests {
		name, actions := name, actions
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			values := executionCaseValuesForActions(
				t,
				[]string{"execution-nonempty"},
				actions,
				completedReadExpectedForActions(t, actions),
			)
			if _, err := NewExecutionCase(values); err == nil {
				t.Fatal("SOT-ENG-026: action の不正な plan 形を受理した")
			}
		})
	}
}

func TestExecutionCaseはreleaseOrderの順列とmeaning内順序を検証する(t *testing.T) {
	t.Parallel()

	read, _ := NewReadSuccessOutcome()
	tests := map[string][]ExecutionAction{
		"release重複": {
			mustCaseAction(t, "meaning-one", 1, 1, read),
			mustCaseAction(t, "meaning-one", 2, 1, read),
		},
		"release欠番": {
			mustCaseAction(t, "meaning-one", 1, 1, read),
			mustCaseAction(t, "meaning-one", 2, 3, read),
		},
		"同一meaning逆転": {
			mustCaseAction(t, "meaning-one", 1, 2, read),
			mustCaseAction(t, "meaning-one", 2, 1, read),
		},
	}
	for name, actions := range tests {
		name, actions := name, actions
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			values := executionCaseValuesForActions(
				t,
				[]string{"execution-nonempty"},
				actions,
				completedReadExpectedForActions(t, actions),
			)
			if _, err := NewExecutionCase(values); err == nil {
				t.Fatal("SOT-ENG-026: 不正な releaseOrder を受理した")
			}
		})
	}

	crossMeaning := []ExecutionAction{
		mustCaseAction(t, "meaning-one", 1, 2, read),
		mustCaseAction(t, "meaning-two", 1, 1, read),
	}
	values := executionCaseValuesForActions(
		t,
		[]string{"execution-reversed-completion"},
		crossMeaning,
		completedReadExpectedForActions(t, crossMeaning),
	)
	if _, err := NewExecutionCase(values); err != nil {
		t.Fatalf("SOT-ENG-026: meaning 間の release 逆転 error = %v", err)
	}
}

func TestExecutionCaseはactionをattemptへ正確に投影する(t *testing.T) {
	t.Parallel()

	collectionZero, _ := NewCollectionSuccessOutcome(0)
	collectionFive, _ := NewCollectionSuccessOutcome(5)
	read, _ := NewReadSuccessOutcome()
	failure, _ := NewFailureOutcome(model.ErrorCodeSourceBusy)
	timeout, _ := NewTimeoutOutcome()

	tests := []struct {
		name      string
		scenarios []string
		action    ExecutionAction
		expected  ExecutionExpected
	}{
		{
			name:      "collection zero",
			scenarios: []string{"execution-empty"},
			action:    mustCaseAction(t, "meaning-one", 1, 1, collectionZero),
			expected: mustCaseExpectedResult(
				t,
				legalquery.LegalQueryResultStatusEmpty,
				0,
				[]ExpectedAttempt{mustEmptyAttempt(t, "meaning-one", 1)},
			),
		},
		{
			name:      "collection nonempty",
			scenarios: []string{"execution-nonempty"},
			action:    mustCaseAction(t, "meaning-one", 1, 1, collectionFive),
			expected: mustCaseExpectedResult(
				t,
				legalquery.LegalQueryResultStatusCompleted,
				3,
				[]ExpectedAttempt{
					mustCompletedCollectionAttempt(t, "meaning-one", 1, 3, true),
				},
			),
		},
		{
			name:      "read",
			scenarios: []string{"execution-nonempty"},
			action:    mustCaseAction(t, "meaning-one", 1, 1, read),
			expected: mustCaseExpectedResult(
				t,
				legalquery.LegalQueryResultStatusCompleted,
				1,
				[]ExpectedAttempt{mustCompletedReadAttempt(t, "meaning-one", 1)},
			),
		},
		{
			name:      "failure",
			scenarios: []string{"execution-all-failed"},
			action:    mustCaseAction(t, "meaning-one", 1, 1, failure),
			expected: mustCaseExpectedError(
				t,
				model.ErrorCodeSourceBusy,
				[]ExpectedAttempt{
					mustFailedAttempt(t, "meaning-one", 1, model.ErrorCodeSourceBusy),
				},
			),
		},
		{
			name:      "timeout",
			scenarios: []string{"execution-timeout"},
			action:    mustCaseAction(t, "meaning-one", 1, 1, timeout),
			expected: mustCaseExpectedError(
				t,
				model.ErrorCodeSourceTimeout,
				[]ExpectedAttempt{
					mustFailedAttempt(t, "meaning-one", 1, model.ErrorCodeSourceTimeout),
				},
			),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			values := executionCaseValuesForActions(
				t,
				test.scenarios,
				[]ExecutionAction{test.action},
				test.expected,
			)
			if _, err := NewExecutionCase(values); err != nil {
				t.Fatalf("SOT-ENG-026: action 投影 error = %v", err)
			}
		})
	}
}

func TestExecutionCaseはactionとattemptの不整合を拒否する(t *testing.T) {
	t.Parallel()

	collectionZero, _ := NewCollectionSuccessOutcome(0)
	collectionFive, _ := NewCollectionSuccessOutcome(5)
	read, _ := NewReadSuccessOutcome()
	failure, _ := NewFailureOutcome(model.ErrorCodeSourceBusy)
	timeout, _ := NewTimeoutOutcome()

	tests := []struct {
		name     string
		action   ExecutionAction
		attempts []ExpectedAttempt
	}{
		{
			name:   "meaningId",
			action: mustCaseAction(t, "meaning-one", 1, 1, read),
			attempts: []ExpectedAttempt{
				mustCompletedReadAttempt(t, "meaning-two", 1),
			},
		},
		{
			name:   "stepOrdinal",
			action: mustCaseAction(t, "meaning-one", 1, 1, read),
			attempts: []ExpectedAttempt{
				mustCompletedReadAttempt(t, "meaning-one", 2),
			},
		},
		{
			name:   "collection zero",
			action: mustCaseAction(t, "meaning-one", 1, 1, collectionZero),
			attempts: []ExpectedAttempt{
				mustCompletedCollectionAttempt(t, "meaning-one", 1, 1, false),
			},
		},
		{
			name:   "collection source count",
			action: mustCaseAction(t, "meaning-one", 1, 1, collectionFive),
			attempts: []ExpectedAttempt{
				mustCompletedCollectionAttempt(t, "meaning-one", 1, 6, false),
			},
		},
		{
			name:   "collection hasMore",
			action: mustCaseAction(t, "meaning-one", 1, 1, collectionFive),
			attempts: []ExpectedAttempt{
				mustCompletedCollectionAttempt(t, "meaning-one", 1, 3, false),
			},
		},
		{
			name:   "read shape",
			action: mustCaseAction(t, "meaning-one", 1, 1, read),
			attempts: []ExpectedAttempt{
				mustCompletedCollectionAttempt(t, "meaning-one", 1, 1, false),
			},
		},
		{
			name:   "failure code",
			action: mustCaseAction(t, "meaning-one", 1, 1, failure),
			attempts: []ExpectedAttempt{
				mustFailedAttempt(t, "meaning-one", 1, model.ErrorCodeInternalError),
			},
		},
		{
			name:   "timeout code",
			action: mustCaseAction(t, "meaning-one", 1, 1, timeout),
			attempts: []ExpectedAttempt{
				mustFailedAttempt(t, "meaning-one", 1, model.ErrorCodeSourceBusy),
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			expected := expectedForPossiblyFailedAttempts(t, test.attempts)
			values := executionCaseValuesForActions(
				t,
				[]string{"execution-nonempty"},
				[]ExecutionAction{test.action},
				expected,
			)
			if _, err := NewExecutionCase(values); err == nil {
				t.Fatal("SOT-ENG-026: action と attempt の不整合を受理した")
			}
		})
	}

	action := mustCaseAction(t, "meaning-one", 1, 1, read)
	expected := mustCaseExpectedResult(
		t,
		legalquery.LegalQueryResultStatusCompleted,
		2,
		[]ExpectedAttempt{
			mustCompletedReadAttempt(t, "meaning-one", 1),
			mustCompletedReadAttempt(t, "meaning-one", 2),
		},
	)
	values := executionCaseValuesForActions(
		t,
		[]string{"execution-nonempty"},
		[]ExecutionAction{action},
		expected,
	)
	if _, err := NewExecutionCase(values); err == nil {
		t.Fatal("SOT-ENG-026: action と attempt の件数不一致を受理した")
	}
}

func TestExecutionCaseは八Scenarioの局所必要条件を検証する(t *testing.T) {
	t.Parallel()

	for name, values := range validExecutionScenarioCases(t) {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewExecutionCase(values); err != nil {
				t.Fatalf("SOT-ENG-026: scenario の局所条件 error = %v", err)
			}
		})
	}

	// semanticCaseId の実在、single/hedged decision、選択 meaning と実 step、
	// effectiveLimit は参照先 SemanticCase が必要なため、この単体では検証しない。
	for name, values := range invalidExecutionScenarioCases(t) {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewExecutionCase(values); err == nil {
				t.Fatal("SOT-ENG-026: scenario の局所必要条件違反を受理した")
			}
		})
	}
}

func validExecutionCaseValuesForTest(t *testing.T) ExecutionCaseValues {
	t.Helper()
	outcome, _ := NewCollectionSuccessOutcome(1)
	action := mustCaseAction(t, "law-search", 1, 1, outcome)
	expected := mustCaseExpectedResult(
		t,
		legalquery.LegalQueryResultStatusCompleted,
		1,
		[]ExpectedAttempt{
			mustCompletedCollectionAttempt(t, "law-search", 1, 1, false),
		},
	)
	return executionCaseValuesForActions(
		t,
		[]string{"execution-nonempty"},
		[]ExecutionAction{action},
		expected,
	)
}

func executionCaseValuesForActions(
	t *testing.T,
	scenarios []string,
	actions []ExecutionAction,
	expected ExecutionExpected,
) ExecutionCaseValues {
	t.Helper()
	return ExecutionCaseValues{
		ArtifactKind:   ArtifactKindExecutionCase,
		SchemaVersion:  1,
		CaseID:         "execution-law-search",
		ScenarioIDs:    scenarios,
		SemanticCaseID: "development-law-search",
		Actions:        actions,
		Expected:       expected,
	}
}

func mustCaseAction(
	t *testing.T,
	meaningID string,
	stepOrdinal int,
	releaseOrder int,
	outcome ExecutionOutcome,
) ExecutionAction {
	t.Helper()
	action, err := NewExecutionAction(ExecutionActionValues{
		MeaningID:    meaningID,
		StepOrdinal:  stepOrdinal,
		ReleaseOrder: releaseOrder,
		Outcome:      outcome,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: ExecutionAction を作成できません: %v", err)
	}
	return action
}

func mustCaseExpectedResult(
	t *testing.T,
	status legalquery.LegalQueryResultStatus,
	returned int,
	attempts []ExpectedAttempt,
) ExecutionExpectedResult {
	t.Helper()
	expected, err := NewExecutionExpectedResult(ExecutionExpectedResultValues{
		Status:            status,
		ReturnedItemCount: returned,
		Attempts:          attempts,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: ExecutionExpectedResult を作成できません: %v", err)
	}
	return expected
}

func mustCaseExpectedError(
	t *testing.T,
	code model.ErrorCode,
	attempts []ExpectedAttempt,
) ExecutionExpectedError {
	t.Helper()
	expected, err := NewExecutionExpectedError(ExecutionExpectedErrorValues{
		ErrorCode: code,
		Attempts:  attempts,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: ExecutionExpectedError を作成できません: %v", err)
	}
	return expected
}

func completedReadExpectedForActions(
	t *testing.T,
	actions []ExecutionAction,
) ExecutionExpectedResult {
	t.Helper()
	attempts := make([]ExpectedAttempt, 0, len(actions))
	for _, action := range actions {
		attempts = append(
			attempts,
			mustCompletedReadAttempt(t, action.MeaningID(), action.StepOrdinal()),
		)
	}
	return mustCaseExpectedResult(
		t,
		legalquery.LegalQueryResultStatusCompleted,
		len(attempts),
		attempts,
	)
}

func expectedForPossiblyFailedAttempts(
	t *testing.T,
	attempts []ExpectedAttempt,
) ExecutionExpected {
	t.Helper()
	if len(attempts) == 1 {
		if failed, ok := attempts[0].(ExpectedFailedAttempt); ok {
			return mustCaseExpectedError(t, failed.ErrorCode(), attempts)
		}
	}
	return mustCaseExpectedResult(
		t,
		legalquery.LegalQueryResultStatusCompleted,
		publishedCountForCaseAttempts(attempts),
		attempts,
	)
}

func publishedCountForCaseAttempts(attempts []ExpectedAttempt) int {
	count := 0
	for _, attempt := range attempts {
		switch typed := attempt.(type) {
		case ExpectedCompletedReadAttempt:
			count += typed.PublishedItemCount()
		case ExpectedCompletedCollectionAttempt:
			count += typed.PublishedItemCount()
		}
	}
	return count
}

func validExecutionScenarioCases(
	t *testing.T,
) map[string]ExecutionCaseValues {
	t.Helper()
	collectionZero, _ := NewCollectionSuccessOutcome(0)
	collectionLarge, _ := NewCollectionSuccessOutcome(1000)
	read, _ := NewReadSuccessOutcome()
	failure, _ := NewFailureOutcome(model.ErrorCodeSourceBusy)
	timeout, _ := NewTimeoutOutcome()

	nonemptyAction := mustCaseAction(t, "meaning-one", 1, 1, read)
	emptyAction := mustCaseAction(t, "meaning-one", 1, 1, collectionZero)
	failedAction := mustCaseAction(t, "meaning-one", 1, 1, failure)
	timeoutAction := mustCaseAction(t, "meaning-one", 1, 1, timeout)
	partialActions := []ExecutionAction{
		mustCaseAction(t, "meaning-one", 1, 1, read),
		mustCaseAction(t, "meaning-one", 2, 2, failure),
	}
	reversedActions := []ExecutionAction{
		mustCaseAction(t, "meaning-one", 1, 2, read),
		mustCaseAction(t, "meaning-two", 1, 1, read),
	}
	budgetAction := mustCaseAction(t, "meaning-one", 1, 1, collectionLarge)
	mixedActions := []ExecutionAction{
		mustCaseAction(t, "meaning-one", 1, 1, read),
		mustCaseAction(t, "meaning-one", 2, 2, collectionLarge),
	}

	return map[string]ExecutionCaseValues{
		"execution-nonempty": executionCaseValuesForActions(
			t,
			[]string{"execution-nonempty"},
			[]ExecutionAction{nonemptyAction},
			completedReadExpectedForActions(t, []ExecutionAction{nonemptyAction}),
		),
		"execution-empty": executionCaseValuesForActions(
			t,
			[]string{"execution-empty"},
			[]ExecutionAction{emptyAction},
			mustCaseExpectedResult(
				t,
				legalquery.LegalQueryResultStatusEmpty,
				0,
				[]ExpectedAttempt{mustEmptyAttempt(t, "meaning-one", 1)},
			),
		),
		"execution-partial-failure": executionCaseValuesForActions(
			t,
			[]string{"execution-partial-failure"},
			partialActions,
			mustCaseExpectedResult(
				t,
				legalquery.LegalQueryResultStatusPartial,
				1,
				[]ExpectedAttempt{
					mustCompletedReadAttempt(t, "meaning-one", 1),
					mustFailedAttempt(t, "meaning-one", 2, model.ErrorCodeSourceBusy),
				},
			),
		),
		"execution-all-failed": executionCaseValuesForActions(
			t,
			[]string{"execution-all-failed"},
			[]ExecutionAction{failedAction},
			mustCaseExpectedError(
				t,
				model.ErrorCodeSourceBusy,
				[]ExpectedAttempt{
					mustFailedAttempt(t, "meaning-one", 1, model.ErrorCodeSourceBusy),
				},
			),
		),
		"execution-timeout": executionCaseValuesForActions(
			t,
			[]string{"execution-timeout"},
			[]ExecutionAction{timeoutAction},
			mustCaseExpectedError(
				t,
				model.ErrorCodeSourceTimeout,
				[]ExpectedAttempt{
					mustFailedAttempt(t, "meaning-one", 1, model.ErrorCodeSourceTimeout),
				},
			),
		),
		"execution-reversed-completion": executionCaseValuesForActions(
			t,
			[]string{"execution-reversed-completion"},
			reversedActions,
			completedReadExpectedForActions(t, reversedActions),
		),
		"execution-item-budget": executionCaseValuesForActions(
			t,
			[]string{"execution-item-budget"},
			[]ExecutionAction{budgetAction},
			mustCaseExpectedResult(
				t,
				legalquery.LegalQueryResultStatusCompleted,
				20,
				[]ExpectedAttempt{
					mustCompletedCollectionAttempt(t, "meaning-one", 1, 20, true),
				},
			),
		),
		"execution-mixed-composition": executionCaseValuesForActions(
			t,
			[]string{"execution-mixed-composition"},
			mixedActions,
			mustCaseExpectedResult(
				t,
				legalquery.LegalQueryResultStatusCompleted,
				11,
				[]ExpectedAttempt{
					mustCompletedReadAttempt(t, "meaning-one", 1),
					mustCompletedCollectionAttempt(
						t,
						"meaning-one",
						2,
						10,
						true,
					),
				},
			),
		),
	}
}

func invalidExecutionScenarioCases(
	t *testing.T,
) map[string]ExecutionCaseValues {
	t.Helper()
	valid := validExecutionScenarioCases(t)
	read, _ := NewReadSuccessOutcome()
	collectionOne, _ := NewCollectionSuccessOutcome(1)
	nonReversedActions := []ExecutionAction{
		mustCaseAction(t, "meaning-one", 1, 1, read),
		mustCaseAction(t, "meaning-two", 1, 2, read),
	}
	untrimmedBudgetAction := mustCaseAction(
		t,
		"meaning-one",
		1,
		1,
		collectionOne,
	)
	return map[string]ExecutionCaseValues{
		"nonemptyにitemなし": func() ExecutionCaseValues {
			values := valid["execution-empty"]
			values.ScenarioIDs = []string{"execution-nonempty"}
			return values
		}(),
		"nonemptyがerror終端": func() ExecutionCaseValues {
			values := valid["execution-all-failed"]
			values.ScenarioIDs = []string{"execution-nonempty"}
			return values
		}(),
		"emptyに非空成功": func() ExecutionCaseValues {
			values := valid["execution-nonempty"]
			values.ScenarioIDs = []string{"execution-empty"}
			return values
		}(),
		"partialに失敗なし": func() ExecutionCaseValues {
			values := valid["execution-nonempty"]
			values.ScenarioIDs = []string{"execution-partial-failure"}
			return values
		}(),
		"partialに成功なし": func() ExecutionCaseValues {
			values := valid["execution-all-failed"]
			values.ScenarioIDs = []string{"execution-partial-failure"}
			return values
		}(),
		"all-failedに成功": func() ExecutionCaseValues {
			values := valid["execution-nonempty"]
			values.ScenarioIDs = []string{"execution-all-failed"}
			return values
		}(),
		"timeoutにtimeoutなし": func() ExecutionCaseValues {
			values := valid["execution-all-failed"]
			values.ScenarioIDs = []string{"execution-timeout"}
			return values
		}(),
		"reversedにmeaning間逆転なし": func() ExecutionCaseValues {
			return executionCaseValuesForActions(
				t,
				[]string{"execution-reversed-completion"},
				nonReversedActions,
				completedReadExpectedForActions(t, nonReversedActions),
			)
		}(),
		"item-budgetに切詰めなし": func() ExecutionCaseValues {
			return executionCaseValuesForActions(
				t,
				[]string{"execution-item-budget"},
				[]ExecutionAction{untrimmedBudgetAction},
				mustCaseExpectedResult(
					t,
					legalquery.LegalQueryResultStatusCompleted,
					1,
					[]ExpectedAttempt{
						mustCompletedCollectionAttempt(
							t,
							"meaning-one",
							1,
							1,
							false,
						),
					},
				),
			)
		}(),
		"mixed-compositionにcollectionなし": func() ExecutionCaseValues {
			values := valid["execution-nonempty"]
			values.ScenarioIDs = []string{"execution-mixed-composition"}
			return values
		}(),
	}
}
