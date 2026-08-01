package legalquerycandidateeval

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

const (
	verificationDeterministicReplay  = "candidate-evaluation-deterministic-replay"
	verificationOutcomeExitSemantics = "candidate-evaluation-outcome-exit-semantics"
	verificationSuccessHandoff       = "candidate-evaluation-success-handoff"
	verificationFailureHistory       = "candidate-evaluation-failure-history"
	verificationImmutableVersion     = "candidate-evaluation-immutable-version"
)

func TestEvaluationResultは同じ入力から同じCanonicalByteを再現する(t *testing.T) {
	t.Parallel()

	request := validEvaluationRequest(t, manifestWithID(t))
	requestRaw := mustCanonicalJSON(t, request)
	reportRaw := syntheticEvaluationReportRaw("deterministic")
	wantRequestDigest := RawSHA256(requestRaw)
	wantReportDigest := RawSHA256(reportRaw)

	first, err := NewEvaluationResult(request, requestRaw, reportRaw, EvaluationOutcomePassed)
	if err != nil {
		t.Fatalf("%s: result を構成できません: %v", verificationDeterministicReplay, err)
	}
	second, err := NewEvaluationResult(request, requestRaw, reportRaw, EvaluationOutcomePassed)
	if err != nil {
		t.Fatalf("%s: result を再構成できません: %v", verificationDeterministicReplay, err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("%s: 同じ入力から異なる result が生成されました", verificationDeterministicReplay)
	}
	firstRaw := mustCanonicalJSON(t, first)
	secondRaw := mustCanonicalJSON(t, second)
	if !bytes.Equal(firstRaw, secondRaw) {
		t.Fatalf("%s: replay byte が一致しません", verificationDeterministicReplay)
	}

	requestRaw[0] = ' '
	reportRaw[0] = ' '
	if first.RequestSHA256 != wantRequestDigest || first.ReportSHA256 != wantReportDigest {
		t.Fatalf("%s: constructor 入力 byte の変更が result identity に反映されました", verificationImmutableVersion)
	}

	decoded, err := DecodeEvaluationResult(firstRaw)
	if err != nil {
		t.Fatalf("%s: 生成した result を decode できません: %v", verificationDeterministicReplay, err)
	}
	if !reflect.DeepEqual(decoded, first) {
		t.Fatalf("%s: decode 後の result が変わりました", verificationDeterministicReplay)
	}
}

func TestEvaluationResultはPassedとFailedをどちらも有効なHandoffとして構成する(t *testing.T) {
	t.Parallel()

	request := validEvaluationRequest(t, manifestWithID(t))
	requestRaw := mustCanonicalJSON(t, request)
	for _, outcome := range []string{EvaluationOutcomePassed, EvaluationOutcomeFailed} {
		result, err := NewEvaluationResult(
			request,
			requestRaw,
			syntheticEvaluationReportRaw(outcome),
			outcome,
		)
		if err != nil {
			t.Fatalf("%s: outcome=%s を有効な handoff にできません: %v", verificationOutcomeExitSemantics, outcome, err)
		}
		if result.Outcome != outcome {
			t.Fatalf("%s: outcome = %q", verificationOutcomeExitSemantics, result.Outcome)
		}
	}

	if _, err := NewEvaluationResult(
		request,
		requestRaw,
		syntheticEvaluationReportRaw("unknown"),
		"unknown",
	); err == nil {
		t.Fatalf("%s: 未知 outcome を受理しました", verificationOutcomeExitSemantics)
	}
}

func TestPassedResultはRequestとReportの原Byteへ結合する(t *testing.T) {
	t.Parallel()

	request := validEvaluationRequest(t, manifestWithID(t))
	requestRaw := mustCanonicalJSON(t, request)
	reportRaw := syntheticEvaluationReportRaw("passed")
	result := mustSyntheticEvaluationResult(
		t,
		request,
		requestRaw,
		reportRaw,
		EvaluationOutcomePassed,
	)

	if result.ArtifactKind != "legal_query_candidate_evaluation_result" ||
		result.SchemaVersion != SchemaVersionV2 ||
		result.EvaluationID != request.EvaluationID ||
		result.RequestSHA256 != RawSHA256(requestRaw) ||
		result.ReportSHA256 != RawSHA256(reportRaw) {
		t.Fatalf("%s: passed result の handoff binding が一致しません: %+v", verificationSuccessHandoff, result)
	}
	if _, err := DecodeEvaluationResult(mustCanonicalJSON(t, result)); err != nil {
		t.Fatalf("%s: canonical passed result を拒否しました: %v", verificationSuccessHandoff, err)
	}
}

func TestFailedResultは不変な履歴成果物としてだけDecodeできる(t *testing.T) {
	t.Parallel()

	request := validEvaluationRequest(t, manifestWithID(t))
	requestRaw := mustCanonicalJSON(t, request)
	result := mustSyntheticEvaluationResult(
		t,
		request,
		requestRaw,
		syntheticEvaluationReportRaw("failed"),
		EvaluationOutcomeFailed,
	)
	if _, err := DecodeEvaluationResult(mustCanonicalJSON(t, result)); err != nil {
		t.Fatalf("%s: canonical failed result を拒否しました: %v", verificationFailureHistory, err)
	}

	wrongVersion := result
	wrongVersion.SchemaVersion = 1
	if _, err := DecodeEvaluationResult(mustCanonicalJSON(t, wrongVersion)); err == nil {
		t.Fatalf("%s: schema version 1 の result を受理しました", verificationImmutableVersion)
	}
	unknownOutcome := result
	unknownOutcome.Outcome = "unknown"
	if _, err := DecodeEvaluationResult(mustCanonicalJSON(t, unknownOutcome)); err == nil {
		t.Fatalf("%s: 未知 outcome の result を受理しました", verificationFailureHistory)
	}
	nonCanonical := append([]byte{' '}, mustCanonicalJSON(t, result)...)
	if _, err := DecodeEvaluationResult(nonCanonical); err == nil {
		t.Fatalf("%s: non-canonical result を受理しました", verificationImmutableVersion)
	}
}

func mustSyntheticEvaluationResult(
	t *testing.T,
	request EvaluationRequest,
	requestRaw []byte,
	reportRaw []byte,
	outcome string,
) EvaluationResult {
	t.Helper()
	result, err := NewEvaluationResult(request, requestRaw, reportRaw, outcome)
	if err != nil {
		t.Fatalf("synthetic evaluation result を構成できません: %v", err)
	}
	return result
}

func syntheticEvaluationReportRaw(label string) []byte {
	return []byte(fmt.Sprintf(
		"{\"artifactKind\":\"synthetic_legal_query_evaluation\",\"schemaVersion\":1,\"label\":%q}\n",
		label,
	))
}
