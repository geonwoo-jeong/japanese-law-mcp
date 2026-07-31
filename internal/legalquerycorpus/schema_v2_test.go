package legalquerycorpus

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCorpusSchemaV2は三成果物と追加項目を閉じて受理する(t *testing.T) {
	t.Parallel()
	schema := resolvedCorpusSchemaV2(t)
	instances := []map[string]any{
		validManifestV2(),
		validSemanticCaseV2(),
		validExecutionCaseV2(),
	}
	for _, instance := range instances {
		assertSchemaAccepts(t, schema.resolved, instance)
		data := mustJSONBytes(t, instance)
		header, err := inspectJSONDocument(data)
		if err != nil {
			t.Fatalf("%s: header を検査できません: %v", corpusV2DevelopmentAssertionsVerificationID, err)
		}
		if _, err := schema.validateAndDecode(data, header); err != nil {
			t.Fatalf("%s: v2 typed decoder error = %v", corpusV2DevelopmentAssertionsVerificationID, err)
		}
	}
}

func TestCorpusSchemaV1はV2項目とCoverageを拒否する(t *testing.T) {
	t.Parallel()
	_, schema := resolvedCorpusSchemaV1(t)
	manifestLeakage := validManifest()
	manifestLeakage["holdoutLeakageGroupDigests"] = []any{strings.Repeat("a", 64)}
	manifestAssertions := validManifest()
	manifestAssertions["requiredDevelopmentAssertionIds"] = stringValues(
		manifestRequiredDevelopmentAssertionIDs()...,
	)
	semanticAssertions := validSemanticCase(validLawSearchStep())
	semanticAssertions["developmentAssertionIds"] = []any{
		manifestRequiredDevelopmentAssertionIDs()[0],
	}
	semanticCoverage := validSemanticCase(validLawSearchStep())
	semanticCoverage["coverageIds"] = []any{"structure-shared-terminal-cue"}
	for _, instance := range []map[string]any{
		manifestLeakage,
		manifestAssertions,
		semanticAssertions,
		semanticCoverage,
	} {
		assertSchemaRejects(t, schema, instance)
	}
}

func TestCorpusSchemaV2はAssertionとLeakageDigestの不正形を拒否する(
	t *testing.T,
) {
	t.Parallel()
	schema := resolvedCorpusSchemaV2(t)
	tests := []map[string]any{
		func() map[string]any {
			value := validManifestV2()
			value["holdoutLeakageGroupDigests"] = []any{
				strings.Repeat("a", 64),
				strings.Repeat("a", 64),
			}
			return value
		}(),
		func() map[string]any {
			value := validManifestV2()
			value["requiredDevelopmentAssertionIds"] = stringValues(
				manifestRequiredDevelopmentAssertionIDs()[:10]...,
			)
			return value
		}(),
		func() map[string]any {
			value := validSemanticCaseV2()
			value["caseId"] = "holdout-v2-assertion"
			return value
		}(),
		func() map[string]any {
			value := validSemanticCaseV2()
			value["developmentAssertionIds"] = []any{"unknown-assertion"}
			return value
		}(),
	}
	for _, instance := range tests {
		assertSchemaRejects(t, schema.resolved, instance)
	}
}

func TestCorpusSchemaV2ManifestGetterはV2一覧を深く複製する(t *testing.T) {
	manifest, err := decodeManifestV2(mustJSONBytes(t, validManifestV2()))
	if err != nil {
		t.Fatalf("%s: manifest v2 decode error = %v", corpusV2LeakageDigestsVerificationID, err)
	}
	digests := manifest.HoldoutLeakageGroupDigests()
	assertions := manifest.RequiredDevelopmentAssertionIDs()
	digests[0] = "changed"
	assertions[0] = "changed"
	if manifest.HoldoutLeakageGroupDigests()[0] == "changed" ||
		manifest.RequiredDevelopmentAssertionIDs()[0] == "changed" {
		t.Fatalf("%s: manifest getter から内部 slice が変更されました", corpusV2LeakageDigestsVerificationID)
	}

	reversed := validManifestV2()
	reversed["holdoutLeakageGroupDigests"] = []any{
		strings.Repeat("b", 64),
		strings.Repeat("a", 64),
	}
	if _, err := decodeManifestV2(mustJSONBytes(t, reversed)); err == nil {
		t.Fatalf("%s: digest の逆順を受理しました", corpusV2LeakageDigestsVerificationID)
	}
}

func resolvedCorpusSchemaV2(t *testing.T) corpusSchema {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		corpusSchemaV2Filename,
	))
	if err != nil {
		t.Fatalf("SOT-ENG-026: 固定 v2 schema を読めません: %v", err)
	}
	schema, err := newCorpusSchemaV2(data)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 固定 v2 schema を解決できません: %v", err)
	}
	return schema
}

func validManifestV2() map[string]any {
	value := validManifest()
	value["schemaVersion"] = float64(corpusSchemaVersionV2)
	value["corpusVersion"] = "corpus-v10"
	value["holdoutLeakageGroupDigests"] = []any{strings.Repeat("a", 64)}
	value["requiredDevelopmentAssertionIds"] = stringValues(
		manifestRequiredDevelopmentAssertionIDs()...,
	)
	return value
}

func validSemanticCaseV2() map[string]any {
	value := validSemanticCase(validLawSearchStep())
	value["schemaVersion"] = float64(corpusSchemaVersionV2)
	value["developmentAssertionIds"] = []any{
		manifestRequiredDevelopmentAssertionIDs()[0],
	}
	return value
}

func validExecutionCaseV2() map[string]any {
	value := validExecutionCase(
		map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)},
		validResultExpectation("completed"),
	)
	value["schemaVersion"] = float64(corpusSchemaVersionV2)
	return value
}

func TestCorpusSchemaV2CoverageCatalogはASCII順である(t *testing.T) {
	values := semanticCoverageIDsForSchemaVersion(corpusSchemaVersionV2)
	if !slices.IsSorted(values) || len(values) != 61 {
		t.Fatalf("%s: v2 coverage catalog = %#v", corpusV2HoldoutCoverageVerificationID, values)
	}
}

func TestCorpusSchemaV2はStepLimitだけ空Meaningを許す(t *testing.T) {
	t.Parallel()
	schema := resolvedCorpusSchemaV2(t)
	stepLimit := validSemanticCaseV2()
	stepLimit["expected"] = map[string]any{
		"kind":               "plan",
		"decision":           "needs_clarification",
		"reasonCodes":        []any{"step_limit_exceeded"},
		"meanings":           []any{},
		"selectedMeaningIds": []any{},
	}
	assertSchemaAccepts(t, schema.resolved, stepLimit)

	ordinary := validSemanticCaseV2()
	ordinary["expected"] = map[string]any{
		"kind":               "plan",
		"decision":           "needs_clarification",
		"reasonCodes":        []any{"below_execution_threshold"},
		"meanings":           []any{},
		"selectedMeaningIds": []any{},
	}
	assertSchemaRejects(t, schema.resolved, ordinary)
}
