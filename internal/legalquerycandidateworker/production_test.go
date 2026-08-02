package legalquerycandidateworker

import (
	"context"
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
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

func TestProductionPreparationはHoldoutを開く前にCandidatePayloadを閉じる(t *testing.T) {
	const expectedEvaluationID = "evaluation-sha256-bf3567625d79634f6be2621e870459bd50221ac041dd146dbcfededec2676cb1"
	prepared, err := loadPreparedEvaluation(context.Background(), "../..")
	if err != nil {
		t.Fatalf("production candidate を準備できません: %v", err)
	}
	if prepared.EvaluationID != expectedEvaluationID || len(prepared.RequestRaw) == 0 ||
		prepared.request.EvaluationID != prepared.EvaluationID ||
		prepared.content.CandidateContentID != prepared.request.CandidateContentID ||
		prepared.repository != "../.." {
		t.Fatal("production preparation payload の identity が不正です")
	}
	if prepared.request.CorpusVersion != "corpus-v14" ||
		prepared.request.BaselineVersion != "default-7" {
		t.Fatalf("production preparation request が後続予約と一致しません: %#v", prepared.request)
	}
	if prepared.request.EvaluatorVersion != evaluators.Version2 {
		t.Fatalf("production preparation request の evaluatorVersion = %q", prepared.request.EvaluatorVersion)
	}
	if prepared.tracked != nil || len(prepared.trackedRaw) != 0 ||
		len(prepared.trackedReportRaw) != 0 {
		t.Fatal("production preparation が未評価 current に tracked replay を結び付けました")
	}
}

func TestCurrentCandidateはDevelopment全件でReport構成前提を満たす(
	t *testing.T,
) {
	const verificationID = "candidate-evaluation-development-structural-preflight"

	prepared, err := loadPreparedEvaluation(context.Background(), "../..")
	if err != nil {
		t.Fatalf("%s: candidate preparation に失敗しました", verificationID)
	}
	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		t.Fatalf("%s: candidate profile load に失敗しました", verificationID)
	}
	if err := verifyCandidatePlanningIdentity(candidate, prepared.content); err != nil {
		t.Fatalf("%s: candidate planning identity が一致しません", verificationID)
	}
	development, err := legalquerycorpus.LoadDevelopment(
		context.Background(),
		"../..",
		"testdata/legalquery/"+prepared.request.CorpusVersion+"/development",
	)
	if err != nil {
		t.Fatalf("%s: development corpus load に失敗しました", verificationID)
	}
	evaluator, err := newCandidateEvaluator(prepared.request.EvaluatorVersion, candidate)
	if err != nil {
		t.Fatalf("%s: candidate evaluator の構築に失敗しました", verificationID)
	}
	completed := 0
	cases := development.Cases()
	for _, semanticCase := range cases {
		if _, _, _, err := evaluator.EvaluateWithPlan(
			context.Background(),
			semanticCase,
		); err != nil {
			t.Fatalf(
				"%s: development case の構造評価に失敗しました（完了 %d/%d）",
				verificationID,
				completed,
				len(cases),
			)
		}
		completed++
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
