package lawupdatelist_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type testPort struct {
	page lawupdatelist.Page
}

func (p testPort) List(
	_ context.Context,
	_ lawupdatelist.Request,
) (lawupdatelist.Page, error) {
	return p.page, nil
}

var _ lawupdatelist.Port = testPort{}

func TestPageCopiesItemsAndExposesTypedPort(t *testing.T) {
	t.Parallel()

	item := newLawUpdateItem(t, updateItemValues{})
	sourcePage := newExactSourcePage(t, 1)
	input := []model.SourcedResource[model.LawUpdate]{item}
	targetDate := newUpdateDate(t, "2026-07-26")
	page, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
		Items: input,
		Page:  sourcePage,
		Date:  targetDate,
	})
	if err != nil {
		t.Fatalf("SOT-IF-015/034: NewPage() のエラー = %v", err)
	}

	input[0] = model.SourcedResource[model.LawUpdate]{}
	got := page.Items()
	if len(got) != 1 || got[0].Data().LawID() != "law-001" {
		t.Fatalf("SOT-IF-034: Items() = %#v", got)
	}
	got[0] = model.SourcedResource[model.LawUpdate]{}
	if preserved := page.Items(); preserved[0].Data().LawID() != "law-001" {
		t.Fatal("SOT-IF-015/034: items が外部から変更された")
	}
	if page.Page() != sourcePage {
		t.Fatalf("SOT-IF-034: Page() = %#v", page.Page())
	}
	if page.Date() != targetDate {
		t.Fatalf("SOT-IF-034: Date() = %q", page.Date().String())
	}
	if err := page.Validate(); err != nil {
		t.Fatalf("SOT-IF-034: Validate() のエラー = %v", err)
	}

	requestDate, dateErr := model.NewDate("2026-07-26")
	if dateErr != nil {
		t.Fatalf("試験用 Date を作成できない: %v", dateErr)
	}
	request, requestErr := lawupdatelist.NewRequest(lawupdatelist.RequestValues{
		Date: requestDate,
	})
	if requestErr != nil {
		t.Fatalf("試験用 Request を作成できない: %v", requestErr)
	}
	portResult, portErr := (testPort{page: page}).List(context.Background(), request)
	if portErr != nil || len(portResult.Items()) != 1 {
		t.Fatalf("SOT-ARCH-009: Port.List() = %#v, %v", portResult, portErr)
	}
}

func TestPageSupportsSuccessfulEmptyResult(t *testing.T) {
	t.Parallel()

	page, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
		Page: newExactSourcePage(t, 0),
		Date: newUpdateDate(t, "2026-07-26"),
	})
	if err != nil {
		t.Fatalf("SOT-IF-034: 空結果の NewPage() エラー = %v", err)
	}
	if items := page.Items(); items == nil || len(items) != 0 {
		t.Fatalf("SOT-IF-034: 空の Items() = %#v", items)
	}

	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("SOT-IF-034: 空結果の json.Marshal() エラー = %v", err)
	}
	if string(encoded) != `{"items":[],"page":{"returnedCount":0,"totalCount":0,"totalRelation":"exact"}}` {
		t.Fatalf("SOT-IF-034: 空結果の JSON = %s", encoded)
	}
}

func TestPageRejectsInvalidPageMetadata(t *testing.T) {
	t.Parallel()

	totalOne := 1
	tests := map[string]model.SourcePageValues{
		"returnedCount 不一致": {
			ReturnedCount: 0,
			TotalCount:    &totalOne,
			TotalRelation: model.TotalRelationExact,
		},
		"totalCount の欠落": {
			ReturnedCount: 1,
		},
		"totalRelation が exact ではない": {
			ReturnedCount: 1,
			TotalCount:    &totalOne,
			TotalRelation: model.TotalRelationLowerBound,
		},
		"totalCount と returnedCount の不一致": {
			ReturnedCount: 1,
			TotalCount:    intPointer(2),
			TotalRelation: model.TotalRelationExact,
		},
		"nextToken の存在": {
			ReturnedCount: 1,
			NextToken:     "opaque",
			TotalCount:    &totalOne,
			TotalRelation: model.TotalRelationExact,
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sourcePage, err := model.NewSourcePage(values)
			if err != nil {
				t.Fatalf("試験用 SourcePage を作成できない: %v", err)
			}
			if _, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
				Items: []model.SourcedResource[model.LawUpdate]{
					newLawUpdateItem(t, updateItemValues{}),
				},
				Page: sourcePage,
				Date: newUpdateDate(t, "2026-07-26"),
			}); err == nil {
				t.Fatal("SOT-IF-034: 不整合な page を受理した")
			}
		})
	}
}

