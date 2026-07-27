package hanrei

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestJudicialDecisionReadAdapterMapsOfficialLikeFixture(t *testing.T) {
	t.Parallel()
	adapter := newTestReadAdapter(t, staticHTMLDoer([]byte(readDetailHighCourtHTML)))
	request := mustReadRequest(t, providerID, sourceID, "95878/detail3")

	result, err := adapter.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-042/SOT-IF-045: Read() のエラー = %v", err)
	}
	if got := result.Ref().Key().ResourceID(); got != "95878/detail3" {
		t.Fatalf("ref.key.resourceId = %q", got)
	}
	details := result.Data()
	summary := details.Summary()
	if summary.PublicationCategory() != model.JudicialPublicationCategoryHighCourt {
		t.Fatalf("publicationCategory = %q", summary.PublicationCategory())
	}
	if summary.SourceCategoryLabel() != "高等裁判所" {
		t.Fatalf("sourceCategoryLabel = %q", summary.SourceCategoryLabel())
	}
	if summary.CourtName() != "東京高等裁判所 第６刑事部" {
		t.Fatalf("courtName = %q", summary.CourtName())
	}
	if outcome, _ := summary.Outcome(); outcome != "棄却" {
		t.Fatalf("outcome = %q", outcome)
	}
	if citation, _ := details.ReporterCitation(); citation != "第77巻2号23頁" {
		t.Fatalf("reporterCitation = %q", citation)
	}
	if holding, _ := details.HoldingText(); !strings.Contains(holding, "刑法１７７条") {
		t.Fatalf("holdingText = %q", holding)
	}
	if summaryText, _ := details.SummaryText(); !strings.Contains(summaryText, "憲法１４条") {
		t.Fatalf("summaryText = %q", summaryText)
	}
	documents := summary.Documents()
	if len(documents) != 1 || documents[0].Kind() != model.JudicialDocumentKindFullText {
		t.Fatalf("documents = %#v", documents)
	}
}

func TestJudicialDecisionReadAdapterMapsAliasLabelsAndAttachments(t *testing.T) {
	t.Parallel()
	adapter := newTestReadAdapter(t, staticHTMLDoer([]byte(readDetailLowerCourtHTML)))
	request := mustReadRequest(t, providerID, sourceID, "95877/detail4")

	result, err := adapter.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("Read() のエラー = %v", err)
	}
	summary := result.Data().Summary()
	if summary.CourtName() != "東京地方裁判所 立川支部 第１刑事部" {
		t.Fatalf("courtName = %q", summary.CourtName())
	}
	if holding, _ := result.Data().HoldingText(); !strings.Contains(holding, "判示事項") {
		t.Fatalf("holdingText = %q", holding)
	}
	documents := summary.Documents()
	if len(documents) != 2 || documents[1].Kind() != model.JudicialDocumentKindAttachment {
		t.Fatalf("documents = %#v", documents)
	}
}

func TestJudicialDecisionReadAdapterMapsIPSummaryAliasWithoutIPOnlyFields(t *testing.T) {
	t.Parallel()
	adapter := newTestReadAdapter(t, staticHTMLDoer([]byte(readDetailIPHTML)))
	request := mustReadRequest(t, providerID, sourceID, "95878/detail8")

	result, err := adapter.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("Read() のエラー = %v", err)
	}
	if summaryText, exists := result.Data().SummaryText(); !exists || summaryText != "争点の要約" {
		t.Fatalf("summaryText = %q, %t", summaryText, exists)
	}
	if division, exists := result.Data().Summary().DivisionName(); !exists || division != "第４部" {
		t.Fatalf("divisionName = %q, %t", division, exists)
	}
	if decisionType, exists := result.Data().Summary().DecisionType(); exists {
		t.Fatalf("事件種別を decisionType に混入した: %q", decisionType)
	}
}

