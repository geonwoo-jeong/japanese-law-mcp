package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/qualitygate"
)

func TestRunParsesProfileRepositoryAndRepeatedGitRanges(t *testing.T) {
	t.Parallel()

	var got qualitygate.Options
	var stderr bytes.Buffer
	code := run(
		[]string{
			"--profile=pre-push",
			"--repository=/tmp/snapshot",
			"--git-repository=/work/original",
			"--git-range=main..first",
			"--git-range=first..second",
		},
		&bytes.Buffer{},
		&stderr,
		func(options qualitygate.Options) error {
			got = options
			return nil
		},
	)

	if code != 0 {
		t.Fatalf("終了コード = %d, stderr = %q", code, stderr.String())
	}
	want := qualitygate.Options{
		Profile:       qualitygate.ProfilePrePush,
		Repository:    "/tmp/snapshot",
		GitRepository: "/work/original",
		GitRanges:     []string{"main..first", "first..second"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("オプションが一致しません:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestRunRejectsMissingAndInvalidArgumentsInJapanese(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "プロファイルがない", args: []string{"--repository=/tmp/snapshot"}},
		{name: "リポジトリがない", args: []string{"--profile=ci"}},
		{name: "未知のプロファイル", args: []string{"--profile=fast", "--repository=/tmp/snapshot"}},
		{name: "位置引数", args: []string{"--profile=ci", "--repository=/tmp/snapshot", "extra"}},
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
				func(qualitygate.Options) error {
					t.Fatal("不正な引数でゲートが実行されました")
					return nil
				},
			)
			if code == 0 {
				t.Fatal("不正な引数が成功扱いになりました")
			}
			if !strings.Contains(stderr.String(), "品質ゲート") {
				t.Fatalf("日本語のエラー説明がありません: %q", stderr.String())
			}
		})
	}
}

func TestRunReportsGateFailures(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	code := run(
		[]string{"--profile=ci", "--repository=/tmp/snapshot"},
		&bytes.Buffer{},
		&stderr,
		func(qualitygate.Options) error { return errors.New("検査失敗") },
	)
	if code == 0 {
		t.Fatal("失敗が成功扱いになりました")
	}
	if !strings.Contains(stderr.String(), "検査失敗") {
		t.Fatalf("エラー内容が不足しています: %q", stderr.String())
	}
}

func TestRunDefaultsGitRepositoryToRepository(t *testing.T) {
	t.Parallel()

	var got qualitygate.Options
	code := run(
		[]string{"--profile=ci", "--repository=/tmp/snapshot"},
		&bytes.Buffer{},
		&bytes.Buffer{},
		func(options qualitygate.Options) error {
			got = options
			return nil
		},
	)
	if code != 0 {
		t.Fatalf("終了コード = %d", code)
	}
	if got.GitRepository != "/tmp/snapshot" {
		t.Fatalf("GitRepository = %q, want %q", got.GitRepository, "/tmp/snapshot")
	}
}