func TestPageRejectsInvalidItemIdentity(t *testing.T) {
	t.Parallel()

	tests := map[string]updateItemValues{
		"resourceType 不一致": {
			resourceType: "law",
		},
		"resourceId 不一致": {
			resourceID: "2026-07-25",
		},
		"versionId の存在": {
			versionID: "revision-001",
		},
		"sourceId 不一致": {
			refSourceID: "other-source",
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
				Items: []model.SourcedResource[model.LawUpdate]{
					newLawUpdateItem(t, values),
				},
				Page: newExactSourcePage(t, 1),
				Date: newUpdateDate(t, "2026-07-26"),
			}); err == nil {
				t.Fatal("SOT-IF-015/034: 不整合な item を受理した")
			}
		})
	}
}

func TestPageRejectsInvalidItemAndDirectJSONDecode(t *testing.T) {
	t.Parallel()

	if _, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
		Items: []model.SourcedResource[model.LawUpdate]{{}},
		Page:  newExactSourcePage(t, 1),
		Date:  newUpdateDate(t, "2026-07-26"),
	}); err == nil {
		t.Fatal("SOT-IF-015/034: 無効な item を受理した")
	}

	var page lawupdatelist.Page
	if err := json.Unmarshal(
		[]byte(`{"items":[],"page":{"returnedCount":0,"totalCount":0,"totalRelation":"exact"}}`),
		&page,
	); err == nil {
		t.Fatal("SOT-ENG-002/SOT-IF-034: Page を JSON から直接復元できた")
	}
}

func TestPageRequiresDateAndRejectsAnotherTargetDate(t *testing.T) {
	t.Parallel()

	if _, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
		Page: newExactSourcePage(t, 0),
	}); err == nil {
		t.Fatal("SOT-IF-034: 空結果で Date のゼロ値を受理した")
	}

	if _, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
		Items: []model.SourcedResource[model.LawUpdate]{
			newLawUpdateItem(t, updateItemValues{updatedOn: "2026-07-25"}),
		},
		Page: newExactSourcePage(t, 1),
		Date: newUpdateDate(t, "2026-07-26"),
	}); err == nil {
		t.Fatal("SOT-IF-034: 要求日と異なる日の item を受理した")
	}
}

type updateItemValues struct {
	resourceType string
	resourceID   string
	versionID    string
	refSourceID  string
	updatedOn    string
}

func newLawUpdateItem(
	t *testing.T,
	values updateItemValues,
) model.SourcedResource[model.LawUpdate] {
	t.Helper()

	const (
		defaultSourceID     = "official-law-source"
		defaultResourceType = "law-update-list"
		defaultUpdatedOn    = "2026-07-26"
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
	updatedOnValue := values.updatedOn
	if updatedOnValue == "" {
		updatedOnValue = defaultUpdatedOn
	}
	if resourceID == "" {
		resourceID = updatedOnValue
	}

	dataInformationSource := newUpdateInformationSource(t, defaultSourceID)
	legalSource, err := model.NewLegalSource(dataInformationSource)
	if err != nil {
		t.Fatalf("LegalSource を作成できない: %v", err)
	}
	updatedOn, err := model.NewDate(updatedOnValue)
	if err != nil {
		t.Fatalf("Date を作成できない: %v", err)
	}
	update, err := model.NewLawUpdate(model.LawUpdateValues{
		UpdatedOn: updatedOn,
		LawID:     "law-001",
		Title:     "行政手続法",
		Source:    legalSource,
	})
	if err != nil {
		t.Fatalf("LawUpdate を作成できない: %v", err)
	}

	refInformationSource := newUpdateInformationSource(t, refSourceID)
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
		ProviderID: "test-law-provider",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         refInformationSource,
		ResourceKey:    key,
		URL:            "https://example.go.jp/lawlists/" + resourceID,
		RetrievedAt:    time.Date(2026, time.July, 26, 1, 2, 3, 0, time.UTC),
		MediaType:      "application/xml",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-034",
	})
	if err != nil {
		t.Fatalf("Provenance を作成できない: %v", err)
	}
	item, err := model.NewSourcedResource(model.SourcedResourceValues[model.LawUpdate]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       update,
	})
	if err != nil {
		t.Fatalf("SourcedResource を作成できない: %v", err)
	}
	return item
}

func newExactSourcePage(t *testing.T, count int) model.SourcePage {
	t.Helper()

	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: count,
		TotalCount:    &count,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("SourcePage を作成できない: %v", err)
	}
	return page
}

func newUpdateInformationSource(
	t *testing.T,
	sourceID string,
) model.InformationSource {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "法令更新情報",
		Publisher:  "試験公開者",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://example.go.jp/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できない: %v", err)
	}
	return source
}

func intPointer(value int) *int {
	return &value
}

func newUpdateDate(t *testing.T, value string) model.Date {
	t.Helper()

	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("Date を作成できない: %v", err)
	}
	return date
}
