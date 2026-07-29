package legalquerycorpus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

const corpusDraft202012 = "https://json-schema.org/draft/2020-12/schema"

func TestCorpusSchemaV1はDraft202012として自己解決できる(t *testing.T) {
	t.Parallel()

	raw, resolved := resolvedCorpusSchemaV1(t)
	if got := raw["$schema"]; got != corpusDraft202012 {
		t.Fatalf("$schema = %v、期待値は %q です", got, corpusDraft202012)
	}
	assertSchemaObjectsClosed(t, raw, "$")
	assertSchemaReferencesAreLocal(t, raw, "$")
	if resolved == nil {
		t.Fatal("schema の解決結果が nil です")
	}
}

func TestCorpusSchemaV1は三つの成果物を受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	instances := map[string]map[string]any{
		"manifest":      validManifest(),
		"semantic_case": validSemanticCase(validLawSearchStep()),
		"execution_case": validExecutionCase(
			map[string]any{"kind": "collection_success", "sourceItemCount": float64(3)},
			validResultExpectation("completed"),
		),
	}
	for name, instance := range instances {
		name, instance := name, instance
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertSchemaAccepts(t, schema, instance)
		})
	}
}

func TestCorpusSchemaV1はSemantic期待値の二つのvariantを受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	plan := validSemanticCase(validLawSearchStep())
	requestError := validSemanticCase(validLawSearchStep())
	requestError["caseId"] = "development-empty-query"
	requestError["coverageIds"] = []any{"input-query-empty"}
	requestError["request"] = map[string]any{
		"query":           "",
		"limitPerAttempt": float64(0),
		"ref": map[string]any{
			"providerId": "",
			"key": map[string]any{
				"sourceId":     "",
				"resourceType": "",
				"resourceId":   "",
			},
		},
	}
	requestError["expected"] = map[string]any{
		"kind":      "request_error",
		"errorCode": "invalid_argument",
		"field":     "query",
	}

	assertSchemaAccepts(t, schema, plan)
	assertSchemaAccepts(t, schema, requestError)
}

func TestCorpusSchemaV1はUnsupportedで未選択の取得候補を保持できる(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	for _, keepMeanings := range []bool{false, true} {
		instance := validSemanticCase(validLawSearchStep())
		expected := instance["expected"].(map[string]any)
		expected["decision"] = "unsupported"
		expected["reasonCodes"] = []any{"mixed_unsupported_intent"}
		expected["selectedMeaningIds"] = []any{}
		if !keepMeanings {
			expected["meanings"] = []any{}
		}
		assertSchemaAccepts(t, schema, instance)
	}
}

func TestCorpusSchemaV1は七つのLogicalInputを受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	steps := map[string]map[string]any{
		"law_search":               validLawSearchStep(),
		"law_content_search":       validLawContentSearchStep(),
		"law_read":                 validLawReadStep(),
		"law_article_read":         validLawArticleReadStep(),
		"law_updates":              validLawUpdatesStep(),
		"judicial_decision_search": validJudicialSearchStep(),
		"judicial_decision_read":   validJudicialReadStep(),
	}
	for name, step := range steps {
		name, step := name, step
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertSchemaAccepts(t, schema, validSemanticCase(step))
		})
	}
}

