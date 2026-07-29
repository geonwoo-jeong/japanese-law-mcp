package legalquery

import (
	"encoding/json"
	"fmt"
)

// LegalQueryQuestion は、利用者へ返す決定的な明確化質問を表す。
type LegalQueryQuestion string

const (
	// LegalQueryQuestionTask は、実行する作業の指定を求める。
	LegalQueryQuestionTask LegalQueryQuestion = "検索、読取りまたは更新一覧のどれを行うか指定してください。"
	// LegalQueryQuestionResource は、対象資源の指定を求める。
	LegalQueryQuestionResource LegalQueryQuestion = "法令、条文または裁判例のどれを対象にするか指定してください。"
	// LegalQueryQuestionLaw は、法令または条文の識別情報を求める。
	LegalQueryQuestionLaw LegalQueryQuestion = "対象の法令名、法令 ID または条番号を指定してください。"
	// LegalQueryQuestionJudicialDecision は、裁判例の検索語または参照を求める。
	LegalQueryQuestionJudicialDecision LegalQueryQuestion = "対象の裁判例を検索する語または裁判例の ref を指定してください。"
	// LegalQueryQuestionStepLimitExceeded は、明示する取得項目を四件以下へ分割するよう求める。
	LegalQueryQuestionStepLimitExceeded LegalQueryQuestion = "一度に取得する項目を4件以下に分けて指定してください。"
)

// LegalQueryStepSummary は、logical input を除いた公開用 step を表す。
type LegalQueryStepSummary struct {
	stepID                 string
	task                   Task
	resource               Resource
	capabilityID           string
	capabilityMajorVersion int
	inputKind              LogicalInputKind
}

// NewLegalQueryStepSummary は、検証済み candidate step の公開項目だけを返す。
func NewLegalQueryStepSummary(
	step LegalQueryCandidateStep,
) (LegalQueryStepSummary, error) {
	if err := step.Validate(); err != nil {
		return LegalQueryStepSummary{}, fmt.Errorf("step が有効ではありません: %w", err)
	}
	summary := LegalQueryStepSummary{
		stepID:                 step.StepID(),
		task:                   step.Task(),
		resource:               step.Resource(),
		capabilityID:           step.CapabilityID(),
		capabilityMajorVersion: step.CapabilityMajorVersion(),
		inputKind:              step.InputKind(),
	}
	if err := summary.Validate(); err != nil {
		return LegalQueryStepSummary{}, err
	}
	return summary, nil
}

// StepID は、plan の step 識別子を返す。
func (s LegalQueryStepSummary) StepID() string {
	return s.stepID
}

// Task は、step が行う作業を返す。
func (s LegalQueryStepSummary) Task() Task {
	return s.task
}

// Resource は、step が対象にする資源を返す。
func (s LegalQueryStepSummary) Resource() Resource {
	return s.resource
}

// CapabilityID は、step が使用する能力 ID を返す。
func (s LegalQueryStepSummary) CapabilityID() string {
	return s.capabilityID
}

// CapabilityMajorVersion は、能力のメジャーバージョンを返す。
func (s LegalQueryStepSummary) CapabilityMajorVersion() int {
	return s.capabilityMajorVersion
}

// Validate は、公開 step と採用済み capability の対応を確認する。
func (s LegalQueryStepSummary) Validate() error {
	if err := validateQueryPlanID("stepId", s.stepID); err != nil {
		return err
	}
	specification, exists := stepSpecificationFor(s.inputKind)
	if !exists ||
		s.task != specification.task ||
		s.resource != specification.resource ||
		s.capabilityID != specification.capabilityID ||
		s.capabilityMajorVersion != specification.majorVersion {
		return fmt.Errorf("公開 step と採用済み capability の対応が一致しません")
	}
	return nil
}

// MarshalJSON は、logical input を含まない公開 step を表す。
func (s LegalQueryStepSummary) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		StepID                 string   `json:"stepId"`
		Task                   Task     `json:"task"`
		Resource               Resource `json:"resource"`
		CapabilityID           string   `json:"capabilityId"`
		CapabilityMajorVersion int      `json:"capabilityMajorVersion"`
	}{
		StepID:                 s.stepID,
		Task:                   s.task,
		Resource:               s.resource,
		CapabilityID:           s.capabilityID,
		CapabilityMajorVersion: s.capabilityMajorVersion,
	})
}

// UnmarshalJSON は、candidate step を介さない直接復元を拒否する。
func (*LegalQueryStepSummary) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryStepSummary は JSON から直接復元できません。NewLegalQueryStepSummary を使用してください",
	)
}

// LegalQueryInterpretation は、選択した意味候補の公開可能な項目を保持する。
type LegalQueryInterpretation struct {
	interpretationID string
	confidence       Confidence
	evidenceCodes    []EvidenceCode
	conceptSources   []LegalConceptSource
	availability     SelectionAvailability
	requiredPacks    []string
	steps            []LegalQueryStepSummary
}

