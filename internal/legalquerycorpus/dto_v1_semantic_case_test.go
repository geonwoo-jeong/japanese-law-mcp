package legalquerycorpus

import (
	"reflect"
	"testing"
)

func TestSemanticCaseV1DecodeはplanとrequestErrorを復元する(t *testing.T) {
	t.Parallel()

	for name, source := range map[string]map[string]any{
		"plan": validSemanticCase(validLawSearchStep()),
		"request error": {
			"artifactKind":   "semantic_case",
			"schemaVersion":  float64(1),
			"caseId":         "holdout-query-empty",
			"leakageGroupId": "query-empty",
			"coverageIds":    []any{"input-query-empty"},
			"enabledPacks":   []any{},
			"request":        map[string]any{"query": ""},
			"expected": map[string]any{
				"kind":      "request_error",
				"errorCode": "invalid_argument",
				"field":     "query",
			},
		},
	} {
		name, source := name, source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			semanticCase, err := decodeSemanticCaseV1(mustJSONBytes(t, source))
			if err != nil {
				t.Fatalf("SOT-ENG-026: semantic case decode error = %v", err)
			}
			if semanticCase.ArtifactKind() != ArtifactKindSemanticCase {
				t.Fatalf("SOT-ENG-026: artifactKind = %q", semanticCase.ArtifactKind())
			}
		})
	}
}

func TestSemanticCaseV1Decodeは必須項目と未知項目を拒否する(t *testing.T) {
	t.Parallel()

	required := []string{
		"artifactKind",
		"schemaVersion",
		"caseId",
		"leakageGroupId",
		"coverageIds",
		"enabledPacks",
		"request",
		"expected",
	}
	for _, field := range required {
		field := field
		t.Run(field+"欠落", func(t *testing.T) {
			t.Parallel()
			source := validSemanticCase(validLawSearchStep())
			delete(source, field)
			if _, err := decodeSemanticCaseV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 必須項目欠落を受理した")
			}
		})
	}

	tests := map[string]func(map[string]any){
		"top-level未知項目": func(source map[string]any) {
			source["profileVersion"] = "forbidden"
		},
		"request未知項目": func(source map[string]any) {
			source["request"].(map[string]any)["normalizedQuery"] = "forbidden"
		},
		"expected未知項目": func(source map[string]any) {
			source["expected"].(map[string]any)["candidateId"] = "forbidden"
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source := validSemanticCase(validLawSearchStep())
			mutate(source)
			if _, err := decodeSemanticCaseV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 未知項目を受理した")
			}
		})
	}
	trailing := append(
		mustJSONBytes(t, validSemanticCase(validLawSearchStep())),
		[]byte(` {}`)...,
	)
	if _, err := decodeSemanticCaseV1(trailing); err == nil {
		t.Fatal("SOT-ENG-026: 末尾の別 JSON 値を受理した")
	}
}

func TestSemanticCaseV1DecodeはcrossField違反を拒否する(t *testing.T) {
	t.Parallel()

	planWithInvalidRequest := validSemanticCase(validLawSearchStep())
	planWithInvalidRequest["request"].(map[string]any)["query"] = ""

	requestErrorWithValidRequest := validSemanticCase(validLawSearchStep())
	requestErrorWithValidRequest["expected"] = map[string]any{
		"kind":      "request_error",
		"errorCode": "invalid_argument",
		"field":     "query",
	}

	disabledSingle := validSemanticCase(validJudicialSearchStep())
	disabledMeaning := disabledSingle["expected"].(map[string]any)["meanings"].([]any)[0].(map[string]any)
	disabledMeaning["meaningId"] = "judicial-search"
	disabledMeaning["requiredPacks"] = []any{"judicial-cases"}
	disabledSingle["expected"].(map[string]any)["selectedMeaningIds"] =
		[]any{"judicial-search"}

	unavailableButEnabled := validSemanticCase(validJudicialSearchStep())
	unavailableButEnabled["enabledPacks"] = []any{"judicial-cases"}
	unavailableExpected := unavailableButEnabled["expected"].(map[string]any)
	unavailableExpected["decision"] = "capability_unavailable"
	unavailableExpected["reasonCodes"] = []any{"required_pack_disabled"}
	unavailableExpected["selectedMeaningIds"] = []any{"judicial-search"}
	unavailableMeaning := unavailableExpected["meanings"].([]any)[0].(map[string]any)
	unavailableMeaning["meaningId"] = "judicial-search"
	unavailableMeaning["requiredPacks"] = []any{"judicial-cases"}

	for name, source := range map[string]map[string]any{
		"planと不正request":          planWithInvalidRequest,
		"request errorと有効request": requestErrorWithValidRequest,
		"singleと無効pack":           disabledSingle,
		"unavailableと有効pack":      unavailableButEnabled,
	} {
		name, source := name, source
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := decodeSemanticCaseV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: cross-field 違反を受理した")
			}
		})
	}
}

func TestSemanticCase公開型は動的JSONを保持しない(t *testing.T) {
	t.Parallel()

	for _, current := range []reflect.Type{
		reflect.TypeOf(SemanticCase{}),
		reflect.TypeOf(SemanticCaseValues{}),
	} {
		for index := 0; index < current.NumField(); index++ {
			field := current.Field(index)
			if field.Type.Kind() == reflect.Map ||
				field.Type.String() == "json.RawMessage" ||
				field.Type.String() == "[]json.RawMessage" {
				t.Fatalf(
					"SOT-ENG-026: 公開型 %s.%s が動的 JSON を保持する",
					current.Name(),
					field.Name,
				)
			}
		}
	}
}

func mustSemanticCaseFromMap(t *testing.T, source map[string]any) SemanticCase {
	t.Helper()
	semanticCase, err := decodeSemanticCaseV1(mustJSONBytes(t, source))
	if err != nil {
		t.Fatalf("SOT-ENG-026: SemanticCase を作成できません: %v", err)
	}
	return semanticCase
}
