package lawv2

import (
	"net/url"
	"strings"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	lawSearchResourceType    = "law"
	lawSearchMediaType       = "application/json"
	lawSearchMappingMethod   = "SOT-IF-054"
	eGovOfficialLawURLPrefix = "https://laws.e-gov.go.jp/law/"
)

func mapLawSearchItems(
	response lawSearchResponse,
	retrievedAt time.Time,
) ([]model.SourcedResource[model.LawSummary], error) {
	source := informationSource()
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		return nil, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}

	items := make([]model.SourcedResource[model.LawSummary], len(response.laws))
	for index, law := range response.laws {
		item, mapErr := mapLawSearchItem(law, source, legalSource, retrievedAt)
		if mapErr != nil {
			return nil, mapErr
		}
		items[index] = item
	}
	return items, nil
}

func mapLawSearchItem(
	law lawSearchLaw,
	source model.InformationSource,
	legalSource model.LegalSource,
	retrievedAt time.Time,
) (model.SourcedResource[model.LawSummary], error) {
	promulgationDate, err := optionalLawSearchDate(law.promulgationDate)
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, err
	}
	revisionEffectiveDate, err := optionalLawSearchDate(law.revisionEffectiveDate)
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, err
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:                 law.lawID,
		RevisionID:            law.revisionID,
		Title:                 law.title,
		LawNumber:             law.lawNumber,
		PromulgationDate:      promulgationDate,
		RevisionEffectiveDate: revisionEffectiveDate,
		Source:                legalSource,
	})
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     providerID,
		ResourceType: lawSearchResourceType,
		ResourceID:   law.lawID,
		VersionID:    law.revisionID,
	})
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	officialURL, err := officialLawURL(law.lawID, law.revisionID)
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, err
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            officialURL,
		RetrievedAt:    retrievedAt,
		MediaType:      lawSearchMediaType,
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       lawSearchMappingMethod,
	})
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.LawSummary]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       summary,
	})
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return resource, nil
}

func optionalLawSearchDate(value string) (*model.Date, error) {
	if value == "" {
		return nil, nil
	}
	date, err := model.NewDate(value)
	if err != nil {
		return nil, newSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	return &date, nil
}

func officialLawURL(lawID string, revisionID string) (string, error) {
	return officialLawURLWithError(lawID, revisionID, newSourceError)
}

func officialLawURLWithError(
	lawID string,
	revisionID string,
	sourceError sourceErrorFactory,
) (string, error) {
	revisionPrefix := lawID + "_"
	if lawID == "" ||
		!strings.HasPrefix(revisionID, revisionPrefix) ||
		len(revisionID) == len(revisionPrefix) {
		return "", sourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	revisionPart := strings.TrimPrefix(revisionID, revisionPrefix)
	if url.PathEscape(lawID) != lawID || url.PathEscape(revisionPart) != revisionPart {
		return "", sourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return eGovOfficialLawURLPrefix + lawID + "/" + revisionPart, nil
}
