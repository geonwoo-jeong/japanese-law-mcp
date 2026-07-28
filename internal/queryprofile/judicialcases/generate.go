package judicialcases

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const maximumGeneratedCandidates = 16

type candidateDraft struct {
	evidence []legalquery.EvidenceCode
	concepts []legalquery.LegalConceptSource
	steps    []stepDraft
}

type stepDraft struct {
	startByte int
	input     legalquery.LogicalInput
}

// Generate は、位置付き前処理事実だけから裁判例候補を生成する。
func (p *Profile) Generate(
	input legalquery.CandidateGenerationInput,
	scope legalquery.CandidateIDScope,
) (legalquery.CandidateGeneration, error) {
	if p == nil {
		return legalquery.CandidateGeneration{}, fmt.Errorf(
			"judicial-cases profile が初期化されていません",
		)
	}
	if err := p.metadata.Validate(); err != nil {
		return legalquery.CandidateGeneration{}, fmt.Errorf(
			"profile metadata が有効ではありません: %w",
			err,
		)
	}
	if err := input.Validate(); err != nil {
		return legalquery.CandidateGeneration{}, fmt.Errorf(
			"candidate generation input が有効ではありません: %w",
			err,
		)
	}
	if input.StandaloneStructuredQuery() {
		return p.newGeneration(
			nil,
			[]legalquery.CandidateGenerationSignal{
				legalquery.CandidateSignalStandaloneStructuredQuery,
			},
			legalquery.QuerySelectionModeAutomatic,
		)
	}
	cues, err := p.resolveCues(input.CueMentions())
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	if input.Language() == legalquery.QueryLanguageNonJapanese {
		return p.newGeneration(
			nil,
			[]legalquery.CandidateGenerationSignal{
				legalquery.CandidateSignalNonJapaneseQuery,
			},
			legalquery.QuerySelectionModeAutomatic,
		)
	}

	read, err := buildReadDraft(input, cues)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	if read != nil {
		candidates, materializeErr := p.materializeCandidates(
			[]candidateDraft{*read},
			scope,
		)
		if materializeErr != nil {
			return legalquery.CandidateGeneration{}, materializeErr
		}
		return p.newGeneration(
			candidates,
			nil,
			legalquery.QuerySelectionModeAutomatic,
		)
	}

	drafts, tooMany, ambiguous, err := p.buildSearchDrafts(input, cues)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	mode := legalquery.QuerySelectionModeAutomatic
	if tooMany || ambiguous {
		mode = legalquery.QuerySelectionModeClarificationRequired
	}
	candidates, err := p.materializeCandidates(drafts, scope)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	return p.newGeneration(candidates, nil, mode)
}

func buildReadDraft(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) (*candidateDraft, error) {
	if !cues.has("task", "read") {
		return nil, nil
	}
	ref, exists := input.Ref()
	if !exists || ref.Key().ResourceType() != "judicial-decision" {
		return nil, nil
	}
	readInput, err := legalquery.NewJudicialDecisionReadIntentV1(
		legalquery.JudicialDecisionReadIntentV1Values{Ref: ref},
	)
	if err != nil {
		return nil, err
	}
	return &candidateDraft{
		evidence: []legalquery.EvidenceCode{
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
		},
		steps: []stepDraft{{input: readInput}},
	}, nil
}

func (p *Profile) newGeneration(
	candidates []legalquery.LegalQueryCandidate,
	signals []legalquery.CandidateGenerationSignal,
	mode legalquery.QuerySelectionMode,
) (legalquery.CandidateGeneration, error) {
	return legalquery.NewCandidateGeneration(
		legalquery.CandidateGenerationValues{
			ProfileID:      p.metadata.ProfileID(),
			ProfileVersion: p.metadata.ProfileVersion(),
			RankingVersion: p.metadata.RankingVersion(),
			Candidates:     candidates,
			Signals:        signals,
			SelectionMode:  mode,
		},
	)
}
