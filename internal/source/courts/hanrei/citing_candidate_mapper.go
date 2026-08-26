package hanrei

import (
	"context"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const citingCandidateMappingMethodID = "SOT-IF-073"

func mapCitingCandidateOccurrences(
	ctx context.Context,
	response searchResponse,
	retrievedAt time.Time,
) ([]judicialcitingcandidatesearch.Candidate, bool, error) {
	if err := validateSearchMappingInput(
		response,
		searchEndpoint,
		retrievedAt,
		1,
	); err != nil {
		return nil, false, err
	}
	mapped, err := mapSearchRowsUnlimited(response.rows, searchEndpoint, retrievedAt)
	if err != nil {
		return nil, false, err
	}
	items := make([]judicialcitingcandidatesearch.Candidate, 0, len(mapped))
	for _, resource := range mapped {
		if err := searchHTMLContextError(ctx); err != nil {
			return nil, false, err
		}
		candidate, err := mapCitingCandidateOccurrence(resource, retrievedAt)
		if err != nil {
			return nil, false, err
		}
		items = append(items, candidate)
	}
	return items, response.totalCount > len(response.rows), nil
}

func mapCitingCandidateOccurrence(
	mapped model.SourcedResource[model.JudicialDecisionSummary],
	retrievedAt time.Time,
) (judicialcitingcandidatesearch.Candidate, error) {
	mappedProvenance := mapped.Provenance()
	location, exists := mappedProvenance[len(mappedProvenance)-1].Location()
	if !exists {
		return judicialcitingcandidatesearch.Candidate{}, invalidSearchResponseError()
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         informationSource(),
		ResourceKey:    mapped.Ref().Key(),
		URL:            searchEndpoint,
		RetrievedAt:    retrievedAt,
		MediaType:      searchHTMLMediaType,
		Location:       location,
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       citingCandidateMappingMethodID,
	})
	if err != nil {
		return judicialcitingcandidatesearch.Candidate{}, invalidSearchResponseError()
	}
	decision, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionSummary]{
			Ref:        mapped.Ref(),
			Provenance: []model.Provenance{provenance},
			Data:       mapped.Data(),
		},
	)
	if err != nil {
		return judicialcitingcandidatesearch.Candidate{}, invalidSearchResponseError()
	}
	evidence, err := model.NewJudicialCitationEvidence(
		model.JudicialCitationEvidenceValues{
			EvidenceLevel: model.JudicialCitationEvidenceLevelOfficialSearchCandidate,
			Provenance:    provenance,
		},
	)
	if err != nil {
		return judicialcitingcandidatesearch.Candidate{}, invalidSearchResponseError()
	}
	candidate, err := judicialcitingcandidatesearch.NewCandidate(
		judicialcitingcandidatesearch.CandidateValues{
			Decision: decision,
			Evidence: []model.JudicialCitationEvidence{evidence},
		},
	)
	if err != nil {
		return judicialcitingcandidatesearch.Candidate{}, invalidSearchResponseError()
	}
	return candidate, nil
}
