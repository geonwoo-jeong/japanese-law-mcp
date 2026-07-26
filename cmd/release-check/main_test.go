package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/releasecheck"
)

func TestExecute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args       []string
		checkError error
		wantCode   int
		wantOutput string
	}{
		"成功": {
			args: []string{
				"--release-notes", "notes.md",
				"--tag", "v1.2.3",
				"--repository", ".",
			},
			wantCode:   0,
			wantOutput: "リリース検証に成功しました",
		},
		"help": {
			args:       []string{"--help"},
			wantCode:   0,
			wantOutput: "使用方法",
		},
		"必須フラグなし": {
			args:       []string{"--tag", "v1.2.3"},
			wantCode:   2,
			wantOutput: "必須",
		},
		"未知のフラグ": {
			args:       []string{"--unknown"},
			wantCode:   2,
			wantOutput: "コマンドライン引数",
		},
		"位置引数": {
			args: []string{
				"--release-notes", "notes.md",
				"--tag", "v1.2.3",
				"--repository", ".",
				"extra",
			},
			wantCode:   2,
			wantOutput: "位置引数",
		},
		"検証失敗": {
			args: []string{
				"--release-notes", "notes.md",
				"--tag", "v1.2.3",
				"--repository", ".",
			},
			checkError: errors.New("検証できません"),
			wantCode:   1,
			wantOutput: "検証できません",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			checker := func(context.Context, releasecheck.Request) error {
				return test.checkError
			}
			code := execute(
				context.Background(),
				append([]string(nil), test.args...),
				&stdout,
				&stderr,
				checker,
			)
			combined := stdout.String() + stderr.String()
			if code != test.wantCode {
				t.Fatalf("execute() = %d, want %d: %s", code, test.wantCode, combined)
			}
			if !strings.Contains(combined, test.wantOutput) {
				t.Fatalf("出力 = %q, want %q", combined, test.wantOutput)
			}
		})
	}
}

func TestExecuteCopiesArgumentsIntoRequest(t *testing.T) {
	t.Parallel()

	args := []string{
		"--release-notes", "notes.md",
		"--tag", "v1.2.3",
		"--repository", ".",
		"--dist", "dist",
		"--commit", testCommit,
		"--target-os", "darwin",
		"--target-arch", "arm64",
	}
	var got releasecheck.Request
	code := execute(
		context.Background(),
		args,
		&bytes.Buffer{},
		&bytes.Buffer{},
		func(_ context.Context, request releasecheck.Request) error {
			got = request
			return nil
		},
	)
	if code != 0 {
		t.Fatalf("execute() = %d", code)
	}
	args[1] = "changed.md"
	if got.ReleaseNotes != "notes.md" ||
		got.Repository != "." ||
		got.Dist != "dist" ||
		got.Commit != testCommit ||
		got.TargetOS != "darwin" ||
		got.TargetArch != "arm64" {
		t.Fatalf("Request = %#v", got)
	}
}

const testCommit = "0123456789abcdef0123456789abcdef01234567"
