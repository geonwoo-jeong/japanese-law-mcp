package model_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestJudicialCitationDecisionMentionPreservesStrictIdentityAndEvidence(t *testing.T) {
	t.Parallel()

	evidence := exactDecisionMentionEvidence(t, "令和6年（受）第123号")
	mention, err := model.NewJudicialCitationDecisionMention(
		model.JudicialCitationDecisionMentionValues{
			ReferenceText:        "令和6年（受）第123号",
			DecisionIdentityText: "令和6(受)123",
			Evidence:             evidence,
		},
	)
	if err != nil {
		t.Fatalf("有効な判例言及を拒否しました: %v", err)
	}
	if mention.ReferenceText() != "令和6年（受）第123号" ||
		mention.DecisionIdentityText() != "令和6(受)123" ||
		mention.Evidence().EvidenceLevel() != model.JudicialCitationEvidenceLevelExactTextMatch {
		t.Fatalf("判例言及の値が変化しました: %#v", mention)
	}
	encoded, err := json.Marshal(mention)
	if err != nil {
		t.Fatalf("判例言及を JSON 化できません: %v", err)
	}
	if !strings.Contains(string(encoded), `"decisionIdentityText":"令和6(受)123"`) {
		t.Fatalf("判例言及 JSON が identity を保持していません: %s", encoded)
	}
}

func TestJudicialCitationDecisionMentionRejectsUnsafeOrOversizedText(t *testing.T) {
	t.Parallel()

	evidence := exactDecisionMentionEvidence(t, "令和6年（受）第123号")
	tests := []struct {
		name          string
		referenceText string
		identityText  string
	}{
		{name: "空の原文", referenceText: "", identityText: "令和6(受)123"},
		{name: "空白だけの原文", referenceText: " \t", identityText: "令和6(受)123"},
		{name: "原文の制御文字", referenceText: "令和6\n(受)123", identityText: "令和6(受)123"},
		{name: "identityの制御文字", referenceText: "令和6(受)123", identityText: "令和6\u0000(受)123"},
		{name: "不正UTF-8の原文", referenceText: string([]byte{0xff}), identityText: "令和6(受)123"},
		{name: "不正UTF-8のidentity", referenceText: "令和6(受)123", identityText: string([]byte{0xff})},
		{name: "過大な原文", referenceText: strings.Repeat("a", 4097), identityText: "令和6(受)123"},
		{name: "過大なidentity", referenceText: "令和6(受)123", identityText: strings.Repeat("a", 4097)},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := model.NewJudicialCitationDecisionMention(
				model.JudicialCitationDecisionMentionValues{
					ReferenceText:        test.referenceText,
					DecisionIdentityText: test.identityText,
					Evidence:             evidence,
				},
			); err == nil {
				t.Fatal("安全でない判例言及を受理しました")
			}
		})
	}
}

func TestJudicialCitationDecisionMentionAcceptsTextSizeBoundary(t *testing.T) {
	t.Parallel()

	boundary := strings.Repeat("a", 4096)
	if _, err := model.NewJudicialCitationDecisionMention(
		model.JudicialCitationDecisionMentionValues{
			ReferenceText:        boundary,
			DecisionIdentityText: boundary,
			Evidence:             exactDecisionMentionEvidence(t, "令和6(受)123"),
		},
	); err != nil {
		t.Fatalf("4096 byte の判例言及を拒否しました: %v", err)
	}
}

func TestJudicialCitationDecisionMentionRejectsWrongEvidenceAndDirectRestore(t *testing.T) {
	t.Parallel()

	fixture := newJudicialCitationFixture(t)
	metadataEdge := fixture.newEdge(
		t,
		"metadata",
		"root",
		"law",
		"references_law_provision",
		"official_metadata",
		"参照法条",
	)
	if _, err := model.NewJudicialCitationDecisionMention(
		model.JudicialCitationDecisionMentionValues{
			ReferenceText:        "令和6(受)123",
			DecisionIdentityText: "令和6(受)123",
			Evidence:             metadataEdge.Evidence()[0],
		},
	); err == nil {
		t.Fatal("exact_text_match 以外の evidence を受理しました")
	}

	var zero model.JudicialCitationDecisionMention
	if zero.Validate() == nil {
		t.Fatal("JudicialCitationDecisionMention のゼロ値を受理しました")
	}
	if json.Unmarshal(
		[]byte(`{"referenceText":"令和6(受)123","decisionIdentityText":"令和6(受)123","evidence":{}}`),
		&zero,
	) == nil {
		t.Fatal("JudicialCitationDecisionMention の直接 JSON 復元を受理しました")
	}
}

