package judicialcases

import "github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"

// generateJudicialEvidence は、schema version 2 の意味経路である。
// 現行の組込み schema version 1 からは到達しない。
func (p *Profile) generateJudicialEvidence(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	scope legalquery.CandidateIDScope,
) (legalquery.CandidateGeneration, error) {
	read, err := buildJudicialEvidenceReadDraft(input, cues)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	drafts, tooMany, ambiguous, err := p.buildJudicialEvidenceSearchDrafts(
		input,
		cues,
	)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	mode := legalquery.QuerySelectionModeAutomatic
	if tooMany || ambiguous {
		mode = legalquery.QuerySelectionModeClarificationRequired
	}
	constraint := legalquery.QueryCompositionConstraintNone
	if tooMany {
		constraint = legalquery.QueryCompositionConstraintStepLimitExceeded
		drafts = nil
	} else {
		var overLimit bool
		drafts, overLimit = combineJudicialReadAndSearch(read, drafts)
		if overLimit {
			mode = legalquery.QuerySelectionModeClarificationRequired
			constraint = legalquery.QueryCompositionConstraintStepLimitExceeded
			drafts = nil
		}
	}
	materialized, forcedClarification, err :=
		p.materializeJudicialEvidenceCandidateRecords(
			input,
			cues,
			drafts,
			scope,
		)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	if forcedClarification {
		mode = legalquery.QuerySelectionModeClarificationRequired
	}
	candidates := make([]legalquery.LegalQueryCandidate, 0, len(materialized))
	for _, current := range materialized {
		candidates = append(candidates, current.candidate)
	}
	members, err := judicialCompositionMembers(materialized, mode)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	return p.newGeneration(candidates, nil, mode, members, constraint)
}
