package parliamentspeechsearch_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type testSpeechSearchPort struct {
	page parliamentspeechsearch.Page
}

func (p testSpeechSearchPort) Search(
	_ context.Context,
	_ parliamentspeechsearch.Request,
) (parliamentspeechsearch.Page, error) {
	return p.page, nil
}

var _ parliamentspeechsearch.Port = testSpeechSearchPort{}

func TestPageCopiesItemsAndPreservesOrderAndDuplicates(t *testing.T) {
	t.Parallel()

	first := newSpeechItem(t, speechItemValues{speechID: "100"})
	second := newSpeechItem(t, speechItemValues{speechID: "200"})
	input := []model.SourcedResource[model.ParliamentSpeech]{second, first, first}
	page, err := parliamentspeechsearch.NewPage(
		parliamentspeechsearch.PageValues{
			Items: input,
			Page:  newSourcePage(t, 3),
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-015/SOT-IF-062: NewPage() のエラー = %v", err)
	}

	input[0] = model.SourcedResource[model.ParliamentSpeech]{}
	got := page.Items()
	if len(got) != 3 ||
		got[0].Data().SpeechID() != "200" ||
		got[1].Data().SpeechID() != "100" ||
		got[2].Data().SpeechID() != "100" {
		t.Fatalf("SOT-IF-062: Items() = %#v", got)
	}
	got[0] = model.SourcedResource[model.ParliamentSpeech]{}
	if page.Items()[0].Data().SpeechID() != "200" {
		t.Fatal("SOT-IF-015/SOT-IF-062: items が外部から変更された")
	}
	if err := page.Validate(); err != nil {
		t.Fatalf("SOT-IF-062: Validate() のエラー = %v", err)
	}

	request, err := parliamentspeechsearch.NewRequest(
		parliamentspeechsearch.RequestValues{Query: "永住許可"},
	)
	if err != nil {
		t.Fatalf("Request を作成できない: %v", err)
	}
	portPage, portErr := (testSpeechSearchPort{page: page}).Search(
		context.Background(),
		request,
	)
	if portErr != nil || len(portPage.Items()) != 3 {
		t.Fatalf("SOT-IF-062: Port.Search() = %#v, %v", portPage, portErr)
	}
}

func TestPageSupportsSuccessfulEmptyResult(t *testing.T) {
	t.Parallel()

	page, err := parliamentspeechsearch.NewPage(
		parliamentspeechsearch.PageValues{Page: newSourcePage(t, 0)},
	)
	if err != nil {
		t.Fatalf("SOT-IF-062: 空結果の NewPage() エラー = %v", err)
	}
	if items := page.Items(); items == nil || len(items) != 0 {
		t.Fatalf("SOT-IF-062: 空の Items() = %#v", items)
	}
	encoded, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("SOT-IF-062: 空結果の json.Marshal() エラー = %v", err)
	}
	if string(encoded) != `{"items":[],"page":{"returnedCount":0,"totalCount":0,"totalRelation":"exact"}}` {
		t.Fatalf("SOT-IF-062: 空結果の JSON = %s", encoded)
	}
}

func TestPageRejectsReturnedCountMismatchAndInvalidItem(t *testing.T) {
	t.Parallel()

	if _, err := parliamentspeechsearch.NewPage(
		parliamentspeechsearch.PageValues{Page: newSourcePage(t, 1)},
	); err == nil {
		t.Fatal("SOT-IF-062: returnedCount と items の不一致を受理した")
	}

	tests := map[string]speechItemValues{
		"resourceType 不一致": {resourceType: "law"},
		"resourceId 不一致":   {resourceID: "other"},
		"versionId が存在":    {versionID: "v1"},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			item := newSpeechItem(t, values)
			if _, err := parliamentspeechsearch.NewPage(
				parliamentspeechsearch.PageValues{
					Items: []model.SourcedResource[model.ParliamentSpeech]{item},
					Page:  newSourcePage(t, 1),
				},
			); err == nil {
				t.Fatal("SOT-IF-015/SOT-IF-062: 不整合な item を受理した")
			}
		})
	}
}

