package searchlawcontent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type fakeSearchLawContentPort struct {
	search func(context.Context, Request) (model.LawContentSearchResult, error)
}

func (p fakeSearchLawContentPort) Search(
	ctx context.Context,
	request Request,
) (model.LawContentSearchResult, error) {
	return p.search(ctx, request)
}

func TestServiceAppliesRequestTimeoutAndReturnsProviderResult(t *testing.T) {
	t.Parallel()

	want := mustEmptyLawContentSearchResult(t)
	port := fakeSearchLawContentPort{search: func(
		ctx context.Context,
		request Request,
	) (model.LawContentSearchResult, error) {
		deadline, exists := ctx.Deadline()
		if !exists {
			t.Fatal("SOT-IF-033: request timeout が設定されていない")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > time.Second {
			t.Fatalf("SOT-IF-033: deadline までの時間 = %s", remaining)
		}
		if request.Query() != "情報 公開" {
			t.Fatalf("SOT-IF-033: query = %q", request.Query())
		}
		return want, nil
	}}
	service, err := NewService(port, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{Query: "情報 公開"})
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

	port := fakeSearchLawContentPort{search: func(
		ctx context.Context,
		_ Request,
	) (model.LawContentSearchResult, error) {
		<-ctx.Done()
		return model.LawContentSearchResult{}, ctx.Err()
	}}
	service, err := NewService(port, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{Query: "情報"})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}
	_, err = service.Search(context.Background(), request)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SOT-IF-033: error = %v", err)
	}
}

func TestServiceRejectsInvalidDependenciesInputsAndResults(t *testing.T) {
	t.Parallel()

	if _, err := NewService(nil, time.Second); err == nil {
		t.Fatal("nil port を受理した")
	}
	validPort := fakeSearchLawContentPort{search: func(
		context.Context,
		Request,
	) (model.LawContentSearchResult, error) {
		return mustEmptyLawContentSearchResult(t), nil
	}}
	if _, err := NewService(validPort, 0); err == nil {
		t.Fatal("0 の requestTimeout を受理した")
	}
	service, err := NewService(validPort, time.Second)
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

	invalidResultPort := fakeSearchLawContentPort{search: func(
		context.Context,
		Request,
	) (model.LawContentSearchResult, error) {
		return model.LawContentSearchResult{}, nil
	}}
	invalidResultService, err := NewService(invalidResultPort, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{Query: "情報"})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}
	if _, err := invalidResultService.Search(
		context.Background(),
		request,
	); err == nil {
		t.Fatal("provider の不正な結果を受理した")
	}
}

func mustEmptyLawContentSearchResult(t *testing.T) model.LawContentSearchResult {
	t.Helper()
	result, err := model.NewLawContentSearchResult(
		model.LawContentSearchResultValues{},
	)
	if err != nil {
		t.Fatalf("LawContentSearchResult の作成エラー = %v", err)
	}
	return result
}
