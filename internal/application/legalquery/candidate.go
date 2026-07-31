package legalquery

import (
	"fmt"
	"regexp"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
)

const (
	maximumQueryPlanIDBytes = 64
)

var queryPlanIDPattern = regexp.MustCompile(
	`^[a-z0-9]+(?:-[a-z0-9]+)*$`,
)

// Confidence は、意味候補の確信度区分を表す。
type Confidence string

const (
	// ConfidenceHigh は、高い確信度を表す。
	ConfidenceHigh Confidence = "high"
	// ConfidenceMedium は、中程度の確信度を表す。
	ConfidenceMedium Confidence = "medium"
	// ConfidenceLow は、低い確信度を表す。
	ConfidenceLow Confidence = "low"
)

// EvidenceCode は、意味候補を支持する根拠分類を表す。
type EvidenceCode string

const (
	// EvidenceOfficialIdentifier は、公式識別子または入力参照による根拠を表す。
	EvidenceOfficialIdentifier EvidenceCode = "official_identifier"
	// EvidenceStructuredReference は、構造化された法的参照による根拠を表す。
	EvidenceStructuredReference EvidenceCode = "structured_reference"
	// EvidenceExplicitTask は、明示された作業による根拠を表す。
	EvidenceExplicitTask EvidenceCode = "explicit_task"
	// EvidenceExplicitResource は、明示された資源による根拠を表す。
	EvidenceExplicitResource EvidenceCode = "explicit_resource"
	// EvidenceOfficialAlias は、出典付きの正式名称または別名による根拠を表す。
	EvidenceOfficialAlias EvidenceCode = "official_alias"
	// EvidenceLegalConcept は、出典付きの法概念による根拠を表す。
	EvidenceLegalConcept EvidenceCode = "legal_concept"
	// EvidenceMorphologicalContext は、形態素と周辺語の文脈による根拠を表す。
	EvidenceMorphologicalContext EvidenceCode = "morphological_context"
	// EvidenceUniqueTypoCorrection は、一意な軽微誤記補正による根拠を表す。
	EvidenceUniqueTypoCorrection EvidenceCode = "unique_typo_correction"
	// EvidenceGeneralTerm は、一般語による弱い根拠を表す。
	EvidenceGeneralTerm EvidenceCode = "general_term"
)

// Order は、SOT-MODEL-022 が固定する強い根拠からの順序と有無を返す。
func (c EvidenceCode) Order() (int, bool) {
	return evidenceRank(c)
}

// Task は、候補 step が行う作業を表す。
type Task string

const (
	// TaskSearch は、検索を表す。
	TaskSearch Task = "search"
	// TaskRead は、一意な資源の読取りを表す。
	TaskRead Task = "read"
	// TaskListUpdates は、更新一覧の取得を表す。
	TaskListUpdates Task = "list_updates"
)

// Resource は、候補 step が対象にする法情報資源を表す。
type Resource string

const (
	// ResourceLaw は、法令を表す。
	ResourceLaw Resource = "law"
	// ResourceLawProvision は、法令の条文を表す。
	ResourceLawProvision Resource = "law_provision"
	// ResourceJudicialDecision は、公表裁判例を表す。
	ResourceJudicialDecision Resource = "judicial_decision"
)

// LegalQueryCandidateStepValues は、候補 step の作成に必要な値を保持する。
type LegalQueryCandidateStepValues struct {
	StepID                 string
	Task                   Task
	Resource               Resource
	CapabilityID           string
	CapabilityMajorVersion int
	InputKind              LogicalInputKind
	LogicalInput           LogicalInput
}

// LegalQueryCandidateStep は、provider を選ぶ前の一つの能力呼出しを表す。
type LegalQueryCandidateStep struct {
	stepID                 string
	task                   Task
	resource               Resource
	capabilityID           string
	capabilityMajorVersion int
	inputKind              LogicalInputKind
	logicalInput           LogicalInput
}

