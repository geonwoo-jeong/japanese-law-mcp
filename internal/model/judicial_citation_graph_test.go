package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestJudicialCitationGraphResultCopiesAndMarshalsJSON(t *testing.T) {
	t.Parallel()

	ref := newJudicialDecisionRef(t, "12345")
	summary := newJudicialDecisionSummaryForCitation(t, "12345", "令和6年（受）第1号")
	node, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:          "node-root",
		NodeType:        model.JudicialCitationNodeTypeDecision,
		Label:           "最高裁判所 令和6年（受）第1号",
		Ref:             &ref,
		DecisionSummary: &summary,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-035: NewJudicialCitationNode() のエラー = %v", err)
	}
	lawLocation, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: "177",
	})
	if err != nil {
		t.Fatalf("条文位置を構築できません: %v", err)
	}
	lawReference, err := model.NewJudicialCitationLawReference(
		model.JudicialCitationLawReferenceValues{
			LawID:    "129AC0000000089",
			LawTitle: "民法",
			Location: lawLocation,
		},
	)
	if err != nil {
		t.Fatalf("法条参照を構築できません: %v", err)
	}
	lawNode, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:       "node-law",
		NodeType:     model.JudicialCitationNodeTypeLawProvision,
		Label:        "民法第177条",
		LawReference: &lawReference,
	})
	if err != nil {
		t.Fatalf("法条 node を構築できません: %v", err)
	}
	provenance := newCitationProvenance(t, ref.Key())
	excerpt := "参照法条 民法第177条"
	evidence, err := model.NewJudicialCitationEvidence(
		model.JudicialCitationEvidenceValues{
			EvidenceLevel: model.JudicialCitationEvidenceLevelOfficialMetadata,
			Provenance:    provenance,
			Excerpt:       &excerpt,
		},
	)
	if err != nil {
		t.Fatalf("根拠を構築できません: %v", err)
	}
	edge, err := model.NewJudicialCitationEdge(model.JudicialCitationEdgeValues{
		EdgeID:       "edge-1",
		FromNodeID:   "node-root",
		ToNodeID:     "node-law",
		RelationType: model.JudicialCitationRelationTypeReferencesLawProvision,
		Evidence:     []model.JudicialCitationEvidence{evidence},
	})
	if err != nil {
		t.Fatalf("edge を構築できません: %v", err)
	}
	yearBucket, _ := model.NewJudicialCitationYearBucket(2026, 1)
	categoryBucket, _ := model.NewJudicialCitationCategoryBucket(
		model.JudicialPublicationCategorySupremeCourt,
		1,
	)
	summaryBlock, err := model.NewJudicialCitationSummary(model.JudicialCitationSummaryValues{
		ConfirmedOutgoingDecisionCount: 0,
		IncomingCandidateCount:         1,
		ReferencedProvisionCount:       1,
		LowerCourtRelationCount:        0,
		UnresolvedMentionCount:         0,
		IncomingObservedYearBuckets:    []model.JudicialCitationYearBucket{yearBucket},
		IncomingObservedCategoryBuckets: []model.JudicialCitationCategoryBucket{
			categoryBucket,
		},
	})
	if err != nil {
		t.Fatalf("summary を構築できません: %v", err)
	}
	outgoingCoverage, _ := model.NewJudicialCitationDirectionCoverage(
		model.JudicialCitationDirectionCoverageValues{
			Status:    model.JudicialCitationDirectionStatusComplete,
			Methods:   []model.JudicialCitationMethod{model.JudicialCitationMethodOfficialDetailMetadata},
			Truncated: false,
		},
	)
	limit := 3
	attempted := 1
	completed := 1
	incomingCoverage, _ := model.NewJudicialCitationDirectionCoverage(
		model.JudicialCitationDirectionCoverageValues{
			Status:            model.JudicialCitationDirectionStatusComplete,
			Methods:           []model.JudicialCitationMethod{model.JudicialCitationMethodOfficialCaseSearch},
			Truncated:         false,
			Limit:             &limit,
			AttemptedSearches: &attempted,
			CompletedSearches: &completed,
		},
	)
	coverage, err := model.NewJudicialCitationCoverage(
		model.JudicialCitationCoverageValues{
			RequestedDirection: model.JudicialCitationRequestedDirectionBoth,
			Outgoing:           outgoingCoverage,
			Incoming:           incomingCoverage,
		},
	)
	if err != nil {
		t.Fatalf("coverage を構築できません: %v", err)
	}
	graph, err := model.NewJudicialCitationGraph(model.JudicialCitationGraphValues{
		RootNodeID:         "node-root",
		Nodes:              []model.JudicialCitationNode{node, lawNode},
		Edges:              []model.JudicialCitationEdge{edge},
		UnresolvedMentions: []model.JudicialCitationUnresolvedMention{},
		Summary:            summaryBlock,
		Coverage:           coverage,
	})
	if err != nil {
		t.Fatalf("graph を構築できません: %v", err)
	}
	result, err := model.NewJudicialCitationGraphResult(
		model.JudicialCitationGraphResultValues{
			Status:         model.JudicialCitationResultStatusComplete,
			CoverageNotice: "固定注意文",
			Graph:          graph,
			Issues:         []model.JudicialCitationIssue{},
		},
	)
	if err != nil {
		t.Fatalf("result を構築できません: %v", err)
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("Validate() のエラー = %v", err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("JSON を再解析できません: %v", err)
	}
	if object["status"] != "complete" || object["coverageNotice"] != "固定注意文" {
		t.Fatalf("SOT-MODEL-035: result JSON = %#v", object)
	}
	graphObject, ok := object["graph"].(map[string]any)
	if !ok || graphObject["rootNodeId"] != "node-root" {
		t.Fatalf("SOT-MODEL-035: graph JSON = %#v", graphObject)
	}
}

