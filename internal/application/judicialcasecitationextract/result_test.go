package judicialcasecitationextract

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestResultAcceptsAvailableAndUnavailableStates(t *testing.T) {
	t.Parallel()

	mention := testDecisionMention(t)
	unresolved := testUnresolvedMention(t)
	result, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{mention},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{unresolved},
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         3,
		OccurrenceCount:           2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OccurrenceCount() != 2 || len(result.ConfirmedDecisionMentions()) != 1 {
		t.Fatalf("Result accessor = %#v", result)
	}

	unavailable, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusDocumentTextUnavailable,
		ExaminedPageCount:         1,
		OccurrenceCount:           0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if unavailable.DocumentTextStatus() != DocumentTextStatusDocumentTextUnavailable {
		t.Fatal("documentTextStatus を保持していません")
	}
}

func TestResultRejectsImpossibleStatesAndDirectRestore(t *testing.T) {
	t.Parallel()

	mention := testDecisionMention(t)
	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{mention},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusDocumentTextUnavailable,
		ExaminedPageCount:         1,
		OccurrenceCount:           1,
	}); err == nil {
		t.Fatal("document_text_unavailable なのに mention ありを受理しました")
	}
	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{mention},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         1,
		OccurrenceCount:           2,
	}); err == nil {
		t.Fatal("配列合計と異なる occurrenceCount を受理しました")
	}

	result, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{mention},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         1,
		OccurrenceCount:           1,
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(result)
	if err != nil || strings.Contains(string(encoded), "full_text_document_unavailable") {
		t.Fatalf("Result JSON = %s, %v", encoded, err)
	}
	var decoded Result
	if json.Unmarshal(encoded, &decoded) == nil || decoded.Validate() == nil {
		t.Fatal("Result の直接復元またはゼロ値を受理しました")
	}
}

func TestResultEnforcesPageAndOccurrenceBoundaries(t *testing.T) {
	t.Parallel()

	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         300,
	}); err != nil {
		t.Fatalf("300 page の結果を拒否しました: %v", err)
	}
	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         301,
	}); err == nil {
		t.Fatal("300 page を超える結果を受理しました")
	}

	mention := testDecisionMention(t)
	mentions := make([]model.JudicialCitationDecisionMention, 256)
	for index := range mentions {
		mentions[index] = mention
	}
	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: mentions,
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         1,
		OccurrenceCount:           256,
		Truncated:                 true,
	}); err != nil {
		t.Fatalf("256 occurrence の結果を拒否しました: %v", err)
	}
	mentions = append(mentions, mention)
	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: mentions,
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         1,
		OccurrenceCount:           257,
		Truncated:                 true,
	}); err == nil {
		t.Fatal("256 occurrence を超える結果を受理しました")
	}
}

func TestResultRequiresNonNilArraysAndUnavailableInvariant(t *testing.T) {
	t.Parallel()

	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: nil,
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusAvailable,
	}); err == nil {
		t.Fatal("nil の confirmedDecisionMentions を受理しました")
	}
	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
		UnresolvedMentions:        nil,
		DocumentTextStatus:        DocumentTextStatusAvailable,
	}); err == nil {
		t.Fatal("nil の unresolvedMentions を受理しました")
	}
	if (Result{
		confirmedDecisionMentions: nil,
		unresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		documentTextStatus:        DocumentTextStatusAvailable,
		initialized:               true,
	}).Validate() == nil {
		t.Fatal("Validate が nil 配列を受理しました")
	}
	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusDocumentTextUnavailable,
		ExaminedPageCount:         0,
		Truncated:                 true,
	}); err == nil {
		t.Fatal("document_text_unavailable の truncated=true を受理しました")
	}
	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        "unknown",
	}); err == nil {
		t.Fatal("未知の documentTextStatus を受理しました")
	}
	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         -1,
	}); err == nil {
		t.Fatal("負の examinedPageCount を受理しました")
	}
}

func TestResultOnlyMarksOccurrenceLimitAsTruncated(t *testing.T) {
	t.Parallel()

	if _, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: []model.JudicialCitationDecisionMention{},
		UnresolvedMentions:        []model.JudicialCitationUnresolvedMention{},
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         1,
		Truncated:                 true,
	}); err == nil {
		t.Fatal("出現上限未満なのに truncated=true の結果を受理しました")
	}
}