// NewLegalQueryCandidateStep は、許可された対応を持つ不変な step を返す。
func NewLegalQueryCandidateStep(
	values LegalQueryCandidateStepValues,
) (LegalQueryCandidateStep, error) {
	logicalInput, err := cloneLogicalInput(values.LogicalInput)
	if err != nil {
		return LegalQueryCandidateStep{}, err
	}
	step := LegalQueryCandidateStep{
		stepID:                 values.StepID,
		task:                   values.Task,
		resource:               values.Resource,
		capabilityID:           values.CapabilityID,
		capabilityMajorVersion: values.CapabilityMajorVersion,
		inputKind:              values.InputKind,
		logicalInput:           logicalInput,
	}
	if err := step.Validate(); err != nil {
		return LegalQueryCandidateStep{}, err
	}
	return step, nil
}

// StepID は、plan 内で一意な step 識別子を返す。
func (s LegalQueryCandidateStep) StepID() string {
	return s.stepID
}

// Task は、step が行う作業を返す。
func (s LegalQueryCandidateStep) Task() Task {
	return s.task
}

// Resource は、step が対象にする資源を返す。
func (s LegalQueryCandidateStep) Resource() Resource {
	return s.resource
}

// CapabilityID は、必要な能力識別子を返す。
func (s LegalQueryCandidateStep) CapabilityID() string {
	return s.capabilityID
}

// CapabilityMajorVersion は、必要な能力のメジャーバージョンを返す。
func (s LegalQueryCandidateStep) CapabilityMajorVersion() int {
	return s.capabilityMajorVersion
}

// InputKind は、logical input の variant を返す。
func (s LegalQueryCandidateStep) InputKind() LogicalInputKind {
	return s.inputKind
}

// LogicalInput は、provider 選択前の取得条件の複製を返す。
func (s LegalQueryCandidateStep) LogicalInput() LogicalInput {
	return mustCloneLogicalInput(s.logicalInput)
}

// Validate は、step ID、七つの対応および logical input を検証する。
func (s LegalQueryCandidateStep) Validate() error {
	if err := validateQueryPlanID("stepId", s.stepID); err != nil {
		return err
	}
	specification, exists := stepSpecificationFor(s.inputKind)
	if !exists {
		return fmt.Errorf("inputKind が定義されていません")
	}
	if s.task != specification.task ||
		s.resource != specification.resource ||
		s.capabilityID != specification.capabilityID ||
		s.capabilityMajorVersion != specification.majorVersion {
		return fmt.Errorf("task、resource、capability および inputKind の対応が許可されていません")
	}
	input, err := cloneLogicalInput(s.logicalInput)
	if err != nil {
		return err
	}
	if input.InputKind() != s.inputKind {
		return fmt.Errorf("inputKind と logicalInput の variant が一致しません")
	}
	if err := input.Validate(); err != nil {
		return fmt.Errorf("logicalInput が有効ではありません: %w", err)
	}
	return nil
}

// UnmarshalJSON は、planner を介さない直接復元を拒否する。
func (*LegalQueryCandidateStep) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryCandidateStep は JSON から直接復元できません。NewLegalQueryCandidateStep を使用してください",
	)
}

type stepSpecification struct {
	task         Task
	resource     Resource
	capabilityID string
	majorVersion int
}

func stepSpecificationFor(
	kind LogicalInputKind,
) (stepSpecification, bool) {
	switch kind {
	case InputKindLawSearch:
		return stepSpecification{
			task: TaskSearch, resource: ResourceLaw,
			capabilityID: lawsearch.CapabilityID,
			majorVersion: lawsearch.MajorVersion,
		}, true
	case InputKindLawContentSearch:
		return stepSpecification{
			task: TaskSearch, resource: ResourceLawProvision,
			capabilityID: lawcontentsearch.CapabilityID,
			majorVersion: lawcontentsearch.MajorVersion,
		}, true
	case InputKindLawRead:
		return stepSpecification{
			task: TaskRead, resource: ResourceLaw,
			capabilityID: lawdocumentread.CapabilityID,
			majorVersion: lawdocumentread.MajorVersion,
		}, true
	case InputKindLawArticleRead:
		return stepSpecification{
			task: TaskRead, resource: ResourceLawProvision,
			capabilityID: lawarticleread.CapabilityID,
			majorVersion: lawarticleread.MajorVersion,
		}, true
	case InputKindLawUpdates:
		return stepSpecification{
			task: TaskListUpdates, resource: ResourceLaw,
			capabilityID: lawupdatelist.CapabilityID,
			majorVersion: lawupdatelist.MajorVersion,
		}, true
	case InputKindJudicialDecisionSearch:
		return stepSpecification{
			task: TaskSearch, resource: ResourceJudicialDecision,
			capabilityID: judicialdecisionsearch.CapabilityID,
			majorVersion: judicialdecisionsearch.MajorVersion,
		}, true
	case InputKindJudicialDecisionRead:
		return stepSpecification{
			task: TaskRead, resource: ResourceJudicialDecision,
			capabilityID: judicialdecisionread.CapabilityID,
			majorVersion: judicialdecisionread.MajorVersion,
		}, true
	default:
		return stepSpecification{}, false
	}
}

