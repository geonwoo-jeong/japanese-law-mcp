package profileevidence

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

var localIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// NewMapping は、profile が明示した事実と step 対応を複製して検証する。
func NewMapping(values MappingValues) (Mapping, error) {
	if err := validateLocalID("profileId", values.ProfileID); err != nil {
		return Mapping{}, err
	}
	facts, err := buildFacts(values.Facts)
	if err != nil {
		return Mapping{}, err
	}
	if len(values.Drafts) > maximumDrafts {
		return Mapping{}, fmt.Errorf(
			"drafts は %d 件以下でなければなりません",
			maximumDrafts,
		)
	}

	drafts := make(map[string]draft, len(values.Drafts))
	for index, value := range values.Drafts {
		if _, exists := drafts[value.DraftID]; exists {
			return Mapping{}, fmt.Errorf("drafts[%d].draftId が重複しています", index)
		}
		built, buildErr := buildDraft(value, facts)
		if buildErr != nil {
			return Mapping{}, fmt.Errorf("drafts[%d]: %w", index, buildErr)
		}
		drafts[built.draftID] = built
	}
	return Mapping{
		profileID: values.ProfileID,
		facts:     facts,
		drafts:    drafts,
	}, nil
}

// StepEvidence は、指定した step の決定的に並んだ根拠の複製を返す。
func (m Mapping) StepEvidence(draftID string, stepID string) ([]Evidence, error) {
	value, exists := m.drafts[draftID]
	if !exists {
		return nil, fmt.Errorf("draftId %q は mapping に存在しません", draftID)
	}
	index, exists := value.stepByID[stepID]
	if !exists {
		return nil, fmt.Errorf("stepId %q は draft %q に存在しません", stepID, draftID)
	}
	return append([]Evidence(nil), value.steps[index].evidence...), nil
}

// NormalizedStepEvidence は、同じ step・同じ group だけで閉じた優越表を
// 適用した根拠の複製を返す。
func (m Mapping) NormalizedStepEvidence(
	draftID string,
	stepID string,
) ([]Evidence, error) {
	value, exists := m.drafts[draftID]
	if !exists {
		return nil, fmt.Errorf("draftId %q は mapping に存在しません", draftID)
	}
	index, exists := value.stepByID[stepID]
	if !exists {
		return nil, fmt.Errorf("stepId %q は draft %q に存在しません", stepID, draftID)
	}
	return append([]Evidence(nil), value.steps[index].normalizedEvidence...), nil
}

func buildFacts(values []FactValues) (map[string]fact, error) {
	if len(values) > maximumFacts {
		return nil, fmt.Errorf("facts は %d 件以下でなければなりません", maximumFacts)
	}
	result := make(map[string]fact, len(values))
	for index, value := range values {
		if err := validateLocalID("factId", value.FactID); err != nil {
			return nil, fmt.Errorf("facts[%d]: %w", index, err)
		}
		if _, exists := result[value.FactID]; exists {
			return nil, fmt.Errorf("facts[%d].factId が重複しています", index)
		}
		var span *legalquery.QuerySpan
		if value.Span != nil {
			if err := value.Span.Validate(); err != nil {
				return nil, fmt.Errorf("facts[%d].span: %w", index, err)
			}
			cloned := *value.Span
			span = &cloned
		}
		result[value.FactID] = fact{
			factID: value.FactID,
			span:   span,
		}
	}
	return result, nil
}

