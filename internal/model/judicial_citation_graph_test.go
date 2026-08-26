package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestJudicialCitationGraphResult(t *testing.T) {
	t.Parallel()

	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key: newSourceResourceKey(t, model.SourceResourceKeyValues{
			SourceID:     "courts-hanrei",
			ResourceType: "judicial-decision",
			ResourceID:   "95878/detail3",
		}),
	})
	if err != nil {
		t.Fatalf("ref を作成できません: %v", err)
	}
	summary := validJudicialDecisionSummaryValues(t)
	decisionSummary, err := model.NewJudicialDecisionSummary(summary)
	if err != nil {
		t.Fatalf("summary を作成できません: %v", err)
	}
	decisionNode, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:          "n1",
		NodeType:        "judicial_decision",
		Label:           "令和7年(受)第1号",
		Ref:             &ref,
		DecisionSummary: &decisionSummary,
	})
	if err != nil {
		t.Fatalf("decision node を作成できません: %v", err)
	}
	location, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: "709",
	})
	if err != nil {
		t.Fatalf("location を作成できません: %v", err)
	}
	lawReference, err := model.NewJudicialCitationLawReference(model.JudicialCitationLawReferenceValues{
		LawID:    "129AC0000000089",
		LawTitle: "民法",
		Location: location,
	})
	if err != nil {
		t.Fatalf("law reference を作成できません: %v", err)
	}
	lawNode, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:       "n2",
		NodeType:     "law_provision",
		Label:        "民法709条",
		LawReference: &lawReference,
	})
	if err != nil {
		t.Fatalf("law node を作成できません: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         decisionSummary.Source(),
		ResourceKey:    ref.Key(),
		URL:            decisionSummary.DetailURL(),
		RetrievedAt:    time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-075",
	})
	if err != nil {
		t.Fatalf("provenance を作成できません: %v", err)
	}
	excerpt := "参照法条 民法709条"
	evidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: "official_metadata",
		Provenance:    provenance,
		Excerpt:       &excerpt,
	})
	if err != nil {
		t.Fatalf("evidence を作成できません: %v", err)
	}
	edge, err := model.NewJudicialCitationEdge(model.JudicialCitationEdgeValues{
		EdgeID:       "e1",
		FromNodeID:   "n1",
		ToNodeID:     "n2",
		RelationType: "references_law_provision",
		Evidence:     []model.JudicialCitationEvidence{evidence},
	})
	if err != nil {
		t.Fatalf("edge を作成できません: %v", err)
	}
	directionCoverage, err := model.NewJudicialCitationDirectionCoverage(model.JudicialCitationDirectionCoverageValues{
		Status:    "complete",
		Methods:   []model.JudicialCitationMethod{"official_detail_metadata", "official_pdf_text"},
		Truncated: false,
	})
	if err != nil {
		t.Fatalf("direction coverage を作成できません: %v", err)
	}
	notRequested, err := model.NewJudicialCitationDirectionCoverage(model.JudicialCitationDirectionCoverageValues{
		Status:    "not_requested",
		Methods:   []model.JudicialCitationMethod{},
		Truncated: false,
	})
	if err != nil {
		t.Fatalf("not requested coverage を作成できません: %v", err)
	}
	coverage, err := model.NewJudicialCitationCoverage(model.JudicialCitationCoverageValues{
		RequestedDirection: "outgoing",
		HopDepth:           1,
		Outgoing:           directionCoverage,
		Incoming:           notRequested,
	})
	if err != nil {
		t.Fatalf("coverage を作成できません: %v", err)
	}
	summaryModel, err := model.NewJudicialCitationSummary(model.JudicialCitationSummaryValues{
		ConfirmedOutgoingDecisionCount:  0,
		IncomingCandidateCount:          0,
		ReferencedProvisionCount:        1,
		LowerCourtRelationCount:         0,
		UnresolvedMentionCount:          0,
		IncomingObservedYearBuckets:     []model.JudicialCitationYearBucket{},
		IncomingObservedCategoryBuckets: []model.JudicialCitationCategoryBucket{},
	})
	if err != nil {
		t.Fatalf("summary を作成できません: %v", err)
	}
	graph, err := model.NewJudicialCitationGraph(model.JudicialCitationGraphValues{
		RootNodeID:         "n1",
		Nodes:              []model.JudicialCitationNode{decisionNode, lawNode},
		Edges:              []model.JudicialCitationEdge{edge},
		UnresolvedMentions: []model.JudicialCitationUnresolvedMention{},
		Summary:            summaryModel,
		Coverage:           coverage,
	})
	if err != nil {
		t.Fatalf("graph を作成できません: %v", err)
	}
	result, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         "complete",
		CoverageNotice: "固定注意文",
		Graph:          graph,
		Issues:         []model.JudicialCitationIssue{},
	})
	if err != nil {
		t.Fatalf("result を作成できません: %v", err)
	}
	if _, err := json.Marshal(result); err != nil {
		t.Fatalf("MarshalJSON() のエラー = %v", err)
	}
}

func TestJudicialCitationGraphRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	if _, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         "partial",
		CoverageNotice: "固定注意文",
	}); err == nil {
		t.Fatal("graph なしの partial を受理しました")
	}

	issue, err := model.NewJudicialCitationIssue(model.JudicialCitationIssueValues{
		Direction: "shared",
		Stage:     "law_reference_resolution",
		Code:      "document_text_unavailable",
		Message:   "利用不可",
		Retryable: false,
	})
	if err != nil {
		t.Fatalf("issue を作成できません: %v", err)
	}
	if _, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         "complete",
		CoverageNotice: "固定注意文",
		Graph:          model.JudicialCitationGraph{},
		Issues:         []model.JudicialCitationIssue{issue},
	}); err == nil {
		t.Fatal("不正な graph を受理しました")
	}
}

func TestJudicialCitationGraphRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.JudicialCitationGraphResult
	if err := json.Unmarshal([]byte(`{}`), &got); err == nil {
		t.Fatal("JudicialCitationGraphResult を JSON から直接復元できました")
	}
}

func TestJudicialCitationGraphValidatesRelationMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		relationType string
		fromNodeID   string
		toNodeID     string
		evidence     string
		wantError    bool
	}{
		{
			name:         "確認済み引用",
			relationType: "cites_judicial_decision",
			fromNodeID:   "root",
			toNodeID:     "reference",
			evidence:     "exact_text_match",
		},
		{
			name:         "被引用候補",
			relationType: "possible_cites_judicial_decision",
			fromNodeID:   "candidate",
			toNodeID:     "root",
			evidence:     "official_search_candidate",
		},
		{
			name:         "参照法条",
			relationType: "references_law_provision",
			fromNodeID:   "root",
			toNodeID:     "law",
			evidence:     "official_metadata",
		},
		{
			name:         "原審",
			relationType: "has_lower_court_decision",
			fromNodeID:   "root",
			toNodeID:     "reference",
			evidence:     "official_metadata",
		},
		{
			name:         "確認済み引用の始点がルートではない",
			relationType: "cites_judicial_decision",
			fromNodeID:   "candidate",
			toNodeID:     "reference",
			evidence:     "exact_text_match",
			wantError:    true,
		},
		{
			name:         "被引用候補の終点がルートではない",
			relationType: "possible_cites_judicial_decision",
			fromNodeID:   "candidate",
			toNodeID:     "reference",
			evidence:     "official_search_candidate",
			wantError:    true,
		},
		{
			name:         "参照法条の始点がルートではない",
			relationType: "references_law_provision",
			fromNodeID:   "candidate",
			toNodeID:     "law",
			evidence:     "official_metadata",
			wantError:    true,
		},
		{
			name:         "原審の終点が法条",
			relationType: "has_lower_court_decision",
			fromNodeID:   "root",
			toNodeID:     "law",
			evidence:     "official_metadata",
			wantError:    true,
		},
		{
			name:         "確認済み引用に検索候補根拠",
			relationType: "cites_judicial_decision",
			fromNodeID:   "root",
			toNodeID:     "reference",
			evidence:     "official_search_candidate",
			wantError:    true,
		},
		{
			name:         "被引用候補に完全一致根拠",
			relationType: "possible_cites_judicial_decision",
			fromNodeID:   "candidate",
			toNodeID:     "root",
			evidence:     "exact_text_match",
			wantError:    true,
		},
		{
			name:         "参照法条に完全一致根拠",
			relationType: "references_law_provision",
			fromNodeID:   "root",
			toNodeID:     "law",
			evidence:     "exact_text_match",
			wantError:    true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fixture := newJudicialCitationFixture(t)
			edge := fixture.newEdge(
				t,
				"edge",
				test.fromNodeID,
				test.toNodeID,
				test.relationType,
				test.evidence,
				"根拠",
			)
			_, err := fixture.newGraph(t, []model.JudicialCitationEdge{edge}, nil)
			if test.wantError && err == nil {
				t.Fatal("SOT-MODEL-035 の relation 行列に反する edge を受理しました")
			}
			if !test.wantError && err != nil {
				t.Fatalf("SOT-MODEL-035 の有効な edge を拒否しました: %v", err)
			}
		})
	}
}