// NewLegalQueryInterpretations は、selection 順に公開解釈を決定的に作る。
func NewLegalQueryInterpretations(
	plan LegalQueryPlan,
) ([]LegalQueryInterpretation, error) {
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("plan が有効ではありません: %w", err)
	}
	candidates := make(
		map[string]LegalQueryCandidate,
		len(plan.rankedCandidates),
	)
	for _, candidate := range plan.rankedCandidates {
		candidates[candidate.CandidateID()] = candidate
	}
	interpretations := make(
		[]LegalQueryInterpretation,
		0,
		len(plan.selected),
	)
	for index, selection := range plan.selected {
		candidate, exists := candidates[selection.CandidateID()]
		if !exists {
			return nil, fmt.Errorf("selection が参照する candidate がありません")
		}
		interpretation, err := newLegalQueryInterpretation(
			interpretationIDForIndex(index),
			candidate,
			selection,
		)
		if err != nil {
			return nil, fmt.Errorf("interpretations[%d] を作成できません: %w", index, err)
		}
		interpretations = append(interpretations, interpretation)
	}
	return interpretations, nil
}

// InterpretationID は、結果内で一意な公開識別子を返す。
func (i LegalQueryInterpretation) InterpretationID() string {
	return i.interpretationID
}

// Confidence は、候補の公開確信度を返す。
func (i LegalQueryInterpretation) Confidence() Confidence {
	return i.confidence
}

// EvidenceCodes は、公開可能な根拠分類の複製を返す。
func (i LegalQueryInterpretation) EvidenceCodes() []EvidenceCode {
	return append([]EvidenceCode{}, i.evidenceCodes...)
}

// ConceptSources は、法概念の公的資料の複製を返す。
func (i LegalQueryInterpretation) ConceptSources() []LegalConceptSource {
	return append([]LegalConceptSource{}, i.conceptSources...)
}

// Availability は、必要な pack の利用可否を返す。
func (i LegalQueryInterpretation) Availability() SelectionAvailability {
	return i.availability
}

// RequiredPacks は、実行に必要な pack ID の複製を返す。
func (i LegalQueryInterpretation) RequiredPacks() []string {
	return append([]string{}, i.requiredPacks...)
}

// Steps は、計画順の公開 step の複製を返す。
func (i LegalQueryInterpretation) Steps() []LegalQueryStepSummary {
	return append([]LegalQueryStepSummary{}, i.steps...)
}

// Validate は、公開解釈の識別子、根拠、pack および step を確認する。
func (i LegalQueryInterpretation) Validate() error {
	if !isInterpretationID(i.interpretationID) {
		return fmt.Errorf("interpretationId は interpretation-1 または interpretation-2 でなければなりません")
	}
	if !isConfidence(i.confidence) {
		return fmt.Errorf("confidence は high、medium または low でなければなりません")
	}
	hasLegalConcept, err := validateEvidenceCodes(i.evidenceCodes)
	if err != nil {
		return err
	}
	if err := validateConceptSources(i.conceptSources, hasLegalConcept); err != nil {
		return err
	}
	if err := validateRequiredPacks(i.requiredPacks); err != nil {
		return err
	}
	if i.availability != SelectionAvailabilityAvailable &&
		i.availability != SelectionAvailabilityPackDisabled {
		return fmt.Errorf("availability が定義されていません")
	}
	if i.availability == SelectionAvailabilityPackDisabled &&
		len(i.requiredPacks) == 0 {
		return fmt.Errorf("pack_disabled の解釈には requiredPacks が一件以上必要です")
	}
	return validateStepSummaries(i.steps)
}

// MarshalJSON は、内部候補 ID と score を除いた公開解釈を表す。
func (i LegalQueryInterpretation) MarshalJSON() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		InterpretationID string                  `json:"interpretationId"`
		Confidence       Confidence              `json:"confidence"`
		EvidenceCodes    []EvidenceCode          `json:"evidenceCodes"`
		ConceptSources   []LegalConceptSource    `json:"conceptSources"`
		Availability     SelectionAvailability   `json:"availability"`
		RequiredPacks    []string                `json:"requiredPacks"`
		Steps            []LegalQueryStepSummary `json:"steps"`
	}{
		InterpretationID: i.interpretationID,
		Confidence:       i.confidence,
		EvidenceCodes:    i.EvidenceCodes(),
		ConceptSources:   i.ConceptSources(),
		Availability:     i.availability,
		RequiredPacks:    i.RequiredPacks(),
		Steps:            i.Steps(),
	})
}

// UnmarshalJSON は、plan selection を介さない直接復元を拒否する。
func (*LegalQueryInterpretation) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryInterpretation は JSON から直接復元できません。NewLegalQueryInterpretations を使用してください",
	)
}

