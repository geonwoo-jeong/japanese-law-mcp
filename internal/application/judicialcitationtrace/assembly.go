package judicialcitationtrace

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	maximumDecisionEdges = 64
	maximumLawEdges      = 32
)

type graphAssembly struct {
	rootNodeID            string
	rootRef               model.SourceResourceRef
	nodes                 []model.JudicialCitationNode
	edges                 []model.JudicialCitationEdge
	unresolved            []model.JudicialCitationUnresolvedMention
	edgeIndexes           map[assemblyEdgeKey]int
	decisionReferenceNode map[string]string
	decisionNode          map[model.SourceResourceRef]string
	lawNode               map[string]string
	decisionEdgeCount     int
	lawEdgeCount          int
}

type assemblyEdgeKey struct {
	from     string
	to       string
	relation model.JudicialCitationRelationType
}

func newGraphAssembly(
	root model.SourcedResource[model.JudicialDecisionDetails],
) (*graphAssembly, error) {
	summary := root.Data().Summary()
	ref := root.Ref()
	rootNode, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:          "node-1",
		NodeType:        model.JudicialCitationNodeTypeDecision,
		Label:           decisionLabel(summary),
		Ref:             &ref,
		DecisionSummary: &summary,
	})
	if err != nil {
		return nil, err
	}
	return &graphAssembly{
		rootNodeID:            rootNode.NodeID(),
		rootRef:               ref,
		nodes:                 []model.JudicialCitationNode{rootNode},
		edges:                 []model.JudicialCitationEdge{},
		unresolved:            []model.JudicialCitationUnresolvedMention{},
		edgeIndexes:           make(map[assemblyEdgeKey]int),
		decisionReferenceNode: make(map[string]string),
		decisionNode:          map[model.SourceResourceRef]string{ref: rootNode.NodeID()},
		lawNode:               make(map[string]string),
	}, nil
}

func (a *graphAssembly) addUnresolved(values []model.JudicialCitationUnresolvedMention) {
	a.unresolved = append(a.unresolved, values...)
}

func (a *graphAssembly) addLowerCourt(
	reference judicialcitation.LowerCourtDecisionReference,
	evidence model.JudicialCitationEvidence,
) (bool, error) {
	key := "lower\x00" + reference.CourtName() + "\x00" + reference.CaseNumberSearch()
	nodeID, exists := a.decisionReferenceNode[key]
	if !exists {
		if a.decisionEdgeCount >= maximumDecisionEdges {
			return false, nil
		}
		text := reference.CourtName() + " " + reference.CaseNumber()
		created, err := a.addDecisionReferenceNode(text, text)
		if err != nil {
			return false, err
		}
		nodeID = created
		a.decisionReferenceNode[key] = nodeID
	}
	return a.addEdge(
		a.rootNodeID,
		nodeID,
		model.JudicialCitationRelationTypeHasLowerCourtDecision,
		[]model.JudicialCitationEvidence{evidence},
	)
}

func (a *graphAssembly) addLawReference(
	reference model.JudicialCitationLawReference,
	evidence model.JudicialCitationEvidence,
) (bool, error) {
	key := lawReferenceKey(reference)
	nodeID, exists := a.lawNode[key]
	if !exists {
		if a.lawEdgeCount >= maximumLawEdges {
			return false, nil
		}
		node, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
			NodeID:       a.nextNodeID(),
			NodeType:     model.JudicialCitationNodeTypeLawProvision,
			Label:        lawReferenceLabel(reference),
			LawReference: &reference,
		})
		if err != nil {
			return false, err
		}
		a.nodes = append(a.nodes, node)
		nodeID = node.NodeID()
		a.lawNode[key] = nodeID
	}
	return a.addEdge(
		a.rootNodeID,
		nodeID,
		model.JudicialCitationRelationTypeReferencesLawProvision,
		[]model.JudicialCitationEvidence{evidence},
	)
}

func (a *graphAssembly) addOutgoingMention(
	mention model.JudicialCitationDecisionMention,
) (bool, error) {
	key := "outgoing\x00" + mention.DecisionIdentityText()
	nodeID, exists := a.decisionReferenceNode[key]
	if !exists {
		if a.decisionEdgeCount >= maximumDecisionEdges {
			return false, nil
		}
		created, err := a.addDecisionReferenceNode(
			mention.ReferenceText(),
			mention.ReferenceText(),
		)
		if err != nil {
			return false, err
		}
		nodeID = created
		a.decisionReferenceNode[key] = nodeID
	}
	return a.addEdge(
		a.rootNodeID,
		nodeID,
		model.JudicialCitationRelationTypeCitesDecision,
		[]model.JudicialCitationEvidence{mention.Evidence()},
	)
}

