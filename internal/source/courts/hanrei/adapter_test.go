package hanrei

import (
	"bytes"
	"compress/gzip"
	"context"
	"embed"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

//go:embed testdata/*.html
var searchFixtureFiles embed.FS

func TestJudicialDecisionSearchAdapterSearchMapsOfficialFixture(t *testing.T) {
	t.Parallel()
	body := readFixture(t, "search_all_categories.html")
	var calls atomic.Int32
	adapter := newTestAdapter(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		if request.Header.Get("Cookie") != "" {
			t.Fatal("SOT-IF-043: Cookie を送信した")
		}
		return htmlResponse(request, body), nil
	}))
	limit := 4
	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{Query: "判例 %2F", Limit: &limit},
	)
	if err != nil {
		t.Fatal(err)
	}

	page, err := adapter.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-041/SOT-IF-044: Search() のエラー = %v", err)
	}
	assertMappedPage(t, page, 4, 12)
	if calls.Load() != 1 {
		t.Fatalf("SOT-IF-043: 外部呼出し回数 = %d、期待値 = 1", calls.Load())
	}
}

func TestJudicialDecisionSearchAdapterMapsAllCategoriesAndDocuments(t *testing.T) {
	t.Parallel()
	adapter := newTestAdapter(t, staticHTMLDoer(readFixture(t, "search_all_categories.html")))
	request := mustSearchRequest(t, "公式裁判例", 30, "")

	page, err := adapter.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-044: Search() のエラー = %v", err)
	}
	items := page.Items()
	wantCategories := []model.JudicialPublicationCategory{
		model.JudicialPublicationCategorySupremeCourt,
		model.JudicialPublicationCategoryHighCourt,
		model.JudicialPublicationCategoryLowerCourt,
		model.JudicialPublicationCategoryAdministrative,
		model.JudicialPublicationCategoryLabor,
		model.JudicialPublicationCategoryIntellectualProperty,
		model.JudicialPublicationCategoryIntellectualProperty,
	}
	wantDates := []string{
		"1912-07-29",
		"1926-12-24",
		"1989-01-07",
		"2019-04-30",
		"2019-05-01",
		"2025-04-24",
		"2026-06-23",
	}
	for index, item := range items {
		if got := item.Data().PublicationCategory(); got != wantCategories[index] {
			t.Errorf("items[%d].publicationCategory = %q", index, got)
		}
		if got := item.Data().DecisionDate().String(); got != wantDates[index] {
			t.Errorf("items[%d].decisionDate = %q", index, got)
		}
	}
	assertOptionalMappings(t, items)
	assertDocuments(t, items[6].Data().Documents())
	assertResourceAndProvenance(t, items[6])
}

func TestJudicialDecisionSearchAdapterMapsEmptyOfficialFixture(t *testing.T) {
	t.Parallel()
	adapter := newTestAdapter(t, staticHTMLDoer(readFixture(t, "search_empty.html")))
	page, err := adapter.Search(
		context.Background(),
		mustSearchRequest(t, "該当なし", 20, ""),
	)
	if err != nil {
		t.Fatalf("SOT-IF-041/SOT-IF-044: 空結果のエラー = %v", err)
	}
	assertMappedPage(t, page, 0, 0)
	if page.Items() == nil {
		t.Fatal("SOT-IF-041: 空結果の items が nil")
	}
}

func TestJudicialDecisionSearchAdapterAcceptsDeclaredShiftJIS(t *testing.T) {
	t.Parallel()
	source := bytes.Replace(
		readFixture(t, "search_empty.html"),
		[]byte(`charset="utf-8"`),
		[]byte(`charset="Shift_JIS"`),
		1,
	)
	body, _, err := transform.Bytes(japanese.ShiftJIS.NewEncoder(), source)
	if err != nil {
		t.Fatal(err)
	}
	adapter := newTestAdapter(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := htmlResponse(request, body)
		response.Header.Set("Content-Type", "text/html; charset=Shift_JIS")
		return response, nil
	}))

	page, err := adapter.Search(
		context.Background(),
		mustSearchRequest(t, "文字コード", 20, ""),
	)
	if err != nil {
		t.Fatalf("SOT-IF-043/SOT-IF-044: Shift_JIS HTML のエラー = %v", err)
	}
	assertMappedPage(t, page, 0, 0)
}