func newLegalQueryInterpretation(
	interpretationID string,
	candidate LegalQueryCandidate,
	selection LegalQueryPlanSelection,
) (LegalQueryInterpretation, error) {
	if candidate.CandidateID() != selection.CandidateID() {
		return LegalQueryInterpretation{}, fmt.Errorf("candidate と selection が一致しません")
	}
	steps := make([]LegalQueryStepSummary, 0, len(candidate.steps))
	for _, step := range candidate.steps {
		summary, err := NewLegalQueryStepSummary(step)
		if err != nil {
			return LegalQueryInterpretation{}, err
		}
		steps = append(steps, summary)
	}
	interpretation := LegalQueryInterpretation{
		interpretationID: interpretationID,
		confidence:       candidate.Confidence(),
		evidenceCodes:    candidate.EvidenceCodes(),
		conceptSources:   candidate.ConceptSources(),
		availability:     selection.Availability(),
		requiredPacks:    selection.RequiredPacks(),
		steps:            steps,
	}
	if err := interpretation.Validate(); err != nil {
		return LegalQueryInterpretation{}, err
	}
	return interpretation, nil
}

func validateStepSummaries(values []LegalQueryStepSummary) error {
	if len(values) < 1 || len(values) > MaxCapabilityCalls {
		return fmt.Errorf("steps は一件以上四件以下でなければなりません")
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("steps[%d] が有効ではありません: %w", index, err)
		}
		if _, exists := seen[value.StepID()]; exists {
			return fmt.Errorf("steps の stepId を重複させることはできません")
		}
		seen[value.StepID()] = struct{}{}
	}
	return nil
}

func interpretationIDForIndex(index int) string {
	return fmt.Sprintf("interpretation-%d", index+1)
}

func isInterpretationID(value string) bool {
	return value == "interpretation-1" || value == "interpretation-2"
}

// LegalQueryClarification は、明確化の理由と固定質問を保持する。
type LegalQueryClarification struct {
	reasonCodes []ReasonCode
	questions   []LegalQueryQuestion
}

// NewLegalQueryClarification は、needs_clarification plan の理由と固定質問を結び付ける。
func NewLegalQueryClarification(
	plan LegalQueryPlan,
	questions []LegalQueryQuestion,
) (LegalQueryClarification, error) {
	if err := plan.Validate(); err != nil {
		return LegalQueryClarification{}, fmt.Errorf("plan が有効ではありません: %w", err)
	}
	if plan.Decision() != PlanDecisionNeedsClarification {
		return LegalQueryClarification{}, fmt.Errorf("clarification には needs_clarification plan が必要です")
	}
	clarification := LegalQueryClarification{
		reasonCodes: plan.ReasonCodes(),
		questions:   append([]LegalQueryQuestion{}, questions...),
	}
	if err := clarification.Validate(); err != nil {
		return LegalQueryClarification{}, err
	}
	return clarification, nil
}

// ReasonCodes は、明確化理由の複製を返す。
func (c LegalQueryClarification) ReasonCodes() []ReasonCode {
	return append([]ReasonCode{}, c.reasonCodes...)
}

// Questions は、固定質問の複製を返す。
func (c LegalQueryClarification) Questions() []LegalQueryQuestion {
	return append([]LegalQueryQuestion{}, c.questions...)
}

// Validate は、明確化理由と質問の件数、値および順序を確認する。
func (c LegalQueryClarification) Validate() error {
	if err := validateReasonCodes(
		PlanDecisionNeedsClarification,
		c.reasonCodes,
	); err != nil {
		return err
	}
	if len(c.reasonCodes) == 1 &&
		c.reasonCodes[0] == ReasonCodeStepLimitExceeded {
		if len(c.questions) != 1 ||
			c.questions[0] != LegalQueryQuestionStepLimitExceeded {
			return fmt.Errorf("step_limit_exceeded には専用の質問が一件必要です")
		}
		return nil
	}
	for _, question := range c.questions {
		if question == LegalQueryQuestionStepLimitExceeded {
			return fmt.Errorf("step 上限の専用質問には step_limit_exceeded が必要です")
		}
	}
	if len(c.questions) < 1 || len(c.questions) > 2 {
		return fmt.Errorf("questions は一件以上二件以下でなければなりません")
	}
	previousRank := -1
	for _, question := range c.questions {
		rank, exists := legalQueryQuestionRank(question)
		if !exists {
			return fmt.Errorf("questions に定義されていない値があります")
		}
		if rank <= previousRank {
			return fmt.Errorf("questions は重複させず規定の順序で並べなければなりません")
		}
		previousRank = rank
	}
	return nil
}

// MarshalJSON は、固定理由と質問だけを表す。
func (c LegalQueryClarification) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ReasonCodes []ReasonCode         `json:"reasonCodes"`
		Questions   []LegalQueryQuestion `json:"questions"`
	}{
		ReasonCodes: c.ReasonCodes(),
		Questions:   c.Questions(),
	})
}

// UnmarshalJSON は、plan を介さない直接復元を拒否する。
func (*LegalQueryClarification) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryClarification は JSON から直接復元できません。NewLegalQueryClarification を使用してください",
	)
}

func legalQueryQuestionRank(value LegalQueryQuestion) (int, bool) {
	switch value {
	case LegalQueryQuestionTask:
		return 0, true
	case LegalQueryQuestionResource:
		return 1, true
	case LegalQueryQuestionLaw:
		return 2, true
	case LegalQueryQuestionJudicialDecision:
		return 3, true
	case LegalQueryQuestionStepLimitExceeded:
		return 4, true
	default:
		return 0, false
	}
}
