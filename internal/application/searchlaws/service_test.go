package searchlaws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type fakeSearchPort struct {
	search func(context.Context, Request) (model.LawSearchResult, error)
}

func (p fakeSearchPort) Search(
	ctx context.Context,
	request Request,
) (model.LawSearchResult, error) {
	return p.search(ctx, request)
}

type fakeQueryResolver struct {
	resolve func(context.Context, string) (string, bool, error)
}

func (r fakeQueryResolver) Resolve(
	ctx context.Context,
	query string,
) (string, bool, error) {
	return r.resolve(ctx, query)
}

type pointerSearchPort struct{}

func (*pointerSearchPort) Search(
	context.Context,
	Request,
) (model.LawSearchResult, error) {
	return model.LawSearchResult{}, nil
}

type pointerQueryResolver struct{}

func (*pointerQueryResolver) Resolve(
	context.Context,
	string,
) (string, bool, error) {
	return "", false, nil
}

func TestServiceAppliesRequestTimeoutAndReturnsProviderResult(t *testing.T) {
	t.Parallel()

	want := mustEmptyLawSearchResult(t)
	port := fakeSearchPort{search: func(
		ctx context.Context,
		request Request,
	) (model.LawSearchResult, error) {
		deadline, exists := ctx.Deadline()
		if !exists {
			t.Fatal("SOT-IF-029: request timeout が設定されていない")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > time.Second {
			t.Fatalf("SOT-IF-029: deadline までの時間 = %s", remaining)
		}
		if request.Query() != "民法" {
			t.Fatalf("SOT-IF-049: query = %q", request.Query())
		}
		return want, nil
	}}
	service, err := NewService(port, noMatchQueryResolver(), time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{Query: "民法"})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}
	got, err := service.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("Search() のエラー = %v", err)
	}
	if got.TotalCount() != want.TotalCount() {
		t.Fatalf("Search() = %#v", got)
	}
}

func TestServiceStopsProviderAtRequestTimeout(t *testing.T) {
	t.Parallel()

	port := fakeSearchPort{search: func(
		ctx context.Context,
		_ Request,
	) (model.LawSearchResult, error) {
		<-ctx.Done()
		return model.LawSearchResult{}, ctx.Err()
	}}
	service, err := NewService(
		port,
		noMatchQueryResolver(),
		10*time.Millisecond,
	)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{Query: "民法"})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}
	_, err = service.Search(context.Background(), request)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SOT-IF-029: error = %v", err)
	}
}

func TestServiceRejectsInvalidDependenciesAndInputs(t *testing.T) {
	t.Parallel()

	if _, err := NewService(nil, noMatchQueryResolver(), time.Second); err == nil {
		t.Fatal("nil port を受理した")
	}
	port := fakeSearchPort{search: func(
		context.Context,
		Request,
	) (model.LawSearchResult, error) {
		return mustEmptyLawSearchResult(t), nil
	}}
	if _, err := NewService(port, nil, time.Second); err == nil {
		t.Fatal("nil query resolver を受理した")
	}
	var typedNilPort *pointerSearchPort
	if _, err := NewService(
		typedNilPort,
		noMatchQueryResolver(),
		time.Second,
	); err == nil {
		t.Fatal("typed nil port を受理した")
	}
	var typedNilResolver *pointerQueryResolver
	if _, err := NewService(port, typedNilResolver, time.Second); err == nil {
		t.Fatal("typed nil query resolver を受理した")
	}
	if _, err := NewService(port, noMatchQueryResolver(), 0); err == nil {
		t.Fatal("0 の requestTimeout を受理した")
	}
	service, err := NewService(port, noMatchQueryResolver(), time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	var nilContext context.Context
	if _, err := service.Search(nilContext, Request{}); err == nil {
		t.Fatal("nil context を受理した")
	}
	if _, err := service.Search(context.Background(), Request{}); err == nil {
		t.Fatal("zero value request を受理した")
	}
}

func TestServiceKeepsOriginalNonEmptyPageWithoutResolving(t *testing.T) {
	t.Parallel()

	original := mustLawSearchResultWithTotalCount(t, 4)
	providerCalls := 0
	port := fakeSearchPort{search: func(
		context.Context,
		Request,
	) (model.LawSearchResult, error) {
		providerCalls++
		return original, nil
	}}
	resolverCalls := 0
	resolver := fakeQueryResolver{resolve: func(
		context.Context,
		string,
	) (string, bool, error) {
		resolverCalls++
		return "道路交通法", true, nil
	}}
	service, err := NewService(port, resolver, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	offset := 100
	request, err := NewRequest(RequestValues{
		Query:  "道交法",
		Offset: &offset,
	})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}

	got, err := service.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-049: Search() のエラー = %v", err)
	}
	if got.TotalCount() != 4 || providerCalls != 1 || resolverCalls != 0 {
		t.Fatalf(
			"SOT-IF-049: total=%d, provider=%d, resolver=%d",
			got.TotalCount(),
			providerCalls,
			resolverCalls,
		)
	}
}

