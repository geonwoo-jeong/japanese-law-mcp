package mcp

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func querySchemaSourcedLawSummary() map[string]any {
	return querySchemaSourced(
		querySchemaLawKey(),
		querySchemaLawSummary(),
		"",
	)
}

func querySchemaSourcedLawContent() map[string]any {
	return querySchemaSourced(
		querySchemaLawKey(),
		map[string]any{
			"law":      querySchemaLawSummary(),
			"location": "第一条",
			"text":     "法令本文",
			"citation": querySchemaCitation("第一条"),
		},
		"第一条",
	)
}

func querySchemaSourcedLawDocument() map[string]any {
	return querySchemaSourced(
		querySchemaLawKey(),
		map[string]any{
			"law":      querySchemaLawSummary(),
			"format":   "xml",
			"content":  "<Law />",
			"citation": querySchemaCitation(""),
		},
		"",
	)
}

func querySchemaSourcedLawArticle() map[string]any {
	return querySchemaSourced(
		querySchemaLawKey(),
		map[string]any{
			"law": querySchemaLawSummary(),
			"location": map[string]any{
				"provision":     "main",
				"articleNumber": "1",
			},
			"format":   "text",
			"content":  "第一条",
			"citation": querySchemaCitation("第一条"),
		},
		"第一条",
	)
}

func querySchemaSourcedLawUpdate() map[string]any {
	key := map[string]any{
		"sourceId":     "e-gov-law-api",
		"resourceType": "law-update-list",
		"resourceId":   "2026-07-28",
	}
	return querySchemaSourced(
		key,
		map[string]any{
			"updatedOn": "2026-07-28",
			"lawId":     "law-1",
			"title":     "民法",
			"source":    querySchemaLegalSource(),
		},
		"",
	)
}

func querySchemaSourcedJudicialSummary() map[string]any {
	key := querySchemaJudicialKey()
	return querySchemaSourced(key, querySchemaJudicialSummary(), "")
}

func querySchemaSourcedJudicialDetails() map[string]any {
	key := querySchemaJudicialKey()
	return querySchemaSourced(
		key,
		map[string]any{"summary": querySchemaJudicialSummary()},
		"",
	)
}

func querySchemaSourced(
	key map[string]any,
	data map[string]any,
	location string,
) map[string]any {
	provenance := map[string]any{
		"source":         querySchemaInformationSource(key["sourceId"].(string)),
		"resourceKey":    key,
		"url":            "https://example.go.jp/resource",
		"retrievedAt":    "2026-07-28T10:00:00+09:00",
		"mediaType":      "application/json",
		"transformation": "unchanged",
	}
	if location != "" {
		provenance["location"] = location
	}
	return map[string]any{
		"ref": map[string]any{
			"providerId": "test-provider",
			"key":        key,
		},
		"provenance": []any{provenance},
		"data":       data,
	}
}

func querySchemaLawKey() map[string]any {
	return map[string]any{
		"sourceId":     "e-gov-law-api",
		"resourceType": "law",
		"resourceId":   "law-1",
		"versionId":    "revision-1",
	}
}

func querySchemaJudicialKey() map[string]any {
	return map[string]any{
		"sourceId":     "courts-hanrei",
		"resourceType": "judicial-decision",
		"resourceId":   "detail-1",
	}
}

func querySchemaLawSummary() map[string]any {
	return map[string]any{
		"lawId":      "law-1",
		"revisionId": "revision-1",
		"title":      "民法",
		"source":     querySchemaLegalSource(),
	}
}

func querySchemaLegalSource() map[string]any {
	return map[string]any{
		"id":         "e-gov-law-api",
		"name":       "e-Gov 法令 API",
		"authority":  "official",
		"serviceUrl": "https://laws.e-gov.go.jp/",
	}
}

func querySchemaInformationSource(sourceID string) map[string]any {
	serviceURL := "https://laws.e-gov.go.jp/"
	name := "e-Gov 法令 API"
	publisher := "デジタル庁"
	if sourceID == "courts-hanrei" {
		serviceURL = "https://www.courts.go.jp/"
		name = "裁判例検索"
		publisher = "最高裁判所"
	}
	return map[string]any{
		"id":         sourceID,
		"name":       name,
		"publisher":  publisher,
		"authority":  "official",
		"serviceUrl": serviceURL,
	}
}

func querySchemaCitation(location string) map[string]any {
	citation := map[string]any{
		"source":     querySchemaLegalSource(),
		"lawId":      "law-1",
		"revisionId": "revision-1",
		"url":        "https://laws.e-gov.go.jp/law/law-1",
	}
	if location != "" {
		citation["location"] = location
	}
	return citation
}

func querySchemaJudicialSummary() map[string]any {
	return map[string]any{
		"decisionId":          "detail-1",
		"publicationCategory": "supreme_court",
		"sourceCategoryLabel": "最高裁判例",
		"caseNumber":          "令和8年(受)第1号",
		"decisionDate":        "2026-07-28",
		"courtName":           "最高裁判所",
		"detailUrl":           "https://www.courts.go.jp/app/hanrei_jp/detail2?id=1",
		"documents":           []any{},
		"source":              querySchemaInformationSource("courts-hanrei"),
	}
}

func querySchemaSetPartialAttempts(instance map[string]any) {
	emptyStep := querySchemaStep(
		"step-empty",
		"search",
		"law",
		"law.search",
	)
	failedStep := querySchemaStep(
		"step-failed",
		"search",
		"law",
		"law.search",
	)
	instance["status"] = "partial"
	instance["interpretations"] = []any{querySchemaInterpretation(
		"available",
		[]any{},
		[]any{emptyStep, failedStep},
	)}
	instance["attempts"] = []any{
		querySchemaEmptyLawSearchAttempt("step-empty"),
		querySchemaFailedAttempt("step-failed"),
	}
	instance["notices"] = []any{
		legalquery.LegalQuerySeparateAttemptsNotice,
		legalquery.LegalQueryPartialFailureNotice,
	}
}

func querySchemaAttemptAt(
	instance map[string]any,
	index int,
) map[string]any {
	return instance["attempts"].([]any)[index].(map[string]any)
}

func querySchemaAttemptResult(
	instance map[string]any,
	index int,
) map[string]any {
	return querySchemaAttemptAt(instance, index)["result"].(map[string]any)
}

func querySchemaSourcedItemRefKey(
	instance map[string]any,
	index int,
) map[string]any {
	item := querySchemaAttemptResult(instance, index)["items"].([]any)[0].(map[string]any)
	return item["ref"].(map[string]any)["key"].(map[string]any)
}

func querySchemaInterpretationAt(
	instance map[string]any,
	index int,
) map[string]any {
	return instance["interpretations"].([]any)[index].(map[string]any)
}

func querySchemaCloneMap(
	t *testing.T,
	source map[string]any,
) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(source)
	if err != nil {
		t.Fatalf("fixture を複製できません: %v", err)
	}
	var cloned map[string]any
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		t.Fatalf("fixture を復元できません: %v", err)
	}
	return cloned
}
