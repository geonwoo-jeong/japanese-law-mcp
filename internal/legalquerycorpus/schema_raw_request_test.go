package legalquerycorpus

import (
	"strings"
	"testing"
)

func TestCorpusSchemaV1は境界違反のRawRequest値を保持する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	for name, query := range map[string]string{
		"query上限超過": strings.Repeat("a", 2049),
		"ASCII制御文字": "行政手続法\nを検索",
	} {
		name, query := name, query
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			instance := validRequestErrorCase()
			instance["request"].(map[string]any)["query"] = query
			assertSchemaAccepts(t, schema, instance)
		})
	}
}

func TestCorpusSchemaV1はRawRequestの型違反を拒否する(t *testing.T) {
	t.Parallel()

	_, schema := resolvedCorpusSchemaV1(t)
	tests := map[string]func(map[string]any){
		"queryがnull": func(request map[string]any) {
			request["query"] = nil
		},
		"queryが数値": func(request map[string]any) {
			request["query"] = float64(1)
		},
		"limitが文字列": func(request map[string]any) {
			request["limitPerAttempt"] = "20"
		},
		"refがnull": func(request map[string]any) {
			request["ref"] = nil
		},
		"refのkeyがnull": func(request map[string]any) {
			request["ref"] = map[string]any{
				"providerId": "provider-a",
				"key":        nil,
			}
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			instance := validRequestErrorCase()
			mutate(instance["request"].(map[string]any))
			assertSchemaRejects(t, schema, instance)
		})
	}
}

func validRequestErrorCase() map[string]any {
	instance := validSemanticCase(validLawSearchStep())
	instance["caseId"] = "development-request-error"
	instance["coverageIds"] = []any{"input-query-empty"}
	instance["request"] = map[string]any{"query": ""}
	instance["expected"] = map[string]any{
		"kind":      "request_error",
		"errorCode": "invalid_argument",
		"field":     "query",
	}
	return instance
}