func TestCorpusSchemaV1はExecutionのvariantを受理する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	outcomes := map[string]map[string]any{
		"collection_success": {"kind": "collection_success", "sourceItemCount": float64(3)},
		"read_success":       {"kind": "read_success"},
		"failure":            {"kind": "failure", "errorCode": "source_unavailable"},
		"timeout":            {"kind": "timeout"},
	}
	for name, outcome := range outcomes {
		name, outcome := name, outcome
		t.Run("action_"+name, func(t *testing.T) {
			t.Parallel()
			assertSchemaAccepts(
				t,
				schema,
				validExecutionCase(outcome, validResultExpectation("completed")),
			)
		})
	}

	expectations := map[string]map[string]any{
		"completed_result": validResultExpectation("completed"),
		"empty_result": {
			"terminal":          "result",
			"status":            "empty",
			"returnedItemCount": float64(0),
			"attempts": []any{map[string]any{
				"meaningId":          "law-search",
				"stepOrdinal":        float64(1),
				"outcome":            "empty",
				"publishedItemCount": float64(0),
				"hasMore":            false,
			}},
		},
		"partial_result": {
			"terminal":          "result",
			"status":            "partial",
			"returnedItemCount": float64(1),
			"attempts": []any{
				map[string]any{
					"meaningId":          "law-search",
					"stepOrdinal":        float64(1),
					"outcome":            "completed",
					"publishedItemCount": float64(1),
					"hasMore":            false,
				},
				map[string]any{
					"meaningId":   "law-search",
					"stepOrdinal": float64(2),
					"outcome":     "failed",
					"errorCode":   "source_timeout",
				},
			},
		},
		"terminal_error": {
			"terminal":  "error",
			"errorCode": "source_unavailable",
			"attempts": []any{map[string]any{
				"meaningId":   "law-search",
				"stepOrdinal": float64(1),
				"outcome":     "failed",
				"errorCode":   "source_unavailable",
			}},
		},
	}
	for name, expected := range expectations {
		name, expected := name, expected
		t.Run("expected_"+name, func(t *testing.T) {
			t.Parallel()
			assertSchemaAccepts(
				t,
				schema,
				validExecutionCase(
					map[string]any{"kind": "collection_success", "sourceItemCount": float64(3)},
					expected,
				),
			)
		})
	}
}

func TestCorpusSchemaV1はSafetyVariantの条件を検証する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	for _, coverageID := range safetyCoverageIDs() {
		for _, variant := range []string{"ordinary", "adversarial"} {
			instance := validSemanticCase(validLawSearchStep())
			instance["coverageIds"] = []any{coverageID}
			instance["safetyVariant"] = variant
			assertSchemaAccepts(t, schema, instance)
		}

		missing := validSemanticCase(validLawSearchStep())
		missing["coverageIds"] = []any{coverageID}
		assertSchemaRejects(t, schema, missing)
	}

	unexpected := validSemanticCase(validLawSearchStep())
	unexpected["safetyVariant"] = "ordinary"
	assertSchemaRejects(t, schema, unexpected)
}

func TestCorpusSchemaV1はLogicalInputとMeaningの条件違反を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	tests := map[string]func() map[string]any{
		"本文検索語が空": func() map[string]any {
			instance := validSemanticCase(validLawContentSearchStep())
			input := firstSemanticStep(instance)["logicalInput"].(map[string]any)
			input["allTerms"] = []any{}
			input["anyTerms"] = []any{}
			return instance
		},
		"revisionIdとasOfを併用": func() map[string]any {
			instance := validSemanticCase(validLawReadStep())
			firstSemanticStep(instance)["logicalInput"].(map[string]any)["asOf"] = "2026-07-28"
			return instance
		},
		"法概念にconceptIdsがない": func() map[string]any {
			instance := validSemanticCase(validLawSearchStep())
			meaning := firstExpectedMeaning(instance)
			meaning["evidenceCodes"] = []any{"legal_concept"}
			return instance
		},
		"法概念でないのにconceptIdsがある": func() map[string]any {
			instance := validSemanticCase(validLawSearchStep())
			firstExpectedMeaning(instance)["conceptIds"] = []any{"permanent-residence"}
			return instance
		},
	}
	for name, build := range tests {
		name, build := name, build
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertSchemaRejects(t, schema, build())
		})
	}
}

