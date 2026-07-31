package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRunParsesAdoptionOptionsAndWritesJSON(t *testing.T) {
	t.Parallel()

	var got options
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	code := run(
		context.Background(),
		standardArguments(),
		&stdout,
		&stderr,
		func(_ context.Context, current options) ([]byte, error) {
			got = current
			return []byte("{\"ok\":true}\n"), nil
		},
	)
	if code != 0 {
		t.Fatalf("終了コード = %d, stderr = %q", code, stderr.String())
	}
	want := options{Adoption: standardAdoptionPath, Format: "json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("オプションが一致しません:\n got: %#v\nwant: %#v", got, want)
	}
	if stdout.String() != "{\"ok\":true}\n" {
		t.Fatalf("標準出力 = %q", stdout.String())
	}
}

func TestParseOptionsはWindows区切りでも固定AdoptionPathを受理する(t *testing.T) {
	t.Parallel()

	got, err := parseOptions([]string{
		`--adoption=.\testdata\legalquery\adoptions\current.json`,
		"--format=json",
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Windows の path 区切りを受理できません: %v", err)
	}
	if got.Adoption != standardAdoptionPath {
		t.Fatalf("固定 path へ正規化されません: %#v", got)
	}
}

func TestRunは受入失敗でも診断JSONを一つ出力する(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(
		context.Background(),
		standardArguments(),
		&stdout,
		&stderr,
		func(context.Context, options) ([]byte, error) {
			return []byte("{\"artifactKind\":\"legal_query_evaluation\"}\n"),
				errors.New("受入基準を満たしません")
		},
	)
	if code != 1 {
		t.Fatalf("終了コード = %d, want 1", code)
	}
	if stdout.String() != "{\"artifactKind\":\"legal_query_evaluation\"}\n" {
		t.Fatalf("標準出力 = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "受入基準") {
		t.Fatalf("失敗理由がありません: %q", stderr.String())
	}
}

func TestRunRejectsLegacyAndInvalidArgumentsInJapanese(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "adoption がない", args: []string{"--format=json"}},
		{
			name: "未知の adoption path",
			args: []string{
				"--adoption=./testdata/legalquery/adoptions/other.json",
				"--format=json",
			},
		},
		{
			name: "format がない",
			args: []string{"--adoption=./testdata/legalquery/adoptions/current.json"},
		},
		{
			name: "format が json ではない",
			args: []string{
				"--adoption=./testdata/legalquery/adoptions/current.json",
				"--format=text",
			},
		},
		{
			name: "repository 上書き",
			args: append(standardArguments(), "--repository=/tmp/other"),
		},
		{
			name: "legacy corpus override",
			args: append(standardArguments(), "--corpus=./testdata/legalquery/corpus-v9"),
		},
		{
			name: "legacy profile override",
			args: append(standardArguments(), "--profile-set=default"),
		},
		{
			name: "legacy baseline override",
			args: append(
				standardArguments(),
				"--baseline=./testdata/legalquery/baselines/default.json",
			),
		},
		{
			name: "adoption path traversal",
			args: []string{"--adoption=../current.json", "--format=json"},
		},
		{
			name: "adoption 重複",
			args: []string{
				"--adoption=./testdata/legalquery/adoptions/current.json",
				"--adoption=./testdata/legalquery/adoptions/current.json",
				"--format=json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			code := run(
				context.Background(),
				tt.args,
				&bytes.Buffer{},
				&stderr,
				func(context.Context, options) ([]byte, error) {
					t.Fatal("不正な引数で実行されました")
					return nil, nil
				},
			)
			if code == 0 {
				t.Fatal("不正な引数が成功扱いになりました")
			}
			if !strings.Contains(stderr.String(), "統合照会評価") {
				t.Fatalf("日本語のエラー説明がありません: %q", stderr.String())
			}
		})
	}
}

func TestExecuteはCurrentAdoptionとReview済みBaselineを完全一致で評価する(
	t *testing.T,
) {
	t.Chdir("../..")

	result, err := execute(context.Background(), options{
		Adoption: standardAdoptionPath,
		Format:   "json",
	})
	if err != nil {
		t.Fatalf("profile-set-initial-adoption-bootstrap: 標準評価に失敗しました: %v", err)
	}
	baseline, err := os.ReadFile("testdata/legalquery/baselines/default.json")
	if err != nil {
		t.Fatalf("review 済み baseline を読めません: %v", err)
	}
	if string(result) != string(baseline) {
		t.Fatal("profile-set-production-evaluation-identity: report byte が一致しません")
	}
	for _, marker := range []string{
		`"corpusVersion":"corpus-v9"`,
		`"baselineVersion":"default-1"`,
	} {
		if !strings.Contains(string(result), marker) {
			t.Fatalf("採用済み report marker がありません: %s", marker)
		}
	}
}

func standardArguments() []string {
	return []string{
		"--adoption=./testdata/legalquery/adoptions/current.json",
		"--format=json",
	}
}
