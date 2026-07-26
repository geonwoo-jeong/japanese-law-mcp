package searchlawcontent

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequestAcceptsDefinedSearchExpressions(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"AND":                "情報 公開",
		"複数の U+0020":         "情報   公開",
		"OR":                 "情報公開|個人情報",
		"NOT":                "!個人情報",
		"group":              "(情報 公開)|個人",
		"AND と NOT":          "情報 !個人情報",
		"入れ子の group":         "!((情報 公開)|個人)",
		"ワイルドカード":            "であって*として*定める",
		"一文字ワイルドカード":         "第?条",
		"U+0020 以外の Unicode": "情報\u3000公開",
	}
	for name, query := range tests {
		name, query := name, query
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			request, err := NewRequest(RequestValues{Query: "  " + query + "  "})
			if err != nil {
				t.Fatalf("SOT-IF-033: NewRequest(%q) のエラー = %v", query, err)
			}
			if request.Query() != query {
				t.Fatalf("SOT-IF-033: Query() = %q", request.Query())
			}
			if err := request.Validate(); err != nil {
				t.Fatalf("SOT-IF-033: Validate() のエラー = %v", err)
			}
		})
	}
}

func TestRequestAppliesDefaultsAndCopiesValues(t *testing.T) {
	t.Parallel()

	asOf := mustSearchLawContentDate(t, "2026-07-26")
	limit := 100
	offset := math.MaxInt32
	request, err := NewRequest(RequestValues{
		Query:  "情報 公開",
		AsOf:   &asOf,
		Limit:  &limit,
		Offset: &offset,
	})
	if err != nil {
		t.Fatalf("SOT-IF-033: NewRequest() のエラー = %v", err)
	}
	asOf = mustSearchLawContentDate(t, "2020-01-01")
	limit = 1
	offset = 0

	gotAsOf, exists := request.AsOf()
	if !exists || gotAsOf.String() != "2026-07-26" {
		t.Fatalf("SOT-IF-033: AsOf() = %q, %t", gotAsOf.String(), exists)
	}
	if request.Limit() != 100 || request.Offset() != math.MaxInt32 {
		t.Fatalf(
			"SOT-IF-033: limit = %d, offset = %d",
			request.Limit(),
			request.Offset(),
		)
	}

	defaults, err := NewRequest(RequestValues{Query: "情報"})
	if err != nil {
		t.Fatalf("SOT-IF-033: 既定値の NewRequest() のエラー = %v", err)
	}
	if defaults.Limit() != DefaultLimit || defaults.Offset() != 0 {
		t.Fatalf(
			"SOT-IF-033: defaults = limit %d, offset %d",
			defaults.Limit(),
			defaults.Offset(),
		)
	}
	if _, exists := defaults.AsOf(); exists {
		t.Fatal("SOT-IF-033: 省略した asOf が設定された")
	}
}

func TestRequestAcceptsMaximumQueryBytes(t *testing.T) {
	t.Parallel()

	query := strings.Repeat("a", MaxQueryBytes)
	request, err := NewRequest(RequestValues{Query: query})
	if err != nil {
		t.Fatalf("SOT-IF-033: %d byte の query のエラー = %v", len(query), err)
	}
	if len(request.Query()) != MaxQueryBytes {
		t.Fatalf("SOT-IF-033: query の byte 数 = %d", len(request.Query()))
	}
}

func TestRequestRejectsInvalidSearchExpressions(t *testing.T) {
	t.Parallel()

	tests := []string{
		"*",
		"?",
		"***???",
		"情報* 公開",
		"情報*|公開",
		"!情報*",
		"(情報*)",
		"情報||公開",
		"情報|",
		"|情報",
		"情報 |公開",
		"情報| 公開",
		"!",
		"!!情報",
		"()",
		"(情報",
		"情報)",
		"情報(公開)",
		"情報 \t公開",
		"情報\n公開",
		"情報\x00公開",
		"情報\x1f公開",
		"情報\x7f公開",
		string([]byte{0xff}),
	}
	for _, query := range tests {
		query := query
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			if _, err := NewRequest(RequestValues{Query: query}); err == nil {
				t.Fatalf("SOT-IF-033: 不正な検索式 %q を受理した", query)
			}
		})
	}
}

func TestRequestRejectsInvalidBoundaryValues(t *testing.T) {
	t.Parallel()

	oldDate := mustSearchLawContentDate(t, "2017-03-31")
	limitZero := 0
	limitTooLarge := 101
	offsetNegative := -1
	offsetTooLarge := math.MaxInt32 + 1
	tests := []RequestValues{
		{},
		{Query: "   "},
		{Query: strings.Repeat("a", MaxQueryBytes+1)},
		{Query: "情報", AsOf: &oldDate},
		{Query: "情報", Limit: &limitZero},
		{Query: "情報", Limit: &limitTooLarge},
		{Query: "情報", Offset: &offsetNegative},
		{Query: "情報", Offset: &offsetTooLarge},
	}
	for _, values := range tests {
		if _, err := NewRequest(values); err == nil {
			t.Fatalf("SOT-IF-033: 不正な入力を受理した: %#v", values)
		}
	}
	if err := (Request{}).Validate(); err == nil {
		t.Fatal("SOT-IF-033: zero value request を受理した")
	}
	var request Request
	if err := json.Unmarshal([]byte(`{"query":"情報"}`), &request); err == nil {
		t.Fatal("SOT-IF-033: Request を JSON から直接復元できた")
	}
}

func mustSearchLawContentDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("Date の作成エラー = %v", err)
	}
	return date
}
