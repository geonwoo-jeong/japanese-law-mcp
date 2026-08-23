package lawv2

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawrevisionlist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLawRevisionListAdapterMapsCompleteHistory(t *testing.T) {
	t.Parallel()

	retrievedAt := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	attempts := 0
	adapter := newTestLawRevisionListAdapter(t, retrievedAt, make(chan struct{}, 1),
		doerFunc(func(request *http.Request) (*http.Response, error) {
			attempts++
			if request.Method != http.MethodGet ||
				request.URL.EscapedPath() != "/api/2/law_revisions/%E4%BB%A4%E5%92%8C%E4%B8%89%E5%B9%B4%E6%B3%95%E5%BE%8B%E7%AC%AC%E4%B8%89%E5%8D%81%E5%85%AD%E5%8F%B7" ||
				request.URL.Query().Get("response_format") != "json" ||
				len(request.URL.Query()) != 1 {
				t.Fatalf("e-Gov request = %s %s", request.Method, request.URL.String())
			}
			return response(http.StatusOK, string(fixture(t, "fixtures/law-revisions-normal.json")),
				map[string]string{"Content-Type": "application/json"}), nil
		}),
	)

	request, err := lawrevisionlist.NewRequest(lawrevisionlist.RequestValues{
		LawIDOrNumber: "令和三年法律第三十六号",
	})
	if err != nil {
		t.Fatalf("request を作成できません: %v", err)
	}
	page, err := adapter.List(context.Background(), request)
	if err != nil {
		t.Fatalf("List() のエラー = %v", err)
	}
	items := page.Items()
	if attempts != 1 || page.LawID() != "503AC0000000036" || len(items) != 4 {
		t.Fatalf("page = attempts:%d lawId:%q items:%d", attempts, page.LawID(), len(items))
	}

	first := items[0].Data()
	if first.RevisionID() != "503AC0000000036_20280116_508AC0000000057" ||
		mustRevisionKind(t, first) != model.LawRevisionKindAffectedLaw ||
		mustRevisionCurrentStatus(t, first) != model.LawRevisionCurrentStatusFuture {
		t.Fatalf("先頭履歴 = %#v", first)
	}
	remain, exists := first.RemainInForce()
	if !exists || remain {
		t.Fatalf("remainInForce = %v, %v", remain, exists)
	}
	if value, exists := first.LawNumber(); !exists || value != "令和三年法律第三十六号" {
		t.Fatalf("lawNumber = %q, %v", value, exists)
	}
	if value, exists := first.AmendmentLawTitleKana(); !exists || value == "" {
		t.Fatalf("amendmentLawTitleKana = %q, %v", value, exists)
	}
	if value, exists := first.SourceUpdatedAt(); !exists ||
		value != "2026-07-23T16:29:38+09:00" {
		t.Fatalf("sourceUpdatedAt = %q, %v", value, exists)
	}
	if value, exists := first.ScheduledEffectiveDate(); !exists ||
		value.String() != "2028-01-16" {
		t.Fatalf("scheduledEffectiveDate = %v, %v", value, exists)
	}

	if mustRevisionKind(t, items[1].Data()) != model.LawRevisionKindEnactment ||
		mustRevisionKind(t, items[2].Data()) != model.LawRevisionKindPartialAmendment ||
		mustRevisionKind(t, items[3].Data()) != model.LawRevisionKindRepeal {
		t.Fatal("revisionKind の正規化が一致しません")
	}
	if _, exists := items[1].Data().TitleKana(); exists {
		t.Fatal("null の titleKana が共通値へ残りました")
	}
	if _, exists := items[1].Data().Abbreviation(); exists {
		t.Fatal("空文字の abbreviation が共通値へ残りました")
	}
	if _, exists := items[2].Data().RemainInForce(); exists {
		t.Fatal("null の remainInForce が共通値へ残りました")
	}
	last := items[3].Data()
	if mustRevisionRepealStatus(t, last) != model.LawRevisionRepealStatusExpired ||
		mustRevisionCurrentStatus(t, last) != model.LawRevisionCurrentStatusRepealed {
		t.Fatal("廃止状態の正規化が一致しません")
	}
	if value, exists := last.RepealRecordedDate(); !exists || value.String() != "2027-04-02" {
		t.Fatalf("repealRecordedDate = %v, %v", value, exists)
	}

	for index, item := range items {
		key := item.Ref().Key()
		versionID, versionExists := key.VersionID()
		provenance := item.Provenance()
		if len(provenance) != 1 {
			t.Fatalf("items[%d] の provenance 件数 = %d", index, len(provenance))
		}
		methodID, methodExists := provenance[0].MethodID()
		location, locationExists := provenance[0].Location()
		if item.Ref().ProviderID() != providerID ||
			key.SourceID() != providerID || key.ResourceType() != "law" ||
			key.ResourceID() != page.LawID() || !versionExists ||
			versionID != item.Data().RevisionID() ||
			provenance[0].RetrievedAt() != retrievedAt ||
			provenance[0].URL() !=
				"https://laws.e-gov.go.jp/api/2/law_revisions/%E4%BB%A4%E5%92%8C%E4%B8%89%E5%B9%B4%E6%B3%95%E5%BE%8B%E7%AC%AC%E4%B8%89%E5%8D%81%E5%85%AD%E5%8F%B7?response_format=json" ||
			provenance[0].MediaType() != "application/json" ||
			provenance[0].Transformation() != model.ProvenanceTransformationNormalized ||
			!locationExists || location != fmt.Sprintf("revisions[%d]", index) ||
			!methodExists || methodID != "SOT-IF-057" {
			t.Fatalf("items[%d] の ref/provenance = %#v %#v", index, key, provenance)
		}
	}
	pageInfo := page.Page()
	total, totalExists := pageInfo.TotalCount()
	if !totalExists || total != 4 || pageInfo.ReturnedCount() != 4 {
		t.Fatalf("page info = %#v", pageInfo)
	}
}

