package lawv2

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/continuation"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLawContentAdapterMapsResultAndContinues(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC)
	lastPage := `{
		"total_count":3,
		"sentence_count":1,
		"next_offset":null,
		"items":[{
			"law_info":{"law_id":"129AC0000000089"},
			"revision_info":{
				"law_revision_id":"129AC0000000089_20260401_000000000000000",
				"law_title":"民法"
			},
			"sentences":[{
				"position":"MainProvision/Article[3]",
				"text":"第三の一致"
			}]
		}]
	}`
	var mutex sync.Mutex
	var requests []*http.Request
	call := 0
	adapter := newTestLawContentAdapter(
		t,
		now,
		make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			mutex.Lock()
			defer mutex.Unlock()
			requests = append(requests, request.Clone(request.Context()))
			body := string(fixture(t, "fixtures/law-content-normal.json"))
			if call == 1 {
				body = lastPage
			}
			call++
			return response(
				http.StatusOK,
				body,
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
	)
	limit := 2
	initial := mustLawContentSearchRequest(t, lawcontentsearch.RequestValues{
		AllTerms: []string{"地方"},
		AnyTerms: []string{"自治", "行政"},
		Limit:    &limit,
	})

	page, err := adapter.Search(context.Background(), initial)
	if err != nil {
		t.Fatalf("SOT-IF-023/028: 初回 Search() のエラー = %v", err)
	}
	if len(page.Items()) != 2 {
		t.Fatalf("SOT-IF-023/028: 初回 items = %d", len(page.Items()))
	}
	token, exists := page.Page().NextToken()
	if !exists {
		t.Fatal("SOT-IF-016/028: nextToken がない")
	}
	resume := mustLawContentSearchRequest(t, lawcontentsearch.RequestValues{
		AllTerms:          []string{"地方"},
		AnyTerms:          []string{"自治", "行政"},
		Limit:             &limit,
		ContinuationToken: token,
	})
	page, err = adapter.Search(context.Background(), resume)
	if err != nil {
		t.Fatalf("SOT-IF-016/028: 継続 Search() のエラー = %v", err)
	}
	if len(page.Items()) != 1 {
		t.Fatalf("SOT-IF-023/028: 継続 items = %d", len(page.Items()))
	}
	if _, exists := page.Page().NextToken(); exists {
		t.Fatal("SOT-IF-016/028: 最終ページに nextToken がある")
	}

	mutex.Lock()
	defer mutex.Unlock()
	if len(requests) != 2 {
		t.Fatalf("SOT-IF-028: HTTP requests = %d", len(requests))
	}
	for index, request := range requests {
		if request.URL.Query().Get("asof") != "2026-07-26" {
			t.Errorf("SOT-IF-028: requests[%d].asof = %q", index, request.URL.Query().Get("asof"))
		}
	}
	if requests[0].URL.Query().Get("offset") != "0" ||
		requests[1].URL.Query().Get("offset") != "2" {
		t.Fatalf(
			"SOT-IF-016/028: offsets = %q, %q",
			requests[0].URL.Query().Get("offset"),
			requests[1].URL.Query().Get("offset"),
		)
	}
}

