package legalquerycorpus

import (
	"reflect"
	"testing"
)

func TestExecutionCaseV1Decodeは局所的に整合するfixtureを復元する(t *testing.T) {
	t.Parallel()

	executionCase, err := decodeExecutionCaseV1(
		mustJSONBytes(t, validExecutionCaseV1ForModelTest()),
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: execution case decode error = %v", err)
	}
	if executionCase.ArtifactKind() != ArtifactKindExecutionCase ||
		executionCase.SchemaVersion() != 1 ||
		executionCase.CaseID() != "execution-law-search" ||
		executionCase.SemanticCaseID() != "development-law-search" {
		t.Fatalf("SOT-ENG-026: execution case = %#v", executionCase)
	}
	if got := executionCase.Actions(); len(got) != 1 ||
		got[0].Outcome().Kind() != ExecutionOutcomeKindCollectionSuccess {
		t.Fatalf("SOT-ENG-026: actions = %#v", got)
	}
	if executionCase.Expected().Terminal() != ExecutionExpectedTerminalResult {
		t.Fatalf("SOT-ENG-026: expected = %#v", executionCase.Expected())
	}
}

func TestExecutionCaseV1Decodeは必須項目を拒否する(t *testing.T) {
	t.Parallel()

	required := []string{
		"artifactKind",
		"schemaVersion",
		"caseId",
		"scenarioIds",
		"semanticCaseId",
		"actions",
		"expected",
	}
	for _, field := range required {
		field := field
		t.Run(field+"欠落", func(t *testing.T) {
			t.Parallel()
			source := validExecutionCaseV1ForModelTest()
			delete(source, field)
			if _, err := decodeExecutionCaseV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 必須項目欠落を受理した")
			}
		})
	}
}

func TestExecutionCaseV1Decodeは未知項目と末尾値を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]func(map[string]any){
		"top-level": func(source map[string]any) {
			source["checksum"] = "forbidden"
		},
		"action": func(source map[string]any) {
			firstExecutionActionV1ForTest(source)["providerId"] = "forbidden"
		},
		"outcome": func(source map[string]any) {
			firstExecutionActionV1ForTest(source)["outcome"].(map[string]any)["response"] = "forbidden"
		},
		"expected": func(source map[string]any) {
			source["expected"].(map[string]any)["notices"] = []any{}
		},
		"attempt": func(source map[string]any) {
			firstExecutionAttemptV1ForTest(source)["stepId"] = "forbidden"
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source := validExecutionCaseV1ForModelTest()
			mutate(source)
			if _, err := decodeExecutionCaseV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 未知項目を受理した")
			}
		})
	}

	trailing := append(
		mustJSONBytes(t, validExecutionCaseV1ForModelTest()),
		[]byte(` {}`)...,
	)
	if _, err := decodeExecutionCaseV1(trailing); err == nil {
		t.Fatal("SOT-ENG-026: execution case の末尾値を受理した")
	}
}

func TestExecutionCaseV1Decodeはactionとattemptの入力順を保持する(t *testing.T) {
	t.Parallel()

	source := map[string]any{
		"artifactKind":   "execution_case",
		"schemaVersion":  float64(1),
		"caseId":         "execution-reversed-read",
		"scenarioIds":    []any{"execution-reversed-completion"},
		"semanticCaseId": "development-reversed-read",
		"actions": []any{
			map[string]any{
				"meaningId": "meaning-one", "stepOrdinal": float64(1),
				"releaseOrder": float64(2),
				"outcome":      map[string]any{"kind": "read_success"},
			},
			map[string]any{
				"meaningId": "meaning-two", "stepOrdinal": float64(1),
				"releaseOrder": float64(1),
				"outcome":      map[string]any{"kind": "read_success"},
			},
		},
		"expected": map[string]any{
			"terminal":          "result",
			"status":            "completed",
			"returnedItemCount": float64(2),
			"attempts": []any{
				completedReadAttemptV1ForTest("meaning-one", 1),
				completedReadAttemptV1ForTest("meaning-two", 1),
			},
		},
	}
	executionCase, err := decodeExecutionCaseV1(mustJSONBytes(t, source))
	if err != nil {
		t.Fatalf("SOT-ENG-026: reversed fixture decode error = %v", err)
	}
	actions := executionCase.Actions()
	attempts := executionCase.Expected().Attempts()
	if actions[0].MeaningID() != "meaning-one" ||
		actions[1].MeaningID() != "meaning-two" ||
		attempts[0].MeaningID() != "meaning-one" ||
		attempts[1].MeaningID() != "meaning-two" {
		t.Fatal("SOT-ENG-026: action または attempt の入力順が変更された")
	}
}

