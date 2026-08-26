package judicialcasecitationextract

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequestAcceptsMatchingFullTextDocument(t *testing.T) {
	t.Parallel()

	decision, document := testDecisionResource(t)
	request, err := NewRequest(RequestValues{Decision: decision, Document: document})
	if err != nil {
		t.Fatal(err)
	}
	if request.Decision().Ref() != decision.Ref() || request.Document() != document {
		t.Fatal("Request accessor が入力を保持していません")
	}
}

func TestRequestRejectsMismatchedDocumentBeforeExternalCall(t *testing.T) {
	t.Parallel()

	decision, _ := testDecisionResource(t)
	other, err := model.NewJudicialDocumentLink(model.JudicialDocumentLinkValues{
		Kind:      model.JudicialDocumentKindFullText,
		Label:     "別文書",
		MediaType: model.JudicialDocumentMediaTypePDF,
		URL:       "https://www.courts.go.jp/assets/hanrei/99999.pdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewRequest(RequestValues{Decision: decision, Document: other})
	if err == nil {
		t.Fatal("documents に含まれない PDF を受理しました")
	}
	var argumentError ArgumentError
	if !errors.As(err, &argumentError) || argumentError.Field() != "document" {
		t.Fatalf("document の ArgumentError ではありません: %v", err)
	}
	if strings.Contains(err.Error(), other.URL()) || strings.Contains(argumentError.Reason(), other.URL()) {
		t.Fatal("公開エラーに入力 URL が含まれています")
	}
}

func TestRequestRejectsNonFullTextAndVersionedRef(t *testing.T) {
	t.Parallel()

	decision, _ := testDecisionResource(t)
	summaryDoc, err := model.NewJudicialDocumentLink(model.JudicialDocumentLinkValues{
		Kind:      model.JudicialDocumentKindSummary,
		Label:     "要旨",
		MediaType: model.JudicialDocumentMediaTypePDF,
		URL:       "https://www.courts.go.jp/assets/hanrei/00001-summary.pdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewRequest(RequestValues{Decision: decision, Document: summaryDoc}); err == nil {
		t.Fatal("summary PDF を受理しました")
	}

	versionedDecision := testDecisionResourceWithVersion(t)
	if _, err := NewRequest(RequestValues{Decision: versionedDecision, Document: versionedDecision.Data().Summary().Documents()[0]}); err == nil {
		t.Fatal("versionId 付き ref を受理しました")
	}
}

func TestRequestRequiresDocumentToAppearExactlyOnce(t *testing.T) {
	t.Parallel()

	decision, document := testDecisionResource(t)
	duplicate := testDecisionResourceWithDocuments(
		t,
		decision,
		[]model.JudicialDocumentLink{document, document},
	)
	if _, err := NewRequest(RequestValues{Decision: duplicate, Document: document}); err == nil {
		t.Fatal("同じ document が二回ある decision を受理しました")
	}
}

func TestRequestRejectsSummaryAndLastProvenanceSourceMismatch(t *testing.T) {
	t.Parallel()

	decision, document := testDecisionResource(t)
	differentSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "courts-hanrei-different",
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatal(err)
	}
	summaryMismatch := testDecisionResourceWithSource(t, decision, differentSource, nil)
	if _, err := NewRequest(RequestValues{Decision: summaryMismatch, Document: document}); err == nil {
		t.Fatal("ref と異なる summary.source を受理しました")
	}

	sameIDChangedMetadata, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         decision.Data().Summary().Source().ID(),
		Name:       "異なる情報源名",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatal(err)
	}
	provenanceMismatch := testDecisionResourceWithSource(
		t,
		decision,
		decision.Data().Summary().Source(),
		&sameIDChangedMetadata,
	)
	if _, err := NewRequest(RequestValues{Decision: provenanceMismatch, Document: document}); err == nil {
		t.Fatal("summary.source と異なる最後の provenance.source を受理しました")
	}
}

func TestRequestIsImmutableAndRejectsZeroValueAndDirectRestore(t *testing.T) {
	t.Parallel()

	decision, document := testDecisionResource(t)
	request, err := NewRequest(RequestValues{Decision: decision, Document: document})
	if err != nil {
		t.Fatal(err)
	}
	provenance := request.Decision().Provenance()
	provenance[0] = model.Provenance{}
	if request.Decision().Provenance()[0].Validate() != nil {
		t.Fatal("Request の decision が getter の slice 変更で変化しました")
	}

	var zero Request
	if zero.Validate() == nil {
		t.Fatal("Request のゼロ値を受理しました")
	}
	if json.Unmarshal([]byte(`{"decision":{},"document":{}}`), &zero) == nil {
		t.Fatal("Request の直接 JSON 復元を受理しました")
	}
}

func testDecisionResource(
	t *testing.T,
) (model.SourcedResource[model.JudicialDecisionDetails], model.JudicialDocumentLink) {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "courts-hanrei",
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatal(err)
	}
	document, err := model.NewJudicialDocumentLink(model.JudicialDocumentLinkValues{
		Kind:      model.JudicialDocumentKindFullText,
		Label:     "判決文",
		MediaType: model.JudicialDocumentMediaTypePDF,
		URL:       "https://www.courts.go.jp/assets/hanrei/00001.pdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          "00001",
		PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
		SourceCategoryLabel: "最高裁判例",
		CaseNumber:          "令和6(受)123",
		DecisionDate:        mustDate(t, "2025-01-02"),
		CourtName:           "最高裁判所",
		DetailURL:           "https://www.courts.go.jp/app/hanrei_jp/detail2?id=00001",
		Documents:           []model.JudicialDocumentLink{document},
		Source:              source,
	})
	if err != nil {
		t.Fatal(err)
	}
	details, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary: summary,
	})
	if err != nil {
		t.Fatal(err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "00001",
	})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            summary.DetailURL(),
		RetrievedAt:    time.Date(2026, 8, 26, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationUnchanged,
	})
	if err != nil {
		t.Fatal(err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionDetails]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       details,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resource, document
}

