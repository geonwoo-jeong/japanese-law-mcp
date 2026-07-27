package judicialdecisionsearch_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type testSearchPort struct {
	page judicialdecisionsearch.Page
}

func (p testSearchPort) Search(
	_ context.Context,
	_ judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	return p.page, nil
}

var _ judicialdecisionsearch.Port = testSearchPort{}

func TestPageCopiesItemsAndPreservesOrderAndDuplicates(t *testing.T) {
	t.Parallel()

	first := newJudicialSearchItem(t, judicialItemValues{
		decisionID: "100",
		resourceID: "provider:opaque-first",
	})
	second := newJudicialSearchItem(t, judicialItemValues{
		decisionID: "200",
		resourceID: "provider:opaque-second",
	})
	input := []model.SourcedResource[model.JudicialDecisionSummary]{
		second,
		first,
		first,
	}
	page, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{
			Items: input,
			Page:  newSourcePage(t, 3),
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-015/SOT-IF-041: NewPage() のエラー = %v", err)
	}

	input[0] = model.SourcedResource[model.JudicialDecisionSummary]{}
	got := page.Items()
	if len(got) != 3 ||
		got[0].Data().DecisionID() != "200" ||
		got[1].Data().DecisionID() != "100" ||
		got[2].Data().DecisionID() != "100" {
		t.Fatalf("SOT-IF-041: Items() = %#v", got)
	}
	got[0] = model.SourcedResource[model.JudicialDecisionSummary]{}
	if page.Items()[0].Data().DecisionID() != "200" {
		t.Fatal("SOT-IF-015/SOT-IF-041: items が外部から変更された")
	}
	if page.Page().ReturnedCount() != 3 {
		t.Fatalf("SOT-IF-041: returnedCount = %d", page.Page().ReturnedCount())
	}
	if err := page.Validate(); err != nil {
		t.Fatalf("SOT-IF-041: Validate() のエラー = %v", err)
	}

	request := mustSearchRequest(t)
	portPage, portErr := (testSearchPort{page: page}).Search(
		context.Background(),
		request,
	)
	if portErr != nil || len(portPage.Items()) != 3 {
		t.Fatalf("SOT-ARCH-017/SOT-IF-041: Port.Search() = %#v, %v", portPage, portErr)
	}
}

func TestPageAcceptsProviderOwnedResourceID(t *testing.T) {
	t.Parallel()

	item := newJudicialSearchItem(t, judicialItemValues{
		decisionID: "95570",
		resourceID: "vendor:decision/95570/revision-current",
	})
	if _, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{
			Items: []model.SourcedResource[model.JudicialDecisionSummary]{item},
			Page:  newSourcePage(t, 1),
		},
	); err != nil {
		t.Fatalf("SOT-IF-041: provider 所有の resourceId を拒否した: %v", err)
	}
}

func TestPageValidatesJudicialTypeVersionAndSourceIdentity(t *testing.T) {
	t.Parallel()

	tests := map[string]judicialItemValues{
		"resourceType 不一致": {
			resourceType: "law",
		},
		"versionId が存在": {
			versionID: "revision-1",
		},
		"sourceId 不一致": {
			refSourceID: "courts-mirror",
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			item := newJudicialSearchItem(t, values)
			if _, err := judicialdecisionsearch.NewPage(
				judicialdecisionsearch.PageValues{
					Items: []model.SourcedResource[model.JudicialDecisionSummary]{item},
					Page:  newSourcePage(t, 1),
				},
			); err == nil {
				t.Fatal("SOT-IF-015/SOT-IF-041: 不整合な item を受理した")
			}
		})
	}
}

func TestPageSupportsSuccessfulEmptyResult(t *testing.T) {
	t.Parallel()

	page, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{Page: newSourcePage(t, 0)},
	)
	if err != nil {
		t.Fatalf("SOT-IF-041: 空結果の NewPage() エラー = %v", err)
	}
	if items := page.Items(); items == nil || len(items) != 0 {
		t.Fatalf("SOT-IF-041: 空の Items() = %#v", items)
	}

	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("SOT-IF-041: 空結果の json.Marshal() エラー = %v", err)
	}
	if string(encoded) != `{"items":[],"page":{"returnedCount":0}}` {
		t.Fatalf("SOT-IF-041: 空結果の JSON = %s", encoded)
	}
}

