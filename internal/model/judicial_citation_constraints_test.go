package model_test

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestJudicialCitationGraphRejectsMixedEvidenceLevelsInOneEdge(t *testing.T) {
	t.Parallel()

	fixture := newJudicialCitationFixture(t)
	exact := fixture.newEdge(
		t, "exact", "root", "reference", "cites_judicial_decision", "exact_text_match", "完全一致",
	)
	metadata := fixture.newEdge(
		t, "metadata", "root", "reference", "cites_judicial_decision", "official_metadata", "メタデータ",
	)
	mixed, err := model.NewJudicialCitationEdge(model.JudicialCitationEdgeValues{
		EdgeID:       "mixed",
		FromNodeID:   "root",
		ToNodeID:     "reference",
		RelationType: "cites_judicial_decision",
		Evidence:     append(exact.Evidence(), metadata.Evidence()...),
	})
	if err != nil {
		t.Fatalf("edge 自体を作成できません: %v", err)
	}
	if _, err := fixture.newGraph(t, []model.JudicialCitationEdge{mixed}, nil); err == nil {
		t.Fatal("relationType と異なる evidenceLevel を含む edge を受理しました")
	}
}

func TestJudicialCitationGraphRejectsSummaryMismatch(t *testing.T) {
	t.Parallel()

	fixture := newJudicialCitationFixture(t)
	edge := fixture.newEdge(
		t, "edge", "root", "law", "references_law_provision", "official_metadata", "参照法条",
	)
	wrongSummary, err := model.NewJudicialCitationSummary(model.JudicialCitationSummaryValues{
		IncomingObservedYearBuckets:     []model.JudicialCitationYearBucket{},
		IncomingObservedCategoryBuckets: []model.JudicialCitationCategoryBucket{},
	})
	if err != nil {
		t.Fatalf("summary を作成できません: %v", err)
	}
	if _, err := fixture.newGraph(t, []model.JudicialCitationEdge{edge}, &wrongSummary); err == nil {
		t.Fatal("edge 件数と一致しない summary を受理しました")
	}
}

func TestJudicialCitationCoverageRejectsDirectionMismatch(t *testing.T) {
	t.Parallel()

	complete := mustDirectionCoverage(t, "complete", []string{"official_pdf_text"})
	if _, err := model.NewJudicialCitationCoverage(model.JudicialCitationCoverageValues{
		RequestedDirection: "outgoing",
		HopDepth:           1,
		Outgoing:           complete,
		Incoming:           complete,
	}); err == nil {
		t.Fatal("要求していない incoming が complete の coverage を受理しました")
	}
}

func TestJudicialCitationNodeEnforcesConditionalFields(t *testing.T) {
	t.Parallel()

	ref, summary := newCitationDecision(t, "300", "令和7年（受）第3号")
	fixture := newJudicialCitationFixture(t)
	lawReference, ok := fixture.nodes[3].LawReference()
	if !ok {
		t.Fatal("fixture の lawReference がありません")
	}
	referenceText := "令和元年（受）第3号"
	tests := []struct {
		name   string
		values model.JudicialCitationNodeValues
	}{
		{
			name: "裁判例にreferenceTextを併記",
			values: model.JudicialCitationNodeValues{
				NodeID: "node", NodeType: "judicial_decision", Label: "裁判例",
				Ref: &ref, DecisionSummary: &summary, ReferenceText: &referenceText,
			},
		},
		{
			name: "法条に裁判例refを併記",
			values: model.JudicialCitationNodeValues{
				NodeID: "node", NodeType: "law_provision", Label: "法条",
				Ref: &ref, LawReference: &lawReference,
			},
		},
		{
			name: "裁判例参照にlawReferenceを併記",
			values: model.JudicialCitationNodeValues{
				NodeID: "node", NodeType: "judicial_decision_reference", Label: "参照",
				LawReference: &lawReference, ReferenceText: &referenceText,
			},
		},
		{
			name: "未知のnodeType",
			values: model.JudicialCitationNodeValues{
				NodeID: "node", NodeType: "unknown", Label: "未知",
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := model.NewJudicialCitationNode(test.values); err == nil {
				t.Fatal("nodeType の条件付き field 契約に反する node を受理しました")
			}
		})
	}
}

