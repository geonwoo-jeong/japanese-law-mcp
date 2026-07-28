package application_test

import (
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type judicialFacadePayload struct {
	ref     model.SourceResourceRef
	summary model.SourcedResource[model.JudicialDecisionSummary]
	details model.SourcedResource[model.JudicialDecisionDetails]
	page    judicialdecisionsearch.Page
}

func newJudicialFacadePayload(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceID string,
) judicialFacadePayload {
	t.Helper()

	source := mustJudicialFacadeInformationSource(t, sourceID)
	summary, err := model.NewJudicialDecisionSummary(
		model.JudicialDecisionSummaryValues{
			DecisionID:          resourceID,
			PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
			SourceCategoryLabel: "最高裁判例",
			CaseNumber:          "令和六年（受）第一号",
			DecisionDate:        mustCoreFacadeDate(t, "2026-07-27"),
			CourtName:           "最高裁判所",
			DetailURL:           "https://www.courts.go.jp/hanrei/1/detail2/index.html",
			Documents:           []model.JudicialDocumentLink{},
			Source:              source,
		},
	)
	if err != nil {
		t.Fatalf("試験用 JudicialDecisionSummary を作成できません: %v", err)
	}
	details, err := model.NewJudicialDecisionDetails(
		model.JudicialDecisionDetailsValues{Summary: summary},
	)
	if err != nil {
		t.Fatalf("試験用 JudicialDecisionDetails を作成できません: %v", err)
	}
	ref := mustJudicialFacadeRef(t, providerID, sourceID, resourceID)
	sourcedSummary := mustJudicialFacadeSourced(t, source, ref, summary)
	sourcedDetails := mustJudicialFacadeSourced(t, source, ref, details)
	return judicialFacadePayload{
		ref:     ref,
		summary: sourcedSummary,
		details: sourcedDetails,
		page:    mustJudicialFacadePage(t, []model.SourcedResource[model.JudicialDecisionSummary]{sourcedSummary}),
	}
}

func mustJudicialFacadeInformationSource(
	t *testing.T,
	sourceID string,
) model.InformationSource {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "裁判所裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/",
	})
	if err != nil {
		t.Fatalf("試験用 InformationSource を作成できません: %v", err)
	}
	return source
}

func mustJudicialFacadeRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceID string,
) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "judicial-decision",
		ResourceID:   resourceID,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を作成できません: %v", err)
	}
	return ref
}

func mustJudicialFacadeSourced[T interface{ Validate() error }](
	t *testing.T,
	source model.InformationSource,
	ref model.SourceResourceRef,
	data T,
) model.SourcedResource[T] {
	t.Helper()

	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    ref.Key(),
		URL:            "https://www.courts.go.jp/hanrei/1/detail2/index.html",
		RetrievedAt:    time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationUnchanged,
	})
	if err != nil {
		t.Fatalf("試験用 Provenance を作成できません: %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[T]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       data,
	})
	if err != nil {
		t.Fatalf("試験用 SourcedResource を作成できません: %v", err)
	}
	return resource
}

func mustJudicialFacadePage(
	t *testing.T,
	items []model.SourcedResource[model.JudicialDecisionSummary],
) judicialdecisionsearch.Page {
	t.Helper()

	totalCount := len(items)
	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: len(items),
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("試験用 SourcePage を作成できません: %v", err)
	}
	result, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{Items: items, Page: page},
	)
	if err != nil {
		t.Fatalf("試験用 judicial decision search page を作成できません: %v", err)
	}
	return result
}