func TestJudicialCitationGraphMergesDuplicateEdgesAndPreservesEvidenceOrder(t *testing.T) {
	t.Parallel()

	fixture := newJudicialCitationFixture(t)
	first := fixture.newEdge(
		t,
		"edge-first",
		"root",
		"reference",
		"cites_judicial_decision",
		"exact_text_match",
		"第一の根拠",
	)
	second := fixture.newEdge(
		t,
		"edge-second",
		"root",
		"reference",
		"cites_judicial_decision",
		"exact_text_match",
		"第二の根拠",
	)
	duplicate := fixture.newEdge(
		t,
		"edge-duplicate",
		"root",
		"reference",
		"cites_judicial_decision",
		"exact_text_match",
		"第一の根拠",
	)
	graph, err := fixture.newGraph(
		t,
		[]model.JudicialCitationEdge{first, second, duplicate},
		nil,
	)
	if err != nil {
		t.Fatalf("重複 edge を統合できません: %v", err)
	}
	edges := graph.Edges()
	if len(edges) != 1 {
		t.Fatalf("統合後の edge 数 = %d, want 1", len(edges))
	}
	if edges[0].EdgeID() != "edge-first" {
		t.Fatalf("統合後の edgeId = %q, want edge-first", edges[0].EdgeID())
	}
	evidence := edges[0].Evidence()
	if len(evidence) != 2 {
		t.Fatalf("統合後の evidence 数 = %d, want 2", len(evidence))
	}
	firstExcerpt, _ := evidence[0].Excerpt()
	secondExcerpt, _ := evidence[1].Excerpt()
	if firstExcerpt != "第一の根拠" || secondExcerpt != "第二の根拠" {
		t.Fatalf("evidence の順序 = %q, %q", firstExcerpt, secondExcerpt)
	}
}

