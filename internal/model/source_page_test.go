package model_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestSourcePageEmptyResult(t *testing.T) {
	t.Parallel()

	got, err := model.NewSourcePage(model.SourcePageValues{})
	if err != nil {
		t.Fatalf("SOT-MODEL-014: NewSourcePage() のエラー = %v", err)
	}
	if got.ReturnedCount() != 0 {
		t.Fatalf("SOT-MODEL-014: ReturnedCount() = %d", got.ReturnedCount())
	}
	if value, ok := got.NextToken(); ok || value != "" {
		t.Fatalf("SOT-MODEL-009/014: NextToken() = %q, %t", value, ok)
	}
	if value, ok := got.TotalCount(); ok || value != 0 {
		t.Fatalf("SOT-MODEL-009/014: TotalCount() = %d, %t", value, ok)
	}
	if value, ok := got.TotalRelation(); ok || value != "" {
		t.Fatalf("SOT-MODEL-009/014: TotalRelation() = %q, %t", value, ok)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-014: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/014: json.Marshal() のエラー = %v", err)
	}
	if string(encoded) != `{"returnedCount":0}` {
		t.Fatalf("SOT-MODEL-009/014: JSON = %s", encoded)
	}
}

func TestSourcePageWithExactTotal(t *testing.T) {
	t.Parallel()

	totalCount := 25
	opaqueToken := strings.Repeat("a", 16)
	got, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: 10,
		NextToken:     opaqueToken,
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-014: NewSourcePage() のエラー = %v", err)
	}

	totalCount = 99
	if got.ReturnedCount() != 10 {
		t.Fatalf("SOT-MODEL-014: ReturnedCount() = %d", got.ReturnedCount())
	}
	if value, ok := got.NextToken(); !ok || value != opaqueToken {
		t.Fatalf("SOT-MODEL-014: NextToken() = %q, %t", value, ok)
	}
	if value, ok := got.TotalCount(); !ok || value != 25 {
		t.Fatalf("SOT-MODEL-014: TotalCount() = %d, %t", value, ok)
	}
	if value, ok := got.TotalRelation(); !ok || value != model.TotalRelationExact {
		t.Fatalf("SOT-MODEL-014: TotalRelation() = %q, %t", value, ok)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/014: json.Marshal() のエラー = %v", err)
	}
	wantJSON := `{"returnedCount":10,"nextToken":"` +
		opaqueToken +
		`","totalCount":25,"totalRelation":"exact"}`
	if string(encoded) != wantJSON {
		t.Fatalf("SOT-MODEL-009/014: JSON = %s", encoded)
	}
}

func TestSourcePagePreservesPresentZeroTotal(t *testing.T) {
	t.Parallel()

	totalCount := 0
	got, err := model.NewSourcePage(model.SourcePageValues{
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-014: NewSourcePage() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/014: json.Marshal() のエラー = %v", err)
	}
	if string(encoded) != `{"returnedCount":0,"totalCount":0,"totalRelation":"exact"}` {
		t.Fatalf("SOT-MODEL-009/014: JSON = %s", encoded)
	}
}

func TestSourcePageAcceptsLowerBoundBelowReturnedCount(t *testing.T) {
	t.Parallel()

	totalCount := 1
	got, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: 10,
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationLowerBound,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-014: lower_bound を拒否した: %v", err)
	}
	if value, ok := got.TotalRelation(); !ok || value != model.TotalRelationLowerBound {
		t.Fatalf("SOT-MODEL-014: TotalRelation() = %q, %t", value, ok)
	}
}

func TestSourcePageContinuationTokenBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		nextToken string
		wantError bool
	}{
		{
			name:      "4096 byte",
			nextToken: strings.Repeat("a", 4096),
		},
		{
			name:      "4097 byte",
			nextToken: strings.Repeat("a", 4097),
			wantError: true,
		},
		{
			name:      "複数 byte 文字を含む 4096 byte",
			nextToken: strings.Repeat("あ", 1365) + "a",
		},
		{
			name:      "複数 byte 文字を含む 4097 byte",
			nextToken: strings.Repeat("あ", 1365) + "ab",
			wantError: true,
		},
		{
			name:      "UTF-8 として不正",
			nextToken: string([]byte{0xff}),
			wantError: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := model.NewSourcePage(model.SourcePageValues{
				NextToken: test.nextToken,
			})
			if (err != nil) != test.wantError {
				t.Fatalf(
					"SOT-MODEL-014/SOT-IF-016: NewSourcePage() のエラー = %v、期待値 = %t",
					err,
					test.wantError,
				)
			}
		})
	}
}

func TestSourcePageRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	zero := 0
	negative := -1
	tests := map[string]model.SourcePageValues{
		"returnedCount が負": {
			ReturnedCount: -1,
		},
		"totalCount が負": {
			TotalCount:    &negative,
			TotalRelation: model.TotalRelationExact,
		},
		"totalRelation が欠落": {
			TotalCount: &zero,
		},
		"totalCount が欠落": {
			TotalRelation: model.TotalRelationExact,
		},
		"totalRelation が未知": {
			TotalCount:    &zero,
			TotalRelation: model.TotalRelation("estimated"),
		},
		"exact の総数より返却数が多い": {
			ReturnedCount: 2,
			TotalCount:    &zero,
			TotalRelation: model.TotalRelationExact,
		},
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewSourcePage(values); err == nil {
				t.Fatal("SOT-MODEL-014: 不正な値を受理した")
			}
		})
	}
}

func TestSourcePageZeroValueRepresentsEmptyResult(t *testing.T) {
	t.Parallel()

	var page model.SourcePage
	if err := page.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-014: ゼロ値の Validate() エラー = %v", err)
	}
	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/014: ゼロ値の json.Marshal() エラー = %v", err)
	}
	if string(encoded) != `{"returnedCount":0}` {
		t.Fatalf("SOT-MODEL-009/014: JSON = %s", encoded)
	}
}

func TestSourcePageRejectsDirectJSONDecode(t *testing.T) {
	t.Parallel()

	var page model.SourcePage
	err := json.Unmarshal([]byte(`{"returnedCount":0}`), &page)
	if err == nil {
		t.Fatal("SOT-ENG-002: 境界専用の入力型を介さない JSON 復元を受理した")
	}
}
