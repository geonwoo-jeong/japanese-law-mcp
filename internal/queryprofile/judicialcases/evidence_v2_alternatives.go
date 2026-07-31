package judicialcases

import (
	"fmt"
	"slices"
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type judicialTopicOption struct {
	draft     candidateDraft
	meaning   string
	evidence  []legalquery.EvidenceCode
	score     int
	startByte int
}

func (p *Profile) buildJudicialEvidenceSearchDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, bool, bool, error) {
	if !cues.has("task", "search") ||
		!cues.has("resource", "judicial_decision") {
		return nil, false, false, nil
	}
	raw, ambiguous, err := p.buildSearchSubjects(input, cues)
	if err != nil {
		return nil, false, false, err
	}
	bound, err := withJudicialEvidenceBindings(input, cues, raw)
	if err != nil {
		return nil, false, false, err
	}
	if !cues.has("operator", "individual") || len(bound) < 2 {
		return bound, false, ambiguous, nil
	}

	groups, err := p.judicialTopicOptionGroups(bound)
	if err != nil {
		return nil, false, false, err
	}
	if len(groups) > legalquery.MaxCapabilityCalls {
		return nil, true, ambiguous, nil
	}
	if len(groups) < 2 {
		return bound, false, ambiguous, nil
	}
	choices := boundedJudicialTopicChoices(groups)
	result := make([]candidateDraft, 0, len(choices))
	for _, choice := range choices {
		draft, buildErr := buildJudicialTopicChoiceDraft(choice)
		if buildErr != nil {
			return nil, false, false, buildErr
		}
		result = append(result, draft)
	}
	return result, false, ambiguous, nil
}

func (p *Profile) judicialTopicOptionGroups(
	values []candidateDraft,
) ([][]judicialTopicOption, error) {
	ordered := append([]candidateDraft(nil), values...)
	sort.SliceStable(ordered, func(left int, right int) bool {
		return ordered[left].steps[0].startByte < ordered[right].steps[0].startByte
	})
	byStart := make(map[int]int)
	groups := make([][]candidateDraft, 0)
	for _, value := range ordered {
		if len(value.steps) != 1 {
			return nil, fmt.Errorf("judicial topic option は一 step でなければなりません")
		}
		start := value.steps[0].startByte
		index, exists := byStart[start]
		if !exists {
			index = len(groups)
			byStart[start] = index
			groups = append(groups, nil)
		}
		groups[index] = append(groups[index], cloneJudicialDraft(value))
	}

	result := make([][]judicialTopicOption, 0, len(groups))
	for _, group := range groups {
		options, err := p.rankJudicialTopicOptions(group)
		if err != nil {
			return nil, err
		}
		if len(options) == 0 {
			return nil, fmt.Errorf("judicial topic に有効な意味候補がありません")
		}
		result = append(result, options)
	}
	return result, nil
}

func (p *Profile) rankJudicialTopicOptions(
	values []candidateDraft,
) ([]judicialTopicOption, error) {
	unique := make([]candidateDraft, 0, len(values))
	byMeaning := make(map[string]int, len(values))
	for _, value := range values {
		meaning, err := judicialDraftMeaningSignature(value)
		if err != nil {
			return nil, err
		}
		if index, exists := byMeaning[meaning]; exists {
			merged, mergeErr := mergeJudicialEquivalentDrafts(
				unique[index],
				value,
			)
			if mergeErr != nil {
				return nil, mergeErr
			}
			unique[index] = merged
			continue
		}
		byMeaning[meaning] = len(unique)
		unique = append(unique, cloneJudicialDraft(value))
	}

	result := make([]judicialTopicOption, 0, len(unique))
	for _, value := range unique {
		meaning, err := judicialDraftMeaningSignature(value)
		if err != nil {
			return nil, err
		}
		codes := judicialCodesFromBindings(value.steps[0].evidenceBindings)
		if len(codes) == 0 {
			codes = judicialUnionEvidenceCodes(value.steps[0].evidenceCodes)
		}
		score, err := p.metadata.Score().Score(codes)
		if err != nil {
			return nil, err
		}
		result = append(result, judicialTopicOption{
			draft:     cloneJudicialDraft(value),
			meaning:   meaning,
			evidence:  codes,
			score:     score,
			startByte: value.steps[0].startByte,
		})
	}
	sort.Slice(result, func(left int, right int) bool {
		if result[left].score != result[right].score {
			return result[left].score > result[right].score
		}
		leftEvidence := evidenceSignature(result[left].evidence)
		rightEvidence := evidenceSignature(result[right].evidence)
		if leftEvidence != rightEvidence {
			return leftEvidence < rightEvidence
		}
		if result[left].meaning != result[right].meaning {
			return result[left].meaning < result[right].meaning
		}
		return result[left].startByte < result[right].startByte
	})
	return result, nil
}

