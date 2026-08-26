package model

import (
	"encoding/json"
	"fmt"
	"slices"
	"unicode/utf8"
)

// JudicialCitationEdgeValues は、引用 graph の edge 作成値を保持する。
type JudicialCitationEdgeValues struct {
	EdgeID       string
	FromNodeID   string
	ToNodeID     string
	RelationType JudicialCitationRelationType
	Evidence     []JudicialCitationEvidence
}

// JudicialCitationEdge は、graph 内の有向関係を表す。
type JudicialCitationEdge struct {
	edgeID       string
	fromNodeID   string
	toNodeID     string
	relationType JudicialCitationRelationType
	evidence     []JudicialCitationEvidence
}

type judicialCitationEdgeKey struct {
	fromNodeID   string
	toNodeID     string
	relationType JudicialCitationRelationType
}

func NewJudicialCitationEdge(values JudicialCitationEdgeValues) (JudicialCitationEdge, error) {
	evidence, err := uniqueJudicialCitationEvidence(values.Evidence)
	if err != nil {
		return JudicialCitationEdge{}, err
	}
	edge := JudicialCitationEdge{
		edgeID:       values.EdgeID,
		fromNodeID:   values.FromNodeID,
		toNodeID:     values.ToNodeID,
		relationType: values.RelationType,
		evidence:     evidence,
	}
	if err := edge.Validate(); err != nil {
		return JudicialCitationEdge{}, err
	}
	return edge, nil
}

func uniqueJudicialCitationEvidence(
	values []JudicialCitationEvidence,
) ([]JudicialCitationEvidence, error) {
	if values == nil {
		return nil, nil
	}
	unique := make([]JudicialCitationEvidence, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for index, evidence := range values {
		if err := evidence.Validate(); err != nil {
			return nil, fmt.Errorf("evidence[%d] が有効ではありません: %w", index, err)
		}
		encoded, err := json.Marshal(evidence)
		if err != nil {
			return nil, fmt.Errorf("evidence[%d] を比較可能な形にできません: %w", index, err)
		}
		key := string(encoded)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, evidence)
	}
	return unique, nil
}

func (e JudicialCitationEdge) EdgeID() string                             { return e.edgeID }
func (e JudicialCitationEdge) FromNodeID() string                         { return e.fromNodeID }
func (e JudicialCitationEdge) ToNodeID() string                           { return e.toNodeID }
func (e JudicialCitationEdge) RelationType() JudicialCitationRelationType { return e.relationType }
func (e JudicialCitationEdge) Evidence() []JudicialCitationEvidence {
	return slices.Clone(e.evidence)
}

func (e JudicialCitationEdge) key() judicialCitationEdgeKey {
	return judicialCitationEdgeKey{e.fromNodeID, e.toNodeID, e.relationType}
}

func (e JudicialCitationEdge) Validate() error {
	if !utf8.ValidString(e.edgeID) || e.edgeID == "" {
		return fmt.Errorf("edgeId は必須の UTF-8 文字列です")
	}
	if !utf8.ValidString(e.fromNodeID) || e.fromNodeID == "" ||
		!utf8.ValidString(e.toNodeID) || e.toNodeID == "" {
		return fmt.Errorf("fromNodeId と toNodeId は必須の UTF-8 文字列です")
	}
	if e.fromNodeID == e.toNodeID {
		return fmt.Errorf("fromNodeId と toNodeId は異なる必要があります")
	}
	if !e.relationType.valid() {
		return fmt.Errorf("relationType が有効ではありません")
	}
	if len(e.evidence) == 0 {
		return fmt.Errorf("evidence は一件以上必要です")
	}
	for index, evidence := range e.evidence {
		if err := evidence.Validate(); err != nil {
			return fmt.Errorf("evidence[%d] が有効ではありません: %w", index, err)
		}
	}
	return nil
}

func (e JudicialCitationEdge) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		EdgeID       string                       `json:"edgeId"`
		FromNodeID   string                       `json:"fromNodeId"`
		ToNodeID     string                       `json:"toNodeId"`
		RelationType JudicialCitationRelationType `json:"relationType"`
		Evidence     []JudicialCitationEvidence   `json:"evidence"`
	}{e.edgeID, e.fromNodeID, e.toNodeID, e.relationType, slices.Clone(e.evidence)})
}

func (*JudicialCitationEdge) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationEdge は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationEdge を使用してください",
	)
}

func mergeJudicialCitationEdges(
	values []JudicialCitationEdge,
) ([]JudicialCitationEdge, error) {
	merged := make([]JudicialCitationEdge, 0, len(values))
	indexes := make(map[judicialCitationEdgeKey]int, len(values))
	for index, edge := range values {
		if err := edge.Validate(); err != nil {
			return nil, fmt.Errorf("edges[%d] が有効ではありません: %w", index, err)
		}
		key := edge.key()
		mergedIndex, exists := indexes[key]
		if !exists {
			indexes[key] = len(merged)
			merged = append(merged, edge)
			continue
		}
		combined := slices.Concat(merged[mergedIndex].evidence, edge.evidence)
		updated, err := NewJudicialCitationEdge(JudicialCitationEdgeValues{
			EdgeID:       merged[mergedIndex].edgeID,
			FromNodeID:   edge.fromNodeID,
			ToNodeID:     edge.toNodeID,
			RelationType: edge.relationType,
			Evidence:     combined,
		})
		if err != nil {
			return nil, fmt.Errorf("重複 edge を統合できません: %w", err)
		}
		merged[mergedIndex] = updated
	}
	return merged, nil
}

