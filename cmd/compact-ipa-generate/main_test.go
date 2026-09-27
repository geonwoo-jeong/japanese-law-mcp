package main

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/compactipagenerate"
)

func TestRunDelegatesValidatedOptions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ctx := context.Background()
	called := false
	code := run(ctx, []string{"--repository", "synthetic", "--check"}, &stdout, &stderr, func(got context.Context, options compactipagenerate.Options) error {
		called = true
		if got != ctx || options.Repository != "synthetic" || !options.Check {
			t.Fatal("SOT-ENG-051: 検証した生成設定が委譲されません")
		}
		return nil
	})
	if code != 0 || !called || stderr.Len() != 0 {
		t.Fatal("SOT-ENG-051: 生成コマンドが成功しません")
	}
}

func TestRunRejectsInvalidArgumentsAndReportsFailure(t *testing.T) {
	for _, args := range [][]string{{"--unknown"}, {"unexpected"}, {"--repository", ""}} {
		var stdout, stderr bytes.Buffer
		code := run(context.Background(), args, &stdout, &stderr, func(context.Context, compactipagenerate.Options) error {
			t.Fatal("SOT-ENG-051: 不正な引数を実行しました")
			return nil
		})
		if code != 2 {
			t.Fatal("SOT-ENG-051: 引数エラーの終了コードが不正です")
		}
	}
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), nil, &stdout, &stderr, func(context.Context, compactipagenerate.Options) error { return fmt.Errorf("合成エラー") })
	if code != 1 || stderr.Len() == 0 {
		t.Fatal("SOT-ENG-051: 生成失敗が伝わりません")
	}
}
