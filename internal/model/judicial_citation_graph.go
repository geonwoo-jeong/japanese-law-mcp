package model

import (
	"encoding/json"
	"fmt"
	"slices"
	"unicode/utf8"
)

const judicialCitationExcerptMaxBytes = 256

type JudicialCitationResultStatus string

const (
	JudicialCitationResultStatusComplete JudicialCitationResultStatus = "complete"
	JudicialCitationResultStatusPartial  JudicialCitationResultStatus = "partial"
)

type JudicialCitationNodeType string

const (
	JudicialCitationNodeTypeDecision          JudicialCitationNodeType = "judicial_decision"
	JudicialCitationNodeTypeLawProvision      JudicialCitationNodeType = "law_provision"
	JudicialCitationNodeTypeDecisionReference JudicialCitationNodeType = "judicial_decision_reference"
)

type JudicialCitationRelationType string

const (
	JudicialCitationRelationTypeCitesDecision          JudicialCitationRelationType = "cites_judicial_decision"
	JudicialCitationRelationTypePossibleCitesDecision  JudicialCitationRelationType = "possible_cites_judicial_decision"
	JudicialCitationRelationTypeReferencesLawProvision JudicialCitationRelationType = "references_law_provision"
	JudicialCitationRelationTypeHasLowerCourtDecision  JudicialCitationRelationType = "has_lower_court_decision"
)

type JudicialCitationEvidenceLevel string

const (
	JudicialCitationEvidenceLevelOfficialMetadata        JudicialCitationEvidenceLevel = "official_metadata"
	JudicialCitationEvidenceLevelExactTextMatch          JudicialCitationEvidenceLevel = "exact_text_match"
	JudicialCitationEvidenceLevelOfficialSearchCandidate JudicialCitationEvidenceLevel = "official_search_candidate"
)

type JudicialCitationMentionType string

const (
	JudicialCitationMentionTypeDecision     JudicialCitationMentionType = "judicial_decision"
	JudicialCitationMentionTypeLawProvision JudicialCitationMentionType = "law_provision"
)

type JudicialCitationUnresolvedReason string

const (
	JudicialCitationUnresolvedReasonAmbiguousTarget        JudicialCitationUnresolvedReason = "ambiguous_target"
	JudicialCitationUnresolvedReasonNoPublishedTargetMatch JudicialCitationUnresolvedReason = "no_published_target_match"
	JudicialCitationUnresolvedReasonInsufficientIdentity   JudicialCitationUnresolvedReason = "insufficient_identity"
	JudicialCitationUnresolvedReasonUnsupportedReference   JudicialCitationUnresolvedReason = "unsupported_reference_form"
	JudicialCitationUnresolvedReasonUnregisteredLawName    JudicialCitationUnresolvedReason = "unregistered_law_name"
	JudicialCitationUnresolvedReasonAmbiguousLawLocation   JudicialCitationUnresolvedReason = "ambiguous_law_location"
	JudicialCitationUnresolvedReasonFuzzyMatchOnly         JudicialCitationUnresolvedReason = "fuzzy_match_only"
)

type JudicialCitationRequestedDirection string

const (
	JudicialCitationRequestedDirectionOutgoing JudicialCitationRequestedDirection = "outgoing"
	JudicialCitationRequestedDirectionIncoming JudicialCitationRequestedDirection = "incoming"
	JudicialCitationRequestedDirectionBoth     JudicialCitationRequestedDirection = "both"
)

type JudicialCitationDirectionStatus string

const (
	JudicialCitationDirectionStatusComplete     JudicialCitationDirectionStatus = "complete"
	JudicialCitationDirectionStatusPartial      JudicialCitationDirectionStatus = "partial"
	JudicialCitationDirectionStatusUnavailable  JudicialCitationDirectionStatus = "unavailable"
	JudicialCitationDirectionStatusNotRequested JudicialCitationDirectionStatus = "not_requested"
)

type JudicialCitationMethod string

const (
	JudicialCitationMethodOfficialDetailMetadata JudicialCitationMethod = "official_detail_metadata"
	JudicialCitationMethodOfficialPDFText        JudicialCitationMethod = "official_pdf_text"
	JudicialCitationMethodOfficialCaseSearch     JudicialCitationMethod = "official_case_search"
)

type JudicialCitationIssueDirection string

const (
	JudicialCitationIssueDirectionOutgoing JudicialCitationIssueDirection = "outgoing"
	JudicialCitationIssueDirectionIncoming JudicialCitationIssueDirection = "incoming"
	JudicialCitationIssueDirectionShared   JudicialCitationIssueDirection = "shared"
)

type JudicialCitationIssueStage string