func TestJudicialDecisionReadAdapterMapsEveryDetailCategory(t *testing.T) {
	t.Parallel()
	cases := []struct {
		resourceID string
		label      string
		category   model.JudicialPublicationCategory
	}{
		{"1/detail2", "最高裁判所", model.JudicialPublicationCategorySupremeCourt},
		{"2/detail3", "高等裁判所", model.JudicialPublicationCategoryHighCourt},
		{"3/detail4", "下級裁判所(速報)", model.JudicialPublicationCategoryLowerCourt},
		{"4/detail5", "行政事件", model.JudicialPublicationCategoryAdministrative},
		{"5/detail6", "労働事件", model.JudicialPublicationCategoryLabor},
		{"6/detail7", "知的財産事件", model.JudicialPublicationCategoryIntellectualProperty},
		{"7/detail8", "知的財産高等裁判所", model.JudicialPublicationCategoryIntellectualProperty},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.resourceID, func(t *testing.T) {
			t.Parallel()
			adapter := newTestReadAdapter(
				t,
				staticHTMLDoer([]byte(minimalReadDetailHTML(testCase.label, ""))),
			)
			result, err := adapter.Read(
				context.Background(),
				mustReadRequest(t, providerID, sourceID, testCase.resourceID),
			)
			if err != nil {
				t.Fatalf("Read() のエラー = %v", err)
			}
			summary := result.Data().Summary()
			if summary.PublicationCategory() != testCase.category ||
				summary.SourceCategoryLabel() != testCase.label {
				t.Fatalf(
					"カテゴリー = %q, %q",
					summary.PublicationCategory(),
					summary.SourceCategoryLabel(),
				)
			}
		})
	}
}

func TestJudicialDecisionReadAdapterRejectsCategoryHeadingMismatch(t *testing.T) {
	t.Parallel()
	adapter := newTestReadAdapter(
		t,
		staticHTMLDoer([]byte(minimalReadDetailHTML("高等裁判所", ""))),
	)
	_, err := adapter.Read(
		context.Background(),
		mustReadRequest(t, providerID, sourceID, "1/detail2"),
	)
	assertReadSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
}

