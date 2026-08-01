// legal-query-candidate-worker は、CI bootstrap だけが起動する候補評価 worker である。
package main

import (
	"context"
	"fmt"
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
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) != 2 ||
		args[0] != "--repository="+fixedRepository ||
		args[1] != "--output-directory="+fixedOutputDirectory {
		_, _ = fmt.Fprintln(os.Stderr, "候補評価 worker の固定引数が不正です")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	handoff, err := legalquerycandidateworker.Execute(ctx, legalquerycandidateworker.Input{
		RepositoryRoot: fixedRepository,
		OutputRoot:     fixedOutputDirectory,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "候補評価 worker を正常に完了できませんでした")
		return 1
	}
	_, _ = fmt.Fprintf(
		os.Stdout,
		"evaluationId=%s outcome=%s reportSha256=%s resultSha256=%s\n",
		handoff.EvaluationID,
		handoff.Outcome,
		handoff.ReportSHA256,
		handoff.ResultSHA256,
	)
	return 0
}