func TestJudicialCitationGraphRequiresDecisionRoot(t *testing.T) {
	t.Parallel()

	fixture := newJudicialCitationFixture(t)
	referenceText := "令和元年（受）第3号"
	root, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID: "root", NodeType: "judicial_decision_reference", Label: "参照",
		ReferenceText: &referenceText,
	})
	if err != nil {
		t.Fatalf("reference node を作成できません: %v", err)
	}
	fixture.nodes[0] = root
	if _, err := fixture.newGraph(t, []model.JudicialCitationEdge{}, nil); err == nil {
		t.Fatal("judicial_decision ではない root を受理しました")
	}
}

func TestJudicialCitationGraphRejectsRelationFromUnrequestedDirection(t *testing.T) {
	t.Parallel()

	fixture := newJudicialCitationFixture(t)
	incomingEdge := fixture.newEdge(
		t,
		"incoming",
		"candidate",
		"root",
		"possible_cites_judicial_decision",
		"official_search_candidate",
		"候補",
	)
	summary := judicialCitationSummaryForEdges(t, []model.JudicialCitationEdge{incomingEdge})
	if _, err := model.NewJudicialCitationGraph(model.JudicialCitationGraphValues{
		RootNodeID:         "root",
		Nodes:              fixture.nodes,
		Edges:              []model.JudicialCitationEdge{incomingEdge},
		UnresolvedMentions: []model.JudicialCitationUnresolvedMention{},
		Summary:            summary,
		Coverage:           mustCitationCoverage(t, "complete", "not_requested"),
	}); err == nil {
		t.Fatal("outgoing だけの要求に incoming relation を含む graph を受理しました")
	}

	outgoingEdge := fixture.newEdge(
		t,
		"outgoing",
		"root",
		"reference",
		"cites_judicial_decision",
		"exact_text_match",
		"確認済み引用",
	)
	if _, err := model.NewJudicialCitationGraph(model.JudicialCitationGraphValues{
		RootNodeID:         "root",
		Nodes:              fixture.nodes,
		Edges:              []model.JudicialCitationEdge{outgoingEdge},
		UnresolvedMentions: []model.JudicialCitationUnresolvedMention{},
		Summary:            judicialCitationSummaryForEdges(t, []model.JudicialCitationEdge{outgoingEdge}),
		Coverage:           mustIncomingCitationCoverage(t),
	}); err == nil {
		t.Fatal("incoming だけの要求に outgoing relation を含む graph を受理しました")
	}
}

func TestJudicialCitationSummaryEnforcesObservedBuckets(t *testing.T) {
	t.Parallel()

	year2025, _ := model.NewJudicialCitationYearBucket(2025, 1)
	year2024, _ := model.NewJudicialCitationYearBucket(2024, 1)
	category, _ := model.NewJudicialCitationCategoryBucket(
		model.JudicialPublicationCategorySupremeCourt,
		1,
	)
	tests := []struct {
		name   string
		values model.JudicialCitationSummaryValues
	}{
		{
			name: "年別bucketが降順",
			values: model.JudicialCitationSummaryValues{
				IncomingCandidateCount:          2,
				IncomingObservedYearBuckets:     []model.JudicialCitationYearBucket{year2025, year2024},
				IncomingObservedCategoryBuckets: []model.JudicialCitationCategoryBucket{},
			},
		},
		{
			name: "年別合計が候補数と不一致",
			values: model.JudicialCitationSummaryValues{
				IncomingCandidateCount:          2,
				IncomingObservedYearBuckets:     []model.JudicialCitationYearBucket{year2025},
				IncomingObservedCategoryBuckets: []model.JudicialCitationCategoryBucket{category},
			},
		},
		{
			name: "必須配列がnil",
			values: model.JudicialCitationSummaryValues{
				IncomingObservedYearBuckets: nil,
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := model.NewJudicialCitationSummary(test.values); err == nil {
				t.Fatal("summary の観測 bucket 制約に反する値を受理しました")
			}
		})
	}
	if _, err := model.NewJudicialCitationYearBucket(2025, 0); err == nil {
		t.Fatal("count 0 の year bucket を受理しました")
	}
}

