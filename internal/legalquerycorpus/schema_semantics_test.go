package legalquerycorpus

import "testing"

func TestCorpusSchemaV1はReasonCodeの逆順を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	needsClarification := validSemanticCase(validLawSearchStep())
	needsExpected := needsClarification["expected"].(map[string]any)
	needsExpected["decision"] = "needs_clarification"
	needsExpected["reasonCodes"] = []any{
		"ambiguous_candidates",
		"below_execution_threshold",
	}
	needsExpected["selectedMeaningIds"] = []any{}

	unsupported := validSemanticCase(validLawSearchStep())
	unsupportedExpected := unsupported["expected"].(map[string]any)
	unsupportedExpected["decision"] = "unsupported"
	unsupportedExpected["reasonCodes"] = []any{
		"unsupported_task_or_resource",
		"non_japanese_query",
	}
	unsupportedExpected["selectedMeaningIds"] = []any{}

	assertSchemaRejects(t, schema, needsClarification)
	assertSchemaRejects(t, schema, unsupported)
}

func TestCorpusSchemaV1は版付きRefとAsOfの併用を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	instance := validSemanticCase(validLawArticleReadStep())
	step := firstSemanticStep(instance)
	location := step["logicalInput"].(map[string]any)["location"]
	ref := validResourceRef("law")
	ref["key"].(map[string]any)["versionId"] = "revision-1"
	step["logicalInput"] = map[string]any{
		"ref":      ref,
		"location": location,
		"asOf":     "2026-07-28",
	}

	assertSchemaRejects(t, schema, instance)
}