const (
	JudicialCitationIssueStageOfficialDetailMetadata JudicialCitationIssueStage = "official_detail_metadata"
	JudicialCitationIssueStageOfficialPDFText        JudicialCitationIssueStage = "official_pdf_text"
	JudicialCitationIssueStageOfficialCaseSearch     JudicialCitationIssueStage = "official_case_search"
	JudicialCitationIssueStageLawReferenceResolution JudicialCitationIssueStage = "law_reference_resolution"
)

type JudicialCitationGraphResultValues struct {
	Status         JudicialCitationResultStatus
	CoverageNotice string
	Graph          JudicialCitationGraph
	Issues         []JudicialCitationIssue
}

type JudicialCitationGraphResult struct {
	status         JudicialCitationResultStatus
	coverageNotice string
	graph          JudicialCitationGraph
	issues         []JudicialCitationIssue
}

func NewJudicialCitationGraphResult(
	values JudicialCitationGraphResultValues,
) (JudicialCitationGraphResult, error) {
	result := JudicialCitationGraphResult{
		status:         values.Status,
		coverageNotice: values.CoverageNotice,
		graph:          values.Graph,
		issues:         cloneJudicialCitationIssues(values.Issues),
	}
	if err := result.Validate(); err != nil {
		return JudicialCitationGraphResult{}, err
	}
	return result, nil
}

func (r JudicialCitationGraphResult) Status() JudicialCitationResultStatus { return r.status }
func (r JudicialCitationGraphResult) CoverageNotice() string               { return r.coverageNotice }
func (r JudicialCitationGraphResult) Graph() JudicialCitationGraph         { return r.graph }
func (r JudicialCitationGraphResult) Issues() []JudicialCitationIssue {
	return cloneJudicialCitationIssues(r.issues)
}

func (r JudicialCitationGraphResult) Validate() error {
	if !r.status.valid() {
		return fmt.Errorf("status が有効ではありません")
	}
	if !utf8.ValidString(r.coverageNotice) || r.coverageNotice == "" {
		return fmt.Errorf("coverageNotice は必須の UTF-8 文字列です")
	}
	if err := r.graph.Validate(); err != nil {
		return fmt.Errorf("graph が有効ではありません: %w", err)
	}
	for index, issue := range r.issues {
		if err := issue.Validate(); err != nil {
			return fmt.Errorf("issues[%d] が有効ではありません: %w", index, err)
		}
	}
	if r.status == JudicialCitationResultStatusPartial && len(r.issues) == 0 {
		return fmt.Errorf("partial の status では issues が必須です")
	}
	return nil
}

func (r JudicialCitationGraphResult) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Status         JudicialCitationResultStatus `json:"status"`
		CoverageNotice string                       `json:"coverageNotice"`
		Graph          JudicialCitationGraph        `json:"graph"`
		Issues         []JudicialCitationIssue      `json:"issues"`
	}{
		Status:         r.status,
		CoverageNotice: r.coverageNotice,
		Graph:          r.graph,
		Issues:         cloneJudicialCitationIssues(r.issues),
	})
}

func (*JudicialCitationGraphResult) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationGraphResult は JSON から直接復元できません。")
}

type JudicialCitationGraphValues struct {
	RootNodeID         string
	Nodes              []JudicialCitationNode
	Edges              []JudicialCitationEdge
	UnresolvedMentions []JudicialCitationUnresolvedMention
	Summary            JudicialCitationSummary
	Coverage           JudicialCitationCoverage
}

type JudicialCitationGraph struct {
	rootNodeID         string
	nodes              []JudicialCitationNode
	edges              []JudicialCitationEdge
	unresolvedMentions []JudicialCitationUnresolvedMention
	summary            JudicialCitationSummary
	coverage           JudicialCitationCoverage
}

func NewJudicialCitationGraph(
	values JudicialCitationGraphValues,
) (JudicialCitationGraph, error) {
	graph := JudicialCitationGraph{
		rootNodeID:         values.RootNodeID,
		nodes:              cloneJudicialCitationNodes(values.Nodes),
		edges:              cloneJudicialCitationEdges(values.Edges),
		unresolvedMentions: cloneJudicialCitationUnresolvedMentions(values.UnresolvedMentions),
		summary:            values.Summary,
		coverage:           values.Coverage,
	}
	if err := graph.Validate(); err != nil {
		return JudicialCitationGraph{}, err
	}
	return graph, nil
}

func (g JudicialCitationGraph) RootNodeID() string { return g.rootNodeID }
func (g JudicialCitationGraph) Nodes() []JudicialCitationNode {
	return cloneJudicialCitationNodes(g.nodes)
}
func (g JudicialCitationGraph) Edges() []JudicialCitationEdge {
	return cloneJudicialCitationEdges(g.edges)
}
func (g JudicialCitationGraph) UnresolvedMentions() []JudicialCitationUnresolvedMention {
	return cloneJudicialCitationUnresolvedMentions(g.unresolvedMentions)
}
func (g JudicialCitationGraph) Summary() JudicialCitationSummary   { return g.summary }
func (g JudicialCitationGraph) Coverage() JudicialCitationCoverage { return g.coverage }

