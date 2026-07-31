package core

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type coreSharedTerminalDraft struct {
	draft            candidateDraft
	meaningSignature string
}

type coreTopicOptionRank struct {
	option           coreTopicOption
	evidence         []legalquery.EvidenceCode
	score            int
	meaningSignature string
}

type coreSharedTerminalPreparedDraft struct {
	value      coreSharedTerminalDraft
	cluster    string
	evidence   []legalquery.EvidenceCode
	score      int
	confidence legalquery.Confidence
}

type coreSharedTerminalRetention struct {
	drafts                []coreSharedTerminalPreparedDraft
	clarificationRequired bool
}

func (p *Profile) orderCoreTopicOptions(
	values []coreTopicOption,
) ([]coreTopicOption, error) {
	unique := make([]coreTopicOption, 0, len(values))
	byMeaning := make(map[string]int, len(values))
	for _, value := range values {
		signature, err := coreTopicOptionMeaningSignature(value)
		if err != nil {
			return nil, err
		}
		if index, exists := byMeaning[signature]; exists {
			unique[index].evidence = dedupeCoreEvidenceValues(append(
				unique[index].evidence,
				value.evidence...,
			))
			concepts, mergeErr := mergeCoreSharedConceptSources(
				unique[index].concepts,
				value.concepts,
			)
			if mergeErr != nil {
				return nil, mergeErr
			}
			unique[index].concepts = concepts
			unique[index].startByte = min(
				unique[index].startByte,
				value.startByte,
			)
			continue
		}
		byMeaning[signature] = len(unique)
		unique = append(unique, cloneCoreTopicOption(value))
	}

	ranked := make([]coreTopicOptionRank, 0, len(unique))
	for _, value := range unique {
		evidence := normalizedCoreTopicEvidence(value.evidence)
		score, err := p.metadata.Score().Score(evidence)
		if err != nil {
			return nil, err
		}
		signature, err := coreTopicOptionMeaningSignature(value)
		if err != nil {
			return nil, err
		}
		ranked = append(ranked, coreTopicOptionRank{
			option:           value,
			evidence:         evidence,
			score:            score,
			meaningSignature: signature,
		})
	}
	sort.Slice(ranked, func(left, right int) bool {
		return compareCoreTopicOptionRank(ranked[left], ranked[right]) < 0
	})
	result := make([]coreTopicOption, 0, len(ranked))
	for _, value := range ranked {
		result = append(result, cloneCoreTopicOption(value.option))
	}
	return result, nil
}

func cloneCoreTopicOption(value coreTopicOption) coreTopicOption {
	return coreTopicOption{
		startByte:  value.startByte,
		input:      value.input,
		evidence:   slices.Clone(value.evidence),
		concepts:   slices.Clone(value.concepts),
		meaningKey: value.meaningKey,
	}
}

func coreTopicOptionMeaningSignature(value coreTopicOption) (string, error) {
	if value.meaningKey == "" {
		return "", fmt.Errorf("共有末尾 topic の意味 identity は必須です")
	}
	signature, err := logicalInputSignature(value.input)
	if err != nil {
		return "", err
	}
	return signature + "\x1f" + value.meaningKey, nil
}

func normalizedCoreTopicEvidence(
	values []profileevidence.EvidenceValues,
) []legalquery.EvidenceCode {
	present := make(map[legalquery.EvidenceCode]struct{}, len(values))
	for _, value := range values {
		present[value.Code] = struct{}{}
	}
	// topic-local option は一つの typed target だけを持つため、既存の閉じた
	// 優越表を step 内に限って適用できる。
	return normalizeEvidence(present, false, false, false)
}

