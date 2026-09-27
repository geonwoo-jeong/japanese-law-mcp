// sdk-patch-check は、SOT-ENG-052 の SDK copy 照合だけを行う開発用 command である。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/sdkpatchcheck"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr, sdkpatchcheck.Run)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, check func(context.Context, string) error) int {
	flags := flag.NewFlagSet("sdk-patch-check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repository := flags.String("repository", ".", "照合対象の repository")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintln(stdout, "使用方法: sdk-patch-check [--repository <path>]")
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "SDK 照合の引数が不正です: %v\n", err)
		return 2
	}
	if flags.NArg() != 0 || *repository == "" {
		_, _ = fmt.Fprintln(stderr, "SDK 照合には repository と名前付き引数だけを指定してください")
		return 2
	}
	if err := check(ctx, *repository); err != nil {
		_, _ = fmt.Fprintf(stderr, "SDK copy の照合が失敗しました: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "SDK copy と固定 patch の照合が成功しました")
	return 0
}