func (g JudicialCitationGraph) Validate() error {
	if !utf8.ValidString(g.rootNodeID) || g.rootNodeID == "" {
		return fmt.Errorf("rootNodeId は必須の UTF-8 文字列です")
	}
	if g.nodes == nil || g.edges == nil || g.unresolvedMentions == nil {
		return fmt.Errorf("nodes、edges および unresolvedMentions は nil にできません")
	}
	nodeIDs := make(map[string]struct{}, len(g.nodes))
	for index, node := range g.nodes {
		if err := node.Validate(); err != nil {
			return fmt.Errorf("nodes[%d] が有効ではありません: %w", index, err)
		}
		id := node.NodeID()
		if _, exists := nodeIDs[id]; exists {
			return fmt.Errorf("nodes に重複した nodeId があります")
		}
		nodeIDs[id] = struct{}{}
	}
	if _, exists := nodeIDs[g.rootNodeID]; !exists {
		return fmt.Errorf("rootNodeId が nodes に存在しません")
	}
	for index, edge := range g.edges {
		if err := edge.Validate(); err != nil {
			return fmt.Errorf("edges[%d] が有効ではありません: %w", index, err)
		}
		if _, exists := nodeIDs[edge.FromNodeID()]; !exists {
			return fmt.Errorf("edges[%d].fromNodeId が nodes に存在しません", index)
		}
		if _, exists := nodeIDs[edge.ToNodeID()]; !exists {
			return fmt.Errorf("edges[%d].toNodeId が nodes に存在しません", index)
		}
	}
	for index, mention := range g.unresolvedMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("unresolvedMentions[%d] が有効ではありません: %w", index, err)
		}
	}
	if err := g.summary.Validate(); err != nil {
		return fmt.Errorf("summary が有効ではありません: %w", err)
	}
	if err := g.coverage.Validate(); err != nil {
		return fmt.Errorf("coverage が有効ではありません: %w", err)
	}
	return nil
}

func (g JudicialCitationGraph) MarshalJSON() ([]byte, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		RootNodeID         string                              `json:"rootNodeId"`
		Nodes              []JudicialCitationNode              `json:"nodes"`
		Edges              []JudicialCitationEdge              `json:"edges"`
		UnresolvedMentions []JudicialCitationUnresolvedMention `json:"unresolvedMentions"`
		Summary            JudicialCitationSummary             `json:"summary"`
		Coverage           JudicialCitationCoverage            `json:"coverage"`
	}{
		RootNodeID:         g.rootNodeID,
		Nodes:              cloneJudicialCitationNodes(g.nodes),
		Edges:              cloneJudicialCitationEdges(g.edges),
		UnresolvedMentions: cloneJudicialCitationUnresolvedMentions(g.unresolvedMentions),
		Summary:            g.summary,
		Coverage:           g.coverage,
	})
}

func (*JudicialCitationGraph) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationGraph は JSON から直接復元できません。")
}

type JudicialCitationNodeValues struct {
	NodeID          string
	NodeType        JudicialCitationNodeType
	Label           string
	Ref             *SourceResourceRef
	DecisionSummary *JudicialDecisionSummary
	LawReference    *JudicialCitationLawReference
	ReferenceText   *string
}

type JudicialCitationNode struct {
	nodeID          string
	nodeType        JudicialCitationNodeType
	label           string
	ref             *SourceResourceRef
	decisionSummary *JudicialDecisionSummary
	lawReference    *JudicialCitationLawReference
	referenceText   *string
}

func NewJudicialCitationNode(values JudicialCitationNodeValues) (JudicialCitationNode, error) {
	node := JudicialCitationNode{
		nodeID:   values.NodeID,
		nodeType: values.NodeType,
		label:    values.Label,
	}
	if values.Ref != nil {
		ref := *values.Ref
		node.ref = &ref
	}
	if values.DecisionSummary != nil {
		summary := *values.DecisionSummary
		node.decisionSummary = &summary
	}
	if values.LawReference != nil {
		reference := *values.LawReference
		node.lawReference = &reference
	}
	node.referenceText = cloneOptionalString(values.ReferenceText)
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
		if n.ref == nil || n.decisionSummary == nil ||
			n.lawReference != nil || n.referenceText != nil {
			return fmt.Errorf("judicial_decision node の構造が不正です")
		}
		if err := n.ref.Validate(); err != nil {
			return fmt.Errorf("ref が有効ではありません: %w", err)
		}
		if err := n.decisionSummary.Validate(); err != nil {
			return fmt.Errorf("decisionSummary が有効ではありません: %w", err)
		}
	case JudicialCitationNodeTypeLawProvision:
		if n.ref != nil || n.decisionSummary != nil ||
			n.lawReference == nil || n.referenceText != nil {
			return fmt.Errorf("law_provision node の構造が不正です")
		}
		if err := n.lawReference.Validate(); err != nil {
			return fmt.Errorf("lawReference が有効ではありません: %w", err)
		}
	case JudicialCitationNodeTypeDecisionReference:
		if n.ref != nil || n.decisionSummary != nil ||
			n.lawReference != nil || n.referenceText == nil {
			return fmt.Errorf("judicial_decision_reference node の構造が不正です")
		}
		if !utf8.ValidString(*n.referenceText) || *n.referenceText == "" {
			return fmt.Errorf("referenceText は必須の UTF-8 文字列です")
		}
	}
	return nil
}

