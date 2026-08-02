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
	if output != "" {
		t.Fatalf("stderr=%q, want empty", output)
	}
	if strings.Contains(output, forbiddenSample) {
		t.Fatalf("stderr leaked secret: %q", output)
	}
}

func TestRunは固定境界失敗でも出力を空に保つ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctx      context.Context
		args     []string
		executor workerExecutor
		wantCode int
	}{
		{name: "usage", ctx: context.Background(), args: nil, executor: func(context.Context, legalquerycandidateworker.Input) (legalquerycandidateworker.Handoff, error) {
			return legalquerycandidateworker.Handoff{}, nil
		}, wantCode: 2},
		{name: "context", ctx: nil, args: []string{"--repository=.", "--output-directory=./.artifacts/legal-query-candidate-evaluation"}, executor: func(context.Context, legalquerycandidateworker.Input) (legalquerycandidateworker.Handoff, error) {
			return legalquerycandidateworker.Handoff{}, nil
		}, wantCode: 1},
		{name: "executor", ctx: context.Background(), args: []string{"--repository=.", "--output-directory=./.artifacts/legal-query-candidate-evaluation"}, executor: nil, wantCode: 1},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := run(test.ctx, test.args, &stdout, &stderr, test.executor)
			if code != test.wantCode || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("candidate-evaluation-failure-stage-privacy: code=%d output=(%q,%q)", code, stdout.String(), stderr.String())
			}
		})
	}
}
