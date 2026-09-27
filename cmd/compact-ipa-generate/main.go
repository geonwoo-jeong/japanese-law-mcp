// compact-ipa-generate は、SOT-ENG-051 の IPA 生成物を更新・照合する開発用コマンドである。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/compactipagenerate"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr, compactipagenerate.Run)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, execute func(context.Context, compactipagenerate.Options) error) int {
	flags := flag.NewFlagSet("compact-ipa-generate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	options := compactipagenerate.Options{}
	flags.StringVar(&options.Repository, "repository", ".", "生成対象の repository")
	flags.BoolVar(&options.Check, "check", false, "保存済み生成物と再生成結果を照合する")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(stdout, "使用方法: compact-ipa-generate [--repository <path>] [--check]")
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "IPA 生成器の引数が不正です: %v\n", err)
		return 2
	}
	if flags.NArg() != 0 || options.Repository == "" {
		_, _ = fmt.Fprintln(stderr, "IPA 生成器には repository と名前付き引数だけを指定してください")
		return 2
	}
	if err := execute(ctx, options); err != nil {
		_, _ = fmt.Fprintf(stderr, "IPA 生成物の処理が失敗しました: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "IPA 生成物の処理が成功しました（実行 Go: %s、照合専用: %t）\n", runtime.Version(), options.Check)
	return 0
}
