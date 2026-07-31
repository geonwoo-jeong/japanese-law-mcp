package judicialcases

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type judicialEvidencePreparedDraft struct {
	draft      candidateDraft
	cluster    string
	evidence   []legalquery.EvidenceCode
	concepts   []legalquery.LegalConceptSource
	score      int
	confidence legalquery.Confidence
	signature  string
}

func (p *Profile) materializeJudicialEvidenceCandidateRecords(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	drafts []candidateDraft,
	scope legalquery.CandidateIDScope,
) ([]materializedCandidate, bool, error) {
	ordered, err := orderJudicialEvidenceDrafts(drafts)
	if err != nil {
		return nil, false, err
	}
	bound, err := withJudicialEvidenceBindings(input, cues, ordered)
	if err != nil {
		return nil, false, err
	}
	evaluation, err := buildJudicialEvidenceEvaluation(input, cues, bound)
	if err != nil {
		return nil, false, err
	}
	prepared, err := p.prepareJudicialEvidenceDrafts(bound, evaluation)
	if err != nil {
		return nil, false, err
	}
	prepared, err = p.mergeJudicialEvidenceEquivalentPrepared(prepared)
	if err != nil {
		return nil, false, err
	}
	retained, forced, err := p.retainJudicialEvidenceBranches(prepared)
	if err != nil {
		return nil, false, err
	}
	materialized, err := assembleJudicialEvidenceCandidates(retained, scope)
	return materialized, forced, err
}

func orderJudicialEvidenceDrafts(
	values []candidateDraft,
) ([]candidateDraft, error) {
	result := make([]candidateDraft, 0, len(values))
	for _, value := range values {
		current := cloneJudicialDraft(value)
		sort.SliceStable(current.steps, func(left int, right int) bool {
			return current.steps[left].startByte < current.steps[right].startByte
		})
		if len(current.steps) < 1 ||
			len(current.steps) > legalquery.MaxCapabilityCalls {
			return nil, fmt.Errorf(
				"一候補の logical step は一件以上四件以下でなければなりません",
			)
		}
		assignJudicialStepOrdinals(&current)
		result = append(result, current)
	}
	return result, nil
}

func (p *Profile) prepareJudicialEvidenceDrafts(
	values []candidateDraft,
	evaluation judicialEvidenceEvaluation,
) ([]judicialEvidencePreparedDraft, error) {
	result := make([]judicialEvidencePreparedDraft, 0, len(evaluation.drafts))
	for _, reference := range evaluation.drafts {
		if reference.draftIndex < 0 || reference.draftIndex >= len(values) {
			return nil, fmt.Errorf(
				"judicial evidence draft 参照 %q が有効ではありません",
				reference.draftID,
			)
		}
		cluster, eligible, err := evaluation.mapping.ClusterKey(reference.draftID)
		if err != nil {
			return nil, err
		}
		if !eligible {
			continue
		}
		summary, err := judicialNormalizedEvidenceFor(
			evaluation.mapping,
			reference,
			evaluation.facts,
		)
		if err != nil {
			return nil, err
		}
		concepts, err := judicialConceptSourcesForEvidence(
			values[reference.draftIndex].concepts,
			summary.conceptIDs,
		)
		if err != nil {
			return nil, err
		}
		score, err := p.metadata.Score().Score(summary.codes)
		if err != nil {
			return nil, err
		}
		confidence, err := p.metadata.Score().ConfidenceFor(score)
		if err != nil {
			return nil, err
		}
		signature, err := judicialDraftMeaningSignature(
			values[reference.draftIndex],
		)
		if err != nil {
			return nil, err
		}
		result = append(result, judicialEvidencePreparedDraft{
			draft:      cloneJudicialDraft(values[reference.draftIndex]),
			cluster:    cluster.Canonical(),
			evidence:   summary.codes,
			concepts:   concepts,
			score:      score,
			confidence: confidence,
			signature:  signature,
		})
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return compareJudicialEvidencePrepared(result[left], result[right]) < 0
	})
	return result, nil
}

type judicialNormalizedEvidence struct {
	codes      []legalquery.EvidenceCode
	conceptIDs map[string]struct{}
}

func judicialNormalizedEvidenceFor(
	mapping profileevidence.Mapping,
	reference judicialEvidenceDraftRef,
	facts judicialEvidenceFactSet,
) (judicialNormalizedEvidence, error) {
	var values []legalquery.EvidenceCode
	conceptIDs := make(map[string]struct{})
	for _, stepID := range reference.stepIDs {
		evidence, err := mapping.NormalizedStepEvidence(reference.draftID, stepID)
		if err != nil {
			return judicialNormalizedEvidence{}, err
		}
		for _, current := range evidence {
			values = append(values, current.Code())
			if current.Code() != legalquery.EvidenceLegalConcept {
				continue
			}
			fact, exists := facts.byID[current.FactID()]
			if !exists || fact.conceptID == "" {
				return judicialNormalizedEvidence{}, fmt.Errorf(
					"legal_concept fact %q の source 対応がありません",
					current.FactID(),
				)
			}
			conceptIDs[fact.conceptID] = struct{}{}
		}
	}
	return judicialNormalizedEvidence{
		codes:      judicialUnionEvidenceCodes(values),
		conceptIDs: conceptIDs,
	}, nil
}

