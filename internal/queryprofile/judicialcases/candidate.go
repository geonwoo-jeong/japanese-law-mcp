package judicialcases

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

type preparedDraft struct {
	evidence   []legalquery.EvidenceCode
	concepts   []legalquery.LegalConceptSource
	steps      []stepDraft
	score      int
	confidence legalquery.Confidence
	signature  string
}

type materializedCandidate struct {
	candidate    legalquery.LegalQueryCandidate
	sourceStarts []int
}

func (p *Profile) materializeCandidates(
	drafts []candidateDraft,
	scope legalquery.CandidateIDScope,
) ([]legalquery.LegalQueryCandidate, error) {
	materialized, err := p.materializeCandidateRecords(drafts, scope)
	if err != nil {
		return nil, err
	}
	result := make([]legalquery.LegalQueryCandidate, 0, len(materialized))
	for _, current := range materialized {
		result = append(result, current.candidate)
	}
	return result, nil
}

func (p *Profile) materializeCandidateRecords(
	drafts []candidateDraft,
	scope legalquery.CandidateIDScope,
) ([]materializedCandidate, error) {
	prepared := make([]preparedDraft, 0, len(drafts))
	bySignature := make(map[string]int, len(drafts))
	for _, original := range drafts {
		steps := append([]stepDraft(nil), original.steps...)
		sort.SliceStable(steps, func(left, right int) bool {
			return steps[left].startByte < steps[right].startByte
		})
		if len(steps) == 0 || len(steps) > 4 {
			return nil, fmt.Errorf(
				"一候補の logical step は一件以上四件以下でなければなりません",
			)
		}
		signature, err := judicialMeaningSignature(steps)
		if err != nil {
			return nil, err
		}
		evidence := normalizeEvidence(
			original.evidence,
			preservesLegalConceptForDistinctStep(
				steps,
				original.concepts,
			),
		)
		concepts := conceptSourcesForEvidence(
			evidence,
			original.concepts,
		)
		if index, exists := bySignature[signature]; exists {
			current := prepared[index]
			mergedSteps := append([]stepDraft(nil), current.steps...)
			for stepIndex := range mergedSteps {
				if steps[stepIndex].startByte <
					mergedSteps[stepIndex].startByte {
					mergedSteps[stepIndex].startByte =
						steps[stepIndex].startByte
				}
			}
			mergedConcepts := append(
				append(
					[]legalquery.LegalConceptSource(nil),
					current.concepts...,
				),
				concepts...,
			)
			mergedEvidence := mergeEvidence(
				current.evidence,
				evidence,
				preservesLegalConceptForDistinctStep(
					mergedSteps,
					mergedConcepts,
				),
			)
			prepared[index] = preparedDraft{
				evidence: mergedEvidence,
				concepts: conceptSourcesForEvidence(
					mergedEvidence,
					mergedConcepts,
				),
				steps:      mergedSteps,
				score:      current.score,
				confidence: current.confidence,
				signature:  current.signature,
			}
			continue
		}
		score, err := p.metadata.Score().Score(evidence)
		if err != nil {
			return nil, err
		}
		confidence, err := p.metadata.Score().ConfidenceFor(score)
		if err != nil {
			return nil, err
		}
		bySignature[signature] = len(prepared)
		prepared = append(prepared, preparedDraft{
			evidence:   evidence,
			concepts:   concepts,
			steps:      steps,
			score:      score,
			confidence: confidence,
			signature:  signature,
		})
	}
	if len(prepared) > maximumGeneratedCandidates {
		return nil, fmt.Errorf(
			"judicial-cases profile の候補は %d 件以下でなければなりません",
			maximumGeneratedCandidates,
		)
	}
	for index := range prepared {
		score, err := p.metadata.Score().Score(prepared[index].evidence)
		if err != nil {
			return nil, err
		}
		confidence, err := p.metadata.Score().ConfidenceFor(score)
		if err != nil {
			return nil, err
		}
		prepared[index].score = score
		prepared[index].confidence = confidence
	}
	sort.SliceStable(prepared, func(left, right int) bool {
		return comparePreparedDrafts(
			p.metadata.TieBreak(),
			prepared[left],
			prepared[right],
		) < 0
	})

	result := make([]materializedCandidate, 0, len(prepared))
	for index, current := range prepared {
		inputs := make([]legalquery.LogicalInput, 0, len(current.steps))
		sourceStarts := make([]int, 0, len(current.steps))
		for _, step := range current.steps {
			inputs = append(inputs, step.input)
			sourceStarts = append(sourceStarts, step.startByte)
		}
		candidate, err := legalquery.AssembleLegalQueryCandidate(
			legalquery.CandidateAssemblyValues{
				IDScope:          scope,
				CandidateOrdinal: index + 1,
				SemanticScore:    current.score,
				Confidence:       current.confidence,
				EvidenceCodes:    current.evidence,
				ConceptSources:   current.concepts,
				RequiredPacks:    []string{requiredPackID},
				LogicalInputs:    inputs,
			},
		)
		if err != nil {
			return nil, err
		}
		result = append(result, materializedCandidate{
			candidate:    candidate,
			sourceStarts: sourceStarts,
		})
	}
	return result, nil
}

