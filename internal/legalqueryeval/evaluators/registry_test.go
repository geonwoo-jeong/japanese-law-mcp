package evaluators

import "testing"

func TestNewはExactEvaluatorVersionだけを受理する(t *testing.T) {
	t.Parallel()
	const verificationID = "candidate-evaluator-v4-exact-version-routing"

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
	v4, err := New(Version4)
	if err != nil {
		t.Fatalf("%s: v4 を拒否しました: %v", verificationID, err)
	}
	if v1.ScoresCandidatePlanningFailure() ||
		v2.ScoresCandidatePlanningFailure() ||
		!v3.ScoresCandidatePlanningFailure() ||
		!v4.ScoresCandidatePlanningFailure() {
		t.Fatalf("%s: v1/v2/v3/v4 の候補失敗 policy が一致しません", verificationID)
	}
	for _, invalid := range []string{"current", "legal-query-evaluator-v*", "legal-query-evaluator-v5"} {
		if _, err := New(invalid); err == nil {
			t.Fatalf("profile-set-evaluator-version-identity: %q を受理しました", invalid)
		}
	}
}

func TestCurrentVersionはVersion4を指す(t *testing.T) {
	t.Parallel()

	if CurrentVersion != Version4 {
		t.Fatalf("candidate-evaluation-schema-v5-exact-evaluator-binding: current=%q version4=%q", CurrentVersion, Version4)
	}
}

func TestIsSupportedは実装済みのExactVersionだけを返す(t *testing.T) {
	t.Parallel()

	for _, version := range []string{Version1, Version2, Version3, Version4} {
		if !IsSupported(version) {
			t.Fatalf("profile-set-evaluator-version-identity: %q を未対応と判定しました", version)
		}
	}
	for _, version := range []string{"", "current", "legal-query-evaluator-v5"} {
		if IsSupported(version) {
			t.Fatalf("profile-set-evaluator-version-identity: %q を対応済みと判定しました", version)
		}
	}
}