func compareCoreTopicOptionRank(left, right coreTopicOptionRank) int {
	if left.score != right.score {
		return right.score - left.score
	}
	if leftEvidence, rightEvidence :=
		evidenceSignature(left.evidence),
		evidenceSignature(right.evidence); leftEvidence != rightEvidence {
		return strings.Compare(leftEvidence, rightEvidence)
	}
	// topic-local draft の step 数は常に一件である。
	if left.meaningSignature != right.meaningSignature {
		return strings.Compare(left.meaningSignature, right.meaningSignature)
	}
	return left.option.startByte - right.option.startByte
}

func coreBoundedTopicChoices(
	options [][]coreTopicOption,
) [][]coreTopicOption {
	if len(options) == 0 {
		return nil
	}
	baseline := make([]coreTopicOption, len(options))
	for index := range options {
		if len(options[index]) == 0 {
			return nil
		}
		baseline[index] = cloneCoreTopicOption(options[index][0])
	}
	result := [][]coreTopicOption{baseline}
	for topicIndex := range options {
		for optionIndex := 1; optionIndex < len(options[topicIndex]); optionIndex++ {
			choice := make([]coreTopicOption, len(baseline))
			for index := range baseline {
				choice[index] = cloneCoreTopicOption(baseline[index])
			}
			choice[topicIndex] = cloneCoreTopicOption(
				options[topicIndex][optionIndex],
			)
			result = append(result, choice)
		}
	}
	return result
}

func buildCoreSharedTerminalDraft(
	options []coreTopicOption,
) (coreSharedTerminalDraft, error) {
	draft := newCandidateDraft()
	byMeaning := make(map[string]int, len(options))
	selectedMeanings := make([]string, 0, len(options))
	for _, option := range options {
		meaning, err := coreTopicOptionMeaningSignature(option)
		if err != nil {
			return coreSharedTerminalDraft{}, err
		}
		selectedMeanings = append(selectedMeanings, meaning)
		if index, exists := byMeaning[meaning]; exists {
			draft.steps[index].evidenceBindings = dedupeCoreEvidenceValues(append(
				draft.steps[index].evidenceBindings,
				option.evidence...,
			))
			draft.steps[index].startByte = min(
				draft.steps[index].startByte,
				option.startByte,
			)
			continue
		}
		byMeaning[meaning] = len(draft.steps)
		draft.steps = append(draft.steps, stepDraft{
			startByte:        option.startByte,
			topicOrdinal:     len(draft.steps) + 1,
			input:            option.input,
			evidenceBindings: slices.Clone(option.evidence),
		})
		draft.concepts = append(draft.concepts, option.concepts...)
	}
	concepts, err := mergeCoreSharedConceptSources(nil, draft.concepts)
	if err != nil {
		return coreSharedTerminalDraft{}, err
	}
	draft.concepts = concepts
	for _, step := range draft.steps {
		for _, evidence := range step.evidenceBindings {
			draft.evidence[evidence.Code] = struct{}{}
		}
	}
	return coreSharedTerminalDraft{
		draft:            draft,
		meaningSignature: strings.Join(selectedMeanings, "\x1e"),
	}, nil
}

func (p *Profile) retainCoreSharedTerminalDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	values []coreSharedTerminalDraft,
) (coreSharedTerminalRetention, error) {
	prepared, err := p.prepareCoreSharedTerminalDrafts(input, cues, values)
	if err != nil {
		return coreSharedTerminalRetention{}, err
	}
	sortCoreSharedTerminalPrepared(prepared)
	merged, err := mergeCoreSharedTerminalEquivalentDrafts(prepared)
	if err != nil {
		return coreSharedTerminalRetention{}, err
	}
	prepared, err = p.prepareCoreSharedTerminalDrafts(input, cues, merged)
	if err != nil {
		return coreSharedTerminalRetention{}, err
	}
	sortCoreSharedTerminalPrepared(prepared)
	return p.retainPreparedCoreSharedTerminalBranches(prepared)
}

