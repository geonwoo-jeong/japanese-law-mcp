package legalquerycorpus

import (
	"reflect"
	"strings"
	"testing"
)

func TestValidateExecutionReferencesは参照先planとaction列を受理する(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		development []SemanticCase
		execution   []ExecutionCase
	}{
		"single with default limit": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{
						caseID: "development-law-search",
						steps:  []map[string]any{validLawSearchStep()},
					},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-law-search",
						semanticCaseID: "development-law-search",
						actions: []map[string]any{
							executionReferenceTestAction(
								"law-search",
								1,
								1,
								map[string]any{"kind": "collection_success", "sourceItemCount": float64(10)},
							),
						},
						expected: executionReferenceTestResultExpectation(
							"law-search",
							1,
							"completed",
							10,
							10,
							false,
						),
					},
				),
			},
		},
		"hedged explicit limit with read and collection": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{
						caseID:   "development-hedged",
						limit:    intPointer(7),
						decision: "hedged",
						meanings: []executionReferenceTestMeaningValues{
							{
								id:    "law-read",
								steps: []map[string]any{validLawReadStep()},
							},
							{
								id:    "law-search",
								steps: []map[string]any{validLawSearchStep()},
							},
						},
						selectedMeaningIDs: []string{"law-read", "law-search"},
						reasonCodes:        []string{"hedged_close_candidates"},
					},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-hedged",
						semanticCaseID: "development-hedged",
						actions: []map[string]any{
							executionReferenceTestAction(
								"law-read",
								1,
								1,
								map[string]any{"kind": "read_success"},
							),
							executionReferenceTestAction(
								"law-search",
								1,
								2,
								map[string]any{"kind": "collection_success", "sourceItemCount": float64(7)},
							),
						},
						expected: map[string]any{
							"terminal":          "result",
							"status":            "completed",
							"returnedItemCount": float64(8),
							"attempts": []any{
								map[string]any{
									"meaningId":          "law-read",
									"stepOrdinal":        float64(1),
									"outcome":            "completed",
									"publishedItemCount": float64(1),
								},
								map[string]any{
									"meaningId":          "law-search",
									"stepOrdinal":        float64(1),
									"outcome":            "completed",
									"publishedItemCount": float64(7),
									"hasMore":            false,
								},
							},
						},
					},
				),
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := validateExecutionReferences(test.development, test.execution); err != nil {
				t.Fatalf("SOT-ENG-026: 正常な execution reference error = %v", err)
			}
		})
	}
}