func TestLawRevisionListAdapterDistinguishesEmptyAndNotFound(t *testing.T) {
	t.Parallel()

	request := mustLawRevisionListRequest(t, "503AC0000000036")
	empty := newTestLawRevisionListAdapter(t, time.Now(), make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			return response(http.StatusOK, string(fixture(t, "fixtures/law-revisions-empty.json")),
				map[string]string{"Content-Type": "application/json"}), nil
		}),
	)
	page, err := empty.List(context.Background(), request)
	if err != nil || page.LawID() != "503AC0000000036" || len(page.Items()) != 0 {
		t.Fatalf("空一覧 = %#v, %v", page, err)
	}

	missing := newTestLawRevisionListAdapter(t, time.Now(), make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			return response(http.StatusNotFound, `{"code":"404001"}`,
				map[string]string{"Content-Type": "application/json"}), nil
		}),
	)
	if _, err := missing.List(context.Background(), request); !errors.Is(err, lawrevisionlist.ErrNotFound) {
		t.Fatalf("404 error = %v", err)
	}
}

func TestLawRevisionListAdapterAcceptsOnlyHTTP200JSON(t *testing.T) {
	t.Parallel()

	request := mustLawRevisionListRequest(t, "503AC0000000036")
	adapter := newTestLawRevisionListAdapter(t, time.Now(), make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			return response(http.StatusCreated, `{"law_info":{"law_id":"law-1"},"revisions":[]}`,
				map[string]string{"Content-Type": "application/json"}), nil
		}),
	)
	_, err := adapter.List(context.Background(), request)
	assertSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
}

func newTestLawRevisionListAdapter(
	t *testing.T,
	now time.Time,
	gate chan struct{},
	doer httpDoer,
) *LawRevisionListAdapter {
	t.Helper()
	client := mustTestClient(t, clientDependencies{doer: doer, now: func() time.Time { return now }, sleep: sleepWithContext})
	adapter, err := newLawRevisionListAdapter(lawRevisionAdapterDependencies{client: client, gate: gate})
	if err != nil {
		t.Fatalf("adapter を作成できません: %v", err)
	}
	return adapter
}

func mustLawRevisionListRequest(t *testing.T, value string) lawrevisionlist.Request {
	t.Helper()
	request, err := lawrevisionlist.NewRequest(lawrevisionlist.RequestValues{LawIDOrNumber: value})
	if err != nil {
		t.Fatalf("request を作成できません: %v", err)
	}
	return request
}

func mustRevisionKind(t *testing.T, revision model.LawRevision) model.LawRevisionKind {
	t.Helper()
	value, exists := revision.RevisionKind()
	if !exists {
		t.Fatal("revisionKind がありません")
	}
	return value
}

func mustRevisionRepealStatus(t *testing.T, revision model.LawRevision) model.LawRevisionRepealStatus {
	t.Helper()
	value, exists := revision.RepealStatus()
	if !exists {
		t.Fatal("repealStatus がありません")
	}
	return value
}

func mustRevisionCurrentStatus(t *testing.T, revision model.LawRevision) model.LawRevisionCurrentStatus {
	t.Helper()
	value, exists := revision.CurrentStatus()
	if !exists {
		t.Fatal("currentStatus がありません")
	}
	return value
}