func TestJudicialDecisionReadAdapterRejectsInvalidOptionalValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		extra string
	}{
		{
			name:  "conflicting outcome aliases",
			extra: readDL("結果", "棄却") + readDL("判決結果", "破棄"),
		},
		{
			name:  "invalid lower court date",
			extra: readDL("原審裁判年月日", "令和元年4月30日"),
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			adapter := newTestReadAdapter(
				t,
				staticHTMLDoer([]byte(
					minimalReadDetailHTML("最高裁判所", testCase.extra),
				)),
			)
			_, err := adapter.Read(
				context.Background(),
				mustReadRequest(t, providerID, sourceID, "1/detail2"),
			)
			assertReadSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestJudicialDecisionReadAdapterKeepsSeparateBranchAndMultilineText(t *testing.T) {
	t.Parallel()
	extra := readDL("支部名", "立川支部") +
		readDL("判示事項", "第一行\n\n第二行")
	adapter := newTestReadAdapter(
		t,
		staticHTMLDoer([]byte(minimalReadDetailHTML("下級裁判所(速報)", extra))),
	)
	result, err := adapter.Read(
		context.Background(),
		mustReadRequest(t, providerID, sourceID, "3/detail4"),
	)
	if err != nil {
		t.Fatalf("Read() のエラー = %v", err)
	}
	if branch, exists := result.Data().Summary().BranchName(); !exists || branch != "立川支部" {
		t.Fatalf("branchName = %q, %t", branch, exists)
	}
	if holding, exists := result.Data().HoldingText(); !exists || holding != "第一行\n\n第二行" {
		t.Fatalf("holdingText = %q, %t", holding, exists)
	}
}

func TestJudicialDecisionReadAdapterSeparatesMixedSummaryTextAndPDF(t *testing.T) {
	t.Parallel()
	extra := `<dl><dt>要旨</dt><dd><p>争点の要約
<a href="./../../../assets/hanrei/summary-7.pdf">要旨</a></p></dd></dl>`
	adapter := newTestReadAdapter(
		t,
		staticHTMLDoer([]byte(
			minimalReadDetailHTML("知的財産高等裁判所", extra),
		)),
	)
	result, err := adapter.Read(
		context.Background(),
		mustReadRequest(t, providerID, sourceID, "7/detail8"),
	)
	if err != nil {
		t.Fatalf("Read() のエラー = %v", err)
	}
	if summaryText, exists := result.Data().SummaryText(); !exists || summaryText != "争点の要約" {
		t.Fatalf("summaryText = %q, %t", summaryText, exists)
	}
	documents := result.Data().Summary().Documents()
	if len(documents) != 1 ||
		documents[0].Kind() != model.JudicialDocumentKindSummary ||
		documents[0].Label() != "要旨" {
		t.Fatalf("documents = %#v", documents)
	}
}

func TestJudicialDecisionReadAdapterUsesReadErrorForInvalidMappedSource(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		extra string
	}{
		{name: "invalid primary date", extra: ""},
		{
			name:  "unsafe document URL",
			extra: `<dl><dt>全文</dt><dd><a href="https://example.com/case.pdf">全文</a></dd></dl>`,
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			body := minimalReadDetailHTML("最高裁判所", testCase.extra)
			if testCase.name == "invalid primary date" {
				body = strings.Replace(
					body,
					"令和7年10月16日",
					"令和元年4月30日",
					1,
				)
			}
			adapter := newTestReadAdapter(t, staticHTMLDoer([]byte(body)))
			_, err := adapter.Read(
				context.Background(),
				mustReadRequest(t, providerID, sourceID, "1/detail2"),
			)
			assertReadSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestJudicialDecisionReadAdapterKeepsReadProvenance(t *testing.T) {
	t.Parallel()
	adapter := newTestReadAdapter(
		t,
		staticHTMLDoer([]byte(minimalReadDetailHTML("最高裁判所", ""))),
	)
	request := mustReadRequest(t, providerID, sourceID, "1/detail2")
	result, err := adapter.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("Read() のエラー = %v", err)
	}
	wantURL := "https://www.courts.go.jp/hanrei/1/detail2/index.html"
	provenance := result.Provenance()
	methodID, hasMethodID := provenance[0].MethodID()
	if result.Ref() != request.Ref() ||
		result.Data().Summary().DetailURL() != wantURL ||
		len(provenance) != 1 ||
		provenance[0].ResourceKey() != request.Ref().Key() ||
		provenance[0].URL() != wantURL ||
		provenance[0].MediaType() != "text/html" ||
		provenance[0].Transformation() != model.ProvenanceTransformationNormalized ||
		!hasMethodID ||
		methodID != "SOT-IF-045" {
		t.Fatalf("詳細取得の ref または provenance が一致しない: %#v", result)
	}
}

func TestJudicialDecisionReadAdapterKeepsMixedSummaryTextAndPDF(t *testing.T) {
	t.Parallel()
	adapter := newTestReadAdapter(t, staticHTMLDoer([]byte(readDetailMixedSummaryHTML)))
	request := mustReadRequest(t, providerID, sourceID, "95879/detail8")

	result, err := adapter.Read(context.Background(), request)
	if err != nil {
		t.Fatalf("Read() のエラー = %v", err)
	}
	if summaryText, exists := result.Data().SummaryText(); !exists || summaryText != "テキスト要旨" {
		t.Fatalf("summaryText = %q, %t", summaryText, exists)
	}
	documents := result.Data().Summary().Documents()
	if len(documents) != 2 || documents[0].Kind() != model.JudicialDocumentKindSummary {
		t.Fatalf("documents = %#v", documents)
	}
}

func TestJudicialDecisionReadAdapterRejectsInvalidRefBeforeFetch(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	adapter := newTestReadAdapter(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("呼び出されてはなりません")
	}))
	request := mustReadRequest(t, providerID, sourceID, "bad")

	_, err := adapter.Read(context.Background(), request)
	var argumentError judicialdecisionread.ArgumentError
	if !errors.As(err, &argumentError) || argumentError.Code() != model.ErrorCodeInvalidArgument {
		t.Fatalf("invalid_argument ではない: %T %v", err, err)
	}
	if calls.Load() != 0 {
		t.Fatalf("外部呼出し回数 = %d", calls.Load())
	}
}

func TestJudicialDecisionReadAdapterReturnsNotFoundOn404(t *testing.T) {
	t.Parallel()
	adapter := newTestReadAdapter(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := htmlResponse(request, []byte("<html></html>"))
		response.StatusCode = http.StatusNotFound
		return response, nil
	}))

	_, err := adapter.Read(
		context.Background(),
		mustReadRequest(t, providerID, sourceID, "95878/detail3"),
	)
	if !errors.Is(err, judicialdecisionread.ErrNotFound) {
		t.Fatalf("ErrNotFound ではない: %v", err)
	}
}

