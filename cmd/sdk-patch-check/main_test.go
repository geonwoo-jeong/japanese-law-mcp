package main

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestSDK照合commandの引数と終了値(t *testing.T) {
	for _, current := range []struct {
		args    []string
		failure bool
		code    int
		calls   int
	}{
		{args: []string{"--repository=fixture"}, code: 0, calls: 1},
		{args: []string{"--repository=fixture"}, failure: true, code: 1, calls: 1},
		{args: []string{"--help"}, code: 0},
		{args: []string{"--unknown"}, code: 2},
		{args: []string{"--repository="}, code: 2},
		{args: []string{"extra"}, code: 2},
	} {
		var stdout, stderr bytes.Buffer
		calls := 0
		code := run(t.Context(), current.args, &stdout, &stderr, func(ctx context.Context, repository string) error {
			calls++
			if ctx != t.Context() || repository != "fixture" {
				t.Fatal("SOT-ENG-014: 検証対象と context の委譲が不正です")
			}
			if current.failure {
				return errors.New("合成した失敗")
			}
			return nil
		})
		if code != current.code || calls != current.calls {
			t.Fatalf("SOT-ENG-014: code=%d calls=%d", code, calls)
		}
		if code != 0 && stderr.Len() == 0 {
			t.Fatal("失敗理由がありません")
		}
	}
}
