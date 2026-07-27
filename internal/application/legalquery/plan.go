package legalquery

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

const (
	legalQueryLanguage        = "ja"
	maximumProfileVersionSize = 128
)

// PlanDecision は、統合照会を実行するかどうかと実行方法を表す。
type PlanDecision string

const (
	// PlanDecisionSingle は、一つの意味候補を実行する決定を表す。
	PlanDecisionSingle PlanDecision = "single"
	// PlanDecisionHedged は、独立した二つの意味候補を実行する決定を表す。
	PlanDecisionHedged PlanDecision = "hedged"
	// PlanDecisionNeedsClarification は、外部呼出しを行わず明確化を求める決定を表す。
	PlanDecisionNeedsClarification PlanDecision = "needs_clarification"
	// PlanDecisionCapabilityUnavailable は、必要な拡張パックが無効な決定を表す。
	PlanDecisionCapabilityUnavailable PlanDecision = "capability_unavailable"
	// PlanDecisionUnsupported は、採用範囲外として実行しない決定を表す。
	PlanDecisionUnsupported PlanDecision = "unsupported"
)

// SelectionAvailability は、意味候補に必要な拡張パックの利用可否を表す。
type SelectionAvailability string

const (
	// SelectionAvailabilityAvailable は、必要な全 pack が有効であることを表す。
	SelectionAvailabilityAvailable SelectionAvailability = "available"
	// SelectionAvailabilityPackDisabled は、必要な pack の一つ以上が無効であることを表す。
	SelectionAvailabilityPackDisabled SelectionAvailability = "pack_disabled"
)

// ReasonCode は、候補選択または非実行の決定的な理由を表す。
type ReasonCode string

const (
	// ReasonCodeSingleClearCandidate は、単独候補の score と margin が基準を満たしたことを表す。
	ReasonCodeSingleClearCandidate ReasonCode = "single_clear_candidate"
	// ReasonCodeHedgedCloseCandidates は、上位二候補が hedge 条件を満たしたことを表す。
	ReasonCodeHedgedCloseCandidates ReasonCode = "hedged_close_candidates"
	// ReasonCodeBelowExecutionThreshold は、候補が最低実行閾値を満たさないことを表す。
	ReasonCodeBelowExecutionThreshold ReasonCode = "below_execution_threshold"
	// ReasonCodeAmbiguousCandidates は、安全に候補を絞れないことを表す。
	ReasonCodeAmbiguousCandidates ReasonCode = "ambiguous_candidates"
	// ReasonCodeRequiredPackDisabled は、必要な採用済み pack が無効であることを表す。
	ReasonCodeRequiredPackDisabled ReasonCode = "required_pack_disabled"
	// ReasonCodeNonJapaneseQuery は、日本語入力境界を満たさないことを表す。
	ReasonCodeNonJapaneseQuery ReasonCode = "non_japanese_query"
	// ReasonCodeMixedUnsupportedIntent は、取得意図と対象外意図が混在することを表す。
	ReasonCodeMixedUnsupportedIntent ReasonCode = "mixed_unsupported_intent"
	// ReasonCodeUnsupportedTaskOrResource は、task または resource が採用範囲外であることを表す。
	ReasonCodeUnsupportedTaskOrResource ReasonCode = "unsupported_task_or_resource"
)

// LegalQueryPlanSelectionValues は、plan selection の作成に必要な値を保持する。
type LegalQueryPlanSelectionValues struct {
	CandidateID   string
	Availability  SelectionAvailability
	RequiredPacks []string
}

// LegalQueryPlanSelection は、順位付け済み候補の利用可否を不変に保持する。
type LegalQueryPlanSelection struct {
	candidateID   string
	availability  SelectionAvailability
	requiredPacks []string
}

// NewLegalQueryPlanSelection は、候補参照と pack 可否を複製して返す。
func NewLegalQueryPlanSelection(
	values LegalQueryPlanSelectionValues,
) (LegalQueryPlanSelection, error) {
	selection := LegalQueryPlanSelection{
		candidateID:   values.CandidateID,
		availability:  values.Availability,
		requiredPacks: append([]string{}, values.RequiredPacks...),
	}
	if err := selection.Validate(); err != nil {
		return LegalQueryPlanSelection{}, err
	}
	return selection, nil
}

