package legalquerycandidateeval

import (
	"errors"
	"testing"
)

func TestCurrentStaleErrorは閉じた理由集合を固定順で保持する(t *testing.T) {
	t.Parallel()

	err := NewCurrentStaleError(
		StaleReasonCurrentEvaluatorDrift,
		StaleReasonCandidateContentDrift,
		StaleReasonCurrentEvaluatorDrift,
		StaleReasonReviewSOTDigestDrift,
	)
	reasons, ok := StaleReasonsFromError(err)
	if !ok {
		t.Fatal("candidate-evaluation-stale-reason-closed-set: stale error として抽出できません")
	}
	want := []StaleReason{
		StaleReasonCandidateContentDrift,
		StaleReasonReviewSOTDigestDrift,
		StaleReasonCurrentEvaluatorDrift,
	}
	if !EqualStaleReasons(reasons, want) {
		t.Fatalf("candidate-evaluation-stale-reason-closed-set: reasons=%v want=%v", reasons, want)
	}
	if !IsCurrentStale(err) {
		t.Fatal("candidate-evaluation-stale-reason-closed-set: stale 判定に失敗しました")
	}
}

func TestCurrentStaleErrorは通常Errorと区別する(t *testing.T) {
	t.Parallel()

	if reasons, ok := StaleReasonsFromError(errors.New("plain")); ok || reasons != nil {
		t.Fatalf("candidate-evaluation-stale-reason-closed-set: plain error を stale と誤判定しました: %v", reasons)
	}
	if IsCurrentStale(nil) {
		t.Fatal("candidate-evaluation-stale-reason-closed-set: nil error を stale と誤判定しました")
	}
}

func TestCurrentStaleErrorは未知理由をStaleへ変換しない(t *testing.T) {
	t.Parallel()

	err := NewCurrentStaleError(StaleReason("unknown_reason"))
	if err == nil {
		t.Fatal("candidate-evaluation-stale-reason-closed-set: 未知理由を無視しました")
	}
	if IsCurrentStale(err) {
		t.Fatal("candidate-evaluation-stale-reason-closed-set: 未知理由を stale として受理しました")
	}
	if _, err := AppendStaleReasons(nil, StaleReason("unknown_reason")); err == nil {
		t.Fatal("candidate-evaluation-stale-reason-closed-set: 未知理由を理由集合へ追加しました")
	}
}
