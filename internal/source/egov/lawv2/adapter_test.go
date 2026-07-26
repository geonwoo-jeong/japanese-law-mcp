package lawv2

import (
	"context"
	"embed"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/continuation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

//go:embed fixtures/*.json
var fixtureFiles embed.FS

func TestDescriptorKeepsFixedCapabilityOrder(t *testing.T) {
	t.Parallel()

	descriptor := Descriptor()
	if descriptor.ProviderID() != providerID {
		t.Fatalf("ProviderID = %q", descriptor.ProviderID())
	}
	capabilities := descriptor.Capabilities()
	expected := []string{
		"law.article.read",
		"law.content.search",
		"law.document.read",
		"law.search",
	}
	if len(capabilities) != len(expected) {
		t.Fatalf("capabilities = %d", len(capabilities))
	}
	for index, capability := range capabilities {
		if capability.ID() != expected[index] ||
			capability.MajorVersion() != 1 ||
			capability.Level() != model.CapabilityLevelCore ||
			capability.Stability() != model.CapabilityStabilityStable {
			t.Errorf("capabilities[%d] = %#v", index, capability)
		}
	}
}

func TestProductionAdapterCanBeConstructedWithoutRegistration(t *testing.T) {
	t.Parallel()

	manager, err := continuation.NewManager()
	if err != nil {
		t.Fatalf("continuation.NewManager() error = %v", err)
	}
	adapter, err := NewLawSearchAdapter(manager)
	if err != nil {
		t.Fatalf("NewLawSearchAdapter() error = %v", err)
	}
	if adapter == nil {
		t.Fatal("adapter is nil")
	}
	if _, err := NewLawSearchAdapter(nil); err == nil {
		t.Fatal("nil manager was accepted")
	}
	if err := operation("unknown").ValidateSourceOperation(); err == nil {
		t.Fatal("unknown operation was accepted")
	}
}

func TestLawSearchAdapterMapsResultAndContinuesWithSnapshot(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC)
	first := fixture(t, "fixtures/law-search-normal.json")
	second := []byte(`{
		"total_count": 2,
		"count": 1,
		"next_offset": null,
		"laws": [{
			"law_info": {"law_id":"129AC0000000089"},
			"revision_info": {
				"law_revision_id":"129AC0000000089_20260401_000000000000000",
				"law_title":"民法"
			}
		}]
	}`)

	var mutex sync.Mutex
	var requests []*http.Request
	call := 0
	adapter := newTestAdapter(t, now, make(chan struct{}, 1), doerFunc(
		func(request *http.Request) (*http.Response, error) {
			mutex.Lock()
			defer mutex.Unlock()
			requests = append(requests, request.Clone(request.Context()))
			body := first
			if call == 1 {
				body = second
			}
			call++
			return response(
				http.StatusOK,
				string(body),
				map[string]string{"Content-Type": "application/json"},
			), nil
		},
	))

	limit := 1
	request := mustSearchRequest(t, lawsearch.RequestValues{
		Query: "地方自治", Limit: &limit,
	})
	page, err := adapter.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("Search(first) error = %v", err)
	}
	items := page.Items()
	if len(items) != 1 {
		t.Fatalf("items = %d", len(items))
	}
	item := items[0]
	if item.Data().LawID() != "322CO0000000016" ||
		item.Ref().ProviderID() != providerID ||
		item.Ref().Key().ResourceID() != item.Data().LawID() {
		t.Fatalf("mapped item = %#v", item)
	}
	provenance := item.Provenance()
	if len(provenance) != 1 ||
		provenance[0].URL() !=
			"https://laws.e-gov.go.jp/law/322CO0000000016/20240401_506CO0000000161" {
		t.Fatalf("provenance = %#v", provenance)
	}
	token, exists := page.Page().NextToken()
	if !exists {
		t.Fatal("nextToken is missing")
	}

	resume := mustSearchRequest(t, lawsearch.RequestValues{
		Query: "地方自治", Limit: &limit, ContinuationToken: token,
	})
	lastPage, err := adapter.Search(context.Background(), resume)
	if err != nil {
		t.Fatalf("Search(resume) error = %v", err)
	}
	if _, exists := lastPage.Page().NextToken(); exists {
		t.Fatal("last page must not contain nextToken")
	}

	mutex.Lock()
	defer mutex.Unlock()
	if len(requests) != 2 {
		t.Fatalf("requests = %d", len(requests))
	}
	if got := requests[0].URL.Query().Get("asof"); got != "2026-07-26" {
		t.Errorf("first asof = %q", got)
	}
	if got := requests[1].URL.Query().Get("asof"); got != "2026-07-26" {
		t.Errorf("resume asof = %q", got)
	}
	if got := requests[1].URL.Query().Get("offset"); got != "1" {
		t.Errorf("resume offset = %q", got)
	}
}