func (n JudicialCitationNode) MarshalJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	var ref *SourceResourceRef
	if n.ref != nil {
		cloned := *n.ref
		ref = &cloned
	}
	var summary *JudicialDecisionSummary
	if n.decisionSummary != nil {
		cloned := *n.decisionSummary
		summary = &cloned
	}
	var lawReference *JudicialCitationLawReference
	if n.lawReference != nil {
		cloned := *n.lawReference
		lawReference = &cloned
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
		NodeID:          n.nodeID,
		NodeType:        n.nodeType,
		Label:           n.label,
		Ref:             ref,
		DecisionSummary: summary,
		LawReference:    lawReference,
		ReferenceText:   cloneOptionalString(n.referenceText),
	})
}

func (*JudicialCitationNode) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationNode は JSON から直接復元できません。")
}

type JudicialCitationLawReferenceValues struct {
	LawID    string
	LawTitle string
	Location LawArticleLocation
}

type JudicialCitationLawReference struct {
	lawID    string
	lawTitle string
	location LawArticleLocation
}

func NewJudicialCitationLawReference(
	values JudicialCitationLawReferenceValues,
) (JudicialCitationLawReference, error) {
	reference := JudicialCitationLawReference{
		lawID:    values.LawID,
		lawTitle: values.LawTitle,
		location: values.Location,
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
	}{
		LawID:    r.lawID,
		LawTitle: r.lawTitle,
		Location: r.location,
	})
}

func (*JudicialCitationLawReference) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationLawReference は JSON から直接復元できません。")
}

type JudicialCitationEdgeValues struct {
	EdgeID       string
	FromNodeID   string
	ToNodeID     string
	RelationType JudicialCitationRelationType
	Evidence     []JudicialCitationEvidence
}

type JudicialCitationEdge struct {
	edgeID       string
	fromNodeID   string
	toNodeID     string
	relationType JudicialCitationRelationType
	evidence     []JudicialCitationEvidence
}

func NewJudicialCitationEdge(values JudicialCitationEdgeValues) (JudicialCitationEdge, error) {
	edge := JudicialCitationEdge{
		edgeID:       values.EdgeID,
		fromNodeID:   values.FromNodeID,
		toNodeID:     values.ToNodeID,
		relationType: values.RelationType,
		evidence:     cloneJudicialCitationEvidence(values.Evidence),
	}
	if err := edge.Validate(); err != nil {
		return JudicialCitationEdge{}, err
	}
	return edge, nil
}

func (e JudicialCitationEdge) EdgeID() string                             { return e.edgeID }
func (e JudicialCitationEdge) FromNodeID() string                         { return e.fromNodeID }
func (e JudicialCitationEdge) ToNodeID() string                           { return e.toNodeID }
func (e JudicialCitationEdge) RelationType() JudicialCitationRelationType { return e.relationType }
func (e JudicialCitationEdge) Evidence() []JudicialCitationEvidence {
	return cloneJudicialCitationEvidence(e.evidence)
}

func (e JudicialCitationEdge) Validate() error {
	if !utf8.ValidString(e.edgeID) || e.edgeID == "" {
		return fmt.Errorf("edgeId は必須の UTF-8 文字列です")
	}
	if !utf8.ValidString(e.fromNodeID) || e.fromNodeID == "" ||
		!utf8.ValidString(e.toNodeID) || e.toNodeID == "" {
		return fmt.Errorf("fromNodeId と toNodeId は必須の UTF-8 文字列です")
	}
	if !e.relationType.valid() {
		return fmt.Errorf("relationType が有効ではありません")
	}
	if e.evidence == nil || len(e.evidence) == 0 {
		return fmt.Errorf("evidence は一件以上必須です")
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
	}{
		EdgeID:       e.edgeID,
		FromNodeID:   e.fromNodeID,
		ToNodeID:     e.toNodeID,
		RelationType: e.relationType,
		Evidence:     cloneJudicialCitationEvidence(e.evidence),
	})
}

