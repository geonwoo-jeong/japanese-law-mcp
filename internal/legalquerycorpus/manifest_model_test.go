package legalquerycorpus

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestManifestV1Decodeと不変getter(t *testing.T) {
	t.Parallel()

	source := validManifest()
	manifest, err := decodeManifestV1(mustJSONBytes(t, source))
	if err != nil {
		t.Fatalf("SOT-ENG-026: manifest の decode error = %v", err)
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("SOT-ENG-026: manifest の Validate() error = %v", err)
	}
	if manifest.ArtifactKind() != ArtifactKindCorpusManifest ||
		manifest.SchemaVersion() != 1 ||
		manifest.CorpusVersion() != "corpus-v4" ||
		manifest.Seed() != 20260728 ||
		manifest.HoldoutDigest() != strings.Repeat("a", 64) {
		t.Fatalf("SOT-ENG-026: manifest header = %#v", manifest)
	}

	categories := manifest.RequiredCategoryIDs()
	categories[0] = "changed"
	if manifest.RequiredCategoryIDs()[0] != "ambiguity" {
		t.Fatal("SOT-ENG-026: requiredCategoryIds が getter から変更された")
	}
	entries := manifest.Development().Cases()
	entries[0] = ManifestEntry{}
	if manifest.Development().Cases()[0].CaseID() != "development-law-search" {
		t.Fatal("SOT-ENG-026: development cases が getter から変更された")
	}
}

func TestManifestConstructorは入力sliceを複製する(t *testing.T) {
	t.Parallel()

	development := mustManifestSet(
		t,
		ManifestSetDevelopment,
		"development-a",
		strings.Repeat("a", 64),
	)
	holdout := mustManifestSet(
		t,
		ManifestSetHoldout,
		"holdout-a",
		strings.Repeat("b", 64),
	)
	execution := mustManifestSet(
		t,
		ManifestSetExecution,
		"execution-a",
		strings.Repeat("c", 64),
	)
	categories := requiredCategoryIDs()
	scenarios := requiredExecutionScenarioIDs()
	manifest, err := NewManifest(ManifestValues{
		ArtifactKind:                 ArtifactKindCorpusManifest,
		SchemaVersion:                1,
		CorpusVersion:                "corpus-v4",
		Seed:                         1,
		HoldoutDigest:                strings.Repeat("d", 64),
		RequiredCategoryIDs:          categories,
		RequiredExecutionScenarioIDs: scenarios,
		Development:                  development,
		Holdout:                      holdout,
		Execution:                    execution,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewManifest() error = %v", err)
	}
	categories[0] = "changed"
	scenarios[0] = "changed"
	if manifest.RequiredCategoryIDs()[0] != "ambiguity" ||
		manifest.RequiredExecutionScenarioIDs()[0] != "execution-all-failed" {
		t.Fatal("SOT-ENG-026: constructor の入力 slice が共有された")
	}
}

func TestManifestはzero値と不正値を拒否する(t *testing.T) {
	t.Parallel()

	if err := (Manifest{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: Manifest の zero value を受理した")
	}
	if err := (ManifestSet{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: ManifestSet の zero value を受理した")
	}
	if err := (ManifestEntry{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: ManifestEntry の zero value を受理した")
	}

	tests := map[string]func() map[string]any{
		"artifactKind": func() map[string]any {
			value := validManifest()
			value["artifactKind"] = "semantic_case"
			return value
		},
		"schemaVersion": func() map[string]any {
			value := validManifest()
			value["schemaVersion"] = 2
			return value
		},
		"corpusVersion": func() map[string]any {
			value := validManifest()
			value["corpusVersion"] = "corpus-v01"
			return value
		},
		"seed": func() map[string]any {
			value := validManifest()
			value["seed"] = 2147483648
			return value
		},
		"holdoutDigest": func() map[string]any {
			value := validManifest()
			value["holdoutDigest"] = strings.Repeat("A", 64)
			return value
		},
		"category順序": func() map[string]any {
			value := validManifest()
			ids := requiredCategoryIDs()
			ids[0], ids[1] = ids[1], ids[0]
			value["requiredCategoryIds"] = stringValues(ids...)
			return value
		},
		"scenario不足": func() map[string]any {
			value := validManifest()
			value["requiredExecutionScenarioIds"] = stringValues(
				requiredExecutionScenarioIDs()[:6]...,
			)
			return value
		},
		"legacy versionに新scenario一覧": func() map[string]any {
			value := validManifest()
			value["corpusVersion"] = "corpus-v1"
			value["requiredExecutionScenarioIds"] = stringValues(
				requiredExecutionScenarioIDs()...,
			)
			return value
		},
		"caseCount": func() map[string]any {
			value := validManifest()
			value["sets"].(map[string]any)["development"].(map[string]any)["caseCount"] = 2
			return value
		},
		"caseId prefix": func() map[string]any {
			value := validManifest()
			entry := firstManifestEntry(value, "development")
			entry["caseId"] = "holdout-a"
			return value
		},
		"entry sha256": func() map[string]any {
			value := validManifest()
			firstManifestEntry(value, "holdout")["sha256"] = "bad"
			return value
		},
		"entry順序": func() map[string]any {
			value := validManifest()
			set := value["sets"].(map[string]any)["development"].(map[string]any)
			first := set["cases"].([]any)[0]
			set["caseCount"] = 2
			set["cases"] = []any{
				map[string]any{
					"caseId": "development-z",
					"sha256": strings.Repeat("d", 64),
				},
				first,
			}
			return value
		},
		"entry重複": func() map[string]any {
			value := validManifest()
			set := value["sets"].(map[string]any)["execution"].(map[string]any)
			first := set["cases"].([]any)[0]
			set["caseCount"] = 2
			set["cases"] = []any{first, first}
			return value
		},
	}
	for name, build := range tests {
		name, build := name, build
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeManifestV1(mustJSONBytes(t, build())); err == nil {
				t.Fatal("SOT-ENG-026: 不正な manifest を受理した")
			}
		})
	}
}

func TestManifestV1Decodeは未知項目とtrailing値を拒否する(t *testing.T) {
	t.Parallel()

	unknown := validManifest()
	unknown["unknown"] = true
	for _, data := range [][]byte{
		mustJSONBytes(t, unknown),
		append(mustJSONBytes(t, validManifest()), []byte(` {}`)...),
	} {
		if _, err := decodeManifestV1(data); err == nil {
			t.Fatal("SOT-ENG-026: 未知項目または trailing value を受理した")
		}
	}
}

func TestManifestV1DTOは必須文字列の欠落を空値と区別する(t *testing.T) {
	t.Parallel()

	tests := map[string]func(map[string]any){
		"artifactKind": func(value map[string]any) {
			delete(value, "artifactKind")
		},
		"corpusVersion": func(value map[string]any) {
			delete(value, "corpusVersion")
		},
		"holdoutDigest": func(value map[string]any) {
			delete(value, "holdoutDigest")
		},
		"entry caseId": func(value map[string]any) {
			delete(firstManifestEntry(value, "development"), "caseId")
		},
		"entry sha256": func(value map[string]any) {
			delete(firstManifestEntry(value, "development"), "sha256")
		},
	}
	for name, removeRequiredField := range tests {
		name, removeRequiredField := name, removeRequiredField
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			value := validManifest()
			removeRequiredField(value)
			_, err := decodeManifestV1(mustJSONBytes(t, value))
			if err == nil || !strings.Contains(err.Error(), "必須項目") {
				t.Fatalf("SOT-ENG-026: 必須項目の欠落 error = %v", err)
			}
		})
	}
}