func TestJudicialDecisionReadAdapterSharesGateWithSearch(t *testing.T) {
	t.Parallel()
	gate := make(chan struct{}, 1)
	if _, err := newJudicialDecisionSearchAdapter(searchAdapterDependencies{
		doer:         staticHTMLDoer(readFixture(t, "search_empty.html")),
		now:          fixedNow,
		gate:         gate,
		parseTimeout: searchParseTimeout,
	}); err != nil {
		t.Fatal(err)
	}
	adapter, err := newJudicialDecisionReadAdapter(readAdapterDependencies{
		doer:         staticHTMLDoer([]byte(readDetailHighCourtHTML)),
		now:          fixedNow,
		gate:         gate,
		parseTimeout: readParseTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	gate <- struct{}{}
	defer func() { <-gate }()

	_, err = adapter.Read(
		context.Background(),
		mustReadRequest(t, providerID, sourceID, "95878/detail3"),
	)
	assertReadSourceError(t, err, model.SourceErrorCodeSourceBusy)
}

func TestJudicialDecisionReadAdapterRoundTripsSearchRef(t *testing.T) {
	t.Parallel()
	searchAdapter := newTestAdapter(t, staticHTMLDoer(readFixture(t, "search_all_categories.html")))
	page, err := searchAdapter.Search(
		context.Background(),
		mustSearchRequest(t, "公式裁判例", 30, ""),
	)
	if err != nil {
		t.Fatal(err)
	}
	readAdapter := newTestReadAdapter(t, staticHTMLDoer([]byte(readDetailHighCourtHTML)))
	result, err := readAdapter.Read(
		context.Background(),
		mustReadRequestFromRef(t, page.Items()[1].Ref()),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Ref().Key().ResourceID(); got != page.Items()[1].Ref().Key().ResourceID() {
		t.Fatalf("resourceId = %q", got)
	}
}

func newTestReadAdapter(t *testing.T, doer httpDoer) *JudicialDecisionReadAdapter {
	t.Helper()
	adapter, err := newJudicialDecisionReadAdapter(readAdapterDependencies{
		doer:         doer,
		now:          fixedNow,
		gate:         make(chan struct{}, 1),
		parseTimeout: readParseTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	return adapter
}

func assertReadSourceError(t *testing.T, err error, code model.SourceErrorCode) {
	t.Helper()
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("情報源エラーではない: %T %v", err, err)
	}
	if sourceError.Code() != code ||
		sourceError.ProviderID() != providerID ||
		sourceError.SourceID() != sourceID ||
		sourceError.CapabilityID() != judicialdecisionread.CapabilityID ||
		sourceError.Operation() != string(operationRead) {
		t.Errorf("情報源エラー = %#v", sourceError)
	}
}

func mustReadRequest(t *testing.T, provider string, source string, resourceID string) judicialdecisionread.Request {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     source,
		ResourceType: "judicial-decision",
		ResourceID:   resourceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: provider,
		Key:        key,
	})
	if err != nil {
		t.Fatal(err)
	}
	request, err := judicialdecisionread.NewRequest(judicialdecisionread.RequestValues{Ref: ref})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func mustReadRequestFromRef(t *testing.T, ref model.SourceResourceRef) judicialdecisionread.Request {
	t.Helper()
	request, err := judicialdecisionread.NewRequest(judicialdecisionread.RequestValues{Ref: ref})
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func minimalReadDetailHTML(categoryLabel string, extra string) string {
	return fmt.Sprintf(`<!doctype html><html><head><title>裁判例結果詳細</title></head><body>
<main id="main-contents">
  <div class="module-search-title"><h4>%s</h4></div>
  <div class="module-sub-page-parts-table">
    <dl><dt>事件番号</dt><dd><p>令和7(行ツ)1</p></dd></dl>
    <dl><dt>裁判年月日</dt><dd><p>令和7年10月16日</p></dd></dl>
    <dl><dt>裁判所名</dt><dd><p>最高裁判所</p></dd></dl>
    %s
  </div>
</main></body></html>`, categoryLabel, extra)
}

func readDL(label string, value string) string {
	return fmt.Sprintf("<dl><dt>%s</dt><dd><p>%s</p></dd></dl>", label, value)
}

const readDetailHighCourtHTML = `<!doctype html><html><head><title>裁判例結果詳細 | 最高裁判所</title></head><body>
<main id="main-contents">
  <div class="module-search-title"><h4>高等裁判所</h4></div>
  <div class="module-sub-page-parts-table">
    <dl><dt>事件番号</dt><dd><p>令和7(う)1040</p></dd></dl>
    <dl><dt>事件名</dt><dd><p>不同意性交等被告事件</p></dd></dl>
    <dl><dt>裁判年月日</dt><dd><p>令和7年10月16日</p></dd></dl>
    <dl><dt>裁判所名・部</dt><dd><p>東京高等裁判所  第６刑事部</p></dd></dl>
    <dl><dt>結果</dt><dd><p>棄却</p></dd></dl>
    <dl><dt>高裁判例集登載巻・号・頁</dt><dd><p>第77巻2号23頁</p></dd></dl>
  </div>
  <div class="module-sub-page-parts-table">
    <dl><dt>原審裁判所名</dt><dd><p>東京地方裁判所<br>立川支部</p></dd></dl>
    <dl><dt>原審事件番号</dt><dd><p>令和7(わ)64</p></dd></dl>
  </div>
  <div class="module-sub-page-parts-table">
    <dl><dt>判示事項</dt><dd><p style="white-space: pre-line;">口腔性交等についての刑法１７７条と憲法１４条</p></dd></dl>
    <dl><dt>裁判要旨</dt><dd><p style="white-space: pre-line;">口腔性交等につき、不同意わいせつ罪の場合と区別し、不同意性交等罪として加重処罰することとされた刑法１７７条の規定は、憲法１４条に違反しない。</p></dd></dl>
    <dl><dt>全文</dt><dd><p><a href="./../../../assets/hanrei/hanrei-pdf-95878.pdf">全文</a></p></dd></dl>
  </div>
</main></body></html>`

const readDetailLowerCourtHTML = `<!doctype html><html><head><title>裁判例結果詳細 | 最高裁判所</title></head><body>
<main id="main-contents">
  <div class="module-search-title"><h4>下級裁判所(速報)</h4></div>
  <div class="module-sub-page-parts-table">
    <dl><dt>事件番号</dt><dd><p>令和7(わ)11</p></dd></dl>
    <dl><dt>事件名</dt><dd><p>住居侵入被告事件</p></dd></dl>
    <dl><dt>裁判年月日</dt><dd><p>令和7年9月1日</p></dd></dl>
    <dl><dt>裁判所名・部</dt><dd><p>東京地方裁判所 立川支部 第１刑事部</p></dd></dl>
    <dl><dt>判示事項の要旨</dt><dd><p>判示事項の要旨です。</p></dd></dl>
    <dl><dt>全文</dt><dd><p><a href="./../../../assets/hanrei/hanrei-pdf-95877.pdf">全文</a><a href="./../../../assets/hanrei/additional-95877.pdf">添付文書1</a></p></dd></dl>
  </div>
</main></body></html>`

const readDetailIPHTML = `<!doctype html><html><head><title>裁判例結果詳細 | 最高裁判所</title></head><body>
<main id="main-contents">
  <div class="module-search-title"><h4>知的財産高等裁判所</h4></div>
  <div class="module-sub-page-parts-table">
    <dl><dt>事件番号</dt><dd><p>令和7(ネ)1</p></dd></dl>
    <dl><dt>事件名</dt><dd><p>特許権侵害差止請求控訴事件</p></dd></dl>
    <dl><dt>事件種別</dt><dd><p>民事</p></dd></dl>
    <dl><dt>事件種類</dt><dd><p>特許権</p></dd></dl>
    <dl><dt>発明等の名称等</dt><dd><p>発明A</p></dd></dl>
    <dl><dt>判決結果</dt><dd><p>棄却</p></dd></dl>
    <dl><dt>裁判年月日</dt><dd><p>令和7年10月16日</p></dd></dl>
    <dl><dt>裁判所名</dt><dd><p>知的財産高等裁判所</p></dd></dl>
    <dl><dt>部名</dt><dd><p>第４部</p></dd></dl>
    <dl><dt>要旨</dt><dd><p>争点の要約</p></dd></dl>
    <dl><dt>全文</dt><dd><p><a href="./../../../assets/hanrei/hanrei-pdf-95878.pdf">全文</a></p></dd></dl>
  </div>
</main></body></html>`

const readDetailMixedSummaryHTML = `<!doctype html><html><head><title>裁判例結果詳細 | 最高裁判所</title></head><body>
<main id="main-contents">
  <div class="module-search-title"><h4>知的財産高等裁判所</h4></div>
  <div class="module-sub-page-parts-table">
    <dl><dt>事件番号</dt><dd><p>令和7(ネ)2</p></dd></dl>
    <dl><dt>事件名</dt><dd><p>商標権侵害差止請求控訴事件</p></dd></dl>
    <dl><dt>裁判年月日</dt><dd><p>令和7年10月17日</p></dd></dl>
    <dl><dt>裁判所名</dt><dd><p>知的財産高等裁判所</p></dd></dl>
    <dl><dt>要旨</dt><dd><p>テキスト要旨<a href="./../../../assets/hanrei/summary-95879.pdf">要旨</a></p></dd></dl>
    <dl><dt>全文</dt><dd><p><a href="./../../../assets/hanrei/hanrei-pdf-95879.pdf">全文</a></p></dd></dl>
  </div>
</main></body></html>`