func TestCorpusSchemaV1は固定一覧と識別子の変更を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	tests := map[string]func() map[string]any{
		"カテゴリ不足": func() map[string]any {
			instance := validManifest()
			instance["requiredCategoryIds"] = stringValues(requiredCategoryIDs()[:11]...)
			return instance
		},
		"カテゴリ追加": func() map[string]any {
			instance := validManifest()
			instance["requiredCategoryIds"] = stringValues(append(requiredCategoryIDs(), "unknown")...)
			return instance
		},
		"カテゴリ順序変更": func() map[string]any {
			instance := validManifest()
			values := requiredCategoryIDs()
			values[0], values[1] = values[1], values[0]
			instance["requiredCategoryIds"] = stringValues(values...)
			return instance
		},
		"scenario不足": func() map[string]any {
			instance := validManifest()
			instance["requiredExecutionScenarioIds"] = stringValues(requiredExecutionScenarioIDs()[:6]...)
			return instance
		},
		"scenario追加": func() map[string]any {
			instance := validManifest()
			values := append(requiredExecutionScenarioIDs(), "execution-unknown")
			instance["requiredExecutionScenarioIds"] = stringValues(values...)
			return instance
		},
		"scenario順序変更": func() map[string]any {
			instance := validManifest()
			values := requiredExecutionScenarioIDs()
			values[0], values[1] = values[1], values[0]
			instance["requiredExecutionScenarioIds"] = stringValues(values...)
			return instance
		},
		"corpusVersionが非正規形": func() map[string]any {
			instance := validManifest()
			instance["corpusVersion"] = "corpus-v01"
			return instance
		},
		"semantic caseIdが不正": func() map[string]any {
			instance := validSemanticCase(validLawSearchStep())
			instance["caseId"] = "semantic-law-search"
			return instance
		},
		"execution caseIdが不正": func() map[string]any {
			instance := validExecutionCase(
				map[string]any{"kind": "read_success"},
				validResultExpectation("completed"),
			)
			instance["caseId"] = "holdout-law-search"
			return instance
		},
		"semanticCaseIdがholdout": func() map[string]any {
			instance := validExecutionCase(
				map[string]any{"kind": "read_success"},
				validResultExpectation("completed"),
			)
			instance["semanticCaseId"] = "holdout-law-search"
			return instance
		},
	}
	for name, build := range tests {
		name, build := name, build
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertSchemaRejects(t, schema, build())
		})
	}
}

func TestCorpusSchemaV1は未知値と交差variantを拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	tests := map[string]func() map[string]any{
		"未知artifact": func() map[string]any {
			instance := validManifest()
			instance["artifactKind"] = "unknown"
			return instance
		},
		"未知coverage": func() map[string]any {
			instance := validSemanticCase(validLawSearchStep())
			instance["coverageIds"] = []any{"unknown-coverage"}
			return instance
		},
		"logical input交差": func() map[string]any {
			instance := validSemanticCase(validLawSearchStep())
			step := firstSemanticStep(instance)
			step["resource"] = "judicial_decision"
			step["logicalInput"] = map[string]any{"query": "裁判例"}
			return instance
		},
		"planにrequest_error項目": func() map[string]any {
			instance := validSemanticCase(validLawSearchStep())
			instance["expected"].(map[string]any)["errorCode"] = "invalid_argument"
			return instance
		},
		"action outcome交差": func() map[string]any {
			return validExecutionCase(
				map[string]any{
					"kind":            "collection_success",
					"sourceItemCount": float64(1),
					"errorCode":       "source_unavailable",
				},
				validResultExpectation("completed"),
			)
		},
		"terminal error交差": func() map[string]any {
			expected := map[string]any{
				"terminal":  "error",
				"errorCode": "source_unavailable",
				"status":    "partial",
				"attempts": []any{map[string]any{
					"meaningId":   "law-search",
					"stepOrdinal": float64(1),
					"outcome":     "failed",
					"errorCode":   "source_unavailable",
				}},
			}
			return validExecutionCase(
				map[string]any{"kind": "failure", "errorCode": "source_unavailable"},
				expected,
			)
		},
		"request ref未知項目": func() map[string]any {
			instance := validSemanticCase(validLawSearchStep())
			instance["request"].(map[string]any)["ref"] = validResourceRef("law")
			instance["request"].(map[string]any)["ref"].(map[string]any)["unknown"] = true
			return instance
		},
		"最上位未知項目": func() map[string]any {
			instance := validSemanticCase(validLawSearchStep())
			instance["unknown"] = true
			return instance
		},
	}
	for name, build := range tests {
		name, build := name, build
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertSchemaRejects(t, schema, build())
		})
	}
}

func TestCorpusSchemaV1は全成果物の未知versionを拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	instances := []map[string]any{
		validManifest(),
		validSemanticCase(validLawSearchStep()),
		validExecutionCase(
			map[string]any{"kind": "collection_success", "sourceItemCount": float64(1)},
			validResultExpectation("completed"),
		),
	}
	for _, instance := range instances {
		instance["schemaVersion"] = float64(2)
		assertSchemaRejects(t, schema, instance)
	}
}

