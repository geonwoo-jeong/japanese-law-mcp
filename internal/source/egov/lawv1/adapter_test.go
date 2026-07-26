package lawv1

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestListは公式五件fixtureを損失なく対応する(t *testing.T) {
	t.Parallel()

	retrievedAt := time.Date(2026, 7, 26, 3, 4, 5, 0, time.UTC)
	adapter := mustTestAdapter(t, func(*http.Request) (*http.Response, error) {
		return testResponse(
			http.StatusOK,
			fixture(t, "law-update-list-v1-normal.xml"),
			map[string]string{"Content-Type": "text/xml;charset=UTF-8"},
		), nil
	}, retrievedAt, make(chan struct{}, 1))

	page, err := adapter.List(
		context.Background(),
		mustRequest(t, "2023-02-01"),
	)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.Date().String() != "2023-02-01" {
		t.Fatalf("date = %q", page.Date())
	}
	items := page.Items()
	if len(items) != 5 || page.Page().ReturnedCount() != 5 {
		t.Fatalf("items = %d, returnedCount = %d", len(items), page.Page().ReturnedCount())
	}
	total, exists := page.Page().TotalCount()
	if !exists || total != 5 {
		t.Fatalf("totalCount = %d, %t", total, exists)
	}
	if relation, exists := page.Page().TotalRelation(); !exists ||
		relation != model.TotalRelationExact {
		t.Fatalf("totalRelation = %q, %t", relation, exists)
	}

	first := items[0]
	data := first.Data()
	if data.LawID() != "421CO0000000220" ||
		data.Title() != "消費者安全法施行令" {
		t.Fatalf("first data = %q, %q", data.LawID(), data.Title())
	}
	assertOptionalString(t, data.LawType, "政令")
	assertOptionalString(t, data.LawNumber, "平成二十一年政令第二百二十号")
	assertOptionalString(t, data.TitleKana, "しょうひしゃあんぜんほうしこうれい")
	assertOptionalString(t, data.PreviousTitle, "")
	assertOptionalDate(t, data.PromulgationDate, "2009-08-14")
	assertOptionalString(t, data.AmendmentLawNumber, "令和五年政令第五号")
	assertOptionalDate(t, data.AmendmentPromulgationDate, "2023-01-18")
	assertOptionalDate(t, data.EffectiveDate, "2023-06-01")
	assertOptionalString(t, data.EffectiveDateNote, "")
	assertOptionalString(
		t,
		data.DocumentURL,
		"https://elaws.e-gov.go.jp/document?lawid=421CO0000000220_20230601_505CO0000000005",
	)
	assertOptionalBool(t, data.EnforcementPending, true)
	assertOptionalBool(t, data.AuthorityReviewPending, false)

	ref := first.Ref()
	if ref.ProviderID() != providerID ||
		ref.Key().SourceID() != providerID ||
		ref.Key().ResourceType() != "law-update-list" ||
		ref.Key().ResourceID() != "2023-02-01" {
		t.Fatalf("ref = %#v", ref)
	}
	if _, exists := ref.Key().VersionID(); exists {
		t.Fatal("versionId が設定されました")
	}
	provenance := first.Provenance()
	if len(provenance) != 1 ||
		provenance[0].MediaType() != "text/xml" ||
		provenance[0].Transformation() != model.ProvenanceTransformationNormalized {
		t.Fatalf("provenance = %#v", provenance)
	}
	if method, exists := provenance[0].MethodID(); !exists || method != "SOT-IF-036" {
		t.Fatalf("methodId = %q, %t", method, exists)
	}
	if provenance[0].RetrievedAt() != retrievedAt {
		t.Fatalf("retrievedAt = %v", provenance[0].RetrievedAt())
	}
}

func TestListは404Code1の該当なしを空一覧にする(t *testing.T) {
	t.Parallel()

	adapter := mustTestAdapter(t, func(*http.Request) (*http.Response, error) {
		return testResponse(
			http.StatusNotFound,
			fixture(t, "law-update-list-v1-empty.xml"),
			map[string]string{"Content-Type": "text/xml;charset=UTF-8"},
		), nil
	}, time.Now(), make(chan struct{}, 1))
	page, err := adapter.List(
		context.Background(),
		mustRequest(t, "2023-02-02"),
	)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Items()) != 0 || page.Page().ReturnedCount() != 0 {
		t.Fatalf("empty page = %#v", page)
	}
}

func TestListは提供範囲外をHTTP前に拒否する(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	now := time.Date(2026, 7, 26, 16, 0, 0, 0, time.UTC)
	adapter := mustTestAdapter(t, func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("到達してはいけません")
	}, now, make(chan struct{}, 1))
	for _, date := range []string{"2020-11-23", "2026-07-28"} {
		_, err := adapter.List(context.Background(), mustRequest(t, date))
		assertSourceErrorCode(t, err, model.SourceErrorCodeUnsupportedQuery)
	}
	if calls.Load() != 0 {
		t.Fatalf("HTTP calls = %d", calls.Load())
	}
}