func TestPageRejectsReturnedCountMismatchAndInvalidItem(t *testing.T) {
	t.Parallel()

	if _, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{Page: newSourcePage(t, 1)},
	); err == nil {
		t.Fatal("SOT-IF-041: returnedCount と items の不一致を受理した")
	}
	if _, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{
			Items: []model.SourcedResource[model.JudicialDecisionSummary]{{}},
			Page:  newSourcePage(t, 1),
		},
	); err == nil {
		t.Fatal("SOT-IF-015/SOT-IF-041: 無効な item を受理した")
	}
}

func TestPageRejectsDirectJSONDecodeAndInvalidZeroValue(t *testing.T) {
	t.Parallel()

	var page judicialdecisionsearch.Page
	if err := json.Unmarshal(
		[]byte(`{"items":[],"page":{"returnedCount":0}}`),
		&page,
	); err == nil {
		t.Fatal("SOT-ENG-002/SOT-IF-041: Page を JSON から直接復元できた")
	}
	if err := page.Validate(); err == nil {
		t.Fatal("SOT-IF-041: Page のゼロ値を受理した")
	}
}

func newSourcePage(t *testing.T, returnedCount int) model.SourcePage {
	t.Helper()

	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: returnedCount,
	})
	if err != nil {
		t.Fatalf("SourcePage を作成できない: %v", err)
	}
	return page
}

type judicialItemValues struct {
	decisionID   string
	resourceType string
	resourceID   string
	versionID    string
	refSourceID  string
}

func newJudicialSearchItem(
	t *testing.T,
	values judicialItemValues,
) model.SourcedResource[model.JudicialDecisionSummary] {
	t.Helper()

	decisionID := values.decisionID
	if decisionID == "" {
		decisionID = "95570"
	}
	resourceType := values.resourceType
	if resourceType == "" {
		resourceType = "judicial-decision"
	}
	resourceID := values.resourceID
	if resourceID == "" {
		resourceID = decisionID + "/detail2"
	}
	refSourceID := values.refSourceID
	if refSourceID == "" {
		refSourceID = "courts-hanrei"
	}

	dataSource := newJudicialInformationSource(t, "courts-hanrei")
	decisionDate, err := model.NewDate("2025-03-03")
	if err != nil {
		t.Fatalf("Date を作成できない: %v", err)
	}
	summary, err := model.NewJudicialDecisionSummary(
		model.JudicialDecisionSummaryValues{
			DecisionID:          decisionID,
			PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
			SourceCategoryLabel: "最高裁判例",
			CaseNumber:          "令和6年（受）第1号",
			DecisionDate:        decisionDate,
			CourtName:           "最高裁判所",
			DetailURL: "https://www.courts.go.jp/hanrei/" +
				decisionID + "/detail2/index.html",
			Documents: []model.JudicialDocumentLink{},
			Source:    dataSource,
		},
	)
	if err != nil {
		t.Fatalf("JudicialDecisionSummary を作成できない: %v", err)
	}

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     refSourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		VersionID:    values.versionID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "test-judicial-provider",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	refSource := newJudicialInformationSource(t, refSourceID)
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:      refSource,
		ResourceKey: key,
		URL: "https://www.courts.go.jp/hanrei/search1/index.html?" +
			"query1=%E6%B0%91%E6%B3%95",
		RetrievedAt:    time.Date(2026, time.July, 27, 1, 2, 3, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-044",
	})
	if err != nil {
		t.Fatalf("Provenance を作成できない: %v", err)
	}
	item, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionSummary]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       summary,
		},
	)
	if err != nil {
		t.Fatalf("SourcedResource を作成できない: %v", err)
	}
	return item
}

func newJudicialInformationSource(
	t *testing.T,
	sourceID string,
) model.InformationSource {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できない: %v", err)
	}
	return source
}
