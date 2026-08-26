package model

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// JudicialCitationNodeValues は、引用 graph ノードの作成値を保持する。
type JudicialCitationNodeValues struct {
	NodeID          string
	NodeType        JudicialCitationNodeType
	Label           string
	Ref             *SourceResourceRef
	DecisionSummary *JudicialDecisionSummary
	LawReference    *JudicialCitationLawReference
	ReferenceText   *string
}

// JudicialCitationNode は、graph 内の一意なノードを表す。
type JudicialCitationNode struct {
	nodeID          string
	nodeType        JudicialCitationNodeType
	label           string
	ref             *SourceResourceRef
	decisionSummary *JudicialDecisionSummary
	lawReference    *JudicialCitationLawReference
	referenceText   *string
}

// NewJudicialCitationNode は、条件付き field を検証した不変ノードを返す。
func NewJudicialCitationNode(values JudicialCitationNodeValues) (JudicialCitationNode, error) {
	node := JudicialCitationNode{
		nodeID:          values.NodeID,
		nodeType:        values.NodeType,
		label:           values.Label,
		ref:             cloneJudicialCitationRef(values.Ref),
		decisionSummary: cloneJudicialCitationDecisionSummary(values.DecisionSummary),
		lawReference:    cloneJudicialCitationLawReference(values.LawReference),
		referenceText:   cloneOptionalString(values.ReferenceText),
	}
	if err := node.Validate(); err != nil {
		return JudicialCitationNode{}, err
	}
	return node, nil
}

func (n JudicialCitationNode) NodeID() string                     { return n.nodeID }
func (n JudicialCitationNode) NodeType() JudicialCitationNodeType { return n.nodeType }
func (n JudicialCitationNode) Label() string                      { return n.label }
func (n JudicialCitationNode) Ref() (SourceResourceRef, bool) {
	if n.ref == nil {
		return SourceResourceRef{}, false
	}
	return *n.ref, true
}
func (n JudicialCitationNode) DecisionSummary() (JudicialDecisionSummary, bool) {
	if n.decisionSummary == nil {
		return JudicialDecisionSummary{}, false
	}
	return *n.decisionSummary, true
}
func (n JudicialCitationNode) LawReference() (JudicialCitationLawReference, bool) {
	if n.lawReference == nil {
		return JudicialCitationLawReference{}, false
	}
	return *n.lawReference, true
}
func (n JudicialCitationNode) ReferenceText() (string, bool) {
	return optionalStringValue(n.referenceText)
}

// Validate は、nodeType ごとに必須 field と禁止 field を確認する。
func (n JudicialCitationNode) Validate() error {
	if !utf8.ValidString(n.nodeID) || n.nodeID == "" {
		return fmt.Errorf("nodeId は必須の UTF-8 文字列です")
	}
	if !n.nodeType.valid() {
		return fmt.Errorf("nodeType が有効ではありません")
	}
	if !utf8.ValidString(n.label) || n.label == "" {
		return fmt.Errorf("label は必須の UTF-8 文字列です")
	}
	switch n.nodeType {
	case JudicialCitationNodeTypeDecision:
		return n.validateDecisionNode()
	case JudicialCitationNodeTypeLawProvision:
		return n.validateLawProvisionNode()
	case JudicialCitationNodeTypeDecisionReference:
		return n.validateDecisionReferenceNode()
	default:
		return fmt.Errorf("nodeType が有効ではありません")
	}
}

func (n JudicialCitationNode) validateDecisionNode() error {
	if n.ref == nil || n.decisionSummary == nil || n.lawReference != nil || n.referenceText != nil {
		return fmt.Errorf("judicial_decision node の条件付き field が不正です")
	}
	if err := n.ref.Validate(); err != nil {
		return fmt.Errorf("ref が有効ではありません: %w", err)
	}
	if n.ref.Key().ResourceType() != "judicial-decision" {
		return fmt.Errorf("judicial_decision node の ref.resourceType は judicial-decision でなければなりません")
	}
	if err := n.decisionSummary.Validate(); err != nil {
		return fmt.Errorf("decisionSummary が有効ではありません: %w", err)
	}
	return nil
}