func resolvedCorpusSchemaV1(t *testing.T) (map[string]any, *jsonschema.Resolved) {
	t.Helper()

	data, err := os.ReadFile(corpusSchemaV1Path(t))
	if err != nil {
		t.Fatalf("SOT-ENG-026: corpus schema v1 を読み込めません: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("SOT-ENG-026: corpus schema v1 は JSON でなければなりません: %v", err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("SOT-ENG-026: corpus schema v1 を解釈できません: %v", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("SOT-ENG-026: corpus schema v1 を自己解決できません: %v", err)
	}
	return raw, resolved
}

func corpusSchemaV1Path(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test file の場所を取得できません")
	}
	return filepath.Join(
		filepath.Dir(filename),
		"..",
		"..",
		"testdata",
		"legalquery",
		"schemas",
		"legal-query-corpus-v1.schema.json",
	)
}

func assertSchemaObjectsClosed(t *testing.T, value any, path string) {
	t.Helper()

	switch current := value.(type) {
	case map[string]any:
		if current["type"] == "object" {
			closed, ok := current["additionalProperties"].(bool)
			if !ok || closed {
				t.Fatalf("%s の object schema は additionalProperties=false ではありません", path)
			}
		}
		for key, child := range current {
			assertSchemaObjectsClosed(t, child, path+"."+key)
		}
	case []any:
		for index, child := range current {
			assertSchemaObjectsClosed(t, child, path+"["+strconv.Itoa(index)+"]")
		}
	}
}

func assertSchemaReferencesAreLocal(t *testing.T, value any, path string) {
	t.Helper()

	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			if key == "$ref" {
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#/") {
					t.Fatalf("%s.$ref = %v、local fragment ではありません", path, child)
				}
			}
			assertSchemaReferencesAreLocal(t, child, path+"."+key)
		}
	case []any:
		for index, child := range current {
			assertSchemaReferencesAreLocal(t, child, path+"["+strconv.Itoa(index)+"]")
		}
	}
}

func assertSchemaAccepts(t *testing.T, schema *jsonschema.Resolved, instance map[string]any) {
	t.Helper()

	if err := schema.Validate(instance); err != nil {
		t.Fatalf("正常な成果物を拒否しました: %v", err)
	}
}

func assertSchemaRejects(t *testing.T, schema *jsonschema.Resolved, instance map[string]any) {
	t.Helper()

	if err := schema.Validate(instance); err == nil {
		t.Fatal("不正な成果物を受理しました")
	}
}

func validManifest() map[string]any {
	digest := strings.Repeat("a", 64)
	return map[string]any{
		"artifactKind":                 "corpus_manifest",
		"schemaVersion":                float64(1),
		"corpusVersion":                "corpus-v4",
		"seed":                         float64(20260728),
		"holdoutDigest":                digest,
		"requiredCategoryIds":          stringValues(requiredCategoryIDs()...),
		"requiredExecutionScenarioIds": stringValues(requiredExecutionScenarioIDs()...),
		"sets": map[string]any{
			"development": validManifestSet("development-law-search", digest),
			"holdout":     validManifestSet("holdout-law-search", strings.Repeat("b", 64)),
			"execution":   validManifestSet("execution-law-search", strings.Repeat("c", 64)),
		},
	}
}

func validManifestSet(caseID, digest string) map[string]any {
	return map[string]any{
		"caseCount": float64(1),
		"cases": []any{map[string]any{
			"caseId": caseID,
			"sha256": digest,
		}},
	}
}

func validSemanticCase(step map[string]any) map[string]any {
	return map[string]any{
		"artifactKind":   "semantic_case",
		"schemaVersion":  float64(1),
		"caseId":         "development-law-search",
		"leakageGroupId": "law-search",
		"coverageIds":    []any{"intent-law-search"},
		"enabledPacks":   []any{},
		"request": map[string]any{
			"query":           "行政手続法を検索",
			"limitPerAttempt": float64(20),
		},
		"expected": map[string]any{
			"kind":               "plan",
			"decision":           "single",
			"reasonCodes":        []any{"single_clear_candidate"},
			"selectedMeaningIds": []any{"law-search"},
			"meanings": []any{map[string]any{
				"meaningId":     "law-search",
				"evidenceCodes": []any{"explicit_task", "explicit_resource"},
				"conceptIds":    []any{},
				"requiredPacks": []any{},
				"steps":         []any{step},
			}},
		},
	}
}

func validLawSearchStep() map[string]any {
	return map[string]any{
		"task":      "search",
		"resource":  "law",
		"inputKind": "law_search",
		"logicalInput": map[string]any{
			"query": "行政手続法",
			"asOf":  "2026-07-28",
		},
	}
}

