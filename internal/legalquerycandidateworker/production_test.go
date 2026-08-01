package legalquerycandidateworker

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
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
	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		t.Fatalf("candidate planning を構成できません: %v", err)
	}
	if err := verifyCandidatePlanningIdentity(candidate, prepared.content); err != nil {
		t.Fatalf("candidate planning identity が一致しません: %v", err)
	}
}
