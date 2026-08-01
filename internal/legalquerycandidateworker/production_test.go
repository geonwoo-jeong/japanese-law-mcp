package legalquerycandidateworker

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

func TestProductionPreparationはHoldoutを開く前にCandidatePayloadを閉じる(t *testing.T) {
	const expectedEvaluationID = "evaluation-sha256-398e801b2d7edd6068f36fa34fe94827d7d44891d59976fdc8630e4d5be7e89c"
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
	if prepared.tracked == nil ||
		prepared.tracked.EvaluationID != expectedEvaluationID ||
		prepared.tracked.Outcome != legalquerycandidateeval.EvaluationOutcomeFailed {
		t.Fatal("production tracked replay payload が failed result と一致しません")
	}
	if len(prepared.trackedRaw) == 0 || len(prepared.trackedReportRaw) == 0 ||
		legalquerycandidateeval.RawSHA256(prepared.RequestRaw) != prepared.tracked.RequestSHA256 ||
		legalquerycandidateeval.RawSHA256(prepared.trackedReportRaw) != prepared.tracked.ReportSHA256 {
		t.Fatal("production tracked replay payload の原 byte binding が一致しません")
	}
	handoff, err := Execute(context.Background(), Input{
		RepositoryRoot: "../..",
		OutputRoot:     filepath.Join(t.TempDir(), "handoff"),
	})
	if err != nil {
		t.Fatalf("評価済み current を追跡済み byte で replay できません: %v", err)
	}
	if handoff.EvaluationID != expectedEvaluationID ||
		handoff.Outcome != legalquerycandidateeval.EvaluationOutcomeFailed ||
		handoff.ReportSHA256 != prepared.tracked.ReportSHA256 ||
		handoff.ResultSHA256 != legalquerycandidateeval.RawSHA256(prepared.trackedRaw) {
		t.Fatalf("production tracked replay handoff が一致しません: %#v", handoff)
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
