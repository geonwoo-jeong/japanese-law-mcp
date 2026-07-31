package judicialcases

import (
	"fmt"
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const maximumGeneratedCandidates = 16

type candidateDraft struct {
	evidence                     []legalquery.EvidenceCode
	concepts                     []legalquery.LegalConceptSource
	steps                        []stepDraft
	preserveMorphologicalContext bool
}

type stepDraft struct {
	startByte        int
	topicOrdinal     int
	input            legalquery.LogicalInput
	evidenceCodes    []legalquery.EvidenceCode
	evidenceBindings []profileevidence.EvidenceValues
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
			nil,
			legalquery.QueryCompositionConstraintNone,
		)
	}
	cues, err := p.resolveCues(input.CueMentions())
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	if p.intentEvidenceMode == cueIntentEvidenceRelationV2 ||
		p.intentEvidenceMode == cueIntentEvidenceJudicial {
		cues, err = p.resolveRelationV2Cues(input, cues)
		if err != nil {
			return legalquery.CandidateGeneration{}, err
		}
	}
	if input.Language() == legalquery.QueryLanguageNonJapanese {
		return p.newGeneration(
			nil,
			[]legalquery.CandidateGenerationSignal{
				legalquery.CandidateSignalNonJapaneseQuery,
			},
			legalquery.QuerySelectionModeAutomatic,
			nil,
			legalquery.QueryCompositionConstraintNone,
		)
	}
	if p.intentEvidenceMode == cueIntentEvidenceJudicial {
		return p.generateJudicialEvidence(input, cues, scope)
	}

	read, err := buildReadDraft(input, cues)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	drafts, tooMany, ambiguous, err := p.buildSearchDrafts(input, cues)
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
	}
	if tooMany {
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
	materialized, err := p.materializeCandidateRecords(drafts, scope)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
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
	startByte, exists := readSourceStartByte(input, cues)
	if !exists {
		return nil, nil
	}
	return &candidateDraft{
		evidence: []legalquery.EvidenceCode{
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
		},
		steps: []stepDraft{{
			startByte:    startByte,
			topicOrdinal: 1,
			input:        readInput,
			evidenceCodes: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
		}},
	}, nil
}

func readSourceStartByte(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) (int, bool) {
	start := 0
	found := false
	searchCues := cues.mentions[cueMeaningKey("task", "search")]
	resourceCues := cues.mentions[cueMeaningKey("resource", "judicial_decision")]
	readTargets := make([]legalquery.QueryTermMention, 0)
	for _, term := range input.QueryTermMentions() {
		if queryTermBelongsToReadReference(term, input, cues) {
			readTargets = append(readTargets, term)
		}
	}
	for _, read := range cues.mentions[cueMeaningKey("task", "read")] {
		current := read.Span().StartByte()
		for _, resource := range resourceCues {
			if resource.Span().EndByte() > read.Span().StartByte() ||
				judicialCueStartsBetween(
					searchCues,
					resource.Span().EndByte(),
					read.Span().StartByte(),
				) {
				continue
			}
			if resource.Span().StartByte() < current {
				current = resource.Span().StartByte()
			}
		}
		for _, target := range readTargets {
			if target.Span().EndByte() > read.Span().StartByte() ||
				judicialCueStartsBetween(
					searchCues,
					target.Span().EndByte(),
					read.Span().StartByte(),
				) {
				continue
			}
			if target.Span().StartByte() < current {
				current = target.Span().StartByte()
			}
		}
		if !found || current < start {
			start = current
			found = true
		}
	}
	return start, found
}

func combineJudicialReadAndSearch(
	read *candidateDraft,
	search []candidateDraft,
) ([]candidateDraft, bool) {
	if read == nil {
		return search, false
	}
	if len(search) == 0 {
		return []candidateDraft{*read}, false
	}
	result := make([]candidateDraft, 0, len(search))
	for _, current := range search {
		combined := candidateDraft{
			evidence: append(
				append([]legalquery.EvidenceCode(nil), read.evidence...),
				current.evidence...,
			),
			concepts: append(
				append([]legalquery.LegalConceptSource(nil), read.concepts...),
				current.concepts...,
			),
			steps: append(
				append([]stepDraft(nil), read.steps...),
				current.steps...,
			),
			preserveMorphologicalContext: read.preserveMorphologicalContext ||
				current.preserveMorphologicalContext,
		}
		sort.SliceStable(combined.steps, func(left int, right int) bool {
			return combined.steps[left].startByte <
				combined.steps[right].startByte
		})
		assignJudicialStepOrdinals(&combined)
		if len(combined.steps) > legalquery.MaxCapabilityCalls {
			return nil, true
		}
		result = append(result, combined)
	}
	return result, false
}

func assignJudicialStepOrdinals(draft *candidateDraft) {
	if draft == nil {
		return
	}
	for index := range draft.steps {
		draft.steps[index].topicOrdinal = index + 1
	}
}

func judicialCompositionMembers(
	values []materializedCandidate,
	mode legalquery.QuerySelectionMode,
) ([]legalquery.QueryCandidateCompositionMember, error) {
	if mode != legalquery.QuerySelectionModeAutomatic {
		return nil, nil
	}
	result := make(
		[]legalquery.QueryCandidateCompositionMember,
		0,
		len(values),
	)
	for _, current := range values {
		steps := current.candidate.Steps()
		if len(steps) != len(current.sourceStarts) {
			return nil, fmt.Errorf("裁判例候補の step と原文位置が一致しません")
		}
		origins := make([]legalquery.QueryCandidateStepOrigin, 0, len(steps))
		positioned := true
		for index, step := range steps {
			if current.sourceStarts[index] < 0 {
				positioned = false
				break
			}
			origin, err := legalquery.NewQueryCandidateStepOrigin(
				legalquery.QueryCandidateStepOriginValues{
					StepID:          step.StepID(),
					SourceStartByte: current.sourceStarts[index],
				},
			)
			if err != nil {
				return nil, err
			}
			origins = append(origins, origin)
		}
		if !positioned {
			continue
		}
		member, err := legalquery.NewQueryCandidateCompositionMember(
			legalquery.QueryCandidateCompositionMemberValues{
				CandidateID: current.candidate.CandidateID(),
				Role:        legalquery.QueryCandidateCompositionRoleRequiredMember,
				StepOrigins: origins,
			},
		)
		if err != nil {
			return nil, err
		}
		result = append(result, member)
	}
	return result, nil
}

func (p *Profile) newGeneration(
	candidates []legalquery.LegalQueryCandidate,
	signals []legalquery.CandidateGenerationSignal,
	mode legalquery.QuerySelectionMode,
	members []legalquery.QueryCandidateCompositionMember,
	constraint legalquery.QueryCompositionConstraint,
) (legalquery.CandidateGeneration, error) {
	return legalquery.NewCandidateGeneration(
		legalquery.CandidateGenerationValues{
			ProfileID:             p.metadata.ProfileID(),
			ProfileVersion:        p.metadata.ProfileVersion(),
			RankingVersion:        p.metadata.RankingVersion(),
			Candidates:            candidates,
			Signals:               signals,
			SelectionMode:         mode,
			CompositionMembers:    members,
			CompositionConstraint: constraint,
		},
	)
}