func TestJudicialDecisionSearchAdapterPreservesDOMOrderAndDuplicates(t *testing.T) {
	t.Parallel()
	body := readFixture(t, "search_duplicates.html")
	adapter := newTestAdapter(t, staticHTMLDoer(body))

	page, err := adapter.Search(
		context.Background(),
		mustSearchRequest(t, "重複", 3, ""),
	)
	if err != nil {
		t.Fatalf("SOT-IF-041/SOT-IF-044: Search() のエラー = %v", err)
	}
	assertMappedPage(t, page, 3, 3)
	items := page.Items()
	wantIDs := []string{"00101", "00202", "00101"}
	for index, wantID := range wantIDs {
		if got := items[index].Data().DecisionID(); got != wantID {
			t.Errorf("items[%d].decisionId = %q、期待値 = %q", index, got, wantID)
		}
	}

	limitedAdapter := newTestAdapter(t, staticHTMLDoer(body))
	limited, err := limitedAdapter.Search(
		context.Background(),
		mustSearchRequest(t, "重複", 2, ""),
	)
	if err != nil {
		t.Fatalf("SOT-IF-044: limit 適用時の Search() のエラー = %v", err)
	}
	assertMappedPage(t, limited, 2, 3)
}

func TestJudicialDecisionSearchAdapterIgnoresTheadRowsInLocation(t *testing.T) {
	t.Parallel()

	body := []byte(`<!doctype html><html><head><title>裁判例検索</title></head><body>
<p>1件中</p>
<table class="search-result-table">
<thead><tr><th>区分</th><th>内容</th><th>ファイル</th></tr></thead>
<tbody><tr>
<th><a href="./../12345/detail2/index.html">最高裁判例</a></th>
<td><p>令和6年（受）第1号
損害賠償請求事件</p><p>令和6年5月1日
最高裁判所</p></td>
<td class="file-col"><a href="/app/files/hanrei_jp/345/12345_hanrei.pdf">全文</a></td>
</tr></tbody>
</table>
</body></html>`)
	adapter := newTestAdapter(t, staticHTMLDoer(body))

	page, err := adapter.Search(
		context.Background(),
		mustSearchRequest(t, "直接行", 20, ""),
	)
	if err != nil {
		t.Fatalf("direct tr の Search() エラー = %v", err)
	}
	items := page.Items()
	if len(items) != 1 {
		t.Fatalf("items の件数 = %d", len(items))
	}
	location, exists := items[0].Provenance()[0].Location()
	if !exists || location != "table.search-result-table tbody tr[1]" {
		t.Fatalf("thead 混在時の provenance.location = %q, %t", location, exists)
	}
}

func TestJudicialDecisionSearchAdapterIgnoresHiddenRowContent(t *testing.T) {
	t.Parallel()
	body := []byte(`<!doctype html><html><head><title>裁判例検索</title></head><body>
<p>1件中</p>
<table class="search-result-table"><tbody><tr>
<th>
  <a href="./../12345/detail2/index.html">最高裁判例</a>
  <span hidden><a href="./../54321/detail3/index.html">高裁判例</a></span>
</th>
<td>
  <p>令和6年（受）第1号<span aria-hidden="true">非表示事件</span>
損害賠償請求事件</p>
  <p>令和6年5月1日
最高裁判所</p>
  <div class="modal"><p>非表示日付</p><p>非表示裁判所</p></div>
</td>
<td class="file-col">
  <a href="/assets/hanrei/12345.pdf">全文</a>
  <span class="d-none"><a href="https://example.com/secret.pdf">非表示</a></span>
</td>
</tr></tbody></table>
</body></html>`)
	adapter := newTestAdapter(t, staticHTMLDoer(body))

	page, err := adapter.Search(
		context.Background(),
		mustSearchRequest(t, "非表示", 20, ""),
	)
	if err != nil {
		t.Fatalf("SOT-IF-044: 非表示要素を含む Search() のエラー = %v", err)
	}
	items := page.Items()
	if len(items) != 1 {
		t.Fatalf("items の件数 = %d", len(items))
	}
	summary := items[0].Data()
	caseName, exists := summary.CaseName()
	if !exists || caseName != "損害賠償請求事件" {
		t.Errorf("caseName = %q, %t", caseName, exists)
	}
	if documents := summary.Documents(); len(documents) != 1 ||
		documents[0].URL() != "https://www.courts.go.jp/assets/hanrei/12345.pdf" {
		t.Errorf("documents = %#v", documents)
	}
}

