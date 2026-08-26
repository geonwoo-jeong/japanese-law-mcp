package model

import (
	"encoding/json"
	"fmt"
	"slices"
	"unicode/utf8"
)

// JudicialCitationGraphResultValues は、引用追跡結果の作成値を保持する。
type JudicialCitationGraphResultValues struct {
	Status         JudicialCitationResultStatus
	CoverageNotice string
	Graph          JudicialCitationGraph
	Issues         []JudicialCitationIssue
}

// JudicialCitationGraphResult は、一件の裁判例から追跡した引用関係結果を表す。
type JudicialCitationGraphResult struct {
	status         JudicialCitationResultStatus
	coverageNotice string
	graph          JudicialCitationGraph
	issues         []JudicialCitationIssue
}

// NewJudicialCitationGraphResult は、検証済みの引用追跡結果を返す。
func NewJudicialCitationGraphResult(
	values JudicialCitationGraphResultValues,
) (JudicialCitationGraphResult, error) {
	result := JudicialCitationGraphResult{
		status:         values.Status,
		coverageNotice: values.CoverageNotice,
		graph:          values.Graph,
		issues:         slices.Clone(values.Issues),
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
	return slices.Clone(r.issues)
}

// Validate は、結果状態と方向別 coverage の整合を確認する。
func (r JudicialCitationGraphResult) Validate() error {
	if !r.status.valid() {
		return fmt.Errorf("status が有効ではありません")
	}
	if !utf8.ValidString(r.coverageNotice) || r.coverageNotice == "" {
		return fmt.Errorf("coverageNotice は必須の UTF-8 文字列です")
	}
	if r.issues == nil {
		return fmt.Errorf("issues は空配列または値を持つ配列でなければなりません")
	}
	if err := r.graph.Validate(); err != nil {
		return fmt.Errorf("graph が有効ではありません: %w", err)
	}
	for index, issue := range r.issues {
		if err := issue.Validate(); err != nil {
			return fmt.Errorf("issues[%d] が有効ではありません: %w", index, err)
		}
	}
	allComplete := r.graph.Coverage().requestedDirectionsComplete()
	hasSharedIssue := slices.ContainsFunc(r.issues, func(issue JudicialCitationIssue) bool {
		return issue.Direction() == JudicialCitationIssueDirectionShared
	})
	switch r.status {
	case JudicialCitationResultStatusComplete:
		if !allComplete || hasSharedIssue {
			return fmt.Errorf("complete では要求方向がすべて complete で shared issue がないことが必要です")
		}
	case JudicialCitationResultStatusPartial:
		if len(r.issues) == 0 {
			return fmt.Errorf("partial では issue が一件以上必要です")
		}
		if allComplete && !hasSharedIssue {
			return fmt.Errorf("要求方向がすべて complete で shared issue がない結果は partial にできません")
		}
		if err := r.validateIncompleteDirectionIssues(hasSharedIssue); err != nil {
			return err
		}
	}
	return nil
}

func (r JudicialCitationGraphResult) validateIncompleteDirectionIssues(hasSharedIssue bool) error {
	if hasSharedIssue {
		return nil
	}
	coverage := r.graph.Coverage()
	tests := []struct {
		requested bool
		status    JudicialCitationDirectionStatus
		direction JudicialCitationIssueDirection
	}{
		{
			requested: coverage.RequestedDirection() != JudicialCitationRequestedDirectionIncoming,
			status:    coverage.Outgoing().Status(),
			direction: JudicialCitationIssueDirectionOutgoing,
		},
		{
			requested: coverage.RequestedDirection() != JudicialCitationRequestedDirectionOutgoing,
			status:    coverage.Incoming().Status(),
			direction: JudicialCitationIssueDirectionIncoming,
		},
	}
	for _, test := range tests {
		if !test.requested || test.status == JudicialCitationDirectionStatusComplete {
			continue
		}
		if !slices.ContainsFunc(r.issues, func(issue JudicialCitationIssue) bool {
			return issue.Direction() == test.direction
		}) {
			return fmt.Errorf("未完了の要求方向には同じ方向の issue が必要です")
		}
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-035 の公開項目名で結果を表す。
func (r JudicialCitationGraphResult) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Status         JudicialCitationResultStatus `json:"status"`
		CoverageNotice string                       `json:"coverageNotice"`
		Graph          JudicialCitationGraph        `json:"graph"`
		Issues         []JudicialCitationIssue      `json:"issues"`
	}{r.status, r.coverageNotice, r.graph, slices.Clone(r.issues)})
}

func (*JudicialCitationGraphResult) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationGraphResult は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationGraphResult を使用してください",
	)
}

// JudicialCitationGraphValues は、引用 graph の作成値を保持する。
type JudicialCitationGraphValues struct {
	RootNodeID         string
	Nodes              []JudicialCitationNode
	Edges              []JudicialCitationEdge
	UnresolvedMentions []JudicialCitationUnresolvedMention
	Summary            JudicialCitationSummary
	Coverage           JudicialCitationCoverage
}

