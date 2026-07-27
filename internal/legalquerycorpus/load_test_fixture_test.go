package legalquerycorpus

import (
	"fmt"
	"strings"
	"testing"
)

func loadTestWriteValidCorpus(t *testing.T) filesystemReadTestLayout {
	t.Helper()

	fixtures := make([]manifestIntegrityTestFixture, 0, 254)
	fixtures = append(fixtures, loadTestDevelopmentFixtures(t)...)
	fixtures = append(fixtures, loadTestHoldoutFixtures(t)...)
	fixtures = append(fixtures, loadTestExecutionFixtures(t)...)
	layout, _, _ := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v1",
		"",
	)
	filesystemReadTestWriteFile(
		t,
		layout.schemaPath,
		schemaRuntimeTestFixedSchema(t),
	)
	return layout
}

func loadTestDevelopmentFixtures(
	t *testing.T,
) []manifestIntegrityTestFixture {
	t.Helper()

	fixtures := make([]manifestIntegrityTestFixture, 0, 7)
	for _, scenarioID := range manifestRequiredExecutionScenarioIDs() {
		caseID := "development-" + strings.TrimPrefix(scenarioID, "execution-")
		fixtures = append(fixtures, manifestIntegrityTestFixture{
			set:    ManifestSetDevelopment,
			caseID: caseID,
			data: mustJSONBytes(
				t,
				loadTestDevelopmentSource(scenarioID, caseID),
			),
		})
	}
	return fixtures
}

func loadTestDevelopmentSource(
	scenarioID string,
	caseID string,
) map[string]any {
	source := validSemanticCase(validLawSearchStep())
	source["caseId"] = caseID
	source["leakageGroupId"] = "group-" + caseID
	source["request"].(map[string]any)["query"] = "開発照会-" + caseID

	expected := source["expected"].(map[string]any)
	expected["selectedMeaningIds"] = []any{"meaning-one"}
	meaningOne := loadTestMeaning("meaning-one", []any{validLawSearchStep()})
	expected["meanings"] = []any{meaningOne}

	switch scenarioID {
	case string(ExecutionScenarioIDNonempty):
		meaningOne["steps"] = []any{validLawReadStep()}
	case string(ExecutionScenarioIDPartialFailure):
		meaningOne["steps"] = []any{validLawReadStep(), validLawReadStep()}
	case string(ExecutionScenarioIDReversedCompletion):
		expected["decision"] = "hedged"
		expected["reasonCodes"] = []any{"hedged_close_candidates"}
		expected["selectedMeaningIds"] = []any{"meaning-one", "meaning-two"}
		expected["meanings"] = []any{
			loadTestMeaning("meaning-one", []any{validLawReadStep()}),
			loadTestMeaning("meaning-two", []any{validLawReadStep()}),
		}
	}
	return source
}

func loadTestMeaning(meaningID string, steps []any) map[string]any {
	return map[string]any{
		"meaningId":     meaningID,
		"evidenceCodes": []any{"explicit_task", "explicit_resource"},
		"conceptIds":    []any{},
		"requiredPacks": []any{},
		"steps":         steps,
	}
}

func loadTestHoldoutFixtures(t *testing.T) []manifestIntegrityTestFixture {
	t.Helper()

	fixtures := make([]manifestIntegrityTestFixture, 0, 240)
	safetyCounts := make(map[string]int)
	for _, category := range holdoutRequirementsTestCatalog() {
		for offset := 0; offset < 20; offset++ {
			index := len(fixtures)
			coverageID := category.coverages[offset%len(category.coverages)]
			source := validSemanticCase(validLawSearchStep())
			caseID := fmt.Sprintf("holdout-case-%03d", index)
			source["caseId"] = caseID
			source["leakageGroupId"] = fmt.Sprintf("holdout-group-%03d", index)
			source["coverageIds"] = []any{coverageID}
			source["request"].(map[string]any)["query"] =
				fmt.Sprintf("保持照会-%03d", index)
			if isSemanticSafetyCoverageID(coverageID) {
				variant := "ordinary"
				if safetyCounts[coverageID]%2 == 1 {
					variant = "adversarial"
				}
				source["safetyVariant"] = variant
				safetyCounts[coverageID]++
			}
			fixtures = append(fixtures, manifestIntegrityTestFixture{
				set:    ManifestSetHoldout,
				caseID: caseID,
				data:   mustJSONBytes(t, source),
			})
		}
	}
	return fixtures
}

func loadTestExecutionFixtures(
	t *testing.T,
) []manifestIntegrityTestFixture {
	t.Helper()

	fixtures := make([]manifestIntegrityTestFixture, 0, 7)
	for _, scenarioID := range manifestRequiredExecutionScenarioIDs() {
		fixtures = append(fixtures, manifestIntegrityTestFixture{
			set:    ManifestSetExecution,
			caseID: scenarioID,
			data: mustJSONBytes(
				t,
				loadTestExecutionSource(scenarioID),
			),
		})
	}
	return fixtures
}

