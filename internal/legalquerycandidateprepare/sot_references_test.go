package legalquerycandidateprepare

import (
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

func TestBuildRequiredSOTReferencesは固定Indexから有効文書だけを解決する(
	t *testing.T,
) {
	t.Parallel()

	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("candidate-evaluation-review-content-binding: repository を解決できません: %v", err)
	}
	references, err := BuildRequiredSOTReferences(t.Context(), repository)
	if err != nil {
		t.Fatalf("candidate-evaluation-review-content-binding: SOT 集合を解決できません: %v", err)
	}
	want := legalquerycandidateeval.RequiredReviewSOTIDs()
	if len(references) != len(want) {
		t.Fatalf("candidate-evaluation-review-content-binding: SOT 件数 = %d, want %d", len(references), len(want))
	}
	for index, reference := range references {
		if reference.SOTID != want[index] || len(reference.SOTDocumentSHA256) != 64 {
			t.Fatalf("candidate-evaluation-review-content-binding: SOT[%d] = %#v", index, reference)
		}
	}
	if got := legalquerycandidateeval.SOTSetSHA256(references); len(got) != 64 {
		t.Fatalf("candidate-evaluation-review-content-binding: set digest = %q", got)
	}
}