func validLawContentSearchStep() map[string]any {
	return map[string]any{
		"task":      "search",
		"resource":  "law_provision",
		"inputKind": "law_content_search",
		"logicalInput": map[string]any{
			"allTerms":     []any{"永住許可"},
			"anyTerms":     []any{"在留資格"},
			"excludeTerms": []any{},
			"asOf":         "2026-07-28",
		},
	}
}

func validLawReadStep() map[string]any {
	return map[string]any{
		"task":      "read",
		"resource":  "law",
		"inputKind": "law_read",
		"logicalInput": map[string]any{
			"lawId":      "503AC0000000037",
			"revisionId": "503AC0000000037_20250601",
		},
	}
}

func validLawArticleReadStep() map[string]any {
	return map[string]any{
		"task":      "read",
		"resource":  "law_provision",
		"inputKind": "law_article_read",
		"logicalInput": map[string]any{
			"lawId": "503AC0000000037",
			"location": map[string]any{
				"provision":       "main",
				"articleNumber":   "22_2",
				"paragraphNumber": float64(1),
			},
			"asOf": "2026-07-28",
		},
	}
}

func validLawUpdatesStep() map[string]any {
	return map[string]any{
		"task":         "list_updates",
		"resource":     "law",
		"inputKind":    "law_updates",
		"logicalInput": map[string]any{"date": "2026-07-28"},
	}
}

func validJudicialSearchStep() map[string]any {
	return map[string]any{
		"task":         "search",
		"resource":     "judicial_decision",
		"inputKind":    "judicial_decision_search",
		"logicalInput": map[string]any{"query": "永住許可"},
	}
}

func validJudicialReadStep() map[string]any {
	return map[string]any{
		"task":         "read",
		"resource":     "judicial_decision",
		"inputKind":    "judicial_decision_read",
		"logicalInput": map[string]any{"ref": validResourceRef("judicial-decision")},
	}
}

func validResourceRef(resourceType string) map[string]any {
	return map[string]any{
		"providerId": "test-provider",
		"key": map[string]any{
			"sourceId":     "test-source",
			"resourceType": resourceType,
			"resourceId":   "resource-1",
		},
	}
}

func validExecutionCase(outcome, expected map[string]any) map[string]any {
	return map[string]any{
		"artifactKind":   "execution_case",
		"schemaVersion":  float64(1),
		"caseId":         "execution-law-search",
		"scenarioIds":    []any{"execution-nonempty"},
		"semanticCaseId": "development-law-search",
		"actions": []any{map[string]any{
			"meaningId":    "law-search",
			"stepOrdinal":  float64(1),
			"releaseOrder": float64(1),
			"outcome":      outcome,
		}},
		"expected": expected,
	}
}

func validResultExpectation(status string) map[string]any {
	return map[string]any{
		"terminal":          "result",
		"status":            status,
		"returnedItemCount": float64(1),
		"attempts": []any{map[string]any{
			"meaningId":          "law-search",
			"stepOrdinal":        float64(1),
			"outcome":            "completed",
			"publishedItemCount": float64(1),
			"hasMore":            false,
		}},
	}
}

func firstSemanticStep(instance map[string]any) map[string]any {
	return firstExpectedMeaning(instance)["steps"].([]any)[0].(map[string]any)
}

func firstExpectedMeaning(instance map[string]any) map[string]any {
	expected := instance["expected"].(map[string]any)
	return expected["meanings"].([]any)[0].(map[string]any)
}

func stringValues(values ...string) []any {
	result := make([]any, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func requiredCategoryIDs() []string {
	return []string{
		"ambiguity",
		"budget-boundary",
		"capability-intent",
		"input-boundary",
		"law-name-and-concept",
		"official-reference",
		"pack-state",
		"safety-execution-boundary",
		"structured-location-and-date",
		"surface-variation",
		"typo-variation",
		"unsupported-scope",
	}
}

func requiredExecutionScenarioIDs() []string {
	return []string{
		"execution-all-failed",
		"execution-empty",
		"execution-item-budget",
		"execution-mixed-composition",
		"execution-nonempty",
		"execution-partial-failure",
		"execution-reversed-completion",
		"execution-timeout",
	}
}

func safetyCoverageIDs() []string {
	return []string{
		"boundary-budget-limit",
		"boundary-mixed-unsupported",
		"boundary-no-implicit-first-read",
		"boundary-non-japanese",
		"boundary-pack-disabled",
	}
}