func TestLawContentAdapterRejectsContinuationConditionMismatchBeforeHTTP(
	t *testing.T,
) {
	t.Parallel()

	attempts := 0
	limit := 2
	adapter := newTestLawContentAdapter(
		t,
		time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
		make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return response(
				http.StatusOK,
				string(fixture(t, "fixtures/law-content-normal.json")),
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
	)
	initial := mustLawContentSearchRequest(t, lawcontentsearch.RequestValues{
		AllTerms: []string{"地方"},
		Limit:    &limit,
	})
	page, err := adapter.Search(context.Background(), initial)
	if err != nil {
		t.Fatalf("SOT-IF-016/028: 初回 Search() のエラー = %v", err)
	}
	token, _ := page.Page().NextToken()
	mismatch := mustLawContentSearchRequest(t, lawcontentsearch.RequestValues{
		AllTerms:          []string{"民法"},
		Limit:             &limit,
		ContinuationToken: token,
	})
	_, err = adapter.Search(context.Background(), mismatch)
	if !errors.Is(err, continuation.ErrInvalidToken) {
		t.Fatalf("SOT-IF-016: error = %v、期待値 = ErrInvalidToken", err)
	}
	macStart := strings.LastIndexByte(token, '.') + 1
	replacement := byte('A')
	if token[macStart] == replacement {
		replacement = 'B'
	}
	tampered := mustLawContentSearchRequest(t, lawcontentsearch.RequestValues{
		AllTerms:          []string{"地方"},
		Limit:             &limit,
		ContinuationToken: token[:macStart] + string(replacement) + token[macStart+1:],
	})
	_, err = adapter.Search(context.Background(), tampered)
	if !errors.Is(err, continuation.ErrInvalidToken) {
		t.Fatalf("SOT-IF-016: 改変 token の error = %v", err)
	}
	if attempts != 1 {
		t.Fatalf("SOT-IF-016: HTTP attempts = %d", attempts)
	}
}

func TestLawContentAdapterRejectsOldDateBusyAndCancellationBeforeHTTP(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name    string
		gate    chan struct{}
		asOf    *model.Date
		context func() context.Context
		code    model.SourceErrorCode
		raw     error
	}{
		{
			name: "対象期間外",
			gate: make(chan struct{}, 1),
			asOf: func() *model.Date {
				value := mustDate("2017-03-31")
				return &value
			}(),
			context: context.Background,
			code:    model.SourceErrorCodeUnsupportedQuery,
		},
		{
			name: "共通 gate が使用中",
			gate: func() chan struct{} {
				value := make(chan struct{}, 1)
				value <- struct{}{}
				return value
			}(),
			context: context.Background,
			code:    model.SourceErrorCodeSourceBusy,
		},
		{
			name: "呼出し前の cancellation",
			gate: make(chan struct{}, 1),
			context: func() context.Context {
				value, cancel := context.WithCancel(context.Background())
				cancel()
				return value
			},
			raw: context.Canceled,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			attempts := 0
			adapter := newTestLawContentAdapter(
				t,
				time.Now(),
				test.gate,
				doerFunc(func(*http.Request) (*http.Response, error) {
					attempts++
					return nil, errors.New("呼び出してはならない HTTP")
				}),
			)
			request := mustLawContentSearchRequest(t, lawcontentsearch.RequestValues{
				AllTerms: []string{"地方"},
				AsOf:     test.asOf,
			})
			_, err := adapter.Search(test.context(), request)
			if test.raw != nil {
				if !errors.Is(err, test.raw) {
					t.Fatalf("SOT-ENG-010: error = %v", err)
				}
			} else {
				assertLawContentSourceError(t, err, test.code)
			}
			if attempts != 0 {
				t.Fatalf("SOT-IF-017/028: HTTP attempts = %d", attempts)
			}
		})
	}
}

func TestLawContentAdapterAcceptsEmptyResult(t *testing.T) {
	t.Parallel()

	adapter := newTestLawContentAdapter(
		t,
		time.Now(),
		make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			return response(
				http.StatusOK,
				string(fixture(t, "fixtures/law-content-empty.json")),
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
	)
	request := mustLawContentSearchRequest(t, lawcontentsearch.RequestValues{
		AllTerms: []string{"該当なし"},
	})
	page, err := adapter.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-023/028: 空結果のエラー = %v", err)
	}
	if page.Items() == nil ||
		len(page.Items()) != 0 ||
		page.Page().ReturnedCount() != 0 {
		t.Fatalf("SOT-IF-023: 空結果 = %#v", page)
	}
}

func newTestLawContentAdapter(
	t *testing.T,
	now time.Time,
	gate chan struct{},
	doer httpDoer,
) *LawContentSearchAdapter {
	t.Helper()

	manager, err := continuation.NewManager()
	if err != nil {
		t.Fatalf("continuation.NewManager() のエラー = %v", err)
	}
	client := mustTestClient(t, clientDependencies{
		doer: doer,
		now:  func() time.Time { return now },
		sleep: func(context.Context, time.Duration) error {
			return nil
		},
	})
	adapter, err := newLawContentSearchAdapter(
		manager,
		lawContentAdapterDependencies{
			client: client,
			now:    func() time.Time { return now },
			gate:   gate,
		},
	)
	if err != nil {
		t.Fatalf("newLawContentSearchAdapter() のエラー = %v", err)
	}
	return adapter
}

func mustLawContentSearchRequest(
	t *testing.T,
	values lawcontentsearch.RequestValues,
) lawcontentsearch.Request {
	t.Helper()

	request, err := lawcontentsearch.NewRequest(values)
	if err != nil {
		t.Fatalf("lawcontentsearch.NewRequest() のエラー = %v", err)
	}
	return request
}
