package model_test

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLawSearchResultPreservesPageAndJSON(t *testing.T) {
	t.Parallel()

	items := []model.LawSummary{newSearchResultLaw(t)}
	nextOffset := 20
	result, err := model.NewLawSearchResult(model.LawSearchResultValues{
		TotalCount: 21,
		Items:      items,
		NextOffset: &nextOffset,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-006: NewLawSearchResult() のエラー = %v", err)
	}
	items[0] = model.LawSummary{}
	nextOffset = 99

	if result.TotalCount() != 21 || len(result.Items()) != 1 {
		t.Fatalf("SOT-MODEL-006: result = %#v", result)
	}
	if value, exists := result.NextOffset(); !exists || value != 20 {
		t.Fatalf("SOT-MODEL-006: NextOffset() = %d, %t", value, exists)
	}
	cloned := result.Items()
	cloned[0] = model.LawSummary{}
	if err := result.Items()[0].Validate(); err != nil {
		t.Fatalf("SOT-MODEL-006: getter 経由で items が変更された: %v", err)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("SOT-MODEL-006/009: JSON 変換のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-006: JSON を再解析できない: %v", err)
	}
	if object["totalCount"] != float64(21) ||
		object["nextOffset"] != float64(20) {
		t.Fatalf("SOT-MODEL-006: JSON = %s", encoded)
	}
	if values, ok := object["items"].([]any); !ok || len(values) != 1 {
		t.Fatalf("SOT-MODEL-006: items = %#v", object["items"])
	}
}

func TestLawSearchResultRepresentsEmptyResult(t *testing.T) {
	t.Parallel()

	result, err := model.NewLawSearchResult(model.LawSearchResultValues{})
	if err != nil {
		t.Fatalf("SOT-MODEL-006: 空結果のエラー = %v", err)
	}
	if result.Items() == nil || len(result.Items()) != 0 {
		t.Fatalf("SOT-MODEL-006: 空 items = %#v", result.Items())
	}
	if _, exists := result.NextOffset(); exists {
		t.Fatal("SOT-MODEL-006: 空結果に nextOffset がある")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("SOT-MODEL-006: JSON 変換のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-006: JSON を再解析できない: %v", err)
	}
	if _, exists := object["nextOffset"]; exists {
		t.Fatalf("SOT-MODEL-006: nextOffset が省略されていない: %s", encoded)
	}
	if !reflect.DeepEqual(object["items"], []any{}) {
		t.Fatalf("SOT-MODEL-006: items = %#v", object["items"])
	}
}

func TestLawSearchResultRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	validLaw := newSearchResultLaw(t)
	negative := -1
	tooLarge := math.MaxInt32 + 1
	tests := []model.LawSearchResultValues{
		{TotalCount: -1},
		{TotalCount: 0, Items: []model.LawSummary{validLaw}},
		{TotalCount: 1, Items: []model.LawSummary{{}}},
		{TotalCount: 1, NextOffset: &negative},
		{TotalCount: 1, NextOffset: &tooLarge},
		{TotalCount: 1, NextOffset: func() *int { value := 2; return &value }()},
	}
	for _, values := range tests {
		if _, err := model.NewLawSearchResult(values); err == nil {
			t.Fatalf("SOT-MODEL-006: 不正な値を受理した: %#v", values)
		}
	}
	if _, err := json.Marshal(model.LawSearchResult{}); err == nil {
		t.Fatal("SOT-MODEL-006: zero value を JSON に変換できた")
	}
	var result model.LawSearchResult
	if err := json.Unmarshal([]byte(`{"totalCount":0,"items":[]}`), &result); err == nil {
		t.Fatal("SOT-MODEL-006: JSON から直接復元できた")
	}
}

func newSearchResultLaw(t *testing.T) model.LawSummary {
	t.Helper()
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-001",
		RevisionID: "law-001_20260401_revision",
		Title:      "行政手続法",
		Source:     newLegalSource(t),
	})
	if err != nil {
		t.Fatalf("LawSummary の作成エラー = %v", err)
	}
	return law
}