func TestListは応答値の不整合を成功にしない(t *testing.T) {
	t.Parallel()

	base := fixture(t, "law-update-list-v1-normal.xml")
	tests := map[string]string{
		"対象日不一致":        strings.Replace(base, "<Date>20230201</Date>", "<Date>20230202</Date>", 1),
		"法令ID欠落":        strings.Replace(base, "<LawId>421CO0000000220</LawId>", "<LawId/>", 1),
		"日付不正":          strings.Replace(base, "<PromulgationDate>20090814</PromulgationDate>", "<PromulgationDate>20090230</PromulgationDate>", 1),
		"URL不正":         strings.Replace(base, "https://elaws.e-gov.go.jp/document?", "http://elaws.e-gov.go.jp/document?", 1),
		"flag不正":        strings.Replace(base, "<EnforcementFlg>1</EnforcementFlg>", "<EnforcementFlg>2</EnforcementFlg>", 1),
		"result code不正": strings.Replace(base, "<Code>0</Code>", "<Code>1</Code>", 1),
	}
	for name, body := range tests {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			adapter := mustTestAdapter(t, func(*http.Request) (*http.Response, error) {
				return testResponse(
					http.StatusOK,
					body,
					map[string]string{"Content-Type": "text/xml;charset=UTF-8"},
				), nil
			}, time.Now(), make(chan struct{}, 1))
			_, err := adapter.List(
				context.Background(),
				mustRequest(t, "2023-02-01"),
			)
			assertSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestListは同時実行上限と取消を守る(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	adapter := mustTestAdapter(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("上限超過時に HTTP へ到達しました")
		return nil, nil
	}, time.Now(), gate)
	_, err := adapter.List(
		context.Background(),
		mustRequest(t, "2023-02-01"),
	)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceBusy)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	emptyGate := make(chan struct{}, 1)
	adapter = mustTestAdapter(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("取消後に HTTP へ到達しました")
		return nil, nil
	}, time.Now(), emptyGate)
	_, err = adapter.List(ctx, mustRequest(t, "2023-02-01"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v、context.Canceled ではありません", err)
	}
}

func TestAdapterは不正な依存関係とnilContextを拒否する(t *testing.T) {
	t.Parallel()

	if _, err := newLawUpdateListAdapter(adapterDependencies{}); err == nil {
		t.Fatal("依存関係のない adapter を受理しました")
	}
	adapter := mustTestAdapter(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("nil context で HTTP へ到達しました")
		return nil, nil
	}, time.Now(), make(chan struct{}, 1))
	//nolint:staticcheck // SOT-IF-015: nil context を拒否する境界契約を直接確認する。
	if _, err := adapter.List(nil, mustRequest(t, "2023-02-01")); err == nil {
		t.Fatal("nil context を受理しました")
	}
}

func mustRequest(t *testing.T, value string) lawupdatelist.Request {
	t.Helper()
	request, err := lawupdatelist.NewRequest(lawupdatelist.RequestValues{
		Date: mustTestDate(t, value),
	})
	if err != nil {
		t.Fatalf("request を作成できません: %v", err)
	}
	return request
}

func mustTestAdapter(
	t *testing.T,
	doer doerFunc,
	now time.Time,
	gate chan struct{},
) *LawUpdateListAdapter {
	t.Helper()
	client := mustTestClient(t, clientDependencies{
		doer:  doer,
		now:   func() time.Time { return now },
		sleep: noSleep,
	})
	adapter, err := newLawUpdateListAdapter(adapterDependencies{
		client: client,
		now:    func() time.Time { return now },
		gate:   gate,
	})
	if err != nil {
		t.Fatalf("adapter を作成できません: %v", err)
	}
	return adapter
}

func assertOptionalString(
	t *testing.T,
	getter func() (string, bool),
	want string,
) {
	t.Helper()
	value, exists := getter()
	if value != want || exists != (want != "") {
		t.Fatalf("optional string = %q, %t、期待値は %q", value, exists, want)
	}
}

func assertOptionalDate(
	t *testing.T,
	getter func() (model.Date, bool),
	want string,
) {
	t.Helper()
	value, exists := getter()
	if !exists || value.String() != want {
		t.Fatalf("optional date = %q, %t、期待値は %q", value, exists, want)
	}
}

func assertOptionalBool(
	t *testing.T,
	getter func() (bool, bool),
	want bool,
) {
	t.Helper()
	value, exists := getter()
	if !exists || value != want {
		t.Fatalf("optional bool = %t, %t、期待値は %t", value, exists, want)
	}
}
