// Package requestpacing は、一つの公開リクエスト内だけで共有する開始間隔を管理する。
package requestpacing

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type scopeContextKey struct{}

type scope struct {
	mu     sync.Mutex
	groups map[string]*groupState
}

type groupState struct {
	turn          chan struct{}
	lastStartedAt time.Time
	started       bool
}

// WithScope は、親 context の値と中断を保った新しいリクエスト単位の状態を返す。
func WithScope(parent context.Context) context.Context {
	return context.WithValue(parent, scopeContextKey{}, &scope{
		groups: make(map[string]*groupState),
	})
}

// RunAtStart は、同じ scope と group の実開始間隔を満たした直後に start を実行する。
// scope がない内部呼出しでは、待機状態をプロセス全体へ拡張せず start を直ちに実行する。
func RunAtStart(
	ctx context.Context,
	group string,
	minimumInterval time.Duration,
	now func() time.Time,
	sleep func(context.Context, time.Duration) error,
	start func(),
) error {
	if ctx == nil {
		return fmt.Errorf("context は必須です")
	}
	if group == "" {
		return fmt.Errorf("開始間隔グループは必須です")
	}
	if minimumInterval <= 0 {
		return fmt.Errorf("minimumInterval は 0 秒より長くなければなりません")
	}
	if now == nil || sleep == nil || start == nil {
		return fmt.Errorf("リクエスト開始間隔管理の依存関係が不足しています")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	requestScope, ok := ctx.Value(scopeContextKey{}).(*scope)
	if !ok || requestScope == nil {
		start()
		return nil
	}
	state := requestScope.stateFor(group)
	if err := state.acquire(ctx); err != nil {
		return err
	}
	defer state.release()
	if err := ctx.Err(); err != nil {
		return err
	}
	current := now()
	if state.started {
		delay := state.lastStartedAt.Add(minimumInterval).Sub(current)
		if delay > 0 {
			if deadline, exists := ctx.Deadline(); exists &&
				current.Add(delay).After(deadline) {
				return context.DeadlineExceeded
			}
			if err := sleep(ctx, delay); err != nil {
				return err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	state.lastStartedAt = now()
	state.started = true
	start()
	return nil
}

func (s *scope) stateFor(group string) *groupState {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, exists := s.groups[group]
	if exists {
		return state
	}
	state = &groupState{turn: make(chan struct{}, 1)}
	state.turn <- struct{}{}
	s.groups[group] = state
	return state
}

func (s *groupState) acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.turn:
		return nil
	}
}

func (s *groupState) release() {
	s.turn <- struct{}{}
}
