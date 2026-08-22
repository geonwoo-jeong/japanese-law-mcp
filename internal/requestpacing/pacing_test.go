package requestpacing

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

func TestRunAtStartKeepsStartsOneIntervalApartWithinScope(t *testing.T) {
	t.Parallel()

	clock := newTestClock()
	ctx := WithScope(context.Background())
	starts := make([]time.Time, 0, 3)
	for range 3 {
		if err := RunAtStart(
			ctx,
			"egov-http",
			time.Second,
			clock.Now,
			clock.Sleep,
			func() { starts = append(starts, clock.Now()) },
		); err != nil {
			t.Fatalf("RunAtStart() error = %v", err)
		}
	}

	wantStarts := []time.Time{
		clock.origin,
		clock.origin.Add(time.Second),
		clock.origin.Add(2 * time.Second),
	}
	if len(starts) != len(wantStarts) {
		t.Fatalf("starts = %v", starts)
	}
	for index := range wantStarts {
		if !starts[index].Equal(wantStarts[index]) {
			t.Fatalf("starts[%d] = %v, want %v", index, starts[index], wantStarts[index])
		}
	}
	if got := clock.Delays(); !equalDurations(got, []time.Duration{
		time.Second,
		time.Second,
	}) {
		t.Fatalf("delays = %v", got)
	}
}

func TestRunAtStartDoesNotShareStateAcrossScopes(t *testing.T) {
	t.Parallel()

	clock := newTestClock()
	first := WithScope(context.Background())
	second := WithScope(context.Background())
	for _, ctx := range []context.Context{first, second} {
		if err := RunAtStart(
			ctx,
			"egov-http",
			time.Second,
			clock.Now,
			clock.Sleep,
			func() {},
		); err != nil {
			t.Fatalf("RunAtStart() error = %v", err)
		}
	}
	if delays := clock.Delays(); len(delays) != 0 {
		t.Fatalf("別 scope の初回開始で待機しました: %v", delays)
	}
}

func TestRunAtStartDoesNotAdvanceStateWhenDeadlineCannotFitWait(t *testing.T) {
	t.Parallel()

	clock := newTestClock()
	scope := WithScope(context.Background())
	if err := RunAtStart(
		scope,
		"egov-http",
		time.Second,
		clock.Now,
		clock.Sleep,
		func() {},
	); err != nil {
		t.Fatalf("最初の RunAtStart() error = %v", err)
	}
	deadlineContext, cancel := context.WithDeadline(
		scope,
		clock.Now().Add(500*time.Millisecond),
	)
	defer cancel()
	called := false
	err := RunAtStart(
		deadlineContext,
		"egov-http",
		time.Second,
		clock.Now,
		clock.Sleep,
		func() { called = true },
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline を越える RunAtStart() error = %v", err)
	}
	if called {
		t.Fatal("deadline を越える開始処理が実行されました")
	}
	if err := RunAtStart(
		scope,
		"egov-http",
		time.Second,
		clock.Now,
		clock.Sleep,
		func() {},
	); err != nil {
		t.Fatalf("後続の RunAtStart() error = %v", err)
	}
	if got := clock.Delays(); !equalDurations(got, []time.Duration{time.Second}) {
		t.Fatalf("失敗した開始が状態を進めました: delays = %v", got)
	}
}

func TestRunAtStartDoesNotAdvanceStateWhenWaitIsCanceled(t *testing.T) {
	t.Parallel()

	clock := newTestClock()
	scope := WithScope(context.Background())
	if err := RunAtStart(
		scope,
		"egov-http",
		time.Second,
		clock.Now,
		clock.Sleep,
		func() {},
	); err != nil {
		t.Fatalf("最初の RunAtStart() error = %v", err)
	}
	called := false
	err := RunAtStart(
		scope,
		"egov-http",
		time.Second,
		clock.Now,
		func(context.Context, time.Duration) error {
			return context.Canceled
		},
		func() { called = true },
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("中断した RunAtStart() error = %v", err)
	}
	if called {
		t.Fatal("中断した開始処理が実行されました")
	}
	if err := RunAtStart(
		scope,
		"egov-http",
		time.Second,
		clock.Now,
		clock.Sleep,
		func() {},
	); err != nil {
		t.Fatalf("後続の RunAtStart() error = %v", err)
	}
	if got := clock.Delays(); !equalDurations(got, []time.Duration{time.Second}) {
		t.Fatalf("中断した開始が状態を進めました: delays = %v", got)
	}
}

func TestRunAtStartCancellationDoesNotWaitForRunningStart(t *testing.T) {
	t.Parallel()

	clock := newTestClock()
	scope := WithScope(context.Background())
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- RunAtStart(
			scope,
			"egov-http",
			time.Second,
			clock.Now,
			clock.Sleep,
			func() {
				close(firstStarted)
				<-releaseFirst
			},
		)
	}()
	<-firstStarted

	waitingContext, cancel := context.WithCancel(scope)
	observed := make(chan struct{}, 1)
	waitingContext = &valueObservedContext{
		Context:  waitingContext,
		observed: observed,
	}
	secondCalled := false
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- RunAtStart(
			waitingContext,
			"egov-http",
			time.Second,
			clock.Now,
			clock.Sleep,
			func() { secondCalled = true },
		)
	}()
	<-observed
	cancel()

	select {
	case err := <-secondDone:
		close(releaseFirst)
		if firstErr := <-firstDone; firstErr != nil {
			t.Fatalf("先行 RunAtStart() error = %v", firstErr)
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("待機中断の RunAtStart() error = %v", err)
		}
		if secondCalled {
			t.Fatal("待機を中断した開始処理が実行されました")
		}
	case <-time.After(250 * time.Millisecond):
		close(releaseFirst)
		<-firstDone
		<-secondDone
		t.Fatal("後続処理が先行 start の完了まで中断を返しませんでした")
	}
}

func TestRunAtStartWithoutScopeIsImmediate(t *testing.T) {
	t.Parallel()

	clock := newTestClock()
	starts := 0
	for range 2 {
		if err := RunAtStart(
			context.Background(),
			"egov-http",
			time.Second,
			clock.Now,
			clock.Sleep,
			func() { starts++ },
		); err != nil {
			t.Fatalf("RunAtStart() error = %v", err)
		}
	}
	if starts != 2 {
		t.Fatalf("starts = %d, want 2", starts)
	}
	if delays := clock.Delays(); len(delays) != 0 {
		t.Fatalf("scope がない処理で待機しました: %v", delays)
	}
}

type valueObservedContext struct {
	context.Context
	observed chan<- struct{}
}

func (c *valueObservedContext) Value(key any) any {
	value := c.Context.Value(key)
	select {
	case c.observed <- struct{}{}:
	default:
	}
	return value
}

type testClock struct {
	origin  time.Time
	current time.Time
	delays  []time.Duration
}

func newTestClock() *testClock {
	origin := time.Now().Round(0)
	return &testClock{origin: origin, current: origin}
}

func (c *testClock) Now() time.Time {
	return c.current
}

func (c *testClock) Sleep(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.delays = append(c.delays, delay)
	c.current = c.current.Add(delay)
	return nil
}

func (c *testClock) Delays() []time.Duration {
	return append([]time.Duration(nil), c.delays...)
}

func equalDurations(left, right []time.Duration) bool {
	return slices.Equal(left, right)
}
