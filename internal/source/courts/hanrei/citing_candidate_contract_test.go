package hanrei

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestCitingCandidateSkipsIdenticalReporterCitation(t *testing.T) {
	t.Parallel()

	target := targetWithReporter(t, "令和6(受)123")
	doer := &recordingHTMLDoer{responses: map[string][]byte{
		"令和6(受)123": mustReadHanreiTestdata(t, "search_empty.html"),
	}}
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	result, err := adapter.Search(context.Background(), citingCandidateRequest(t, target, 5))
	if err != nil {
		t.Fatal(err)
	}
	if len(doer.queries) != 1 || len(result.Coverage().Attempts()) != 1 {
		t.Fatalf("SOT-IF-073: 同一検索値の calls/attempts = %d/%d", len(doer.queries), len(result.Coverage().Attempts()))
	}
}

func TestCitingCandidateKeepsDOMOrderAndDuplicateEvidence(t *testing.T) {
	t.Parallel()

	body := mustReadHanreiTestdata(t, "search_duplicates.html")
	doer := &recordingHTMLDoer{responses: map[string][]byte{
		"令和6(受)123":  body,
		"民集第80巻1号1頁": body,
	}}
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	result, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 10),
	)
	if err != nil {
		t.Fatal(err)
	}
	items := result.Items()
	if len(items) != 1 || items[0].Decision().Data().DecisionID() != "00202" {
		t.Fatalf("SOT-IF-073: 自己除外・DOM 順・ref 重複排除後の items = %#v", items)
	}
	if len(items[0].Evidence()) != 2 {
		t.Fatalf("SOT-IF-073: 検索順の evidence 件数 = %d", len(items[0].Evidence()))
	}
	coverage := result.Coverage()
	if coverage.ObservedItemCount() != 6 || coverage.DedupedCandidateCount() != 1 || coverage.Truncated() {
		t.Fatalf("SOT-IF-069: coverage = %#v", coverage)
	}
}

func TestCitingCandidateAppliesLimitAfterDedupAndPreservesOfficialTruncation(t *testing.T) {
	t.Parallel()

	fixture := mustReadHanreiTestdata(t, "search_all_categories.html")
	for _, limit := range []int{1, 10} {
		limit := limit
		t.Run("limit", func(t *testing.T) {
			doer := &recordingHTMLDoer{responses: map[string][]byte{
				"令和6(受)123":  fixture,
				"民集第80巻1号1頁": fixture,
			}}
			adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
			result, err := adapter.Search(
				context.Background(),
				citingCandidateRequest(t, newSearchTargetResource(t), limit),
			)
			if err != nil {
				t.Fatal(err)
			}
			wantItems := limit
			if wantItems > 7 {
				wantItems = 7
			}
			if len(result.Items()) != wantItems ||
				result.Coverage().DedupedCandidateCount() != 7 ||
				!result.Coverage().Truncated() {
				t.Fatalf("SOT-IF-069/073: limit=%d result=%#v", limit, result)
			}
		})
	}
}

func TestCitingCandidateReturnsCompleteEmptyResult(t *testing.T) {
	t.Parallel()

	empty := mustReadHanreiTestdata(t, "search_empty.html")
	doer := &recordingHTMLDoer{responses: map[string][]byte{
		"令和6(受)123":  empty,
		"民集第80巻1号1頁": empty,
	}}
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	result, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 5),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status() != judicialcitingcandidatesearch.SearchStatusComplete ||
		result.Items() == nil || len(result.Items()) != 0 ||
		result.Coverage().ObservedItemCount() != 0 {
		t.Fatalf("SOT-IF-069: 空結果 = %#v", result)
	}
}

func TestCitingCandidateRunsSecondSearchAfterFirstFailure(t *testing.T) {
	t.Parallel()

	doer := &recordingHTMLDoer{
		responses: map[string][]byte{
			"民集第80巻1号1頁": mustReadHanreiTestdata(t, "search_duplicates.html"),
		},
		statusByQuery: map[string]int{"令和6(受)123": http.StatusServiceUnavailable},
	}
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	result, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 5),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status() != judicialcitingcandidatesearch.SearchStatusPartial ||
		len(doer.queries) != 2 || len(result.Items()) != 1 || len(result.Issues()) != 1 {
		t.Fatalf("SOT-IF-069: first failure 後の部分結果 = %#v, calls=%d", result, len(doer.queries))
	}
}

