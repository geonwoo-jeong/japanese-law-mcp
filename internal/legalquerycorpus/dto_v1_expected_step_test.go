package legalquerycorpus

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestExpectedStepV1Decodeは法令readの代替形を保持する(t *testing.T) {
	t.Parallel()

	lawReadByDate := validLawReadStep()
	lawReadByDate["logicalInput"] = map[string]any{
		"lawId": "503AC0000000037",
		"asOf":  "2025-06-01",
	}
	lawReadByRef := validLawReadStep()
	lawReadByRef["logicalInput"] = map[string]any{
		"ref": validResourceRef("law"),
	}
	articleByVersionlessRef := validLawArticleReadStep()
	articleByVersionlessRef["logicalInput"] = map[string]any{
		"ref": validResourceRef("law"),
		"location": map[string]any{
			"provision":     "supplementary",
			"articleNumber": "2",
		},
		"asOf": "2025-06-01",
	}
	articleByVersionedRef := validLawArticleReadStep()
	versionedRef := validResourceRef("law")
	versionedRef["key"].(map[string]any)["versionId"] = "revision-1"
	articleByVersionedRef["logicalInput"] = map[string]any{
		"ref": versionedRef,
		"location": map[string]any{
			"provision":       "main",
			"articleNumber":   "22_2",
			"paragraphNumber": 1,
		},
	}

	tests := []struct {
		name   string
		source map[string]any
		assert func(*testing.T, ExpectedStep)
	}{
		{
			name:   "law read by date",
			source: lawReadByDate,
			assert: func(t *testing.T, step ExpectedStep) {
				t.Helper()
				input := step.LogicalInput().(legalquery.LawReadIntentV1)
				if _, exists := input.RevisionID(); exists {
					t.Fatal("SOT-MODEL-022: revisionId が存在します")
				}
				if date, exists := input.AsOf(); !exists || date.String() != "2025-06-01" {
					t.Fatalf("SOT-MODEL-022: asOf = (%q, %t)", date.String(), exists)
				}
			},
		},
		{
			name:   "law read by ref",
			source: lawReadByRef,
			assert: func(t *testing.T, step ExpectedStep) {
				t.Helper()
				input := step.LogicalInput().(legalquery.LawReadIntentV1)
				ref, exists := input.Ref()
				if !exists || ref.Key().ResourceType() != "law" {
					t.Fatalf("SOT-MODEL-022: ref = (%#v, %t)", ref, exists)
				}
			},
		},
		{
			name:   "article by versionless ref and date",
			source: articleByVersionlessRef,
			assert: func(t *testing.T, step ExpectedStep) {
				t.Helper()
				input := step.LogicalInput().(legalquery.LawArticleReadIntentV1)
				if _, exists := input.AsOf(); !exists {
					t.Fatal("SOT-MODEL-022: asOf が失われた")
				}
				if input.Location().Provision() != "supplementary" {
					t.Fatalf("SOT-MODEL-022: location = %#v", input.Location())
				}
			},
		},
		{
			name:   "article by versioned ref",
			source: articleByVersionedRef,
			assert: func(t *testing.T, step ExpectedStep) {
				t.Helper()
				input := step.LogicalInput().(legalquery.LawArticleReadIntentV1)
				ref, exists := input.Ref()
				if !exists {
					t.Fatal("SOT-MODEL-022: ref が失われた")
				}
				versionID, exists := ref.Key().VersionID()
				if !exists || versionID != "revision-1" {
					t.Fatalf("SOT-MODEL-022: versionId = (%q, %t)", versionID, exists)
				}
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			step, err := decodeExpectedStepV1(mustJSONBytes(t, test.source))
			if err != nil {
				t.Fatalf("SOT-ENG-026: expected step decode error = %v", err)
			}
			test.assert(t, step)
		})
	}
}

