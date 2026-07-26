package listlawupdates

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestServiceCallsCommonPortWithTimeoutAndProjectsResult(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	lister := &recordingLawUpdateLister{
		page: mustCommonLawUpdatePage(t, date),
	}
	service, err := NewService(lister, time.Second)
	if err != nil {
		t.Fatalf("SOT-IF-038: NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{Date: date})
	if err != nil {
		t.Fatalf("試験用 Request を作成できません: %v", err)
	}

	result, err := service.List(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-IF-038: List() のエラー = %v", err)
	}
	if lister.calls != 1 || lister.request.Date() != date {
		t.Fatalf(
			"SOT-IF-038: provider 呼出し = %d, date = %q",
			lister.calls,
			lister.request.Date().String(),
		)
	}
	if !lister.hadDeadline {
		t.Fatal("SOT-ENG-010/SOT-IF-038: request timeout が設定されていない")
	}
	if result.Date() != date ||
		result.TotalCount() != 1 ||
		len(result.Items()) != 1 ||
		result.Items()[0].LawID() != "law-001" {
		t.Fatalf("SOT-IF-038: Result = %#v", result)
	}
}

func TestServiceRevalidatesProviderPage(t *testing.T) {
	t.Parallel()

	requestDate := mustListLawUpdatesDate(t, "2026-07-26")
	lister := &recordingLawUpdateLister{
		page: mustCommonLawUpdatePage(
			t,
			mustListLawUpdatesDate(t, "2026-07-25"),
		),
	}
	service, err := NewService(lister, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{Date: requestDate})
	if err != nil {
		t.Fatalf("試験用 Request を作成できません: %v", err)
	}

	if _, err := service.List(context.Background(), request); !errors.Is(
		err,
		ErrInvalidSourceResponse,
	) {
		t.Fatalf(
			"SOT-IF-038: 要求日と異なる provider page のエラー = %v",
			err,
		)
	}

	lister.page = lawupdatelist.Page{}
	if _, err := service.List(context.Background(), request); !errors.Is(
		err,
		ErrInvalidSourceResponse,
	) {
		t.Fatalf("SOT-IF-038: 無効な provider page のエラー = %v", err)
	}
}

func TestServicePropagatesCancellationAndProviderError(t *testing.T) {
	t.Parallel()

	date := mustListLawUpdatesDate(t, "2026-07-26")
	request, err := NewRequest(RequestValues{Date: date})
	if err != nil {
		t.Fatalf("試験用 Request を作成できません: %v", err)
	}

	waiting := &recordingLawUpdateLister{waitForContext: true}
	service, err := NewService(waiting, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	if _, err := service.List(context.Background(), request); !errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		t.Fatalf("SOT-ENG-010/SOT-IF-038: timeout error = %v", err)
	}

	want := errors.New("provider failure")
	failing := &recordingLawUpdateLister{err: want}
	service, err = NewService(failing, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	if _, err := service.List(context.Background(), request); !errors.Is(err, want) {
		t.Fatalf("SOT-IF-038: provider error = %v", err)
	}
}

func TestServiceRejectsInvalidDependenciesContextAndRequest(t *testing.T) {
	t.Parallel()

	if _, err := NewService(nil, time.Second); err == nil {
		t.Fatal("nil law.update.list port を受理した")
	}
	lister := &recordingLawUpdateLister{}
	if _, err := NewService(lister, 0); err == nil {
		t.Fatal("0 の requestTimeout を受理した")
	}
	service, err := NewService(lister, time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	var nilContext context.Context
	if _, err := service.List(nilContext, Request{}); err == nil {
		t.Fatal("nil context を受理した")
	}
	if _, err := service.List(context.Background(), Request{}); err == nil {
		t.Fatal("zero value request を受理した")
	}
	if lister.calls != 0 {
		t.Fatalf("不正入力で provider を %d 回呼び出した", lister.calls)
	}
}

type recordingLawUpdateLister struct {
	page           lawupdatelist.Page
	err            error
	request        lawupdatelist.Request
	calls          int
	hadDeadline    bool
	waitForContext bool
}

func (l *recordingLawUpdateLister) List(
	ctx context.Context,
	request lawupdatelist.Request,
) (lawupdatelist.Page, error) {
	l.calls++
	l.request = request
	_, l.hadDeadline = ctx.Deadline()
	if l.waitForContext {
		<-ctx.Done()
		return lawupdatelist.Page{}, ctx.Err()
	}
	return l.page, l.err
}

func mustCommonLawUpdatePage(
	t *testing.T,
	date model.Date,
) lawupdatelist.Page {
	t.Helper()

	update := mustListLawUpdate(t, date, "law-001")
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         update.Source().ID(),
		Name:       update.Source().Name(),
		Publisher:  "デジタル庁",
		Authority:  update.Source().Authority(),
		ServiceURL: update.Source().ServiceURL(),
	})
	if err != nil {
		t.Fatalf("試験用 InformationSource を作成できません: %v", err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     informationSource.ID(),
		ResourceType: "law-update-list",
		ResourceID:   date.String(),
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "e-gov-law-api-v1",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を作成できません: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         informationSource,
		ResourceKey:    key,
		URL:            "https://laws.e-gov.go.jp/api/1/updatelawlists/20260726",
		RetrievedAt:    time.Date(2026, time.July, 26, 1, 2, 3, 0, time.UTC),
		MediaType:      "text/xml",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-036",
	})
	if err != nil {
		t.Fatalf("試験用 Provenance を作成できません: %v", err)
	}
	resource, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.LawUpdate]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       update,
		},
	)
	if err != nil {
		t.Fatalf("試験用 SourcedResource を作成できません: %v", err)
	}
	totalCount := 1
	sourcePage, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: totalCount,
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("試験用 SourcePage を作成できません: %v", err)
	}
	page, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
		Items: []model.SourcedResource[model.LawUpdate]{resource},
		Page:  sourcePage,
		Date:  date,
	})
	if err != nil {
		t.Fatalf("試験用 lawupdatelist.Page を作成できません: %v", err)
	}
	return page
}