func TestManifestGetterは並行読取りで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	manifest, err := decodeManifestV1(mustJSONBytes(t, validManifest()))
	if err != nil {
		t.Fatalf("SOT-ENG-026: manifest の decode error = %v", err)
	}
	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for repeat := 0; repeat < 100; repeat++ {
				categories := manifest.RequiredCategoryIDs()
				categories[0] = "changed"
				cases := manifest.Holdout().Cases()
				cases[0] = ManifestEntry{}
			}
		}()
	}
	wait.Wait()
	if manifest.RequiredCategoryIDs()[0] != "ambiguity" ||
		manifest.Holdout().Cases()[0].CaseID() != "holdout-law-search" {
		t.Fatal("SOT-ENG-026: 並行 getter が manifest を変更した")
	}
}

func Test公開Corpus型はmapとRawMessageを持たない(t *testing.T) {
	t.Parallel()

	types := []reflect.Type{
		reflect.TypeOf(Manifest{}),
		reflect.TypeOf(ManifestValues{}),
		reflect.TypeOf(ManifestSet{}),
		reflect.TypeOf(ManifestSetValues{}),
		reflect.TypeOf(ManifestEntry{}),
		reflect.TypeOf(ManifestEntryValues{}),
		reflect.TypeOf(Request{}),
		reflect.TypeOf(RequestValues{}),
		reflect.TypeOf(RequestRef{}),
		reflect.TypeOf(RequestRefValues{}),
		reflect.TypeOf(RequestKey{}),
		reflect.TypeOf(RequestKeyValues{}),
	}
	for _, typ := range types {
		assertNoPublicDynamicJSONType(t, typ, map[reflect.Type]bool{})
	}
}

func mustJSONBytes(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("JSON を作成できません: %v", err)
	}
	return data
}

func mustManifestSet(
	t *testing.T,
	kind ManifestSetKind,
	caseID string,
	digest string,
) ManifestSet {
	t.Helper()
	entry, err := NewManifestEntry(ManifestEntryValues{CaseID: caseID, SHA256: digest})
	if err != nil {
		t.Fatalf("NewManifestEntry() error = %v", err)
	}
	set, err := NewManifestSet(ManifestSetValues{
		Kind:      kind,
		CaseCount: 1,
		Cases:     []ManifestEntry{entry},
	})
	if err != nil {
		t.Fatalf("NewManifestSet() error = %v", err)
	}
	return set
}

func firstManifestEntry(manifest map[string]any, setName string) map[string]any {
	return manifest["sets"].(map[string]any)[setName].(map[string]any)["cases"].([]any)[0].(map[string]any)
}

func assertNoPublicDynamicJSONType(
	t *testing.T,
	typ reflect.Type,
	seen map[reflect.Type]bool,
) {
	t.Helper()
	if typ == nil || seen[typ] {
		return
	}
	seen[typ] = true
	if typ == reflect.TypeOf(json.RawMessage{}) {
		t.Fatalf("公開型 %s に json.RawMessage があります", typ)
	}
	if typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice ||
		typ.Kind() == reflect.Array {
		assertNoPublicDynamicJSONType(t, typ.Elem(), seen)
		return
	}
	if typ.Kind() == reflect.Map {
		t.Fatalf("公開型 %s に動的 JSON 型があります", typ)
	}
	if typ.Kind() != reflect.Struct {
		return
	}
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		if field.IsExported() {
			assertNoPublicDynamicJSONType(t, field.Type, seen)
		}
	}
}