func TestPageRejectsDirectJSONDecodeAndInvalidZeroValue(t *testing.T) {
	t.Parallel()

	var page parliamentspeechsearch.Page
	if err := json.Unmarshal(
		[]byte(`{"items":[],"page":{"returnedCount":0}}`),
		&page,
	); err == nil {
		t.Fatal("SOT-IF-062: Page を JSON から直接復元できた")
	}
	if err := page.Validate(); err == nil {
		t.Fatal("SOT-IF-062: Page のゼロ値を受理した")
	}
}

func newSourcePage(t *testing.T, returnedCount int) model.SourcePage {
	t.Helper()

	total := returnedCount
	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: returnedCount,
		TotalCount:    &total,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("SourcePage を作成できない: %v", err)
	}
	return page
}

type speechItemValues struct {
	speechID     string
	resourceType string
	resourceID   string
	versionID    string
	refSourceID  string
}

func newSpeechItem(
	t *testing.T,
	values speechItemValues,
) model.SourcedResource[model.ParliamentSpeech] {
	t.Helper()

	speechID := values.speechID
	if speechID == "" {
		speechID = "123456"
	}
	resourceType := values.resourceType
	if resourceType == "" {
		resourceType = "parliament-speech"
	}
	resourceID := values.resourceID
	if resourceID == "" {
		resourceID = speechID
	}
	refSourceID := values.refSourceID
	if refSourceID == "" {
		refSourceID = "ndl-diet-records"
	}

	resourceKey, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     refSourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		VersionID:    values.versionID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "ndl-diet-speech-api",
		Key:        resourceKey,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         mustInformationSource(t),
		ResourceKey:    resourceKey,
		URL:            "https://kokkai.ndl.go.jp/api/speech?any=%E6%B0%B8%E4%BD%8F",
		RetrievedAt:    time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC),
		MediaType:      "application/json",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-064",
		Location:       "/speechRecord/0",
	})
	if err != nil {
		t.Fatalf("Provenance を作成できない: %v", err)
	}
	data, err := model.NewParliamentSpeech(model.ParliamentSpeechValues{
		SpeechID:    speechID,
		SpeechOrder: 1,
		Speaker:     mustSpeaker(t),
		SpeechText:  "永住許可について質問します。",
		SpeechURL:   "https://kokkai.ndl.go.jp/txt/213214889X00520240315/1",
		Meeting:     mustMeeting(t),
		Source:      mustInformationSource(t),
	})
	if err != nil {
		t.Fatalf("ParliamentSpeech を作成できない: %v", err)
	}
	item, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.ParliamentSpeech]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       data,
		},
	)
	if err != nil {
		t.Fatalf("SourcedResource を作成できない: %v", err)
	}
	return item
}

func mustSpeaker(t *testing.T) model.ParliamentSpeaker {
	t.Helper()
	speaker, err := model.NewParliamentSpeaker(model.ParliamentSpeakerValues{
		Name: "山田太郎",
	})
	if err != nil {
		t.Fatalf("ParliamentSpeaker を作成できない: %v", err)
	}
	return speaker
}

func mustMeeting(t *testing.T) model.ParliamentMeetingReference {
	t.Helper()
	date, err := model.NewDate("2024-03-15")
	if err != nil {
		t.Fatalf("Date を作成できない: %v", err)
	}
	meeting, err := model.NewParliamentMeetingReference(
		model.ParliamentMeetingReferenceValues{
			MeetingRecordID: "213214889X00520240315",
			ImageKind:       "会議録",
			Session:         213,
			HouseName:       "参議院",
			MeetingName:     "法務委員会",
			Issue:           "5号",
			MeetingDate:     date,
			MeetingURL:      "https://kokkai.ndl.go.jp/txt/213214889X00520240315",
		},
	)
	if err != nil {
		t.Fatalf("ParliamentMeetingReference を作成できない: %v", err)
	}
	return meeting
}

func mustInformationSource(t *testing.T) model.InformationSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "ndl-diet-records",
		Name:       "国会会議録検索システム",
		Publisher:  "国立国会図書館",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://kokkai.ndl.go.jp/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できない: %v", err)
	}
	return source
}