func (*JudicialCitationEdge) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationEdge は JSON から直接復元できません。")
}

type JudicialCitationEvidenceValues struct {
	EvidenceLevel JudicialCitationEvidenceLevel
	Provenance    Provenance
	Excerpt       *string
}

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
			return fmt.Errorf("excerpt は 256 byte 以下でなければなりません")
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
	}{
		EvidenceLevel: e.evidenceLevel,
		Provenance:    e.provenance,
		Excerpt:       cloneOptionalString(e.excerpt),
	})
}

func (*JudicialCitationEvidence) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationEvidence は JSON から直接復元できません。")
}

type JudicialCitationUnresolvedMentionValues struct {
	MentionType JudicialCitationMentionType
	MentionText string
	Reason      JudicialCitationUnresolvedReason
	Provenance  Provenance
}

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
	}{
		MentionType: m.mentionType,
		MentionText: m.mentionText,
		Reason:      m.reason,
		Provenance:  m.provenance,
	})
}

func (*JudicialCitationUnresolvedMention) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationUnresolvedMention は JSON から直接復元できません。")
}

type JudicialCitationSummaryValues struct {
	ConfirmedOutgoingDecisionCount  int
	IncomingCandidateCount          int
	ReferencedProvisionCount        int
	LowerCourtRelationCount         int
	UnresolvedMentionCount          int
	IncomingObservedYearBuckets     []JudicialCitationYearBucket
	IncomingObservedCategoryBuckets []JudicialCitationCategoryBucket
}

type JudicialCitationSummary struct {
	confirmedOutgoingDecisionCount  int
	incomingCandidateCount          int
	referencedProvisionCount        int
	lowerCourtRelationCount         int
	unresolvedMentionCount          int
	incomingObservedYearBuckets     []JudicialCitationYearBucket
	incomingObservedCategoryBuckets []JudicialCitationCategoryBucket
}

func NewJudicialCitationSummary(values JudicialCitationSummaryValues) (JudicialCitationSummary, error) {
	summary := JudicialCitationSummary{
		confirmedOutgoingDecisionCount:  values.ConfirmedOutgoingDecisionCount,
		incomingCandidateCount:          values.IncomingCandidateCount,
		referencedProvisionCount:        values.ReferencedProvisionCount,
		lowerCourtRelationCount:         values.LowerCourtRelationCount,
		unresolvedMentionCount:          values.UnresolvedMentionCount,
		incomingObservedYearBuckets:     cloneJudicialCitationYearBuckets(values.IncomingObservedYearBuckets),
		incomingObservedCategoryBuckets: cloneJudicialCitationCategoryBuckets(values.IncomingObservedCategoryBuckets),
	}
	if err := summary.Validate(); err != nil {
		return JudicialCitationSummary{}, err
	}
	return summary, nil
}

func (s JudicialCitationSummary) Validate() error {
	for _, value := range []int{
		s.confirmedOutgoingDecisionCount,
		s.incomingCandidateCount,
		s.referencedProvisionCount,
		s.lowerCourtRelationCount,
		s.unresolvedMentionCount,
	} {
		if value < 0 {
			return fmt.Errorf("summary の件数は 0 以上でなければなりません")
		}
	}
	if s.incomingObservedYearBuckets == nil || s.incomingObservedCategoryBuckets == nil {
		return fmt.Errorf("summary の bucket 配列は nil にできません")
	}
	for index, bucket := range s.incomingObservedYearBuckets {
		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("incomingObservedYearBuckets[%d] が有効ではありません: %w", index, err)
		}
	}
	for index, bucket := range s.incomingObservedCategoryBuckets {
		if err := bucket.Validate(); err != nil {
			return fmt.Errorf("incomingObservedCategoryBuckets[%d] が有効ではありません: %w", index, err)
		}
	}
	return nil
}

func (s JudicialCitationSummary) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ConfirmedOutgoingDecisionCount  int                              `json:"confirmedOutgoingDecisionCount"`
		IncomingCandidateCount          int                              `json:"incomingCandidateCount"`
		ReferencedProvisionCount        int                              `json:"referencedProvisionCount"`
		LowerCourtRelationCount         int                              `json:"lowerCourtRelationCount"`
		UnresolvedMentionCount          int                              `json:"unresolvedMentionCount"`
		IncomingObservedYearBuckets     []JudicialCitationYearBucket     `json:"incomingObservedYearBuckets"`
		IncomingObservedCategoryBuckets []JudicialCitationCategoryBucket `json:"incomingObservedCategoryBuckets"`
	}{
		ConfirmedOutgoingDecisionCount:  s.confirmedOutgoingDecisionCount,
		IncomingCandidateCount:          s.incomingCandidateCount,
		ReferencedProvisionCount:        s.referencedProvisionCount,
		LowerCourtRelationCount:         s.lowerCourtRelationCount,
		UnresolvedMentionCount:          s.unresolvedMentionCount,
		IncomingObservedYearBuckets:     cloneJudicialCitationYearBuckets(s.incomingObservedYearBuckets),
		IncomingObservedCategoryBuckets: cloneJudicialCitationCategoryBuckets(s.incomingObservedCategoryBuckets),
	})
}