// LegalQueryCandidateValues は、意味候補の作成に必要な値を保持する。
type LegalQueryCandidateValues struct {
	CandidateID    string
	SemanticScore  int
	Confidence     Confidence
	EvidenceCodes  []EvidenceCode
	ConceptSources []LegalConceptSource
	RequiredPacks  []string
	Steps          []LegalQueryCandidateStep
}

// LegalQueryCandidate は、一つの照会文から導いた一つの意味解釈を表す。
type LegalQueryCandidate struct {
	candidateID    string
	semanticScore  int
	confidence     Confidence
	evidenceCodes  []EvidenceCode
	conceptSources []LegalConceptSource
	requiredPacks  []string
	steps          []LegalQueryCandidateStep
}

// NewLegalQueryCandidate は、入力を複製して検証済みの意味候補を返す。
func NewLegalQueryCandidate(
	values LegalQueryCandidateValues,
) (LegalQueryCandidate, error) {
	candidate, err := cloneCandidateValues(values)
	if err != nil {
		return LegalQueryCandidate{}, err
	}
	if err := candidate.Validate(); err != nil {
		return LegalQueryCandidate{}, err
	}
	return candidate, nil
}

// CandidateID は、plan 内で一意な不透明識別子を返す。
func (c LegalQueryCandidate) CandidateID() string {
	return c.candidateID
}

// SemanticScore は、同じ profile version 内の順位値を返す。
func (c LegalQueryCandidate) SemanticScore() int {
	return c.semanticScore
}

// Confidence は、意味候補の確信度区分を返す。
func (c LegalQueryCandidate) Confidence() Confidence {
	return c.confidence
}

// EvidenceCodes は、強い順に並んだ根拠分類の複製を返す。
func (c LegalQueryCandidate) EvidenceCodes() []EvidenceCode {
	return append([]EvidenceCode{}, c.evidenceCodes...)
}

// ConceptSources は、使用した法概念の公的資料の複製を返す。
func (c LegalQueryCandidate) ConceptSources() []LegalConceptSource {
	return append([]LegalConceptSource{}, c.conceptSources...)
}

// RequiredPacks は、必要な拡張パック ID の複製を返す。
func (c LegalQueryCandidate) RequiredPacks() []string {
	return append([]string{}, c.requiredPacks...)
}

// Steps は、計画上の実行順に並んだ step の複製を返す。
func (c LegalQueryCandidate) Steps() []LegalQueryCandidateStep {
	cloned := make([]LegalQueryCandidateStep, 0, len(c.steps))
	for _, step := range c.steps {
		cloned = append(cloned, step.mustClone())
	}
	return cloned
}

// Validate は、候補の識別子、根拠、pack および step を検証する。
func (c LegalQueryCandidate) Validate() error {
	if err := validateQueryPlanID("candidateId", c.candidateID); err != nil {
		return err
	}
	if !isConfidence(c.confidence) {
		return fmt.Errorf("confidence は high、medium または low でなければなりません")
	}
	hasLegalConcept, err := validateEvidenceCodes(c.evidenceCodes)
	if err != nil {
		return err
	}
	if err := validateConceptSources(c.conceptSources, hasLegalConcept); err != nil {
		return err
	}
	if err := validateCandidateSteps(c.steps); err != nil {
		return err
	}
	if err := validateRequiredPacks(c.requiredPacks); err != nil {
		return err
	}
	return nil
}

// UnmarshalJSON は、planner を介さない直接復元を拒否する。
func (*LegalQueryCandidate) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryCandidate は JSON から直接復元できません。NewLegalQueryCandidate を使用してください",
	)
}