func buildDraft(value DraftValues, facts map[string]fact) (draft, error) {
	if err := validateLocalID("draftId", value.DraftID); err != nil {
		return draft{}, err
	}
	if len(value.Steps) < 1 || len(value.Steps) > legalquery.MaxCapabilityCalls {
		return draft{}, fmt.Errorf(
			"steps は 1 件以上 %d 件以下でなければなりません",
			legalquery.MaxCapabilityCalls,
		)
	}

	steps := make([]step, 0, len(value.Steps))
	stepIDs := make(map[string]struct{}, len(value.Steps))
	sourceOrdinals := make(map[int]struct{}, len(value.Steps))
	for index, stepValue := range value.Steps {
		if err := validateLocalID("stepId", stepValue.StepID); err != nil {
			return draft{}, fmt.Errorf("steps[%d]: %w", index, err)
		}
		if _, exists := stepIDs[stepValue.StepID]; exists {
			return draft{}, fmt.Errorf("steps[%d].stepId が重複しています", index)
		}
		if stepValue.SourceOrdinal < 1 ||
			stepValue.SourceOrdinal > len(value.Steps) {
			return draft{}, fmt.Errorf(
				"steps[%d].sourceOrdinal は 1 から step 件数まででなければなりません",
				index,
			)
		}
		if _, exists := sourceOrdinals[stepValue.SourceOrdinal]; exists {
			return draft{}, fmt.Errorf("steps[%d].sourceOrdinal が重複しています", index)
		}
		if stepValue.TopicOrdinal < 1 {
			return draft{}, fmt.Errorf("steps[%d].topicOrdinal は 1 以上でなければなりません", index)
		}
		if len(stepValue.StepMeaningSignature) < 1 ||
			len(stepValue.StepMeaningSignature) > maximumStepMeaningSignatureBytes {
			return draft{}, fmt.Errorf(
				"steps[%d].stepMeaningSignature は 1 byte 以上 %d byte 以下でなければなりません",
				index,
				maximumStepMeaningSignatureBytes,
			)
		}
		evidence, err := buildEvidence(stepValue.Evidence, facts)
		if err != nil {
			return draft{}, fmt.Errorf("steps[%d].evidence: %w", index, err)
		}
		stepIDs[stepValue.StepID] = struct{}{}
		sourceOrdinals[stepValue.SourceOrdinal] = struct{}{}
		steps = append(steps, step{
			stepID:               stepValue.StepID,
			sourceOrdinal:        stepValue.SourceOrdinal,
			topicOrdinal:         stepValue.TopicOrdinal,
			stepMeaningSignature: stepValue.StepMeaningSignature,
			evidence:             evidence,
			normalizedEvidence:   normalizeStepEvidence(evidence),
		})
	}
	slices.SortFunc(steps, func(left step, right step) int {
		return cmp.Compare(left.sourceOrdinal, right.sourceOrdinal)
	})
	if err := validateTopicOrdinals(steps); err != nil {
		return draft{}, err
	}

	stepByID := make(map[string]int, len(steps))
	for index, value := range steps {
		stepByID[value.stepID] = index
	}
	return draft{
		draftID:  value.DraftID,
		steps:    steps,
		stepByID: stepByID,
	}, nil
}

func buildEvidence(values []EvidenceValues, facts map[string]fact) ([]Evidence, error) {
	if len(values) < 1 || len(values) > maximumStepEvidence {
		return nil, fmt.Errorf(
			"一 step の evidence は 1 件以上 %d 件以下でなければなりません",
			maximumStepEvidence,
		)
	}
	result := make([]Evidence, 0, len(values))
	seen := make(map[string]Evidence, len(values))
	factGroups := make(map[string]string, len(values))
	groupFacts := make(map[string]map[string]struct{})
	for index, value := range values {
		registered, exists := facts[value.FactID]
		if !exists {
			return nil, fmt.Errorf("evidence[%d].factId が facts に存在しません", index)
		}
		if err := validateEvidenceLayer(value.Layer, value.Code, registered.span); err != nil {
			return nil, fmt.Errorf("evidence[%d]: %w", index, err)
		}
		if value.ClusterSpan &&
			(registered.span == nil || !clusterSpanLayer(value.Layer)) {
			return nil, fmt.Errorf(
				"evidence[%d] は cluster の evidenceSpan に利用できません",
				index,
			)
		}
		if value.NormalizationGroup != "" {
			if err := validateLocalID(
				"normalizationGroup",
				value.NormalizationGroup,
			); err != nil {
				return nil, fmt.Errorf("evidence[%d]: %w", index, err)
			}
		}
		if previous, exists := factGroups[value.FactID]; exists &&
			previous != value.NormalizationGroup {
			return nil, fmt.Errorf(
				"evidence[%d] は同じ fact を複数の normalizationGroup へ対応させています",
				index,
			)
		}
		factGroups[value.FactID] = value.NormalizationGroup
		if value.NormalizationGroup != "" {
			if groupFacts[value.NormalizationGroup] == nil {
				groupFacts[value.NormalizationGroup] = make(map[string]struct{})
			}
			groupFacts[value.NormalizationGroup][value.FactID] = struct{}{}
		}
		built := Evidence{
			factID:              registered.factID,
			layer:               value.Layer,
			code:                value.Code,
			span:                cloneSpan(registered.span),
			independentPositive: value.IndependentPositive,
			clusterSpan:         value.ClusterSpan,
			normalizationGroup:  value.NormalizationGroup,
		}
		key := value.FactID + "\x00" + string(value.Code)
		if previous, duplicate := seen[key]; duplicate {
			if !sameEvidence(previous, built) {
				return nil, fmt.Errorf(
					"evidence[%d] は同じ fact と code に競合する対応を持ちます",
					index,
				)
			}
			continue
		}
		seen[key] = built
		result = append(result, built)
	}
	for group, factIDs := range groupFacts {
		if len(factIDs) < 2 {
			return nil, fmt.Errorf(
				"normalizationGroup %q は二件以上の fact を必要とします",
				group,
			)
		}
	}
	if err := validateEvidenceCombinations(result); err != nil {
		return nil, err
	}
	slices.SortFunc(result, compareEvidence)
	return result, nil
}

