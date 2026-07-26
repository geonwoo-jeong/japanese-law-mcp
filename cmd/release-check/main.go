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

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/releasecheck"
)

const (
	exitSuccess    = 0
	exitValidation = 1
	exitUsage      = 2
)

type checker func(context.Context, releasecheck.Request) error

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	code := execute(
		ctx,
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		releasecheck.Check,
	)
	stop()
	os.Exit(code)
}

func execute(
	ctx context.Context,
	args []string,
	stdout, stderr io.Writer,
	check checker,
) int {
	flags := flag.NewFlagSet("release-check", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var request releasecheck.Request
	flags.StringVar(&request.ReleaseNotes, "release-notes", "", "リリース情報のパス")
	flags.StringVar(&request.Tag, "tag", "", "v で始まる SemVer tag")
	flags.StringVar(&request.Repository, "repository", "", "SOT を含むリポジトリ")
	flags.StringVar(&request.Dist, "dist", "", "GoReleaser dist directory")
	flags.StringVar(&request.Commit, "commit", "", "生成元 commit")
	flags.StringVar(&request.TargetOS, "target-os", "", "smoke test 対象 OS")
	flags.StringVar(&request.TargetArch, "target-arch", "", "smoke test 対象 architecture")
	if err := flags.Parse(append([]string(nil), args...)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			writeUsage(stdout)
			return exitSuccess
		}
		_, _ = fmt.Fprintln(stderr, "エラー: コマンドライン引数が不正です")
		return exitUsage
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintln(stderr, "エラー: 位置引数は使用できません")
		return exitUsage
	}
	if err := releasecheck.ValidateRequest(request); err != nil {
		_, _ = fmt.Fprintf(stderr, "エラー: %s\n", err)
		return exitUsage
	}
	if check == nil {
		_, _ = fmt.Fprintln(stderr, "エラー: 検証処理がありません")
		return exitValidation
	}
	if err := check(ctx, request); err != nil {
		_, _ = fmt.Fprintf(stderr, "エラー: %s\n", err)
		return exitValidation
	}
	_, _ = fmt.Fprintln(stdout, "リリース検証に成功しました")
	return exitSuccess
}

func writeUsage(output io.Writer) {
	_, _ = fmt.Fprintln(
		output,
		"使用方法: release-check --release-notes <path> --tag <vSemVer> "+
			"--repository <path> "+
			"[--dist <path> --commit <sha> "+
			"[--target-os <os> --target-arch <arch>]]",
	)
}
