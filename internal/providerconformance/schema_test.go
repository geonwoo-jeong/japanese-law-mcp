package providerconformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

const draft202012 = "https://json-schema.org/draft/2020-12/schema"

func TestCanonicalSchemaDraft202012として解決できる(t *testing.T) {
	t.Parallel()

	schemaPath := filepath.Join(testRepositoryRoot(t), "conformance", "provider-capability.schema.json")
	//nolint:gosec // SOT-ENG-017: schemaPath は test file の場所から導出した repository 内の固定 canonical path である。
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("canonical schema を読み込めません: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("canonical schema は JSON でなければなりません: %v", err)
	}
	if got := raw["$schema"]; got != draft202012 {
		t.Fatalf("$schema = %v、期待値は %q です", got, draft202012)
	}
	assertCanonicalRowShape(t, raw)

	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("canonical schema を解釈できません: %v", err)
	}
	if _, err := schema.Resolve(nil); err != nil {
		t.Fatalf("canonical schema を Draft 2020-12 として解決できません: %v", err)
	}
}

func TestCanonicalSchemaは19列と境界制約を検証する(t *testing.T) {
	t.Parallel()

	resolved := resolvedCanonicalSchema(t)
	valid := validSchemaInstance()
	if err := resolved.Validate(valid); err != nil {
		t.Fatalf("正常な matrix を拒否しました: %v", err)
	}

	row := valid["rows"].([]any)[0].(map[string]any)
	if got := len(row); got != 19 {
		t.Fatalf("test fixture の row 列数 = %d、期待値は 19 です", got)
	}

	for column := range row {
		column := column
		t.Run("必須列_"+column, func(t *testing.T) {
			instance := cloneJSONValue(t, valid)
			delete(instance["rows"].([]any)[0].(map[string]any), column)
			assertSchemaRejects(t, resolved, instance)
		})
	}

	tests := map[string]func(map[string]any){
		"schemaVersion": func(instance map[string]any) {
			instance["schemaVersion"] = float64(2)
		},
		"最上位の未知列": func(instance map[string]any) {
			instance["unknown"] = true
		},
		"空のrows": func(instance map[string]any) {
			instance["rows"] = []any{}
		},
		"rowの未知列": func(instance map[string]any) {
			firstRow(instance)["unknown"] = true
		},
		"providerId形式": func(instance map[string]any) {
			firstRow(instance)["providerId"] = "E-Gov"
		},
		"capabilityId形式": func(instance map[string]any) {
			firstRow(instance)["capabilityId"] = "law"
		},
		"majorVersion最小値": func(instance map[string]any) {
			firstRow(instance)["majorVersion"] = float64(0)
		},
		"operation最小長": func(instance map[string]any) {
			firstRow(instance)["operation"] = ""
		},
		"interfaceSotIds最小件数": func(instance map[string]any) {
			firstRow(instance)["interfaceSotIds"] = []any{}
		},
		"interfaceSotIds形式": func(instance map[string]any) {
			firstRow(instance)["interfaceSotIds"] = []any{"IF-004"}
		},
		"interfaceSotIds重複": func(instance map[string]any) {
			firstRow(instance)["interfaceSotIds"] = []any{"SOT-IF-004", "SOT-IF-004"}
		},
		"budgetSotId形式": func(instance map[string]any) {
			firstRow(instance)["budgetSotId"] = "ENG-016"
		},
		"budgetKey形式": func(instance map[string]any) {
			firstRow(instance)["budgetKey"] = "laws_json"
		},
		"concurrencyGroup形式": func(instance map[string]any) {
			firstRow(instance)["concurrencyGroup"] = "eGov"
		},
		"artifactType列挙": func(instance map[string]any) {
			firstRow(instance)["artifactType"] = "CSV"
		},
		"fixtureSet形式": func(instance map[string]any) {
			firstRow(instance)["fixtureSet"] = "LawSearch"
		},
		"requiredCases最小件数": func(instance map[string]any) {
			firstRow(instance)["requiredCases"] = []any{}
		},
		"requiredCases重複": func(instance map[string]any) {
			firstRow(instance)["requiredCases"] = []any{"descriptor", "descriptor"}
		},
		"requiredCases列挙": func(instance map[string]any) {
			firstRow(instance)["requiredCases"] = []any{"unknown"}
		},
		"notApplicableCases理由": func(instance map[string]any) {
			firstRow(instance)["notApplicableCases"] = []any{map[string]any{"case": "authentication", "reason": ""}}
		},
		"supportsContinuation型": func(instance map[string]any) {
			firstRow(instance)["supportsContinuation"] = "true"
		},
		"supportsAuth型": func(instance map[string]any) {
			firstRow(instance)["supportsAuth"] = float64(0)
		},
		"publicErrorSet最小件数": func(instance map[string]any) {
			firstRow(instance)["publicErrorSet"] = []any{}
		},
		"publicErrorSet重複": func(instance map[string]any) {
			firstRow(instance)["publicErrorSet"] = []any{"invalid_argument", "invalid_argument"}
		},
		"publicErrorSet列挙": func(instance map[string]any) {
			firstRow(instance)["publicErrorSet"] = []any{"internal_error"}
		},
		"parserContractVersion形式": func(instance map[string]any) {
			firstRow(instance)["parserContractVersion"] = "version-one"
		},
		"implementedBy形式": func(instance map[string]any) {
			firstRow(instance)["implementedBy"] = "/tmp/provider"
		},
		"conformanceTarget形式": func(instance map[string]any) {
			firstRow(instance)["conformanceTarget"] = "../internal/source/provider"
		},
		"status列挙": func(instance map[string]any) {
			firstRow(instance)["status"] = "ready"
		},
	}

	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			instance := cloneJSONValue(t, valid)
			mutate(instance)
			assertSchemaRejects(t, resolved, instance)
		})
	}
}

