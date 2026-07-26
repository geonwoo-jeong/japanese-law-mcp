package lawv2

import (
	"strings"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	lawContentResourceType  = "law"
	lawContentMediaType     = "application/json"
	lawContentMappingMethod = "SOT-IF-028"
)

var lawContentHighlightTagStripper = strings.NewReplacer("<mark>", "", "</mark>", "")

func mapLawContentItems(
	response lawContentSearchResponse,
	retrievedAt time.Time,
) ([]model.SourcedResource[model.LawContentMatch], error) {
	source := informationSource()
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		return invalidLawContentResult()
	}

	items := make([]model.SourcedResource[model.LawContentMatch], 0, response.sentenceCount)
	for _, law := range response.items {
		summary, ref, officialURL, err := lawContentSummaryAndRef(
			law.law,
			source,
			legalSource,
		)
		if err != nil {
			return invalidLawContentResult()
		}
		for _, sentence := range law.sentences {
			item, err := mapLawContentSentence(
				summary,
				ref,
				officialURL,
				sentence,
				source,
				retrievedAt,
			)
			if err != nil {
				return invalidLawContentResult()
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func lawContentSummaryAndRef(
	law lawSearchLaw,
	source model.InformationSource,
	legalSource model.LegalSource,
) (
	model.LawSummary,
	model.SourceResourceRef,
	string,
	error,
) {
	promulgationDate, err := optionalLawSearchDate(law.promulgationDate)
	if err != nil {
		return model.LawSummary{}, model.SourceResourceRef{}, "", err
	}
	revisionEffectiveDate, err := optionalLawSearchDate(law.revisionEffectiveDate)
	if err != nil {
		return model.LawSummary{}, model.SourceResourceRef{}, "", err
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
		return model.LawSummary{}, model.SourceResourceRef{}, "", err
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     providerID,
		ResourceType: lawContentResourceType,
		ResourceID:   law.lawID,
		VersionID:    law.revisionID,
	})
	if err != nil {
		return model.LawSummary{}, model.SourceResourceRef{}, "", err
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return model.LawSummary{}, model.SourceResourceRef{}, "", err
	}
	officialURL, err := officialLawURLWithError(
		law.lawID,
		law.revisionID,
		newLawContentSourceError,
	)
	if err != nil {
		return model.LawSummary{}, model.SourceResourceRef{}, "", err
	}
	_ = source
	return summary, ref, officialURL, nil
}

func mapLawContentSentence(
	summary model.LawSummary,
	ref model.SourceResourceRef,
	officialURL string,
	sentence lawContentSentence,
	source model.InformationSource,
	retrievedAt time.Time,
) (model.SourcedResource[model.LawContentMatch], error) {
	citation, err := model.NewCitation(model.CitationValues{
		Source:     summary.Source(),
		LawID:      summary.LawID(),
		RevisionID: summary.RevisionID(),
		Location:   sentence.position,
		URL:        officialURL,
	})
	if err != nil {
		return model.SourcedResource[model.LawContentMatch]{}, err
	}
	match, err := model.NewLawContentMatch(model.LawContentMatchValues{
		Law:      summary,
		Location: sentence.position,
		Text:     lawContentHighlightTagStripper.Replace(sentence.text),
		Citation: citation,
	})
	if err != nil {
		return model.SourcedResource[model.LawContentMatch]{}, err
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    ref.Key(),
		URL:            officialURL,
		RetrievedAt:    retrievedAt,
		MediaType:      lawContentMediaType,
		Location:       sentence.position,
		Transformation: model.ProvenanceTransformationExtracted,
		MethodID:       lawContentMappingMethod,
	})
	if err != nil {
		return model.SourcedResource[model.LawContentMatch]{}, err
	}
	return model.NewSourcedResource(
		model.SourcedResourceValues[model.LawContentMatch]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       match,
		},
	)
}

func invalidLawContentResult() ([]model.SourcedResource[model.LawContentMatch], error) {
	return nil, newLawContentSourceError(
		model.SourceErrorCodeInvalidSourceResponse,
		"",
	)
}