func TestServiceFallsBackOnceAndPreservesRequestControls(t *testing.T) {
	t.Parallel()

	empty := mustEmptyLawSearchResult(t)
	resolved := mustLawSearchResultWithTotalCount(t, 2)
	asOf := mustSearchLawsDate(t, "2026-07-27")
	limit := 7
	offset := 41
	request, err := NewRequest(RequestValues{
		Query:  "個情法",
		AsOf:   &asOf,
		Limit:  &limit,
		Offset: &offset,
	})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}

	queries := make([]string, 0, 2)
	port := fakeSearchPort{search: func(
		_ context.Context,
		gotRequest Request,
	) (model.LawSearchResult, error) {
		queries = append(queries, gotRequest.Query())
		gotAsOf, exists := gotRequest.AsOf()
		if !exists || gotAsOf.String() != "2026-07-27" ||
			gotRequest.Limit() != 7 || gotRequest.Offset() != 41 {
			t.Fatalf("SOT-IF-049: request = %#v", gotRequest)
		}
		if gotRequest.Query() == "個情法" {
			return empty, nil
		}
		return resolved, nil
	}}
	resolver := fakeQueryResolver{resolve: func(
		ctx context.Context,
		query string,
	) (string, bool, error) {
		if err := ctx.Err(); err != nil {
			return "", false, err
		}
		if query != "個情法" {
			t.Fatalf("SOT-IF-049: resolver query = %q", query)
		}
		return "個人情報の保護に関する法律", true, nil
	}}
	service, err := NewService(port, resolver, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}

	got, err := service.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-049: Search() のエラー = %v", err)
	}
	if got.TotalCount() != 2 ||
		len(queries) != 2 ||
		queries[0] != "個情法" ||
		queries[1] != "個人情報の保護に関する法律" {
		t.Fatalf(
			"SOT-IF-049: total=%d, queries=%#v",
			got.TotalCount(),
			queries,
		)
	}
}

func TestServiceDoesNotHideProviderOrResolverErrors(t *testing.T) {
	t.Parallel()

	request, err := NewRequest(RequestValues{Query: "個情法"})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}
	providerErr := errors.New("情報源エラー")
	resolverCalls := 0
	resolver := fakeQueryResolver{resolve: func(
		context.Context,
		string,
	) (string, bool, error) {
		resolverCalls++
		return "", false, nil
	}}
	service, err := NewService(
		fakeSearchPort{search: func(
			context.Context,
			Request,
		) (model.LawSearchResult, error) {
			return model.LawSearchResult{}, providerErr
		}},
		resolver,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	if _, err := service.Search(context.Background(), request); !errors.Is(
		err,
		providerErr,
	) {
		t.Fatalf("SOT-IF-049: provider error = %v", err)
	}
	if resolverCalls != 0 {
		t.Fatalf("SOT-IF-049: resolver calls = %d", resolverCalls)
	}

	resolverErr := errors.New("検索語解決エラー")
	service, err = NewService(
		fakeSearchPort{search: func(
			context.Context,
			Request,
		) (model.LawSearchResult, error) {
			return mustEmptyLawSearchResult(t), nil
		}},
		fakeQueryResolver{resolve: func(
			context.Context,
			string,
		) (string, bool, error) {
			return "", false, resolverErr
		}},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	if _, err := service.Search(context.Background(), request); !errors.Is(
		err,
		resolverErr,
	) {
		t.Fatalf("SOT-IF-049: resolver error = %v", err)
	}

	fallbackErr := errors.New("候補検索の情報源エラー")
	providerCalls := 0
	service, err = NewService(
		fakeSearchPort{search: func(
			context.Context,
			Request,
		) (model.LawSearchResult, error) {
			providerCalls++
			if providerCalls == 1 {
				return mustEmptyLawSearchResult(t), nil
			}
			return model.LawSearchResult{}, fallbackErr
		}},
		fakeQueryResolver{resolve: func(
			context.Context,
			string,
		) (string, bool, error) {
			return "個人情報の保護に関する法律", true, nil
		}},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	if _, err := service.Search(context.Background(), request); !errors.Is(
		err,
		fallbackErr,
	) {
		t.Fatalf("SOT-IF-049: fallback provider error = %v", err)
	}
}

func TestServiceRejectsInvalidProviderResultBeforeFallback(t *testing.T) {
	t.Parallel()

	resolverCalls := 0
	service, err := NewService(
		fakeSearchPort{search: func(
			context.Context,
			Request,
		) (model.LawSearchResult, error) {
			return model.LawSearchResult{}, nil
		}},
		fakeQueryResolver{resolve: func(
			context.Context,
			string,
		) (string, bool, error) {
			resolverCalls++
			return "道路交通法", true, nil
		}},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{Query: "道交法"})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}
	if _, err := service.Search(context.Background(), request); err == nil {
		t.Fatal("SOT-IF-049: 不正な provider result を受理しました")
	}
	if resolverCalls != 0 {
		t.Fatalf("SOT-IF-049: resolver calls = %d", resolverCalls)
	}
}

func mustEmptyLawSearchResult(t *testing.T) model.LawSearchResult {
	t.Helper()
	result, err := model.NewLawSearchResult(model.LawSearchResultValues{})
	if err != nil {
		t.Fatalf("LawSearchResult の作成エラー = %v", err)
	}
	return result
}

func mustLawSearchResultWithTotalCount(
	t *testing.T,
	totalCount int,
) model.LawSearchResult {
	t.Helper()
	result, err := model.NewLawSearchResult(model.LawSearchResultValues{
		TotalCount: totalCount,
		Items:      []model.LawSummary{},
	})
	if err != nil {
		t.Fatalf("LawSearchResult の作成エラー = %v", err)
	}
	return result
}

func noMatchQueryResolver() fakeQueryResolver {
	return fakeQueryResolver{resolve: func(
		context.Context,
		string,
	) (string, bool, error) {
		return "", false, nil
	}}
}
