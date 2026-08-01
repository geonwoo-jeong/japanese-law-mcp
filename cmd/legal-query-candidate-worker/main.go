// legal-query-candidate-worker は、CI bootstrap だけが起動する候補評価 worker である。
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateworker"
)

const (
	fixedRepository      = "."
	fixedOutputDirectory = "./.artifacts/legal-query-candidate-evaluation"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr, legalquerycandidateworker.Execute)
	stop()
	os.Exit(code)
}

type workerExecutor func(context.Context, legalquerycandidateworker.Input) (legalquerycandidateworker.Handoff, error)

func run(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	execute workerExecutor,
) int {
	if len(args) != 2 ||
		args[0] != "--repository="+fixedRepository ||
		args[1] != "--output-directory="+fixedOutputDirectory {
		_, _ = fmt.Fprintln(stderr, "候補評価 worker の固定引数が不正です")
		return 2
	}
	if ctx == nil || ctx.Err() != nil {
		_, _ = fmt.Fprintln(stderr, "候補評価 worker の context が不正です")
		return 1
	}
	if execute == nil {
		_, _ = fmt.Fprintln(stderr, "候補評価 worker の実行境界がありません")
		return 1
	}
	handoff, err := execute(ctx, legalquerycandidateworker.Input{
		RepositoryRoot: fixedRepository,
		OutputRoot:     fixedOutputDirectory,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "候補評価 worker を正常に完了できませんでした")
		return legalquerycandidateworker.FailureExitCode(err)
	}
	_, _ = fmt.Fprintf(
		stdout,
		"evaluationId=%s outcome=%s reportSha256=%s resultSha256=%s\n",
		handoff.EvaluationID,
		handoff.Outcome,
		handoff.ReportSHA256,
		handoff.ResultSHA256,
	)
	return 0
}
