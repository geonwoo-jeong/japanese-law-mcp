package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLawUpdateCopiesValuesAndMarshalsJSON(t *testing.T) {
	t.Parallel()

	updatedOn := newDate(t, "2026-07-26")
	promulgationDate := newDate(t, "2024-01-01")
	amendmentPromulgationDate := newDate(t, "2025-12-01")
	effectiveDate := newDate(t, "2026-04-01")
	enforcementPending := false
	authorityReviewPending := true
	source := newLegalSource(t)

	got, err := model.NewLawUpdate(model.LawUpdateValues{
		UpdatedOn:                 updatedOn,
		LawID:                     "law-001",
		Title:                     "行政手続法",
		LawType:                   "法律",
		LawNumber:                 "平成五年法律第八十八号",
		TitleKana:                 "ぎょうせいてつづきほう",
		PreviousTitle:             "旧行政手続法",
		PromulgationDate:          &promulgationDate,
		AmendmentTitle:            "行政手続法の一部を改正する法律",
		AmendmentLawNumber:        "令和七年法律第一号",
		AmendmentPromulgationDate: &amendmentPromulgationDate,
		EffectiveDate:             &effectiveDate,
		EffectiveDateNote:         "一部の規定を除く",
		DocumentURL:               "https://laws.e-gov.go.jp/law/law-001",
		EnforcementPending:        &enforcementPending,
		AuthorityReviewPending:    &authorityReviewPending,
		Source:                    source,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-019: NewLawUpdate() のエラー = %v", err)
	}

	promulgationDate = newDate(t, "2030-01-01")
	amendmentPromulgationDate = newDate(t, "2030-02-01")
	effectiveDate = newDate(t, "2030-03-01")
	enforcementPending = true
	authorityReviewPending = false

	if got.UpdatedOn() != updatedOn ||
		got.LawID() != "law-001" ||
		got.Title() != "行政手続法" ||
		got.Source() != source {
		t.Fatalf("SOT-MODEL-019: LawUpdate = %#v", got)
	}
	assertOptionalString(t, "LawType", got.LawType, "法律")
	assertOptionalString(t, "LawNumber", got.LawNumber, "平成五年法律第八十八号")
	assertOptionalString(t, "TitleKana", got.TitleKana, "ぎょうせいてつづきほう")
	assertOptionalString(t, "PreviousTitle", got.PreviousTitle, "旧行政手続法")
	assertOptionalString(
		t,
		"AmendmentTitle",
		got.AmendmentTitle,
		"行政手続法の一部を改正する法律",
	)
	assertOptionalString(
		t,
		"AmendmentLawNumber",
		got.AmendmentLawNumber,
		"令和七年法律第一号",
	)
	assertOptionalString(t, "EffectiveDateNote", got.EffectiveDateNote, "一部の規定を除く")
	assertOptionalString(
		t,
		"DocumentURL",
		got.DocumentURL,
		"https://laws.e-gov.go.jp/law/law-001",
	)
	assertOptionalDate(t, "PromulgationDate", got.PromulgationDate, "2024-01-01")
	assertOptionalDate(
		t,
		"AmendmentPromulgationDate",
		got.AmendmentPromulgationDate,
		"2025-12-01",
	)
	assertOptionalDate(t, "EffectiveDate", got.EffectiveDate, "2026-04-01")
	assertOptionalBool(t, "EnforcementPending", got.EnforcementPending, false)
	assertOptionalBool(t, "AuthorityReviewPending", got.AuthorityReviewPending, true)
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-019: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/019: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/019: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"updatedOn":                 "2026-07-26",
		"lawId":                     "law-001",
		"title":                     "行政手続法",
		"lawType":                   "法律",
		"lawNumber":                 "平成五年法律第八十八号",
		"titleKana":                 "ぎょうせいてつづきほう",
		"previousTitle":             "旧行政手続法",
		"promulgationDate":          "2024-01-01",
		"amendmentTitle":            "行政手続法の一部を改正する法律",
		"amendmentLawNumber":        "令和七年法律第一号",
		"amendmentPromulgationDate": "2025-12-01",
		"effectiveDate":             "2026-04-01",
		"effectiveDateNote":         "一部の規定を除く",
		"documentUrl":               "https://laws.e-gov.go.jp/law/law-001",
		"enforcementPending":        false,
		"authorityReviewPending":    true,
		"source": map[string]any{
			"id":         "e-gov-law-api-v2",
			"name":       "e-Gov 法令 API",
			"authority":  "official",
			"serviceUrl": "https://laws.e-gov.go.jp/api/2/",
		},
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/019: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestLawUpdateOmitsAbsentOptionalValues(t *testing.T) {
	t.Parallel()

	got, err := model.NewLawUpdate(model.LawUpdateValues{
		UpdatedOn: newDate(t, "2020-11-24"),
		LawID:     "law-001",
		Title:     "行政手続法",
		Source:    newLegalSource(t),
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-019: NewLawUpdate() のエラー = %v", err)
	}

	for name, getter := range map[string]func() (string, bool){
		"lawType":            got.LawType,
		"lawNumber":          got.LawNumber,
		"titleKana":          got.TitleKana,
		"previousTitle":      got.PreviousTitle,
		"amendmentTitle":     got.AmendmentTitle,
		"amendmentLawNumber": got.AmendmentLawNumber,
		"effectiveDateNote":  got.EffectiveDateNote,
		"documentUrl":        got.DocumentURL,
	} {
		value, exists := getter()
		if exists || value != "" {
			t.Fatalf("SOT-MODEL-019: %s = %q, %t", name, value, exists)
		}
	}
	for name, getter := range map[string]func() (model.Date, bool){
		"promulgationDate":          got.PromulgationDate,
		"amendmentPromulgationDate": got.AmendmentPromulgationDate,
		"effectiveDate":             got.EffectiveDate,
	} {
		value, exists := getter()
		if exists || !value.IsZero() {
			t.Fatalf("SOT-MODEL-019: %s = %q, %t", name, value.String(), exists)
		}
	}
	for name, getter := range map[string]func() (bool, bool){
		"enforcementPending":     got.EnforcementPending,
		"authorityReviewPending": got.AuthorityReviewPending,
	} {
		value, exists := getter()
		if exists || value {
			t.Fatalf("SOT-MODEL-019: %s = %t, %t", name, value, exists)
		}
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/019: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/019: JSON を再解析できない: %v", err)
	}
	for _, key := range []string{
		"lawType",
		"lawNumber",
		"titleKana",
		"previousTitle",
		"promulgationDate",
		"amendmentTitle",
		"amendmentLawNumber",
		"amendmentPromulgationDate",
		"effectiveDate",
		"effectiveDateNote",
		"documentUrl",
		"enforcementPending",
		"authorityReviewPending",
	} {
		if _, exists := object[key]; exists {
			t.Fatalf("SOT-MODEL-009/019: 欠落した %s が省略されていない: %s", key, encoded)
		}
	}
}

func TestLawUpdateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	zeroDate := model.Date{}
	valid := model.LawUpdateValues{
		UpdatedOn: newDate(t, "2026-07-26"),
		LawID:     "law-001",
		Title:     "行政手続法",
		Source:    newLegalSource(t),
	}
	tests := map[string]model.LawUpdateValues{
		"updatedOn のゼロ値": withLawUpdateChange(valid, func(values *model.LawUpdateValues) {
			values.UpdatedOn = zeroDate
		}),
		"lawId の欠落": withLawUpdateChange(valid, func(values *model.LawUpdateValues) {
			values.LawID = ""
		}),
		"title の欠落": withLawUpdateChange(valid, func(values *model.LawUpdateValues) {
			values.Title = ""
		}),
		"source のゼロ値": withLawUpdateChange(valid, func(values *model.LawUpdateValues) {
			values.Source = model.LegalSource{}
		}),
		"promulgationDate のゼロ値": withLawUpdateChange(valid, func(values *model.LawUpdateValues) {
			values.PromulgationDate = &zeroDate
		}),
		"amendmentPromulgationDate のゼロ値": withLawUpdateChange(valid, func(values *model.LawUpdateValues) {
			values.AmendmentPromulgationDate = &zeroDate
		}),
		"effectiveDate のゼロ値": withLawUpdateChange(valid, func(values *model.LawUpdateValues) {
			values.EffectiveDate = &zeroDate
		}),
		"HTTP documentUrl": withLawUpdateChange(valid, func(values *model.LawUpdateValues) {
			values.DocumentURL = "http://laws.e-gov.go.jp/law/law-001"
		}),
		"認証情報を含む documentUrl": withLawUpdateChange(valid, func(values *model.LawUpdateValues) {
			values.DocumentURL = "https://user:password@laws.e-gov.go.jp/law/law-001"
		}),
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewLawUpdate(values); err == nil {
				t.Fatalf("SOT-MODEL-019: NewLawUpdate(%#v) が成功した", values)
			}
		})
	}
}