func resolvedCanonicalSchema(t *testing.T) *jsonschema.Resolved {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(testRepositoryRoot(t), "conformance", "provider-capability.schema.json"))
	if err != nil {
		t.Fatalf("canonical schema を読み込めません: %v", err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("canonical schema を解釈できません: %v", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("canonical schema を解決できません: %v", err)
	}
	return resolved
}

func validSchemaInstance() map[string]any {
	return map[string]any{
		"schemaVersion": float64(1),
		"rows": []any{
			map[string]any{
				"providerId":            "test-provider",
				"capabilityId":          "law.search",
				"majorVersion":          float64(1),
				"operation":             "GET /laws",
				"interfaceSotIds":       []any{"SOT-IF-004", "SOT-IF-050", "SOT-IF-022"},
				"budgetSotId":           "SOT-IF-004",
				"budgetKey":             "laws-json",
				"concurrencyGroup":      "provider-http",
				"artifactType":          "JSON",
				"fixtureSet":            "law-search-v1",
				"requiredCases":         []any{"descriptor"},
				"notApplicableCases":    []any{},
				"supportsContinuation":  true,
				"supportsAuth":          false,
				"publicErrorSet":        []any{"invalid_argument"},
				"parserContractVersion": "1.0.0",
				"implementedBy":         "github.com/geonwoo-jeong/japanese-law-mcp/internal/source/test/provider",
				"conformanceTarget":     "./internal/source/test/provider",
				"status":                "planned",
			},
		},
	}
}

func cloneJSONValue(t *testing.T, source map[string]any) map[string]any {
	t.Helper()

	data, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("test value を複製できません: %v", err)
	}
	var clone map[string]any
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatalf("test value を復元できません: %v", err)
	}
	return clone
}

func firstRow(instance map[string]any) map[string]any {
	return instance["rows"].([]any)[0].(map[string]any)
}

func assertSchemaRejects(t *testing.T, schema *jsonschema.Resolved, instance map[string]any) {
	t.Helper()

	if err := schema.Validate(instance); err == nil {
		t.Fatal("不正な matrix を受理しました")
	}
}

func assertCanonicalRowShape(t *testing.T, schema map[string]any) {
	t.Helper()

	definitions, ok := schema["$defs"].(map[string]any)
	if !ok {
		t.Fatal("$defs が object ではありません")
	}
	row, ok := definitions["row"].(map[string]any)
	if !ok {
		t.Fatal("$defs.row が object ではありません")
	}
	properties, ok := row["properties"].(map[string]any)
	if !ok {
		t.Fatal("$defs.row.properties が object ではありません")
	}
	requiredValues, ok := row["required"].([]any)
	if !ok {
		t.Fatal("$defs.row.required が array ではありません")
	}
	if got := len(properties); got != 19 {
		t.Fatalf("$defs.row.properties の列数 = %d、期待値は 19 です", got)
	}
	if got := len(requiredValues); got != 19 {
		t.Fatalf("$defs.row.required の列数 = %d、期待値は 19 です", got)
	}
	for _, value := range requiredValues {
		name, ok := value.(string)
		if !ok {
			t.Fatalf("$defs.row.required に文字列以外の値 %v があります", value)
		}
		if _, exists := properties[name]; !exists {
			t.Fatalf("$defs.row.required の %q が properties にありません", name)
		}
	}
	if additional, ok := row["additionalProperties"].(bool); !ok || additional {
		t.Fatal("$defs.row.additionalProperties は false でなければなりません")
	}
}

func testRepositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test file の場所を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