func TestCitingCandidateReturnsErrorWhenAllSearchesFailAndReleasesGate(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	doer := &recordingHTMLDoer{statusByQuery: map[string]int{
		"令和6(受)123":  http.StatusServiceUnavailable,
		"民集第80巻1号1頁": http.StatusServiceUnavailable,
	}}
	adapter := newCitingCandidateTestAdapter(t, doer, gate)
	_, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 5),
	)
	assertCitingCandidateSourceError(t, err, model.SourceErrorCodeSourceUnavailable)
	if len(doer.queries) != 2 || len(gate) != 0 {
		t.Fatalf("SOT-IF-069/072: calls=%d gate=%d", len(doer.queries), len(gate))
	}
}

func TestCitingCandidateRejectsInvalidInputBeforeCall(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	doer := roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("呼び出されてはなりません")
	})
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	if _, err := adapter.Search(context.Background(), judicialcitingcandidatesearch.Request{}); err == nil {
		t.Fatal("SOT-IF-069: zero request を受理した")
	}
	if calls.Load() != 0 {
		t.Fatalf("SOT-IF-069: 不正入力の外部 calls = %d", calls.Load())
	}
}

func TestCitingCandidateCancellationAndBusyDoNotLeakGate(t *testing.T) {
	t.Parallel()

	t.Run("canceled before call", func(t *testing.T) {
		var calls atomic.Int32
		doer := roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, errors.New("呼び出されてはなりません")
		})
		gate := make(chan struct{}, 1)
		adapter := newCitingCandidateTestAdapter(t, doer, gate)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := adapter.Search(ctx, citingCandidateRequest(t, newSearchTargetResource(t), 5))
		if !errors.Is(err, context.Canceled) || calls.Load() != 0 || len(gate) != 0 {
			t.Fatalf("SOT-IF-069/072: canceled error=%v calls=%d gate=%d", err, calls.Load(), len(gate))
		}
	})

	t.Run("busy", func(t *testing.T) {
		gate := make(chan struct{}, 1)
		gate <- struct{}{}
		var calls atomic.Int32
		adapter := newCitingCandidateTestAdapter(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, nil
		}), gate)
		_, err := adapter.Search(
			context.Background(),
			citingCandidateRequest(t, newSearchTargetResource(t), 5),
		)
		assertCitingCandidateSourceError(t, err, model.SourceErrorCodeSourceBusy)
		if calls.Load() != 0 || len(gate) != 1 {
			t.Fatalf("SOT-IF-072: busy calls=%d gate=%d", calls.Load(), len(gate))
		}
		<-gate
	})

	t.Run("canceled during first response", func(t *testing.T) {
		gate := make(chan struct{}, 1)
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		doer := roundTripFunc(func(request *http.Request) (*http.Response, error) {
			calls++
			cancel()
			return htmlResponse(request, mustReadHanreiTestdata(t, "search_empty.html")), nil
		})
		adapter := newCitingCandidateTestAdapter(t, doer, gate)
		_, err := adapter.Search(ctx, citingCandidateRequest(t, newSearchTargetResource(t), 5))
		if !errors.Is(err, context.Canceled) || calls != 1 || len(gate) != 0 {
			t.Fatalf("SOT-IF-069/072: mid-cancel error=%v calls=%d gate=%d", err, calls, len(gate))
		}
	})
}

func TestCitingCandidateUsesCumulativeEncodedByteBudgetAcrossSearches(t *testing.T) {
	first := paddedEmptySearchHTML(
		t,
		maximumCitingCandidateResponseBytes-len(emptySearchHTML())-8,
	)
	second := paddedEmptySearchHTML(t, 64)
	doer := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch searchQueryFromURL(request.URL.String()) {
		case "令和6(受)123":
			return htmlResponse(request, first), nil
		case "民集第80巻1号1頁":
			return htmlResponse(request, second), nil
		default:
			t.Fatalf("unexpected query: %s", request.URL.Redacted())
			return nil, nil
		}
	})
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	result, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 5),
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleBudgetIssue(t, result, model.ErrorCodeSourceResponseTooLarge)
}