func TestLawUpdateRejectsDirectJSONDecodeAndInvalidZeroValue(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.LawUpdate{}); err == nil {
		t.Fatal("SOT-MODEL-009/019: LawUpdate のゼロ値を JSON に変換できた")
	}

	var update model.LawUpdate
	if err := json.Unmarshal(
		[]byte(`{"updatedOn":"2026-07-26","lawId":"law-001","title":"行政手続法"}`),
		&update,
	); err == nil {
		t.Fatal("SOT-MODEL-019: LawUpdate を JSON から直接復元できた")
	}
}

func assertOptionalString(
	t *testing.T,
	name string,
	getter func() (string, bool),
	want string,
) {
	t.Helper()

	value, exists := getter()
	if !exists || value != want {
		t.Fatalf("SOT-MODEL-019: %s() = %q, %t", name, value, exists)
	}
}

func assertOptionalDate(
	t *testing.T,
	name string,
	getter func() (model.Date, bool),
	want string,
) {
	t.Helper()

	value, exists := getter()
	if !exists || value.String() != want {
		t.Fatalf("SOT-MODEL-019: %s() = %q, %t", name, value.String(), exists)
	}
}

func assertOptionalBool(
	t *testing.T,
	name string,
	getter func() (bool, bool),
	want bool,
) {
	t.Helper()

	value, exists := getter()
	if !exists || value != want {
		t.Fatalf("SOT-MODEL-019: %s() = %t, %t", name, value, exists)
	}
}

func withLawUpdateChange(
	values model.LawUpdateValues,
	change func(*model.LawUpdateValues),
) model.LawUpdateValues {
	change(&values)
	return values
}