func judicialConceptSourcesForEvidence(
	values []legalquery.LegalConceptSource,
	conceptIDs map[string]struct{},
) ([]legalquery.LegalConceptSource, error) {
	if len(conceptIDs) == 0 {
		return nil, nil
	}
	filtered := make([]legalquery.LegalConceptSource, 0, len(conceptIDs))
	for _, value := range values {
		if _, exists := conceptIDs[value.ConceptID()]; exists {
			filtered = append(filtered, value)
		}
	}
	result, err := mergeJudicialConceptSources(filtered, nil)
	if err != nil {
		return nil, err
	}
	if len(result) != len(conceptIDs) {
		return nil, fmt.Errorf("保持した legal_concept fact の source が不足しています")
	}
	return result, nil
}

func (p *Profile) mergeJudicialEvidenceEquivalentPrepared(
	values []judicialEvidencePreparedDraft,
) ([]judicialEvidencePreparedDraft, error) {
	result := make([]judicialEvidencePreparedDraft, 0, len(values))
	byKey := make(map[string]int, len(values))
	for _, value := range values {
		key := value.cluster + "\x1e" + value.signature
		index, exists := byKey[key]
		if !exists {
			byKey[key] = len(result)
			result = append(result, value)
			continue
		}
		current := result[index]
		current.evidence = judicialUnionEvidenceCodes(append(
			current.evidence,
			value.evidence...,
		))
		concepts, err := mergeJudicialConceptSources(
			current.concepts,
			value.concepts,
		)
		if err != nil {
			return nil, err
		}
		current.concepts = concepts
		for stepIndex := range current.draft.steps {
			if value.draft.steps[stepIndex].startByte <
				current.draft.steps[stepIndex].startByte {
				current.draft.steps[stepIndex].startByte =
					value.draft.steps[stepIndex].startByte
			}
		}
		current.score, err = p.metadata.Score().Score(current.evidence)
		if err != nil {
			return nil, err
		}
		current.confidence, err = p.metadata.Score().ConfidenceFor(current.score)
		if err != nil {
			return nil, err
		}
		result[index] = current
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return compareJudicialEvidencePrepared(result[left], result[right]) < 0
	})
	return result, nil
}

func compareJudicialEvidencePrepared(
	left judicialEvidencePreparedDraft,
	right judicialEvidencePreparedDraft,
) int {
	if left.score != right.score {
		return right.score - left.score
	}
	leftEvidence := evidenceSignature(left.evidence)
	rightEvidence := evidenceSignature(right.evidence)
	if leftEvidence != rightEvidence {
		return strings.Compare(leftEvidence, rightEvidence)
	}
	if len(left.draft.steps) != len(right.draft.steps) {
		return len(left.draft.steps) - len(right.draft.steps)
	}
	if left.signature != right.signature {
		return strings.Compare(left.signature, right.signature)
	}
	return sourcePosition(left.draft.steps) - sourcePosition(right.draft.steps)
}

func (p *Profile) retainJudicialEvidenceBranches(
	values []judicialEvidencePreparedDraft,
) ([]judicialEvidencePreparedDraft, bool, error) {
	margin, present := p.metadata.Selection().BranchRetentionMargin()
	if !present {
		return nil, false, fmt.Errorf(
			"judicial evidence profile に branchRetentionMargin がありません",
		)
	}
	leaders := make(map[string]int)
	counts := make(map[string]int)
	result := make([]judicialEvidencePreparedDraft, 0, len(values))
	for _, value := range values {
		leader, exists := leaders[value.cluster]
		if !exists {
			leaders[value.cluster] = value.score
			counts[value.cluster] = 1
			result = append(result, value)
			continue
		}
		if leader-value.score > margin {
			continue
		}
		counts[value.cluster]++
		if counts[value.cluster] <= 3 {
			result = append(result, value)
		}
	}
	forced := false
	for _, count := range counts {
		if count > 3 {
			forced = true
			break
		}
	}
	if len(result) > maximumGeneratedCandidates {
		return nil, false, fmt.Errorf(
			"judicial-cases profile の候補は %d 件以下でなければなりません",
			maximumGeneratedCandidates,
		)
	}
	return result, forced, nil
}

func assembleJudicialEvidenceCandidates(
	values []judicialEvidencePreparedDraft,
	scope legalquery.CandidateIDScope,
) ([]materializedCandidate, error) {
	result := make([]materializedCandidate, 0, len(values))
	for index, value := range values {
		inputs := make([]legalquery.LogicalInput, 0, len(value.draft.steps))
		starts := make([]int, 0, len(value.draft.steps))
		for _, step := range value.draft.steps {
			inputs = append(inputs, step.input)
			starts = append(starts, step.startByte)
		}
		candidate, err := legalquery.AssembleLegalQueryCandidate(
			legalquery.CandidateAssemblyValues{
				IDScope:          scope,
				CandidateOrdinal: index + 1,
				SemanticScore:    value.score,
				Confidence:       value.confidence,
				EvidenceCodes:    slices.Clone(value.evidence),
				ConceptSources:   slices.Clone(value.concepts),
				RequiredPacks:    []string{requiredPackID},
				LogicalInputs:    inputs,
			},
		)
		if err != nil {
			return nil, err
		}
		result = append(result, materializedCandidate{
			candidate:    candidate,
			sourceStarts: starts,
		})
	}
	return result, nil
}