func testDecisionResourceWithVersion(
	t *testing.T,
) model.SourcedResource[model.JudicialDecisionDetails] {
	t.Helper()
	resource, document := testDecisionResource(t)
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     resource.Ref().Key().SourceID(),
		ResourceType: resource.Ref().Key().ResourceType(),
		ResourceID:   resource.Ref().Key().ResourceID(),
		VersionID:    "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: resource.Ref().ProviderID(),
		Key:        key,
	})
	if err != nil {
		t.Fatal(err)
	}
	provenance := resource.Provenance()
	provenance[0], err = model.NewProvenance(model.ProvenanceValues{
		Source:         provenance[0].Source(),
		ResourceKey:    key,
		URL:            provenance[0].URL(),
		RetrievedAt:    provenance[0].RetrievedAt(),
		MediaType:      provenance[0].MediaType(),
		Transformation: provenance[0].Transformation(),
	})
	if err != nil {
		t.Fatal(err)
	}
	versioned, err := model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionDetails]{
		Ref:        ref,
		Provenance: provenance,
		Data:       resource.Data(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = document, versioned
	return versioned
}

func testDecisionResourceWithDocuments(
	t *testing.T,
	resource model.SourcedResource[model.JudicialDecisionDetails],
	documents []model.JudicialDocumentLink,
) model.SourcedResource[model.JudicialDecisionDetails] {
	t.Helper()
	return testDecisionResourceWithSummaryValues(
		t,
		resource,
		resource.Data().Summary().Source(),
		documents,
		resource.Provenance(),
	)
}

func testDecisionResourceWithSource(
	t *testing.T,
	resource model.SourcedResource[model.JudicialDecisionDetails],
	summarySource model.InformationSource,
	lastProvenanceSource *model.InformationSource,
) model.SourcedResource[model.JudicialDecisionDetails] {
	t.Helper()
	provenance := resource.Provenance()
	if lastProvenanceSource != nil {
		var err error
		last := provenance[len(provenance)-1]
		provenance[len(provenance)-1], err = model.NewProvenance(model.ProvenanceValues{
			Source:         *lastProvenanceSource,
			ResourceKey:    last.ResourceKey(),
			URL:            last.URL(),
			RetrievedAt:    last.RetrievedAt(),
			MediaType:      last.MediaType(),
			Transformation: last.Transformation(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return testDecisionResourceWithSummaryValues(
		t,
		resource,
		summarySource,
		resource.Data().Summary().Documents(),
		provenance,
	)
}

func testDecisionResourceWithSummaryValues(
	t *testing.T,
	resource model.SourcedResource[model.JudicialDecisionDetails],
	source model.InformationSource,
	documents []model.JudicialDocumentLink,
	provenance []model.Provenance,
) model.SourcedResource[model.JudicialDecisionDetails] {
	t.Helper()
	current := resource.Data().Summary()
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          current.DecisionID(),
		PublicationCategory: current.PublicationCategory(),
		SourceCategoryLabel: current.SourceCategoryLabel(),
		CaseNumber:          current.CaseNumber(),
		DecisionDate:        current.DecisionDate(),
		CourtName:           current.CourtName(),
		DetailURL:           current.DetailURL(),
		Documents:           documents,
		Source:              source,
	})
	if err != nil {
		t.Fatal(err)
	}
	details, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{Summary: summary})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionDetails]{
		Ref:        resource.Ref(),
		Provenance: provenance,
		Data:       details,
	})
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func mustDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatal(err)
	}
	return date
}