func TestJudicialCitationIncomingCoverageContract(t *testing.T) {
	t.Parallel()

	limit, attempted, completed := 3, 2, 2
	incoming, err := model.NewJudicialCitationDirectionCoverage(
		model.JudicialCitationDirectionCoverageValues{
			Status:            "complete",
			Methods:           []model.JudicialCitationMethod{"official_case_search"},
			Truncated:         true,
			Limit:             &limit,
			AttemptedSearches: &attempted,
			CompletedSearches: &completed,
		},
	)
	if err != nil {
		t.Fatalf("incoming coverage を作成できません: %v", err)
	}
	notRequested := mustDirectionCoverage(t, "not_requested", []string{})
	coverage, err := model.NewJudicialCitationCoverage(model.JudicialCitationCoverageValues{
		RequestedDirection: "incoming",
		Outgoing:           notRequested,
		Incoming:           incoming,
	})
	if err != nil {
		t.Fatalf("有効な incoming coverage を拒否しました: %v", err)
	}
	raw, err := json.Marshal(coverage)
	if err != nil {
		t.Fatalf("coverage を JSON 化できません: %v", err)
	}
	if string(raw) == "" || coverage.HopDepth() != 1 {
		t.Fatalf("coverage JSON 又は hopDepth が不正です: %s", raw)
	}

	failedCompleted := 1
	incomplete, err := model.NewJudicialCitationDirectionCoverage(
		model.JudicialCitationDirectionCoverageValues{
			Status:            "complete",
			Methods:           []model.JudicialCitationMethod{"official_case_search"},
			Limit:             &limit,
			AttemptedSearches: &attempted,
			CompletedSearches: &failedCompleted,
		},
	)
	if err != nil {
		t.Fatalf("単方向 coverage の作成エラー = %v", err)
	}
	if _, err := model.NewJudicialCitationCoverage(model.JudicialCitationCoverageValues{
		RequestedDirection: "incoming",
		Outgoing:           notRequested,
		Incoming:           incomplete,
	}); err == nil {
		t.Fatal("未完了検索を complete とする incoming coverage を受理しました")
	}
}

func TestJudicialCitationPartialRequiresIssueForIncompleteDirection(t *testing.T) {
	t.Parallel()

	fixture := newJudicialCitationFixture(t)
	graph := fixture.newGraphWithCoverage(t, "partial", "not_requested")
	wrongIssue, err := model.NewJudicialCitationIssue(model.JudicialCitationIssueValues{
		Direction: "incoming",
		Stage:     "official_case_search",
		Code:      "search_failed",
		Message:   "候補検索に失敗しました",
	})
	if err != nil {
		t.Fatalf("issue を作成できません: %v", err)
	}
	if _, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         "partial",
		CoverageNotice: "固定注意文",
		Graph:          graph,
		Issues:         []model.JudicialCitationIssue{wrongIssue},
	}); err == nil {
		t.Fatal("未完了方向と一致する issue がない partial を受理しました")
	}
	result, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         "partial",
		CoverageNotice: "固定注意文",
		Graph:          graph,
		Issues:         []model.JudicialCitationIssue{mustIssue(t, "outgoing")},
	})
	if err != nil {
		t.Fatalf("有効な partial を拒否しました: %v", err)
	}
	issues := result.Issues()
	issues[0] = model.JudicialCitationIssue{}
	if result.Issues()[0].Code() != "parser_failed" {
		t.Fatal("result の issues が getter の slice 変更で変化しました")
	}
}