func loadTestExecutionSource(scenarioID string) map[string]any {
	caseID := scenarioID
	semanticCaseID := "development-" +
		strings.TrimPrefix(scenarioID, "execution-")
	source := map[string]any{
		"artifactKind":   "execution_case",
		"schemaVersion":  float64(1),
		"caseId":         caseID,
		"scenarioIds":    []any{scenarioID},
		"semanticCaseId": semanticCaseID,
	}

	switch scenarioID {
	case string(ExecutionScenarioIDAllFailed):
		source["actions"] = []any{
			loadTestAction(
				"meaning-one",
				1,
				1,
				map[string]any{
					"kind":      "failure",
					"errorCode": "source_busy",
				},
			),
		}
		source["expected"] = loadTestErrorExpected(
			"source_busy",
			[]any{loadTestFailedAttempt("meaning-one", 1, "source_busy")},
		)
	case string(ExecutionScenarioIDEmpty):
		source["actions"] = []any{
			loadTestAction(
				"meaning-one",
				1,
				1,
				map[string]any{
					"kind":            "collection_success",
					"sourceItemCount": float64(0),
				},
			),
		}
		source["expected"] = loadTestResultExpected(
			"empty",
			0,
			[]any{loadTestCollectionAttempt("meaning-one", 1, 0, false, "empty")},
		)
	case string(ExecutionScenarioIDItemBudget):
		source["actions"] = []any{
			loadTestAction(
				"meaning-one",
				1,
				1,
				map[string]any{
					"kind":            "collection_success",
					"sourceItemCount": float64(1000),
				},
			),
		}
		source["expected"] = loadTestResultExpected(
			"completed",
			20,
			[]any{loadTestCollectionAttempt("meaning-one", 1, 20, true, "completed")},
		)
	case string(ExecutionScenarioIDNonempty):
		source["actions"] = []any{
			loadTestAction(
				"meaning-one",
				1,
				1,
				map[string]any{"kind": "read_success"},
			),
		}
		source["expected"] = loadTestResultExpected(
			"completed",
			1,
			[]any{loadTestReadAttempt("meaning-one", 1)},
		)
	case string(ExecutionScenarioIDPartialFailure):
		source["actions"] = []any{
			loadTestAction(
				"meaning-one",
				1,
				1,
				map[string]any{"kind": "read_success"},
			),
			loadTestAction(
				"meaning-one",
				2,
				2,
				map[string]any{
					"kind":      "failure",
					"errorCode": "source_busy",
				},
			),
		}
		source["expected"] = loadTestResultExpected(
			"partial",
			1,
			[]any{
				loadTestReadAttempt("meaning-one", 1),
				loadTestFailedAttempt("meaning-one", 2, "source_busy"),
			},
		)
	case string(ExecutionScenarioIDReversedCompletion):
		source["actions"] = []any{
			loadTestAction(
				"meaning-one",
				1,
				2,
				map[string]any{"kind": "read_success"},
			),
			loadTestAction(
				"meaning-two",
				1,
				1,
				map[string]any{"kind": "read_success"},
			),
		}
		source["expected"] = loadTestResultExpected(
			"completed",
			2,
			[]any{
				loadTestReadAttempt("meaning-one", 1),
				loadTestReadAttempt("meaning-two", 1),
			},
		)
	case string(ExecutionScenarioIDTimeout):
		source["actions"] = []any{
			loadTestAction(
				"meaning-one",
				1,
				1,
				map[string]any{"kind": "timeout"},
			),
		}
		source["expected"] = loadTestErrorExpected(
			"source_timeout",
			[]any{
				loadTestFailedAttempt(
					"meaning-one",
					1,
					"source_timeout",
				),
			},
		)
	}
	return source
}

func loadTestAction(
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

func loadTestResultExpected(
	status string,
	returnedItemCount int,
	attempts []any,
) map[string]any {
	return map[string]any{
		"terminal":          "result",
		"status":            status,
		"returnedItemCount": float64(returnedItemCount),
		"attempts":          attempts,
	}
}

func loadTestErrorExpected(errorCode string, attempts []any) map[string]any {
	return map[string]any{
		"terminal":  "error",
		"errorCode": errorCode,
		"attempts":  attempts,
	}
}

func loadTestReadAttempt(meaningID string, stepOrdinal int) map[string]any {
	return map[string]any{
		"meaningId":          meaningID,
		"stepOrdinal":        float64(stepOrdinal),
		"outcome":            "completed",
		"publishedItemCount": float64(1),
	}
}

func loadTestCollectionAttempt(
	meaningID string,
	stepOrdinal int,
	publishedItemCount int,
	hasMore bool,
	outcome string,
) map[string]any {
	return map[string]any{
		"meaningId":          meaningID,
		"stepOrdinal":        float64(stepOrdinal),
		"outcome":            outcome,
		"publishedItemCount": float64(publishedItemCount),
		"hasMore":            hasMore,
	}
}

func loadTestFailedAttempt(
	meaningID string,
	stepOrdinal int,
	errorCode string,
) map[string]any {
	return map[string]any{
		"meaningId":   meaningID,
		"stepOrdinal": float64(stepOrdinal),
		"outcome":     "failed",
		"errorCode":   errorCode,
	}
}