// CandidateID は、順位付け済み候補の識別子を返す。
func (s LegalQueryPlanSelection) CandidateID() string {
	return s.candidateID
}

// Availability は、候補に必要な pack の利用可否を返す。
func (s LegalQueryPlanSelection) Availability() SelectionAvailability {
	return s.availability
}

// RequiredPacks は、候補に必要な pack ID の複製を返す。
func (s LegalQueryPlanSelection) RequiredPacks() []string {
	return append([]string{}, s.requiredPacks...)
}

// Validate は、候補参照、availability および pack ID を検証する。
func (s LegalQueryPlanSelection) Validate() error {
	if err := validateQueryPlanID("candidateId", s.candidateID); err != nil {
		return err
	}
	if s.availability != SelectionAvailabilityAvailable &&
		s.availability != SelectionAvailabilityPackDisabled {
		return fmt.Errorf("availability は available または pack_disabled でなければなりません")
	}
	if err := validateRequiredPacks(s.requiredPacks); err != nil {
		return err
	}
	if s.availability == SelectionAvailabilityPackDisabled &&
		len(s.requiredPacks) == 0 {
		return fmt.Errorf("pack_disabled の selection には requiredPacks が一件以上必要です")
	}
	return nil
}

// UnmarshalJSON は、selector を介さない直接復元を拒否する。
func (*LegalQueryPlanSelection) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryPlanSelection は JSON から直接復元できません。NewLegalQueryPlanSelection を使用してください",
	)
}

func (s LegalQueryPlanSelection) clone() LegalQueryPlanSelection {
	return LegalQueryPlanSelection{
		candidateID:   s.candidateID,
		availability:  s.availability,
		requiredPacks: append([]string{}, s.requiredPacks...),
	}
}

// LegalQueryPlanValues は、統合照会 plan の作成に必要な値を保持する。
type LegalQueryPlanValues struct {
	ProfileVersion   string
	Decision         PlanDecision
	RankedCandidates []LegalQueryCandidate
	Selected         []LegalQueryPlanSelection
	ReasonCodes      []ReasonCode
	LimitPerAttempt  int
}

// LegalQueryPlan は、順位、選択、可用性および固定済み予算を不変に保持する。
type LegalQueryPlan struct {
	language         string
	profileVersion   string
	decision         PlanDecision
	rankedCandidates []LegalQueryCandidate
	selected         []LegalQueryPlanSelection
	reasonCodes      []ReasonCode
	budget           LegalQueryBudget
}

// NewLegalQueryPlan は、候補と選択を複製し、実行前の固定予算まで確定する。
func NewLegalQueryPlan(
	values LegalQueryPlanValues,
) (LegalQueryPlan, error) {
	rankedCandidates, err := cloneLegalQueryCandidates(values.RankedCandidates)
	if err != nil {
		return LegalQueryPlan{}, err
	}
	plan := LegalQueryPlan{
		language:         legalQueryLanguage,
		profileVersion:   values.ProfileVersion,
		decision:         values.Decision,
		rankedCandidates: rankedCandidates,
		selected:         clonePlanSelections(values.Selected),
		reasonCodes:      append([]ReasonCode{}, values.ReasonCodes...),
	}
	if err := validatePlanDefinition(plan); err != nil {
		return LegalQueryPlan{}, err
	}
	budget, err := newLegalQueryBudget(
		plan.decision,
		values.LimitPerAttempt,
		plan.rankedCandidates,
		plan.selected,
	)
	if err != nil {
		return LegalQueryPlan{}, err
	}
	plan.budget = budget
	if err := plan.Validate(); err != nil {
		return LegalQueryPlan{}, err
	}
	return plan, nil
}

// Language は、固定の照会言語を返す。
func (p LegalQueryPlan) Language() string {
	return p.language
}

