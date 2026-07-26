package model_test

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLawContentSearchResultPreservesPageAndJSON(t *testing.T) {
	t.Parallel()

	items := []model.LawContentMatch{mustLawContentMatch(t)}
	nextOffset := 1
	result, err := model.NewLawContentSearchResult(model.LawContentSearchResultValues{
		TotalCount: 2,
		Items:      items,
		NextOffset: &nextOffset,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-008: NewLawContentSearchResult() のエラー = %v", err)
	}
	items[0] = model.LawContentMatch{}
	nextOffset = 99

	if result.TotalCount() != 2 || len(result.Items()) != 1 {
		t.Fatalf("SOT-MODEL-008: result = %#v", result)
	}
	if value, exists := result.NextOffset(); !exists || value != 1 {
		t.Fatalf("SOT-MODEL-008: NextOffset() = %d, %t", value, exists)
	}
	cloned := result.Items()
	cloned[0] = model.LawContentMatch{}
	if err := result.Items()[0].Validate(); err != nil {
		t.Fatalf("SOT-MODEL-008: getter 経由で items が変更された: %v", err)
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("SOT-MODEL-008/009: JSON 変換のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-008: JSON を再解析できない: %v", err)
	}
	if object["totalCount"] != float64(2) ||
		object["nextOffset"] != float64(1) {
		t.Fatalf("SOT-MODEL-008: JSON = %s", encoded)
	}
	if values, ok := object["items"].([]any); !ok || len(values) != 1 {
		t.Fatalf("SOT-MODEL-008: items = %#v", object["items"])
	}
}

func TestLawContentSearchResultRepresentsEmptyResult(t *testing.T) {
	t.Parallel()

	result, err := model.NewLawContentSearchResult(model.LawContentSearchResultValues{})
	if err != nil {
		t.Fatalf("SOT-MODEL-008: 空結果のエラー = %v", err)
	}
	if result.TotalCount() != 0 || result.Items() == nil || len(result.Items()) != 0 {
		t.Fatalf("SOT-MODEL-008: 空結果 = %#v", result)
	}
	if _, exists := result.NextOffset(); exists {
		t.Fatal("SOT-MODEL-008: 空結果に nextOffset がある")
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("SOT-MODEL-008: JSON 変換のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-008: JSON を再解析できない: %v", err)
	}
	if _, exists := object["nextOffset"]; exists {
		t.Fatalf("SOT-MODEL-008: nextOffset が省略されていない: %s", encoded)
	}
	if !reflect.DeepEqual(object["items"], []any{}) {
		t.Fatalf("SOT-MODEL-008: items = %#v", object["items"])
	}
}

func TestLawContentSearchResultRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	validMatch := mustLawContentMatch(t)
	negative := -1
	zero := 0
	tooLarge := math.MaxInt32 + 1
	tests := []model.LawContentSearchResultValues{
		{TotalCount: -1},
		{TotalCount: 0, Items: []model.LawContentMatch{validMatch}},
		{TotalCount: 1, Items: []model.LawContentMatch{{}}},
		{TotalCount: 1, NextOffset: &negative},
		{TotalCount: 1, NextOffset: &zero},
		{TotalCount: math.MaxInt32 + 2, NextOffset: &tooLarge},
		{TotalCount: 1, NextOffset: func() *int { value := 2; return &value }()},
	}
	for _, values := range tests {
		if _, err := model.NewLawContentSearchResult(values); err == nil {
			t.Fatalf("SOT-MODEL-008: 不正な値を受理した: %#v", values)
		}
	}
	if _, err := json.Marshal(model.LawContentSearchResult{}); err == nil {
		t.Fatal("SOT-MODEL-008: zero value を JSON に変換できた")
	}
	var result model.LawContentSearchResult
	if err := json.Unmarshal(
		[]byte(`{"totalCount":0,"items":[]}`),
		&result,
	); err == nil {
		t.Fatal("SOT-MODEL-008: JSON から直接復元できた")
	}
}

func mustLawContentMatch(t *testing.T) model.LawContentMatch {
	t.Helper()
	match, err := model.NewLawContentMatch(validLawContentMatchValues(t))
	if err != nil {
		t.Fatalf("LawContentMatch の作成エラー = %v", err)
	}
	return match
}
