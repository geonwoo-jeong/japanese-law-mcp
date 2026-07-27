package legalquerycorpus

import (
	"fmt"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const maximumExpectedMeaningSteps = 4

// ExpectedMeaningValues は、評価上の一つの正しい意味を構成する値を保持する。
type ExpectedMeaningValues struct {
	MeaningID     string
	EvidenceCodes []legalquery.EvidenceCode
	ConceptIDs    []string
	RequiredPacks []string
	Steps         []ExpectedStep
}

// ExpectedMeaning は、候補の内部 ID や score を持たない意味署名と根拠 assertion である。
type ExpectedMeaning struct {
	meaningID     string
	evidenceCodes []legalquery.EvidenceCode
	conceptIDs    []string
	requiredPacks []string
	steps         []ExpectedStep
	initialized   bool
}

// NewExpectedMeaning は、意味署名と根拠を複製して不変な期待意味を返す。
func NewExpectedMeaning(values ExpectedMeaningValues) (ExpectedMeaning, error) {
	steps, err := cloneExpectedSteps(values.Steps)
	if err != nil {
		return ExpectedMeaning{}, err
	}
	meaning := ExpectedMeaning{
		meaningID:     values.MeaningID,
		evidenceCodes: append([]legalquery.EvidenceCode{}, values.EvidenceCodes...),
		conceptIDs:    cloneStrings(values.ConceptIDs),
		requiredPacks: cloneStrings(values.RequiredPacks),
		steps:         steps,
		initialized:   true,
	}
	if err := meaning.Validate(); err != nil {
		return ExpectedMeaning{}, err
	}
	return meaning, nil
}

// MeaningID は、fixture 内だけで安定した意味識別子を返す。
func (m ExpectedMeaning) MeaningID() string {
	return m.meaningID
}

// EvidenceCodes は、正確な根拠分類列の複製を返す。
func (m ExpectedMeaning) EvidenceCodes() []legalquery.EvidenceCode {
	return append([]legalquery.EvidenceCode{}, m.evidenceCodes...)
}

// ConceptIDs は、辞書根拠の不透明 ID 列の複製を返す。
func (m ExpectedMeaning) ConceptIDs() []string {
	return cloneStrings(m.conceptIDs)
}

// RequiredPacks は、実行に必要な pack ID 列の複製を返す。
func (m ExpectedMeaning) RequiredPacks() []string {
	return cloneStrings(m.requiredPacks)
}

// Steps は、意味署名を構成する計画順 step の複製を返す。
func (m ExpectedMeaning) Steps() []ExpectedStep {
	steps, err := cloneExpectedSteps(m.steps)
	if err != nil {
		panic(fmt.Sprintf("検証済み ExpectedMeaning の steps 複製に失敗しました: %v", err))
	}
	return steps
}

// Validate は、意味 ID、根拠、pack および step の不変条件を確認する。
func (m ExpectedMeaning) Validate() error {
	if !m.initialized {
		return fmt.Errorf("ExpectedMeaning は NewExpectedMeaning で作成しなければなりません")
	}
	if err := validateExpectedIdentifier("meaningId", m.meaningID); err != nil {
		return err
	}
	hasLegalConcept, err := validateExpectedEvidenceCodes(m.evidenceCodes)
	if err != nil {
		return err
	}
	if err := validateExpectedConceptIDs(m.conceptIDs, hasLegalConcept); err != nil {
		return err
	}
	if err := validateExpectedRequiredPacks(m.requiredPacks); err != nil {
		return err
	}
	return validateExpectedSteps(m.steps)
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExpectedMeaning) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ExpectedMeaning は JSON から直接復元できません。version 別 DTO を使用してください",
	)
}

func (m ExpectedMeaning) clone() ExpectedMeaning {
	meaning, err := NewExpectedMeaning(ExpectedMeaningValues{
		MeaningID:     m.meaningID,
		EvidenceCodes: m.evidenceCodes,
		ConceptIDs:    m.conceptIDs,
		RequiredPacks: m.requiredPacks,
		Steps:         m.steps,
	})
	if err != nil {
		panic(fmt.Sprintf("検証済み ExpectedMeaning の複製に失敗しました: %v", err))
	}
	return meaning
}