// JudicialCitationGraph は、1-hop の引用関係を表す。
type JudicialCitationGraph struct {
	rootNodeID         string
	nodes              []JudicialCitationNode
	edges              []JudicialCitationEdge
	unresolvedMentions []JudicialCitationUnresolvedMention
	summary            JudicialCitationSummary
	coverage           JudicialCitationCoverage
}

// NewJudicialCitationGraph は、重複 edge を統合して検証済み graph を返す。
func NewJudicialCitationGraph(values JudicialCitationGraphValues) (JudicialCitationGraph, error) {
	if values.Nodes == nil || values.Edges == nil || values.UnresolvedMentions == nil {
		return JudicialCitationGraph{}, fmt.Errorf(
			"nodes、edges および unresolvedMentions は空配列または値を持つ配列でなければなりません",
		)
	}
	edges, err := mergeJudicialCitationEdges(values.Edges)
	if err != nil {
		return JudicialCitationGraph{}, err
	}
	graph := JudicialCitationGraph{
		rootNodeID:         values.RootNodeID,
		nodes:              slices.Clone(values.Nodes),
		edges:              edges,
		unresolvedMentions: slices.Clone(values.UnresolvedMentions),
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
	return slices.Clone(g.nodes)
}
func (g JudicialCitationGraph) Edges() []JudicialCitationEdge {
	return slices.Clone(g.edges)
}
func (g JudicialCitationGraph) UnresolvedMentions() []JudicialCitationUnresolvedMention {
	return slices.Clone(g.unresolvedMentions)
}
func (g JudicialCitationGraph) Summary() JudicialCitationSummary   { return g.summary }
func (g JudicialCitationGraph) Coverage() JudicialCitationCoverage { return g.coverage }

// Validate は、root、ノード、edge 行列、件数および coverage を確認する。
func (g JudicialCitationGraph) Validate() error {
	if !utf8.ValidString(g.rootNodeID) || g.rootNodeID == "" {
		return fmt.Errorf("rootNodeId は必須の UTF-8 文字列です")
	}
	if g.nodes == nil || g.edges == nil || g.unresolvedMentions == nil {
		return fmt.Errorf("nodes、edges および unresolvedMentions は nil にできません")
	}
	if len(g.nodes) == 0 {
		return fmt.Errorf("nodes は一件以上必要です")
	}
	nodes, err := validateJudicialCitationNodes(g.nodes, g.rootNodeID)
	if err != nil {
		return err
	}
	if err := validateJudicialCitationEdges(g.edges, nodes, g.rootNodeID); err != nil {
		return err
	}
	for index, mention := range g.unresolvedMentions {
		if err := mention.Validate(); err != nil {
			return fmt.Errorf("unresolvedMentions[%d] が有効ではありません: %w", index, err)
		}
	}
	if err := g.summary.validateAgainst(g.edges, g.unresolvedMentions, nodes); err != nil {
		return fmt.Errorf("summary が graph と一致しません: %w", err)
	}
	if err := g.coverage.Validate(); err != nil {
		return fmt.Errorf("coverage が有効ではありません: %w", err)
	}
	if err := validateJudicialCitationRequestedRelations(g.edges, g.coverage); err != nil {
		return err
	}
	return nil
}

func validateJudicialCitationRequestedRelations(
	edges []JudicialCitationEdge,
	coverage JudicialCitationCoverage,
) error {
	for _, edge := range edges {
		switch {
		case coverage.RequestedDirection() == JudicialCitationRequestedDirectionOutgoing &&
			edge.RelationType() == JudicialCitationRelationTypePossibleCitesDecision:
			return fmt.Errorf("outgoing だけの要求に被引用候補 relation は含められません")
		case coverage.RequestedDirection() == JudicialCitationRequestedDirectionIncoming &&
			edge.RelationType() == JudicialCitationRelationTypeCitesDecision:
			return fmt.Errorf("incoming だけの要求に確認済み引用 relation は含められません")
		}
	}
	return nil
}

func validateJudicialCitationNodes(
	nodes []JudicialCitationNode,
	rootNodeID string,
) (map[string]JudicialCitationNode, error) {
	byID := make(map[string]JudicialCitationNode, len(nodes))
	for index, node := range nodes {
		if err := node.Validate(); err != nil {
			return nil, fmt.Errorf("nodes[%d] が有効ではありません: %w", index, err)
		}
		if _, exists := byID[node.NodeID()]; exists {
			return nil, fmt.Errorf("nodeId %q が重複しています", node.NodeID())
		}
		byID[node.NodeID()] = node
	}
	root, exists := byID[rootNodeID]
	if !exists {
		return nil, fmt.Errorf("rootNodeId が nodes に存在しません")
	}
	if root.NodeType() != JudicialCitationNodeTypeDecision {
		return nil, fmt.Errorf("rootNodeId は judicial_decision ノードでなければなりません")
	}
	return byID, nil
}

func validateJudicialCitationEdges(
	edges []JudicialCitationEdge,
	nodes map[string]JudicialCitationNode,
	rootNodeID string,
) error {
	edgeIDs := make(map[string]struct{}, len(edges))
	keys := make(map[judicialCitationEdgeKey]struct{}, len(edges))
	decisionEdges := 0
	lawEdges := 0
	for index, edge := range edges {
		if err := edge.Validate(); err != nil {
			return fmt.Errorf("edges[%d] が有効ではありません: %w", index, err)
		}
		if _, exists := edgeIDs[edge.EdgeID()]; exists {
			return fmt.Errorf("edgeId %q が重複しています", edge.EdgeID())
		}
		edgeIDs[edge.EdgeID()] = struct{}{}
		key := edge.key()
		if _, exists := keys[key]; exists {
			return fmt.Errorf("同じ始点、終点および relationType の edge が統合されていません")
		}
		keys[key] = struct{}{}
		from, fromExists := nodes[edge.FromNodeID()]
		to, toExists := nodes[edge.ToNodeID()]
		if !fromExists || !toExists {
			return fmt.Errorf("edges[%d] は存在する nodeId を参照しなければなりません", index)
		}
		if err := validateJudicialCitationRelation(edge, from, to, rootNodeID); err != nil {
			return fmt.Errorf("edges[%d] が relation 行列に適合しません: %w", index, err)
		}
		if edge.RelationType() == JudicialCitationRelationTypeReferencesLawProvision {
			lawEdges++
		} else {
			decisionEdges++
		}
	}
	if decisionEdges > judicialCitationDecisionEdgeMax {
		return fmt.Errorf("判例関係 edge は %d 件以下でなければなりません", judicialCitationDecisionEdgeMax)
	}
	if lawEdges > judicialCitationLawEdgeMax {
		return fmt.Errorf("法条 edge は %d 件以下でなければなりません", judicialCitationLawEdgeMax)
	}
	return nil
}

func validateJudicialCitationRelation(
	edge JudicialCitationEdge,
	from JudicialCitationNode,
	to JudicialCitationNode,
	rootNodeID string,
) error {
	var expectedEvidence JudicialCitationEvidenceLevel
	switch edge.RelationType() {
	case JudicialCitationRelationTypeCitesDecision:
		expectedEvidence = JudicialCitationEvidenceLevelExactTextMatch
		if edge.FromNodeID() != rootNodeID || from.NodeType() != JudicialCitationNodeTypeDecision ||
			(to.NodeType() != JudicialCitationNodeTypeDecision &&
				to.NodeType() != JudicialCitationNodeTypeDecisionReference) {
			return fmt.Errorf("確認済み引用の向き又はノード種別が不正です")
		}
	case JudicialCitationRelationTypePossibleCitesDecision:
		expectedEvidence = JudicialCitationEvidenceLevelOfficialSearchCandidate
		if edge.ToNodeID() != rootNodeID || from.NodeType() != JudicialCitationNodeTypeDecision ||
			to.NodeType() != JudicialCitationNodeTypeDecision {
			return fmt.Errorf("被引用候補の向き又はノード種別が不正です")
		}
	case JudicialCitationRelationTypeReferencesLawProvision:
		expectedEvidence = JudicialCitationEvidenceLevelOfficialMetadata
		if edge.FromNodeID() != rootNodeID || from.NodeType() != JudicialCitationNodeTypeDecision ||
			to.NodeType() != JudicialCitationNodeTypeLawProvision {
			return fmt.Errorf("参照法条の向き又はノード種別が不正です")
		}
	case JudicialCitationRelationTypeHasLowerCourtDecision:
		expectedEvidence = JudicialCitationEvidenceLevelOfficialMetadata
		if edge.FromNodeID() != rootNodeID || from.NodeType() != JudicialCitationNodeTypeDecision ||
			(to.NodeType() != JudicialCitationNodeTypeDecision &&
				to.NodeType() != JudicialCitationNodeTypeDecisionReference) {
			return fmt.Errorf("原審関係の向き又はノード種別が不正です")
		}
	}
	for _, evidence := range edge.Evidence() {
		if evidence.EvidenceLevel() != expectedEvidence {
			return fmt.Errorf("relationType と evidenceLevel が一致しません")
		}
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-035 の graph 項目名を使用する。
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
		g.rootNodeID,
		slices.Clone(g.nodes),
		slices.Clone(g.edges),
		slices.Clone(g.unresolvedMentions),
		g.summary,
		g.coverage,
	})
}

func (*JudicialCitationGraph) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationGraph は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationGraph を使用してください",
	)
}
