package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/qualitygate"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		func(options qualitygate.Options) error {
			return qualitygate.Run(ctx, options, os.Stdout, os.Stderr)
		},
	)
	stop()
	os.Exit(code)
}

func run(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	execute func(qualitygate.Options) error,
) int {
	options, err := parseOptions(args, stderr)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if err := execute(options); err != nil {
		_, _ = fmt.Fprintf(stderr, "品質ゲートが失敗しました: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, "品質ゲートが成功しました")
	return 0
}

func parseOptions(args []string, stderr io.Writer) (qualitygate.Options, error) {
	flags := flag.NewFlagSet("quality-gate", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var profileValue string
	var repository string
	var gitRepository string
	var gitRanges stringList
	flags.StringVar(&profileValue, "profile", "", "検査プロファイル: pre-commit、pre-push、ci")
	flags.StringVar(&repository, "repository", "", "検査対象のスナップショット")
	flags.StringVar(&gitRepository, "git-repository", "", "Git index と履歴を持つ元のリポジトリ")
	flags.Var(&gitRanges, "git-range", "pre-push で検査する Git revision 範囲（複数指定可）")

	if err := flags.Parse(args); err != nil {
		return qualitygate.Options{}, fmt.Errorf("品質ゲートの引数を解釈できません: %w", err)
	}
	if flags.NArg() != 0 {
		return qualitygate.Options{}, fmt.Errorf("品質ゲートに位置引数は指定できません")
	}
	profile, err := qualitygate.ParseProfile(profileValue)
	if err != nil {
		return qualitygate.Options{}, err
	}
	if repository == "" {
		return qualitygate.Options{}, fmt.Errorf("品質ゲートの --repository を指定してください")
	}
	if gitRepository == "" {
		gitRepository = repository
	}

	return qualitygate.Options{
		Profile:       profile,
		Repository:    repository,
		GitRepository: gitRepository,
		GitRanges:     append([]string(nil), gitRanges...),
	}, nil
}

type stringList []string

func (values *stringList) String() string {
	if values == nil {
		return ""
	}
	return fmt.Sprint([]string(*values))
}

func (values *stringList) Set(value string) error {
	*values = append(*values, value)
	return nil
}