func TestValidateExecutionReferencesは参照整合違反を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		development []SemanticCase
		execution   []ExecutionCase
	}{
		"semanticCaseId missing": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{caseID: "development-other", steps: []map[string]any{validLawSearchStep()}},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-law-search",
						semanticCaseID: "development-law-search",
						actions: []map[string]any{
							executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)}),
						},
						expected: executionReferenceTestResultExpectation("law-search", 1, "completed", 1, 1, false),
					},
				),
			},
		},
		"request_error semantic case": {
			development: []SemanticCase{
				executionReferenceTestRequestErrorCase(t, "development-law-search"),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-law-search",
						semanticCaseID: "development-law-search",
						actions: []map[string]any{
							executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)}),
						},
						expected: executionReferenceTestResultExpectation("law-search", 1, "completed", 1, 1, false),
					},
				),
			},
		},
		"needs clarification decision": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{
						caseID:             "development-law-search",
						decision:           "needs_clarification",
						steps:              []map[string]any{validLawSearchStep()},
						selectedMeaningIDs: []string{},
						reasonCodes:        []string{"below_execution_threshold"},
					},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-law-search",
						semanticCaseID: "development-law-search",
						actions: []map[string]any{
							executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)}),
						},
						expected: executionReferenceTestResultExpectation("law-search", 1, "completed", 1, 1, false),
					},
				),
			},
		},
		"selected meaning order mismatch": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{
						caseID:   "development-hedged",
						decision: "hedged",
						meanings: []executionReferenceTestMeaningValues{
							{id: "law-read", steps: []map[string]any{validLawReadStep()}},
							{id: "law-search", steps: []map[string]any{validLawSearchStep()}},
						},
						selectedMeaningIDs: []string{"law-read", "law-search"},
						reasonCodes:        []string{"hedged_close_candidates"},
					},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-hedged",
						semanticCaseID: "development-hedged",
						actions: []map[string]any{
							executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)}),
							executionReferenceTestAction("law-read", 1, 2, map[string]any{"kind": "read_success"}),
						},
						expected: map[string]any{
							"terminal":          "result",
							"status":            "completed",
							"returnedItemCount": float64(2),
							"attempts": []any{
								map[string]any{"meaningId": "law-search", "stepOrdinal": float64(1), "outcome": "completed", "publishedItemCount": float64(1), "hasMore": false},
								map[string]any{"meaningId": "law-read", "stepOrdinal": float64(1), "outcome": "completed", "publishedItemCount": float64(1)},
							},
						},
					},
				),
			},
		},
		"missing selected step": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{
						caseID: "development-two-steps",
						steps:  []map[string]any{validLawSearchStep(), validLawUpdatesStep()},
					},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-two-steps",
						semanticCaseID: "development-two-steps",
						actions: []map[string]any{
							executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)}),
						},
						expected: executionReferenceTestResultExpectation("law-search", 1, "completed", 1, 1, false),
					},
				),
			},
		},
		"unknown meaning reference": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{caseID: "development-law-search", steps: []map[string]any{validLawSearchStep()}},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-law-search",
						semanticCaseID: "development-law-search",
						actions: []map[string]any{
							executionReferenceTestAction("other-meaning", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)}),
						},
						expected: executionReferenceTestResultExpectation("other-meaning", 1, "completed", 1, 1, false),
					},
				),
			},
		},
		"collection outcome on read step": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{caseID: "development-law-read", steps: []map[string]any{validLawReadStep()}},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-law-read",
						semanticCaseID: "development-law-read",
						actions: []map[string]any{
							executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)}),
						},
						expected: executionReferenceTestResultExpectation("law-search", 1, "completed", 1, 1, false),
					},
				),
			},
		},
		"read outcome on collection step": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{caseID: "development-law-search", steps: []map[string]any{validLawSearchStep()}},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-law-search",
						semanticCaseID: "development-law-search",
						actions: []map[string]any{
							executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "read_success"}),
						},
						expected: map[string]any{
							"terminal":          "result",
							"status":            "completed",
							"returnedItemCount": float64(1),
							"attempts": []any{
								map[string]any{"meaningId": "law-search", "stepOrdinal": float64(1), "outcome": "completed", "publishedItemCount": float64(1)},
							},
						},
					},
				),
			},
		},
		"default limit overflow": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{caseID: "development-law-search", steps: []map[string]any{validLawSearchStep()}},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-law-search",
						semanticCaseID: "development-law-search",
						actions: []map[string]any{
							executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(11)}),
						},
						expected: executionReferenceTestResultExpectation("law-search", 1, "completed", 11, 11, false),
					},
				),
			},
		},
		"explicit limit overflow": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{
						caseID: "development-law-search",
						limit:  intPointer(7),
						steps:  []map[string]any{validLawSearchStep()},
					},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-law-search",
						semanticCaseID: "development-law-search",
						actions: []map[string]any{
							executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(8)}),
						},
						expected: executionReferenceTestResultExpectation("law-search", 1, "completed", 8, 8, false),
					},
				),
			},
		},
		"R/C effectiveLimit overflow": {
			development: []SemanticCase{
				executionReferenceTestSemanticCase(
					t,
					executionReferenceTestSemanticCaseValues{
						caseID:   "development-budget",
						limit:    intPointer(20),
						decision: "hedged",
						meanings: []executionReferenceTestMeaningValues{
							{id: "law-read", steps: []map[string]any{validLawReadStep()}},
							{id: "law-search", steps: []map[string]any{validLawSearchStep(), validLawUpdatesStep()}},
						},
						selectedMeaningIDs: []string{"law-read", "law-search"},
						reasonCodes:        []string{"hedged_close_candidates"},
					},
				),
			},
			execution: []ExecutionCase{
				executionReferenceTestExecutionCase(
					t,
					executionReferenceTestExecutionCaseValues{
						caseID:         "execution-budget",
						semanticCaseID: "development-budget",
						actions: []map[string]any{
							executionReferenceTestAction("law-read", 1, 1, map[string]any{"kind": "read_success"}),
							executionReferenceTestAction("law-search", 1, 2, map[string]any{"kind": "collection_success", "sourceItemCount": float64(20)}),
							executionReferenceTestAction("law-search", 2, 3, map[string]any{"kind": "collection_success", "sourceItemCount": float64(20)}),
						},
						expected: map[string]any{
							"terminal":          "result",
							"status":            "completed",
							"returnedItemCount": float64(40),
							"attempts": []any{
								map[string]any{"meaningId": "law-read", "stepOrdinal": float64(1), "outcome": "completed", "publishedItemCount": float64(1)},
								map[string]any{"meaningId": "law-search", "stepOrdinal": float64(1), "outcome": "completed", "publishedItemCount": float64(20), "hasMore": false},
								map[string]any{"meaningId": "law-search", "stepOrdinal": float64(2), "outcome": "completed", "publishedItemCount": float64(19), "hasMore": true},
							},
						},
					},
				),
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := validateExecutionReferences(test.development, test.execution); err == nil {
				t.Fatal("SOT-ENG-026: 不正な execution reference を受理した")
			}
		})
	}
}