func (*JudicialCitationSummary) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationSummary は JSON から直接復元できません。")
}

type JudicialCitationYearBucket struct {
	year  int
	count int
}

func NewJudicialCitationYearBucket(year, count int) (JudicialCitationYearBucket, error) {
	bucket := JudicialCitationYearBucket{year: year, count: count}
	if err := bucket.Validate(); err != nil {
		return JudicialCitationYearBucket{}, err
	}
	return bucket, nil
}

func (b JudicialCitationYearBucket) Year() int  { return b.year }
func (b JudicialCitationYearBucket) Count() int { return b.count }
func (b JudicialCitationYearBucket) Validate() error {
	if b.year < 1 || b.count < 0 {
		return fmt.Errorf("year bucket が有効ではありません")
	}
	return nil
}
func (b JudicialCitationYearBucket) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Year  int `json:"year"`
		Count int `json:"count"`
	}{Year: b.year, Count: b.count})
}
func (*JudicialCitationYearBucket) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationYearBucket は JSON から直接復元できません。")
}

type JudicialCitationCategoryBucket struct {
	publicationCategory JudicialPublicationCategory
	count               int
}

func NewJudicialCitationCategoryBucket(
	publicationCategory JudicialPublicationCategory,
	count int,
) (JudicialCitationCategoryBucket, error) {
	bucket := JudicialCitationCategoryBucket{
		publicationCategory: publicationCategory,
		count:               count,
	}
	if err := bucket.Validate(); err != nil {
		return JudicialCitationCategoryBucket{}, err
	}
	return bucket, nil
}

func (b JudicialCitationCategoryBucket) PublicationCategory() JudicialPublicationCategory {
	return b.publicationCategory
}
func (b JudicialCitationCategoryBucket) Count() int { return b.count }
func (b JudicialCitationCategoryBucket) Validate() error {
	if !b.publicationCategory.valid() || b.count < 0 {
		return fmt.Errorf("category bucket が有効ではありません")
	}
	return nil
}
func (b JudicialCitationCategoryBucket) MarshalJSON() ([]byte, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		PublicationCategory JudicialPublicationCategory `json:"publicationCategory"`
		Count               int                         `json:"count"`
	}{PublicationCategory: b.publicationCategory, Count: b.count})
}
func (*JudicialCitationCategoryBucket) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationCategoryBucket は JSON から直接復元できません。")
}

type JudicialCitationCoverageValues struct {
	RequestedDirection JudicialCitationRequestedDirection
	Outgoing           JudicialCitationDirectionCoverage
	Incoming           JudicialCitationDirectionCoverage
}

type JudicialCitationCoverage struct {
	requestedDirection JudicialCitationRequestedDirection
	outgoing           JudicialCitationDirectionCoverage
	incoming           JudicialCitationDirectionCoverage
}

func NewJudicialCitationCoverage(values JudicialCitationCoverageValues) (JudicialCitationCoverage, error) {
	coverage := JudicialCitationCoverage{
		requestedDirection: values.RequestedDirection,
		outgoing:           values.Outgoing,
		incoming:           values.Incoming,
	}
	if err := coverage.Validate(); err != nil {
		return JudicialCitationCoverage{}, err
	}
	return coverage, nil
}

func (c JudicialCitationCoverage) Validate() error {
	if !c.requestedDirection.valid() {
		return fmt.Errorf("requestedDirection が有効ではありません")
	}
	if err := c.outgoing.Validate(); err != nil {
		return fmt.Errorf("outgoing が有効ではありません: %w", err)
	}
	if err := c.incoming.Validate(); err != nil {
		return fmt.Errorf("incoming が有効ではありません: %w", err)
	}
	return nil
}

func (c JudicialCitationCoverage) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		RequestedDirection JudicialCitationRequestedDirection `json:"requestedDirection"`
		HopDepth           int                                `json:"hopDepth"`
		Outgoing           JudicialCitationDirectionCoverage  `json:"outgoing"`
		Incoming           JudicialCitationDirectionCoverage  `json:"incoming"`
	}{
		RequestedDirection: c.requestedDirection,
		HopDepth:           1,
		Outgoing:           c.outgoing,
		Incoming:           c.incoming,
	})
}

func (*JudicialCitationCoverage) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationCoverage は JSON から直接復元できません。")
}

