package legalquerycandidateworker

import (
	"context"
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
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
	const expectedEvaluationID = "evaluation-sha256-c53a7d0d28ef35bd2aab081680c1112b6aee9e649f19fb789ec2f0e0e35a4a87"
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
	if prepared.request.CorpusVersion != "corpus-v12" ||
		prepared.request.BaselineVersion != "default-5" {
		t.Fatalf("production preparation request が後続予約と一致しません: %#v", prepared.request)
	}
	if prepared.tracked != nil || len(prepared.trackedRaw) != 0 ||
		len(prepared.trackedReportRaw) != 0 {
		t.Fatal("production preparation が未評価 current に tracked replay を結び付けました")
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
