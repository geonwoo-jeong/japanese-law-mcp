package model_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLawArticleLocation(t *testing.T) {
	t.Parallel()

	paragraph := 2
	got, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:       model.LawArticleProvisionSupplementary,
		ArticleNumber:   "38_3_2",
		ParagraphNumber: &paragraph,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-018: NewLawArticleLocation() のエラー = %v", err)
	}
	paragraph = 9

	if got.Provision() != model.LawArticleProvisionSupplementary {
		t.Fatalf("SOT-MODEL-018: Provision() = %q", got.Provision())
	}
	if got.ArticleNumber() != "38_3_2" {
		t.Fatalf("SOT-MODEL-018: ArticleNumber() = %q", got.ArticleNumber())
	}
	if value, exists := got.ParagraphNumber(); !exists || value != 2 {
		t.Fatalf("SOT-MODEL-018: ParagraphNumber() = %d, %t", value, exists)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-018: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/018: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/018: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"provision":       "supplementary",
		"articleNumber":   "38_3_2",
		"paragraphNumber": float64(2),
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/018: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestLawArticleLocationOmitsParagraphNumber(t *testing.T) {
	t.Parallel()

	got, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: "1",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-018: NewLawArticleLocation() のエラー = %v", err)
	}
	if value, exists := got.ParagraphNumber(); exists || value != 0 {
		t.Fatalf("SOT-MODEL-018: ParagraphNumber() = %d, %t", value, exists)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/018: json.Marshal() のエラー = %v", err)
	}
	if string(encoded) != `{"provision":"main","articleNumber":"1"}` {
		t.Fatalf("SOT-MODEL-009/018: JSON = %s", encoded)
	}
}

func TestLawArticleLocationArticleNumberBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		articleNumber string
		wantError     bool
	}{
		{name: "一桁", articleNumber: "1"},
		{name: "枝番号", articleNumber: "38_3_2"},
		{name: "64 byte", articleNumber: strings.Repeat("1", 64)},
		{name: "65 byte", articleNumber: strings.Repeat("1", 65), wantError: true},
		{name: "空", articleNumber: "", wantError: true},
		{name: "零", articleNumber: "0", wantError: true},
		{name: "先頭の零", articleNumber: "01", wantError: true},
		{name: "枝番号の先頭の零", articleNumber: "1_02", wantError: true},
		{name: "空の segment", articleNumber: "1__2", wantError: true},
		{name: "末尾の区切り", articleNumber: "1_", wantError: true},
		{name: "全角数字", articleNumber: "１", wantError: true},
		{name: "漢数字", articleNumber: "一", wantError: true},
		{name: "別の区切り", articleNumber: "1-2", wantError: true},
		{name: "UTF-8 として不正", articleNumber: string([]byte{0xff}), wantError: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
				Provision:     model.LawArticleProvisionMain,
				ArticleNumber: test.articleNumber,
			})
			if (err != nil) != test.wantError {
				t.Fatalf(
					"SOT-MODEL-018: articleNumber = %q、エラー = %v、期待値 = %t",
					test.articleNumber,
					err,
					test.wantError,
				)
			}
		})
	}
}

func TestLawArticleLocationRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	zero := 0
	negative := -1
	tests := map[string]model.LawArticleLocationValues{
		"provision の欠落": {
			ArticleNumber: "1",
		},
		"未知の provision": {
			Provision:     model.LawArticleProvision("amendment"),
			ArticleNumber: "1",
		},
		"paragraphNumber が零": {
			Provision:       model.LawArticleProvisionMain,
			ArticleNumber:   "1",
			ParagraphNumber: &zero,
		},
		"paragraphNumber が負": {
			Provision:       model.LawArticleProvisionMain,
			ArticleNumber:   "1",
			ParagraphNumber: &negative,
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewLawArticleLocation(values); err == nil {
				t.Fatalf("SOT-MODEL-018: NewLawArticleLocation(%#v) が成功した", values)
			}
		})
	}
}

func TestLawArticleLocationZeroValueCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.LawArticleLocation{}); err == nil {
		t.Fatal("SOT-MODEL-009/018: LawArticleLocation のゼロ値を JSON に変換できた")
	}
}

func TestLawArticleLocationRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.LawArticleLocation
	if err := json.Unmarshal(
		[]byte(`{"provision":"main","articleNumber":"1"}`),
		&got,
	); err == nil {
		t.Fatal("SOT-MODEL-009/018: LawArticleLocation を JSON から直接復元できた")
	}
}
