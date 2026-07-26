package model_test

import (
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func newDocumentLegalSource(t *testing.T) model.LegalSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できない: %v", err)
	}
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("LegalSource を作成できない: %v", err)
	}
	return legalSource
}

func newDocumentSupplementaryLegalSource(t *testing.T) model.LegalSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "supplementary-law-source",
		Name:       "Supplementary Law Source",
		Publisher:  "Example Publisher",
		Authority:  model.AuthoritySupplementary,
		ServiceURL: "https://example.com/law",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できない: %v", err)
	}
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("LegalSource を作成できない: %v", err)
	}
	return legalSource
}

func newLawSummary(t *testing.T) model.LawSummary {
	t.Helper()
	effectiveDate, err := model.NewDate("2025-04-01")
	if err != nil {
		t.Fatalf("date を作成できない: %v", err)
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:                 "325AC0000000105",
		RevisionID:            "325AC0000000105_20250401_505AC0000000044",
		Title:                 "行政手続法",
		RevisionEffectiveDate: &effectiveDate,
		Source:                newDocumentLegalSource(t),
	})
	if err != nil {
		t.Fatalf("LawSummary を作成できない: %v", err)
	}
	return summary
}

func newCitation(t *testing.T) model.Citation {
	t.Helper()
	citation, err := model.NewCitation(model.CitationValues{
		Source:     newDocumentLegalSource(t),
		LawID:      "325AC0000000105",
		RevisionID: "325AC0000000105_20250401_505AC0000000044",
		URL:        "https://laws.e-gov.go.jp/law/325AC0000000105/20250401_505AC0000000044",
	})
	if err != nil {
		t.Fatalf("Citation を作成できない: %v", err)
	}
	return citation
}

func newDatePointer(t *testing.T, value string) *model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("date を作成できない: %v", err)
	}
	return &date
}
