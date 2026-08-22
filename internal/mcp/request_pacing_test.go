package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/requestpacing"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRequestPacingMiddlewareCreatesFreshScopeForEachToolCall(t *testing.T) {
	t.Parallel()

	now := time.Now().Round(0)
	delays := make([]time.Duration, 0, 2)
	sleep := func(ctx context.Context, delay time.Duration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		delays = append(delays, delay)
		now = now.Add(delay)
		return nil
	}
	next := func(
		ctx context.Context,
		_ string,
		_ sdk.Request,
	) (sdk.Result, error) {
		for range 2 {
			if err := requestpacing.RunAtStart(
				ctx,
				"test-group",
				time.Second,
				func() time.Time { return now },
				sleep,
				func() {},
			); err != nil {
				return nil, err
			}
		}
		return &sdk.CallToolResult{}, nil
	}
	handler := requestPacingMiddleware(next)
	for range 2 {
		if _, err := handler(
			context.Background(),
			"tools/call",
			&sdk.CallToolRequest{},
		); err != nil {
			t.Fatalf("tools/call middleware error = %v", err)
		}
	}
	if !samePacingDurations(delays, []time.Duration{time.Second, time.Second}) {
		t.Fatalf("request-local delays = %v", delays)
	}
}

func TestRequestPacingMiddlewareSkipsOtherMethods(t *testing.T) {
	t.Parallel()

	sleeps := 0
	next := func(
		ctx context.Context,
		_ string,
		_ sdk.Request,
	) (sdk.Result, error) {
		for range 2 {
			if err := requestpacing.RunAtStart(
				ctx,
				"test-group",
				time.Second,
				time.Now,
				func(context.Context, time.Duration) error {
					sleeps++
					return nil
				},
				func() {},
			); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	if _, err := requestPacingMiddleware(next)(
		context.Background(),
		"tools/list",
		nil,
	); err != nil {
		t.Fatalf("tools/list middleware error = %v", err)
	}
	if sleeps != 0 {
		t.Fatalf("tools/call 以外の pacing sleeps = %d", sleeps)
	}
}

func samePacingDurations(left, right []time.Duration) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
