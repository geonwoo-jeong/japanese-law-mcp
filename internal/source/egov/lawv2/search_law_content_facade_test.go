package lawv2

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/searchlawcontent"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestSearchLawContentFacadePreservesRawQueryAndOmittedAsOf(t *testing.T) {
	t.Parallel()

	var captured *http.Request
	facade := newTestSearchLawContentFacade(
		t,
		make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Clone(request.Context())
			return response(
				http.StatusOK,
				string(fixture(t, "fixtures/law-content-normal.json")),
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
	)
	queryExpression := "(情報 公開)|個人"
	request := mustPublicContentSearchRequest(t, searchlawcontent.RequestValues{
		Query: queryExpression,
	})

	result, err := facade.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-010/033: Search() のエラー = %v", err)
	}
	if result.TotalCount() != 3 || len(result.Items()) != 2 {
		t.Fatalf("SOT-MODEL-008: result = %#v", result)
	}
	if nextOffset, exists := result.NextOffset(); !exists || nextOffset != 2 {
		t.Fatalf("SOT-IF-010: nextOffset = %d, %t", nextOffset, exists)
	}
	items := result.Items()
	if items[0].Text() != "地方自治に関する第一の一致。" ||
		items[1].Text() != "<em>地方</em>自治に関する第二の一致。" {
		t.Fatalf("SOT-IF-010: items = %#v", items)
	}
	if captured == nil {
		t.Fatal("SOT-IF-010: HTTP request がありません")
	}
	query := captured.URL.Query()
	if query.Get("keyword") != queryExpression {
		t.Fatalf("SOT-IF-010: keyword = %q", query.Get("keyword"))
	}
	if query.Has("asof") {
		t.Fatalf("SOT-IF-010/033: 省略した asof を送信しました: %q", query.Get("asof"))
	}
	if query.Get("limit") != "20" ||
		query.Get("offset") != "0" ||
		query.Get("response_format") != "json" ||
		query.Get("order") != "+law_info.law_id" ||
		query.Get("highlight_tag") != "mark" {
		t.Fatalf("SOT-IF-010: query = %#v", query)
	}
	if query.Has("sentences_limit") {
		t.Fatal("SOT-IF-010: sentences_limit を送信しました")
	}
}

func TestSearchLawContentFacadeSendsExplicitAsOfAndOffset(t *testing.T) {
	t.Parallel()

	var captured *http.Request
	facade := newTestSearchLawContentFacade(
		t,
		make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			captured = request.Clone(request.Context())
			return response(
				http.StatusOK,
				string(fixture(t, "fixtures/law-content-empty.json")),
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
	)
	asOf := mustDate("2026-07-26")
	offset := 40
	request := mustPublicContentSearchRequest(t, searchlawcontent.RequestValues{
		Query:  "該当なし",
		AsOf:   &asOf,
		Offset: &offset,
	})

	result, err := facade.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-010/033: Search() のエラー = %v", err)
	}
	if result.TotalCount() != 0 || result.Items() == nil || len(result.Items()) != 0 {
		t.Fatalf("SOT-MODEL-008: 空結果 = %#v", result)
	}
	if captured.URL.Query().Get("asof") != "2026-07-26" ||
		captured.URL.Query().Get("offset") != "40" {
		t.Fatalf("SOT-IF-010: query = %#v", captured.URL.Query())
	}
}

func TestSearchLawContentFacadeReusesPageValidation(t *testing.T) {
	t.Parallel()

	facade := newTestSearchLawContentFacade(
		t,
		make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			return response(
				http.StatusOK,
				string(fixture(t, "fixtures/law-content-normal.json")),
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
	)
	limit := 1
	request := mustPublicContentSearchRequest(t, searchlawcontent.RequestValues{
		Query: "自治",
		Limit: &limit,
	})

	_, err := facade.Search(context.Background(), request)
	assertSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
}

func TestSearchLawContentFacadeUsesSharedGateAndSourceErrors(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	attempts := 0
	facade := newTestSearchLawContentFacade(
		t,
		gate,
		doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("呼び出してはならない HTTP")
		}),
	)
	request := mustPublicContentSearchRequest(
		t,
		searchlawcontent.RequestValues{Query: "自治"},
	)

	_, err := facade.Search(context.Background(), request)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceBusy)
	if attempts != 0 {
		t.Fatalf("SOT-IF-004: HTTP attempts = %d", attempts)
	}
}

func TestSearchLawContentFacadeRejectsInvalidDependenciesAndInput(t *testing.T) {
	t.Parallel()

	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("未使用")
		}),
		now:   time.Now,
		sleep: sleepWithContext,
	})
	if _, err := newSearchLawContentFacade(searchLawContentFacadeDependencies{
		client: client,
		gate:   make(chan struct{}, 2),
	}); err == nil {
		t.Fatal("容量 2 の gate を受理しました")
	}
	facade := newTestSearchLawContentFacade(
		t,
		make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("呼び出してはならない HTTP")
		}),
	)
	var nilContext context.Context
	if _, err := facade.Search(
		nilContext,
		mustPublicContentSearchRequest(
			t,
			searchlawcontent.RequestValues{Query: "自治"},
		),
	); err == nil {
		t.Fatal("nil context を受理しました")
	}
	if _, err := facade.Search(context.Background(), searchlawcontent.Request{}); err == nil {
		t.Fatal("ゼロ値 Request を受理しました")
	}
}

func newTestSearchLawContentFacade(
	t *testing.T,
	gate chan struct{},
	doer httpDoer,
) *SearchLawContentFacade {
	t.Helper()

	client := mustTestClient(t, clientDependencies{
		doer:  doer,
		now:   time.Now,
		sleep: sleepWithContext,
	})
	facade, err := newSearchLawContentFacade(searchLawContentFacadeDependencies{
		client: client,
		gate:   gate,
	})
	if err != nil {
		t.Fatalf("newSearchLawContentFacade() のエラー = %v", err)
	}
	return facade
}

func mustPublicContentSearchRequest(
	t *testing.T,
	values searchlawcontent.RequestValues,
) searchlawcontent.Request {
	t.Helper()

	request, err := searchlawcontent.NewRequest(values)
	if err != nil {
		t.Fatalf("searchlawcontent.NewRequest() のエラー = %v", err)
	}
	return request
}
