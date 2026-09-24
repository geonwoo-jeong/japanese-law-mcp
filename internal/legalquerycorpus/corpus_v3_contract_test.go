package legalquerycorpus

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestCorpusV3は空入力だけをHoldoutから分離する(t *testing.T) {
	schema := corpusV3TestSchema(t)
	for _, version := range []int{1, 2, 3} {
		for _, set := range []string{"development", "holdout"} {
			for _, query := range []string{"", " \t\u00a0\u3000", "\u0085\u1680\u2000\u200a\u2028\u2029\u202f\u205f"} {
				for _, coverageID := range []string{"input-query-empty", "input-query-ascii-control"} {
					corpusV3TestEmptyBoundary(t, schema, version, set, query, coverageID)
				}
			}
		}
	}
	values := validSemanticCaseValues(t)
	values.SchemaVersion = 3
	values.CaseID = "holdout-renamed-empty"
	values.CoverageIDs = []string{"input-query-empty"}
	if _, err := NewSemanticCase(values); err == nil {
		t.Fatal("SOT-ENG-046: 非空 query でも holdout の空入力 coverage は禁止です")
	}
}

func corpusV3TestEmptyBoundary(
	t *testing.T,
	schema corpusSchema,
	version int,
	set string,
	query string,
	coverageID string,
) {
	t.Helper()
	values := validSemanticCaseValues(t)
	values.SchemaVersion = version
	values.CaseID = set + "-boundary"
	values.CoverageIDs = []string{coverageID}
	values.Request = mustRawRequest(t, query, nil, nil)
	values.Expected = mustExpectedRequestError(t, RequestErrorFieldQuery)
	_, err := NewSemanticCase(values)
	wantRejection := version == 3 && set == "holdout"
	if (err != nil) != wantRejection {
		t.Fatalf("SOT-ENG-046: version=%d set=%s の空入力境界が不一致です: %v", version, set, err)
	}
	if version != 3 {
		return
	}
	source := validSemanticCaseV2()
	delete(source, "developmentAssertionIds")
	source["schemaVersion"] = 3
	source["caseId"] = values.CaseID
	source["coverageIds"] = []any{coverageID}
	source["request"] = map[string]any{"query": query}
	source["expected"] = map[string]any{"kind": "request_error", "errorCode": "invalid_argument", "field": "query"}
	if (schema.validate(mustJSONBytes(t, source)) != nil) != wantRejection {
		t.Fatal("SOT-ENG-046: schema と constructor の空入力境界が不一致です")
	}
}

func TestCorpusV3は残る60Coverageと旧版契約を維持する(t *testing.T) {
	old := semanticHoldoutCoverageDefinitions(2)
	current := semanticHoldoutCoverageDefinitions(3)
	if len(old) != 61 || len(current) != 60 {
		t.Fatal("SOT-ENG-046: coverage 必須集合の件数が不一致です")
	}
	counts := make(map[string]int, len(current))
	for _, definition := range current {
		legacy, exists := semanticCoverageDefinitionForSchemaVersion(2, definition.id)
		if !exists || !reflect.DeepEqual(legacy, definition) {
			t.Fatal("SOT-ENG-046: 残す coverage の条件が変化しました")
		}
		counts[definition.id] = definition.minimumHoldoutCount
	}
	if err := validateHoldoutCoverageCounts(3, counts); err != nil {
		t.Fatalf("SOT-ENG-046: 空入力以外の全 coverage を拒否しました: %v", err)
	}
	if err := validateHoldoutCoverageCounts(2, counts); err == nil {
		t.Fatal("SOT-ENG-046: 旧版が空入力 coverage 不足を受理しました")
	}
	for _, definition := range current {
		reduced := make(map[string]int, len(counts))
		for key, count := range counts {
			reduced[key] = count
		}
		reduced[definition.id]--
		if err := validateHoldoutCoverageCounts(3, reduced); err == nil {
			t.Fatalf("SOT-ENG-046: coverage %s の不足を受理しました", definition.id)
		}
	}
	if _, err := corpusSchemaFilename(4); err == nil {
		t.Fatal("SOT-ENG-046: 未知 schema の fallback を受理しました")
	}
	if err := validateHoldoutRequirementsForSchemaVersion(4, corpusV2ValidHoldout(t)); err == nil {
		t.Fatal("SOT-ENG-046: 未知版の holdout 要件を受理しました")
	}
}