// ProfileVersion は、候補を評価した不透明な profile version を返す。
func (p LegalQueryPlan) ProfileVersion() string {
	return p.profileVersion
}

// Decision は、選択済みの実行または非実行方法を返す。
func (p LegalQueryPlan) Decision() PlanDecision {
	return p.decision
}

// RankedCandidates は、意味順位を保持した候補の複製を返す。
func (p LegalQueryPlan) RankedCandidates() []LegalQueryCandidate {
	candidates, err := cloneLegalQueryCandidates(p.rankedCandidates)
	if err != nil {
		panic(fmt.Sprintf("検証済み plan candidate の複製に失敗しました: %v", err))
	}
	return candidates
}

// Selected は、順位を保持した selection の複製を返す。
func (p LegalQueryPlan) Selected() []LegalQueryPlanSelection {
	return clonePlanSelections(p.selected)
}

// ReasonCodes は、決定理由の複製を返す。
func (p LegalQueryPlan) ReasonCodes() []ReasonCode {
	return append([]ReasonCode{}, p.reasonCodes...)
}

// Budget は、実行前に確定した固定予算の複製を返す。
func (p LegalQueryPlan) Budget() LegalQueryBudget {
	return p.budget.clone()
}

// Validate は、plan の順位、選択、理由および予算の整合を確認する。
func (p LegalQueryPlan) Validate() error {
	if err := validatePlanDefinition(p); err != nil {
		return err
	}
	if err := p.budget.Validate(); err != nil {
		return fmt.Errorf("budget が有効ではありません: %w", err)
	}
	expected, err := newLegalQueryBudget(
		p.decision,
		p.budget.limitPerAttempt,
		p.rankedCandidates,
		p.selected,
	)
	if err != nil {
		return err
	}
	if !p.budget.equal(expected) {
		return fmt.Errorf("budget が selected から決定した固定値と一致しません")
	}
	return nil
}

// UnmarshalJSON は、planner と selector を介さない直接復元を拒否する。
func (*LegalQueryPlan) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryPlan は JSON から直接復元できません。NewLegalQueryPlan を使用してください",
	)
}

func clonePlanSelections(
	values []LegalQueryPlanSelection,
) []LegalQueryPlanSelection {
	cloned := make([]LegalQueryPlanSelection, 0, len(values))
	for _, value := range values {
		cloned = append(cloned, value.clone())
	}
	return cloned
}

func cloneLegalQueryCandidates(
	values []LegalQueryCandidate,
) ([]LegalQueryCandidate, error) {
	cloned := make([]LegalQueryCandidate, 0, len(values))
	for index, value := range values {
		candidate, err := NewLegalQueryCandidate(LegalQueryCandidateValues{
			CandidateID:    value.CandidateID(),
			SemanticScore:  value.SemanticScore(),
			Confidence:     value.Confidence(),
			EvidenceCodes:  value.EvidenceCodes(),
			ConceptSources: value.ConceptSources(),
			RequiredPacks:  value.RequiredPacks(),
			Steps:          value.Steps(),
		})
		if err != nil {
			return nil, fmt.Errorf(
				"rankedCandidates[%d] が有効ではありません: %w",
				index,
				err,
			)
		}
		cloned = append(cloned, candidate)
	}
	return cloned, nil
}

func validateProfileVersion(value string) error {
	switch {
	case !utf8.ValidString(value):
		return fmt.Errorf("profileVersion は有効な UTF-8 でなければなりません")
	case len(value) < 1 || len(value) > maximumProfileVersionSize:
		return fmt.Errorf("profileVersion は 1 byte 以上 128 byte 以下でなければなりません")
	}
	for index, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf("profileVersion に Unicode control character を含めることはできません")
		}
		if (index == 0 || index+utf8.RuneLen(character) == len(value)) &&
			unicode.IsSpace(character) {
			return fmt.Errorf("profileVersion の先頭と末尾に Unicode White_Space を置けません")
		}
	}
	return nil
}