func validateEvidenceCombinations(values []Evidence) error {
	byFactID := make(map[string][]Evidence, len(values))
	for _, value := range values {
		byFactID[value.factID] = append(byFactID[value.factID], value)
	}
	for factID, group := range byFactID {
		if err := validateFactEvidenceCombination(group); err != nil {
			return fmt.Errorf("factId %q: %w", factID, err)
		}
	}
	return nil
}

func validateFactEvidenceCombination(values []Evidence) error {
	if len(values) == 1 {
		if values[0].code == legalquery.EvidenceUniqueTypoCorrection {
			return fmt.Errorf("unique_typo_correction は基本根拠と併記しなければなりません")
		}
		return nil
	}
	if len(values) != 2 {
		return fmt.Errorf("一つの fact に許可されていない複数の code があります")
	}

	var correction Evidence
	var base Evidence
	switch {
	case values[0].code == legalquery.EvidenceUniqueTypoCorrection:
		correction, base = values[0], values[1]
	case values[1].code == legalquery.EvidenceUniqueTypoCorrection:
		correction, base = values[1], values[0]
	default:
		return fmt.Errorf("一つの fact に許可されていない複数の code があります")
	}
	if correction.independentPositive || correction.clusterSpan {
		return fmt.Errorf(
			"unique_typo_correction は正の根拠または cluster span にできません",
		)
	}
	if correction.layer != base.layer ||
		!uniqueTypoBaseCode(correction.layer, base.code) {
		return fmt.Errorf("unique_typo_correction の基本根拠が一致しません")
	}
	return nil
}

func validateEvidenceLayer(
	layer Layer,
	code legalquery.EvidenceCode,
	span *legalquery.QuerySpan,
) error {
	if _, exists := code.Order(); !exists {
		return fmt.Errorf("code が定義されていません")
	}
	switch layer {
	case LayerBoundary:
		if code != legalquery.EvidenceOfficialIdentifier || span != nil {
			return fmt.Errorf("boundary は span のない official_identifier だけを持てます")
		}
	case LayerExplicitTaskResource:
		if span == nil ||
			(code != legalquery.EvidenceExplicitTask &&
				code != legalquery.EvidenceExplicitResource) {
			return fmt.Errorf("explicit_task_resource の fact と code が一致しません")
		}
	case LayerTargetAnchor:
		if span == nil || !targetAnchorCode(code) {
			return fmt.Errorf("target_anchor の fact と code が一致しません")
		}
	case LayerSemanticExpansion:
		if span == nil || !semanticExpansionCode(code) {
			return fmt.Errorf("semantic_expansion の fact と code が一致しません")
		}
	case LayerClarificationOrReject:
		return fmt.Errorf("clarification_or_reject は evidence mapping に記録できません")
	default:
		return fmt.Errorf("layer が定義されていません")
	}
	return nil
}