func TestJudicialCitationGraphRejectsInvalidStructures(t *testing.T) {
	t.Parallel()

	_, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:   "ref-only",
		NodeType: model.JudicialCitationNodeTypeDecisionReference,
		Label:    "x",
	})
	if err == nil {
		t.Fatal("SOT-MODEL-035: referenceText のない judicial_decision_reference を受理しました")
	}

	notRequested, err := model.NewJudicialCitationDirectionCoverage(
		model.JudicialCitationDirectionCoverageValues{
			Status:  model.JudicialCitationDirectionStatusNotRequested,
			Methods: []model.JudicialCitationMethod{model.JudicialCitationMethodOfficialCaseSearch},
		},
	)
	if err == nil || notRequested.Validate() == nil {
		t.Fatal("SOT-MODEL-035: not_requested で methods を持つ coverage を受理しました")
	}
}

func newCitationProvenance(
	t *testing.T,
	key model.SourceResourceKey,
) model.Provenance {
	t.Helper()
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         newJudicialInformationSource(t),
		ResourceKey:    key,
		URL:            "https://www.courts.go.jp/hanrei/12345/detail2/index.html",
		RetrievedAt:    time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-045",
	})
	if err != nil {
		t.Fatalf("provenance を構築できません: %v", err)
	}
	return provenance
}

func newJudicialDecisionRef(t *testing.T, decisionID string) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   decisionID + ":detail2",
	})
	if err != nil {
		t.Fatalf("resource key を構築できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("source ref を構築できません: %v", err)
	}
	return ref
}

func newJudicialDecisionSummaryForCitation(
	t *testing.T,
	decisionID string,
	caseNumber string,
) model.JudicialDecisionSummary {
	t.Helper()
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          decisionID,
		PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
		SourceCategoryLabel: "最高裁判例",
		CaseNumber:          caseNumber,
		DecisionDate:        newDate(t, "2026-08-26"),
		CourtName:           "最高裁判所",
		DetailURL:           "https://www.courts.go.jp/hanrei/" + decisionID + "/detail2/index.html",
		Documents:           []model.JudicialDocumentLink{},
		Source:              newJudicialInformationSource(t),
	})
	if err != nil {
		t.Fatalf("summary を構築できません: %v", err)
	}
	return summary
}
