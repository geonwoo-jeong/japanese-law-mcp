package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateworker"
)

type codedFailure struct {
	code int
	err  error
}

func (e codedFailure) Error() string {
	if e.err == nil {
		return "failed"
	}
	return e.err.Error()
}

func (e codedFailure) Unwrap() error {
	return e.err
}

func (e codedFailure) FailureExitCode() int {
	return e.code
}

func TestRunは段階別失敗Codeだけを返す(t *testing.T) {
	t.Parallel()

	forbiddenSample := "sample-query 永住許可 /private/path redacted-marker"
	var stdout, stderr bytes.Buffer
	code := run(
		context.Background(),
		[]string{
			"--repository=.",
			"--output-directory=./.artifacts/legal-query-candidate-evaluation",
		},
		&stdout,
		&stderr,
		func(context.Context, legalquerycandidateworker.Input) (legalquerycandidateworker.Handoff, error) {
			return legalquerycandidateworker.Handoff{}, codedFailure{
				code: legalquerycandidateworker.FailureCodeHandoffWrite,
				err:  errors.New(forbiddenSample),
			}
		},
	)
	if code != legalquerycandidateworker.FailureCodeHandoffWrite {
		t.Fatalf("stage code=%d, want %d", code, legalquerycandidateworker.FailureCodeHandoffWrite)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout=%q, want empty", stdout.String())
	}
	output := stderr.String()
	if !strings.Contains(output, "候補評価 worker を正常に完了できませんでした") {
		t.Fatalf("stderr=%q, want generic failure", output)
	}
	if strings.Contains(output, forbiddenSample) {
		t.Fatalf("stderr leaked secret: %q", output)
	}
}