func cloneExpectedSteps(values []ExpectedStep) ([]ExpectedStep, error) {
	cloned := make([]ExpectedStep, 0, len(values))
	for _, step := range values {
		if err := step.Validate(); err != nil {
			return nil, fmt.Errorf("expected meaning の step が有効ではありません: %w", err)
		}
		copy, err := NewExpectedStep(ExpectedStepValues{
			Task:         step.Task(),
			Resource:     step.Resource(),
			InputKind:    step.InputKind(),
			LogicalInput: step.LogicalInput(),
		})
		if err != nil {
			return nil, fmt.Errorf("expected meaning の step が有効ではありません: %w", err)
		}
		cloned = append(cloned, copy)
	}
	return cloned, nil
}

func validateExpectedSteps(values []ExpectedStep) error {
	if len(values) < 1 || len(values) > maximumExpectedMeaningSteps {
		return fmt.Errorf("expected meaning の steps は一件以上四件以下でなければなりません")
	}
	for _, step := range values {
		if err := step.Validate(); err != nil {
			return fmt.Errorf("expected meaning の step が有効ではありません: %w", err)
		}
	}
	return nil
}

func validateExpectedEvidenceCodes(
	values []legalquery.EvidenceCode,
) (bool, error) {
	if len(values) < 1 || len(values) > 9 {
		return false, fmt.Errorf("expected meaning の evidenceCodes は一件以上九件以下でなければなりません")
	}
	previous := -1
	hasLegalConcept := false
	for _, value := range values {
		rank, exists := expectedEvidenceRank(value)
		if !exists {
			return false, fmt.Errorf("expected meaning の evidenceCodes に未定義の値があります")
		}
		if rank <= previous {
			return false, fmt.Errorf("expected meaning の evidenceCodes は規定順で重複なく保持してください")
		}
		previous = rank
		hasLegalConcept = hasLegalConcept || value == legalquery.EvidenceLegalConcept
	}
	return hasLegalConcept, nil
}

func expectedEvidenceRank(value legalquery.EvidenceCode) (int, bool) {
	switch value {
	case legalquery.EvidenceOfficialIdentifier:
		return 0, true
	case legalquery.EvidenceStructuredReference:
		return 1, true
	case legalquery.EvidenceExplicitTask:
		return 2, true
	case legalquery.EvidenceExplicitResource:
		return 3, true
	case legalquery.EvidenceOfficialAlias:
		return 4, true
	case legalquery.EvidenceLegalConcept:
		return 5, true
	case legalquery.EvidenceMorphologicalContext:
		return 6, true
	case legalquery.EvidenceUniqueTypoCorrection:
		return 7, true
	case legalquery.EvidenceGeneralTerm:
		return 8, true
	default:
		return 0, false
	}
}

func validateExpectedConceptIDs(values []string, required bool) error {
	if required != (len(values) > 0) {
		return fmt.Errorf("expected meaning の conceptIds が legal_concept 根拠と一致しません")
	}
	previous := ""
	for index, value := range values {
		if value == "" || !utf8.ValidString(value) {
			return fmt.Errorf("expected meaning の conceptId は空でない UTF-8 でなければなりません")
		}
		if index > 0 && previous >= value {
			return fmt.Errorf("expected meaning の conceptIds は昇順で重複なく保持してください")
		}
		previous = value
	}
	return nil
}

func validateExpectedRequiredPacks(values []string) error {
	if len(values) > 64 {
		return fmt.Errorf("expected meaning の requiredPacks は六十四件以下でなければなりません")
	}
	previous := ""
	for index, value := range values {
		if err := validateExpectedIdentifier("requiredPacks", value); err != nil {
			return err
		}
		if index > 0 && previous >= value {
			return fmt.Errorf("expected meaning の requiredPacks は昇順で重複なく保持してください")
		}
		previous = value
	}
	return nil
}

func validateExpectedIdentifier(field string, value string) error {
	if len(value) < 1 ||
		len(value) > 64 ||
		!manifestIdentifierPattern.MatchString(value) {
		return fmt.Errorf("%s は 1 byte 以上 64 byte 以下の正規形でなければなりません", field)
	}
	return nil
}
