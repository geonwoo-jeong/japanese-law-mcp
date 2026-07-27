package legalquerycorpus

import "testing"

func TestValidateExecutionReferencesは不正なTyped入力をErrorとして拒否する(t *testing.T) {
	t.Parallel()

	validDevelopment := executionReferenceTestSemanticCase(
		t,
		executionReferenceTestSemanticCaseValues{
			caseID: "development-law-search",
			steps:  []map[string]any{validLawSearchStep()},
		},
	)
	validExecution := executionReferenceTestExecutionCase(
		t,
		executionReferenceTestExecutionCaseValues{
			caseID:         "execution-law-search",
			semanticCaseID: "development-law-search",
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
	)

	tests := map[string]struct {
		development []SemanticCase
		execution   []ExecutionCase
	}{
		"semantic case": {
			development: []SemanticCase{{
				caseID: "development-law-search",
			}},
			execution: []ExecutionCase{validExecution},
		},
		"execution case": {
			development: []SemanticCase{validDevelopment},
			execution: []ExecutionCase{{
				caseID:         "execution-law-search",
				semanticCaseID: "development-law-search",
			}},
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := validateExecutionReferences(
				test.development,
				test.execution,
			); err == nil {
				t.Fatal("SOT-ENG-026: 不正な typed 入力を受理した")
			}
		})
	}
}