func TestValidateExecutionReferencesのErrorはQueryとRef原文を含まない(t *testing.T) {
	t.Parallel()

	const secretQuery = "secret-query-must-not-leak"
	const secretResourceID = "secret-resource-id"

	development := []SemanticCase{
		executionReferenceTestSemanticCase(
			t,
			executionReferenceTestSemanticCaseValues{
				caseID:        "development-secret",
				query:         secretQuery,
				refResourceID: secretResourceID,
				steps:         []map[string]any{validLawSearchStep()},
			},
		),
	}
	execution := []ExecutionCase{
		executionReferenceTestExecutionCase(
			t,
			executionReferenceTestExecutionCaseValues{
				caseID:         "execution-secret",
				semanticCaseID: "development-secret",
				actions: []map[string]any{
					executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(11)}),
				},
				expected: executionReferenceTestResultExpectation("law-search", 1, "completed", 11, 11, false),
			},
		),
	}

	err := validateExecutionReferences(development, execution)
	if err == nil {
		t.Fatal("SOT-ENG-026: effectiveLimit 違反を受理した")
	}
	if strings.Contains(err.Error(), secretQuery) || strings.Contains(err.Error(), secretResourceID) {
		t.Fatalf("SOT-ENG-026: error が query/ref 原文を含む: %v", err)
	}
}

func TestValidateIntegrityExecutionReferencesは失敗時にZeroResultと入力不変を返す(t *testing.T) {
	t.Parallel()

	development := []SemanticCase{
		executionReferenceTestSemanticCase(
			t,
			executionReferenceTestSemanticCaseValues{caseID: "development-law-search", steps: []map[string]any{validLawSearchStep()}},
		),
	}
	execution := []ExecutionCase{
		executionReferenceTestExecutionCase(
			t,
			executionReferenceTestExecutionCaseValues{
				caseID:         "execution-law-search",
				semanticCaseID: "development-missing",
				actions: []map[string]any{
					executionReferenceTestAction("law-search", 1, 1, map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)}),
				},
				expected: executionReferenceTestResultExpectation("law-search", 1, "completed", 1, 1, false),
			},
		),
	}
	checked := integrityCheckedCorpus{
		manifest:    mustCorpusManifest(t, []string{"development-law-search"}, []string{"holdout-law-search"}, []string{"execution-law-search"}),
		development: development,
		holdout:     []SemanticCase{mustCorpusSemanticCase(t, "holdout-law-search")},
		execution:   execution,
	}
	beforeDevelopment, err := cloneCorpusSemanticCases(checked.development)
	if err != nil {
		t.Fatalf("SOT-ENG-026: development 複製 error = %v", err)
	}
	beforeExecution, err := cloneCorpusExecutionCases(checked.execution)
	if err != nil {
		t.Fatalf("SOT-ENG-026: execution 複製 error = %v", err)
	}

	got, err := validateIntegrityExecutionReferences(checked)
	if err == nil {
		t.Fatal("SOT-ENG-026: 不正な integrityCheckedCorpus を受理した")
	}
	if !reflect.DeepEqual(got, integrityCheckedCorpus{}) {
		t.Fatalf("SOT-ENG-026: 失敗時に部分結果を返した: %#v", got)
	}
	if !reflect.DeepEqual(beforeDevelopment, checked.development) || !reflect.DeepEqual(beforeExecution, checked.execution) {
		t.Fatal("SOT-ENG-026: execution reference 検証が入力を変更した")
	}
}