func (p *Profile) retainPreparedCoreSharedTerminalBranches(
	prepared []coreSharedTerminalPreparedDraft,
) (coreSharedTerminalRetention, error) {
	prepared = slices.Clone(prepared)
	sortCoreSharedTerminalPrepared(prepared)
	margin, present := p.metadata.Selection().BranchRetentionMargin()
	if !present {
		return coreSharedTerminalRetention{}, fmt.Errorf(
			"共有末尾分岐には branchRetentionMargin が必須です",
		)
	}
	leaders := make(map[string]int)
	eligibleCounts := make(map[string]int)
	retained := make([]coreSharedTerminalPreparedDraft, 0, len(prepared))
	for _, value := range prepared {
		leader, exists := leaders[value.cluster]
		if !exists {
			leaders[value.cluster] = value.score
			leader = value.score
		}
		if leader-value.score > margin {
			continue
		}
		eligibleCounts[value.cluster]++
		if eligibleCounts[value.cluster] <= 3 {
			retained = append(retained, cloneCoreSharedTerminalPrepared(value))
		}
	}
	if len(retained) > maximumGeneratedCandidates {
		return coreSharedTerminalRetention{}, fmt.Errorf(
			"core profile の候補は %d 件以下でなければなりません",
			maximumGeneratedCandidates,
		)
	}
	clarification := false
	for _, count := range eligibleCounts {
		if count > 3 {
			clarification = true
			break
		}
	}
	return coreSharedTerminalRetention{
		drafts:                retained,
		clarificationRequired: clarification,
	}, nil
}

func (p *Profile) prepareCoreSharedTerminalDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	values []coreSharedTerminalDraft,
) ([]coreSharedTerminalPreparedDraft, error) {
	result := make([]coreSharedTerminalPreparedDraft, 0, len(values))
	for _, value := range values {
		prepared, retained, err := p.prepareCoreSharedTerminalDraft(
			input,
			cues,
			value,
		)
		if err != nil {
			return nil, err
		}
		if retained {
			result = append(result, prepared)
		}
	}
	return result, nil
}

func (p *Profile) prepareCoreSharedTerminalDraft(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	value coreSharedTerminalDraft,
) (coreSharedTerminalPreparedDraft, bool, error) {
	facts, err := buildCoreEvidenceFacts(input, cues)
	if err != nil {
		return coreSharedTerminalPreparedDraft{}, false, err
	}
	raw, reference := coreEvidenceRawDraftValue("draft", value.draft)
	if err := validateCoreRawDraft(raw, facts.values); err != nil {
		return coreSharedTerminalPreparedDraft{}, false, err
	}
	if err := validateCoreSharedTerminalFactReuse(raw); err != nil {
		return coreSharedTerminalPreparedDraft{}, false, err
	}
	for index, step := range raw.Steps {
		if err := validateCoreSharedTerminalEvidenceBindings(
			value.draft.steps[index].input,
			step.Evidence,
			facts.byID,
		); err != nil {
			return coreSharedTerminalPreparedDraft{}, false, err
		}
	}
	if !coreEvidenceDraftHasPositiveTopics(raw) {
		return coreSharedTerminalPreparedDraft{}, false, nil
	}
	mapping, err := profileevidence.NewMapping(profileevidence.MappingValues{
		ProfileID: profileID,
		Facts:     facts.values,
		Drafts:    []profileevidence.DraftValues{raw},
	})
	if err != nil {
		return coreSharedTerminalPreparedDraft{}, false, err
	}
	cluster, eligible, err := mapping.ClusterKey(reference.draftID)
	if err != nil || !eligible {
		return coreSharedTerminalPreparedDraft{}, false, err
	}
	evidence, err := coreSharedTerminalEvidenceCodes(mapping, reference)
	if err != nil {
		return coreSharedTerminalPreparedDraft{}, false, err
	}
	score, err := p.metadata.Score().Score(evidence)
	if err != nil {
		return coreSharedTerminalPreparedDraft{}, false, err
	}
	confidence, err := p.metadata.Score().ConfidenceFor(score)
	if err != nil {
		return coreSharedTerminalPreparedDraft{}, false, err
	}
	return coreSharedTerminalPreparedDraft{
		value:      cloneCoreSharedTerminalDraft(value),
		cluster:    cluster.Canonical(),
		evidence:   slices.Clone(evidence),
		score:      score,
		confidence: confidence,
	}, true, nil
}

