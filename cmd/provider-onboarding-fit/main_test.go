package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/provideronboarding"
)

// SOT-ENG-018: --base-ref は一回だけ必須で、位置引数を受け付けない。
func TestRunRejectsInvalidArgumentsWithUsageExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "missing", args: nil},
		{name: "empty", args: []string{"--base-ref="}},
		{name: "duplicate", args: []string{"--base-ref", "HEAD", "--base-ref", "HEAD~1"}},
		{name: "positional", args: []string{"--base-ref", "HEAD", "extra"}},
		{name: "unknown", args: []string{"--unknown", "value"}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			code := run(
				tt.args,
				&bytes.Buffer{},
				&stderr,
				func(provideronboarding.Options) error {
					t.Fatal("usage error で gate が実行されました")
					return nil
				},
			)
			if code != 2 {
				t.Fatalf("終了コード = %d, want 2", code)
			}
			if !containsJapanese(stderr.String()) {
				t.Fatalf("日本語の usage error ではありません: %q", stderr.String())
			}
		})
	}
}

func TestRunPassesBaseRefAndCurrentSourceState(t *testing.T) {
	t.Parallel()

	var got provideronboarding.Options
	var stdout bytes.Buffer
	code := run(
		[]string{"--base-ref", "main"},
		&stdout,
		&bytes.Buffer{},
		func(options provideronboarding.Options) error {
			got = options
			return nil
		},
	)
	if code != 0 {
		t.Fatalf("終了コード = %d, want 0", code)
	}
	if got.BaseRef != "main" ||
		got.Repository != "." ||
		got.GitRepository != "." ||
		got.HeadRef != "HEAD" ||
		!got.IncludeIndex ||
		!got.IncludeWorkingTree ||
		!got.IncludeUntracked {
		t.Fatalf("実行オプションが一致しません: %#v", got)
	}
	if !strings.Contains(stdout.String(), "成功") {
		t.Fatalf("成功メッセージがありません: %q", stdout.String())
	}
}

func TestRunMapsInvalidRevisionToUsageAndGateFailureToOne(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		code int
	}{
		{
			name: "invalid revision",
			err:  fmt.Errorf("%w: revision", provideronboarding.ErrInvalidBaseRef),
			code: 2,
		},
		{
			name: "gate failure",
			err:  errors.New("検査失敗"),
			code: 1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			code := run(
				[]string{"--base-ref", "bad"},
				&bytes.Buffer{},
				&stderr,
				func(provideronboarding.Options) error { return tt.err },
			)
			if code != tt.code {
				t.Fatalf("終了コード = %d, want %d", code, tt.code)
			}
			if !containsJapanese(stderr.String()) {
				t.Fatalf("日本語のエラーではありません: %q", stderr.String())
			}
		})
	}
}

func containsJapanese(value string) bool {
	for _, character := range value {
		if '\u3040' <= character && character <= '\u30ff' ||
			'\u4e00' <= character && character <= '\u9fff' {
			return true
		}
	}
	return false
}
