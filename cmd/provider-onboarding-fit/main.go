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

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/provideronboarding"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		func(options provideronboarding.Options) error {
			return provideronboarding.Run(ctx, options)
		},
	)
	stop()
	os.Exit(code)
}

func run(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	execute func(provideronboarding.Options) error,
) int {
	baseRef, err := parseBaseRef(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			writeUsage(stdout)
			return 0
		}
		_, _ = fmt.Fprintf(stderr, "provider 追加 fitness gate の使用法が不正です: %v\n", err)
		return 2
	}
	options := provideronboarding.Options{
		Repository:         ".",
		GitRepository:      ".",
		BaseRef:            baseRef,
		HeadRef:            "HEAD",
		IncludeIndex:       true,
		IncludeWorkingTree: true,
		IncludeUntracked:   true,
		Stdout:             stdout,
		Stderr:             stderr,
	}
	if err := execute(options); err != nil {
		if errors.Is(err, provideronboarding.ErrInvalidBaseRef) {
			_, _ = fmt.Fprintf(stderr, "指定した --base-ref を使用できません: %v\n", err)
			return 2
		}
		_, _ = fmt.Fprintf(stderr, "provider 追加 fitness gate が失敗しました: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "provider 追加 fitness gate が成功しました")
	return 0
}

func writeUsage(output io.Writer) {
	_, _ = fmt.Fprintln(
		output,
		"使用方法: provider-onboarding-fit --base-ref <git-revision>",
	)
}

func parseBaseRef(args []string) (string, error) {
	flags := flag.NewFlagSet("provider-onboarding-fit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var value singleBaseRef
	flags.Var(&value, "base-ref", "比較元の Git revision")
	if err := flags.Parse(args); err != nil {
		return "", fmt.Errorf("引数を解釈できません: %w", err)
	}
	if flags.NArg() != 0 {
		return "", errors.New("位置引数は指定できません")
	}
	if value.count == 0 {
		return "", errors.New("--base-ref を一回指定してください")
	}
	if value.value == "" {
		return "", errors.New("--base-ref に空の値は指定できません")
	}
	return value.value, nil
}

type singleBaseRef struct {
	value string
	count int
}

func (value *singleBaseRef) String() string {
	if value == nil {
		return ""
	}
	return value.value
}

func (value *singleBaseRef) Set(next string) error {
	if value.count != 0 {
		return errors.New("--base-ref は一回だけ指定できます")
	}
	value.count++
	value.value = next
	return nil
}
