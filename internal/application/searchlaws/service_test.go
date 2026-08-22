package searchlaws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawtarget"
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
	resolve func(context.Context, string) (lawtarget.ResolvedLawTarget, bool, error)
}

func (r fakeQueryResolver) Resolve(
	ctx context.Context,
	query string,
) (lawtarget.ResolvedLawTarget, bool, error) {
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
) (lawtarget.ResolvedLawTarget, bool, error) {
	return lawtarget.ResolvedLawTarget{}, false, nil
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
			t.Fatalf("SOT-IF-053: query = %q", request.Query())
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

func TestServiceKeepsOriginalNonEmptyPageWhenResolvedTargetIsAbsentFromPage(t *testing.T) {
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
	) (lawtarget.ResolvedLawTarget, bool, error) {
		resolverCalls++
		return mustResolvedLawTarget(
			t,
			"325AC0000000105",
			"道路交通法",
			lawtarget.MatchKindRegisteredTerm,
		), true, nil
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
		t.Fatalf("SOT-IF-053: Search() のエラー = %v", err)
	}
	if got.TotalCount() != 4 || providerCalls != 1 || resolverCalls != 1 {
		t.Fatalf(
			"SOT-IF-053: total=%d, provider=%d, resolver=%d",
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
			t.Fatalf("SOT-IF-053: request = %#v", gotRequest)
		}
		if gotRequest.Query() == "個情法" {
			return empty, nil
		}
		return resolved, nil
	}}
	resolver := fakeQueryResolver{resolve: func(
		ctx context.Context,
		query string,
	) (lawtarget.ResolvedLawTarget, bool, error) {
		if err := ctx.Err(); err != nil {
			return lawtarget.ResolvedLawTarget{}, false, err
		}
		if query != "個情法" {
			t.Fatalf("SOT-IF-053: resolver query = %q", query)
		}
		return mustResolvedLawTarget(
			t,
			"425AC0000000057",
			"個人情報の保護に関する法律",
			lawtarget.MatchKindRegisteredTerm,
		), true, nil
	}}
	service, err := NewService(port, resolver, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}

	got, err := service.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-053: Search() のエラー = %v", err)
	}
	if got.TotalCount() != 2 ||
		len(queries) != 2 ||
		queries[0] != "個情法" ||
		queries[1] != "個人情報の保護に関する法律" {
		t.Fatalf(
			"SOT-IF-053: total=%d, queries=%#v",
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
	) (lawtarget.ResolvedLawTarget, bool, error) {
		resolverCalls++
		return lawtarget.ResolvedLawTarget{}, false, nil
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
		t.Fatalf("SOT-IF-053: provider error = %v", err)
	}
	if resolverCalls != 1 {
		t.Fatalf("SOT-IF-053: resolver calls = %d", resolverCalls)
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
		) (lawtarget.ResolvedLawTarget, bool, error) {
			return lawtarget.ResolvedLawTarget{}, false, resolverErr
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
		t.Fatalf("SOT-IF-053: resolver error = %v", err)
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
		) (lawtarget.ResolvedLawTarget, bool, error) {
			return mustResolvedLawTarget(
				t,
				"425AC0000000057",
				"個人情報の保護に関する法律",
				lawtarget.MatchKindRegisteredTerm,
			), true, nil
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
		t.Fatalf("SOT-IF-053: fallback provider error = %v", err)
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
		) (lawtarget.ResolvedLawTarget, bool, error) {
			resolverCalls++
			return mustResolvedLawTarget(
				t,
				"325AC0000000105",
				"道路交通法",
				lawtarget.MatchKindRegisteredTerm,
			), true, nil
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
		t.Fatal("SOT-IF-053: 不正な provider result を受理しました")
	}
	if resolverCalls != 1 {
		t.Fatalf("SOT-IF-053: resolver calls = %d", resolverCalls)
	}
}

