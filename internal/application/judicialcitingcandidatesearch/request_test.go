package judicialcitingcandidatesearch

import (
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestNewRequest(t *testing.T) {
	t.Parallel()

	request, err := NewRequest(RequestValues{
		Target: newTestTargetResource(t),
	})
	if err != nil {
		t.Fatalf("SOT-IF-069: NewRequest() のエラー = %v", err)
	}
	if request.Limit() != DefaultLimit {
		t.Fatalf("default limit = %d", request.Limit())
	}
}

func TestRequestRejectsInvalidTargetAndLimit(t *testing.T) {
	t.Parallel()

	invalidVersioned := newTestTargetResource(t)
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "00123/detail2",
		VersionID:    "v1",
	})
	if err != nil {
		t.Fatalf("versioned key: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("versioned ref: %v", err)
	}
	_, err = model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionDetails]{
		Ref:        ref,
		Provenance: invalidVersioned.Provenance(),
		Data:       invalidVersioned.Data(),
	})
	if err == nil {
		t.Fatal("versionId 付き target を作成できました")
	}

	for _, testCase := range []struct {
		name   string
		limit  int
		target model.SourcedResource[model.JudicialDecisionDetails]
	}{
		{name: "0", limit: 0, target: newTestTargetResource(t)},
		{name: "11", limit: 11, target: newTestTargetResource(t)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewRequest(RequestValues{
				Target: testCase.target,
				Limit:  &testCase.limit,
			})
			if err == nil {
				t.Fatal("無効な入力を受理しました")
			}
		})
	}
}

func newTestTargetResource(
	t *testing.T,
) model.SourcedResource[model.JudicialDecisionDetails] {
	t.Helper()

	source := hanreiSource(t)
	date, err := model.NewDate("2026-08-26")
	if err != nil {
		t.Fatalf("date: %v", err)
	}
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          "00123",
		PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
		SourceCategoryLabel: "最高裁判例",
		CaseNumber:          "令和6(受)123",
		DecisionDate:        date,
		CourtName:           "最高裁判所",
		DetailURL:           "https://www.courts.go.jp/hanrei/00123/detail2/index.html",
		Documents:           []model.JudicialDocumentLink{},
		Source:              source,
	})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	reporterCitation := "民集第80巻1号1頁"
	details, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary:          summary,
		ReporterCitation: &reporterCitation,
	})
	if err != nil {
		t.Fatalf("details: %v", err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "00123/detail2",
	})
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("ref: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            summary.DetailURL(),
		RetrievedAt:    time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-045",
	})
	if err != nil {
		t.Fatalf("provenance: %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.JudicialDecisionDetails]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       details,
	})
	if err != nil {
		t.Fatalf("resource: %v", err)
	}
	return resource
}

func hanreiSource(t *testing.T) model.InformationSource {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "courts-hanrei",
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	return source
}
