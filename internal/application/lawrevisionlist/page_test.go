package lawrevisionlist

import (
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestPageAcceptsExactCompleteRevisionListAndClonesItems(t *testing.T) {
	t.Parallel()

	first := newLawRevisionResource(t, "revision-2")
	second := newLawRevisionResource(t, "revision-1")
	items := []model.SourcedResource[model.LawRevision]{first, second}
	page, err := NewPage(PageValues{
		LawID: "law-1",
		Items: items,
		Page:  newLawRevisionSourcePage(t, 2),
	})
	if err != nil {
		t.Fatalf("有効な Page を拒否しました: %v", err)
	}
	items[0] = second
	got := page.Items()
	if got[0].Data().RevisionID() != "revision-2" {
		t.Fatalf("Page が入力 slice の変更を受けました: %q", got[0].Data().RevisionID())
	}
	got[0] = second
	if page.Items()[0].Data().RevisionID() != "revision-2" {
		t.Fatal("Items が内部 slice を公開しました")
	}
}

func TestPageRejectsMismatchedOrDuplicateRevisions(t *testing.T) {
	t.Parallel()

	valid := newLawRevisionResource(t, "revision-1")
	wrongLaw := newLawRevisionResourceForLaw(t, "law-2", "revision-2")
	tests := map[string][]model.SourcedResource[model.LawRevision]{
		"異なる法令": {valid, wrongLaw},
		"重複履歴":  {valid, valid},
	}
	for name, items := range tests {
		name, items := name, items
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewPage(PageValues{
				LawID: "law-1",
				Items: items,
				Page:  newLawRevisionSourcePage(t, len(items)),
			}); err == nil {
				t.Fatal("不正な Page を拒否しませんでした")
			}
		})
	}
}

func TestPageRejectsContinuationAndInexactCounts(t *testing.T) {
	t.Parallel()

	resource := newLawRevisionResource(t, "revision-1")
	count := 1
	tests := map[string]model.SourcePageValues{
		"継続取得": {
			ReturnedCount: count,
			NextToken:     "next",
			TotalCount:    &count,
			TotalRelation: model.TotalRelationExact,
		},
		"件数不一致": {
			ReturnedCount: count,
			TotalCount:    countPointer(count + 1),
			TotalRelation: model.TotalRelationExact,
		},
		"下限件数": {
			ReturnedCount: count,
			TotalCount:    &count,
			TotalRelation: model.TotalRelationLowerBound,
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			page, err := model.NewSourcePage(values)
			if err != nil {
				t.Fatalf("SourcePage を作成できません: %v", err)
			}
			if _, err := NewPage(PageValues{
				LawID: "law-1",
				Items: []model.SourcedResource[model.LawRevision]{resource},
				Page:  page,
			}); err == nil {
				t.Fatal("不正な page 制約を拒否しませんでした")
			}
		})
	}
}

func newLawRevisionSourcePage(t *testing.T, count int) model.SourcePage {
	t.Helper()
	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: count,
		TotalCount:    &count,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("SourcePage を作成できません: %v", err)
	}
	return page
}

func countPointer(value int) *int {
	return &value
}

func newLawRevisionResource(
	t *testing.T,
	revisionID string,
) model.SourcedResource[model.LawRevision] {
	t.Helper()
	return newLawRevisionResourceForLaw(t, "law-1", revisionID)
}

func newLawRevisionResourceForLaw(
	t *testing.T,
	lawID string,
	revisionID string,
) model.SourcedResource[model.LawRevision] {
	t.Helper()
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "provider-source",
		Name:       "試験用公式情報源",
		Publisher:  "試験機関",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://example.test/service",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("LegalSource を投影できません: %v", err)
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
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     source.ID(),
		ResourceType: "law",
		ResourceID:   lawID,
		VersionID:    revisionID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "provider-1",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できません: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         informationSource,
		ResourceKey:    key,
		URL:            "https://example.test/law/" + lawID + "/" + revisionID,
		RetrievedAt:    time.Date(2026, time.August, 23, 0, 0, 0, 0, time.UTC),
		MediaType:      "application/json",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-055",
	})
	if err != nil {
		t.Fatalf("Provenance を作成できません: %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.LawRevision]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       revision,
	})
	if err != nil {
		t.Fatalf("SourcedResource を作成できません: %v", err)
	}
	return resource
}