func TestJudicialCitationGraphUsesImmutableSlicesAndNonNullJSONArrays(t *testing.T) {
	t.Parallel()

	fixture := newJudicialCitationFixture(t)
	edge := fixture.newEdge(
		t,
		"edge",
		"root",
		"reference",
		"cites_judicial_decision",
		"exact_text_match",
		"根拠",
	)
	edges := []model.JudicialCitationEdge{edge}
	graph, err := fixture.newGraph(t, edges, nil)
	if err != nil {
		t.Fatalf("graph を作成できません: %v", err)
	}
	edges[0] = model.JudicialCitationEdge{}
	got := graph.Edges()
	got[0] = model.JudicialCitationEdge{}
	if graph.Edges()[0].EdgeID() != "edge" {
		t.Fatal("graph の edge が入力又は getter の slice 変更で変化しました")
	}

	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("graph を JSON 化できません: %v", err)
	}
	var payload struct {
		Edges              []json.RawMessage `json:"edges"`
		UnresolvedMentions []json.RawMessage `json:"unresolvedMentions"`
		Summary            struct {
			Years      []json.RawMessage `json:"incomingObservedYearBuckets"`
			Categories []json.RawMessage `json:"incomingObservedCategoryBuckets"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("graph JSON を読めません: %v", err)
	}
	if payload.Edges == nil || payload.UnresolvedMentions == nil ||
		payload.Summary.Years == nil || payload.Summary.Categories == nil {
		t.Fatalf("必須配列が null です: %s", raw)
	}
}

func TestJudicialCitationGraphResultValidatesStatusAndCoverage(t *testing.T) {
	t.Parallel()

	fixture := newJudicialCitationFixture(t)
	completeGraph, err := fixture.newGraph(t, nil, nil)
	if err != nil {
		t.Fatalf("complete graph を作成できません: %v", err)
	}
	issue, err := model.NewJudicialCitationIssue(model.JudicialCitationIssueValues{
		Direction: "shared",
		Stage:     "law_reference_resolution",
		Code:      "resolution_failed",
		Message:   "法条を解決できませんでした",
		Retryable: false,
	})
	if err != nil {
		t.Fatalf("issue を作成できません: %v", err)
	}
	if _, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         "complete",
		CoverageNotice: "固定注意文",
		Graph:          completeGraph,
		Issues:         []model.JudicialCitationIssue{issue},
	}); err == nil {
		t.Fatal("shared issue を持つ complete を受理しました")
	}
	if _, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         "partial",
		CoverageNotice: "固定注意文",
		Graph:          completeGraph,
		Issues:         []model.JudicialCitationIssue{mustIssue(t, "outgoing")},
	}); err == nil {
		t.Fatal("要求方向がすべて complete の partial を受理しました")
	}

	partialGraph := fixture.newGraphWithCoverage(t, "partial", "not_requested")
	if _, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         "complete",
		CoverageNotice: "固定注意文",
		Graph:          partialGraph,
		Issues:         []model.JudicialCitationIssue{},
	}); err == nil {
		t.Fatal("要求方向が partial の complete を受理しました")
	}
}

type judicialCitationFixture struct {
	nodes      []model.JudicialCitationNode
	provenance model.Provenance
}

func newJudicialCitationFixture(t *testing.T) judicialCitationFixture {
	t.Helper()

	rootRef, rootSummary := newCitationDecision(t, "100", "令和6年（受）第1号")
	root, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:          "root",
		NodeType:        "judicial_decision",
		Label:           "起点裁判例",
		Ref:             &rootRef,
		DecisionSummary: &rootSummary,
	})
	if err != nil {
		t.Fatalf("root node を作成できません: %v", err)
	}
	candidateRef, candidateSummary := newCitationDecision(t, "200", "令和7年（受）第2号")
	candidate, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:          "candidate",
		NodeType:        "judicial_decision",
		Label:           "候補裁判例",
		Ref:             &candidateRef,
		DecisionSummary: &candidateSummary,
	})
	if err != nil {
		t.Fatalf("candidate node を作成できません: %v", err)
	}
	referenceText := "令和元年（受）第3号"
	reference, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:        "reference",
		NodeType:      "judicial_decision_reference",
		Label:         "未掲載裁判例参照",
		ReferenceText: &referenceText,
	})
	if err != nil {
		t.Fatalf("reference node を作成できません: %v", err)
	}
	location, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: "709",
	})
	if err != nil {
		t.Fatalf("location を作成できません: %v", err)
	}
	lawReference, err := model.NewJudicialCitationLawReference(model.JudicialCitationLawReferenceValues{
		LawID:    "129AC0000000089",
		LawTitle: "民法",
		Location: location,
	})
	if err != nil {
		t.Fatalf("law reference を作成できません: %v", err)
	}
	law, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:       "law",
		NodeType:     "law_provision",
		Label:        "民法第709条",
		LawReference: &lawReference,
	})
	if err != nil {
		t.Fatalf("law node を作成できません: %v", err)
	}
	return judicialCitationFixture{
		nodes:      []model.JudicialCitationNode{root, candidate, reference, law},
		provenance: newCitationGraphProvenance(t, rootRef),
	}
}

func newCitationDecision(
	t *testing.T,
	decisionID string,
	caseNumber string,
) (model.SourceResourceRef, model.JudicialDecisionSummary) {
	t.Helper()
	key := newSourceResourceKey(t, model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   decisionID + "/detail3",
	})
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("ref を作成できません: %v", err)
	}
	values := validJudicialDecisionSummaryValues(t)
	values.DecisionID = decisionID
	values.CaseNumber = caseNumber
	values.DetailURL = "https://www.courts.go.jp/hanrei/" + decisionID + "/detail3/index.html"
	summary, err := model.NewJudicialDecisionSummary(values)
	if err != nil {
		t.Fatalf("decision summary を作成できません: %v", err)
	}
	return ref, summary
}

func newCitationGraphProvenance(t *testing.T, ref model.SourceResourceRef) model.Provenance {
	t.Helper()
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         newJudicialInformationSource(t),
		ResourceKey:    ref.Key(),
		URL:            "https://www.courts.go.jp/hanrei/100/detail3/index.html",
		RetrievedAt:    time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-075",
	})
	if err != nil {
		t.Fatalf("provenance を作成できません: %v", err)
	}
	return provenance
}

func (f judicialCitationFixture) newEdge(
	t *testing.T,
	edgeID string,
	fromNodeID string,
	toNodeID string,
	relationType string,
	evidenceLevel string,
	excerpt string,
) model.JudicialCitationEdge {
	t.Helper()
	evidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevel(evidenceLevel),
		Provenance:    f.provenance,
		Excerpt:       &excerpt,
	})
	if err != nil {
		t.Fatalf("evidence を作成できません: %v", err)
	}
	edge, err := model.NewJudicialCitationEdge(model.JudicialCitationEdgeValues{
		EdgeID:       edgeID,
		FromNodeID:   fromNodeID,
		ToNodeID:     toNodeID,
		RelationType: model.JudicialCitationRelationType(relationType),
		Evidence:     []model.JudicialCitationEvidence{evidence},
	})
	if err != nil {
		t.Fatalf("edge を作成できません: %v", err)
	}
	return edge
}

func (f judicialCitationFixture) newGraph(
	t *testing.T,
	edges []model.JudicialCitationEdge,
	summaryOverride *model.JudicialCitationSummary,
) (model.JudicialCitationGraph, error) {
	t.Helper()
	if edges == nil {
		edges = []model.JudicialCitationEdge{}
	}
	summary := judicialCitationSummaryForEdges(t, edges)
	if summaryOverride != nil {
		summary = *summaryOverride
	}
	coverage := mustCitationCoverage(t, "complete", "not_requested")
	for _, edge := range edges {
		if edge.RelationType() == model.JudicialCitationRelationTypePossibleCitesDecision {
			coverage = mustIncomingCitationCoverage(t)
			break
		}
	}
	return model.NewJudicialCitationGraph(model.JudicialCitationGraphValues{
		RootNodeID:         "root",
		Nodes:              f.nodes,
		Edges:              edges,
		UnresolvedMentions: []model.JudicialCitationUnresolvedMention{},
		Summary:            summary,
		Coverage:           coverage,
	})
}

func (f judicialCitationFixture) newGraphWithCoverage(
	t *testing.T,
	outgoingStatus string,
	incomingStatus string,
) model.JudicialCitationGraph {
	t.Helper()
	summary := judicialCitationSummaryForEdges(t, []model.JudicialCitationEdge{})
	graph, err := model.NewJudicialCitationGraph(model.JudicialCitationGraphValues{
		RootNodeID:         "root",
		Nodes:              f.nodes,
		Edges:              []model.JudicialCitationEdge{},
		UnresolvedMentions: []model.JudicialCitationUnresolvedMention{},
		Summary:            summary,
		Coverage:           mustCitationCoverage(t, outgoingStatus, incomingStatus),
	})
	if err != nil {
		t.Fatalf("graph を作成できません: %v", err)
	}
	return graph
}

func judicialCitationSummaryForEdges(
	t *testing.T,
	edges []model.JudicialCitationEdge,
) model.JudicialCitationSummary {
	t.Helper()
	values := model.JudicialCitationSummaryValues{
		IncomingObservedYearBuckets:     []model.JudicialCitationYearBucket{},
		IncomingObservedCategoryBuckets: []model.JudicialCitationCategoryBucket{},
	}
	seen := make(map[[3]string]struct{}, len(edges))
	for _, edge := range edges {
		key := [3]string{edge.FromNodeID(), edge.ToNodeID(), string(edge.RelationType())}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		switch edge.RelationType() {
		case "cites_judicial_decision":
			values.ConfirmedOutgoingDecisionCount++
		case "possible_cites_judicial_decision":
			values.IncomingCandidateCount++
		case "references_law_provision":
			values.ReferencedProvisionCount++
		case "has_lower_court_decision":
			values.LowerCourtRelationCount++
		}
	}
	if values.IncomingCandidateCount > 0 {
		yearBucket, err := model.NewJudicialCitationYearBucket(2025, values.IncomingCandidateCount)
		if err != nil {
			t.Fatalf("year bucket を作成できません: %v", err)
		}
		categoryBucket, err := model.NewJudicialCitationCategoryBucket(
			model.JudicialPublicationCategorySupremeCourt,
			values.IncomingCandidateCount,
		)
		if err != nil {
			t.Fatalf("category bucket を作成できません: %v", err)
		}
		values.IncomingObservedYearBuckets = []model.JudicialCitationYearBucket{yearBucket}
		values.IncomingObservedCategoryBuckets = []model.JudicialCitationCategoryBucket{categoryBucket}
	}
	summary, err := model.NewJudicialCitationSummary(values)
	if err != nil {
		t.Fatalf("summary を作成できません: %v", err)
	}
	return summary
}

func mustCitationCoverage(
	t *testing.T,
	outgoingStatus string,
	incomingStatus string,
) model.JudicialCitationCoverage {
	t.Helper()
	outgoingMethods := []string{}
	if outgoingStatus != "not_requested" {
		outgoingMethods = []string{"official_pdf_text"}
	}
	outgoing := mustDirectionCoverage(t, outgoingStatus, outgoingMethods)
	incoming := mustDirectionCoverage(t, incomingStatus, []string{})
	coverage, err := model.NewJudicialCitationCoverage(model.JudicialCitationCoverageValues{
		RequestedDirection: "outgoing",
		HopDepth:           1,
		Outgoing:           outgoing,
		Incoming:           incoming,
	})
	if err != nil {
		t.Fatalf("coverage を作成できません: %v", err)
	}
	return coverage
}

func mustDirectionCoverage(
	t *testing.T,
	status string,
	methods []string,
) model.JudicialCitationDirectionCoverage {
	t.Helper()
	coverage, err := model.NewJudicialCitationDirectionCoverage(
		model.JudicialCitationDirectionCoverageValues{
			Status:    model.JudicialCitationDirectionStatus(status),
			Methods:   judicialCitationMethods(methods),
			Truncated: false,
		},
	)
	if err != nil {
		t.Fatalf("direction coverage を作成できません: %v", err)
	}
	return coverage
}

func mustIssue(t *testing.T, direction string) model.JudicialCitationIssue {
	t.Helper()
	issue, err := model.NewJudicialCitationIssue(model.JudicialCitationIssueValues{
		Direction: model.JudicialCitationIssueDirection(direction),
		Stage:     "official_pdf_text",
		Code:      "parser_failed",
		Message:   "判例本文を解析できませんでした",
		Retryable: false,
	})
	if err != nil {
		t.Fatalf("issue を作成できません: %v", err)
	}
	return issue
}
