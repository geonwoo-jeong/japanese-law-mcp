package buildinfo

import "testing"

func TestVersion(t *testing.T) {
	t.Parallel()

	if Version() == "" {
		t.Fatal("SOT-ENG-014: バージョンが空です")
	}
}