type JudicialCitationDirectionCoverageValues struct {
	Status            JudicialCitationDirectionStatus
	Methods           []JudicialCitationMethod
	Truncated         bool
	Limit             *int
	AttemptedSearches *int
	CompletedSearches *int
}

type JudicialCitationDirectionCoverage struct {
	status            JudicialCitationDirectionStatus
	methods           []JudicialCitationMethod
	truncated         bool
	limit             *int
	attemptedSearches *int
	completedSearches *int
}

func NewJudicialCitationDirectionCoverage(
	values JudicialCitationDirectionCoverageValues,
) (JudicialCitationDirectionCoverage, error) {
	coverage := JudicialCitationDirectionCoverage{
		status:            values.Status,
		methods:           slices.Clone(values.Methods),
		truncated:         values.Truncated,
		limit:             cloneOptionalInt(values.Limit),
		attemptedSearches: cloneOptionalInt(values.AttemptedSearches),
		completedSearches: cloneOptionalInt(values.CompletedSearches),
	}
	if err := coverage.Validate(); err != nil {
		return JudicialCitationDirectionCoverage{}, err
	}
	return coverage, nil
}

func (c JudicialCitationDirectionCoverage) Validate() error {
	if !c.status.valid() {
		return fmt.Errorf("status が有効ではありません")
	}
	if c.methods == nil {
		return fmt.Errorf("methods は nil にできません")
	}
	for _, method := range c.methods {
		if !method.valid() {
			return fmt.Errorf("methods に不正な値があります")
		}
	}
	if c.limit != nil && (*c.limit < 1 || *c.limit > 10) {
		return fmt.Errorf("limit は 1 以上 10 以下でなければなりません")
	}
	if c.attemptedSearches != nil && (*c.attemptedSearches < 0 || *c.attemptedSearches > 2) {
		return fmt.Errorf("attemptedSearches は 0 以上 2 以下でなければなりません")
	}
	if c.completedSearches != nil && *c.completedSearches < 0 {
		return fmt.Errorf("completedSearches は 0 以上でなければなりません")
	}
	if c.attemptedSearches == nil && c.completedSearches != nil {
		return fmt.Errorf("completedSearches だけを指定できません")
	}
	if c.attemptedSearches != nil && c.completedSearches != nil &&
		*c.completedSearches > *c.attemptedSearches {
		return fmt.Errorf("completedSearches は attemptedSearches を超えられません")
	}
	if c.status == JudicialCitationDirectionStatusNotRequested && len(c.methods) != 0 {
		return fmt.Errorf("not_requested では methods は空でなければなりません")
	}
	return nil
}

func (c JudicialCitationDirectionCoverage) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Status            JudicialCitationDirectionStatus `json:"status"`
		Methods           []JudicialCitationMethod        `json:"methods"`
		Truncated         bool                            `json:"truncated"`
		Limit             *int                            `json:"limit,omitempty"`
		AttemptedSearches *int                            `json:"attemptedSearches,omitempty"`
		CompletedSearches *int                            `json:"completedSearches,omitempty"`
	}{
		Status:            c.status,
		Methods:           slices.Clone(c.methods),
		Truncated:         c.truncated,
		Limit:             cloneOptionalInt(c.limit),
		AttemptedSearches: cloneOptionalInt(c.attemptedSearches),
		CompletedSearches: cloneOptionalInt(c.completedSearches),
	})
}

func (*JudicialCitationDirectionCoverage) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationDirectionCoverage は JSON から直接復元できません。")
}

type JudicialCitationIssueValues struct {
	Direction JudicialCitationIssueDirection
	Stage     JudicialCitationIssueStage
	Code      string
	Message   string
	Retryable bool
}

type JudicialCitationIssue struct {
	direction JudicialCitationIssueDirection
	stage     JudicialCitationIssueStage
	code      string
	message   string
	retryable bool
}

func NewJudicialCitationIssue(values JudicialCitationIssueValues) (JudicialCitationIssue, error) {
	issue := JudicialCitationIssue{
		direction: values.Direction,
		stage:     values.Stage,
		code:      values.Code,
		message:   values.Message,
		retryable: values.Retryable,
	}
	if err := issue.Validate(); err != nil {
		return JudicialCitationIssue{}, err
	}
	return issue, nil
}

func (i JudicialCitationIssue) Validate() error {
	if !i.direction.valid() {
		return fmt.Errorf("direction が有効ではありません")
	}
	if !i.stage.valid() {
		return fmt.Errorf("stage が有効ではありません")
	}
	if !utf8.ValidString(i.code) || i.code == "" {
		return fmt.Errorf("code は必須の UTF-8 文字列です")
	}
	if !utf8.ValidString(i.message) || i.message == "" {
		return fmt.Errorf("message は必須の UTF-8 文字列です")
	}
	return nil
}