func normalizeEvidence(
	values []legalquery.EvidenceCode,
	preserveLegalConcept bool,
) []legalquery.EvidenceCode {
	present := make(map[legalquery.EvidenceCode]struct{}, len(values))
	for _, value := range values {
		present[value] = struct{}{}
	}
	if _, exists := present[legalquery.EvidenceOfficialIdentifier]; exists {
		delete(present, legalquery.EvidenceOfficialAlias)
		if !preserveLegalConcept {
			delete(present, legalquery.EvidenceLegalConcept)
		}
		delete(present, legalquery.EvidenceMorphologicalContext)
		delete(present, legalquery.EvidenceUniqueTypoCorrection)
		delete(present, legalquery.EvidenceGeneralTerm)
	} else if _, exists := present[legalquery.EvidenceLegalConcept]; exists {
		delete(present, legalquery.EvidenceMorphologicalContext)
		delete(present, legalquery.EvidenceGeneralTerm)
	} else if _, exists := present[legalquery.EvidenceMorphologicalContext]; exists {
		delete(present, legalquery.EvidenceGeneralTerm)
	}
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
	result := make([]legalquery.EvidenceCode, 0, len(present))
	for _, code := range order {
		if _, exists := present[code]; exists {
			result = append(result, code)
		}
	}
	return result
}

func mergeEvidence(
	left []legalquery.EvidenceCode,
	right []legalquery.EvidenceCode,
	preserveLegalConcept bool,
) []legalquery.EvidenceCode {
	return normalizeEvidence(append(
		append([]legalquery.EvidenceCode(nil), left...),
		right...,
	), preserveLegalConcept)
}

func preservesLegalConceptForDistinctStep(
	steps []stepDraft,
	concepts []legalquery.LegalConceptSource,
) bool {
	if len(concepts) == 0 || len(steps) < 2 {
		return false
	}
	for _, step := range steps {
		if step.input.InputKind() ==
			legalquery.InputKindJudicialDecisionSearch {
			return true
		}
	}
	return false
}

func conceptSourcesForEvidence(
	evidence []legalquery.EvidenceCode,
	values []legalquery.LegalConceptSource,
) []legalquery.LegalConceptSource {
	if !slices.Contains(evidence, legalquery.EvidenceLegalConcept) {
		return nil
	}
	return uniqueConceptSources(values)
}

func uniqueConceptSources(
	values []legalquery.LegalConceptSource,
) []legalquery.LegalConceptSource {
	byID := make(map[string]legalquery.LegalConceptSource, len(values))
	for _, value := range values {
		byID[value.ConceptID()] = value
	}
	ids := make([]string, 0, len(byID))
	for conceptID := range byID {
		ids = append(ids, conceptID)
	}
	slices.Sort(ids)
	result := make([]legalquery.LegalConceptSource, 0, len(ids))
	for _, conceptID := range ids {
		result = append(result, byID[conceptID])
	}
	return result
}

func judicialMeaningSignature(steps []stepDraft) (string, error) {
	parts := make([]string, 0, len(steps))
	for _, step := range steps {
		switch input := step.input.(type) {
		case legalquery.JudicialDecisionSearchIntentV1:
			parts = append(parts, "search|"+input.Query())
		case legalquery.JudicialDecisionReadIntentV1:
			ref := input.Ref()
			key := ref.Key()
			parts = append(parts, strings.Join([]string{
				"read",
				ref.ProviderID(),
				key.SourceID(),
				key.ResourceType(),
				key.ResourceID(),
			}, "|"))
		default:
			return "", fmt.Errorf(
				"judicial-cases profile が未対応の logical input を生成しました",
			)
		}
	}
	return strings.Join(parts, "\x1f"), nil
}

func comparePreparedDrafts(
	policy []legalquery.QueryTieBreak,
	left preparedDraft,
	right preparedDraft,
) int {
	if left.score != right.score {
		return right.score - left.score
	}
	for _, rule := range policy {
		switch rule {
		case legalquery.QueryTieBreakEvidenceSet:
			leftEvidence := evidenceSignature(left.evidence)
			rightEvidence := evidenceSignature(right.evidence)
			if leftEvidence != rightEvidence {
				return strings.Compare(leftEvidence, rightEvidence)
			}
		case legalquery.QueryTieBreakStepCount:
			if len(left.steps) != len(right.steps) {
				return len(left.steps) - len(right.steps)
			}
		case legalquery.QueryTieBreakMeaningSignature:
			if left.signature != right.signature {
				return strings.Compare(left.signature, right.signature)
			}
		case legalquery.QueryTieBreakSourcePosition:
			if leftPosition, rightPosition := sourcePosition(left.steps), sourcePosition(right.steps); leftPosition != rightPosition {
				return leftPosition - rightPosition
			}
		}
	}
	return 0
}

func evidenceSignature(values []legalquery.EvidenceCode) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, string(value))
	}
	return strings.Join(parts, "\x00")
}

func sourcePosition(values []stepDraft) int {
	if len(values) == 0 {
		return 0
	}
	position := values[0].startByte
	for _, step := range values[1:] {
		if step.startByte < position {
			position = step.startByte
		}
	}
	return position
}
