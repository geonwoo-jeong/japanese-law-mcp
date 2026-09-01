package listlawupdates

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestResultCopiesItemsAndKeepsCountMetadata(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	item := mustListLawUpdate(t, date, "law-001")
	input := []model.LawUpdate{item}
	result, err := NewResult(ResultValues{
		Date:       date,
		TotalCount: 1,
		Items:      input,
	})
	if err != nil {
		t.Fatalf("SOT-IF-076: NewResult() のエラー = %v", err)
	}

	input[0] = model.LawUpdate{}
	items := result.Items()
	if len(items) != 1 || items[0].LawID() != "law-001" {
		t.Fatalf("SOT-IF-076: Items() = %#v", items)
	}
	items[0] = model.LawUpdate{}
	if result.Items()[0].LawID() != "law-001" {
		t.Fatal("SOT-IF-076: Result.items が外部から変更された")
	}
	if result.Date() != date ||
		result.TotalCount() != 1 ||
		result.ReturnedCount() != 1 ||
		result.OmittedCount() != 0 ||
		result.Truncated() {
		t.Fatalf("SOT-IF-076: Result = %#v", result)
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("SOT-IF-076: Validate() のエラー = %v", err)
	}
}

func TestResultMakesOmissionExplicit(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	result, err := NewResult(ResultValues{
		Date:       date,
		TotalCount: 208,
		Items: []model.LawUpdate{
			mustListLawUpdate(t, date, "law-001"),
			mustListLawUpdate(t, date, "law-002"),
		},
	})
	if err != nil {
		t.Fatalf("SOT-IF-076: NewResult() のエラー = %v", err)
	}
	if result.TotalCount() != 208 ||
		result.ReturnedCount() != 2 ||
		result.OmittedCount() != 206 ||
		!result.Truncated() {
		t.Fatalf("SOT-IF-076: 省略情報 = %#v", result)
	}
}

func TestResultSupportsNonNilEmptyItems(t *testing.T) {
	t.Parallel()

	result, err := NewResult(ResultValues{
		Date: mustListLawUpdatesDate(t, "2026-07-26"),
	})
	if err != nil {
		t.Fatalf("SOT-IF-076: 空結果の NewResult() エラー = %v", err)
	}
	if items := result.Items(); items == nil || len(items) != 0 {
		t.Fatalf("SOT-IF-076: 空の Items() = %#v", items)
	}
}

func TestResultRejectsInconsistentValuesAndDirectJSONDecode(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	tests := map[string]ResultValues{
		"date の欠落": {
			TotalCount: 0,
		},
		"totalCount が items より小さい": {
			Date:       date,
			TotalCount: 0,
			Items:      []model.LawUpdate{mustListLawUpdate(t, date, "law-001")},
		},
		"updatedOn の不一致": {
			Date:       date,
			TotalCount: 1,
			Items: []model.LawUpdate{
				mustListLawUpdate(
					t,
					mustListLawUpdatesDate(t, "2026-07-25"),
					"law-001",
				),
			},
		},
		"無効な item": {
			Date:       date,
			TotalCount: 1,
			Items:      []model.LawUpdate{{}},
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewResult(values); err == nil {
				t.Fatal("SOT-IF-076: 不整合な ResultValues を受理した")
			}
		})
	}

	var result Result
	if err := json.Unmarshal(
		[]byte(`{"date":"2026-07-26","totalCount":0,"items":[]}`),
		&result,
	); err == nil {
		t.Fatal("SOT-IF-076: Result を JSON から直接復元できた")
	}
}

func TestResultRejectsMoreThanMaximumItems(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	item := mustListLawUpdate(t, date, "law-001")
	items := make([]model.LawUpdate, MaxLimit+1)
	for index := range items {
		items[index] = item
	}
	if _, err := NewResult(ResultValues{
		Date:       date,
		TotalCount: len(items),
		Items:      items,
	}); err == nil {
		t.Fatalf("SOT-IF-076: %d 件の公開 items を受理した", len(items))
	}
}

func mustListLawUpdate(
	t *testing.T,
	date model.Date,
	lawID string,
) model.LawUpdate {
	t.Helper()

	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v1",
		Name:       "e-Gov 法令 API Version 1",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/docs/law-data-basic/8529371-law-api-v1/",
	})
	if err != nil {
		t.Fatalf("試験用 InformationSource を作成できません: %v", err)
	}
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("試験用 LegalSource を作成できません: %v", err)
	}
	update, err := model.NewLawUpdate(model.LawUpdateValues{
		UpdatedOn: date,
		LawID:     lawID,
		Title:     "行政手続法",
		Source:    source,
	})
	if err != nil {
		t.Fatalf("試験用 LawUpdate を作成できません: %v", err)
	}
	return update
}