func TestExecutionCaseV1Decodeはscenario順序と局所投影を検証する(t *testing.T) {
	t.Parallel()

	timeout := validTimeoutExecutionCaseV1ForTest()
	timeout["scenarioIds"] = []any{
		"execution-all-failed",
		"execution-timeout",
	}
	if _, err := decodeExecutionCaseV1(mustJSONBytes(t, timeout)); err != nil {
		t.Fatalf("SOT-ENG-026: 昇順 scenarioIds error = %v", err)
	}

	unsorted := validTimeoutExecutionCaseV1ForTest()
	unsorted["scenarioIds"] = []any{
		"execution-timeout",
		"execution-all-failed",
	}
	if _, err := decodeExecutionCaseV1(mustJSONBytes(t, unsorted)); err == nil {
		t.Fatal("SOT-ENG-026: 非昇順 scenarioIds を受理した")
	}

	mismatch := validExecutionCaseV1ForModelTest()
	firstExecutionAttemptV1ForTest(mismatch)["meaningId"] = "other-meaning"
	if _, err := decodeExecutionCaseV1(mustJSONBytes(t, mismatch)); err == nil {
		t.Fatal("SOT-ENG-026: action と attempt の参照不一致を受理した")
	}
}

func TestExecutionCase公開型は動的JSONを保持しない(t *testing.T) {
	t.Parallel()

	for _, current := range []reflect.Type{
		reflect.TypeOf(ExecutionCase{}),
		reflect.TypeOf(ExecutionCaseValues{}),
	} {
		assertNoPublicDynamicJSONType(t, current, map[reflect.Type]bool{})
	}
}

func validExecutionCaseV1ForModelTest() map[string]any {
	return validExecutionCase(
		map[string]any{
			"kind":            "collection_success",
			"sourceItemCount": float64(1),
		},
		validResultExpectation("completed"),
	)
}

func validTimeoutExecutionCaseV1ForTest() map[string]any {
	return map[string]any{
		"artifactKind":   "execution_case",
		"schemaVersion":  float64(1),
		"caseId":         "execution-timeout",
		"scenarioIds":    []any{"execution-timeout"},
		"semanticCaseId": "development-timeout",
		"actions": []any{map[string]any{
			"meaningId":    "meaning-one",
			"stepOrdinal":  float64(1),
			"releaseOrder": float64(1),
			"outcome":      map[string]any{"kind": "timeout"},
		}},
		"expected": map[string]any{
			"terminal":  "error",
			"errorCode": "source_timeout",
			"attempts": []any{map[string]any{
				"meaningId":   "meaning-one",
				"stepOrdinal": float64(1),
				"outcome":     "failed",
				"errorCode":   "source_timeout",
			}},
		},
	}
}

func completedReadAttemptV1ForTest(
	meaningID string,
	stepOrdinal int,
) map[string]any {
	return map[string]any{
		"meaningId":          meaningID,
		"stepOrdinal":        float64(stepOrdinal),
		"outcome":            "completed",
		"publishedItemCount": float64(1),
	}
}

func firstExecutionActionV1ForTest(source map[string]any) map[string]any {
	return source["actions"].([]any)[0].(map[string]any)
}

func firstExecutionAttemptV1ForTest(source map[string]any) map[string]any {
	return source["expected"].(map[string]any)["attempts"].([]any)[0].(map[string]any)
}