func TestJudicialCitationDecisionMentionEvidenceExcerptBoundary(t *testing.T) {
	t.Parallel()

	if _, err := newExactDecisionMentionEvidence(t, strings.Repeat("a", 256)); err != nil {
		t.Fatalf("256 byte の excerpt を拒否しました: %v", err)
	}
	if _, err := newExactDecisionMentionEvidence(t, strings.Repeat("a", 257)); err == nil {
		t.Fatal("256 byte を超える excerpt を受理しました")
	}
}

func TestJudicialCitationDecisionMentionRequiresPDFExtractionEvidence(t *testing.T) {
	t.Parallel()

	valid := exactDecisionMentionEvidence(t, "令和6(受)123")
	values := model.JudicialCitationDecisionMentionValues{
		ReferenceText:        "令和6(受)123",
		DecisionIdentityText: "令和6(受)123",
		Evidence:             valid,
	}
	withoutExcerpt := newDecisionMentionEvidence(t, "application/pdf", "extracted", nil, true)
	nonPDFExcerpt := "令和6(受)123"
	nonPDF := newDecisionMentionEvidence(t, "text/plain", "extracted", &nonPDFExcerpt, true)
	nonExtracted := newDecisionMentionEvidence(t, "application/pdf", "unchanged", &nonPDFExcerpt, true)
	withoutDigest := newDecisionMentionEvidence(t, "application/pdf", "extracted", &nonPDFExcerpt, false)
	for name, evidence := range map[string]model.JudicialCitationEvidence{
		"excerptなし":       withoutExcerpt,
		"PDF以外":           nonPDF,
		"extracted以外":     nonExtracted,
		"contentDigestなし": withoutDigest,
	} {
		name, evidence := name, evidence
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			candidate := values
			candidate.Evidence = evidence
			if _, err := model.NewJudicialCitationDecisionMention(candidate); err == nil {
				t.Fatal("PDF 抽出根拠の条件を満たさない evidence を受理しました")
			}
		})
	}
}

func exactDecisionMentionEvidence(
	t *testing.T,
	excerpt string,
) model.JudicialCitationEvidence {
	t.Helper()
	evidence, err := newExactDecisionMentionEvidence(t, excerpt)
	if err != nil {
		t.Fatalf("exact_text_match evidence を作成できません: %v", err)
	}
	return evidence
}

func newExactDecisionMentionEvidence(
	t *testing.T,
	excerpt string,
) (model.JudicialCitationEvidence, error) {
	t.Helper()
	provenance := newDecisionMentionProvenance(
		t,
		model.JudicialDocumentMediaTypePDF,
		model.ProvenanceTransformationExtracted,
		true,
	)
	return model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelExactTextMatch,
		Provenance:    provenance,
		Excerpt:       &excerpt,
	})
}

func newDecisionMentionEvidence(
	t *testing.T,
	mediaType string,
	transformation model.ProvenanceTransformation,
	excerpt *string,
	withDigest bool,
) model.JudicialCitationEvidence {
	t.Helper()
	provenance := newDecisionMentionProvenance(t, mediaType, transformation, withDigest)
	evidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelExactTextMatch,
		Provenance:    provenance,
		Excerpt:       excerpt,
	})
	if err != nil {
		t.Fatalf("判例言及用 evidence を作成できません: %v", err)
	}
	return evidence
}

func newDecisionMentionProvenance(
	t *testing.T,
	mediaType string,
	transformation model.ProvenanceTransformation,
	withDigest bool,
) model.Provenance {
	t.Helper()
	fixture := newJudicialCitationFixture(t)
	methodID := ""
	if transformation != model.ProvenanceTransformationUnchanged {
		methodID = "SOT-IF-071"
	}
	digest := ""
	if withDigest {
		digest = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         fixture.provenance.Source(),
		ResourceKey:    fixture.provenance.ResourceKey(),
		URL:            "https://www.courts.go.jp/assets/hanrei/00001.pdf",
		RetrievedAt:    time.Date(2026, 8, 27, 1, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		MediaType:      mediaType,
		Transformation: transformation,
		MethodID:       methodID,
		ContentDigest:  digest,
	})
	if err != nil {
		t.Fatalf("判例言及用 provenance を作成できません: %v", err)
	}
	return provenance
}
