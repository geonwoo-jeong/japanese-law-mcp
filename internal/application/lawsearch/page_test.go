package lawsearch_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type testPort struct {
	page lawsearch.Page
}

func (p testPort) Search(_ context.Context, _ lawsearch.Request) (lawsearch.Page, error) {
	return p.page, nil
}

var _ lawsearch.Port = testPort{}

func TestPageCopiesItemsAndExposesTypedPort(t *testing.T) {
	t.Parallel()

	item := newLawSearchItem(t, itemValues{})
	sourcePage := newSourcePage(t, 1)
	input := []model.SourcedResource[model.LawSummary]{item}
	page, err := lawsearch.NewPage(lawsearch.PageValues{
		Items: input,
		Page:  sourcePage,
	})
	if err != nil {
		t.Fatalf("SOT-IF-015/SOT-IF-022: NewPage() のエラー = %v", err)
	}

	input[0] = model.SourcedResource[model.LawSummary]{}
	got := page.Items()
	if len(got) != 1 || got[0].Data().LawID() != "law-001" {
		t.Fatalf("SOT-IF-022: Items() = %#v", got)
	}
	got[0] = model.SourcedResource[model.LawSummary]{}
	if preserved := page.Items(); preserved[0].Data().LawID() != "law-001" {
		t.Fatal("SOT-IF-015/SOT-IF-022: items が外部から変更された")
	}
	if page.Page() != sourcePage {
		t.Fatalf("SOT-IF-022: Page() = %#v", page.Page())
	}
	if err := page.Validate(); err != nil {
		t.Fatalf("SOT-IF-022: Validate() のエラー = %v", err)
	}

	request, requestErr := lawsearch.NewRequest(lawsearch.RequestValues{Query: "民法"})
	if requestErr != nil {
		t.Fatalf("試験用 Request を作成できない: %v", requestErr)
	}
	portResult, portErr := (testPort{page: page}).Search(context.Background(), request)
	if portErr != nil || len(portResult.Items()) != 1 {
		t.Fatalf("SOT-ARCH-009: Port.Search() = %#v, %v", portResult, portErr)
	}
}

func TestPageSupportsSuccessfulEmptyResult(t *testing.T) {
	t.Parallel()

	page, err := lawsearch.NewPage(lawsearch.PageValues{
		Page: newSourcePage(t, 0),
	})
	if err != nil {
		t.Fatalf("SOT-IF-022: 空結果の NewPage() エラー = %v", err)
	}
	if items := page.Items(); items == nil || len(items) != 0 {
		t.Fatalf("SOT-IF-022: 空の Items() = %#v", items)
	}

	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("SOT-IF-022: 空結果の json.Marshal() エラー = %v", err)
	}
	if string(encoded) != `{"items":[],"page":{"returnedCount":0}}` {
		t.Fatalf("SOT-IF-022: 空結果の JSON = %s", encoded)
	}
}

func TestPageValidatesReturnedCountAndLawIdentity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		item          model.SourcedResource[model.LawSummary]
		returnedCount int
	}{
		"returnedCount 不一致": {
			item:          newLawSearchItem(t, itemValues{}),
			returnedCount: 0,
		},
		"resourceType 不一致": {
			item: newLawSearchItem(t, itemValues{resourceType: "article"}),
		},
		"resourceId 不一致": {
			item: newLawSearchItem(t, itemValues{resourceID: "other-law"}),
		},
		"versionId 不一致": {
			item: newLawSearchItem(t, itemValues{versionID: "other-revision"}),
		},
		"sourceId 不一致": {
			item: newLawSearchItem(t, itemValues{refSourceID: "other-source"}),
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			returnedCount := test.returnedCount
			if returnedCount == 0 && name != "returnedCount 不一致" {
				returnedCount = 1
			}
			_, err := lawsearch.NewPage(lawsearch.PageValues{
				Items: []model.SourcedResource[model.LawSummary]{test.item},
				Page:  newSourcePage(t, returnedCount),
			})
			if err == nil {
				t.Fatal("SOT-IF-015/SOT-IF-022: 不整合な検索結果を受理した")
			}
		})
	}
}

