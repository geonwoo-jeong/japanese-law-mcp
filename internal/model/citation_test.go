package model_test

import (
	"encoding/json"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestCitation(t *testing.T) {
	source := newDocumentLegalSource(t)
	got, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      "325AC0000000105",
		RevisionID: "325AC0000000105_20250401_505AC0000000044",
		Location:   "第1条",
		URL:        "https://laws.e-gov.go.jp/law/325AC0000000105/20250401_505AC0000000044",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-004: NewCitation() のエラー = %v", err)
	}
	location, exists := got.Location()
	if !exists || location != "第1条" {
		t.Fatalf("SOT-MODEL-004: location = %q, %t", location, exists)
	}
}

func TestCitationRejectsInvalidValues(t *testing.T) {
	source := newDocumentLegalSource(t)
	testCases := map[string]model.CitationValues{
		"empty_law_id": {
			Source:     source,
			RevisionID: "rev",
			URL:        "https://example.com/law",
		},
		"empty_revision_id": {
			Source: source,
			LawID:  "law",
			URL:    "https://example.com/law",
		},
		"invalid_url": {
			Source:     source,
			LawID:      "law",
			RevisionID: "rev",
			URL:        "http://example.com/law",
		},
	}
	for name, values := range testCases {
		t.Run(name, func(t *testing.T) {
			if _, err := model.NewCitation(values); err == nil {
				t.Fatalf("SOT-MODEL-004: NewCitation(%#v) が成功した", values)
			}
		})
	}
}

func TestCitationRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.Citation
	if err := json.Unmarshal([]byte(`{}`), &got); err == nil {
		t.Fatal("SOT-MODEL-009: Citation を JSON から直接復元できた")
	}
}
