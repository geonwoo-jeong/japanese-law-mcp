package model

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestLawRevisionPreservesNormalizedFieldsAndOptionalFalse(t *testing.T) {
	t.Parallel()

	promulgationDate := mustLawRevisionDate(t, "2021-05-19")
	amendmentPromulgationDate := mustLawRevisionDate(t, "2024-06-07")
	effectiveDate := mustLawRevisionDate(t, "2024-12-01")
	scheduledEffectiveDate := mustLawRevisionDate(t, "2025-04-01")
	repealRecordedDate := mustLawRevisionDate(t, "2026-01-01")
	remainInForce := false
	revision, err := NewLawRevision(LawRevisionValues{
		LawID:                     "503AC0000000036",
		RevisionID:                "503AC0000000036_20241201_506AC0000000046",
		Title:                     "デジタル庁設置法",
		LawType:                   "Act",
		LawNumber:                 "令和三年法律第三十六号",
		TitleKana:                 "でじたるちょうせっちほう",
		Abbreviation:              "デジ庁法",
		Category:                  "行政組織",
		PromulgationDate:          &promulgationDate,
		SourceUpdatedAt:           "2024-06-07T09:30:00+09:00",
		AmendmentPromulgationDate: &amendmentPromulgationDate,
		EffectiveDate:             &effectiveDate,
		EffectiveDateNote:         "一部の規定を除く",
		ScheduledEffectiveDate:    &scheduledEffectiveDate,
		AmendmentLawID:            "506AC0000000046",
		AmendmentLawTitle:         "デジタル社会形成基本法等の一部を改正する法律",
		AmendmentLawTitleKana:     "でじたるしゃかいけいせいきほんほうとうのいちぶをかいせいするほうりつ",
		AmendmentLawNumber:        "令和六年法律第四十六号",
		RevisionKind:              LawRevisionKindPartialAmendment,
		RepealStatus:              LawRevisionRepealStatusExpired,
		RepealRecordedDate:        &repealRecordedDate,
		RemainInForce:             &remainInForce,
		CurrentStatus:             LawRevisionCurrentStatusPrevious,
		Source:                    mustLawRevisionSource(t),
	})
	if err != nil {
		t.Fatalf("LawRevision を作成できません: %v", err)
	}
	if value, exists := revision.RemainInForce(); !exists || value {
		t.Fatalf("false の remainInForce = %t, %t", value, exists)
	}

	payload, err := json.Marshal(revision)
	if err != nil {
		t.Fatalf("LawRevision を JSON 化できません: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("LawRevision JSON を解析できません: %v", err)
	}
	want := map[string]any{
		"lawId":                     "503AC0000000036",
		"revisionId":                "503AC0000000036_20241201_506AC0000000046",
		"title":                     "デジタル庁設置法",
		"lawType":                   "Act",
		"lawNumber":                 "令和三年法律第三十六号",
		"titleKana":                 "でじたるちょうせっちほう",
		"abbreviation":              "デジ庁法",
		"category":                  "行政組織",
		"promulgationDate":          "2021-05-19",
		"sourceUpdatedAt":           "2024-06-07T09:30:00+09:00",
		"amendmentPromulgationDate": "2024-06-07",
		"effectiveDate":             "2024-12-01",
		"effectiveDateNote":         "一部の規定を除く",
		"scheduledEffectiveDate":    "2025-04-01",
		"amendmentLawId":            "506AC0000000046",
		"amendmentLawTitle":         "デジタル社会形成基本法等の一部を改正する法律",
		"amendmentLawTitleKana":     "でじたるしゃかいけいせいきほんほうとうのいちぶをかいせいするほうりつ",
		"amendmentLawNumber":        "令和六年法律第四十六号",
		"revisionKind":              "partial_amendment",
		"repealStatus":              "expired",
		"repealRecordedDate":        "2026-01-01",
		"remainInForce":             false,
		"currentStatus":             "previous",
		"source": map[string]any{
			"id":         "e-gov-law-api-v2",
			"name":       "e-Gov 法令 API Version 2",
			"authority":  "official",
			"serviceUrl": "https://laws.e-gov.go.jp/api/2/redoc/",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LawRevision JSON = %#v, want %#v", got, want)
	}
}

func TestLawRevisionRejectsInvalidRequiredAndNormalizedValues(t *testing.T) {
	t.Parallel()

	base := LawRevisionValues{
		LawID:      "503AC0000000036",
		RevisionID: "503AC0000000036_20241201_506AC0000000046",
		Title:      "デジタル庁設置法",
		Source:     mustLawRevisionSource(t),
	}
	tests := map[string]func(LawRevisionValues) LawRevisionValues{
		"lawId 欠落":           func(value LawRevisionValues) LawRevisionValues { value.LawID = ""; return value },
		"revisionId 欠落":      func(value LawRevisionValues) LawRevisionValues { value.RevisionID = ""; return value },
		"title 欠落":           func(value LawRevisionValues) LawRevisionValues { value.Title = ""; return value },
		"revisionKind 不正":    func(value LawRevisionValues) LawRevisionValues { value.RevisionKind = "raw"; return value },
		"repealStatus 不正":    func(value LawRevisionValues) LawRevisionValues { value.RepealStatus = "Raw"; return value },
		"currentStatus 不正":   func(value LawRevisionValues) LawRevisionValues { value.CurrentStatus = "Raw"; return value },
		"sourceUpdatedAt 不正": func(value LawRevisionValues) LawRevisionValues { value.SourceUpdatedAt = "2024/06/07"; return value },
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewLawRevision(mutate(base)); err == nil {
				t.Fatal("不正な LawRevision を拒否しませんでした")
			}
		})
	}
}

func TestLawRevisionOmitsAbsentOptionalFields(t *testing.T) {
	t.Parallel()

	revision, err := NewLawRevision(LawRevisionValues{
		LawID:      "503AC0000000036",
		RevisionID: "503AC0000000036_20241201_506AC0000000046",
		Title:      "デジタル庁設置法",
		Source:     mustLawRevisionSource(t),
	})
	if err != nil {
		t.Fatalf("LawRevision を作成できません: %v", err)
	}

	payload, err := json.Marshal(revision)
	if err != nil {
		t.Fatalf("LawRevision を JSON 化できません: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("LawRevision JSON を解析できません: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("省略可能項目が公開されました: %#v", got)
	}
	for _, field := range []string{
		"lawType",
		"lawNumber",
		"titleKana",
		"abbreviation",
		"category",
		"promulgationDate",
		"sourceUpdatedAt",
		"amendmentPromulgationDate",
		"effectiveDate",
		"effectiveDateNote",
		"scheduledEffectiveDate",
		"amendmentLawId",
		"amendmentLawTitle",
		"amendmentLawTitleKana",
		"amendmentLawNumber",
		"revisionKind",
		"repealStatus",
		"repealRecordedDate",
		"remainInForce",
		"currentStatus",
	} {
		if _, exists := got[field]; exists {
			t.Fatalf("省略可能項目 %q が公開されました: %#v", field, got)
		}
	}
}

func mustLawRevisionDate(t *testing.T, value string) Date {
	t.Helper()
	date, err := NewDate(value)
	if err != nil {
		t.Fatalf("日付 %q を作成できません: %v", value, err)
	}
	return date
}

func mustLawRevisionSource(t *testing.T) LegalSource {
	t.Helper()
	informationSource, err := NewInformationSource(InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/redoc/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	source, err := NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("LegalSource を投影できません: %v", err)
	}
	return source
}
