package evaluators

import "testing"

func TestNewはExactEvaluatorVersionだけを受理する(t *testing.T) {
	t.Parallel()

	if _, err := New(Version1); err != nil {
		t.Fatalf("profile-set-evaluator-version-identity: v1 を拒否しました: %v", err)
	}
	if _, err := New(Version2); err != nil {
		t.Fatalf("profile-set-evaluator-version-identity: v2 を拒否しました: %v", err)
	}
	for _, invalid := range []string{"current", "legal-query-evaluator-v*", "legal-query-evaluator-v3"} {
		if _, err := New(invalid); err == nil {
			t.Fatalf("profile-set-evaluator-version-identity: %q を受理しました", invalid)
		}
	}
}

func TestCurrentVersionはVersion2を指す(t *testing.T) {
	t.Parallel()

	if CurrentVersion != Version2 {
		t.Fatalf("profile-set-evaluator-version-identity: current=%q version2=%q", CurrentVersion, Version2)
	}
}

func TestIsSupportedは実装済みのExactVersionだけを返す(t *testing.T) {
	t.Parallel()

	for _, version := range []string{Version1, Version2} {
		if !IsSupported(version) {
			t.Fatalf("profile-set-evaluator-version-identity: %q を未対応と判定しました", version)
		}
	}
	for _, version := range []string{"", "current", "legal-query-evaluator-v3"} {
		if IsSupported(version) {
			t.Fatalf("profile-set-evaluator-version-identity: %q を対応済みと判定しました", version)
		}
	}
}
