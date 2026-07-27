package judicialdecisionsearch_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type fakeSearchPort struct {
	search func(context.Context, judicialdecisionsearch.Request) (
		judicialdecisionsearch.Page,
		error,
	)
}

func (p fakeSearchPort) Search(
	ctx context.Context,
	request judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	return p.search(ctx, request)
}

var _ judicialdecisionsearch.Port = fakeSearchPort{}

func TestServiceAppliesTimeoutAndReturnsProviderPage(t *testing.T) {
	t.Parallel()

	want := mustSearchPage(t)
	port := fakeSearchPort{search: func(
		ctx context.Context,
		request judicialdecisionsearch.Request,
	) (judicialdecisionsearch.Page, error) {
		deadline, exists := ctx.Deadline()
		if !exists {
			t.Fatal("SOT-ENG-010/SOT-IF-015: deadline が設定されていない")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > time.Second {
			t.Fatalf("SOT-ENG-010: deadline までの時間 = %s", remaining)
		}
		if request.Query() != "民法" {
			t.Fatalf("SOT-IF-041: query = %q", request.Query())
		}
		return want, nil
	}}
	service, err := judicialdecisionsearch.NewService(port, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request := mustSearchRequest(t)
	got, err := service.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("Search() のエラー = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Search() = %#v、期待値 = %#v", got, want)
	}
}

func TestServicePropagatesParentCancellation(t *testing.T) {
	t.Parallel()

	port := fakeSearchPort{search: func(
		ctx context.Context,
		_ judicialdecisionsearch.Request,
	) (judicialdecisionsearch.Page, error) {
		<-ctx.Done()
		return judicialdecisionsearch.Page{}, ctx.Err()
	}}
	service, err := judicialdecisionsearch.NewService(port, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.Search(ctx, mustSearchRequest(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ENG-010/SOT-IF-015: error = %v", err)
	}
}

func TestServiceStopsProviderAtTimeout(t *testing.T) {
	t.Parallel()

	port := fakeSearchPort{search: func(
		ctx context.Context,
		_ judicialdecisionsearch.Request,
	) (judicialdecisionsearch.Page, error) {
		<-ctx.Done()
		return judicialdecisionsearch.Page{}, ctx.Err()
	}}
	service, err := judicialdecisionsearch.NewService(port, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	_, err = service.Search(context.Background(), mustSearchRequest(t))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SOT-ENG-010/SOT-IF-015: error = %v", err)
	}
}

func TestServicePreservesProviderError(t *testing.T) {
	t.Parallel()

	want := errors.New("試験用 provider error")
	port := fakeSearchPort{search: func(
		context.Context,
		judicialdecisionsearch.Request,
	) (judicialdecisionsearch.Page, error) {
		return judicialdecisionsearch.Page{}, want
	}}
	service, err := judicialdecisionsearch.NewService(port, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	_, err = service.Search(context.Background(), mustSearchRequest(t))
	if !errors.Is(err, want) {
		t.Fatalf("SOT-IF-015: error = %v", err)
	}
}

func TestServiceRejectsInvalidProviderPage(t *testing.T) {
	t.Parallel()

	port := fakeSearchPort{search: func(
		context.Context,
		judicialdecisionsearch.Request,
	) (judicialdecisionsearch.Page, error) {
		return judicialdecisionsearch.Page{}, nil
	}}
	service, err := judicialdecisionsearch.NewService(port, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}

	if _, err := service.Search(context.Background(), mustSearchRequest(t)); err == nil {
		t.Fatal("SOT-IF-041: 無効な provider page を受理した")
	}
}

func TestServiceRejectsInvalidDependenciesAndInputsBeforeProvider(t *testing.T) {
	t.Parallel()

	if _, err := judicialdecisionsearch.NewService(nil, time.Second); err == nil {
		t.Fatal("nil port を受理した")
	}
	var typedNilPort *fakeSearchPort
	if _, err := judicialdecisionsearch.NewService(typedNilPort, time.Second); err == nil {
		t.Fatal("typed nil port を受理した")
	}
	called := false
	port := fakeSearchPort{search: func(
		context.Context,
		judicialdecisionsearch.Request,
	) (judicialdecisionsearch.Page, error) {
		called = true
		return judicialdecisionsearch.Page{}, nil
	}}
	if _, err := judicialdecisionsearch.NewService(port, 0); err == nil {
		t.Fatal("0 の requestTimeout を受理した")
	}
	service, err := judicialdecisionsearch.NewService(port, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	var nilContext context.Context
	if _, err := service.Search(nilContext, mustSearchRequest(t)); err == nil {
		t.Fatal("nil context を受理した")
	}
	if _, err := service.Search(context.Background(), judicialdecisionsearch.Request{}); err == nil {
		t.Fatal("zero value request を受理した")
	}
	if called {
		t.Fatal("不正な入力で provider を呼び出した")
	}
}

func TestServiceRejectsProviderPageOverRequestLimit(t *testing.T) {
	t.Parallel()

	page, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{
			Items: []model.SourcedResource[model.JudicialDecisionSummary]{
				newJudicialSearchItem(t, judicialItemValues{decisionID: "100"}),
				newJudicialSearchItem(t, judicialItemValues{decisionID: "200"}),
			},
			Page: mustSourcePage(t, 2),
		},
	)
	if err != nil {
		t.Fatalf("試験用 Page を作成できない: %v", err)
	}
	port := fakeSearchPort{search: func(
		context.Context,
		judicialdecisionsearch.Request,
	) (judicialdecisionsearch.Page, error) {
		return page, nil
	}}
	service, err := judicialdecisionsearch.NewService(port, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	limit := 1
	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{
			Query: "民法",
			Limit: &limit,
		},
	)
	if err != nil {
		t.Fatalf("試験用 Request を作成できない: %v", err)
	}
	if _, err := service.Search(context.Background(), request); err == nil {
		t.Fatal("SOT-IF-041: request.limit を超える provider page を受理した")
	}
}

func mustSearchRequest(t *testing.T) judicialdecisionsearch.Request {
	t.Helper()

	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{Query: "民法"},
	)
	if err != nil {
		t.Fatalf("試験用 Request を作成できない: %v", err)
	}
	return request
}

func mustSearchPage(t *testing.T) judicialdecisionsearch.Page {
	t.Helper()

	page, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{
			Items: nil,
			Page:  mustSourcePage(t, 0),
		},
	)
	if err != nil {
		t.Fatalf("試験用 Page を作成できない: %v", err)
	}
	return page
}

func mustSourcePage(t *testing.T, returnedCount int) model.SourcePage {
	t.Helper()

	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: returnedCount,
	})
	if err != nil {
		t.Fatalf("試験用 SourcePage を作成できない: %v", err)
	}
	return page
}