func TestJudicialDecisionSearchAdapterRejectsContinuationBeforeFetch(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	adapter := newTestAdapter(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("呼び出されてはなりません")
	}))
	request := mustSearchRequest(t, "秘密の検索語", 20, "秘密の継続トークン")

	_, err := adapter.Search(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "continuationToken") {
		t.Fatalf("SOT-IF-041/SOT-IF-044: 予約 token のエラー = %v", err)
	}
	var argumentError judicialdecisionsearch.ArgumentError
	if !errors.As(err, &argumentError) ||
		argumentError.Code() != model.ErrorCodeInvalidArgument ||
		argumentError.Field() != "continuationToken" {
		t.Fatalf("SOT-IF-044: 予約 token が invalid_argument ではない: %T %v", err, err)
	}
	if strings.Contains(err.Error(), "秘密") {
		t.Fatalf("SOT-IF-043: エラーが入力を含む: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("SOT-ENG-016: 拒否後の外部呼出し回数 = %d", calls.Load())
	}
}

func TestJudicialDecisionSearchAdapterAcceptsEmptyResultTable(t *testing.T) {
	t.Parallel()
	body := []byte(`<html><head><title>裁判例検索</title></head><body>` +
		`<p id="searched">該当する裁判例がありませんでした。</p>` +
		`<table class="search-result-table"><tbody></tbody></table>` +
		`</body></html>`)
	adapter := newTestAdapter(t, staticHTMLDoer(body))

	page, err := adapter.Search(
		context.Background(),
		mustSearchRequest(t, "空テーブル", 20, ""),
	)
	if err != nil {
		t.Fatalf("SOT-IF-044: 空結果 table のエラー = %v", err)
	}
	assertMappedPage(t, page, 0, 0)
}

func TestJudicialDecisionSearchAdapterNormalizesHTTPFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		doer httpDoer
		code model.SourceErrorCode
	}{
		{"429", statusDoer(http.StatusTooManyRequests), model.SourceErrorCodeRateLimited},
		{"500", statusDoer(http.StatusInternalServerError), model.SourceErrorCodeSourceUnavailable},
		{"404", statusDoer(http.StatusNotFound), model.SourceErrorCodeInvalidSourceResponse},
		{"timeout", roundTripFunc(timeoutFailure), model.SourceErrorCodeSourceTimeout},
		{"network", roundTripFunc(networkFailure), model.SourceErrorCodeSourceUnavailable},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			adapter := newTestAdapter(t, testCase.doer)
			_, err := adapter.Search(
				context.Background(),
				mustSearchRequest(t, "漏えい禁止の検索語", 20, ""),
			)
			assertSourceError(t, err, testCase.code)
			if strings.Contains(err.Error(), "漏えい禁止") {
				t.Fatalf("SOT-IF-017: エラーが検索語を含む: %v", err)
			}
		})
	}
}

func TestJudicialDecisionSearchAdapterRejectsBusyBeforeFetch(t *testing.T) {
	t.Parallel()
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	var calls atomic.Int32
	adapter := newTestAdapterWith(t, searchAdapterDependencies{
		doer: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, errors.New("呼び出されてはなりません")
		}),
		now:          fixedNow,
		gate:         gate,
		parseTimeout: searchParseTimeout,
	})

	_, err := adapter.Search(
		context.Background(),
		mustSearchRequest(t, "同時実行", 20, ""),
	)
	assertSourceError(t, err, model.SourceErrorCodeSourceBusy)
	if calls.Load() != 0 {
		t.Fatalf("SOT-ENG-016: source_busy 後の外部呼出し回数 = %d", calls.Load())
	}
}

func TestJudicialDecisionSearchAdapterReleasesGateAfterParserFailure(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	doer := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return htmlResponse(request, []byte("<html><title>別のページ</title></html>")), nil
		}
		return htmlResponse(request, readFixture(t, "search_empty.html")), nil
	})
	adapter := newTestAdapter(t, doer)
	request := mustSearchRequest(t, "契約", 20, "")

	_, firstErr := adapter.Search(context.Background(), request)
	assertSourceError(t, firstErr, model.SourceErrorCodeSourceContractChanged)
	if _, secondErr := adapter.Search(context.Background(), request); secondErr != nil {
		t.Fatalf("SOT-ENG-016: parser failure 後に枠が解放されていない: %v", secondErr)
	}
}

func TestJudicialDecisionSearchAdapterResponseBudgets(t *testing.T) {
	t.Parallel()
	t.Run("responseBytes", func(t *testing.T) {
		t.Parallel()
		body := bytes.Repeat([]byte("x"), maximumSearchResponseBytes+1)
		adapter := newTestAdapter(t, staticHTMLDoer(body))
		_, err := adapter.Search(context.Background(), mustSearchRequest(t, "上限", 20, ""))
		assertSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	})
	t.Run("decompressedBytes", func(t *testing.T) {
		t.Parallel()
		body := gzipBytes(t, bytes.Repeat([]byte("x"), maximumSearchDecompressedBytes+1))
		adapter := newTestAdapter(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
			response := htmlResponse(request, body)
			response.Header.Set("Content-Encoding", "gzip")
			return response, nil
		}))
		_, err := adapter.Search(context.Background(), mustSearchRequest(t, "展開", 20, ""))
		assertSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	})
}

