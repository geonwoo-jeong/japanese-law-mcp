package lawsearch_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestRequestConstants(t *testing.T) {
	t.Parallel()

	if lawsearch.CapabilityID != "law.search" ||
		lawsearch.MajorVersion != 1 ||
		lawsearch.DefaultLimit != 20 ||
		lawsearch.MaxLimit != 100 ||
		lawsearch.MaxTokenBytes != 4096 {
		t.Fatalf("SOT-IF-022: capability 定数が契約と一致しない")
	}
}

func TestNewRequestNormalizesAndCopiesInput(t *testing.T) {
	t.Parallel()

	asOf, err := model.NewDate("2025-01-02")
	if err != nil {
		t.Fatalf("試験用の日付を作成できない: %v", err)
	}
	limit := 50
	opaqueToken := strings.Repeat("t", 16)
	request, err := lawsearch.NewRequest(lawsearch.RequestValues{
		Query:             "  民 法  ",
		AsOf:              &asOf,
		Limit:             &limit,
		ContinuationToken: opaqueToken,
	})
	if err != nil {
		t.Fatalf("SOT-IF-022: NewRequest() のエラー = %v", err)
	}

	otherDate, dateErr := model.NewDate("2026-03-04")
	if dateErr != nil {
		t.Fatalf("試験用の日付を作成できない: %v", dateErr)
	}
	asOf = otherDate
	limit = 1

	if request.Query() != "民 法" {
		t.Fatalf("SOT-IF-022: Query() = %q", request.Query())
	}
	gotAsOf, exists := request.AsOf()
	if !exists || gotAsOf.String() != "2025-01-02" {
		t.Fatalf("SOT-IF-022: AsOf() = %q, %t", gotAsOf.String(), exists)
	}
	if request.Limit() != 50 {
		t.Fatalf("SOT-IF-022: Limit() = %d", request.Limit())
	}
	if token, ok := request.ContinuationToken(); !ok || token != opaqueToken {
		t.Fatalf("SOT-IF-022: ContinuationToken() = %q, %t", token, ok)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("SOT-IF-022: Validate() のエラー = %v", err)
	}
}

func TestNewRequestAppliesDefaultLimit(t *testing.T) {
	t.Parallel()

	request, err := lawsearch.NewRequest(lawsearch.RequestValues{Query: "民法"})
	if err != nil {
		t.Fatalf("SOT-IF-022: NewRequest() のエラー = %v", err)
	}
	if request.Limit() != lawsearch.DefaultLimit {
		t.Fatalf("SOT-IF-022: 既定 limit = %d", request.Limit())
	}
	if _, exists := request.AsOf(); exists {
		t.Fatal("SOT-IF-022: 省略した asOf が存在すると判定された")
	}
	if _, exists := request.ContinuationToken(); exists {
		t.Fatal("SOT-IF-022: 省略した continuationToken が存在すると判定された")
	}
}

func TestNewRequestRejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"空文字":        "",
		"U+0020 だけ":  "   ",
		"タブ":         "民\t法",
		"改行":         "民\n法",
		"DEL":        "民\x7f法",
		"不正な UTF-8":  string([]byte{'a', 0xff}),
		"512 byte 超": strings.Repeat("法", 171),
	}
	for name, query := range tests {
		query := query
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := lawsearch.NewRequest(lawsearch.RequestValues{Query: query}); err == nil {
				t.Fatalf("SOT-IF-022: 不正な query %q を受理した", query)
			}
		})
	}
}

func TestNewRequestAcceptsQueryBoundaryAndTrimsOnlyU0020(t *testing.T) {
	t.Parallel()

	for name, query := range map[string]string{
		"512 byte 以下": strings.Repeat("法", 170),
		"改行でない空白":     "\u3000民法\u3000",
	} {
		query := query
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			request, err := lawsearch.NewRequest(lawsearch.RequestValues{Query: query})
			if err != nil {
				t.Fatalf("SOT-IF-022: 有効な query を拒否した: %v", err)
			}
			if request.Query() != query {
				t.Fatalf("SOT-IF-022: U+0020 以外を変更した: %q", request.Query())
			}
		})
	}
}

func TestNewRequestRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	for _, limit := range []int{0, -1, lawsearch.MaxLimit + 1} {
		limit := limit
		t.Run("limit", func(t *testing.T) {
			t.Parallel()

			if _, err := lawsearch.NewRequest(lawsearch.RequestValues{
				Query: "民法",
				Limit: &limit,
			}); err == nil {
				t.Fatalf("SOT-IF-022: limit=%d を受理した", limit)
			}
		})
	}
}

func TestNewRequestRejectsInvalidContinuationToken(t *testing.T) {
	t.Parallel()

	for name, token := range map[string]string{
		"不正な UTF-8":   string([]byte{'v', '1', '.', 0xff}),
		"4096 byte 超": strings.Repeat("x", lawsearch.MaxTokenBytes+1),
	} {
		token := token
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := lawsearch.NewRequest(lawsearch.RequestValues{
				Query:             "民法",
				ContinuationToken: token,
			}); err == nil {
				t.Fatalf("SOT-IF-016/SOT-IF-022: 不正な token を受理した")
			}
		})
	}
}

func TestRequestConditionObjectHasExactNormalizedKeys(t *testing.T) {
	t.Parallel()

	withoutDate, err := lawsearch.NewRequest(lawsearch.RequestValues{Query: " 民法 "})
	if err != nil {
		t.Fatalf("SOT-IF-022: NewRequest() のエラー = %v", err)
	}
	object, err := withoutDate.ConditionObject()
	if err != nil {
		t.Fatalf("SOT-IF-022: ConditionObject() のエラー = %v", err)
	}
	wantWithoutDate := []byte(`{"asOf":null,"limit":20,"query":"民法"}`)
	if !bytes.Equal(object.Bytes(), wantWithoutDate) {
		t.Fatalf("SOT-IF-022: condition = %s、期待値 = %s", object.Bytes(), wantWithoutDate)
	}

	asOf, dateErr := model.NewDate("2025-01-02")
	if dateErr != nil {
		t.Fatalf("試験用の日付を作成できない: %v", dateErr)
	}
	limit := 40
	withDate, err := lawsearch.NewRequest(lawsearch.RequestValues{
		Query: "行政法",
		AsOf:  &asOf,
		Limit: &limit,
	})
	if err != nil {
		t.Fatalf("SOT-IF-022: NewRequest() のエラー = %v", err)
	}
	object, err = withDate.ConditionObject()
	if err != nil {
		t.Fatalf("SOT-IF-022: ConditionObject() のエラー = %v", err)
	}
	wantWithDate := []byte(`{"asOf":"2025-01-02","limit":40,"query":"行政法"}`)
	if !bytes.Equal(object.Bytes(), wantWithDate) {
		t.Fatalf("SOT-IF-022: condition = %s、期待値 = %s", object.Bytes(), wantWithDate)
	}

	got := object.Bytes()
	got[0] = '['
	if !bytes.Equal(object.Bytes(), wantWithDate) {
		t.Fatal("SOT-IF-016/SOT-IF-022: condition object が外部から変更された")
	}
}

func TestRequestRejectsDirectJSONDecodeAndInvalidZeroValue(t *testing.T) {
	t.Parallel()

	var request lawsearch.Request
	if err := json.Unmarshal([]byte(`{"query":"民法"}`), &request); err == nil {
		t.Fatal("SOT-ENG-002/SOT-IF-022: Request を JSON から直接復元できた")
	}
	if err := request.Validate(); err == nil {
		t.Fatal("SOT-IF-022: Request のゼロ値を受理した")
	}
	if _, err := request.ConditionObject(); err == nil {
		t.Fatal("SOT-IF-022: Request のゼロ値から condition object を作成できた")
	}
}
