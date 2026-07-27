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

func TestParseSearchResponseClassifiesOfficialTooBroadQuery(t *testing.T) {
	t.Parallel()

	marker := `<ul class="errorMessage"><li><span>` +
		`検索結果が2000件を超えました。「全文検索」欄の検索語を追加・変更してください。` +
		`<br>（※上のタブを切り替えることで裁判所や事件の種類を絞り込んだ検索を行うこともできます。）` +
		`</span></li></ul>`
	page := func(content string) []byte {
		return []byte(`<html><head><title>裁判例検索</title></head><body>` +
			content +
			`</body></html>`)
	}
	validResult := `<p>1件中</p><table class="search-result-table"><tbody><tr>` +
		`<th><a href="./../1/detail2/index.html">最高裁判例</a></th>` +
		`<td><p>令和1(オ)1` + "\n" + `事件名</p>` +
		`<p>令和元年5月1日` + "\n" + `最高裁判所</p></td>` +
		`<td class="file-col"></td></tr></tbody></table>`
	cases := []struct {
		name string
		body []byte
		code model.SourceErrorCode
	}{
		{
			name: "visible official marker",
			body: page(marker),
			code: model.SourceErrorCodeUnsupportedQuery,
		},
		{
			name: "duplicate marker",
			body: page(marker + marker),
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "other error message conflict",
			body: page(
				marker +
					`<ul class="errorMessage"><li>別のエラーです。</li></ul>`,
			),
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "empty result conflict",
			body: page(
				marker +
					`<p id="searched">該当する裁判例がありませんでした。</p>`,
			),
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "result table conflict",
			body: page(
				marker +
					`<table class="search-result-table"><tbody></tbody></table>`,
			),
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "hidden marker",
			body: page(`<div hidden>` + marker + `</div>`),
			code: model.SourceErrorCodeSourceContractChanged,
		},
		{
			name: "inline style marker",
			body: page(`<div style="display: none">` + marker + `</div>`),
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "inline style marker descendant",
			body: page(
				`<ul class="errorMessage">` +
					`<li style="display: none">` +
					tooBroadSearchMessagePrefix +
					`</li><li>別のエラーです。</li></ul>`,
			),
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "closed details marker",
			body: page(`<details>` + marker + `</details>`),
			code: model.SourceErrorCodeSourceContractChanged,
		},
		{
			name: "closed details first summary marker",
			body: page(`<details><summary>` + marker + `</summary></details>`),
			code: model.SourceErrorCodeUnsupportedQuery,
		},
		{
			name: "closed details second summary marker",
			body: page(
				`<details><summary>概要</summary><summary>` +
					marker +
					`</summary></details>`,
			),
			code: model.SourceErrorCodeSourceContractChanged,
		},
		{
			name: "closed details hidden descendant text",
			body: page(
				`<ul class="errorMessage"><li><details><summary></summary>` +
					`<span>` + tooBroadSearchMessagePrefix + `</span>` +
					`</details></li></ul>`,
			),
			code: model.SourceErrorCodeSourceContractChanged,
		},
		{
			name: "closed dialog marker",
			body: page(`<dialog>` + marker + `</dialog>`),
			code: model.SourceErrorCodeSourceContractChanged,
		},
		{
			name: "open details marker",
			body: page(`<details open>` + marker + `</details>`),
			code: model.SourceErrorCodeUnsupportedQuery,
		},
		{
			name: "open dialog marker",
			body: page(`<dialog open>` + marker + `</dialog>`),
			code: model.SourceErrorCodeUnsupportedQuery,
		},
		{
			name: "unknown error and empty result conflict",
			body: page(
				`<ul class="errorMessage"><li>別のエラーです。</li></ul>` +
					`<p id="searched">該当する裁判例がありませんでした。</p>`,
			),
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "unknown error and result table conflict",
			body: page(
				`<ul class="errorMessage"><li>別のエラーです。</li></ul>` +
					validResult,
			),
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "unrecognized message",
			body: page(
				`<ul class="errorMessage"><li>別のエラーです。</li></ul>`,
			),
			code: model.SourceErrorCodeSourceContractChanged,
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

func TestParseSearchResponseIgnoresClosedDetailsContent(t *testing.T) {
	t.Parallel()

	body := []byte(`<html><head><title>裁判例検索</title></head><body>` +
		`<p>1件中</p><table class="search-result-table"><tbody><tr>` +
		`<th><a href="./../1/detail2/index.html">最高裁判例</a>` +
		`<details><summary>補足</summary>` +
		`<a href="./../2/detail3/index.html">非表示リンク</a></details></th>` +
		`<td><p>令和1(オ)1` + "\n" + `事件名</p>` +
		`<p>令和元年5月1日` + "\n" + `最高裁判所</p></td>` +
		`<td class="file-col"></td></tr></tbody></table></body></html>`)
	response, err := parseSearchResponse(context.Background(), body)
	if err != nil {
		t.Fatalf("SOT-IF-044: 閉じた details の非表示内容で失敗した: %v", err)
	}
	if len(response.rows) != 1 ||
		response.rows[0].detailHref != "./../1/detail2/index.html" {
		t.Fatalf("SOT-IF-044: 表示された結果行 = %#v", response.rows)
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