// JudicialCitationEvidenceValues は、引用関係の根拠作成値を保持する。
type JudicialCitationEvidenceValues struct {
	EvidenceLevel JudicialCitationEvidenceLevel
	Provenance    Provenance
	Excerpt       *string
}

// JudicialCitationEvidence は、閉じた水準と公式出典を持つ根拠を表す。
type JudicialCitationEvidence struct {
	evidenceLevel JudicialCitationEvidenceLevel
	provenance    Provenance
	excerpt       *string
}

func NewJudicialCitationEvidence(
	values JudicialCitationEvidenceValues,
) (JudicialCitationEvidence, error) {
	evidence := JudicialCitationEvidence{
		evidenceLevel: values.EvidenceLevel,
		provenance:    values.Provenance,
		excerpt:       cloneOptionalString(values.Excerpt),
	}
	if err := evidence.Validate(); err != nil {
		return JudicialCitationEvidence{}, err
	}
	return evidence, nil
}

func (e JudicialCitationEvidence) EvidenceLevel() JudicialCitationEvidenceLevel {
	return e.evidenceLevel
}
func (e JudicialCitationEvidence) Provenance() Provenance { return e.provenance }
func (e JudicialCitationEvidence) Excerpt() (string, bool) {
	return optionalStringValue(e.excerpt)
}

func (e JudicialCitationEvidence) Validate() error {
	if !e.evidenceLevel.valid() {
		return fmt.Errorf("evidenceLevel が有効ではありません")
	}
	if err := e.provenance.Validate(); err != nil {
		return fmt.Errorf("provenance が有効ではありません: %w", err)
	}
	if e.excerpt != nil {
		if !utf8.ValidString(*e.excerpt) || *e.excerpt == "" {
			return fmt.Errorf("excerpt は非空の UTF-8 文字列でなければなりません")
		}
		if len(*e.excerpt) > judicialCitationExcerptMaxBytes {
			return fmt.Errorf("excerpt は %d byte 以下でなければなりません", judicialCitationExcerptMaxBytes)
		}
	}
	return nil
}

func (e JudicialCitationEvidence) MarshalJSON() ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		EvidenceLevel JudicialCitationEvidenceLevel `json:"evidenceLevel"`
		Provenance    Provenance                    `json:"provenance"`
		Excerpt       *string                       `json:"excerpt,omitempty"`
	}{e.evidenceLevel, e.provenance, cloneOptionalString(e.excerpt)})
}

func (*JudicialCitationEvidence) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationEvidence は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationEvidence を使用してください",
	)
}

// JudicialCitationUnresolvedMentionValues は、未解決言及の作成値を保持する。
type JudicialCitationUnresolvedMentionValues struct {
	MentionType JudicialCitationMentionType
	MentionText string
	Reason      JudicialCitationUnresolvedReason
	Provenance  Provenance
}

// JudicialCitationUnresolvedMention は、edge に昇格しなかった原文を表す。
type JudicialCitationUnresolvedMention struct {
	mentionType JudicialCitationMentionType
	mentionText string
	reason      JudicialCitationUnresolvedReason
	provenance  Provenance
}

func NewJudicialCitationUnresolvedMention(
	values JudicialCitationUnresolvedMentionValues,
) (JudicialCitationUnresolvedMention, error) {
	mention := JudicialCitationUnresolvedMention{
		mentionType: values.MentionType,
		mentionText: values.MentionText,
		reason:      values.Reason,
		provenance:  values.Provenance,
	}
	if err := mention.Validate(); err != nil {
		return JudicialCitationUnresolvedMention{}, err
	}
	return mention, nil
}

func (m JudicialCitationUnresolvedMention) MentionType() JudicialCitationMentionType {
	return m.mentionType
}
func (m JudicialCitationUnresolvedMention) MentionText() string { return m.mentionText }
func (m JudicialCitationUnresolvedMention) Reason() JudicialCitationUnresolvedReason {
	return m.reason
}
func (m JudicialCitationUnresolvedMention) Provenance() Provenance { return m.provenance }

func (m JudicialCitationUnresolvedMention) Validate() error {
	if !m.mentionType.valid() {
		return fmt.Errorf("mentionType が有効ではありません")
	}
	if !utf8.ValidString(m.mentionText) || m.mentionText == "" {
		return fmt.Errorf("mentionText は必須の UTF-8 文字列です")
	}
	if !m.reason.valid() {
		return fmt.Errorf("reason が有効ではありません")
	}
	if err := m.provenance.Validate(); err != nil {
		return fmt.Errorf("provenance が有効ではありません: %w", err)
	}
	return nil
}

func (m JudicialCitationUnresolvedMention) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		MentionType JudicialCitationMentionType      `json:"mentionType"`
		MentionText string                           `json:"mentionText"`
		Reason      JudicialCitationUnresolvedReason `json:"reason"`
		Provenance  Provenance                       `json:"provenance"`
	}{m.mentionType, m.mentionText, m.reason, m.provenance})
}

func (*JudicialCitationUnresolvedMention) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationUnresolvedMention は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationUnresolvedMention を使用してください",
	)
}