func validateTopicOrdinals(steps []step) error {
	if steps[0].topicOrdinal != 1 {
		return fmt.Errorf("原文先頭の step は topicOrdinal=1 でなければなりません")
	}
	previous := 1
	for index := 1; index < len(steps); index++ {
		current := steps[index].topicOrdinal
		if current != previous && current != previous+1 {
			return fmt.Errorf(
				"原文順の topicOrdinal は同じ値または一つ大きい値でなければなりません",
			)
		}
		previous = current
	}
	return nil
}

func validateLocalID(field string, value string) error {
	if len(value) < 1 || len(value) > maximumLocalIDBytes {
		return fmt.Errorf("%s は 1 byte 以上 %d byte 以下でなければなりません", field, maximumLocalIDBytes)
	}
	if !localIDPattern.MatchString(value) {
		return fmt.Errorf("%s は小文字英数字の segment を - で連結しなければなりません", field)
	}
	return nil
}

func targetAnchorCode(code legalquery.EvidenceCode) bool {
	return code == legalquery.EvidenceOfficialIdentifier ||
		code == legalquery.EvidenceStructuredReference ||
		code == legalquery.EvidenceOfficialAlias ||
		code == legalquery.EvidenceUniqueTypoCorrection ||
		code == legalquery.EvidenceGeneralTerm
}

func semanticExpansionCode(code legalquery.EvidenceCode) bool {
	return code == legalquery.EvidenceLegalConcept ||
		code == legalquery.EvidenceMorphologicalContext ||
		code == legalquery.EvidenceUniqueTypoCorrection ||
		code == legalquery.EvidenceGeneralTerm
}

func uniqueTypoBaseCode(layer Layer, code legalquery.EvidenceCode) bool {
	return (layer == LayerTargetAnchor &&
		code == legalquery.EvidenceOfficialAlias) ||
		(layer == LayerSemanticExpansion &&
			code == legalquery.EvidenceLegalConcept)
}

func clusterSpanLayer(layer Layer) bool {
	return layer == LayerExplicitTaskResource ||
		layer == LayerTargetAnchor ||
		layer == LayerSemanticExpansion
}

func cloneSpan(value *legalquery.QuerySpan) *legalquery.QuerySpan {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func sameEvidence(left Evidence, right Evidence) bool {
	leftSpan, leftExists := left.Span()
	rightSpan, rightExists := right.Span()
	return left.factID == right.factID &&
		left.layer == right.layer &&
		left.code == right.code &&
		leftExists == rightExists &&
		(!leftExists || leftSpan == rightSpan) &&
		left.independentPositive == right.independentPositive &&
		left.clusterSpan == right.clusterSpan &&
		left.normalizationGroup == right.normalizationGroup
}

func compareEvidence(left Evidence, right Evidence) int {
	leftLayer, _ := layerOrder(left.layer)
	rightLayer, _ := layerOrder(right.layer)
	if order := cmp.Compare(leftLayer, rightLayer); order != 0 {
		return order
	}
	if order := compareOptionalSpan(left.span, right.span); order != 0 {
		return order
	}
	leftCode, _ := left.code.Order()
	rightCode, _ := right.code.Order()
	if order := cmp.Compare(leftCode, rightCode); order != 0 {
		return order
	}
	return cmp.Compare(left.factID, right.factID)
}

func compareOptionalSpan(
	left *legalquery.QuerySpan,
	right *legalquery.QuerySpan,
) int {
	switch {
	case left == nil && right == nil:
		return 0
	case left == nil:
		return -1
	case right == nil:
		return 1
	case left.StartByte() != right.StartByte():
		return cmp.Compare(left.StartByte(), right.StartByte())
	default:
		return cmp.Compare(left.EndByte(), right.EndByte())
	}
}

func layerOrder(value Layer) (int, bool) {
	switch value {
	case LayerBoundary:
		return 0, true
	case LayerExplicitTaskResource:
		return 1, true
	case LayerTargetAnchor:
		return 2, true
	case LayerSemanticExpansion:
		return 3, true
	case LayerClarificationOrReject:
		return 4, true
	default:
		return 0, false
	}
}