func TestCorpusV3は合成Corpusを版別Loaderで受理する(t *testing.T) {
	layout := corpusV3TestWriteCorpus(t)
	ctx := context.Background()
	corpus, err := Load(ctx, layout.repositoryRoot, layout.corpusPath)
	if err != nil {
		t.Fatalf("SOT-ENG-046: 合成 v3 corpus を拒否しました: %v", err)
	}
	if corpus.Manifest().SchemaVersion() != 3 || len(corpus.Holdout()) != 251 || len(corpus.Execution()) != 8 {
		t.Fatal("SOT-ENG-046: 合成 corpus の版又は集合が不一致です")
	}
	artifact, err := LoadManifest(ctx, layout.repositoryRoot, layout.corpusPath)
	if err != nil || artifact.Manifest().SchemaVersion() != 3 {
		t.Fatalf("SOT-ENG-046: manifest 専用 loader が v3 を拒否しました: %v", err)
	}
	development, err := LoadDevelopment(ctx, layout.repositoryRoot, filepath.Join(layout.corpusPath, "development"))
	if err != nil || development.SchemaVersion() != 3 {
		t.Fatalf("SOT-ENG-046: development 専用 loader が v3 を拒否しました: %v", err)
	}
}

func corpusV3TestSchema(t *testing.T) corpusSchema {
	t.Helper()
	data := corpusV3TestSchemaBytes(t)
	schema, err := newCorpusSchema(3, data)
	if err != nil {
		t.Fatalf("SOT-ENG-046: schema を解決できません: %v", err)
	}
	return schema
}

func corpusV3TestSchemaBytes(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(filepath.Dir(corpusSchemaV1Path(t)), corpusSchemaV3Filename))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func corpusV3TestWriteCorpus(t *testing.T) filesystemReadTestLayout {
	t.Helper()
	layout := filesystemReadTestNewLayoutForVersion(t, loadTestCorpusVersion)
	base := append(loadTestDevelopmentFixtures(t), loadTestHoldoutFixtures(t)...)
	base = append(base, loadTestExecutionFixtures(t)...)
	fixtures := make([]manifestIntegrityTestFixture, 0, len(base)+11)
	for _, fixture := range base {
		source := corpusV3TestObject(t, fixture.data)
		source["schemaVersion"] = 3
		if fixture.set == ManifestSetDevelopment {
			source["developmentAssertionIds"] = stringValues(manifestRequiredDevelopmentAssertionIDs()...)
		}
		if fixture.set == ManifestSetHoldout {
			coverage := source["coverageIds"].([]any)
			if slices.Contains(coverage, any("input-query-empty")) {
				source["coverageIds"] = []any{"input-invalid-ref"}
			}
		}
		fixtures = append(fixtures, manifestIntegrityTestFixture{set: fixture.set, caseID: fixture.caseID, data: mustJSONBytes(t, source)})
	}
	for index, coverage := range corpusV2AdditionalCoverageCases() {
		source := validSemanticCase(validLawSearchStep())
		caseID := fmt.Sprintf("holdout-v3-additional-%02d", index)
		source["schemaVersion"], source["caseId"], source["leakageGroupId"] = 3, caseID, caseID
		source["coverageIds"] = []any{coverage.coverageID}
		source["request"] = map[string]any{"query": "追加合成照会-" + caseID}
		if coverage.variant != "" {
			source["safetyVariant"] = string(coverage.variant)
		}
		fixtures = append(fixtures, manifestIntegrityTestFixture{set: ManifestSetHoldout, caseID: caseID, data: mustJSONBytes(t, source)})
	}
	slices.SortFunc(fixtures, func(left, right manifestIntegrityTestFixture) int {
		return strings.Compare(left.caseID, right.caseID)
	})
	manifestRaw, _ := manifestIntegrityTestBuildManifest(t, fixtures, loadTestCorpusVersion, "")
	manifest := corpusV3TestObject(t, manifestRaw)
	manifest["schemaVersion"] = 3
	manifest["requiredDevelopmentAssertionIds"] = stringValues(manifestRequiredDevelopmentAssertionIDs()...)
	var holdout []SemanticCase
	for _, fixture := range fixtures {
		filesystemReadTestWriteFixture(t, layout, fixture.set, fixture.caseID, fixture.data)
		if fixture.set == ManifestSetHoldout {
			semanticCase, err := decodeSemanticCaseV2(fixture.data)
			if err != nil {
				t.Fatal(err)
			}
			holdout = append(holdout, semanticCase)
		}
	}
	manifest["holdoutLeakageGroupDigests"] = computeHoldoutLeakageGroupDigests(holdout)
	filesystemReadTestWriteFile(t, layout.manifestPath, mustJSONBytes(t, manifest))
	filesystemReadTestWriteFile(t, filepath.Join(filepath.Dir(layout.schemaPath), corpusSchemaV3Filename), corpusV3TestSchemaBytes(t))
	return layout
}

func corpusV3TestObject(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
