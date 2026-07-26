package lawv2

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/searchlaws"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestSearchLawsFacadePreservesPublicOffsetAndOmittedAsOf(t *testing.T) {
	t.Parallel()

	var captured *http.Request
	facade := newTestSearchLawsFacade(
		t,
		make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Clone(request.Context())
			return response(
				http.StatusOK,
				string(fixture(t, "fixtures/law-search-page.json")),
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
	)
	limit := 1
	request := mustPublicSearchRequest(t, searchlaws.RequestValues{
		Query:  "地方自治",
		Limit:  &limit,
		Offset: func() *int { value := 0; return &value }(),
	})
	result, err := facade.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-009/030: Search() のエラー = %v", err)
	}
	if result.TotalCount() != 2 || len(result.Items()) != 1 {
		t.Fatalf("SOT-MODEL-006: result = %#v", result)
	}
	if nextOffset, exists := result.NextOffset(); !exists || nextOffset != 1 {
		t.Fatalf("SOT-IF-009: nextOffset = %d, %t", nextOffset, exists)
	}
	if captured == nil {
		t.Fatal("SOT-IF-009: HTTP request がない")
	}
	query := captured.URL.Query()
	if query.Has("asof") {
		t.Fatalf("SOT-IF-009/030: 省略した asof を送信した: %q", query.Get("asof"))
	}
	if query.Get("law_title") != "地方自治" ||
		query.Get("limit") != "1" ||
		query.Get("offset") != "0" ||
		query.Get("response_format") != "json" ||
		query.Get("order") != "+law_info.law_id" {
		t.Fatalf("SOT-IF-009: query = %#v", query)
	}
}

func TestSearchLawsFacadeSendsExplicitAsOfAndOffset(t *testing.T) {
	t.Parallel()

	var captured *http.Request
	facade := newTestSearchLawsFacade(
		t,
		make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Clone(request.Context())
			return response(
				http.StatusOK,
				string(fixture(t, "fixtures/law-search-empty.json")),
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
	)
	asOf := mustDate("2026-07-26")
	offset := 40
	request := mustPublicSearchRequest(t, searchlaws.RequestValues{
		Query:  "該当なし",
		AsOf:   &asOf,
		Offset: &offset,
	})
	result, err := facade.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-009/030: Search() のエラー = %v", err)
	}
	if result.TotalCount() != 0 || result.Items() == nil || len(result.Items()) != 0 {
		t.Fatalf("SOT-MODEL-006: 空結果 = %#v", result)
	}
	if captured.URL.Query().Get("asof") != "2026-07-26" ||
		captured.URL.Query().Get("offset") != "40" {
		t.Fatalf("SOT-IF-009: query = %#v", captured.URL.Query())
	}
}

func TestSearchLawsFacadeUsesSharedGateAndSourceErrors(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	attempts := 0
	facade := newTestSearchLawsFacade(
		t,
		gate,
		doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("呼び出してはならない HTTP")
		}),
	)
	request := mustPublicSearchRequest(t, searchlaws.RequestValues{Query: "民法"})
	_, err := facade.Search(context.Background(), request)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceBusy)
	if attempts != 0 {
		t.Fatalf("SOT-IF-004: HTTP attempts = %d", attempts)
	}
}

func TestSearchLawsFacadeRejectsInvalidDependenciesAndContext(t *testing.T) {
	t.Parallel()

	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("未使用")
		}),
		now:   time.Now,
		sleep: sleepWithContext,
	})
	if _, err := newSearchLawsFacade(searchLawsFacadeDependencies{
		client: client,
		gate:   make(chan struct{}, 2),
	}); err == nil {
		t.Fatal("容量 2 の gate を受理した")
	}
	facade := newTestSearchLawsFacade(
		t,
		make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("未使用")
		}),
	)
	request := mustPublicSearchRequest(t, searchlaws.RequestValues{Query: "民法"})
	var nilContext context.Context
	if _, err := facade.Search(nilContext, request); err == nil {
		t.Fatal("nil context を受理した")
	}
}

func newTestSearchLawsFacade(
	t *testing.T,
	gate chan struct{},
	doer httpDoer,
) *SearchLawsFacade {
	t.Helper()
	client := mustTestClient(t, clientDependencies{
		doer:  doer,
		now:   time.Now,
		sleep: sleepWithContext,
	})
	facade, err := newSearchLawsFacade(searchLawsFacadeDependencies{
		client: client,
		gate:   gate,
	})
	if err != nil {
		t.Fatalf("newSearchLawsFacade() のエラー = %v", err)
	}
	return facade
}

func mustPublicSearchRequest(
	t *testing.T,
	values searchlaws.RequestValues,
) searchlaws.Request {
	t.Helper()
	request, err := searchlaws.NewRequest(values)
	if err != nil {
		t.Fatalf("searchlaws.NewRequest() のエラー = %v", err)
	}
	return request
}
