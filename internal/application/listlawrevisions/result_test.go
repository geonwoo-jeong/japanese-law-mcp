package listlawrevisions

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestResultClonesItemsAndPreservesOrder(t *testing.T) {
	t.Parallel()

	first := mustResultLawRevision(t, "law-1", "revision-2")
	second := mustResultLawRevision(t, "law-1", "revision-1")
	items := []model.LawRevision{first, second}
	result, err := NewResult(ResultValues{
		LawID:      "law-1",
		TotalCount: 2,
		Items:      items,
	})
	if err != nil {
		t.Fatalf("有効な Result を拒否しました: %v", err)
	}

	items[0] = second
	if result.Items()[0].RevisionID() != "revision-2" {
		t.Fatalf("Result が入力 slice の変更を受けました: %#v", result.Items())
	}

	got := result.Items()
	got[0] = second
	if result.Items()[0].RevisionID() != "revision-2" {
		t.Fatal("Items が内部 slice を公開しました")
	}
}

func TestResultRejectsMismatchedCountsAndDuplicateRevisions(t *testing.T) {
	t.Parallel()

	first := mustResultLawRevision(t, "law-1", "revision-2")
	second := mustResultLawRevision(t, "law-1", "revision-1")
	otherLaw := mustResultLawRevision(t, "law-2", "revision-3")
	tests := map[string]ResultValues{
		"件数不一致": {
			LawID:      "law-1",
			TotalCount: 2,
			Items:      []model.LawRevision{first},
		},
		"異なる法令": {
			LawID:      "law-1",
			TotalCount: 2,
			Items:      []model.LawRevision{first, otherLaw},
		},
		"重複履歴": {
			LawID:      "law-1",
			TotalCount: 2,
			Items:      []model.LawRevision{first, first},
		},
		"負の totalCount": {
			LawID:      "law-1",
			TotalCount: -1,
			Items:      nil,
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewResult(values); err == nil {
				t.Fatal("不正な Result を拒否しませんでした")
			}
		})
	}

	if _, err := NewResult(ResultValues{
		LawID:      "law-1",
		TotalCount: 2,
		Items:      []model.LawRevision{first, second},
	}); err != nil {
		t.Fatalf("有効な Result を拒否しました: %v", err)
	}
}

func mustResultLawRevision(t *testing.T, lawID string, revisionID string) model.LawRevision {
	t.Helper()
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "test-source",
		Name:       "試験用情報源",
		Publisher:  "試験機関",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://example.test/service",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("LegalSource を作成できません: %v", err)
	}
	revision, err := model.NewLawRevision(model.LawRevisionValues{
		LawID:      lawID,
		RevisionID: revisionID,
		Title:      "試験法",
		Source:     source,
	})
	if err != nil {
		t.Fatalf("LawRevision を作成できません: %v", err)
	}
	return revision
}