func TestValidateIntegrityRequirementsはHoldoutの後にExecution参照を検証する(t *testing.T) {
	t.Parallel()

	checked := integrityCheckedCorpus{
		development: []SemanticCase{
			executionReferenceTestSemanticCase(
				t,
				executionReferenceTestSemanticCaseValues{
					caseID: "development-law-search",
					steps:  []map[string]any{validLawSearchStep()},
				},
			),
		},
		holdout: holdoutRequirementsTestValidHoldout(t),
		execution: []ExecutionCase{
			executionReferenceTestExecutionCase(
				t,
				executionReferenceTestExecutionCaseValues{
					caseID:         "execution-law-search",
					semanticCaseID: "development-missing",
					actions: []map[string]any{
						executionReferenceTestAction(
							"law-search",
							1,
							1,
							map[string]any{
								"kind":            "collection_success",
								"sourceItemCount": float64(1),
							},
						),
					},
					expected: executionReferenceTestResultExpectation(
						"law-search",
						1,
						"completed",
						1,
						1,
						false,
					),
				},
			),
		},
	}

	got, err := validateIntegrityRequirements(checked)
	if err == nil {
		t.Fatal("SOT-ENG-026: execution 参照違反を受理した")
	}
	if !reflect.DeepEqual(got, integrityCheckedCorpus{}) {
		t.Fatal("SOT-ENG-026: execution 参照失敗時に部分結果を返した")
	}

	checked.holdout = checked.holdout[:minimumHoldoutCaseCount-1]
	_, err = validateIntegrityRequirements(checked)
	if err == nil || !strings.Contains(err.Error(), "holdout") {
		t.Fatalf("SOT-ENG-026: holdout を execution より先に検証しませんでした: %v", err)
	}
}

type executionReferenceTestSemanticCaseValues struct {
	caseID             string
	query              string
	refResourceID      string
	limit              *int
	decision           string
	steps              []map[string]any
	meanings           []executionReferenceTestMeaningValues
	selectedMeaningIDs []string
	reasonCodes        []string
}

type executionReferenceTestMeaningValues struct {
	id    string
	steps []map[string]any
}

type executionReferenceTestExecutionCaseValues struct {
	caseID         string
	semanticCaseID string
	actions        []map[string]any
	expected       map[string]any
}