func TestPageRejectsInvalidItemAndDirectJSONDecode(t *testing.T) {
	t.Parallel()

	if _, err := lawsearch.NewPage(lawsearch.PageValues{
		Items: []model.SourcedResource[model.LawSummary]{{}},
		Page:  newSourcePage(t, 1),
	}); err == nil {
		t.Fatal("SOT-IF-015/SOT-IF-022: 無効な item を受理した")
	}

	var page lawsearch.Page
	if err := json.Unmarshal(
		[]byte(`{"items":[],"page":{"returnedCount":0}}`),
		&page,
	); err == nil {
		t.Fatal("SOT-ENG-002/SOT-IF-022: Page を JSON から直接復元できた")
	}
}

func TestPageMarshalJSONUsesCapabilityShape(t *testing.T) {
	t.Parallel()

	page, err := lawsearch.NewPage(lawsearch.PageValues{
		Items: []model.SourcedResource[model.LawSummary]{
			newLawSearchItem(t, itemValues{}),
		},
		Page: newSourcePage(t, 1),
	})
	if err != nil {
		t.Fatalf("SOT-IF-022: NewPage() のエラー = %v", err)
	}
	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("SOT-IF-022: json.Marshal() のエラー = %v", err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-IF-022: JSON を再解析できない: %v", err)
	}
	if len(object) != 2 || object["items"] == nil || object["page"] == nil {
		t.Fatalf("SOT-IF-022: JSON の項目 = %#v", object)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(object["items"], &items); err != nil || len(items) != 1 {
		t.Fatalf("SOT-IF-022: JSON items = %s, %v", object["items"], err)
	}
}

type itemValues struct {
	resourceType string
	resourceID   string
	versionID    string
	refSourceID  string
}

func newLawSearchItem(
	t *testing.T,
	values itemValues,
) model.SourcedResource[model.LawSummary] {
	t.Helper()

	const (
		defaultSourceID     = "official-law-source"
		defaultResourceType = "law"
		defaultLawID        = "law-001"
		defaultRevisionID   = "revision-001"
	)
	refSourceID := values.refSourceID
	if refSourceID == "" {
		refSourceID = defaultSourceID
	}
	resourceType := values.resourceType
	if resourceType == "" {
		resourceType = defaultResourceType
	}
	resourceID := values.resourceID
	if resourceID == "" {
		resourceID = defaultLawID
	}
	versionID := values.versionID
	if versionID == "" {
		versionID = defaultRevisionID
	}

	dataInformationSource := newInformationSource(t, defaultSourceID)
	legalSource, err := model.NewLegalSource(dataInformationSource)
	if err != nil {
		t.Fatalf("LegalSource を作成できない: %v", err)
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      defaultLawID,
		RevisionID: defaultRevisionID,
		Title:      "民法",
		Source:     legalSource,
	})
	if err != nil {
		t.Fatalf("LawSummary を作成できない: %v", err)
	}

	refInformationSource := newInformationSource(t, refSourceID)
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     refSourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "test-law-provider",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         refInformationSource,
		ResourceKey:    key,
		URL:            "https://example.go.jp/laws/" + resourceID,
		RetrievedAt:    time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
		MediaType:      "application/json",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-022",
	})
	if err != nil {
		t.Fatalf("Provenance を作成できない: %v", err)
	}
	item, err := model.NewSourcedResource(model.SourcedResourceValues[model.LawSummary]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       summary,
	})
	if err != nil {
		t.Fatalf("SourcedResource を作成できない: %v", err)
	}
	return item
}

func newInformationSource(t *testing.T, sourceID string) model.InformationSource {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "法令情報源",
		Publisher:  "公的機関",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://example.go.jp/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できない: %v", err)
	}
	return source
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