func validateCoreSharedTerminalEvidenceBindings(
	input legalquery.LogicalInput,
	values []profileevidence.EvidenceValues,
	facts map[string]coreEvidenceFact,
) error {
	if input == nil || input.InputKind() != legalquery.InputKindLawContentSearch {
		return fmt.Errorf("共有末尾 step は law_content_search でなければなりません")
	}
	for _, value := range values {
		fact, exists := facts[value.FactID]
		if !exists {
			return fmt.Errorf("共有末尾 fact %q が登録されていません", value.FactID)
		}
		allowed := coreEvidenceBindingAllowed(input.InputKind(), value, fact)
		if !allowed &&
			value.Layer == profileevidence.LayerTargetAnchor &&
			value.Code == legalquery.EvidenceOfficialAlias &&
			fact.kind == coreEvidenceFactLawName &&
			fact.matchKind != legalquery.PreprocessMatchUniqueTypoCorrection {
			allowed = true
		}
		if !allowed {
			return fmt.Errorf(
				"共有末尾 fact %q の %s/%s は使用できません",
				value.FactID,
				value.Layer,
				value.Code,
			)
		}
	}
	return nil
}

func validateCoreSharedTerminalFactReuse(
	value profileevidence.DraftValues,
) error {
	uses := make(map[string]int)
	explicit := make(map[string]bool)
	for stepIndex, step := range value.Steps {
		seenInStep := make(map[string]struct{})
		for _, evidence := range step.Evidence {
			if _, seen := seenInStep[evidence.FactID]; seen {
				continue
			}
			seenInStep[evidence.FactID] = struct{}{}
			if previous, exists := uses[evidence.FactID]; exists &&
				previous != stepIndex &&
				(!explicit[evidence.FactID] ||
					evidence.Layer != profileevidence.LayerExplicitTaskResource) {
				return fmt.Errorf(
					"共有末尾の非明示 fact %q を複数 step へ共有できません",
					evidence.FactID,
				)
			}
			if _, exists := uses[evidence.FactID]; !exists {
				uses[evidence.FactID] = stepIndex
				explicit[evidence.FactID] =
					evidence.Layer == profileevidence.LayerExplicitTaskResource
			}
		}
	}
	return nil
}

func coreSharedTerminalEvidenceCodes(
	mapping profileevidence.Mapping,
	reference coreEvidenceDraftRef,
) ([]legalquery.EvidenceCode, error) {
	present := make(map[legalquery.EvidenceCode]struct{})
	for _, stepID := range reference.stepIDs {
		values, err := mapping.StepEvidence(reference.draftID, stepID)
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			present[value.Code()] = struct{}{}
		}
	}
	result := make([]legalquery.EvidenceCode, 0, len(present))
	for _, code := range evidenceOrder {
		if _, exists := present[code]; exists {
			result = append(result, code)
		}
	}
	return result, nil
}

func mergeCoreSharedTerminalEquivalentDrafts(
	values []coreSharedTerminalPreparedDraft,
) ([]coreSharedTerminalDraft, error) {
	result := make([]coreSharedTerminalDraft, 0, len(values))
	byKey := make(map[string]int, len(values))
	for _, value := range values {
		key := value.cluster + "\x1f" + value.value.meaningSignature
		if index, exists := byKey[key]; exists {
			if err := mergeCoreSharedTerminalDraft(
				&result[index],
				value.value,
			); err != nil {
				return nil, err
			}
			continue
		}
		byKey[key] = len(result)
		result = append(result, cloneCoreSharedTerminalDraft(value.value))
	}
	return result, nil
}

