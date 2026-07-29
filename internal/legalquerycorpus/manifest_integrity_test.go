package legalquerycorpus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"reflect"
	"strings"
	"testing"
)

type manifestIntegrityTestFixture struct {
	set         ManifestSetKind
	caseID      string
	data        []byte
	declaredSHA string
}

func TestManifestIntegrityは三集合をmanifest順で返し共有状態を持たない(
	t *testing.T,
) {
	fixtures := manifestIntegrityTestOrderedFixtures(t)
	layout, schema, manifest := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v1",
		"",
	)
	fs := filesystemReadTestOpen(t, layout)

	got, err := validateManifestArtifacts(
		context.Background(),
		fs,
		schema,
		manifest,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 正常な manifest integrity error = %v", err)
	}
	manifestIntegrityTestRequireCaseIDs(
		t,
		got.development,
		[]string{"development-a", "development-z"},
	)
	manifestIntegrityTestRequireCaseIDs(
		t,
		got.holdout,
		[]string{"holdout-a", "holdout-z"},
	)
	manifestIntegrityTestRequireExecutionCaseIDs(
		t,
		got.execution,
		[]string{"execution-a", "execution-z"},
	)
	if err := got.manifest.Validate(); err != nil {
		t.Fatalf("SOT-ENG-026: result manifest = %v", err)
	}

	got.manifest.requiredCategoryIDs[0] = "changed"
	got.development[0].coverageIDs[0] = "changed"
	got.development[0] = SemanticCase{}
	if manifest.RequiredCategoryIDs()[0] != "ambiguity" {
		t.Fatal("SOT-ENG-026: result manifest が入力 manifest と共有された")
	}
	again, err := validateManifestArtifacts(
		context.Background(),
		fs,
		schema,
		manifest,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 二回目の integrity 検証 error = %v", err)
	}
	if again.manifest.RequiredCategoryIDs()[0] != "ambiguity" ||
		again.development[0].CoverageIDs()[0] != "intent-law-search" {
		t.Fatal("SOT-ENG-026: integrity result 間で slice または model を共有した")
	}
}

func TestManifestIntegrityはCorpusVersion不一致を拒否する(t *testing.T) {
	fixtures := manifestIntegrityTestBaseFixtures(t)
	layout, schema, manifest := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v2",
		"",
	)
	fs := filesystemReadTestOpen(t, layout)

	_ = manifestIntegrityTestRequireFailure(
		t,
		context.Background(),
		fs,
		schema,
		manifest,
	)
}

func TestManifestIntegrityは各集合のfile完全一致を要求する(t *testing.T) {
	sets := []ManifestSetKind{
		ManifestSetDevelopment,
		ManifestSetHoldout,
		ManifestSetExecution,
	}
	mutations := []struct {
		name   string
		change func(
			t *testing.T,
			layout filesystemReadTestLayout,
			fixture manifestIntegrityTestFixture,
		)
	}{
		{
			name: "missing",
			change: func(
				t *testing.T,
				layout filesystemReadTestLayout,
				fixture manifestIntegrityTestFixture,
			) {
				path := filesystemReadTestFixturePath(
					layout,
					fixture.set,
					fixture.caseID,
				)
				if err := os.Remove(path); err != nil {
					t.Fatalf("SOT-ENG-026: fixture を削除できない: %v", err)
				}
			},
		},
		{
			name: "extra",
			change: func(
				t *testing.T,
				layout filesystemReadTestLayout,
				fixture manifestIntegrityTestFixture,
			) {
				filesystemReadTestWriteFixture(
					t,
					layout,
					fixture.set,
					string(fixture.set)+"-extra",
					[]byte(`{"extra":true}`),
				)
			},
		},
		{
			name: "renamed",
			change: func(
				t *testing.T,
				layout filesystemReadTestLayout,
				fixture manifestIntegrityTestFixture,
			) {
				from := filesystemReadTestFixturePath(
					layout,
					fixture.set,
					fixture.caseID,
				)
				to := filesystemReadTestFixturePath(
					layout,
					fixture.set,
					string(fixture.set)+"-renamed",
				)
				if err := os.Rename(from, to); err != nil {
					t.Fatalf("SOT-ENG-026: fixture を改名できない: %v", err)
				}
			},
		},
	}

	for _, set := range sets {
		set := set
		for _, mutation := range mutations {
			mutation := mutation
			t.Run(string(set)+"/"+mutation.name, func(t *testing.T) {
				fixtures := manifestIntegrityTestBaseFixtures(t)
				layout, schema, manifest := manifestIntegrityTestPrepare(
					t,
					fixtures,
					"corpus-v1",
					"",
				)
				fixture := manifestIntegrityTestFixtureForSet(
					t,
					fixtures,
					set,
				)
				mutation.change(t, layout, fixture)
				fs := filesystemReadTestOpen(t, layout)
				_ = manifestIntegrityTestRequireFailure(
					t,
					context.Background(),
					fs,
					schema,
					manifest,
				)
			})
		}
	}
}

