package lawv2

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/requestpacing"
)

func TestLawClientsPaceHTTPStartsAcrossOperationsInOneRequest(t *testing.T) {
	t.Parallel()

	clock := newPacingTestClock()
	starts := make([]time.Time, 0, 2)
	doer := doerFunc(func(request *http.Request) (*http.Response, error) {
		starts = append(starts, clock.Now())
		return response(
			http.StatusOK,
			`{"total_count":0,"count":0,"laws":[]}`,
			map[string]string{"Content-Type": "application/json"},
		), nil
	})
	searchClient := mustTestClient(t, clientDependencies{
		doer: doer, now: clock.Now, sleep: clock.Sleep,
	})
	contentClient := mustTestClient(t, clientDependencies{
		doer: doer, now: clock.Now, sleep: clock.Sleep,
	})
	ctx := requestpacing.WithScope(context.Background())

	if _, err := searchClient.fetch(ctx, lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	}); err != nil {
		t.Fatalf("GET /laws error = %v", err)
	}
	if _, err := contentClient.fetchLawContent(ctx, lawContentSearchRequest{
		keyword: "民法", asOf: mustDate("2026-07-26"), limit: 1,
	}); err != nil {
		t.Fatalf("GET /keyword error = %v", err)
	}

	if len(starts) != 2 || starts[1].Sub(starts[0]) != time.Second {
		t.Fatalf("HTTP start times = %v", starts)
	}
	if delays := clock.Delays(); !sameDurations(delays, []time.Duration{time.Second}) {
		t.Fatalf("pacing delays = %v", delays)
	}
}

func TestLawClientDoesNotAddPacingDelayAfterRetryBackoff(t *testing.T) {
	t.Parallel()

	clock := newPacingTestClock()
	attempts := 0
	starts := make([]time.Time, 0, 2)
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			starts = append(starts, clock.Now())
			attempts++
			if attempts == 1 {
				return response(http.StatusServiceUnavailable, "", nil), nil
			}
			return response(
				http.StatusOK,
				`{"total_count":0,"count":0,"laws":[]}`,
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
		now:   clock.Now,
		sleep: clock.Sleep,
	})
	ctx := requestpacing.WithScope(context.Background())

	if _, err := client.fetch(ctx, lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	}); err != nil {
		t.Fatalf("fetch() error = %v", err)
	}
	if len(starts) != 2 || starts[1].Sub(starts[0]) != time.Second {
		t.Fatalf("retry HTTP start times = %v", starts)
	}
	if delays := clock.Delays(); !sameDurations(delays, []time.Duration{time.Second}) {
		t.Fatalf("retry と pacing が加算されました: delays = %v", delays)
	}
}

func TestLawClientStopsBeforePacingPastRequestDeadline(t *testing.T) {
	t.Parallel()

	clock := newPacingTestClock()
	attempts := 0
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return response(
				http.StatusOK,
				`{"total_count":0,"count":0,"laws":[]}`,
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
		now:   clock.Now,
		sleep: clock.Sleep,
	})
	scope := requestpacing.WithScope(context.Background())
	if _, err := client.fetch(scope, lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	}); err != nil {
		t.Fatalf("最初の fetch() error = %v", err)
	}
	ctx, cancel := context.WithDeadline(
		scope,
		clock.Now().Add(500*time.Millisecond),
	)
	defer cancel()
	_, err := client.fetch(ctx, lawSearchRequest{
		query: "商法", asOf: mustDate("2026-07-26"), limit: 20,
	})
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceTimeout)
	if attempts != 1 {
		t.Fatalf("deadline を越える HTTP attempts = %d, want 1", attempts)
	}
	if delays := clock.Delays(); len(delays) != 0 {
		t.Fatalf("deadline を越えて待機しました: %v", delays)
	}
}

type pacingTestClock struct {
	current time.Time
	delays  []time.Duration
}

func newPacingTestClock() *pacingTestClock {
	return &pacingTestClock{current: time.Now().Round(0)}
}

func (c *pacingTestClock) Now() time.Time {
	return c.current
}

func (c *pacingTestClock) Sleep(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.delays = append(c.delays, delay)
	c.current = c.current.Add(delay)
	return nil
}

func (c *pacingTestClock) Delays() []time.Duration {
	return append([]time.Duration(nil), c.delays...)
}

func sameDurations(left, right []time.Duration) bool {
	return slices.Equal(left, right)
}
