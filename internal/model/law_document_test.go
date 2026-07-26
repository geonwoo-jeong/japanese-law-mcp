package model_test

import "testing"

import "github.com/geonwoo-jeong/japanese-law-mcp/internal/model"

func TestNewLawDocumentFromRepresentation(t *testing.T) {
	representation, err := model.NewLawDocumentRepresentation(model.LawDocumentRepresentationValues{
		Law:      newLawSummary(t),
		AsOf:     newDatePointer(t, "2026-01-01"),
		Format:   model.LawDocumentFormatXML,
		Content:  `<Law></Law>`,
		Citation: newCitation(t),
	})
	if err != nil {
		t.Fatalf("representation を作成できない: %v", err)
	}
	got, err := model.NewLawDocumentFromRepresentation(representation)
	if err != nil {
		t.Fatalf("SOT-MODEL-002/017: NewLawDocumentFromRepresentation() のエラー = %v", err)
	}
	if got.Format() != model.LawDocumentFormatXML {
		t.Fatalf("SOT-MODEL-002: format = %q", got.Format())
	}
}

func TestNewLawDocumentFromRepresentationRejectsNonXML(t *testing.T) {
	representation, err := model.NewLawDocumentRepresentation(model.LawDocumentRepresentationValues{
		Law:      newLawSummary(t),
		Format:   model.LawDocumentFormatHTML,
		Content:  `<div>law</div>`,
		Citation: newCitation(t),
	})
	if err != nil {
		t.Fatalf("representation を作成できない: %v", err)
	}
	if _, err := model.NewLawDocumentFromRepresentation(representation); err == nil {
		t.Fatal("SOT-MODEL-002/017: html を LawDocument へ投影できた")
	}
}