func cloneCandidateValues(
	values LegalQueryCandidateValues,
) (LegalQueryCandidate, error) {
	steps := make([]LegalQueryCandidateStep, 0, len(values.Steps))
	for _, step := range values.Steps {
		cloned, err := step.clone()
		if err != nil {
			return LegalQueryCandidate{}, err
		}
		steps = append(steps, cloned)
	}
	return LegalQueryCandidate{
		candidateID:    values.CandidateID,
		semanticScore:  values.SemanticScore,
		confidence:     values.Confidence,
		evidenceCodes:  append([]EvidenceCode{}, values.EvidenceCodes...),
		conceptSources: append([]LegalConceptSource{}, values.ConceptSources...),
		requiredPacks:  append([]string{}, values.RequiredPacks...),
		steps:          steps,
	}, nil
}

func (s LegalQueryCandidateStep) clone() (LegalQueryCandidateStep, error) {
	input, err := cloneLogicalInput(s.logicalInput)
	if err != nil {
		return LegalQueryCandidateStep{}, err
	}
	cloned := s
	cloned.logicalInput = input
	return cloned, nil
}

func (s LegalQueryCandidateStep) mustClone() LegalQueryCandidateStep {
	cloned, err := s.clone()
	if err != nil {
		panic(fmt.Sprintf("検証済み candidate step の複製に失敗しました: %v", err))
	}
	return cloned
}

func validateQueryPlanID(field string, value string) error {
	switch {
	case len(value) < 1 || len(value) > maximumQueryPlanIDBytes:
		return fmt.Errorf("%s は 1 byte 以上 64 byte 以下でなければなりません", field)
	case !queryPlanIDPattern.MatchString(value):
		return fmt.Errorf("%s は小文字英数字の segment を - で連結しなければなりません", field)
	default:
		return nil
	}
}

func isConfidence(value Confidence) bool {
	return value == ConfidenceHigh ||
		value == ConfidenceMedium ||
		value == ConfidenceLow
}

func validateEvidenceCodes(values []EvidenceCode) (bool, error) {
	if len(values) == 0 {
		return false, fmt.Errorf("evidenceCodes は一件以上必要です")
	}
	previousRank := -1
	hasLegalConcept := false
	for _, value := range values {
		rank, exists := evidenceRank(value)
		if !exists {
			return false, fmt.Errorf("evidenceCodes に定義されていない値があります")
		}
		if rank <= previousRank {
			return false, fmt.Errorf("evidenceCodes は重複させず強い根拠から順に並べなければなりません")
		}
		previousRank = rank
		hasLegalConcept = hasLegalConcept || value == EvidenceLegalConcept
	}
	return hasLegalConcept, nil
}

func evidenceRank(value EvidenceCode) (int, bool) {
	switch value {
	case EvidenceOfficialIdentifier:
		return 0, true
	case EvidenceStructuredReference:
		return 1, true
	case EvidenceExplicitTask:
		return 2, true
	case EvidenceExplicitResource:
		return 3, true
	case EvidenceOfficialAlias:
		return 4, true
	case EvidenceLegalConcept:
		return 5, true
	case EvidenceMorphologicalContext:
		return 6, true
	case EvidenceUniqueTypoCorrection:
		return 7, true
	case EvidenceGeneralTerm:
		return 8, true
	default:
		return 0, false
	}
}

func validateConceptSources(
	values []LegalConceptSource,
	hasLegalConcept bool,
) error {
	if hasLegalConcept != (len(values) > 0) {
		return fmt.Errorf("conceptSources は legal_concept の根拠がある場合だけ一件以上必要です")
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return fmt.Errorf("conceptSources[%d] が有効ではありません: %w", index, err)
		}
		if _, exists := seen[value.ConceptID()]; exists {
			return fmt.Errorf("conceptSources の conceptId を重複させることはできません")
		}
		seen[value.ConceptID()] = struct{}{}
	}
	return nil
}

func validateCandidateSteps(
	values []LegalQueryCandidateStep,
) error {
	if len(values) < 1 || len(values) > 4 {
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

func validateRequiredPacks(values []string) error {
	for index, value := range values {
		if err := validateQueryPlanID("requiredPacks", value); err != nil {
			return err
		}
		if index > 0 && values[index-1] >= value {
			return fmt.Errorf("requiredPacks は重複させず昇順に並べなければなりません")
		}
	}
	return nil
}