func TestCitingCandidateUsesCumulativeDecodedByteBudgetAcrossSearches(t *testing.T) {
	first := gzipBytes(t, paddedEmptySearchHTML(
		t,
		maximumCitingCandidateDecompressedBytes-len(emptySearchHTML())-8,
	))
	second := gzipBytes(t, paddedEmptySearchHTML(t, 64))
	doer := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		var body []byte
		switch searchQueryFromURL(request.URL.String()) {
		case "令和6(受)123":
			body = first
		case "民集第80巻1号1頁":
			body = second
		default:
			t.Fatalf("unexpected query: %s", request.URL.Redacted())
			return nil, nil
		}
		response := htmlResponse(request, body)
		response.Header.Set("Content-Encoding", "gzip")
		return response, nil
	})
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	result, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 5),
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleBudgetIssue(t, result, model.ErrorCodeSourceResponseTooLarge)
}

func TestCitingCandidateUsesCumulativeNodeBudgetAcrossSearches(t *testing.T) {
	first := commentHeavyEmptySearchHTML(maximumCitingCandidateNodes - 10)
	second := mustReadHanreiTestdata(t, "search_empty.html")
	doer := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch searchQueryFromURL(request.URL.String()) {
		case "令和6(受)123":
			return htmlResponse(request, first), nil
		case "民集第80巻1号1頁":
			return htmlResponse(request, second), nil
		default:
			t.Fatalf("unexpected query: %s", request.URL.Redacted())
			return nil, nil
		}
	})
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	result, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 5),
	)
	if err != nil {
		t.Fatal(err)
	}
	assertSingleBudgetIssue(t, result, model.ErrorCodeSourceResponseTooLarge)
}

func TestCitingCandidateUsesOneGateForTwoSearchesAndOnlySearchResource(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	empty := mustReadHanreiTestdata(t, "search_empty.html")
	var calls int
	doer := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if len(gate) != 1 || request.Method != http.MethodGet ||
			request.URL.Path != "/hanrei/search1/index.html" ||
			len(request.URL.Query()) != 1 || len(request.URL.Query()["query1"]) != 1 {
			t.Fatalf("SOT-IF-072/073: request=%s gate=%d", request.URL.Redacted(), len(gate))
		}
		return htmlResponse(request, empty), nil
	})
	adapter := newCitingCandidateTestAdapter(t, doer, gate)
	if _, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 5),
	); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(gate) != 0 {
		t.Fatalf("SOT-IF-072/073: calls=%d gate=%d", calls, len(gate))
	}
}

func TestCitingCandidateDoesNotExposeQueriesOrRawHTML(t *testing.T) {
	t.Parallel()

	firstQuery := "令和6(受)123"
	secondQuery := "民集第80巻1号1頁"
	empty := mustReadHanreiTestdata(t, "search_empty.html")
	doer := &recordingHTMLDoer{responses: map[string][]byte{
		firstQuery:  empty,
		secondQuery: empty,
	}}
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	result, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 5),
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{firstQuery, secondQuery, "query1", "該当する裁判例がありませんでした"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("SOT-IF-069/072: result に検索内容が露出しました: %q", secret)
		}
	}
}

func TestCitingCandidateDoesNotExposeSourceAuthFailed(t *testing.T) {
	t.Parallel()

	doer := &recordingHTMLDoer{statusByQuery: map[string]int{
		"令和6(受)123":  http.StatusUnauthorized,
		"民集第80巻1号1頁": http.StatusForbidden,
	}}
	adapter := newCitingCandidateTestAdapter(t, doer, make(chan struct{}, 1))
	_, err := adapter.Search(
		context.Background(),
		citingCandidateRequest(t, newSearchTargetResource(t), 5),
	)
	assertCitingCandidateSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
}

func TestCitingCandidateBudgetHelpersCountFailedWork(t *testing.T) {
	t.Parallel()

	doer := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := htmlResponse(request, []byte("four"))
		response.ContentLength = -1
		return response, nil
	})
	_, consumed, err := fetchCitingCandidateResponse(
		context.Background(),
		doer,
		fixedNow,
		"予算",
		3,
	)
	assertSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	if consumed != 4 {
		t.Fatalf("SOT-IF-072: encoded consumed = %d", consumed)
	}

	_, consumed, err = decodeCitingCandidateResponse(
		context.Background(),
		context.Background(),
		fetchedSearchResponse{
			encodedBody: []byte("four"),
			contentType: "text/html; charset=utf-8",
		},
		3,
	)
	assertSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	if consumed != 4 {
		t.Fatalf("SOT-IF-072: decoded consumed = %d", consumed)
	}

	_, nodes, err := parseSearchResponseWithBudget(
		context.Background(),
		[]byte("x"),
		1024,
		1,
		64,
	)
	assertSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
	if nodes <= 1 {
		t.Fatalf("SOT-IF-072: failed parse nodes = %d", nodes)
	}
}

