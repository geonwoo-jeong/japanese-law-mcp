package core

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type corePreparedDraft struct {
	draft      candidateDraft
	cluster    string
	evidence   []legalquery.EvidenceCode
	concepts   []legalquery.LegalConceptSource
	score      int
	confidence legalquery.Confidence
	signature  string
}

func (p *Profile) materializeCoreEvidenceCandidates(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	drafts []candidateDraft,
	scope legalquery.CandidateIDScope,
) (
	[]legalquery.LegalQueryCandidate,
	[][]int,
	bool,
	error,
) {
	ordered, err := orderCoreEvidenceDraftSteps(drafts)
	if err != nil {
		return nil, nil, false, err
	}
	merged, err := mergeCoreEvidenceEquivalentDrafts(input, cues, ordered)
	if err != nil {
		return nil, nil, false, err
	}
	evaluation, err := buildCoreEvidenceEvaluation(input, cues, merged)
	if err != nil {
		return nil, nil, false, err
	}
	prepared, err := p.prepareCoreEvidenceDrafts(merged, evaluation)
	if err != nil {
		return nil, nil, false, err
	}
	retained, forced, err := p.retainCoreEvidenceBranches(prepared)
	if err != nil {
		return nil, nil, false, err
	}
	candidates, starts, err := assembleCoreEvidenceCandidates(
		retained,
		scope,
	)
	return candidates, starts, forced, err
}

func orderCoreEvidenceDraftSteps(
	values []candidateDraft,
) ([]candidateDraft, error) {
	result := make([]candidateDraft, 0, len(values))
	for _, value := range values {
		current := cloneDraft(value)
		sort.SliceStable(current.steps, func(left int, right int) bool {
			return current.steps[left].startByte < current.steps[right].startByte
		})
		if len(current.steps) < 1 ||
			len(current.steps) > legalquery.MaxCapabilityCalls {
			return nil, fmt.Errorf(
				"一候補の logical step は一件以上四件以下でなければなりません",
			)
		}
		result = append(result, current)
	}
	return result, nil
}

func mergeCoreEvidenceEquivalentDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	values []candidateDraft,
) ([]candidateDraft, error) {
	evaluation, err := buildCoreEvidenceEvaluation(input, cues, values)
	if err != nil {
		return nil, err
	}
	result := make([]candidateDraft, 0, len(evaluation.drafts))
	byKey := make(map[string]int, len(evaluation.drafts))
	for _, reference := range evaluation.drafts {
		index, err := coreEvidenceReferenceIndex(reference, len(values))
		if err != nil {
			return nil, err
		}
		key, eligible, err := evaluation.mapping.ClusterKey(reference.draftID)
		if err != nil {
			return nil, err
		}
		if !eligible {
			continue
		}
		signature, err := draftMeaningSignature(values[index])
		if err != nil {
			return nil, err
		}
		groupKey := key.Canonical() + "\x1e" + signature
		if targetIndex, exists := byKey[groupKey]; exists {
			if err := mergeCoreEvidenceDraft(
				&result[targetIndex],
				values[index],
			); err != nil {
				return nil, err
			}
			continue
		}
		byKey[groupKey] = len(result)
		result = append(result, cloneDraft(values[index]))
	}
	return result, nil
}

func coreEvidenceReferenceIndex(
	reference coreEvidenceDraftRef,
	limit int,
) (int, error) {
	ordinal, err := strconv.Atoi(strings.TrimPrefix(
		reference.draftID,
		"draft-",
	))
	if err != nil || ordinal < 1 || ordinal > limit {
		return 0, fmt.Errorf(
			"core evidence draft 参照 %q が有効ではありません",
			reference.draftID,
		)
	}
	return ordinal - 1, nil
}

