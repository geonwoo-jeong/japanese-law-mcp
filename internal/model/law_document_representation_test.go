package model_test

import (
	"encoding/json"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLawDocumentRepresentation(t *testing.T) {
	got, err := model.NewLawDocumentRepresentation(model.LawDocumentRepresentationValues{
		Law:      newLawSummary(t),
		AsOf:     newDatePointer(t, "2026-01-01"),
		Format:   model.LawDocumentFormatXML,
		Content:  `<Law Era="Reiwa"></Law>`,
		Citation: newCitation(t),
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-017: NewLawDocumentRepresentation() のエラー = %v", err)
	}
	asOf, exists := got.AsOf()
	if !exists || asOf.String() != "2026-01-01" {
		t.Fatalf("SOT-MODEL-017: asOf = %q, %t", asOf.String(), exists)
	}
}

func TestLawDocumentRepresentationRejectsMismatchedCitation(t *testing.T) {
	otherSource := newDocumentSupplementaryLegalSource(t)
	citation, err := model.NewCitation(model.CitationValues{
		Source:     otherSource,
		LawID:      "325AC0000000105",
		RevisionID: "325AC0000000105_20250401_505AC0000000044",
		URL:        "https://example.com/other",
	})
	if err != nil {
		t.Fatalf("citation を作成できない: %v", err)
	}
	if _, err := model.NewLawDocumentRepresentation(model.LawDocumentRepresentationValues{
		Law:      newLawSummary(t),
		Format:   model.LawDocumentFormatXML,
		Content:  `<Law></Law>`,
		Citation: citation,
	}); err == nil {
		t.Fatal("SOT-MODEL-017: citation.source が一致しない表現を受理した")
	}
}

func TestLawDocumentRepresentationRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.LawDocumentRepresentation
	if err := json.Unmarshal([]byte(`{}`), &got); err == nil {
		t.Fatal("SOT-MODEL-009: LawDocumentRepresentation を JSON から直接復元できた")
	}
}
