package lawcontentsearch_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type contentTestPort struct {
	page lawcontentsearch.Page
}

func (p contentTestPort) Search(
	_ context.Context,
	_ lawcontentsearch.Request,
) (lawcontentsearch.Page, error) {
	return p.page, nil
}

var _ lawcontentsearch.Port = contentTestPort{}

func TestPageCopiesItemsAndExposesTypedPort(t *testing.T) {
	t.Parallel()

	item := newContentSearchItem(t, contentItemValues{})
	sourcePage := newContentSourcePage(t, 1)
	input := []model.SourcedResource[model.LawContentMatch]{item}
	page, err := lawcontentsearch.NewPage(lawcontentsearch.PageValues{
		Items: input,
		Page:  sourcePage,
	})
	if err != nil {
		t.Fatalf("SOT-IF-015/023: NewPage() のエラー = %v", err)
	}

	input[0] = model.SourcedResource[model.LawContentMatch]{}
	got := page.Items()
	if len(got) != 1 || got[0].Data().Location() != "main:article=1;paragraph=1" {
		t.Fatalf("SOT-IF-023: Items() = %#v", got)
	}
	got[0] = model.SourcedResource[model.LawContentMatch]{}
	if page.Items()[0].Data().Location() != "main:article=1;paragraph=1" {
		t.Fatal("SOT-IF-015/023: items が外部から変更された")
	}
	if page.Page() != sourcePage {
		t.Fatalf("SOT-IF-023: Page() = %#v", page.Page())
	}
	if err := page.Validate(); err != nil {
		t.Fatalf("SOT-IF-023: Validate() のエラー = %v", err)
	}

	request, requestErr := lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
		AllTerms: []string{"行政"},
	})
	if requestErr != nil {
		t.Fatalf("試験用 Request を作成できない: %v", requestErr)
	}
	portResult, portErr := (contentTestPort{page: page}).Search(context.Background(), request)
	if portErr != nil || len(portResult.Items()) != 1 {
		t.Fatalf("SOT-ARCH-009: Port.Search() = %#v, %v", portResult, portErr)
	}
}

func TestPageSupportsEmptyResultAndSameLawAtDifferentLocations(t *testing.T) {
	t.Parallel()

	empty, err := lawcontentsearch.NewPage(lawcontentsearch.PageValues{
		Page: newContentSourcePage(t, 0),
	})
	if err != nil {
		t.Fatalf("SOT-IF-023: 空結果を拒否した: %v", err)
	}
	if items := empty.Items(); items == nil || len(items) != 0 {
		t.Fatalf("SOT-IF-023: 空の Items() = %#v", items)
	}
	encoded, err := json.Marshal(empty)
	if err != nil {
		t.Fatalf("SOT-IF-023: 空結果の json.Marshal() エラー = %v", err)
	}
	if string(encoded) != `{"items":[],"page":{"returnedCount":0}}` {
		t.Fatalf("SOT-IF-023: 空結果の JSON = %s", encoded)
	}

	page, err := lawcontentsearch.NewPage(lawcontentsearch.PageValues{
		Items: []model.SourcedResource[model.LawContentMatch]{
			newContentSearchItem(t, contentItemValues{
				location: "main:article=1;paragraph=1",
			}),
			newContentSearchItem(t, contentItemValues{
				location: "main:article=2;paragraph=1",
			}),
		},
		Page: newContentSourcePage(t, 2),
	})
	if err != nil {
		t.Fatalf("SOT-IF-023: 同じ ref の異なる location を拒否した: %v", err)
	}
	if len(page.Items()) != 2 {
		t.Fatalf("SOT-IF-023: 異なる location が失われた: %#v", page.Items())
	}
}

func TestPageRejectsInvalidIdentityAndLocation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		item          model.SourcedResource[model.LawContentMatch]
		returnedCount int
	}{
		"returnedCount 不一致": {
			item:          newContentSearchItem(t, contentItemValues{}),
			returnedCount: 0,
		},
		"resourceType 不一致": {
			item: newContentSearchItem(t, contentItemValues{resourceType: "article"}),
		},
		"resourceId 不一致": {
			item: newContentSearchItem(t, contentItemValues{resourceID: "other-law"}),
		},
		"versionId 欠落": {
			item: newContentSearchItem(t, contentItemValues{omitVersionID: true}),
		},
		"versionId 不一致": {
			item: newContentSearchItem(t, contentItemValues{versionID: "other-revision"}),
		},
		"sourceId 不一致": {
			item: newContentSearchItem(t, contentItemValues{refSourceID: "other-source"}),
		},
		"最後の provenance.location 欠落": {
			item: newContentSearchItem(t, contentItemValues{omitProvenanceLocation: true}),
		},
		"最後の provenance.location 不一致": {
			item: newContentSearchItem(t, contentItemValues{
				provenanceLocation: "main:article=9",
			}),
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			returnedCount := test.returnedCount
			if name != "returnedCount 不一致" {
				returnedCount = 1
			}
			if _, err := lawcontentsearch.NewPage(lawcontentsearch.PageValues{
				Items: []model.SourcedResource[model.LawContentMatch]{test.item},
				Page:  newContentSourcePage(t, returnedCount),
			}); err == nil {
				t.Fatal("SOT-IF-015/023: 不整合な検索結果を受理した")
			}
		})
	}
}

