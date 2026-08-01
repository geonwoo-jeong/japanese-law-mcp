package legalquerycandidateworker

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
)

func TestProductionPreparationはHoldoutを開く前にCandidatePayloadを閉じる(t *testing.T) {
	prepared, err := loadPreparedEvaluation(context.Background(), "../..")
	if err != nil {
		t.Fatalf("production candidate を準備できません: %v", err)
	}
	if prepared.EvaluationID == "" || len(prepared.RequestRaw) == 0 ||
		prepared.request.EvaluationID != prepared.EvaluationID ||
		prepared.content.CandidateContentID != prepared.request.CandidateContentID ||
		prepared.repository != "../.." || prepared.tracked != nil {
		t.Fatalf("production preparation payload が不正です: %+v", prepared)
	}
	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		t.Fatalf("candidate planning を構成できません: %v", err)
	}
	if err := verifyCandidatePlanningIdentity(candidate, prepared.content); err != nil {
		t.Fatalf("candidate planning identity が一致しません: %v", err)
	}
}