func (i JudicialCitationIssue) MarshalJSON() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Direction JudicialCitationIssueDirection `json:"direction"`
		Stage     JudicialCitationIssueStage     `json:"stage"`
		Code      string                         `json:"code"`
		Message   string                         `json:"message"`
		Retryable bool                           `json:"retryable"`
	}{
		Direction: i.direction,
		Stage:     i.stage,
		Code:      i.code,
		Message:   i.message,
		Retryable: i.retryable,
	})
}

func (*JudicialCitationIssue) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("JudicialCitationIssue は JSON から直接復元できません。")
}

func (s JudicialCitationResultStatus) valid() bool {
	return s == JudicialCitationResultStatusComplete ||
		s == JudicialCitationResultStatusPartial
}

func (t JudicialCitationNodeType) valid() bool {
	return t == JudicialCitationNodeTypeDecision ||
		t == JudicialCitationNodeTypeLawProvision ||
		t == JudicialCitationNodeTypeDecisionReference
}

func (t JudicialCitationRelationType) valid() bool {
	return t == JudicialCitationRelationTypeCitesDecision ||
		t == JudicialCitationRelationTypePossibleCitesDecision ||
		t == JudicialCitationRelationTypeReferencesLawProvision ||
		t == JudicialCitationRelationTypeHasLowerCourtDecision
}

func (l JudicialCitationEvidenceLevel) valid() bool {
	return l == JudicialCitationEvidenceLevelOfficialMetadata ||
		l == JudicialCitationEvidenceLevelExactTextMatch ||
		l == JudicialCitationEvidenceLevelOfficialSearchCandidate
}

func (t JudicialCitationMentionType) valid() bool {
	return t == JudicialCitationMentionTypeDecision ||
		t == JudicialCitationMentionTypeLawProvision
}

func (r JudicialCitationUnresolvedReason) valid() bool {
	return r == JudicialCitationUnresolvedReasonAmbiguousTarget ||
		r == JudicialCitationUnresolvedReasonNoPublishedTargetMatch ||
		r == JudicialCitationUnresolvedReasonInsufficientIdentity ||
		r == JudicialCitationUnresolvedReasonUnsupportedReference ||
		r == JudicialCitationUnresolvedReasonUnregisteredLawName ||
		r == JudicialCitationUnresolvedReasonAmbiguousLawLocation ||
		r == JudicialCitationUnresolvedReasonFuzzyMatchOnly
}

func (d JudicialCitationRequestedDirection) valid() bool {
	return d == JudicialCitationRequestedDirectionOutgoing ||
		d == JudicialCitationRequestedDirectionIncoming ||
		d == JudicialCitationRequestedDirectionBoth
}

func (s JudicialCitationDirectionStatus) valid() bool {
	return s == JudicialCitationDirectionStatusComplete ||
		s == JudicialCitationDirectionStatusPartial ||
		s == JudicialCitationDirectionStatusUnavailable ||
		s == JudicialCitationDirectionStatusNotRequested
}

func (m JudicialCitationMethod) valid() bool {
	return m == JudicialCitationMethodOfficialDetailMetadata ||
		m == JudicialCitationMethodOfficialPDFText ||
		m == JudicialCitationMethodOfficialCaseSearch
}

func (d JudicialCitationIssueDirection) valid() bool {
	return d == JudicialCitationIssueDirectionOutgoing ||
		d == JudicialCitationIssueDirectionIncoming ||
		d == JudicialCitationIssueDirectionShared
}

func (s JudicialCitationIssueStage) valid() bool {
	return s == JudicialCitationIssueStageOfficialDetailMetadata ||
		s == JudicialCitationIssueStageOfficialPDFText ||
		s == JudicialCitationIssueStageOfficialCaseSearch ||
		s == JudicialCitationIssueStageLawReferenceResolution
}

func cloneJudicialCitationNodes(values []JudicialCitationNode) []JudicialCitationNode {
	return slices.Clone(values)
}

func cloneJudicialCitationEdges(values []JudicialCitationEdge) []JudicialCitationEdge {
	return slices.Clone(values)
}

func cloneJudicialCitationEvidence(values []JudicialCitationEvidence) []JudicialCitationEvidence {
	return slices.Clone(values)
}

func cloneJudicialCitationUnresolvedMentions(values []JudicialCitationUnresolvedMention) []JudicialCitationUnresolvedMention {
	return slices.Clone(values)
}

func cloneJudicialCitationIssues(values []JudicialCitationIssue) []JudicialCitationIssue {
	return slices.Clone(values)
}

func cloneJudicialCitationYearBuckets(values []JudicialCitationYearBucket) []JudicialCitationYearBucket {
	return slices.Clone(values)
}

func cloneJudicialCitationCategoryBuckets(values []JudicialCitationCategoryBucket) []JudicialCitationCategoryBucket {
	return slices.Clone(values)
}