func mergeCoreEvidenceDraft(
	target *candidateDraft,
	source candidateDraft,
) error {
	for _, sourceStep := range source.steps {
		sourceSignature, err := logicalInputSignature(sourceStep.input)
		if err != nil {
			return err
		}
		matched := false
		for index := range target.steps {
			targetSignature, err := logicalInputSignature(
				target.steps[index].input,
			)
			if err != nil {
				return err
			}
			if targetSignature != sourceSignature ||
				target.steps[index].topicOrdinal != sourceStep.topicOrdinal {
				continue
			}
			target.steps[index].evidenceBindings =
				dedupeCoreEvidenceValues(append(
					target.steps[index].evidenceBindings,
					sourceStep.evidenceBindings...,
				))
			if sourceStep.startByte < target.steps[index].startByte {
				target.steps[index].startByte = sourceStep.startByte
			}
			matched = true
			break
		}
		if !matched {
			return fmt.Errorf(
				"同値 draft の logical step 対応が一致しません",
			)
		}
	}
	concepts, err := mergeCoreConceptSources(
		target.concepts,
		source.concepts,
	)
	if err != nil {
		return err
	}
	target.concepts = concepts
	target.requiredPacks = append(
		target.requiredPacks,
		source.requiredPacks...,
	)
	return nil
}

