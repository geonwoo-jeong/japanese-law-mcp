package legalquerycorpus

import (
	"strconv"
	"strings"
	"testing"
)

func TestCorpusSchemaV1はManifestの件数上限超過を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	instance := validManifest()
	cases := make([]any, 4097)
	for index := range cases {
		cases[index] = map[string]any{
			"caseId": "development-case-" + strconv.Itoa(index),
			"sha256": strings.Repeat("a", 64),
		}
	}
	development := instance["sets"].(map[string]any)["development"].(map[string]any)
	development["caseCount"] = float64(4096)
	development["cases"] = cases

	assertSchemaRejects(t, schema, instance)
}

func TestCorpusSchemaV1はMeaningとStepの上限超過を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)

	tooManyMeanings := validSemanticCase(validLawSearchStep())
	meanings := make([]any, 17)
	for index := range meanings {
		meanings[index] = validExpectedMeaning(
			"meaning-"+strconv.Itoa(index),
			validLawSearchStep(),
		)
	}
	tooManyMeanings["expected"].(map[string]any)["meanings"] = meanings

	tooManySteps := validSemanticCase(validLawSearchStep())
	steps := make([]any, 5)
	for index := range steps {
		steps[index] = validLawSearchStep()
	}
	firstExpectedMeaning(tooManySteps)["steps"] = steps

	assertSchemaRejects(t, schema, tooManyMeanings)
	assertSchemaRejects(t, schema, tooManySteps)
}

func TestCorpusSchemaV1はActionとAttemptの上限超過を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)

	tooManyActions := validExecutionCase(
		map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)},
		validResultExpectation("completed"),
	)
	action := tooManyActions["actions"].([]any)[0]
	tooManyActions["actions"] = []any{action, action, action, action, action}

	tooManyAttempts := validExecutionCase(
		map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)},
		validResultExpectation("completed"),
	)
	expected := tooManyAttempts["expected"].(map[string]any)
	attempt := expected["attempts"].([]any)[0]
	expected["attempts"] = []any{attempt, attempt, attempt, attempt, attempt}

	assertSchemaRejects(t, schema, tooManyActions)
	assertSchemaRejects(t, schema, tooManyAttempts)
}

func TestCorpusSchemaV1は件数値の上限超過を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)

	sourceOverflow := validExecutionCase(
		map[string]any{"kind": "collection_success", "sourceItemCount": float64(1001)},
		validResultExpectation("completed"),
	)
	resultOverflow := validExecutionCase(
		map[string]any{"kind": "collection_success", "sourceItemCount": float64(41)},
		validResultExpectation("completed"),
	)
	resultOverflow["expected"].(map[string]any)["returnedItemCount"] = float64(41)

	assertSchemaRejects(t, schema, sourceOverflow)
	assertSchemaRejects(t, schema, resultOverflow)
}
