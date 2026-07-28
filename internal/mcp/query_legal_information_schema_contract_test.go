package mcp

import (
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/google/jsonschema-go/jsonschema"
)

func TestQueryLegalInformationInputSchemaAcceptsOnlyPublicBoundary(t *testing.T) {
	t.Parallel()

	schema := newQueryLegalInformationInputSchema()
	if schema.Type != "object" {
		t.Fatalf("input schema type = %q, want object", schema.Type)
	}
	if len(schema.Required) != 1 || schema.Required[0] != "query" {
		t.Fatalf("input schema required = %v, want [query]", schema.Required)
	}
	if schema.AdditionalProperties == nil {
		t.Fatal("input schema must reject additional properties")
	}

	query := schema.Properties["query"]
	if query == nil || query.Type != "string" {
		t.Fatal("query must be a string schema")
	}
	if query.MinLength == nil || *query.MinLength != 1 {
		t.Fatalf("query minLength = %v, want 1", query.MinLength)
	}
	if got := query.Extra["x-maxUtf8Bytes"]; got != legalquery.MaxQueryBytes {
		t.Fatalf("query x-maxUtf8Bytes = %v, want %d", got, legalquery.MaxQueryBytes)
	}
	if got := query.Extra["x-trimUnicodeWhitespace"]; got != true {
		t.Fatalf("query x-trimUnicodeWhitespace = %v, want true", got)
	}

	limit := schema.Properties["limitPerAttempt"]
	if limit == nil || limit.Type != "integer" {
		t.Fatal("limitPerAttempt must be an integer schema")
	}
	if limit.Minimum == nil || *limit.Minimum != 1 {
		t.Fatalf("limitPerAttempt minimum = %v, want 1", limit.Minimum)
	}
	if limit.Maximum == nil || *limit.Maximum != legalquery.MaxLimitPerAttempt {
		t.Fatalf(
			"limitPerAttempt maximum = %v, want %d",
			limit.Maximum,
			legalquery.MaxLimitPerAttempt,
		)
	}
	if string(limit.Default) != "10" {
		t.Fatalf("limitPerAttempt default = %s, want 10", limit.Default)
	}

	assertQuerySchemaAccepts(t, schema, map[string]any{
		"query": "民法を検索してください",
	})
	assertQuerySchemaAccepts(t, schema, map[string]any{
		"query": "\u3000民法第1条を見せてください\u3000",
	})
	assertQuerySchemaAccepts(t, schema, map[string]any{
		"query":           "裁判例を検索してください",
		"limitPerAttempt": 20,
		"ref": map[string]any{
			"providerId": "courts-go-jp",
			"key": map[string]any{
				"sourceId":     "Courts:official/search-v1",
				"resourceType": "judicial-decision",
				"resourceId":   "decision/95570",
			},
		},
	})
	assertQuerySchemaAccepts(t, schema, map[string]any{
		"query": "法令を読んでください",
		"ref": map[string]any{
			"providerId": strings.Repeat("a", 65),
			"key": map[string]any{
				"sourceId":     "opaque-source",
				"resourceType": "law",
				"resourceId":   "law-1",
			},
		},
	})

	for name, instance := range map[string]any{
		"missing query": map[string]any{},
		"null query": map[string]any{
			"query": nil,
		},
		"control character": map[string]any{
			"query": "民法\n検索",
		},
		"unicode whitespace only": map[string]any{
			"query": "\u3000\u00a0",
		},
		"fractional limit": map[string]any{
			"query":           "民法",
			"limitPerAttempt": 1.5,
		},
		"limit too large": map[string]any{
			"query":           "民法",
			"limitPerAttempt": 21,
		},
		"unknown input": map[string]any{
			"query": "民法",
			"score": 10,
		},
		"unsupported ref type": map[string]any{
			"query": "民法",
			"ref": map[string]any{
				"providerId": "example",
				"key": map[string]any{
					"sourceId":     "example",
					"resourceType": "tax-statistics",
					"resourceId":   "1",
				},
			},
		},
		"nested unknown input": map[string]any{
			"query": "民法",
			"ref": map[string]any{
				"providerId": "example",
				"key": map[string]any{
					"sourceId":     "example",
					"resourceType": "law",
					"resourceId":   "1",
					"continuation": "secret",
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assertQuerySchemaRejects(t, schema, instance)
		})
	}
}

func TestQueryLegalInformationOutputSchemaIsClosedDiscriminatedUnion(t *testing.T) {
	t.Parallel()

	schema := newQueryLegalInformationOutputSchema()
	if schema.Type != "object" {
		t.Fatalf("result schema type = %q, want object", schema.Type)
	}
	if got := len(schema.OneOf); got != 6 {
		t.Fatalf("result oneOf variant count = %d, want 6", got)
	}
	attempt := schema.Defs["LegalQueryAttempt"]
	if attempt == nil {
		t.Fatal("$defs.LegalQueryAttempt is required")
	}
	if got := len(attempt.OneOf); got != 8 {
		t.Fatalf("attempt oneOf variant count = %d, want 8", got)
	}

	for _, name := range []string{
		"LegalQueryCompletedResult",
		"LegalQueryEmptyResult",
		"LegalQueryPartialResult",
		"LegalQueryNeedsClarificationResult",
		"LegalQueryCapabilityUnavailableResult",
		"LegalQueryUnsupportedResult",
		"LegalQueryLawSearchAttempt",
		"LegalQueryLawContentSearchAttempt",
		"LegalQueryLawDocumentAttempt",
		"LegalQueryLawArticleAttempt",
		"LegalQueryLawUpdatesAttempt",
		"LegalQueryJudicialSearchAttempt",
		"LegalQueryJudicialDecisionAttempt",
		"LegalQueryFailedAttempt",
	} {
		if schema.Defs[name] == nil {
			t.Errorf("$defs.%s is required", name)
		}
	}
}

func TestQueryLegalInformationOutputSchemaValidatesNonExecutionResults(t *testing.T) {
	t.Parallel()

	schema := newQueryLegalInformationOutputSchema()
	assertQuerySchemaAccepts(t, schema, map[string]any{
		"status":          string(legalquery.LegalQueryResultStatusNeedsClarification),
		"decision":        string(legalquery.LegalQueryResultDecisionNoExecution),
		"language":        "ja",
		"interpretations": []any{},
		"attempts":        []any{},
		"notices":         []any{},
		"clarification": map[string]any{
			"reasonCodes": []any{
				string(legalquery.ReasonCodeBelowExecutionThreshold),
			},
			"questions": []any{
				string(legalquery.LegalQueryQuestionTask),
			},
		},
	})
	assertQuerySchemaAccepts(t, schema, map[string]any{
		"status":          string(legalquery.LegalQueryResultStatusUnsupported),
		"decision":        string(legalquery.LegalQueryResultDecisionNoExecution),
		"language":        "ja",
		"interpretations": []any{},
		"attempts":        []any{},
		"notices": []any{
			"日本語の法情報取得要求を入力してください。",
		},
	})
	standaloneUnsupported := map[string]any{
		"status":          string(legalquery.LegalQueryResultStatusUnsupported),
		"decision":        string(legalquery.LegalQueryResultDecisionNoExecution),
		"language":        "ja",
		"interpretations": []any{},
		"attempts":        []any{},
		"notices": []any{
			legalquery.LegalQueryStandaloneStructuredNotice,
		},
	}
	assertQuerySchemaAccepts(t, schema, standaloneUnsupported)
	assertQuerySchemaAccepts(t, schema, map[string]any{
		"status": string(
			legalquery.LegalQueryResultStatusCapabilityUnavailable,
		),
		"decision": string(
			legalquery.LegalQueryResultDecisionNoExecution,
		),
		"language": "ja",
		"interpretations": []any{
			map[string]any{
				"interpretationId": "interpretation-1",
				"confidence":       string(legalquery.ConfidenceLow),
				"evidenceCodes": []any{
					string(legalquery.EvidenceExplicitResource),
				},
				"conceptSources": []any{},
				"availability": string(
					legalquery.SelectionAvailabilityPackDisabled,
				),
				"requiredPacks": []any{
					"judicial-cases",
					"legislative-history",
					"tax",
				},
				"steps": []any{
					map[string]any{
						"stepId":                 "step-1",
						"task":                   string(legalquery.TaskSearch),
						"resource":               string(legalquery.ResourceJudicialDecision),
						"capabilityId":           "judicial-decision.search",
						"capabilityMajorVersion": 1,
					},
				},
			},
		},
		"attempts": []any{},
		"notices": []any{
			legalquery.LegalQueryPackDisabledNotice,
		},
	})

	for name, instance := range map[string]any{
		"unsupported with interpretation": map[string]any{
			"status":   string(legalquery.LegalQueryResultStatusUnsupported),
			"decision": string(legalquery.LegalQueryResultDecisionNoExecution),
			"language": "ja",
			"interpretations": []any{
				map[string]any{},
			},
			"attempts": []any{},
			"notices": []any{
				"日本語の法情報取得要求を入力してください。",
			},
		},
		"non-execution with attempt": map[string]any{
			"status":          string(legalquery.LegalQueryResultStatusNeedsClarification),
			"decision":        string(legalquery.LegalQueryResultDecisionNoExecution),
			"language":        "ja",
			"interpretations": []any{},
			"attempts": []any{
				map[string]any{},
			},
			"notices": []any{},
			"clarification": map[string]any{
				"reasonCodes": []any{
					string(legalquery.ReasonCodeBelowExecutionThreshold),
				},
				"questions": []any{
					string(legalquery.LegalQueryQuestionTask),
				},
			},
		},
		"unknown output field": map[string]any{
			"status":          string(legalquery.LegalQueryResultStatusUnsupported),
			"decision":        string(legalquery.LegalQueryResultDecisionNoExecution),
			"language":        "ja",
			"interpretations": []any{},
			"attempts":        []any{},
			"notices": []any{
				"日本語の法情報取得要求を入力してください。",
			},
			"candidateId": "candidate-1",
		},
		"standalone notice と他 notice の併存": map[string]any{
			"status":          string(legalquery.LegalQueryResultStatusUnsupported),
			"decision":        string(legalquery.LegalQueryResultDecisionNoExecution),
			"language":        "ja",
			"interpretations": []any{},
			"attempts":        []any{},
			"notices": []any{
				legalquery.LegalQueryStandaloneStructuredNotice,
				legalquery.LegalQueryNonJapaneseNotice,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assertQuerySchemaRejects(t, schema, instance)
		})
	}
}

func assertQuerySchemaAccepts(
	t *testing.T,
	schema *jsonschema.Schema,
	instance any,
) {
	t.Helper()
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("schema.Resolve() error = %v", err)
	}
	if err := resolved.Validate(instance); err != nil {
		t.Fatalf("schema rejected valid instance: %v", err)
	}
}

func assertQuerySchemaRejects(
	t *testing.T,
	schema *jsonschema.Schema,
	instance any,
) {
	t.Helper()
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("schema.Resolve() error = %v", err)
	}
	if err := resolved.Validate(instance); err == nil {
		t.Fatal("schema accepted invalid instance")
	}
}