func TestServicePrioritizesResolvedTargetWithinCurrentPage(t *testing.T) {
	t.Parallel()

	original := mustLawSearchResult(
		t,
		100,
		[]model.LawSummary{
			mustLawSummary(t, "law-1", "第一法"),
			mustLawSummary(t, "law-traffic", "道路交通法・現行"),
			mustLawSummary(t, "law-2", "第二法"),
			mustLawSummary(t, "law-traffic", "道路交通法・旧版"),
			mustLawSummary(t, "law-3", "第三法"),
		},
		intPtr(40),
	)
	providerCalls := 0
	service, err := NewService(
		fakeSearchPort{search: func(
			context.Context,
			Request,
		) (model.LawSearchResult, error) {
			providerCalls++
			return original, nil
		}},
		fakeQueryResolver{resolve: func(
			context.Context,
			string,
		) (lawtarget.ResolvedLawTarget, bool, error) {
			return mustResolvedLawTarget(
				t,
				"law-traffic",
				"道路交通法",
				lawtarget.MatchKindRegisteredTerm,
			), true, nil
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

	got, err := service.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-053: Search() のエラー = %v", err)
	}
	items := got.Items()
	if len(items) != 5 ||
		items[0].LawID() != "law-traffic" ||
		items[0].Title() != "道路交通法・現行" ||
		items[1].LawID() != "law-traffic" ||
		items[1].Title() != "道路交通法・旧版" ||
		items[2].LawID() != "law-1" ||
		items[3].LawID() != "law-2" ||
		items[4].LawID() != "law-3" {
		t.Fatalf("SOT-ARCH-030: prioritized items = %#v", items)
	}
	if nextOffset, exists := got.NextOffset(); !exists || nextOffset != 40 ||
		got.TotalCount() != 100 || providerCalls != 1 {
		t.Fatalf(
			"law-target-page-stable-partition/law-target-no-extra-fetch: "+
				"nextOffset = (%d, %t), providerCalls = %d",
			nextOffset,
			exists,
			providerCalls,
		)
	}
}

func TestServiceReturnsOriginalWhenConfirmationIsAlsoEmpty(t *testing.T) {
	t.Parallel()

	original := mustEmptyLawSearchResult(t)
	zero := 0
	confirmation := mustLawSearchResult(t, 0, []model.LawSummary{}, &zero)
	providerCalls := 0
	service, err := NewService(
		fakeSearchPort{search: func(
			context.Context,
			Request,
		) (model.LawSearchResult, error) {
			providerCalls++
			if providerCalls == 1 {
				return original, nil
			}
			return confirmation, nil
		}},
		fakeQueryResolver{resolve: func(
			context.Context,
			string,
		) (lawtarget.ResolvedLawTarget, bool, error) {
			return mustResolvedLawTarget(
				t,
				"425AC0000000057",
				"個人情報の保護に関する法律",
				lawtarget.MatchKindRegisteredTerm,
			), true, nil
		}},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{Query: "個情法"})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}

	got, err := service.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-053: Search() のエラー = %v", err)
	}
	if _, exists := got.NextOffset(); exists || providerCalls != 2 {
		t.Fatalf(
			"law-target-no-extra-fetch: 原検索を保持しませんでした: "+
				"nextOffset=%t, providerCalls=%d",
			exists,
			providerCalls,
		)
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
	return mustLawSearchResult(t, totalCount, []model.LawSummary{}, nil)
}

func mustLawSearchResult(
	t *testing.T,
	totalCount int,
	items []model.LawSummary,
	nextOffset *int,
) model.LawSearchResult {
	t.Helper()
	result, err := model.NewLawSearchResult(model.LawSearchResultValues{
		TotalCount: totalCount,
		Items:      items,
		NextOffset: nextOffset,
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
	) (lawtarget.ResolvedLawTarget, bool, error) {
		return lawtarget.ResolvedLawTarget{}, false, nil
	}}
}

func mustResolvedLawTarget(
	t *testing.T,
	lawID string,
	officialTitle string,
	matchKind lawtarget.MatchKind,
) lawtarget.ResolvedLawTarget {
	t.Helper()
	target, err := lawtarget.NewResolvedLawTarget(lawID, officialTitle, matchKind)
	if err != nil {
		t.Fatalf("ResolvedLawTarget の作成エラー = %v", err)
	}
	return target
}

func mustLawSummary(
	t *testing.T,
	lawID string,
	title string,
) model.LawSummary {
	t.Helper()
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov法令検索",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp",
	})
	if err != nil {
		t.Fatalf("InformationSource の作成エラー = %v", err)
	}
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("LegalSource の作成エラー = %v", err)
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      lawID,
		RevisionID: lawID + "-rev",
		Title:      title,
		Source:     source,
	})
	if err != nil {
		t.Fatalf("LawSummary の作成エラー = %v", err)
	}
	return summary
}

func intPtr(value int) *int {
	return &value
}