func mergeCoreConceptSources(
	left []legalquery.LegalConceptSource,
	right []legalquery.LegalConceptSource,
) ([]legalquery.LegalConceptSource, error) {
	byID := make(map[string]legalquery.LegalConceptSource, len(left)+len(right))
	for _, value := range append(
		append([]legalquery.LegalConceptSource(nil), left...),
		right...,
	) {
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

func (p *Profile) prepareCoreEvidenceDrafts(
	values []candidateDraft,
	evaluation coreEvidenceEvaluation,
) ([]corePreparedDraft, error) {
	result := make([]corePreparedDraft, 0, len(evaluation.drafts))
	for _, reference := range evaluation.drafts {
		index, err := coreEvidenceReferenceIndex(reference, len(values))
		if err != nil {
			return nil, err
		}
		cluster, eligible, err := evaluation.mapping.ClusterKey(
			reference.draftID,
		)
		if err != nil {
			return nil, err
		}
		if !eligible {
			continue
		}
		summary, err := coreNormalizedEvidenceFor(
			evaluation.mapping,
			reference,
			evaluation.facts.byID,
		)
		if err != nil {
			return nil, err
		}
		evidence := summary.codes
		concepts, err := coreConceptSourcesForEvidence(
			values[index].concepts,
			summary.conceptIDs,
		)
		if err != nil {
			return nil, err
		}
		score, err := p.metadata.Score().Score(evidence)
		if err != nil {
			return nil, err
		}
		confidence, err := p.metadata.Score().ConfidenceFor(score)
		if err != nil {
			return nil, err
		}
		signature, err := draftMeaningSignature(values[index])
		if err != nil {
			return nil, err
		}
		result = append(result, corePreparedDraft{
			draft:      cloneDraft(values[index]),
			cluster:    cluster.Canonical(),
			evidence:   evidence,
			concepts:   concepts,
			score:      score,
			confidence: confidence,
			signature:  signature,
		})
	}
	sort.SliceStable(result, func(left int, right int) bool {
		return compareCorePreparedDraft(result[left], result[right]) < 0
	})
	return result, nil
}

type coreNormalizedEvidence struct {
	codes      []legalquery.EvidenceCode
	conceptIDs map[string]struct{}
}

func coreNormalizedEvidenceFor(
	mapping profileevidence.Mapping,
	reference coreEvidenceDraftRef,
	facts map[string]coreEvidenceFact,
) (coreNormalizedEvidence, error) {
	present := make(map[legalquery.EvidenceCode]struct{})
	conceptIDs := make(map[string]struct{})
	for _, stepID := range reference.stepIDs {
		values, err := mapping.NormalizedStepEvidence(
			reference.draftID,
			stepID,
		)
		if err != nil {
			return coreNormalizedEvidence{}, err
		}
		for _, value := range values {
			present[value.Code()] = struct{}{}
			if value.Code() != legalquery.EvidenceLegalConcept {
				continue
			}
			fact, exists := facts[value.FactID()]
			if !exists || fact.kind != coreEvidenceFactLegalConcept ||
				fact.conceptID == "" {
				return coreNormalizedEvidence{}, fmt.Errorf(
					"legal_concept fact %q の source 対応がありません",
					value.FactID(),
				)
			}
			conceptIDs[fact.conceptID] = struct{}{}
		}
	}
	codes := make([]legalquery.EvidenceCode, 0, len(present))
	for _, code := range evidenceOrder {
		if _, exists := present[code]; exists {
			codes = append(codes, code)
		}
	}
	return coreNormalizedEvidence{
		codes:      codes,
		conceptIDs: conceptIDs,
	}, nil
}

func coreConceptSourcesForEvidence(
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
	result, err := mergeCoreConceptSources(filtered, nil)
	if err != nil {
		return nil, err
	}
	if len(result) != len(conceptIDs) {
		return nil, fmt.Errorf(
			"保持した legal_concept fact の source が不足しています",
		)
	}
	return result, nil
}

func compareCorePreparedDraft(
	left corePreparedDraft,
	right corePreparedDraft,
) int {
	if left.score != right.score {
		return right.score - left.score
	}
	if leftEvidence, rightEvidence :=
		evidenceSignature(left.evidence),
		evidenceSignature(right.evidence); leftEvidence != rightEvidence {
		return strings.Compare(leftEvidence, rightEvidence)
	}
	if len(left.draft.steps) != len(right.draft.steps) {
		return len(left.draft.steps) - len(right.draft.steps)
	}
	if left.signature != right.signature {
		return strings.Compare(left.signature, right.signature)
	}
	return sourcePosition(left.draft) - sourcePosition(right.draft)
}

func (p *Profile) retainCoreEvidenceBranches(
	values []corePreparedDraft,
) ([]corePreparedDraft, bool, error) {
	margin, present := p.metadata.Selection().BranchRetentionMargin()
	if !present {
		return nil, false, fmt.Errorf(
			"core evidence profile に branchRetentionMargin がありません",
		)
	}
	leaders := make(map[string]int)
	counts := make(map[string]int)
	result := make([]corePreparedDraft, 0, len(values))
	for _, value := range values {
		leaderScore, exists := leaders[value.cluster]
		if !exists {
			leaders[value.cluster] = value.score
			counts[value.cluster] = 1
			result = append(result, value)
			continue
		}
		if leaderScore-value.score > margin {
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
			"core profile の候補は %d 件以下でなければなりません",
			maximumGeneratedCandidates,
		)
	}
	return result, forced, nil
}

func assembleCoreEvidenceCandidates(
	values []corePreparedDraft,
	scope legalquery.CandidateIDScope,
) ([]legalquery.LegalQueryCandidate, [][]int, error) {
	candidates := make([]legalquery.LegalQueryCandidate, 0, len(values))
	starts := make([][]int, 0, len(values))
	for index, value := range values {
		inputs := make([]legalquery.LogicalInput, 0, len(value.draft.steps))
		stepStarts := make([]int, 0, len(value.draft.steps))
		for _, step := range value.draft.steps {
			inputs = append(inputs, step.input)
			stepStarts = append(stepStarts, step.startByte)
		}
		concepts := slices.Clone(value.concepts)
		packs := append([]string(nil), value.draft.requiredPacks...)
		slices.Sort(packs)
		packs = slices.Compact(packs)
		candidate, err := legalquery.AssembleLegalQueryCandidate(
			legalquery.CandidateAssemblyValues{
				IDScope:          scope,
				CandidateOrdinal: index + 1,
				SemanticScore:    value.score,
				Confidence:       value.confidence,
				EvidenceCodes:    value.evidence,
				ConceptSources:   concepts,
				RequiredPacks:    packs,
				LogicalInputs:    inputs,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		candidates = append(candidates, candidate)
		starts = append(starts, stepStarts)
	}
	return candidates, starts, nil
}
