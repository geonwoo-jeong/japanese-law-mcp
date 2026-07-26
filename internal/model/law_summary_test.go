package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLawSummary(t *testing.T) {
	t.Parallel()

	promulgationDate := newDate(t, "2024-01-01")
	effectiveDate := newDate(t, "2024-04-01")
	source := newLegalSource(t)

	got, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:                 "law-001",
		RevisionID:            "revision-002",
		Title:                 "行政手続法",
		LawNumber:             "平成五年法律第八十八号",
		PromulgationDate:      &promulgationDate,
		RevisionEffectiveDate: &effectiveDate,
		Source:                source,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-001: NewLawSummary() のエラー = %v", err)
	}

	promulgationDate = newDate(t, "2030-01-01")
	effectiveDate = newDate(t, "2030-04-01")

	if got.LawID() != "law-001" ||
		got.RevisionID() != "revision-002" ||
		got.Title() != "行政手続法" ||
		got.Source() != source {
		t.Fatalf("SOT-MODEL-001: LawSummary = %#v", got)
	}
	if lawNumber, ok := got.LawNumber(); !ok || lawNumber != "平成五年法律第八十八号" {
		t.Fatalf("SOT-MODEL-001: LawNumber() = %q, %t", lawNumber, ok)
	}
	if date, ok := got.PromulgationDate(); !ok || date.String() != "2024-01-01" {
		t.Fatalf("SOT-MODEL-001: PromulgationDate() = %q, %t", date.String(), ok)
	}
	if date, ok := got.RevisionEffectiveDate(); !ok || date.String() != "2024-04-01" {
		t.Fatalf("SOT-MODEL-001: RevisionEffectiveDate() = %q, %t", date.String(), ok)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-001: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-001/009: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-001/009: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"lawId":                 "law-001",
		"revisionId":            "revision-002",
		"title":                 "行政手続法",
		"lawNumber":             "平成五年法律第八十八号",
		"promulgationDate":      "2024-01-01",
		"revisionEffectiveDate": "2024-04-01",
		"source": map[string]any{
			"id":         "e-gov-law-api-v2",
			"name":       "e-Gov 法令 API",
			"authority":  "official",
			"serviceUrl": "https://laws.e-gov.go.jp/api/2/",
		},
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-001/009: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestLawSummaryOmitsAbsentOptionalValues(t *testing.T) {
	t.Parallel()

	got, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-001",
		RevisionID: "revision-001",
		Title:      "行政手続法",
		Source:     newLegalSource(t),
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-001: NewLawSummary() のエラー = %v", err)
	}

	if value, ok := got.LawNumber(); ok || value != "" {
		t.Fatalf("SOT-MODEL-001: LawNumber() = %q, %t", value, ok)
	}
	if value, ok := got.PromulgationDate(); ok || !value.IsZero() {
		t.Fatalf("SOT-MODEL-001: PromulgationDate() = %q, %t", value.String(), ok)
	}
	if value, ok := got.RevisionEffectiveDate(); ok || !value.IsZero() {
		t.Fatalf("SOT-MODEL-001: RevisionEffectiveDate() = %q, %t", value.String(), ok)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-001/009: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-001/009: JSON を再解析できない: %v", err)
	}
	for _, key := range []string{"lawNumber", "promulgationDate", "revisionEffectiveDate"} {
		if _, exists := object[key]; exists {
			t.Fatalf("SOT-MODEL-001/009: 欠落した %s が省略されていない: %s", key, encoded)
		}
	}
}

func TestLawSummaryRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	zeroDate := model.Date{}
	valid := model.LawSummaryValues{
		LawID:      "law-001",
		RevisionID: "revision-001",
		Title:      "行政手続法",
		Source:     newLegalSource(t),
	}
	tests := map[string]model.LawSummaryValues{
		"lawId の欠落": withLawSummaryChange(valid, func(values *model.LawSummaryValues) {
			values.LawID = ""
		}),
		"revisionId の欠落": withLawSummaryChange(valid, func(values *model.LawSummaryValues) {
			values.RevisionID = ""
		}),
		"title の欠落": withLawSummaryChange(valid, func(values *model.LawSummaryValues) {
			values.Title = ""
		}),
		"source のゼロ値": withLawSummaryChange(valid, func(values *model.LawSummaryValues) {
			values.Source = model.LegalSource{}
		}),
		"promulgationDate のゼロ値": withLawSummaryChange(valid, func(values *model.LawSummaryValues) {
			values.PromulgationDate = &zeroDate
		}),
		"revisionEffectiveDate のゼロ値": withLawSummaryChange(valid, func(values *model.LawSummaryValues) {
			values.RevisionEffectiveDate = &zeroDate
		}),
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewLawSummary(values); err == nil {
				t.Fatalf("SOT-MODEL-001: NewLawSummary(%#v) が成功した", values)
			}
		})
	}
}

func TestLawSummaryAllowsWhitespaceInRequiredStrings(t *testing.T) {
	t.Parallel()

	if _, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      " ",
		RevisionID: "\t",
		Title:      "\n",
		Source:     newLegalSource(t),
	}); err != nil {
		t.Fatalf("SOT-MODEL-001: 空ではない必須文字列を拒否した: %v", err)
	}
}

func TestZeroLawSummaryCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.LawSummary{}); err == nil {
		t.Fatal("SOT-MODEL-001/009: LawSummary のゼロ値を JSON に変換できた")
	}
}

func TestLawSummaryRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.LawSummary
	if err := json.Unmarshal([]byte(
		`{"lawId":"law-001","revisionId":"revision-001","title":"行政手続法","source":{"id":"e-gov-law-api-v2","name":"e-Gov 法令 API","authority":"official","serviceUrl":"https://laws.e-gov.go.jp/api/2/"}}`,
	), &got); err == nil {
		t.Fatal("SOT-MODEL-001: LawSummary を JSON から直接復元できた")
	}
}

func newLegalSource(t *testing.T) model.LegalSource {
	t.Helper()

	informationSource := newInformationSource(t)
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("SOT-MODEL-003: NewLegalSource() のエラー = %v", err)
	}
	return source
}

func withLawSummaryChange(
	values model.LawSummaryValues,
	change func(*model.LawSummaryValues),
) model.LawSummaryValues {
	change(&values)
	return values
}