func mergeCoreSharedTerminalDraft(
	target *coreSharedTerminalDraft,
	source coreSharedTerminalDraft,
) error {
	if target.meaningSignature != source.meaningSignature ||
		len(target.draft.steps) != len(source.draft.steps) {
		return fmt.Errorf("共有末尾の同値 draft の意味が一致しません")
	}
	for index := range target.draft.steps {
		left, err := logicalInputSignature(target.draft.steps[index].input)
		if err != nil {
			return err
		}
		right, err := logicalInputSignature(source.draft.steps[index].input)
		if err != nil {
			return err
		}
		if left != right || target.draft.steps[index].topicOrdinal !=
			source.draft.steps[index].topicOrdinal {
			return fmt.Errorf("共有末尾の同値 draft の step が一致しません")
		}
		target.draft.steps[index].evidenceBindings = dedupeCoreEvidenceValues(append(
			target.draft.steps[index].evidenceBindings,
			source.draft.steps[index].evidenceBindings...,
		))
		target.draft.steps[index].startByte = min(
			target.draft.steps[index].startByte,
			source.draft.steps[index].startByte,
		)
	}
	concepts, err := mergeCoreSharedConceptSources(
		target.draft.concepts,
		source.draft.concepts,
	)
	if err != nil {
		return err
	}
	target.draft.concepts = concepts
	target.draft.requiredPacks = append(
		target.draft.requiredPacks,
		source.draft.requiredPacks...,
	)
	slices.Sort(target.draft.requiredPacks)
	target.draft.requiredPacks = slices.Compact(target.draft.requiredPacks)
	for code := range source.draft.evidence {
		target.draft.evidence[code] = struct{}{}
	}
	return nil
}

func mergeCoreSharedConceptSources(
	left []legalquery.LegalConceptSource,
	right []legalquery.LegalConceptSource,
) ([]legalquery.LegalConceptSource, error) {
	byID := make(map[string]legalquery.LegalConceptSource, len(left)+len(right))
	for _, value := range append(slices.Clone(left), right...) {
		if previous, exists := byID[value.ConceptID()]; exists && previous != value {
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

func sortCoreSharedTerminalPrepared(values []coreSharedTerminalPreparedDraft) {
	sort.Slice(values, func(left, right int) bool {
		return compareCoreSharedTerminalPrepared(values[left], values[right]) < 0
	})
}

func compareCoreSharedTerminalPrepared(
	left coreSharedTerminalPreparedDraft,
	right coreSharedTerminalPreparedDraft,
) int {
	if left.score != right.score {
		return right.score - left.score
	}
	if leftEvidence, rightEvidence :=
		evidenceSignature(left.evidence),
		evidenceSignature(right.evidence); leftEvidence != rightEvidence {
		return strings.Compare(leftEvidence, rightEvidence)
	}
	if len(left.value.draft.steps) != len(right.value.draft.steps) {
		return len(left.value.draft.steps) - len(right.value.draft.steps)
	}
	if left.value.meaningSignature != right.value.meaningSignature {
		return strings.Compare(
			left.value.meaningSignature,
			right.value.meaningSignature,
		)
	}
	if position := sourcePosition(left.value.draft) -
		sourcePosition(right.value.draft); position != 0 {
		return position
	}
	return strings.Compare(left.cluster, right.cluster)
}

func cloneCoreSharedTerminalDraft(
	value coreSharedTerminalDraft,
) coreSharedTerminalDraft {
	return coreSharedTerminalDraft{
		draft:            cloneDraft(value.draft),
		meaningSignature: value.meaningSignature,
	}
}

func cloneCoreSharedTerminalPrepared(
	value coreSharedTerminalPreparedDraft,
) coreSharedTerminalPreparedDraft {
	return coreSharedTerminalPreparedDraft{
		value:      cloneCoreSharedTerminalDraft(value.value),
		cluster:    value.cluster,
		evidence:   slices.Clone(value.evidence),
		score:      value.score,
		confidence: value.confidence,
	}
}