func TestManifestIntegrityは重複checksumを全集合で拒否する(t *testing.T) {
	tests := map[string]func([]manifestIntegrityTestFixture){
		"宣言checksum重複": func(fixtures []manifestIntegrityTestFixture) {
			fixtures[2].declaredSHA = manifestIntegrityTestSHA256(fixtures[0].data)
		},
		"実byte checksum重複": func(fixtures []manifestIntegrityTestFixture) {
			fixtures[2].data = append([]byte{}, fixtures[0].data...)
			fixtures[2].declaredSHA = manifestIntegrityTestSHA256(fixtures[0].data)
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			fixtures := manifestIntegrityTestBaseFixtures(t)
			mutate(fixtures)
			layout, schema, manifest := manifestIntegrityTestPrepare(
				t,
				fixtures,
				"corpus-v1",
				"",
			)
			fs := filesystemReadTestOpen(t, layout)
			_ = manifestIntegrityTestRequireFailure(
				t,
				context.Background(),
				fs,
				schema,
				manifest,
			)
		})
	}
}

func TestManifestIntegrityはHoldoutDigest不一致を拒否する(t *testing.T) {
	fixtures := manifestIntegrityTestBaseFixtures(t)
	layout, schema, manifest := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v1",
		strings.Repeat("f", 64),
	)
	fs := filesystemReadTestOpen(t, layout)

	_ = manifestIntegrityTestRequireFailure(
		t,
		context.Background(),
		fs,
		schema,
		manifest,
	)
}

func manifestIntegrityTestPrepare(
	t *testing.T,
	fixtures []manifestIntegrityTestFixture,
	corpusVersion string,
	holdoutDigestOverride string,
) (
	filesystemReadTestLayout,
	corpusSchemaV1,
	Manifest,
) {
	t.Helper()

	layout := filesystemReadTestNewLayout(t)
	for _, fixture := range fixtures {
		filesystemReadTestWriteFixture(
			t,
			layout,
			fixture.set,
			fixture.caseID,
			fixture.data,
		)
	}
	manifestData, manifest := manifestIntegrityTestBuildManifest(
		t,
		fixtures,
		corpusVersion,
		holdoutDigestOverride,
	)
	filesystemReadTestWriteFile(t, layout.manifestPath, manifestData)
	schema, err := newCorpusSchemaV1(schemaRuntimeTestFixedSchema(t))
	if err != nil {
		t.Fatalf("SOT-ENG-026: integrity 用 schema error = %v", err)
	}
	return layout, schema, manifest
}

func manifestIntegrityTestBuildManifest(
	t *testing.T,
	fixtures []manifestIntegrityTestFixture,
	corpusVersion string,
	holdoutDigestOverride string,
) ([]byte, Manifest) {
	t.Helper()

	source := validManifest()
	source["corpusVersion"] = corpusVersion
	source["requiredExecutionScenarioIds"] = stringValues(
		manifestRequiredExecutionScenarioIDsForVersion(corpusVersion)...,
	)
	sets := make(map[string]any, 3)
	for _, set := range []ManifestSetKind{
		ManifestSetDevelopment,
		ManifestSetHoldout,
		ManifestSetExecution,
	} {
		cases := make([]any, 0)
		for _, fixture := range fixtures {
			if fixture.set != set {
				continue
			}
			checksum := fixture.declaredSHA
			if checksum == "" {
				checksum = manifestIntegrityTestSHA256(fixture.data)
			}
			cases = append(cases, map[string]any{
				"caseId": fixture.caseID,
				"sha256": checksum,
			})
		}
		sets[string(set)] = map[string]any{
			"caseCount": float64(len(cases)),
			"cases":     cases,
		}
	}
	source["sets"] = sets
	holdoutDigest := manifestIntegrityTestHoldoutDigest(fixtures)
	if holdoutDigestOverride != "" {
		holdoutDigest = holdoutDigestOverride
	}
	source["holdoutDigest"] = holdoutDigest
	data := mustJSONBytes(t, source)
	manifest, err := decodeManifestV1(data)
	if err != nil {
		t.Fatalf("SOT-ENG-026: test manifest を復元できない: %v", err)
	}
	return data, manifest
}