func TestExpectedStepV1Decodeは必須項目と未知項目を拒否する(t *testing.T) {
	t.Parallel()

	missingTask := validLawSearchStep()
	delete(missingTask, "task")
	missingLogicalInput := validLawSearchStep()
	delete(missingLogicalInput, "logicalInput")
	unknownStep := validLawSearchStep()
	unknownStep["capabilityId"] = "law.search"
	unknownInput := validLawSearchStep()
	unknownInput["logicalInput"].(map[string]any)["limit"] = 20

	for name, source := range map[string]map[string]any{
		"task欠落":         missingTask,
		"logicalInput欠落": missingLogicalInput,
		"step未知項目":       unknownStep,
		"input未知項目":      unknownInput,
	} {
		name, source := name, source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeExpectedStepV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 不正な expected step を受理した")
			}
		})
	}
}

func TestExpectedStepV1DecodeはlogicalInputの不正な組合せを拒否する(t *testing.T) {
	t.Parallel()

	lawReadConflict := validLawReadStep()
	lawReadConflict["logicalInput"].(map[string]any)["asOf"] = "2025-06-01"

	articleConflict := validLawArticleReadStep()
	versionedRef := validResourceRef("law")
	versionedRef["key"].(map[string]any)["versionId"] = "revision-1"
	articleConflict["logicalInput"] = map[string]any{
		"ref":      versionedRef,
		"location": validLawArticleReadStep()["logicalInput"].(map[string]any)["location"],
		"asOf":     "2025-06-01",
	}

	judicialVersion := validJudicialReadStep()
	judicialVersion["logicalInput"].(map[string]any)["ref"].(map[string]any)["key"].(map[string]any)["versionId"] = "version-1"

	contentMissingRequiredArray := validLawContentSearchStep()
	delete(
		contentMissingRequiredArray["logicalInput"].(map[string]any),
		"excludeTerms",
	)

	for name, source := range map[string]map[string]any{
		"revisionIdとasOf":        lawReadConflict,
		"versioned refとasOf":     articleConflict,
		"judicial versionId":     judicialVersion,
		"content required array": contentMissingRequiredArray,
	} {
		name, source := name, source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeExpectedStepV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 不正な logicalInput を受理した")
			}
		})
	}
}

func TestExpectedStepV1Decodeはvariantごとの必須値を拒否する(t *testing.T) {
	t.Parallel()

	unknownKind := validLawSearchStep()
	unknownKind["inputKind"] = "unknown"

	lawSearchMissingQuery := validLawSearchStep()
	delete(lawSearchMissingQuery["logicalInput"].(map[string]any), "query")

	lawSearchInvalidDate := validLawSearchStep()
	lawSearchInvalidDate["logicalInput"].(map[string]any)["asOf"] = "2025-02-30"

	lawReadMissingTarget := validLawReadStep()
	lawReadMissingTarget["logicalInput"] = map[string]any{}

	articleMissingLocation := validLawArticleReadStep()
	delete(articleMissingLocation["logicalInput"].(map[string]any), "location")

	updatesMissingDate := validLawUpdatesStep()
	delete(updatesMissingDate["logicalInput"].(map[string]any), "date")

	judicialSearchMissingQuery := validJudicialSearchStep()
	delete(judicialSearchMissingQuery["logicalInput"].(map[string]any), "query")

	judicialReadMissingRef := validJudicialReadStep()
	delete(judicialReadMissingRef["logicalInput"].(map[string]any), "ref")

	for name, source := range map[string]map[string]any{
		"unknown inputKind":     unknownKind,
		"law search query":      lawSearchMissingQuery,
		"law search date":       lawSearchInvalidDate,
		"law read target":       lawReadMissingTarget,
		"article location":      articleMissingLocation,
		"updates date":          updatesMissingDate,
		"judicial search query": judicialSearchMissingQuery,
		"judicial decision ref": judicialReadMissingRef,
	} {
		name, source := name, source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeExpectedStepV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 必須値を欠く logicalInput を受理した")
			}
		})
	}
}