func TestResultPreservesMentionOrderAndOwnsSlices(t *testing.T) {
	t.Parallel()

	first := testDecisionMentionWithText(t, "令和6(受)1")
	second := testDecisionMentionWithText(t, "令和6(受)2")
	firstUnresolved := testUnresolvedMentionWithText(t, "平成99(受)1")
	secondUnresolved := testUnresolvedMentionWithText(t, "平成99(受)2")
	confirmedInput := []model.JudicialCitationDecisionMention{first, second}
	unresolvedInput := []model.JudicialCitationUnresolvedMention{firstUnresolved, secondUnresolved}
	result, err := NewResult(ResultValues{
		ConfirmedDecisionMentions: confirmedInput,
		UnresolvedMentions:        unresolvedInput,
		DocumentTextStatus:        DocumentTextStatusAvailable,
		ExaminedPageCount:         2,
		OccurrenceCount:           4,
	})
	if err != nil {
		t.Fatal(err)
	}
	confirmedInput[0] = second
	unresolvedInput[0] = secondUnresolved
	confirmedOutput := result.ConfirmedDecisionMentions()
	unresolvedOutput := result.UnresolvedMentions()
	confirmedOutput[1] = first
	unresolvedOutput[1] = firstUnresolved
	if result.ConfirmedDecisionMentions()[0].ReferenceText() != "令和6(受)1" ||
		result.ConfirmedDecisionMentions()[1].ReferenceText() != "令和6(受)2" ||
		result.UnresolvedMentions()[0].MentionText() != "平成99(受)1" ||
		result.UnresolvedMentions()[1].MentionText() != "平成99(受)2" {
		t.Fatal("Result の順序または slice 所有権が保たれていません")
	}
}

func testDecisionMention(t *testing.T) model.JudicialCitationDecisionMention {
	t.Helper()
	return testDecisionMentionWithText(t, "令和6(受)123")
}

func testDecisionMentionWithText(t *testing.T, text string) model.JudicialCitationDecisionMention {
	t.Helper()
	_, _ = testDecisionResource(t)
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         testHanreiSource(t),
		ResourceKey:    testDecisionKey(t),
		URL:            "https://www.courts.go.jp/assets/hanrei/00001.pdf",
		RetrievedAt:    time.Date(2026, 8, 26, 12, 1, 0, 0, time.FixedZone("JST", 9*60*60)),
		MediaType:      model.JudicialDocumentMediaTypePDF,
		Location:       "page=1",
		Transformation: model.ProvenanceTransformationExtracted,
		MethodID:       "SOT-IF-071",
		ContentDigest:  "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	})
	if err != nil {
		t.Fatal(err)
	}
	excerpt := text + "を引用する。"
	evidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelExactTextMatch,
		Provenance:    provenance,
		Excerpt:       &excerpt,
	})
	if err != nil {
		t.Fatal(err)
	}
	mention, err := model.NewJudicialCitationDecisionMention(model.JudicialCitationDecisionMentionValues{
		ReferenceText:        text,
		DecisionIdentityText: text,
		Evidence:             evidence,
	})
	if err != nil {
		t.Fatal(err)
	}
	return mention
}

func testUnresolvedMention(t *testing.T) model.JudicialCitationUnresolvedMention {
	t.Helper()
	return testUnresolvedMentionWithText(t, "平成99(受)1")
}

func testUnresolvedMentionWithText(t *testing.T, text string) model.JudicialCitationUnresolvedMention {
	t.Helper()
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         testHanreiSource(t),
		ResourceKey:    testDecisionKey(t),
		URL:            "https://www.courts.go.jp/assets/hanrei/00001.pdf",
		RetrievedAt:    time.Date(2026, 8, 26, 12, 2, 0, 0, time.FixedZone("JST", 9*60*60)),
		MediaType:      model.JudicialDocumentMediaTypePDF,
		Location:       "page=2",
		Transformation: model.ProvenanceTransformationExtracted,
		MethodID:       "SOT-IF-071",
		ContentDigest:  "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
	})
	if err != nil {
		t.Fatal(err)
	}
	mention, err := model.NewJudicialCitationUnresolvedMention(model.JudicialCitationUnresolvedMentionValues{
		MentionType: model.JudicialCitationMentionTypeDecision,
		MentionText: text,
		Reason:      model.JudicialCitationUnresolvedReasonInsufficientIdentity,
		Provenance:  provenance,
	})
	if err != nil {
		t.Fatal(err)
	}
	return mention
}

func testHanreiSource(t *testing.T) model.InformationSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "courts-hanrei",
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func testDecisionKey(t *testing.T) model.SourceResourceKey {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "00001",
	})
	if err != nil {
		t.Fatal(err)
	}
	return key
}
