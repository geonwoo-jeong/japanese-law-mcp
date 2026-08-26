package hanrei

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestJudicialCitingCandidateSearchAdapterSearch(t *testing.T) {
	t.Parallel()

	doer := &recordingHTMLDoer{
		responses: map[string][]byte{
			"令和6(受)123":  mustReadHanreiTestdata(t, "search_duplicates.html"),
			"民集第80巻1号1頁": mustReadHanreiTestdata(t, "search_all_categories.html"),
		},
	}
	adapter, err := newJudicialCitingCandidateSearchAdapter(searchAdapterDependencies{
		doer:         doer,
		now:          func() time.Time { return time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC) },
		gate:         make(chan struct{}, 1),
		parseTimeout: 4 * time.Second,
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	limit := 3
	request, err := judicialcitingcandidatesearch.NewRequest(
		judicialcitingcandidatesearch.RequestValues{
			Target: newSearchTargetResource(t),
			Limit:  &limit,
		},
	)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	result, err := adapter.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-069/073: Search() のエラー = %v", err)
	}
	if result.Status() != judicialcitingcandidatesearch.SearchStatusComplete {
		t.Fatalf("status = %q", result.Status())
	}
	if len(doer.queries) != 2 ||
		doer.queries[0] != "令和6(受)123" ||
		doer.queries[1] != "民集第80巻1号1頁" {
		t.Fatalf("queries = %#v", doer.queries)
	}
	if len(result.Items()) != 3 {
		t.Fatalf("items = %d", len(result.Items()))
	}
	if result.Items()[0].Decision().Data().DecisionID() != "00202" {
		t.Fatalf("first candidate = %s", result.Items()[0].Decision().Data().DecisionID())
	}
	if len(result.Items()[1].Evidence()) != 1 || len(result.Items()[2].Evidence()) != 1 {
		t.Fatalf("evidence count = %d, %d", len(result.Items()[1].Evidence()), len(result.Items()[2].Evidence()))
	}
	if !result.Coverage().Truncated() {
		t.Fatal("coverage.truncated が false でした")
	}
}

func TestJudicialCitingCandidateSearchAdapterKeepsPartialResult(t *testing.T) {
	t.Parallel()

	doer := &recordingHTMLDoer{
		responses: map[string][]byte{
			"令和6(受)123": mustReadHanreiTestdata(t, "search_duplicates.html"),
		},
		statusByQuery: map[string]int{
			"民集第80巻1号1頁": http.StatusServiceUnavailable,
		},
	}
	adapter, err := newJudicialCitingCandidateSearchAdapter(searchAdapterDependencies{
		doer:         doer,
		now:          time.Now,
		gate:         make(chan struct{}, 1),
		parseTimeout: 4 * time.Second,
	})
	if err != nil {
		t.Fatalf("adapter: %v", err)
	}
	request, err := judicialcitingcandidatesearch.NewRequest(
		judicialcitingcandidatesearch.RequestValues{Target: newSearchTargetResource(t)},
	)
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	result, err := adapter.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("partial result expected, err = %v", err)
	}
	if result.Status() != judicialcitingcandidatesearch.SearchStatusPartial {
		t.Fatalf("status = %q", result.Status())
	}
	if len(result.Issues()) != 1 ||
		result.Issues()[0].SearchKind() != judicialcitingcandidatesearch.SearchKindReporterCitation {
		t.Fatalf("issues = %#v", result.Issues())
	}
}

type recordingHTMLDoer struct {
	queries       []string
	responses     map[string][]byte
	statusByQuery map[string]int
}

func (d *recordingHTMLDoer) Do(request *http.Request) (*http.Response, error) {
	query := searchQueryFromURL(request.URL.String())
	d.queries = append(d.queries, query)
	status := http.StatusOK
	if d.statusByQuery != nil {
		if candidate, exists := d.statusByQuery[query]; exists {
			status = candidate
		}
	}
	body := d.responses[query]
	if body == nil {
		body = []byte(`<!doctype html><html><head><title>裁判例検索</title></head>` +
			`<body><p id="searched">該当する裁判例がありませんでした。</p></body></html>`)
	}
	response := &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       http.NoBody,
		Request:    request,
	}
	response.Header.Set("Content-Type", "text/html; charset=utf-8")
	if status == http.StatusOK {
		response.Body = newReadCloser(strings.NewReader(string(body)))
	}
	return response, nil
}

func mustReadHanreiTestdata(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name)
	body, err := os.ReadFile(path) //nolint:gosec // SOT-IF-072: testdata 配下の固定 fixture だけを読む。
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return body
}

func newSearchTargetResource(
	t *testing.T,
) model.SourcedResource[model.JudicialDecisionDetails] {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "courts-hanrei",
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	date, err := model.NewDate("2026-08-26")
	if err != nil {
		t.Fatalf("date: %v", err)
	}
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          "00101",
		PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
		SourceCategoryLabel: "最高裁判例",
		CaseNumber:          "令和6(受)123",
		DecisionDate:        date,
		CourtName:           "最高裁判所",
		DetailURL:           "https://www.courts.go.jp/hanrei/00101/detail2/index.html",
		Documents:           []model.JudicialDocumentLink{},
		Source:              source,
	})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	reporterCitation := "民集第80巻1号1頁"
	details, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary:          summary,
		ReporterCitation: &reporterCitation,
	})
	if err != nil {
		t.Fatalf("details: %v", err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "00101/detail2",
	})
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("ref: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            "https://www.courts.go.jp/hanrei/00101/detail2/index.html",
		RetrievedAt:    time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-045",
	})
	if err != nil {
		t.Fatalf("provenance: %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionDetails]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       details,
	})
	if err != nil {
		t.Fatalf("resource: %v", err)
	}
	return resource
}

type readCloser struct {
	reader *strings.Reader
}

func newReadCloser(reader *strings.Reader) *readCloser {
	return &readCloser{reader: reader}
}

func (r *readCloser) Read(p []byte) (int, error) { return r.reader.Read(p) }
func (*readCloser) Close() error                 { return nil }

func TestSearchQueryFromURL(t *testing.T) {
	t.Parallel()

	raw := "https://www.courts.go.jp/hanrei/search1/index.html?query1=%E6%B0%91%E9%9B%86"
	if query := searchQueryFromURL(raw); query != "民集" {
		t.Fatalf("query = %q", query)
	}
	invalid := &url.URL{Scheme: ":", Opaque: "bad"}
	if query := searchQueryFromURL(invalid.String()); query != "" {
		t.Fatalf("invalid query = %q", query)
	}
}

func searchQueryFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("query1")
}