func boundedJudicialTopicChoices(
	groups [][]judicialTopicOption,
) [][]judicialTopicOption {
	baseline := make([]judicialTopicOption, 0, len(groups))
	for _, group := range groups {
		if len(group) == 0 {
			return nil
		}
		baseline = append(baseline, group[0])
	}
	result := [][]judicialTopicOption{baseline}
	for topicIndex, group := range groups {
		for optionIndex := 1; optionIndex < len(group); optionIndex++ {
			choice := append([]judicialTopicOption(nil), baseline...)
			choice[topicIndex] = group[optionIndex]
			result = append(result, choice)
		}
	}
	return result
}

func buildJudicialTopicChoiceDraft(
	values []judicialTopicOption,
) (candidateDraft, error) {
	result := candidateDraft{}
	for index, value := range values {
		if len(value.draft.steps) != 1 {
			return candidateDraft{}, fmt.Errorf(
				"judicial topic choice は一 step でなければなりません",
			)
		}
		step := cloneJudicialStep(value.draft.steps[0])
		step.topicOrdinal = index + 1
		result.steps = append(result.steps, step)
		result.concepts = append(result.concepts, value.draft.concepts...)
	}
	concepts, err := mergeJudicialConceptSources(result.concepts, nil)
	if err != nil {
		return candidateDraft{}, err
	}
	result.concepts = concepts
	return result, nil
}

func judicialCodesFromBindings(
	values []profileevidence.EvidenceValues,
) []legalquery.EvidenceCode {
	result := make([]legalquery.EvidenceCode, 0, len(values))
	for _, value := range values {
		result = append(result, value.Code)
	}
	return judicialUnionEvidenceCodes(result)
}

func judicialUnionEvidenceCodes(
	values []legalquery.EvidenceCode,
) []legalquery.EvidenceCode {
	order := []legalquery.EvidenceCode{
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceStructuredReference,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceOfficialAlias,
		legalquery.EvidenceLegalConcept,
		legalquery.EvidenceMorphologicalContext,
		legalquery.EvidenceUniqueTypoCorrection,
		legalquery.EvidenceGeneralTerm,
	}
	present := make(map[legalquery.EvidenceCode]struct{}, len(values))
	for _, value := range values {
		present[value] = struct{}{}
	}
	result := make([]legalquery.EvidenceCode, 0, len(present))
	for _, value := range order {
		if _, exists := present[value]; exists {
			result = append(result, value)
		}
	}
	return result
}

func cloneJudicialDraft(value candidateDraft) candidateDraft {
	steps := make([]stepDraft, 0, len(value.steps))
	for _, step := range value.steps {
		steps = append(steps, cloneJudicialStep(step))
	}
	return candidateDraft{
		evidence:                     slices.Clone(value.evidence),
		concepts:                     slices.Clone(value.concepts),
		steps:                        steps,
		preserveMorphologicalContext: value.preserveMorphologicalContext,
	}
}

func cloneJudicialStep(value stepDraft) stepDraft {
	value.evidenceCodes = slices.Clone(value.evidenceCodes)
	value.evidenceBindings = slices.Clone(value.evidenceBindings)
	return value
}

func mergeJudicialEquivalentDrafts(
	left candidateDraft,
	right candidateDraft,
) (candidateDraft, error) {
	if len(left.steps) != 1 || len(right.steps) != 1 {
		return candidateDraft{}, fmt.Errorf("同値 judicial draft の step が一致しません")
	}
	result := cloneJudicialDraft(left)
	result.steps[0].evidenceCodes = judicialUnionEvidenceCodes(append(
		result.steps[0].evidenceCodes,
		right.steps[0].evidenceCodes...,
	))
	result.steps[0].evidenceBindings = append(
		result.steps[0].evidenceBindings,
		right.steps[0].evidenceBindings...,
	)
	if right.steps[0].startByte < result.steps[0].startByte {
		result.steps[0].startByte = right.steps[0].startByte
	}
	concepts, err := mergeJudicialConceptSources(left.concepts, right.concepts)
	if err != nil {
		return candidateDraft{}, err
	}
	result.concepts = concepts
	return result, nil
}

func mergeJudicialConceptSources(
	left []legalquery.LegalConceptSource,
	right []legalquery.LegalConceptSource,
) ([]legalquery.LegalConceptSource, error) {
	byID := make(map[string]legalquery.LegalConceptSource, len(left)+len(right))
	for _, value := range append(slices.Clone(left), right...) {
		if previous, exists := byID[value.ConceptID()]; exists &&
			previous != value {
			return nil, fmt.Errorf(
				"conceptId %q の source tuple が競合しています",
				value.ConceptID(),
			)
		}
		byID[value.ConceptID()] = value
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	result := make([]legalquery.LegalConceptSource, 0, len(ids))
	for _, id := range ids {
		result = append(result, byID[id])
	}
	return result, nil
}
