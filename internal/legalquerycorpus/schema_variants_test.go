package legalquerycorpus

import "testing"

func TestCorpusSchemaV1は全PlanDecisionを受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)

	hedged := validSemanticCase(validLawSearchStep())
	hedgedExpected := hedged["expected"].(map[string]any)
	hedgedExpected["decision"] = "hedged"
	hedgedExpected["reasonCodes"] = []any{"hedged_close_candidates"}
	hedgedExpected["meanings"] = append(
		hedgedExpected["meanings"].([]any),
		validExpectedMeaning("law-content-search", validLawContentSearchStep()),
	)
	hedgedExpected["selectedMeaningIds"] = []any{"law-search", "law-content-search"}

	needsClarification := validSemanticCase(validLawSearchStep())
	needsExpected := needsClarification["expected"].(map[string]any)
	needsExpected["decision"] = "needs_clarification"
	needsExpected["reasonCodes"] = []any{
		"below_execution_threshold",
		"ambiguous_candidates",
	}
	needsExpected["selectedMeaningIds"] = []any{}

	capabilityUnavailable := validSemanticCase(validJudicialSearchStep())
	capabilityExpected := capabilityUnavailable["expected"].(map[string]any)
	capabilityExpected["decision"] = "capability_unavailable"
	capabilityExpected["reasonCodes"] = []any{"required_pack_disabled"}
	capabilityExpected["selectedMeaningIds"] = []any{"judicial-search"}
	capabilityMeaning := firstExpectedMeaning(capabilityUnavailable)
	capabilityMeaning["meaningId"] = "judicial-search"
	capabilityMeaning["requiredPacks"] = []any{"judicial-cases"}

	for name, instance := range map[string]map[string]any{
		"hedged":                 hedged,
		"needs_clarification":    needsClarification,
		"capability_unavailable": capabilityUnavailable,
	} {
		name, instance := name, instance
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertSchemaAccepts(t, schema, instance)
		})
	}
}

func TestCorpusSchemaV1はLawReadの代替入力を受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)

	byIDAndDate := validSemanticCase(validLawReadStep())
	firstSemanticStep(byIDAndDate)["logicalInput"] = map[string]any{
		"lawId": "503AC0000000037",
		"asOf":  "2026-07-28",
	}

	byRef := validSemanticCase(validLawReadStep())
	firstSemanticStep(byRef)["logicalInput"] = map[string]any{
		"ref": validResourceRef("law"),
	}

	assertSchemaAccepts(t, schema, byIDAndDate)
	assertSchemaAccepts(t, schema, byRef)
}

func TestCorpusSchemaV1はLawArticleReadの代替入力を受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)

	versionless := validSemanticCase(validLawArticleReadStep())
	versionlessStep := firstSemanticStep(versionless)
	versionlessLocation := versionlessStep["logicalInput"].(map[string]any)["location"]
	versionlessStep["logicalInput"] = map[string]any{
		"ref":      validResourceRef("law"),
		"location": versionlessLocation,
		"asOf":     "2026-07-28",
	}

	versioned := validSemanticCase(validLawArticleReadStep())
	versionedStep := firstSemanticStep(versioned)
	versionedLocation := versionedStep["logicalInput"].(map[string]any)["location"]
	versionedRef := validResourceRef("law")
	versionedRef["key"].(map[string]any)["versionId"] = "revision-1"
	versionedStep["logicalInput"] = map[string]any{
		"ref":      versionedRef,
		"location": versionedLocation,
	}

	assertSchemaAccepts(t, schema, versionless)
	assertSchemaAccepts(t, schema, versioned)
}

func TestCorpusSchemaV1は失敗とTimeoutの整合した期待値を受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	for name, values := range map[string]struct {
		outcome   map[string]any
		errorCode string
	}{
		"failure": {
			outcome:   map[string]any{"kind": "failure", "errorCode": "source_unavailable"},
			errorCode: "source_unavailable",
		},
		"timeout": {
			outcome:   map[string]any{"kind": "timeout"},
			errorCode: "source_timeout",
		},
	} {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			expected := map[string]any{
				"terminal":  "error",
				"errorCode": values.errorCode,
				"attempts": []any{map[string]any{
					"meaningId":   "law-search",
					"stepOrdinal": float64(1),
					"outcome":     "failed",
					"errorCode":   values.errorCode,
				}},
			}
			assertSchemaAccepts(t, schema, validExecutionCase(values.outcome, expected))
		})
	}
}

func validExpectedMeaning(meaningID string, step map[string]any) map[string]any {
	return map[string]any{
		"meaningId":     meaningID,
		"evidenceCodes": []any{"explicit_task", "explicit_resource"},
		"conceptIds":    []any{},
		"requiredPacks": []any{},
		"steps":         []any{step},
	}
}