func executionReferenceTestSemanticCase(
	t *testing.T,
	values executionReferenceTestSemanticCaseValues,
) SemanticCase {
	t.Helper()

	source := validSemanticCase(validLawSearchStep())
	source["caseId"] = values.caseID
	source["leakageGroupId"] = "group-" + values.caseID
	request := source["request"].(map[string]any)
	if values.query != "" {
		request["query"] = values.query
	}
	if values.refResourceID != "" {
		ref := validResourceRef("law")
		ref["key"].(map[string]any)["resourceId"] = values.refResourceID
		request["ref"] = ref
	}
	if values.limit == nil {
		delete(request, "limitPerAttempt")
	} else {
		request["limitPerAttempt"] = float64(*values.limit)
	}
	expected := source["expected"].(map[string]any)
	if values.decision != "" {
		expected["decision"] = values.decision
	}
	if len(values.reasonCodes) > 0 {
		expected["reasonCodes"] = stringValues(values.reasonCodes...)
	}
	if len(values.meanings) == 0 {
		expected["meanings"] = []any{map[string]any{
			"meaningId":     "law-search",
			"evidenceCodes": []any{"explicit_task", "explicit_resource"},
			"conceptIds":    []any{},
			"requiredPacks": []any{},
			"steps":         executionReferenceTestSteps(values.steps),
		}}
	} else {
		meanings := make([]any, 0, len(values.meanings))
		for _, meaning := range values.meanings {
			meanings = append(meanings, map[string]any{
				"meaningId":     meaning.id,
				"evidenceCodes": []any{"explicit_task", "explicit_resource"},
				"conceptIds":    []any{},
				"requiredPacks": []any{},
				"steps":         executionReferenceTestSteps(meaning.steps),
			})
		}
		expected["meanings"] = meanings
	}
	if len(values.selectedMeaningIDs) > 0 {
		expected["selectedMeaningIds"] = stringValues(values.selectedMeaningIDs...)
	}

	semanticCase, err := decodeSemanticCaseV1(mustJSONBytes(t, source))
	if err != nil {
		t.Fatalf("SOT-ENG-026: execution reference 用 semantic case error = %v", err)
	}
	return semanticCase
}

func executionReferenceTestRequestErrorCase(t *testing.T, caseID string) SemanticCase {
	t.Helper()

	values := validSemanticCaseValues(t)
	values.CaseID = caseID
	values.Request = mustRawRequest(t, "", nil, intPointer(20))
	values.Expected = mustExpectedRequestError(t, RequestErrorFieldQuery)
	semanticCase, err := NewSemanticCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: request_error semantic case error = %v", err)
	}
	return semanticCase
}

func executionReferenceTestExecutionCase(
	t *testing.T,
	values executionReferenceTestExecutionCaseValues,
) ExecutionCase {
	t.Helper()

	source := validExecutionCase(
		map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)},
		validResultExpectation("completed"),
	)
	source["caseId"] = values.caseID
	source["semanticCaseId"] = values.semanticCaseID
	source["actions"] = executionReferenceTestActions(values.actions)
	source["expected"] = values.expected
	executionCase, err := decodeExecutionCaseV1(mustJSONBytes(t, source))
	if err != nil {
		t.Fatalf("SOT-ENG-026: execution reference 用 execution case error = %v", err)
	}
	return executionCase
}

func executionReferenceTestAction(
	meaningID string,
	stepOrdinal int,
	releaseOrder int,
	outcome map[string]any,
) map[string]any {
	return map[string]any{
		"meaningId":    meaningID,
		"stepOrdinal":  float64(stepOrdinal),
		"releaseOrder": float64(releaseOrder),
		"outcome":      outcome,
	}
}

func executionReferenceTestResultExpectation(
	meaningID string,
	stepOrdinal int,
	status string,
	returnedItemCount int,
	publishedItemCount int,
	hasMore bool,
) map[string]any {
	return map[string]any{
		"terminal":          "result",
		"status":            status,
		"returnedItemCount": float64(returnedItemCount),
		"attempts": []any{map[string]any{
			"meaningId":          meaningID,
			"stepOrdinal":        float64(stepOrdinal),
			"outcome":            "completed",
			"publishedItemCount": float64(publishedItemCount),
			"hasMore":            hasMore,
		}},
	}
}

func executionReferenceTestSteps(steps []map[string]any) []any {
	if len(steps) == 0 {
		return []any{validLawSearchStep()}
	}
	values := make([]any, 0, len(steps))
	for _, step := range steps {
		values = append(values, step)
	}
	return values
}

func executionReferenceTestActions(actions []map[string]any) []any {
	values := make([]any, 0, len(actions))
	for _, action := range actions {
		values = append(values, action)
	}
	return values
}