func (n JudicialCitationNode) validateLawProvisionNode() error {
	if n.ref != nil || n.decisionSummary != nil || n.lawReference == nil || n.referenceText != nil {
		return fmt.Errorf("law_provision node の条件付き field が不正です")
	}
	if err := n.lawReference.Validate(); err != nil {
		return fmt.Errorf("lawReference が有効ではありません: %w", err)
	}
	return nil
}

func (n JudicialCitationNode) validateDecisionReferenceNode() error {
	if n.ref != nil || n.decisionSummary != nil || n.lawReference != nil || n.referenceText == nil {
		return fmt.Errorf("judicial_decision_reference node の条件付き field が不正です")
	}
	if !utf8.ValidString(*n.referenceText) || *n.referenceText == "" {
		return fmt.Errorf("referenceText は必須の UTF-8 文字列です")
	}
	return nil
}

// MarshalJSON は、nodeType に対応する field だけを出力する。
func (n JudicialCitationNode) MarshalJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		NodeID          string                        `json:"nodeId"`
		NodeType        JudicialCitationNodeType      `json:"nodeType"`
		Label           string                        `json:"label"`
		Ref             *SourceResourceRef            `json:"ref,omitempty"`
		DecisionSummary *JudicialDecisionSummary      `json:"decisionSummary,omitempty"`
		LawReference    *JudicialCitationLawReference `json:"lawReference,omitempty"`
		ReferenceText   *string                       `json:"referenceText,omitempty"`
	}{
		n.nodeID,
		n.nodeType,
		n.label,
		cloneJudicialCitationRef(n.ref),
		cloneJudicialCitationDecisionSummary(n.decisionSummary),
		cloneJudicialCitationLawReference(n.lawReference),
		cloneOptionalString(n.referenceText),
	})
}

func (*JudicialCitationNode) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationNode は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationNode を使用してください",
	)
}

// JudicialCitationLawReferenceValues は、一意に解決した法令参照の作成値を保持する。
type JudicialCitationLawReferenceValues struct {
	LawID    string
	LawTitle string
	Location LawArticleLocation
}

// JudicialCitationLawReference は、一意に解決した法令と条文位置を表す。
type JudicialCitationLawReference struct {
	lawID    string
	lawTitle string
	location LawArticleLocation
}

func NewJudicialCitationLawReference(
	values JudicialCitationLawReferenceValues,
) (JudicialCitationLawReference, error) {
	reference := JudicialCitationLawReference{
		lawID: values.LawID, lawTitle: values.LawTitle, location: values.Location,
	}
	if err := reference.Validate(); err != nil {
		return JudicialCitationLawReference{}, err
	}
	return reference, nil
}

func (r JudicialCitationLawReference) LawID() string                { return r.lawID }
func (r JudicialCitationLawReference) LawTitle() string             { return r.lawTitle }
func (r JudicialCitationLawReference) Location() LawArticleLocation { return r.location }

func (r JudicialCitationLawReference) Validate() error {
	if !utf8.ValidString(r.lawID) || r.lawID == "" {
		return fmt.Errorf("lawId は必須の UTF-8 文字列です")
	}
	if !utf8.ValidString(r.lawTitle) || r.lawTitle == "" {
		return fmt.Errorf("lawTitle は必須の UTF-8 文字列です")
	}
	if err := r.location.Validate(); err != nil {
		return fmt.Errorf("location が有効ではありません: %w", err)
	}
	return nil
}

func (r JudicialCitationLawReference) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		LawID    string             `json:"lawId"`
		LawTitle string             `json:"lawTitle"`
		Location LawArticleLocation `json:"location"`
	}{r.lawID, r.lawTitle, r.location})
}

func (*JudicialCitationLawReference) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationLawReference は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationLawReference を使用してください",
	)
}

func cloneJudicialCitationRef(value *SourceResourceRef) *SourceResourceRef {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneJudicialCitationDecisionSummary(value *JudicialDecisionSummary) *JudicialDecisionSummary {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneJudicialCitationLawReference(
	value *JudicialCitationLawReference,
) *JudicialCitationLawReference {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
