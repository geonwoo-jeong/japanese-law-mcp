package lawv2

import (
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	lawDocumentResourceType  = "law"
	lawDocumentMediaType     = "application/xml"
	lawDocumentMappingMethod = "SOT-IF-011"
)

func mapLawDocument(
	response lawDocumentResponse,
	asOf *model.Date,
	retrievedAt time.Time,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	source := informationSource()
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		return invalidLawDocumentResult()
	}
	promulgationDate, err := optionalLawDocumentDate(
		response.law.promulgationDate,
	)
	if err != nil {
		return invalidLawDocumentResult()
	}
	revisionEffectiveDate, err := optionalLawDocumentDate(
		response.law.revisionEffectiveDate,
	)
	if err != nil {
		return invalidLawDocumentResult()
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:                 response.law.lawID,
		RevisionID:            response.law.revisionID,
		Title:                 response.law.title,
		LawNumber:             response.law.lawNumber,
		PromulgationDate:      promulgationDate,
		RevisionEffectiveDate: revisionEffectiveDate,
		Source:                legalSource,
	})
	if err != nil {
		return invalidLawDocumentResult()
	}
	officialURL, err := officialLawURLWithError(
		response.law.lawID,
		response.law.revisionID,
		newLawDocumentSourceError,
	)
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     legalSource,
		LawID:      response.law.lawID,
		RevisionID: response.law.revisionID,
		URL:        officialURL,
	})
	if err != nil {
		return invalidLawDocumentResult()
	}
	representation, err := model.NewLawDocumentRepresentation(
		model.LawDocumentRepresentationValues{
			Law:      summary,
			AsOf:     cloneLawDocumentDate(asOf),
			Format:   model.LawDocumentFormatXML,
			Content:  response.content,
			Citation: citation,
		},
	)
	if err != nil {
		return invalidLawDocumentResult()
	}

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     providerID,
		ResourceType: lawDocumentResourceType,
		ResourceID:   response.law.lawID,
		VersionID:    response.law.revisionID,
	})
	if err != nil {
		return invalidLawDocumentResult()
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return invalidLawDocumentResult()
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            officialURL,
		RetrievedAt:    retrievedAt,
		MediaType:      lawDocumentMediaType,
		Transformation: model.ProvenanceTransformationExtracted,
		MethodID:       lawDocumentMappingMethod,
	})
	if err != nil {
		return invalidLawDocumentResult()
	}
	result, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.LawDocumentRepresentation]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       representation,
		},
	)
	if err != nil {
		return invalidLawDocumentResult()
	}
	return result, nil
}

func optionalLawDocumentDate(value string) (*model.Date, error) {
	if value == "" {
		return nil, nil
	}
	date, err := model.NewDate(value)
	if err != nil {
		return nil, err
	}
	return &date, nil
}

func cloneLawDocumentDate(value *model.Date) *model.Date {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func invalidLawDocumentResult() (model.SourcedResource[model.LawDocumentRepresentation], error) {
	return model.SourcedResource[model.LawDocumentRepresentation]{},
		newLawDocumentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
}
