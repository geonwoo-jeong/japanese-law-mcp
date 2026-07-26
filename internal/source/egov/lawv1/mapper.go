package lawv1

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	updateListResourceType  = "law-update-list"
	updateListMediaType     = "text/xml"
	updateListMappingMethod = "SOT-IF-036"
)

func mapPage(
	response updateListResponse,
	statusCode int,
	requestDate model.Date,
	retrievedAt time.Time,
) (lawupdatelist.Page, error) {
	responseDate, err := parseUpstreamDate(response.date)
	if err != nil || responseDate == nil || *responseDate != requestDate {
		return lawupdatelist.Page{}, invalidResponseError()
	}
	switch {
	case statusCode == http.StatusOK && response.code == "0":
	case statusCode == http.StatusNotFound &&
		response.code == "1" &&
		len(response.items) == 0:
	default:
		return lawupdatelist.Page{}, invalidResponseError()
	}

	source := informationSource()
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		return lawupdatelist.Page{}, invalidResponseError()
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     providerID,
		ResourceType: updateListResourceType,
		ResourceID:   requestDate.String(),
	})
	if err != nil {
		return lawupdatelist.Page{}, invalidResponseError()
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return lawupdatelist.Page{}, invalidResponseError()
	}
	endpointURL := updateListURL(requestDate)

	items := make(
		[]model.SourcedResource[model.LawUpdate],
		len(response.items),
	)
	for index, item := range response.items {
		resource, mapErr := mapItem(
			item,
			requestDate,
			source,
			legalSource,
			ref,
			key,
			endpointURL,
			retrievedAt,
			index,
		)
		if mapErr != nil {
			return lawupdatelist.Page{}, mapErr
		}
		items[index] = resource
	}
	total := len(items)
	sourcePage, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: total,
		TotalCount:    &total,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		return lawupdatelist.Page{}, invalidResponseError()
	}
	page, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
		Items: items,
		Page:  sourcePage,
		Date:  requestDate,
	})
	if err != nil {
		return lawupdatelist.Page{}, invalidResponseError()
	}
	return page, nil
}

func mapItem(
	item updateListItem,
	requestDate model.Date,
	source model.InformationSource,
	legalSource model.LegalSource,
	ref model.SourceResourceRef,
	key model.SourceResourceKey,
	endpointURL string,
	retrievedAt time.Time,
	index int,
) (model.SourcedResource[model.LawUpdate], error) {
	promulgationDate, err := parseUpstreamDate(item.promulgationDate)
	if err != nil {
		return model.SourcedResource[model.LawUpdate]{}, invalidResponseError()
	}
	amendmentPromulgationDate, err := parseUpstreamDate(
		item.amendmentPromulgation,
	)
	if err != nil {
		return model.SourcedResource[model.LawUpdate]{}, invalidResponseError()
	}
	effectiveDate, err := parseUpstreamDate(item.enforcementDate)
	if err != nil {
		return model.SourcedResource[model.LawUpdate]{}, invalidResponseError()
	}
	enforcementPending, err := parseOptionalFlag(item.enforcementFlag)
	if err != nil {
		return model.SourcedResource[model.LawUpdate]{}, invalidResponseError()
	}
	authorityReviewPending, err := parseOptionalFlag(
		item.authorityReviewFlag,
	)
	if err != nil {
		return model.SourcedResource[model.LawUpdate]{}, invalidResponseError()
	}
	update, err := model.NewLawUpdate(model.LawUpdateValues{
		UpdatedOn:                 requestDate,
		LawID:                     item.lawID,
		Title:                     item.lawName,
		LawType:                   item.lawTypeName,
		LawNumber:                 item.lawNumber,
		TitleKana:                 item.lawNameKana,
		PreviousTitle:             item.oldLawName,
		PromulgationDate:          promulgationDate,
		AmendmentTitle:            item.amendmentName,
		AmendmentLawNumber:        item.amendmentNumber,
		AmendmentPromulgationDate: amendmentPromulgationDate,
		EffectiveDate:             effectiveDate,
		EffectiveDateNote:         item.enforcementComment,
		DocumentURL:               item.lawURL,
		EnforcementPending:        enforcementPending,
		AuthorityReviewPending:    authorityReviewPending,
		Source:                    legalSource,
	})
	if err != nil {
		return model.SourcedResource[model.LawUpdate]{}, invalidResponseError()
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            endpointURL,
		RetrievedAt:    retrievedAt,
		MediaType:      updateListMediaType,
		Location:       fmt.Sprintf("DataRoot/ApplData/LawNameListInfo[%d]", index+1),
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       updateListMappingMethod,
	})
	if err != nil {
		return model.SourcedResource[model.LawUpdate]{}, invalidResponseError()
	}
	resource, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.LawUpdate]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       update,
		},
	)
	if err != nil {
		return model.SourcedResource[model.LawUpdate]{}, invalidResponseError()
	}
	return resource, nil
}

func parseUpstreamDate(value string) (*model.Date, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("20060102", value)
	if err != nil || parsed.Format("20060102") != value {
		return nil, fmt.Errorf("e-Gov 日付が有効ではありません")
	}
	date, err := model.NewDate(parsed.Format(time.DateOnly))
	if err != nil {
		return nil, fmt.Errorf("e-Gov 日付が有効ではありません")
	}
	return &date, nil
}

func parseOptionalFlag(value string) (*bool, error) {
	switch value {
	case "":
		return nil, nil
	case "0":
		result := false
		return &result, nil
	case "1":
		result := true
		return &result, nil
	default:
		return nil, fmt.Errorf("e-Gov flag が有効ではありません")
	}
}

func updateListURL(date model.Date) string {
	endpoint := url.URL{
		Scheme: "https",
		Host:   "laws.e-gov.go.jp",
		Path:   updateListEndpointPath + updateListRequest{date: date}.upstreamDate(),
	}
	return endpoint.String()
}

func invalidResponseError() error {
	return newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}
