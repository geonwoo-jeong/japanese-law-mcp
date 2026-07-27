package hanrei

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestParseJapaneseDecisionDateEraBoundaries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		value string
		want  string
		valid bool
	}{
		{"明治元年1月25日", "1868-01-25", true},
		{"明治45年7月29日", "1912-07-29", true},
		{"大正元年7月30日", "1912-07-30", true},
		{"大正15年12月24日", "1926-12-24", true},
		{"昭和元年12月25日", "1926-12-25", true},
		{"昭和64年1月7日", "1989-01-07", true},
		{"平成元年1月8日", "1989-01-08", true},
		{"平成31年4月30日", "2019-04-30", true},
		{"令和元年5月1日", "2019-05-01", true},
		{"明治元年1月24日", "", false},
		{"大正元年7月29日", "", false},
		{"昭和64年1月8日", "", false},
		{"平成31年5月1日", "", false},
		{"令和元年4月30日", "", false},
		{"令和2年2月30日", "", false},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.value, func(t *testing.T) {
			t.Parallel()
			got, err := parseJapaneseDecisionDate(testCase.value)
			if !testCase.valid {
				if err == nil {
					t.Fatalf("SOT-IF-044: 不正日付を受理した: %s", got.String())
				}
				return
			}
			if err != nil || got.String() != testCase.want {
				t.Fatalf("parseJapaneseDecisionDate() = %q, %v", got.String(), err)
			}
		})
	}
}

func TestSearchHTMLContractErrors(t *testing.T) {
	t.Parallel()
	validRow := `
<table class="search-result-table"><tbody><tr>
<th><a href="./../1/detail2/index.html">最高裁判例</a></th>
<td><p>令和1(オ)1
事件名</p><p>令和元年5月1日
最高裁判所</p></td><td class="file-col"></td>
</tr></tbody></table>`
	cases := []struct {
		name string
		body string
		code model.SourceErrorCode
	}{
		{
			"missing title",
			`<html><head><title>別ページ</title></head><body><p>1件中</p>` + validRow + `</body></html>`,
			model.SourceErrorCodeSourceContractChanged,
		},
		{
			"missing table",
			`<html><head><title>裁判例検索</title></head><body><p>1件中</p></body></html>`,
			model.SourceErrorCodeSourceContractChanged,
		},
		{
			"category mismatch",
			`<html><head><title>裁判例検索</title></head><body><p>1件中</p>` +
				strings.Replace(validRow, "detail2", "detail3", 1) + `</body></html>`,
			model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			"count mismatch",
			`<html><head><title>裁判例検索</title></head><body><p>0件中</p>` +
				validRow + `</body></html>`,
			model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			"external PDF",
			`<html><head><title>裁判例検索</title></head><body><p>1件中</p>` +
				strings.Replace(
					validRow,
					`<td class="file-col"></td>`,
					`<td class="file-col"><a href="https://example.com/a.pdf">全文</a></td>`,
					1,
				) + `</body></html>`,
			model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			"duplicate detail link",
			`<html><head><title>裁判例検索</title></head><body><p>1件中</p>` +
				strings.Replace(
					validRow,
					`<a href="./../1/detail2/index.html">最高裁判例</a>`,
					`<a href="./../1/detail2/index.html">最高裁判例</a>`+
						`<a href="./../1/detail2/index.html">最高裁判例</a>`,
					1,
				) + `</body></html>`,
			model.SourceErrorCodeInvalidSourceResponse,
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			adapter := newTestAdapter(t, staticHTMLDoer([]byte(testCase.body)))
			_, err := adapter.Search(
				context.Background(),
				mustSearchRequest(t, "契約", 20, ""),
			)
			assertSourceError(t, err, testCase.code)
		})
	}
}

func TestParseSearchResponseRejectsUnsafeAndOversizedInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		body []byte
		code model.SourceErrorCode
	}{
		{
			name: "invalid UTF-8",
			body: []byte{0xff, 0xfe},
			code: model.SourceErrorCodeUnsafeSourceContent,
		},
		{
			name: "decompressed body",
			body: bytes.Repeat([]byte("x"), maximumSearchDecompressedBytes+1),
			code: model.SourceErrorCodeSourceResponseTooLarge,
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseSearchResponse(context.Background(), testCase.body)
			assertSourceError(t, err, testCase.code)
		})
	}
}

func TestParseSearchResponseIgnoresHiddenEmptyMarker(t *testing.T) {
	t.Parallel()
	body := []byte(`<html><head><title>裁判例検索</title></head><body>` +
		`<div hidden><p id="searched">該当する裁判例がありませんでした。</p></div>` +
		`</body></html>`)

	_, err := parseSearchResponse(context.Background(), body)
	assertSourceError(t, err, model.SourceErrorCodeSourceContractChanged)
}