func (a *graphAssembly) addIncomingCandidate(
	candidate judicialcitingcandidatesearch.Candidate,
) (bool, error) {
	resource := candidate.Decision()
	if resource.Ref() == a.rootRef {
		return true, nil
	}
	nodeID, exists := a.decisionNode[resource.Ref()]
	if !exists {
		if a.decisionEdgeCount >= maximumDecisionEdges {
			return false, nil
		}
		ref := resource.Ref()
		summary := resource.Data()
		node, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
			NodeID:          a.nextNodeID(),
			NodeType:        model.JudicialCitationNodeTypeDecision,
			Label:           decisionLabel(summary),
			Ref:             &ref,
			DecisionSummary: &summary,
		})
		if err != nil {
			return false, err
		}
		a.nodes = append(a.nodes, node)
		nodeID = node.NodeID()
		a.decisionNode[ref] = nodeID
	}
	return a.addEdge(
		nodeID,
		a.rootNodeID,
		model.JudicialCitationRelationTypePossibleCitesDecision,
		candidate.Evidence(),
	)
}

func (a *graphAssembly) addDecisionReferenceNode(label, referenceText string) (string, error) {
	node, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:        a.nextNodeID(),
		NodeType:      model.JudicialCitationNodeTypeDecisionReference,
		Label:         label,
		ReferenceText: &referenceText,
	})
	if err != nil {
		return "", err
	}
	a.nodes = append(a.nodes, node)
	return node.NodeID(), nil
}

func (a *graphAssembly) addEdge(
	from, to string,
	relation model.JudicialCitationRelationType,
	evidence []model.JudicialCitationEvidence,
) (bool, error) {
	key := assemblyEdgeKey{from: from, to: to, relation: relation}
	if index, exists := a.edgeIndexes[key]; exists {
		combined := append(a.edges[index].Evidence(), evidence...)
		updated, err := model.NewJudicialCitationEdge(model.JudicialCitationEdgeValues{
			EdgeID:       a.edges[index].EdgeID(),
			FromNodeID:   from,
			ToNodeID:     to,
			RelationType: relation,
			Evidence:     combined,
		})
		if err != nil {
			return false, err
		}
		a.edges[index] = updated
		return true, nil
	}
	if relation == model.JudicialCitationRelationTypeReferencesLawProvision {
		if a.lawEdgeCount >= maximumLawEdges {
			return false, nil
		}
	} else if a.decisionEdgeCount >= maximumDecisionEdges {
		return false, nil
	}
	edge, err := model.NewJudicialCitationEdge(model.JudicialCitationEdgeValues{
		EdgeID:       a.nextEdgeID(),
		FromNodeID:   from,
		ToNodeID:     to,
		RelationType: relation,
		Evidence:     evidence,
	})
	if err != nil {
		return false, err
	}
	a.edgeIndexes[key] = len(a.edges)
	a.edges = append(a.edges, edge)
	if relation == model.JudicialCitationRelationTypeReferencesLawProvision {
		a.lawEdgeCount++
	} else {
		a.decisionEdgeCount++
	}
	return true, nil
}

func (a *graphAssembly) graph(
	coverage model.JudicialCitationCoverage,
) (model.JudicialCitationGraph, error) {
	summary, err := a.summary()
	if err != nil {
		return model.JudicialCitationGraph{}, err
	}
	return model.NewJudicialCitationGraph(model.JudicialCitationGraphValues{
		RootNodeID:         a.rootNodeID,
		Nodes:              a.nodes,
		Edges:              a.edges,
		UnresolvedMentions: a.unresolved,
		Summary:            summary,
		Coverage:           coverage,
	})
}

