package legalquerycandidateworker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/evaluators"
)

func TestReport直列化失敗はReport完成前の段階に分類する(t *testing.T) {
	t.Parallel()

	_, err := encodeCandidateReport(
		legalqueryeval.StandardReport{},
		func(any) ([]byte, error) {
			return nil, errors.New("report encode failure")
		},
	)
	if code := FailureExitCode(err); code != FailureCodeReportBinding {
		t.Fatalf("candidate-evaluation-report-completion-boundary: report encode code=%d, want %d", code, FailureCodeReportBinding)
	}
}

func TestProductionPreparationはStaleCurrentをHoldout前に拒否する(t *testing.T) {
	if !useExactCandidateToolchain(t) {
		t.Skip("候補再現用 Go 環境がないため local では実行しません")
	}

	if _, err := loadPreparedEvaluation(
		context.Background(),
		"../..",
	); !legalquerycandidateeval.IsCurrentStale(err) {
		t.Fatalf("candidate-evaluation-stale-candidate-readiness-fail: stale current の preparation error=%v", err)
	}
}

func TestCurrentCandidateのStaleはWorkerとOutputを非到達にする(
	t *testing.T,
) {
	if !useExactCandidateToolchain(t) {
		t.Skip("候補再現用 Go 環境がないため local では実行しません")
	}

	outputRoot := filepath.Join(t.TempDir(), "handoff")
	_, err := Execute(context.Background(), Input{
		RepositoryRoot: "../..",
		OutputRoot:     outputRoot,
	})
	if !legalquerycandidateeval.IsCurrentStale(err) ||
		FailureExitCode(err) != FailureCodePreparedLoad {
		t.Fatalf("candidate-evaluation-stale-worker-unreachable: worker error=%v code=%d", err, FailureExitCode(err))
	}
	if _, statErr := os.Lstat(outputRoot); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("candidate-evaluation-stale-output-unreachable: output が残りました: %v", statErr)
	}
}

func TestEvaluateTrackedCandidateは不変なReplayBindingだけを受理する(
	t *testing.T,
) {
	t.Parallel()

	const evaluationID = "evaluation-sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	requestRaw := []byte("request\n")
	reportRaw := []byte("report\n")
	valid := PreparedEvaluation{
		EvaluationID: evaluationID,
		RequestRaw:   requestRaw,
		request: legalquerycandidateeval.EvaluationRequest{
			EvaluationID: evaluationID,
		},
		tracked: &legalquerycandidateeval.EvaluationResult{
			EvaluationID:  evaluationID,
			RequestSHA256: legalquerycandidateeval.RawSHA256(requestRaw),
			ReportSHA256:  legalquerycandidateeval.RawSHA256(reportRaw),
		},
		trackedReportRaw: reportRaw,
	}
	got, err := evaluateTrackedCandidate(context.Background(), valid)
	if err != nil {
		t.Fatalf("正常な tracked replay を拒否しました: %v", err)
	}
	got[0] ^= 0xff
	if string(valid.trackedReportRaw) != "report\n" {
		t.Fatal("tracked report の原 byte を共有しました")
	}

	tests := []struct {
		name   string
		mutate func(*PreparedEvaluation)
	}{
		{
			name: "result欠落",
			mutate: func(value *PreparedEvaluation) {
				value.tracked = nil
			},
		},
		{
			name: "result評価ID不一致",
			mutate: func(value *PreparedEvaluation) {
				value.tracked.EvaluationID = "evaluation-sha256-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			},
		},
		{
			name: "request評価ID不一致",
			mutate: func(value *PreparedEvaluation) {
				value.request.EvaluationID = "evaluation-sha256-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			},
		},
		{
			name: "request原byte不一致",
			mutate: func(value *PreparedEvaluation) {
				value.RequestRaw = []byte("changed request\n")
			},
		},
		{
			name: "report原byte不一致",
			mutate: func(value *PreparedEvaluation) {
				value.trackedReportRaw = []byte("changed report\n")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			changed := clonePreparedEvaluation(valid)
			test.mutate(&changed)
			if _, err := evaluateTrackedCandidate(
				context.Background(),
				changed,
			); err == nil {
				t.Fatal("不正な tracked replay binding を受理しました")
			}
		})
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := evaluateTrackedCandidate(cancelled, valid); err == nil {
		t.Fatal("取消し済みの tracked replay を受理しました")
	}
}

func TestNewCandidateEvaluatorはVersionごとのBuilder境界を保つ(t *testing.T) {
	t.Parallel()
	const verificationID = "candidate-evaluator-v3-exact-version-routing"

	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		t.Fatalf("candidate evaluator version routing: candidate profile load に失敗しました: %v", err)
	}
	v1, err := newCandidateEvaluator(evaluators.Version1, candidate)
	if err != nil {
		t.Fatalf("candidate evaluator version routing: v1 を拒否しました: %v", err)
	}
	v2, err := newCandidateEvaluator(evaluators.Version2, candidate)
	if err != nil {
		t.Fatalf("candidate evaluator version routing: v2 を拒否しました: %v", err)
	}
	v3, err := newCandidateEvaluator(evaluators.Version3, candidate)
	if err != nil {
		t.Fatalf("candidate evaluator version routing: v3 を拒否しました: %v", err)
	}
	if v1.ScoresRequestBoundaryMismatch() ||
		!v2.ScoresRequestBoundaryMismatch() ||
		!v3.ScoresRequestBoundaryMismatch() {
		t.Fatal("candidate evaluator version routing: v1/v2/v3 の request 境界 policy が逆転しました")
	}
	if v1.ScoresCandidatePlanningFailure() ||
		v2.ScoresCandidatePlanningFailure() ||
		!v3.ScoresCandidatePlanningFailure() {
		t.Fatalf("%s: v1/v2/v3 の候補失敗 policy が一致しません", verificationID)
	}
	if _, err := newCandidateEvaluator("legal-query-evaluator-v999", candidate); err == nil {
		t.Fatal("candidate evaluator version routing: 未知版を受理しました")
	}
}