func TestLawSearchAdapterRejectsContinuationConditionMismatchBeforeHTTP(
	t *testing.T,
) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	attempts := 0
	adapter := newTestAdapter(t, now, make(chan struct{}, 1), doerFunc(
		func(*http.Request) (*http.Response, error) {
			attempts++
			return response(
				http.StatusOK,
				string(fixture(t, "fixtures/law-search-normal.json")),
				map[string]string{"Content-Type": "application/json"},
			), nil
		},
	))
	limit := 1
	initial := mustSearchRequest(t, lawsearch.RequestValues{
		Query: "地方自治", Limit: &limit,
	})
	page, err := adapter.Search(context.Background(), initial)
	if err != nil {
		t.Fatalf("Search(initial) error = %v", err)
	}
	token, _ := page.Page().NextToken()
	mismatch := mustSearchRequest(t, lawsearch.RequestValues{
		Query: "民法", Limit: &limit, ContinuationToken: token,
	})
	_, err = adapter.Search(context.Background(), mismatch)
	if !errors.Is(err, continuation.ErrInvalidToken) {
		t.Fatalf("error = %v, want ErrInvalidToken", err)
	}
	if attempts != 1 {
		t.Fatalf("HTTP attempts = %d, want 1", attempts)
	}
}

func TestLawSearchAdapterRejectsUnsupportedQueryBeforeHTTP(t *testing.T) {
	t.Parallel()

	attempts := 0
	adapter := newTestAdapter(
		t,
		time.Now(),
		make(chan struct{}, 1),
		doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("unexpected HTTP")
		}),
	)
	oldDate := mustDate("2017-03-31")
	cases := []lawsearch.RequestValues{
		{Query: "/民法/"},
		{Query: "民法", AsOf: &oldDate},
	}
	for _, values := range cases {
		request := mustSearchRequest(t, values)
		_, err := adapter.Search(context.Background(), request)
		assertSourceErrorCode(t, err, model.SourceErrorCodeUnsupportedQuery)
	}
	if attempts != 0 {
		t.Fatalf("HTTP attempts = %d", attempts)
	}
}

func TestLawSearchAdapterReturnsBusyBeforeHTTP(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	attempts := 0
	adapter := newTestAdapter(t, time.Now(), gate, doerFunc(
		func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("unexpected HTTP")
		},
	))
	request := mustSearchRequest(t, lawsearch.RequestValues{Query: "民法"})

	_, err := adapter.Search(context.Background(), request)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceBusy)
	if attempts != 0 {
		t.Fatalf("HTTP attempts = %d", attempts)
	}
}

func TestLawSearchAdapterHoldsConcurrencySlotThroughRequest(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	adapter := newTestAdapter(t, time.Now(), gate, doerFunc(
		func(*http.Request) (*http.Response, error) {
			once.Do(func() { close(entered) })
			<-release
			return response(
				http.StatusOK,
				string(fixture(t, "fixtures/law-search-empty.json")),
				map[string]string{"Content-Type": "application/json"},
			), nil
		},
	))
	request := mustSearchRequest(t, lawsearch.RequestValues{Query: "民法"})
	firstDone := make(chan error, 1)
	go func() {
		_, err := adapter.Search(context.Background(), request)
		firstDone <- err
	}()
	<-entered

	_, err := adapter.Search(context.Background(), request)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceBusy)
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first Search() error = %v", err)
	}
}

func TestLawSearchAdapterPropagatesCancellationBeforeHTTP(t *testing.T) {
	t.Parallel()

	attempts := 0
	adapter := newTestAdapter(t, time.Now(), make(chan struct{}, 1), doerFunc(
		func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("unexpected HTTP")
		},
	))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := mustSearchRequest(t, lawsearch.RequestValues{Query: "民法"})

	_, err := adapter.Search(ctx, request)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if attempts != 0 {
		t.Fatalf("HTTP attempts = %d", attempts)
	}
}

func newTestAdapter(
	t *testing.T,
	now time.Time,
	gate chan struct{},
	doer httpDoer,
) *LawSearchAdapter {
	t.Helper()
	manager, err := continuation.NewManager()
	if err != nil {
		t.Fatalf("continuation.NewManager() error = %v", err)
	}
	client := mustTestClient(t, clientDependencies{
		doer: doer,
		now:  func() time.Time { return now },
		sleep: func(context.Context, time.Duration) error {
			return nil
		},
	})
	adapter, err := newLawSearchAdapter(manager, adapterDependencies{
		client: client,
		now:    func() time.Time { return now },
		gate:   gate,
	})
	if err != nil {
		t.Fatalf("newLawSearchAdapter() error = %v", err)
	}
	return adapter
}

func mustSearchRequest(
	t *testing.T,
	values lawsearch.RequestValues,
) lawsearch.Request {
	t.Helper()
	request, err := lawsearch.NewRequest(values)
	if err != nil {
		t.Fatalf("lawsearch.NewRequest() error = %v", err)
	}
	return request
}

func fixture(t *testing.T, path string) []byte {
	t.Helper()
	value, err := fixtureFiles.ReadFile(path)
	if err != nil {
		t.Fatalf("fixture %q: %v", path, err)
	}
	return value
}