func TestPageRejectsExactDuplicateAndInvalidItem(t *testing.T) {
	t.Parallel()

	item := newContentSearchItem(t, contentItemValues{})
	if _, err := lawcontentsearch.NewPage(lawcontentsearch.PageValues{
		Items: []model.SourcedResource[model.LawContentMatch]{item, item},
		Page:  newContentSourcePage(t, 2),
	}); err == nil {
		t.Fatal("SOT-IF-023: 同じ ref と location の重複を受理した")
	}
	if _, err := lawcontentsearch.NewPage(lawcontentsearch.PageValues{
		Items: []model.SourcedResource[model.LawContentMatch]{{}},
		Page:  newContentSourcePage(t, 1),
	}); err == nil {
		t.Fatal("SOT-IF-015/023: 無効な item を受理した")
	}
}

func TestPageJSONAndDirectDecode(t *testing.T) {
	t.Parallel()

	page, err := lawcontentsearch.NewPage(lawcontentsearch.PageValues{
		Items: []model.SourcedResource[model.LawContentMatch]{
			newContentSearchItem(t, contentItemValues{}),
		},
		Page: newContentSourcePage(t, 1),
	})
	if err != nil {
		t.Fatalf("SOT-IF-023: NewPage() のエラー = %v", err)
	}
	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("SOT-IF-023: json.Marshal() のエラー = %v", err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-IF-023: JSON を再解析できない: %v", err)
	}
	if len(object) != 2 || object["items"] == nil || object["page"] == nil {
		t.Fatalf("SOT-IF-023: JSON の項目 = %#v", object)
	}

	var decoded lawcontentsearch.Page
	if err := json.Unmarshal(encoded, &decoded); err == nil {
		t.Fatal("SOT-IF-023: Page を JSON から直接復元できた")
	}
}

type contentItemValues struct {
	resourceType           string
	resourceID             string
	versionID              string
	omitVersionID          bool
	refSourceID            string
	dataSourceID           string
	location               string
	provenanceLocation     string
	omitProvenanceLocation bool
}

func newContentSearchItem(
	t *testing.T,
	values contentItemValues,
) model.SourcedResource[model.LawContentMatch] {
	t.Helper()

	const (
		defaultSourceID     = "official-law-source"
		defaultResourceType = "law"
		defaultLawID        = "law-001"
		defaultRevisionID   = "revision-001"
		defaultLocation     = "main:article=1;paragraph=1"
	)
	refSourceID := defaultString(values.refSourceID, defaultSourceID)
	dataSourceID := defaultString(values.dataSourceID, defaultSourceID)
	resourceType := defaultString(values.resourceType, defaultResourceType)
	resourceID := defaultString(values.resourceID, defaultLawID)
	versionID := defaultString(values.versionID, defaultRevisionID)
	if values.omitVersionID {
		versionID = ""
	}
	location := defaultString(values.location, defaultLocation)
	provenanceLocation := defaultString(values.provenanceLocation, location)
	if values.omitProvenanceLocation {
		provenanceLocation = ""
	}

	dataInformationSource := newContentInformationSource(t, dataSourceID)
	legalSource, err := model.NewLegalSource(dataInformationSource)
	if err != nil {
		t.Fatalf("LegalSource を作成できない: %v", err)
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      defaultLawID,
		RevisionID: defaultRevisionID,
		Title:      "行政手続法",
		Source:     legalSource,
	})
	if err != nil {
		t.Fatalf("LawSummary を作成できない: %v", err)
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     legalSource,
		LawID:      defaultLawID,
		RevisionID: defaultRevisionID,
		Location:   location,
		URL:        "https://example.go.jp/laws/" + defaultLawID,
	})
	if err != nil {
		t.Fatalf("Citation を作成できない: %v", err)
	}
	match, err := model.NewLawContentMatch(model.LawContentMatchValues{
		Law:      summary,
		Location: location,
		Text:     "行政庁は、申請を審査する。",
		Citation: citation,
	})
	if err != nil {
		t.Fatalf("LawContentMatch を作成できない: %v", err)
	}

	refInformationSource := newContentInformationSource(t, refSourceID)
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
		Location:       provenanceLocation,
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-023",
	})
	if err != nil {
		t.Fatalf("Provenance を作成できない: %v", err)
	}
	item, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.LawContentMatch]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       match,
		},
	)
	if err != nil {
		t.Fatalf("SourcedResource を作成できない: %v", err)
	}
	return item
}

func newContentInformationSource(t *testing.T, sourceID string) model.InformationSource {
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

func newContentSourcePage(t *testing.T, returnedCount int) model.SourcePage {
	t.Helper()

	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: returnedCount,
	})
	if err != nil {
		t.Fatalf("SourcePage を作成できない: %v", err)
	}
	return page
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
