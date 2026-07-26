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
			t.Fatalf("SOT-IF-030: query = %q", request.Query())
		}
		return want, nil
	}}
	service, err := NewService(port, time.Second)
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
	service, err := NewService(port, 10*time.Millisecond)
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

	if _, err := NewService(nil, time.Second); err == nil {
		t.Fatal("nil port を受理した")
	}
	port := fakeSearchPort{search: func(
		context.Context,
		Request,
	) (model.LawSearchResult, error) {
		return mustEmptyLawSearchResult(t), nil
	}}
	if _, err := NewService(port, 0); err == nil {
		t.Fatal("0 の requestTimeout を受理した")
	}
	service, err := NewService(port, time.Second)
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

func mustEmptyLawSearchResult(t *testing.T) model.LawSearchResult {
	t.Helper()
	result, err := model.NewLawSearchResult(model.LawSearchResultValues{})
	if err != nil {
		t.Fatalf("LawSearchResult の作成エラー = %v", err)
	}
	return result
}