func TestFetchCitingCandidateRejectsExhaustedBudgetBeforeCall(t *testing.T) {
	t.Parallel()

	for _, remainingBytes := range []int{0, -1} {
		remainingBytes := remainingBytes
		t.Run("exhausted", func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int32
			doer := roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("呼び出されてはなりません")
			})
			_, consumed, err := fetchCitingCandidateResponse(
				context.Background(),
				doer,
				fixedNow,
				"外部へ送信してはならない検索語",
				remainingBytes,
			)
			assertSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
			if calls.Load() != 0 || consumed != 0 {
				t.Fatalf("SOT-IF-072: exhausted budget calls=%d consumed=%d", calls.Load(), consumed)
			}
		})
	}
}

func newCitingCandidateTestAdapter(
	t *testing.T,
	doer httpDoer,
	gate chan struct{},
) *JudicialCitingCandidateSearchAdapter {
	t.Helper()
	adapter, err := newJudicialCitingCandidateSearchAdapter(searchAdapterDependencies{
		doer:         doer,
		now:          fixedNow,
		gate:         gate,
		parseTimeout: citingCandidateParseTimeout,
	})
	if err != nil {
		t.Fatalf("candidate adapter: %v", err)
	}
	return adapter
}

func citingCandidateRequest(
	t *testing.T,
	target model.SourcedResource[model.JudicialDecisionDetails],
	limit int,
) judicialcitingcandidatesearch.Request {
	t.Helper()
	request, err := judicialcitingcandidatesearch.NewRequest(
		judicialcitingcandidatesearch.RequestValues{Target: target, Limit: &limit},
	)
	if err != nil {
		t.Fatalf("candidate request: %v", err)
	}
	return request
}

func targetWithReporter(
	t *testing.T,
	reporterCitation string,
) model.SourcedResource[model.JudicialDecisionDetails] {
	t.Helper()
	target := newSearchTargetResource(t)
	details, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary:          target.Data().Summary(),
		ReporterCitation: &reporterCitation,
	})
	if err != nil {
		t.Fatal(err)
	}
	resource, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionDetails]{
			Ref:        target.Ref(),
			Provenance: target.Provenance(),
			Data:       details,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return resource
}

func assertCitingCandidateSourceError(
	t *testing.T,
	err error,
	code model.SourceErrorCode,
) {
	t.Helper()
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("SourceError ではありません: %T %v", err, err)
	}
	if sourceError.Code() != code ||
		sourceError.CapabilityID() != judicialcitingcandidatesearch.CapabilityID {
		t.Fatalf("candidate SourceError = code=%q capability=%q", sourceError.Code(), sourceError.CapabilityID())
	}
}

func assertSingleBudgetIssue(
	t *testing.T,
	result judicialcitingcandidatesearch.Result,
	code model.ErrorCode,
) {
	t.Helper()
	if result.Status() != judicialcitingcandidatesearch.ResultStatusPartial ||
		len(result.Issues()) != 1 ||
		result.Issues()[0].SearchKind() != judicialcitingcandidatesearch.SearchKindReporterCitation {
		t.Fatalf("partial budget result = %#v", result)
	}
	if result.Issues()[0].ErrorResult().Code() != code {
		t.Fatalf("issue code = %q", result.Issues()[0].ErrorResult().Code())
	}
}

func emptySearchHTML() string {
	return "<!doctype html><html><head><title>裁判例検索</title></head>" +
		"<body><p id=\"searched\">該当する裁判例がありませんでした。</p></body></html>"
}

func paddedEmptySearchHTML(t *testing.T, padding int) []byte {
	t.Helper()
	if padding < 0 {
		t.Fatalf("padding must be non-negative: %d", padding)
	}
	body := "<!doctype html><html><head><title>裁判例検索</title></head><body>" +
		strings.Repeat(" ", padding) +
		"<p id=\"searched\">該当する裁判例がありませんでした。</p></body></html>"
	return []byte(body)
}

func commentHeavyEmptySearchHTML(commentCount int) []byte {
	return []byte("<!doctype html><html><head><title>裁判例検索</title></head><body>" +
		strings.Repeat("<!--x-->", commentCount) +
		"<p id=\"searched\">該当する裁判例がありませんでした。</p></body></html>")
}
