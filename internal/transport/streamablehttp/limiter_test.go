package streamablehttp

import (
	"errors"
	"testing"
)

func TestConcurrencyLimiterTracksOnlyInFlightRequests(t *testing.T) {
	t.Parallel()

	limiter := newConcurrencyLimiter(2)
	releaseFirst, err := limiter.acquire("127.0.0.1", true)
	if err != nil {
		t.Fatalf("一件目を取得できません: %v", err)
	}
	releaseSecond, err := limiter.acquire("127.0.0.1", true)
	if err != nil {
		t.Fatalf("二件目を取得できません: %v", err)
	}
	if _, err := limiter.acquire("127.0.0.1", true); !errors.Is(
		err,
		errTooManyConcurrentRequests,
	) {
		t.Fatalf("三件目のエラー = %v", err)
	}

	releaseFirst()
	releaseFirst()
	releaseThird, err := limiter.acquire("127.0.0.1", true)
	if err != nil {
		t.Fatalf("完了後の取得エラー = %v", err)
	}

	releaseSecond()
	releaseThird()
	if len(limiter.inFlight) != 0 {
		t.Fatalf("完了済みの利用主体が残っています: %#v", limiter.inFlight)
	}
}

func TestConcurrencyLimiterSeparatesRemoteAddresses(t *testing.T) {
	t.Parallel()

	limiter := newConcurrencyLimiter(1)
	releaseFirst, err := limiter.acquire("127.0.0.1", true)
	if err != nil {
		t.Fatalf("一つ目の利用主体を取得できません: %v", err)
	}
	defer releaseFirst()

	releaseSecond, err := limiter.acquire("127.0.0.2", true)
	if err != nil {
		t.Fatalf("別の利用主体を取得できません: %v", err)
	}
	releaseSecond()
}

func TestConcurrencyLimiterIgnoresNonToolRequests(t *testing.T) {
	t.Parallel()

	limiter := newConcurrencyLimiter(1)
	release, err := limiter.acquire("127.0.0.1", false)
	if err != nil {
		t.Fatalf("非 tool request のエラー = %v", err)
	}
	if release != nil {
		t.Fatal("非 tool request に解放関数があります")
	}
	if len(limiter.inFlight) != 0 {
		t.Fatalf("非 tool request の状態が残っています: %#v", limiter.inFlight)
	}
}