func (a *graphAssembly) summary() (model.JudicialCitationSummary, error) {
	counts := map[model.JudicialCitationRelationType]int{}
	years := map[int]int{}
	categories := map[model.JudicialPublicationCategory]int{}
	byNodeID := make(map[string]model.JudicialCitationNode, len(a.nodes))
	for _, node := range a.nodes {
		byNodeID[node.NodeID()] = node
	}
	for _, edge := range a.edges {
		counts[edge.RelationType()]++
		if edge.RelationType() != model.JudicialCitationRelationTypePossibleCitesDecision {
			continue
		}
		summary, exists := byNodeID[edge.FromNodeID()].DecisionSummary()
		if !exists {
			return model.JudicialCitationSummary{}, fmt.Errorf("被引用候補の概要がありません")
		}
		year, err := strconv.Atoi(summary.DecisionDate().String()[:4])
		if err != nil {
			return model.JudicialCitationSummary{}, err
		}
		years[year]++
		categories[summary.PublicationCategory()]++
	}
	yearBuckets, err := makeYearBuckets(years)
	if err != nil {
		return model.JudicialCitationSummary{}, err
	}
	categoryBuckets, err := makeCategoryBuckets(categories)
	if err != nil {
		return model.JudicialCitationSummary{}, err
	}
	return model.NewJudicialCitationSummary(model.JudicialCitationSummaryValues{
		ConfirmedOutgoingDecisionCount:  counts[model.JudicialCitationRelationTypeCitesDecision],
		IncomingCandidateCount:          counts[model.JudicialCitationRelationTypePossibleCitesDecision],
		ReferencedProvisionCount:        counts[model.JudicialCitationRelationTypeReferencesLawProvision],
		LowerCourtRelationCount:         counts[model.JudicialCitationRelationTypeHasLowerCourtDecision],
		UnresolvedMentionCount:          len(a.unresolved),
		IncomingObservedYearBuckets:     yearBuckets,
		IncomingObservedCategoryBuckets: categoryBuckets,
	})
}

func makeYearBuckets(counts map[int]int) ([]model.JudicialCitationYearBucket, error) {
	years := make([]int, 0, len(counts))
	for year := range counts {
		years = append(years, year)
	}
	sort.Ints(years)
	result := make([]model.JudicialCitationYearBucket, 0, len(years))
	for _, year := range years {
		bucket, err := model.NewJudicialCitationYearBucket(year, counts[year])
		if err != nil {
			return nil, err
		}
		result = append(result, bucket)
	}
	return result, nil
}

func makeCategoryBuckets(
	counts map[model.JudicialPublicationCategory]int,
) ([]model.JudicialCitationCategoryBucket, error) {
	ordered := []model.JudicialPublicationCategory{
		model.JudicialPublicationCategorySupremeCourt,
		model.JudicialPublicationCategoryHighCourt,
		model.JudicialPublicationCategoryLowerCourt,
		model.JudicialPublicationCategoryAdministrative,
		model.JudicialPublicationCategoryLabor,
		model.JudicialPublicationCategoryIntellectualProperty,
	}
	result := make([]model.JudicialCitationCategoryBucket, 0, len(counts))
	for _, category := range ordered {
		if counts[category] == 0 {
			continue
		}
		bucket, err := model.NewJudicialCitationCategoryBucket(category, counts[category])
		if err != nil {
			return nil, err
		}
		result = append(result, bucket)
	}
	return result, nil
}

func (a *graphAssembly) nextNodeID() string {
	return "node-" + strconv.Itoa(len(a.nodes)+1)
}

func (a *graphAssembly) nextEdgeID() string {
	return "edge-" + strconv.Itoa(len(a.edges)+1)
}

func decisionLabel(summary model.JudicialDecisionSummary) string {
	if name, exists := summary.CaseName(); exists {
		return name
	}
	return summary.CaseNumber()
}

func lawReferenceKey(reference model.JudicialCitationLawReference) string {
	location := reference.Location()
	paragraph, _ := location.ParagraphNumber()
	return strings.Join([]string{
		reference.LawID(),
		string(location.Provision()),
		location.ArticleNumber(),
		strconv.Itoa(paragraph),
	}, "\x00")
}

func lawReferenceLabel(reference model.JudicialCitationLawReference) string {
	location := reference.Location()
	label := reference.LawTitle()
	if location.Provision() == model.LawArticleProvisionSupplementary {
		label += "附則"
	}
	label += "第" + strings.ReplaceAll(location.ArticleNumber(), "_", "の") + "条"
	if paragraph, exists := location.ParagraphNumber(); exists {
		label += "第" + strconv.Itoa(paragraph) + "項"
	}
	return label
}