func manifestIntegrityTestBaseFixtures(
	t *testing.T,
) []manifestIntegrityTestFixture {
	t.Helper()

	return []manifestIntegrityTestFixture{
		manifestIntegrityTestSemanticFixture(
			t,
			ManifestSetDevelopment,
			"development-a",
			"行政手続法を検索",
		),
		manifestIntegrityTestSemanticFixture(
			t,
			ManifestSetHoldout,
			"holdout-a",
			"行政手続法を確認",
		),
		manifestIntegrityTestExecutionFixture(t, "execution-a"),
	}
}

func manifestIntegrityTestOrderedFixtures(
	t *testing.T,
) []manifestIntegrityTestFixture {
	t.Helper()

	return []manifestIntegrityTestFixture{
		manifestIntegrityTestSemanticFixture(
			t,
			ManifestSetDevelopment,
			"development-a",
			"行政手続法を検索",
		),
		manifestIntegrityTestSemanticFixture(
			t,
			ManifestSetDevelopment,
			"development-z",
			"行政事件訴訟法を検索",
		),
		manifestIntegrityTestSemanticFixture(
			t,
			ManifestSetHoldout,
			"holdout-a",
			"行政手続法を確認",
		),
		manifestIntegrityTestSemanticFixture(
			t,
			ManifestSetHoldout,
			"holdout-z",
			"行政事件訴訟法を確認",
		),
		manifestIntegrityTestExecutionFixture(t, "execution-a"),
		manifestIntegrityTestExecutionFixture(t, "execution-z"),
	}
}

func manifestIntegrityTestSemanticFixture(
	t *testing.T,
	set ManifestSetKind,
	caseID string,
	query string,
) manifestIntegrityTestFixture {
	t.Helper()

	source := validSemanticCase(validLawSearchStep())
	source["caseId"] = caseID
	source["leakageGroupId"] = "group-" + caseID
	source["request"].(map[string]any)["query"] = query
	return manifestIntegrityTestFixture{
		set:    set,
		caseID: caseID,
		data:   mustJSONBytes(t, source),
	}
}

func manifestIntegrityTestExecutionFixture(
	t *testing.T,
	caseID string,
) manifestIntegrityTestFixture {
	t.Helper()

	source := validExecutionCase(
		map[string]any{
			"kind":            "collection_success",
			"sourceItemCount": float64(1),
		},
		validResultExpectation("completed"),
	)
	source["caseId"] = caseID
	return manifestIntegrityTestFixture{
		set:    ManifestSetExecution,
		caseID: caseID,
		data:   mustJSONBytes(t, source),
	}
}

func manifestIntegrityTestSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func manifestIntegrityTestHoldoutDigest(
	fixtures []manifestIntegrityTestFixture,
) string {
	var input strings.Builder
	for _, fixture := range fixtures {
		if fixture.set != ManifestSetHoldout {
			continue
		}
		input.WriteString(fixture.caseID)
		input.WriteByte(' ')
		input.WriteString(manifestIntegrityTestSHA256(fixture.data))
		input.WriteByte('\n')
	}
	return manifestIntegrityTestSHA256([]byte(input.String()))
}

func manifestIntegrityTestRequireFailure(
	t *testing.T,
	ctx context.Context,
	fs *corpusFilesystem,
	schema corpusSchemaV1,
	manifest Manifest,
) error {
	t.Helper()

	got, err := validateManifestArtifacts(ctx, fs, schema, manifest)
	if err == nil {
		t.Fatal("SOT-ENG-026: integrity 違反を受理した")
	}
	if !reflect.DeepEqual(got, integrityCheckedCorpus{}) {
		t.Fatalf("SOT-ENG-026: 失敗時に部分結果を返した: %#v", got)
	}
	return err
}

func manifestIntegrityTestFixtureForSet(
	t *testing.T,
	fixtures []manifestIntegrityTestFixture,
	set ManifestSetKind,
) manifestIntegrityTestFixture {
	t.Helper()

	for _, fixture := range fixtures {
		if fixture.set == set {
			return fixture
		}
	}
	t.Fatalf("SOT-ENG-026: test fixture に %s がない", set)
	return manifestIntegrityTestFixture{}
}

func manifestIntegrityTestRequireCaseIDs(
	t *testing.T,
	cases []SemanticCase,
	want []string,
) {
	t.Helper()

	got := make([]string, len(cases))
	for index, semanticCase := range cases {
		got[index] = semanticCase.CaseID()
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SOT-ENG-026: semantic case 順 = %#v, want %#v", got, want)
	}
}

func manifestIntegrityTestRequireExecutionCaseIDs(
	t *testing.T,
	cases []ExecutionCase,
	want []string,
) {
	t.Helper()

	got := make([]string, len(cases))
	for index, executionCase := range cases {
		got[index] = executionCase.CaseID()
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SOT-ENG-026: execution case 順 = %#v, want %#v", got, want)
	}
}
