package kokkai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestSpeechSearchAdapterSearchMapsResponseAndHoldsGate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	delays := make([]time.Duration, 0, 1)
	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(*http.Request) (*http.Response, error) {
			return speechSearchJSONResponse(
				`{"numberOfRecords":1,"numberOfReturn":1,"startRecord":1,"speechRecord":[{"speechID":"1","speechOrder":0,"speaker":"山田太郎","speech":"発言本文","speechURL":"https://kokkai.ndl.go.jp/txt/1","issueID":"100","imageKind":"会議録","session":1,"nameOfHouse":"参議院","nameOfMeeting":"法務委員会","issue":"1","date":"2024-01-01","meetingURL":"https://kokkai.ndl.go.jp/txt/100"}]}`,
			), nil
		}),
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := newSpeechSearchAdapter(speechSearchAdapterDependencies{
		client: client,
		now:    func() time.Time { return now },
		sleep: func(context.Context, time.Duration) error {
			delays = append(delays, speechSearchGateHold)
			now = now.Add(speechSearchGateHold)
			return nil
		},
		gate: make(chan struct{}, 1),
	})
	if err != nil {
		t.Fatalf("newSpeechSearchAdapter() のエラー = %v", err)
	}

	request := mustSpeechSearchRequest(t, parliamentspeechsearch.RequestValues{Query: "永住許可"})
	page, err := adapter.Search(context.Background(), request)
	if err != nil {
		t.Fatalf("Search() のエラー = %v", err)
	}
	if len(delays) != 1 || delays[0] != speechSearchGateHold {
		t.Fatalf("gate hold delays = %v", delays)
	}
	if len(page.Items()) != 1 || page.Page().ReturnedCount() != 1 {
		t.Fatalf("page = %#v", page)
	}
}

func TestSpeechSearchAdapterRejectsBusyBeforeFetch(t *testing.T) {
	t.Parallel()

	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("busy 状態で外部呼出しした")
			return nil, nil
		}),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := newSpeechSearchAdapter(speechSearchAdapterDependencies{
		client: client,
		now:    time.Now,
		sleep:  sleepSpeechSearchWithContext,
		gate:   gate,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Search(
		context.Background(),
		mustSpeechSearchRequest(t, parliamentspeechsearch.RequestValues{Query: "永住許可"}),
	)
	assertSpeechSearchSourceError(t, err, model.SourceErrorCodeSourceBusy)
}

func TestSpeechSearchAdapterReturnsCanceledDuringGateHold(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return speechSearchJSONResponse(
				`{"numberOfRecords":0,"numberOfReturn":0,"startRecord":1}`,
			), nil
		}),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	adapter, err := newSpeechSearchAdapter(speechSearchAdapterDependencies{
		client: client,
		now:    time.Now,
		sleep: func(context.Context, time.Duration) error {
			cancel()
			return ctx.Err()
		},
		gate: make(chan struct{}, 1),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Search(
		ctx,
		mustSpeechSearchRequest(t, parliamentspeechsearch.RequestValues{Query: "永住許可"}),
	)
	if err == nil || err != context.Canceled {
		t.Fatalf("cancel during gate hold = %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("外部呼出し回数 = %d", calls.Load())
	}
}

type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func speechSearchJSONResponse(body string) *http.Response {
	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	response.Header.Set("Content-Type", "application/json")
	response.ContentLength = int64(len(body))
	return response
}

func mustSpeechSearchRequest(
	t *testing.T,
	values parliamentspeechsearch.RequestValues,
) parliamentspeechsearch.Request {
	t.Helper()
	request, err := parliamentspeechsearch.NewRequest(values)
	if err != nil {
		t.Fatalf("request を作成できません: %v", err)
	}
	return request
}
