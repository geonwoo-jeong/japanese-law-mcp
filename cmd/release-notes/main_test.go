package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestExecute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		args        []string
		buildError  error
		wantCode    int
		wantOutput  string
		wantBuilder bool
	}{
		"成功": {
			args: []string{
				"--changelog", "CHANGELOG.md",
				"--release-notes", "release-notes/CURRENT.md",
				"--tag", "v1.2.3",
				"--repository", ".",
			},
			wantCode:    exitSuccess,
			wantOutput:  "公開用リリース情報\n",
			wantBuilder: true,
		},
		"help": {
			args:       []string{"--help"},
			wantCode:   exitSuccess,
			wantOutput: "使用方法",
		},
		"必須フラグなし": {
			args:       []string{"--tag", "v1.2.3"},
			wantCode:   exitUsage,
			wantOutput: "必須",
		},
		"未知のフラグ": {
			args:       []string{"--unknown"},
			wantCode:   exitUsage,
			wantOutput: "コマンドライン引数",
		},
		"位置引数": {
			args: []string{
				"--changelog", "CHANGELOG.md",
				"--release-notes", "release-notes/CURRENT.md",
				"--tag", "v1.2.3",
				"--repository", ".",
				"extra",
			},
			wantCode:   exitUsage,
			wantOutput: "位置引数",
		},
		"生成失敗": {
			args: []string{
				"--changelog", "CHANGELOG.md",
				"--release-notes", "release-notes/CURRENT.md",
				"--tag", "v1.2.3",
				"--repository", ".",
			},
			buildError:  errors.New("結合できません"),
			wantCode:    exitValidation,
			wantOutput:  "結合できません",
			wantBuilder: true,
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			called := false
			code := execute(
				append([]string(nil), test.args...),
				&stdout,
				&stderr,
				func(changelog, releaseNotes, tag, repository string) ([]byte, error) {
					called = true
					if changelog != "CHANGELOG.md" ||
						releaseNotes != "release-notes/CURRENT.md" ||
						tag != "v1.2.3" ||
						repository != "." {
						t.Fatalf(
							"builder inputs = %q, %q, %q, %q",
							changelog,
							releaseNotes,
							tag,
							repository,
						)
					}
					return []byte("公開用リリース情報\n"), test.buildError
				},
			)
			combined := stdout.String() + stderr.String()
			if code != test.wantCode {
				t.Fatalf("execute() = %d, want %d: %s", code, test.wantCode, combined)
			}
			if !strings.Contains(combined, test.wantOutput) {
				t.Fatalf("出力 = %q, want %q", combined, test.wantOutput)
			}
			if called != test.wantBuilder {
				t.Fatalf("builder 呼出し = %t, want %t", called, test.wantBuilder)
			}
		})
	}
}

func TestExecuteRejectsMissingBuilder(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	code := execute(
		[]string{
			"--changelog", "CHANGELOG.md",
			"--release-notes", "release-notes/CURRENT.md",
			"--tag", "v1.2.3",
			"--repository", ".",
		},
		&bytes.Buffer{},
		&stderr,
		nil,
	)
	if code != exitValidation || !strings.Contains(stderr.String(), "生成処理") {
		t.Fatalf("execute() = %d, stderr = %q", code, stderr.String())
	}
}

func TestExecuteReportsOutputFailure(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	code := execute(
		[]string{
			"--changelog", "CHANGELOG.md",
			"--release-notes", "release-notes/CURRENT.md",
			"--tag", "v1.2.3",
			"--repository", ".",
		},
		failingWriter{},
		&stderr,
		func(_, _, _, _ string) ([]byte, error) {
			return []byte("release notes"), nil
		},
	)
	if code != exitValidation || !strings.Contains(stderr.String(), "標準出力") {
		t.Fatalf("execute() = %d, stderr = %q", code, stderr.String())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}