func TestJudicialDecisionSearchAdapterParseBudgets(t *testing.T) {
	t.Parallel()
	t.Run("nodes", func(t *testing.T) {
		t.Parallel()
		body := []byte("<!doctype html><html><head><title>裁判例検索</title></head><body>" +
			strings.Repeat("<i></i>", maximumSearchHTMLNodes+1) + "</body></html>")
		adapter := newTestAdapter(t, staticHTMLDoer(body))
		_, err := adapter.Search(context.Background(), mustSearchRequest(t, "node", 20, ""))
		assertSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	})
	t.Run("depth", func(t *testing.T) {
		t.Parallel()
		body := []byte("<!doctype html><html><head><title>裁判例検索</title></head><body>" +
			strings.Repeat("<div>", maximumSearchHTMLDepth+1) +
			strings.Repeat("</div>", maximumSearchHTMLDepth+1) + "</body></html>")
		adapter := newTestAdapter(t, staticHTMLDoer(body))
		_, err := adapter.Search(context.Background(), mustSearchRequest(t, "depth", 20, ""))
		assertSourceError(t, err, model.SourceErrorCodeUnsafeSourceContent)
	})
	t.Run("timeout", func(t *testing.T) {
		t.Parallel()
		adapter := newTestAdapterWith(t, searchAdapterDependencies{
			doer:         staticHTMLDoer(readFixture(t, "search_empty.html")),
			now:          fixedNow,
			gate:         make(chan struct{}, 1),
			parseTimeout: 0,
		})
		_, err := adapter.Search(context.Background(), mustSearchRequest(t, "timeout", 20, ""))
		assertSourceError(t, err, model.SourceErrorCodeSourceProcessingLimit)
	})
}

func TestJudicialDecisionSearchAdapterRejectsInvalidContentWithoutSensitiveData(t *testing.T) {
	t.Parallel()
	bodyMarker := "本文にだけある機密文字列"
	adapter := newTestAdapter(t, staticHTMLDoer([]byte(
		"<html><head><title>裁判例検索</title></head><body>"+bodyMarker+"</body></html>",
	)))
	_, err := adapter.Search(
		context.Background(),
		mustSearchRequest(t, "query-secret", 20, ""),
	)
	assertSourceError(t, err, model.SourceErrorCodeSourceContractChanged)
	if strings.Contains(err.Error(), bodyMarker) || strings.Contains(err.Error(), "query-secret") {
		t.Fatalf("SOT-IF-017/SOT-IF-043: エラーが機密内容を含む: %v", err)
	}
}

func assertMappedPage(
	t *testing.T,
	page judicialdecisionsearch.Page,
	returned int,
	total int,
) {
	t.Helper()
	if got := len(page.Items()); got != returned {
		t.Errorf("items の件数 = %d、期待値 = %d", got, returned)
	}
	if got := page.Page().ReturnedCount(); got != returned {
		t.Errorf("returnedCount = %d、期待値 = %d", got, returned)
	}
	gotTotal, exists := page.Page().TotalCount()
	if !exists || gotTotal != total {
		t.Errorf("totalCount = %d, %t、期待値 = %d, true", gotTotal, exists, total)
	}
	if relation, exists := page.Page().TotalRelation(); !exists || relation != model.TotalRelationExact {
		t.Errorf("totalRelation = %q, %t", relation, exists)
	}
	if _, exists := page.Page().NextToken(); exists {
		t.Error("SOT-IF-044: nextToken が発行された")
	}
}

func assertOptionalMappings(
	t *testing.T,
	items []model.SourcedResource[model.JudicialDecisionSummary],
) {
	t.Helper()
	branch, branchExists := items[2].Data().BranchName()
	division, divisionExists := items[2].Data().DivisionName()
	if !branchExists || branch != "沼津支部" || !divisionExists || division != "民事部" {
		t.Errorf("SOT-IF-044: 支部・部の対応 = %q/%t, %q/%t", branch, branchExists, division, divisionExists)
	}
	division, divisionExists = items[6].Data().DivisionName()
	decisionType, typeExists := items[6].Data().DecisionType()
	outcome, outcomeExists := items[6].Data().Outcome()
	if !divisionExists || division != "1部" ||
		!typeExists || decisionType != "判決" ||
		!outcomeExists || outcome != "請求棄却" {
		t.Errorf("SOT-IF-044: 知財高裁の省略可能項目が一致しない")
	}
}

