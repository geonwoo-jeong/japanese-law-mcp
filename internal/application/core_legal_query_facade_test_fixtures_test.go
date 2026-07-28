package application_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type coreFacadePayloadFixture struct {
	lawSummary  model.SourcedResource[model.LawSummary]
	lawContent  model.SourcedResource[model.LawContentMatch]
	lawDocument model.SourcedResource[model.LawDocumentRepresentation]
	lawArticle  model.SourcedResource[model.LawArticleFragment]
}

func newCoreFacadePayloadFixture(
	t *testing.T,
	providerID string,
	sourceID string,
	location model.LawArticleLocation,
) coreFacadePayloadFixture {
	t.Helper()

	source := mustCoreFacadeInformationSource(t, sourceID)
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("試験用 LegalSource を作成できません: %v", err)
	}
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "revision-1",
		Title:      "行政手続法",
		Source:     legalSource,
	})
	if err != nil {
		t.Fatalf("試験用 LawSummary を作成できません: %v", err)
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     legalSource,
		LawID:      law.LawID(),
		RevisionID: law.RevisionID(),
		Location:   "main:article=1",
		URL:        source.ServiceURL(),
	})
	if err != nil {
		t.Fatalf("試験用 Citation を作成できません: %v", err)
	}
	content, err := model.NewLawContentMatch(model.LawContentMatchValues{
		Law:      law,
		Location: "main:article=1",
		Text:     "許可の申請に関する本文",
		Citation: citation,
	})
	if err != nil {
		t.Fatalf("試験用 LawContentMatch を作成できません: %v", err)
	}
	document, err := model.NewLawDocumentRepresentation(
		model.LawDocumentRepresentationValues{
			Law:      law,
			Format:   model.LawDocumentFormatText,
			Content:  "法令本文",
			Citation: citation,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawDocumentRepresentation を作成できません: %v", err)
	}
	article, err := model.NewLawArticleFragment(
		model.LawArticleFragmentValues{
			Law:      law,
			Location: location,
			Format:   model.LawArticleFormatText,
			Content:  "第一条の本文",
			Citation: citation,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawArticleFragment を作成できません: %v", err)
	}
	ref := mustCoreFacadeLawRef(
		t,
		providerID,
		sourceID,
		law.LawID(),
		law.RevisionID(),
	)
	return coreFacadePayloadFixture{
		lawSummary: mustCoreFacadeSourced(
			t,
			ref,
			source,
			"",
			law,
		),
		lawContent: mustCoreFacadeSourced(
			t,
			ref,
			source,
			"main:article=1",
			content,
		),
		lawDocument: mustCoreFacadeSourced(
			t,
			ref,
			source,
			"",
			document,
		),
		lawArticle: mustCoreFacadeSourced(
			t,
			ref,
			source,
			"main:article=1",
			article,
		),
	}
}

func coreFacadeLawSearchPage(
	t *testing.T,
	values []coreFacadeLawResourceValues,
	providerID string,
	sourceID string,
) lawsearch.Page {
	t.Helper()

	source := mustCoreFacadeInformationSource(t, sourceID)
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("試験用 LegalSource を作成できません: %v", err)
	}
	items := make([]model.SourcedResource[model.LawSummary], 0, len(values))
	for index, value := range values {
		law, lawErr := model.NewLawSummary(model.LawSummaryValues{
			LawID:      value.lawID,
			RevisionID: value.revisionID,
			Title:      fmt.Sprintf("試験法令%d", index+1),
			Source:     legalSource,
		})
		if lawErr != nil {
			t.Fatalf("試験用 LawSummary を作成できません: %v", lawErr)
		}
		ref := mustCoreFacadeLawRef(
			t,
			providerID,
			sourceID,
			value.lawID,
			value.revisionID,
		)
		items = append(items, mustCoreFacadeSourced(t, ref, source, "", law))
	}
	return mustCoreFacadeLawSearchPage(t, items)
}

func coreFacadeLawUpdatePage(
	t *testing.T,
	providerID string,
	sourceID string,
	date model.Date,
	count int,
) lawupdatelist.Page {
	t.Helper()

	source := mustCoreFacadeInformationSource(t, sourceID)
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("試験用 LegalSource を作成できません: %v", err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "law-update-list",
		ResourceID:   date.String(),
	})
	if err != nil {
		t.Fatalf("試験用更新一覧 key を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用更新一覧 ref を作成できません: %v", err)
	}
	items := make([]model.SourcedResource[model.LawUpdate], 0, count)
	for index := 0; index < count; index++ {
		update, updateErr := model.NewLawUpdate(model.LawUpdateValues{
			UpdatedOn: date,
			LawID:     fmt.Sprintf("law-update-%d", index+1),
			Title:     fmt.Sprintf("更新法令%d", index+1),
			Source:    legalSource,
		})
		if updateErr != nil {
			t.Fatalf("試験用 LawUpdate を作成できません: %v", updateErr)
		}
		items = append(
			items,
			mustCoreFacadeSourced(t, ref, source, "", update),
		)
	}
	totalCount := count
	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: count,
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("試験用 SourcePage を作成できません: %v", err)
	}
	result, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
		Items: items,
		Page:  page,
		Date:  date,
	})
	if err != nil {
		t.Fatalf("試験用 law update page を作成できません: %v", err)
	}
	return result
}

