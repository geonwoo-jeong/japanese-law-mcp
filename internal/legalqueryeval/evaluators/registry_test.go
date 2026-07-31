package evaluators

import "testing"

func TestNewはExactEvaluatorVersionだけを受理する(t *testing.T) {
	t.Parallel()

	if _, err := New(Version1); err != nil {
		t.Fatalf("profile-set-evaluator-version-identity: v1 を拒否しました: %v", err)
	}
	for _, invalid := range []string{"current", "legal-query-evaluator-v*", "legal-query-evaluator-v2"} {
		if _, err := New(invalid); err == nil {
			t.Fatalf("profile-set-evaluator-version-identity: %q を受理しました", invalid)
		}
	}
}
