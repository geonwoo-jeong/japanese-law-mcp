package kokkai

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	speechResourceType     = "parliament-speech"
	speechSearchMediaType  = "application/json"
	speechSearchMappingSOT = "SOT-IF-064"
)

func mapSpeechSearchPage(
	ctx context.Context,
	response speechSearchResponse,
	requestURL string,
	retrievedAt time.Time,
) (parliamentspeechsearch.Page, error) {
	if err := ctx.Err(); err != nil {
		return parliamentspeechsearch.Page{}, err
	}
	if !isSpeechSearchRequestURL(requestURL) {
		return parliamentspeechsearch.Page{}, invalidSpeechSearchResponse()
	}
	items := make([]model.SourcedResource[model.ParliamentSpeech], len(response.records))
	for index, record := range response.records {
		if err := ctx.Err(); err != nil {
			return parliamentspeechsearch.Page{}, err
		}
		item, err := mapSpeechSearchItem(record, requestURL, retrievedAt, index)
		if err != nil {
			return parliamentspeechsearch.Page{}, invalidSpeechSearchResponse()
		}
		items[index] = item
	}
	totalCount := response.totalCount
	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: len(items),
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		return parliamentspeechsearch.Page{}, invalidSpeechSearchResponse()
	}
	result, err := parliamentspeechsearch.NewPage(parliamentspeechsearch.PageValues{
		Items: items,
		Page:  page,
	})
	if err != nil {
		return parliamentspeechsearch.Page{}, invalidSpeechSearchResponse()
	}
	return result, nil
}

func mapSpeechSearchItem(
	record speechRecord,
	requestURL string,
	retrievedAt time.Time,
	index int,
) (model.SourcedResource[model.ParliamentSpeech], error) {
	source := informationSource()
	speaker, err := model.NewParliamentSpeaker(model.ParliamentSpeakerValues{
		Name:     record.speakerName,
		Reading:  record.speakerReading,
		Group:    record.speakerGroup,
		Position: record.speakerPosition,
		Role:     record.speakerRole,
	})
	if err != nil {
		return model.SourcedResource[model.ParliamentSpeech]{}, err
	}
	meetingDate, err := model.NewDate(record.meetingDate)
	if err != nil {
		return model.SourcedResource[model.ParliamentSpeech]{}, err
	}
	meeting, err := model.NewParliamentMeetingReference(model.ParliamentMeetingReferenceValues{
		MeetingRecordID: record.meetingRecordID,
		ImageKind:       record.imageKind,
		Session:         record.session,
		HouseName:       record.houseName,
		MeetingName:     record.meetingName,
		Issue:           record.issue,
		MeetingDate:     meetingDate,
		Closing:         record.closing,
		MeetingURL:      record.meetingURL,
		PDFURL:          record.pdfURL,
	})
	if err != nil {
		return model.SourcedResource[model.ParliamentSpeech]{}, err
	}
	data, err := model.NewParliamentSpeech(model.ParliamentSpeechValues{
		SpeechID:    record.speechID,
		SpeechOrder: record.speechOrder,
		Speaker:     speaker,
		SpeechText:  record.speechText,
		StartPage:   record.startPage,
		SpeechURL:   record.speechURL,
		Meeting:     meeting,
		Source:      source,
	})
	if err != nil {
		return model.SourcedResource[model.ParliamentSpeech]{}, err
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: speechResourceType,
		ResourceID:   record.speechID,
	})
	if err != nil {
		return model.SourcedResource[model.ParliamentSpeech]{}, err
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return model.SourcedResource[model.ParliamentSpeech]{}, err
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            requestURL,
		RetrievedAt:    retrievedAt,
		MediaType:      speechSearchMediaType,
		Location:       fmt.Sprintf("speechRecord[%d]", index),
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       speechSearchMappingSOT,
	})
	if err != nil {
		return model.SourcedResource[model.ParliamentSpeech]{}, err
	}
	return model.NewSourcedResource(model.SourcedResourceValues[model.ParliamentSpeech]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       data,
	})
}

func isSpeechSearchRequestURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	return parsed != nil &&
		strings.EqualFold(parsed.Scheme, "https") &&
		strings.EqualFold(parsed.Hostname(), "kokkai.ndl.go.jp") &&
		parsed.Path == "/api/speech" &&
		parsed.EscapedPath() == "/api/speech" &&
		parsed.Port() == "" &&
		parsed.User == nil &&
		parsed.Fragment == "" &&
		parsed.Opaque == ""
}
