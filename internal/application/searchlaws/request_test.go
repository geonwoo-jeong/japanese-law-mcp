package searchlaws

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequestNormalizesPublicSearchInput(t *testing.T) {
	t.Parallel()

	asOf := mustSearchLawsDate(t, "2026-07-26")
	limit := 30
	offset := 40
	request, err := NewRequest(RequestValues{
		Query:  "  行政/手続  ",
		AsOf:   &asOf,
		Limit:  &limit,
		Offset: &offset,
	})
	if err != nil {
		t.Fatalf("SOT-IF-030: NewRequest() のエラー = %v", err)
	}
	asOf = mustSearchLawsDate(t, "2020-01-01")
	limit = 1
	offset = 0

	if request.Query() != "行政/手続" ||
		request.Limit() != 30 ||
		request.Offset() != 40 {
		t.Fatalf("SOT-IF-030: request = %#v", request)
	}
	gotAsOf, exists := request.AsOf()
	if !exists || gotAsOf.String() != "2026-07-26" {
		t.Fatalf("SOT-IF-030: AsOf() = %q, %t", gotAsOf.String(), exists)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("SOT-IF-030: Validate() のエラー = %v", err)
	}
}

func TestRequestAppliesDefaultsAndPreservesMissingAsOf(t *testing.T) {
	t.Parallel()

	request, err := NewRequest(RequestValues{Query: "民法"})
	if err != nil {
		t.Fatalf("SOT-IF-030: NewRequest() のエラー = %v", err)
	}
	if request.Limit() != lawsearch.DefaultLimit || request.Offset() != 0 {
		t.Fatalf("SOT-IF-030: defaults = limit %d, offset %d", request.Limit(), request.Offset())
	}
	if _, exists := request.AsOf(); exists {
		t.Fatal("SOT-IF-030: 省略した asOf が設定された")
	}
}

func TestRequestRejectsInvalidPublicValues(t *testing.T) {
	t.Parallel()

	oldDate := mustSearchLawsDate(t, "2017-03-31")
	limitZero := 0
	offsetNegative := -1
	offsetTooLarge := math.MaxInt32 + 1
	tests := []RequestValues{
		{},
		{Query: "   "},
		{Query: "民\n法"},
		{Query: "/民法/"},
		{Query: "民法", AsOf: &oldDate},
		{Query: "民法", Limit: &limitZero},
		{Query: "民法", Offset: &offsetNegative},
		{Query: "民法", Offset: &offsetTooLarge},
	}
	for _, values := range tests {
		if _, err := NewRequest(values); err == nil {
			t.Fatalf("SOT-IF-030: 不正な入力を受理した: %#v", values)
		}
	}
	if err := (Request{}).Validate(); err == nil {
		t.Fatal("SOT-IF-030: zero value request を受理した")
	}
	var request Request
	if err := json.Unmarshal([]byte(`{"query":"民法"}`), &request); err == nil {
		t.Fatal("SOT-IF-030: Request を JSON から直接復元できた")
	}
}

func mustSearchLawsDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("Date の作成エラー = %v", err)
	}
	return date
}
