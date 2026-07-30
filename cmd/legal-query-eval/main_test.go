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

func TestRunParsesOptionsAndWritesJSON(t *testing.T) {
	t.Parallel()

	var got options
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	code := run(
		context.Background(),
		[]string{
			"--corpus=./testdata/legalquery/corpus-v9",
			"--profile-set=default",
			"--baseline=./testdata/legalquery/baselines/default.json",
			"--format=json",
		},
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
	want := options{
		Corpus:     "testdata/legalquery/corpus-v9",
		ProfileSet: "default",
		Baseline:   "testdata/legalquery/baselines/default.json",
		Format:     "json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("オプションが一致しません:\n got: %#v\nwant: %#v", got, want)
	}
	if gotOutput := stdout.String(); gotOutput != "{\"ok\":true}\n" {
		t.Fatalf("標準出力 = %q", gotOutput)
	}
}

func TestParseOptionsはWindows区切りでも固定RepositoryPathを受理する(
	t *testing.T,
) {
	t.Parallel()

	got, err := parseOptions([]string{
		`--corpus=.\testdata\legalquery\corpus-v9`,
		"--profile-set=default",
		`--baseline=.\testdata\legalquery\baselines\default.json`,
		"--format=json",
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Windows の path 区切りを受理できません: %v", err)
	}
	if got.Corpus != standardCorpusPath ||
		got.Baseline != standardBaselinePath {
		t.Fatalf("固定 path へ正規化されません: %#v", got)
	}
}

func TestVerifyStandardBaselineVersionはDefault1以外を拒否する(
	t *testing.T,
) {
	t.Parallel()

	if err := verifyStandardBaselineVersion("changed-1"); err == nil {
		t.Fatal("SOT-ENG-024: 未採用の baseline version を受理しました")
	}
	if err := verifyStandardBaselineVersion("default-1"); err != nil {
		t.Fatalf("SOT-ENG-024: default-1 を拒否しました: %v", err)
	}
}

func TestRunは受入失敗でも診断JSONを一つ出力する(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(
		context.Background(),
		[]string{
			"--corpus=./testdata/legalquery/corpus-v9",
			"--profile-set=default",
			"--baseline=./testdata/legalquery/baselines/default.json",
			"--format=json",
		},
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
	if stdout.String() !=
		"{\"artifactKind\":\"legal_query_evaluation\"}\n" {
		t.Fatalf("標準出力 = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "受入基準") {
		t.Fatalf("失敗理由がありません: %q", stderr.String())
	}
}

func TestRunRejectsInvalidArgumentsInJapanese(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "corpus がない",
			args: []string{
				"--profile-set=default",
				"--baseline=./testdata/legalquery/baselines/default.json",
				"--format=json",
			},
		},
		{
			name: "未知の profile set",
			args: []string{
				"--corpus=./testdata/legalquery/corpus-v9",
				"--profile-set=custom",
				"--baseline=./testdata/legalquery/baselines/default.json",
				"--format=json",
			},
		},
		{
			name: "baseline がない",
			args: []string{
				"--corpus=./testdata/legalquery/corpus-v9",
				"--profile-set=default",
				"--format=json",
			},
		},
		{
			name: "format が json ではない",
			args: []string{
				"--corpus=./testdata/legalquery/corpus-v9",
				"--profile-set=default",
				"--baseline=./testdata/legalquery/baselines/default.json",
				"--format=text",
			},
		},
		{
			name: "repository 上書き",
			args: []string{
				"--repository=/tmp/other",
				"--corpus=./testdata/legalquery/corpus-v9",
				"--profile-set=default",
				"--baseline=./testdata/legalquery/baselines/default.json",
				"--format=json",
			},
		},
		{
			name: "bootstrap baseline version",
			args: []string{
				"--corpus=./testdata/legalquery/corpus-v9",
				"--profile-set=default",
				"--baseline-version=default-1",
				"--format=json",
			},
		},
		{
			name: "corpus path traversal",
			args: []string{
				"--corpus=../corpus-v9",
				"--profile-set=default",
				"--baseline=./testdata/legalquery/baselines/default.json",
				"--format=json",
			},
		},
		{
			name: "baseline path traversal",
			args: []string{
				"--corpus=./testdata/legalquery/corpus-v9",
				"--profile-set=default",
				"--baseline=../default.json",
				"--format=json",
			},
		},
		{
			name: "profile set 重複",
			args: []string{
				"--corpus=./testdata/legalquery/corpus-v9",
				"--profile-set=default",
				"--profile-set=default",
				"--baseline=./testdata/legalquery/baselines/default.json",
				"--format=json",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
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

func TestExecuteはCorpusV9とReview済みBaselineを完全一致で評価する(
	t *testing.T,
) {
	t.Chdir("../..")

	result, err := execute(context.Background(), options{
		Corpus:     standardCorpusPath,
		ProfileSet: "default",
		Baseline:   standardBaselinePath,
		Format:     "json",
	})
	if err != nil {
		t.Fatalf("SOT-ENG-024: 標準評価 command が失敗しました: %v", err)
	}
	baseline, err := os.ReadFile(standardBaselinePath)
	if err != nil {
		t.Fatalf("review 済み baseline を読めません: %v", err)
	}
	if string(result) != string(baseline) {
		t.Fatal("SOT-ENG-024: 評価結果が review 済み baseline と一致しません")
	}
	if !strings.Contains(
		string(result),
		`"corpusVersion":"corpus-v9"`,
	) {
		t.Fatalf("SOT-ENG-024: corpus-v9 の report ではありません: %s", result)
	}
	if !strings.Contains(
		string(result),
		`"baselineVersion":"default-1"`,
	) {
		t.Fatalf("SOT-ENG-024: default-1 の report ではありません: %s", result)
	}
}
