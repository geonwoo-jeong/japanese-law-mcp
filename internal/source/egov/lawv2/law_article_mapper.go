package lawv2

import (
	"strconv"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	lawArticleResourceType  = "law"
	lawArticleMediaType     = "application/xml"
	lawArticleMappingMethod = "SOT-IF-012"
)

func mapLawArticle(
	response lawArticleResponse,
	retrievedAt time.Time,
) (model.SourcedResource[model.LawArticleFragment], error) {
	source := informationSource()
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		return invalidLawArticleResult()
	}
	promulgationDate, err := optionalLawDocumentDate(
		response.law.promulgationDate,
	)
	if err != nil {
		return invalidLawArticleResult()
	}
	revisionEffectiveDate, err := optionalLawDocumentDate(
		response.law.revisionEffectiveDate,
	)
	if err != nil {
		return invalidLawArticleResult()
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
		return invalidLawArticleResult()
	}
	officialURL, err := officialLawURLWithError(
		response.law.lawID,
		response.law.revisionID,
		newLawArticleSourceError,
	)
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	location := lawArticleCitationLocation(response.location)
	citation, err := model.NewCitation(model.CitationValues{
		Source:     legalSource,
		LawID:      response.law.lawID,
		RevisionID: response.law.revisionID,
		Location:   location,
		URL:        officialURL,
	})
	if err != nil {
		return invalidLawArticleResult()
	}
	fragment, err := model.NewLawArticleFragment(model.LawArticleFragmentValues{
		Law:      summary,
		Location: response.location,
		Format:   model.LawArticleFormatXML,
		Content:  response.content,
		Citation: citation,
	})
	if err != nil {
		return invalidLawArticleResult()
	}

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     providerID,
		ResourceType: lawArticleResourceType,
		ResourceID:   response.law.lawID,
		VersionID:    response.law.revisionID,
	})
	if err != nil {
		return invalidLawArticleResult()
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return invalidLawArticleResult()
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            officialURL,
		RetrievedAt:    retrievedAt,
		MediaType:      lawArticleMediaType,
		Location:       location,
		Transformation: model.ProvenanceTransformationExtracted,
		MethodID:       lawArticleMappingMethod,
	})
	if err != nil {
		return invalidLawArticleResult()
	}
	result, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.LawArticleFragment]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       fragment,
		},
	)
	if err != nil {
		return invalidLawArticleResult()
	}
	return result, nil
}

func lawArticleCitationLocation(location model.LawArticleLocation) string {
	value := string(location.Provision()) +
		":article=" +
		location.ArticleNumber()
	if paragraph, exists := location.ParagraphNumber(); exists {
		value += ";paragraph=" + strconv.Itoa(paragraph)
	}
	return value
}

func invalidLawArticleResult() (
	model.SourcedResource[model.LawArticleFragment],
	error,
) {
	return model.SourcedResource[model.LawArticleFragment]{},
		newLawArticleSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
}