func assertDocuments(t *testing.T, documents []model.JudicialDocumentLink) {
	t.Helper()
	if len(documents) != 3 {
		t.Fatalf("documents の件数 = %d", len(documents))
	}
	wantKinds := []model.JudicialDocumentKind{
		model.JudicialDocumentKindFullText,
		model.JudicialDocumentKindSummary,
		model.JudicialDocumentKindAttachment,
	}
	for index, document := range documents {
		if document.Kind() != wantKinds[index] ||
			document.MediaType() != model.JudicialDocumentMediaTypePDF {
			t.Errorf("documents[%d] = kind %q, mediaType %q", index, document.Kind(), document.MediaType())
		}
		if !strings.HasPrefix(document.URL(), "https://www.courts.go.jp/") {
			t.Errorf("documents[%d].url = %q", index, document.URL())
		}
	}
}

func assertResourceAndProvenance(
	t *testing.T,
	item model.SourcedResource[model.JudicialDecisionSummary],
) {
	t.Helper()
	ref := item.Ref()
	if ref.ProviderID() != providerID ||
		ref.Key().SourceID() != sourceID ||
		ref.Key().ResourceType() != "judicial-decision" ||
		ref.Key().ResourceID() != "00078/detail8" {
		t.Errorf("SOT-IF-044: ref が一致しない")
	}
	provenance := item.Provenance()
	if len(provenance) != 1 {
		t.Fatalf("provenance の件数 = %d", len(provenance))
	}
	got := provenance[0]
	methodID, _ := got.MethodID()
	location, _ := got.Location()
	if got.URL() != searchEndpoint+"?query1=%E5%85%AC%E5%BC%8F%E8%A3%81%E5%88%A4%E4%BE%8B" ||
		got.MediaType() != "text/html" ||
		got.Transformation() != model.ProvenanceTransformationNormalized ||
		methodID != "SOT-IF-044" ||
		location != "table.search-result-table tbody tr[7]" ||
		got.ResourceKey() != ref.Key() {
		t.Errorf("SOT-IF-044: provenance が一致しない")
	}
}

func assertSourceError(t *testing.T, err error, code model.SourceErrorCode) {
	t.Helper()
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("情報源エラーではない: %T %v", err, err)
	}
	if sourceError.Code() != code ||
		sourceError.ProviderID() != providerID ||
		sourceError.SourceID() != sourceID ||
		sourceError.CapabilityID() != judicialdecisionsearch.CapabilityID ||
		sourceError.Operation() != string(operationSearch) {
		t.Errorf("情報源エラー = %#v", sourceError)
	}
}

func newTestAdapter(t *testing.T, doer httpDoer) *JudicialDecisionSearchAdapter {
	t.Helper()
	return newTestAdapterWith(t, searchAdapterDependencies{
		doer:         doer,
		now:          fixedNow,
		gate:         make(chan struct{}, 1),
		parseTimeout: searchParseTimeout,
	})
}

func newTestAdapterWith(
	t *testing.T,
	dependencies searchAdapterDependencies,
) *JudicialDecisionSearchAdapter {
	t.Helper()
	adapter, err := newJudicialDecisionSearchAdapter(dependencies)
	if err != nil {
		t.Fatalf("adapter の作成エラー = %v", err)
	}
	return adapter
}

func mustSearchRequest(
	t *testing.T,
	query string,
	limit int,
	token string,
) judicialdecisionsearch.Request {
	t.Helper()
	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{
			Query:             query,
			Limit:             &limit,
			ContinuationToken: token,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := searchFixtureFiles.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func fixedNow() time.Time {
	return time.Date(2026, 7, 27, 1, 2, 3, 456, time.FixedZone("JST", 9*60*60))
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

func staticHTMLDoer(body []byte) httpDoer {
	return roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return htmlResponse(request, body), nil
	})
}

func statusDoer(status int) httpDoer {
	return roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := htmlResponse(request, []byte("secret response body"))
		response.StatusCode = status
		return response, nil
	})
}

func htmlResponse(request *http.Request, body []byte) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Type": []string{"text/html;charset=UTF-8"}},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       request,
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "secret timeout detail" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func timeoutFailure(*http.Request) (*http.Response, error) {
	return nil, timeoutError{}
}

func networkFailure(*http.Request) (*http.Response, error) {
	return nil, &net.DNSError{Err: "secret network detail", Name: "secret.example"}
}

func gzipBytes(t *testing.T, body []byte) []byte {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return compressed.Bytes()
}
