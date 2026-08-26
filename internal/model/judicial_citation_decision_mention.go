package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const judicialCitationDecisionMentionTextMaxBytes = 4096

// JudicialCitationDecisionMentionValues は、PDF 内の確認済み裁判例参照出現の作成値を保持する。
type JudicialCitationDecisionMentionValues struct {
	ReferenceText        string
	DecisionIdentityText string
	Evidence             JudicialCitationEvidence
}

// JudicialCitationDecisionMention は、一つの明示的な裁判例参照出現を表す。
type JudicialCitationDecisionMention struct {
	referenceText        string
	decisionIdentityText string
	evidence             JudicialCitationEvidence
}

// NewJudicialCitationDecisionMention は、厳密に同定できた裁判例参照出現を返す。
func NewJudicialCitationDecisionMention(
	values JudicialCitationDecisionMentionValues,
) (JudicialCitationDecisionMention, error) {
	mention := JudicialCitationDecisionMention{
		referenceText:        values.ReferenceText,
		decisionIdentityText: values.DecisionIdentityText,
		evidence:             values.Evidence,
	}
	if err := mention.Validate(); err != nil {
		return JudicialCitationDecisionMention{}, err
	}
	return mention, nil
}

func (m JudicialCitationDecisionMention) ReferenceText() string {
	return m.referenceText
}

func (m JudicialCitationDecisionMention) DecisionIdentityText() string {
	return m.decisionIdentityText
}

func (m JudicialCitationDecisionMention) Evidence() JudicialCitationEvidence {
	return m.evidence
}

// Validate は、参照原文、裁判例同一性および exact_text_match 根拠を確認する。
func (m JudicialCitationDecisionMention) Validate() error {
	if err := validateJudicialCitationDecisionMentionText(
		"referenceText",
		m.referenceText,
	); err != nil {
		return err
	}
	if err := validateJudicialCitationDecisionMentionText(
		"decisionIdentityText",
		m.decisionIdentityText,
	); err != nil {
		return err
	}
	if err := m.evidence.Validate(); err != nil {
		return fmt.Errorf("evidence が有効ではありません: %w", err)
	}
	if m.evidence.EvidenceLevel() != JudicialCitationEvidenceLevelExactTextMatch {
		return fmt.Errorf("evidenceLevel は exact_text_match でなければなりません")
	}
	if _, exists := m.evidence.Excerpt(); !exists {
		return fmt.Errorf("exact_text_match evidence には excerpt が必須です")
	}
	provenance := m.evidence.Provenance()
	if provenance.MediaType() != JudicialDocumentMediaTypePDF {
		return fmt.Errorf("exact_text_match evidence の provenance は application/pdf でなければなりません")
	}
	if provenance.Transformation() != ProvenanceTransformationExtracted {
		return fmt.Errorf("exact_text_match evidence の provenance は extracted でなければなりません")
	}
	if _, exists := provenance.ContentDigest(); !exists {
		return fmt.Errorf("exact_text_match evidence の provenance には contentDigest が必須です")
	}
	return nil
}

func validateJudicialCitationDecisionMentionText(field string, value string) error {
	if !utf8.ValidString(value) || strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s は必須の UTF-8 文字列です", field)
	}
	if strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return fmt.Errorf("%s に制御文字を含めることはできません", field)
	}
	if len(value) > judicialCitationDecisionMentionTextMaxBytes {
		return fmt.Errorf(
			"%s は %d byte 以下でなければなりません",
			field,
			judicialCitationDecisionMentionTextMaxBytes,
		)
	}
	return nil
}

func (m JudicialCitationDecisionMention) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ReferenceText        string                   `json:"referenceText"`
		DecisionIdentityText string                   `json:"decisionIdentityText"`
		Evidence             JudicialCitationEvidence `json:"evidence"`
	}{
		ReferenceText:        m.referenceText,
		DecisionIdentityText: m.decisionIdentityText,
		Evidence:             m.evidence,
	})
}

func (*JudicialCitationDecisionMention) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationDecisionMention は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationDecisionMention を使用してください",
	)
}
