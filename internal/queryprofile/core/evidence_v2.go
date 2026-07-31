package core

import (
	"errors"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

var errCoreEvidenceStepLimitExceeded = errors.New(
	"core evidence の logical step 上限を超えました",
)

// generateCoreEvidence は、次版 core profile の非公開評価経路である。
//
// この経路は request 単位の evidence mapping と cluster 校正を完成させた後にだけ
// active profile set へ採用する。現段階では LoadEmbedded から到達しない。
func (p *Profile) generateCoreEvidence(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	additionalContent []candidateDraft,
	signals []legalquery.CandidateGenerationSignal,
	scope legalquery.CandidateIDScope,
) (legalquery.CandidateGeneration, error) {
	drafts, err := p.generateCoreEvidenceDrafts(
		input,
		cues,
		additionalContent,
	)
	if err != nil {
		if errors.Is(err, errCoreEvidenceStepLimitExceeded) {
			return p.newGeneration(
				nil,
				signals,
				legalquery.QuerySelectionModeClarificationRequired,
				nil,
				nil,
				legalquery.QueryCompositionConstraintStepLimitExceeded,
			)
		}
		return legalquery.CandidateGeneration{}, err
	}
	drafts = withAsOfEvidence(drafts)
	drafts, err = withCoreEvidenceBindings(input, cues, drafts)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	drafts = p.retainRelationV2SupportedDrafts(
		input,
		cues,
		drafts,
		signals,
	)
	candidates, stepStartBytes, forcedClarification, err :=
		p.materializeCoreEvidenceCandidates(
			input,
			cues,
			drafts,
			scope,
		)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	mode := p.selectionMode(input, cues, candidates)
	var pairs []legalquery.CandidateHedgePair
	if forcedClarification {
		mode = legalquery.QuerySelectionModeClarificationRequired
	} else {
		pairs, err = p.hedgePairs(input, cues, candidates, mode)
		if err != nil {
			return legalquery.CandidateGeneration{}, err
		}
	}
	members, err := buildCompositionMembers(
		candidates,
		stepStartBytes,
		mode,
	)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	return p.newGeneration(
		candidates,
		signals,
		mode,
		pairs,
		members,
		legalquery.QueryCompositionConstraintNone,
	)
}

func (p *Profile) generateCoreEvidenceDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	additionalContent []candidateDraft,
) ([]candidateDraft, error) {
	shared, err := p.buildCoreSharedTerminalDrafts(input, cues)
	if err != nil {
		return nil, err
	}
	if shared.handled {
		if shared.stepLimitExceeded {
			return nil, errCoreEvidenceStepLimitExceeded
		}
		result := make([]candidateDraft, 0, len(shared.drafts))
		for _, draft := range shared.drafts {
			result = append(result, cloneDraft(draft.draft))
		}
		return result, nil
	}
	if hasTooManySeparatedSubjects(input, cues) {
		return nil, errCoreEvidenceStepLimitExceeded
	}
	projected, handled, err := p.buildCoreProjectedLawNameDrafts(input, cues)
	if err != nil {
		return nil, err
	}
	if handled {
		return projected, nil
	}
	return p.generateDrafts(input, cues, additionalContent)
}
