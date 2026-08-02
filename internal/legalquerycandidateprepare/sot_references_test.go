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
	for _, schemaVersion := range []int{
		legalquerycandidateeval.SchemaVersionV2,
		legalquerycandidateeval.SchemaVersionV3,
	} {
		references, err := BuildRequiredSOTReferences(t.Context(), repository, schemaVersion)
		if err != nil {
			t.Fatalf("candidate-evaluation-review-content-binding: schema version %d の SOT 集合を解決できません: %v", schemaVersion, err)
		}
		want, err := legalquerycandidateeval.RequiredReviewSOTIDsForSchema(schemaVersion)
		if err != nil {
			t.Fatalf("schema version %d の SOT ID 集合を解決できません: %v", schemaVersion, err)
		}
		if len(references) != len(want) {
			t.Fatalf("candidate-evaluation-review-content-binding: schema version %d の SOT 件数 = %d, want %d", schemaVersion, len(references), len(want))
		}
		for index, reference := range references {
			if reference.SOTID != want[index] || len(reference.SOTDocumentSHA256) != 64 {
				t.Fatalf("candidate-evaluation-review-content-binding: schema version %d の SOT[%d] = %#v", schemaVersion, index, reference)
			}
		}
		if got := legalquerycandidateeval.SOTSetSHA256(references); len(got) != 64 {
			t.Fatalf("candidate-evaluation-review-content-binding: schema version %d の set digest = %q", schemaVersion, got)
		}
	}
}
