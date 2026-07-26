package streamablehttp

import (
	"errors"
	"sync"
)

var errTooManyConcurrentRequests = errors.New("同時実行数の上限を超えました")

type concurrencyLimiter struct {
	limit int

	mu       sync.Mutex
	inFlight map[string]int
}

func newConcurrencyLimiter(limit int) *concurrencyLimiter {
	return &concurrencyLimiter{
		limit:    limit,
		inFlight: make(map[string]int),
	}
}

func (l *concurrencyLimiter) acquire(identity string, enabled bool) (func(), error) {
	if !enabled {
		return nil, nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.inFlight[identity]
	if current >= l.limit {
		return nil, errTooManyConcurrentRequests
	}
	l.inFlight[identity] = current + 1

	released := false
	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		if released {
			return
		}
		released = true
		remaining := l.inFlight[identity] - 1
		if remaining <= 0 {
			delete(l.inFlight, identity)
			return
		}
		l.inFlight[identity] = remaining
	}, nil
}