func mustCoreFacadeLawSearchPage(
	t *testing.T,
	items []model.SourcedResource[model.LawSummary],
) lawsearch.Page {
	t.Helper()
	page := mustCoreFacadeSourcePage(t, len(items))
	result, err := lawsearch.NewPage(lawsearch.PageValues{
		Items: items,
		Page:  page,
	})
	if err != nil {
		t.Fatalf("試験用 law search page を作成できません: %v", err)
	}
	return result
}

func mustCoreFacadeLawContentPage(
	t *testing.T,
	items []model.SourcedResource[model.LawContentMatch],
) lawcontentsearch.Page {
	t.Helper()
	page := mustCoreFacadeSourcePage(t, len(items))
	result, err := lawcontentsearch.NewPage(lawcontentsearch.PageValues{
		Items: items,
		Page:  page,
	})
	if err != nil {
		t.Fatalf("試験用 law content page を作成できません: %v", err)
	}
	return result
}

func mustCoreFacadeSourcePage(t *testing.T, count int) model.SourcePage {
	t.Helper()
	totalCount := count
	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: count,
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("試験用 SourcePage を作成できません: %v", err)
	}
	return page
}

func mustCoreFacadeInformationSource(
	t *testing.T,
	sourceID string,
) model.InformationSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "公式情報源",
		Publisher:  "公的機関",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://example.test/",
	})
	if err != nil {
		t.Fatalf("試験用 InformationSource を作成できません: %v", err)
	}
	return source
}

func mustCoreFacadeSourced[T interface{ Validate() error }](
	t *testing.T,
	ref model.SourceResourceRef,
	source model.InformationSource,
	location string,
	data T,
) model.SourcedResource[T] {
	t.Helper()
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    ref.Key(),
		URL:            source.ServiceURL(),
		RetrievedAt:    time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC),
		MediaType:      "application/json",
		Location:       location,
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

func coreFacadeZeroDocumentResult() model.SourcedResource[model.LawDocumentRepresentation] {
	return model.SourcedResource[model.LawDocumentRepresentation]{}
}

func coreFacadeLawDocumentResult(
	t *testing.T,
	providerID string,
	sourceID string,
	lawID string,
	revisionID string,
	asOf *model.Date,
) model.SourcedResource[model.LawDocumentRepresentation] {
	t.Helper()
	source := mustCoreFacadeInformationSource(t, sourceID)
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("試験用 LegalSource を作成できません: %v", err)
	}
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      lawID,
		RevisionID: revisionID,
		Title:      "試験法令本文",
		Source:     legalSource,
	})
	if err != nil {
		t.Fatalf("試験用 LawSummary を作成できません: %v", err)
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     legalSource,
		LawID:      law.LawID(),
		RevisionID: law.RevisionID(),
		Location:   "main:article=1",
		URL:        source.ServiceURL(),
	})
	if err != nil {
		t.Fatalf("試験用 Citation を作成できません: %v", err)
	}
	document, err := model.NewLawDocumentRepresentation(
		model.LawDocumentRepresentationValues{
			Law:      law,
			AsOf:     asOf,
			Format:   model.LawDocumentFormatText,
			Content:  "別の法令本文",
			Citation: citation,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawDocumentRepresentation を作成できません: %v", err)
	}
	ref := mustCoreFacadeLawRef(t, providerID, sourceID, lawID, revisionID)
	return mustCoreFacadeSourced(t, ref, source, "", document)
}

func coreFacadeLawArticleResult(
	t *testing.T,
	providerID string,
	sourceID string,
	lawID string,
	revisionID string,
	location model.LawArticleLocation,
) model.SourcedResource[model.LawArticleFragment] {
	t.Helper()
	source := mustCoreFacadeInformationSource(t, sourceID)
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("試験用 LegalSource を作成できません: %v", err)
	}
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      lawID,
		RevisionID: revisionID,
		Title:      "試験条文",
		Source:     legalSource,
	})
	if err != nil {
		t.Fatalf("試験用 LawSummary を作成できません: %v", err)
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     legalSource,
		LawID:      law.LawID(),
		RevisionID: law.RevisionID(),
		Location:   "main:article=1",
		URL:        source.ServiceURL(),
	})
	if err != nil {
		t.Fatalf("試験用 Citation を作成できません: %v", err)
	}
	article, err := model.NewLawArticleFragment(model.LawArticleFragmentValues{
		Law:      law,
		Location: location,
		Format:   model.LawArticleFormatText,
		Content:  "別の条文本文",
		Citation: citation,
	})
	if err != nil {
		t.Fatalf("試験用 LawArticleFragment を作成できません: %v", err)
	}
	ref := mustCoreFacadeLawRef(t, providerID, sourceID, lawID, revisionID)
	return mustCoreFacadeSourced(t, ref, source, "main:article=1", article)
}
