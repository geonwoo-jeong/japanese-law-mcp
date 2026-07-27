package legalquerycorpus

import (
	"strings"
	"testing"
)

func TestCorpusSchemaV1は不透明なConceptIDを受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	instance := validSemanticCase(validLawSearchStep())
	meaning := firstExpectedMeaning(instance)
	meaning["evidenceCodes"] = []any{"legal_concept"}
	meaning["conceptIds"] = []any{"在留資格.永住_許可"}

	assertSchemaAccepts(t, schema, instance)
}

func TestCorpusSchemaV1は長いProviderIDを受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	instance := validSemanticCase(validLawReadStep())
	ref := validResourceRef("law")
	ref["providerId"] = strings.Repeat("p", 65)
	firstSemanticStep(instance)["logicalInput"] = map[string]any{"ref": ref}

	assertSchemaAccepts(t, schema, instance)
}

func TestCorpusSchemaV1は長い情報源識別子を受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	instance := validSemanticCase(validJudicialReadStep())
	ref := firstSemanticStep(instance)["logicalInput"].(map[string]any)["ref"].(map[string]any)
	key := ref["key"].(map[string]any)
	key["sourceId"] = strings.Repeat("s", 4097)
	key["resourceId"] = strings.Repeat("r", 4097)

	assertSchemaAccepts(t, schema, instance)
}
