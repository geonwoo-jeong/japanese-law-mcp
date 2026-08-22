package evaluators

import "testing"

func TestNewはExactEvaluatorVersionだけを受理する(t *testing.T) {
	t.Parallel()
	const verificationID = "candidate-evaluator-v3-exact-version-routing"

	v1, err := New(Version1)
	if err != nil {
		t.Fatalf("profile-set-evaluator-version-identity: v1 を拒否しました: %v", err)
	}
	v2, err := New(Version2)
	if err != nil {
		t.Fatalf("profile-set-evaluator-version-identity: v2 を拒否しました: %v", err)
	}
	v3, err := New(Version3)
	if err != nil {
		t.Fatalf("profile-set-evaluator-version-identity: v3 を拒否しました: %v", err)
	}
	if v1.ScoresCandidatePlanningFailure() ||
		v2.ScoresCandidatePlanningFailure() ||
		!v3.ScoresCandidatePlanningFailure() {
		t.Fatalf("%s: v1/v2/v3 の候補失敗 policy が一致しません", verificationID)
	}
	for _, invalid := range []string{"current", "legal-query-evaluator-v*", "legal-query-evaluator-v4"} {
		if _, err := New(invalid); err == nil {
			t.Fatalf("profile-set-evaluator-version-identity: %q を受理しました", invalid)
		}
	}
}

func TestCurrentVersionはVersion3を指す(t *testing.T) {
	t.Parallel()

	if CurrentVersion != Version3 {
		t.Fatalf("candidate-evaluation-schema-v3-exact-evaluator-binding: current=%q version3=%q", CurrentVersion, Version3)
	}
}

func TestIsSupportedは実装済みのExactVersionだけを返す(t *testing.T) {
	t.Parallel()

	for _, version := range []string{Version1, Version2, Version3} {
		if !IsSupported(version) {
			t.Fatalf("profile-set-evaluator-version-identity: %q を未対応と判定しました", version)
		}
	}
	for _, version := range []string{"", "current", "legal-query-evaluator-v4"} {
		if IsSupported(version) {
			t.Fatalf("profile-set-evaluator-version-identity: %q を対応済みと判定しました", version)
		}
	}
}
