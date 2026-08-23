package listlawrevisions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawrevisionlist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestServiceProjectsCompleteCommonPage(t *testing.T) {
	t.Parallel()

	page := mustListLawRevisionsPage(t, "law-1", "revision-2", "revision-1")
	lister := &recordingLawRevisionLister{page: page}
	service, err := NewService(lister, time.Second)
	if err != nil {
		t.Fatalf("Service を作成できません: %v", err)
	}
	request, err := NewRequest(RequestValues{LawIDOrNumber: "law-1"})
	if err != nil {
		t.Fatalf("Request を作成できません: %v", err)
	}
	result, err := service.List(context.Background(), request)
	if err != nil {
		t.Fatalf("List() のエラー = %v", err)
	}
	if lister.calls != 1 || lister.request.LawIDOrNumber() != "law-1" {
		t.Fatalf("provider 呼出し = %d, %#v", lister.calls, lister.request)
	}
	items := result.Items()
	if result.LawID() != "law-1" || result.TotalCount() != 2 ||
		items[0].RevisionID() != "revision-2" || items[1].RevisionID() != "revision-1" {
		t.Fatalf("公開結果 = lawId:%q count:%d items:%#v", result.LawID(), result.TotalCount(), items)
	}
	items[0] = items[1]
	if result.Items()[0].RevisionID() != "revision-2" {
		t.Fatal("Result.Items が内部 slice を公開しました")
	}
}

func TestServiceRejectsInvalidProviderPage(t *testing.T) {
	t.Parallel()

	service, err := NewService(&recordingLawRevisionLister{}, time.Second)
	if err != nil {
		t.Fatalf("Service を作成できません: %v", err)
	}
	request, err := NewRequest(RequestValues{LawIDOrNumber: "law-1"})
	if err != nil {
		t.Fatalf("Request を作成できません: %v", err)
	}
	if _, err := service.List(context.Background(), request); err == nil {
		t.Fatal("不正な provider page を拒否しませんでした")
	}
}

func TestServicePropagatesTimeoutAndProviderError(t *testing.T) {
	t.Parallel()

	request, err := NewRequest(RequestValues{LawIDOrNumber: "law-1"})
	if err != nil {
		t.Fatalf("Request を作成できません: %v", err)
	}

	waiting := &recordingLawRevisionLister{waitForContext: true}
	service, err := NewService(waiting, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Service を作成できません: %v", err)
	}
	if _, err := service.List(context.Background(), request); !errors.Is(
		err,
		context.DeadlineExceeded,
	) {
		t.Fatalf("timeout error = %v", err)
	}

	want := errors.New("provider failure")
	failing := &recordingLawRevisionLister{err: want}
	service, err = NewService(failing, time.Second)
	if err != nil {
		t.Fatalf("Service を作成できません: %v", err)
	}
	if _, err := service.List(context.Background(), request); !errors.Is(err, want) {
		t.Fatalf("provider error = %v", err)
	}
}

func TestServiceRejectsInvalidDependenciesContextAndRequest(t *testing.T) {
	t.Parallel()

	if _, err := NewService(nil, time.Second); err == nil {
		t.Fatal("nil lister を受理しました")
	}
	lister := &recordingLawRevisionLister{}
	if _, err := NewService(lister, 0); err == nil {
		t.Fatal("0 の requestTimeout を受理しました")
	}
	service, err := NewService(lister, time.Second)
	if err != nil {
		t.Fatalf("Service を作成できません: %v", err)
	}
	var nilContext context.Context
	if _, err := service.List(nilContext, Request{}); err == nil {
		t.Fatal("nil context を受理しました")
	}
	if _, err := service.List(context.Background(), Request{}); err == nil {
		t.Fatal("zero value request を受理しました")
	}
	if lister.calls != 0 {
		t.Fatalf("不正入力で provider を %d 回呼び出しました", lister.calls)
	}
}

type recordingLawRevisionLister struct {
	page           lawrevisionlist.Page
	err            error
	calls          int
	request        lawrevisionlist.Request
	waitForContext bool
}

func (l *recordingLawRevisionLister) List(
	ctx context.Context,
	request lawrevisionlist.Request,
) (lawrevisionlist.Page, error) {
	l.calls++
	l.request = request
	if l.waitForContext {
		<-ctx.Done()
		return lawrevisionlist.Page{}, ctx.Err()
	}
	return l.page, l.err
}

func mustListLawRevisionsPage(
	t *testing.T,
	lawID string,
	revisionIDs ...string,
) lawrevisionlist.Page {
	t.Helper()
	items := make([]model.SourcedResource[model.LawRevision], len(revisionIDs))
	for index, revisionID := range revisionIDs {
		items[index] = mustListLawRevisionResource(t, lawID, revisionID)
	}
	count := len(items)
	sourcePage, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: count,
		TotalCount:    &count,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("SourcePage を作成できません: %v", err)
	}
	page, err := lawrevisionlist.NewPage(lawrevisionlist.PageValues{
		LawID: lawID,
		Items: items,
		Page:  sourcePage,
	})
	if err != nil {
		t.Fatalf("lawrevisionlist.Page を作成できません: %v", err)
	}
	return page
}

func mustListLawRevisionResource(
	t *testing.T,
	lawID string,
	revisionID string,
) model.SourcedResource[model.LawRevision] {
	t.Helper()
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID: "test-source", Name: "試験用情報源", Publisher: "試験機関",
		Authority: model.AuthorityOfficial, ServiceURL: "https://example.test/service",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	legalSource, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("LegalSource を作成できません: %v", err)
	}
	revision, err := model.NewLawRevision(model.LawRevisionValues{
		LawID: lawID, RevisionID: revisionID, Title: "試験法", Source: legalSource,
	})
	if err != nil {
		t.Fatalf("LawRevision を作成できません: %v", err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID: legalSource.ID(), ResourceType: "law", ResourceID: lawID, VersionID: revisionID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{ProviderID: "test-provider", Key: key})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できません: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source: informationSource, ResourceKey: key,
		URL:         "https://example.test/law/" + lawID + "/" + revisionID,
		RetrievedAt: time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC),
		MediaType:   "application/json", Transformation: model.ProvenanceTransformationNormalized,
		MethodID: "SOT-IF-055",
	})
	if err != nil {
		t.Fatalf("Provenance を作成できません: %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.LawRevision]{
		Ref: ref, Provenance: []model.Provenance{provenance}, Data: revision,
	})
	if err != nil {
		t.Fatalf("SourcedResource を作成できません: %v", err)
	}
	return resource
}
