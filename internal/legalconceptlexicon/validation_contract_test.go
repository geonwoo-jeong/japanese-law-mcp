package legalconceptlexicon

import (
	"strings"
	"testing"
)

func TestLoadRejectsMissingOrNullRequiredPacks(t *testing.T) {
	t.Parallel()

	missingRequiredPacks := strings.Replace(
		validFixture,
		`"officialTerm": "永住許可",
          "requiredPacks": []`,
		`"officialTerm": "永住許可"`,
		1,
	)
	if missingRequiredPacks == validFixture {
		t.Fatal("requiredPacks の削除対象が見つかりません")
	}
	nullRequiredPacks := replaceOnce(
		t,
		validFixture,
		`"requiredPacks": []`,
		`"requiredPacks": null`,
	)

	for name, value := range map[string]string{
		"missing": missingRequiredPacks,
		"null":    nullRequiredPacks,
	} {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load([]byte(value)); err == nil {
				t.Fatal("SOT-ENG-023: requiredPacks のない候補を受理しました")
			}
		})
	}
}

func TestLoadRejectsDuplicateCandidateAndUnsortedConcepts(t *testing.T) {
	t.Parallel()

	duplicateCandidate := replaceOnce(
		t,
		validFixture,
		`"resource": "judicial_decision",
          "inputKind": "judicial_decision_search",
          "officialTerm": "永住許可",
          "requiredPacks": ["judicial-cases"]`,
		`"resource": "law_provision",
          "inputKind": "law_content_search",
          "officialTerm": "永住許可",
          "requiredPacks": []`,
	)
	unsortedConcepts := strings.NewReplacer(
		`"conceptId": "permanent-residence",`,
		`"conceptId": "z-concept",`,
		`"conceptId": "permanent-residence-permission",`,
		`"conceptId": "a-concept",`,
	).Replace(validFixture)

	for name, value := range map[string]string{
		"duplicate candidate": duplicateCandidate,
		"unsorted concepts":   unsortedConcepts,
	} {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load([]byte(value)); err == nil {
				t.Fatal("SOT-ENG-023: 決定性を壊す辞書を受理しました")
			}
		})
	}
}

func TestLoadRejectsDuplicateConceptAndUnsafeSourceURL(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"duplicate conceptId": replaceOnce(
			t,
			validFixture,
			`"conceptId": "permanent-residence-permission"`,
			`"conceptId": "permanent-residence"`,
		),
		"URL userinfo": replaceOnce(
			t,
			validFixture,
			`https://www.moj.go.jp/isa/applications/procedures/16-4.html`,
			`https://user@www.moj.go.jp/isa/applications/procedures/16-4.html`,
		),
		"invalid confirmedAt": replaceOnce(
			t,
			validFixture,
			`"confirmedAt": "2026-07-28"`,
			`"confirmedAt": "2026-02-30"`,
		),
		"invalid lexiconVersion": replaceOnce(
			t,
			validFixture,
			`"lexiconVersion": "legal-concept-2026-07-28"`,
			`"lexiconVersion": "Legal Concept 2026"`,
		),
	}

	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load([]byte(value)); err == nil {
				t.Fatal("SOT-ENG-023: 不正な識別子又は出典を受理しました")
			}
		})
	}
}

func TestLoadRejectsInvalidPackDefinitions(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"law candidate requires pack": replaceOnce(
			t,
			validFixture,
			`"requiredPacks": []`,
			`"requiredPacks": ["judicial-cases"]`,
		),
		"judicial candidate omits pack": replaceOnce(
			t,
			validFixture,
			`"requiredPacks": ["judicial-cases"]`,
			`"requiredPacks": []`,
		),
		"packs are unsorted": replaceOnce(
			t,
			validFixture,
			`"requiredPacks": ["judicial-cases"]`,
			`"requiredPacks": ["z-pack", "a-pack"]`,
		),
		"packs are duplicated": replaceOnce(
			t,
			validFixture,
			`"requiredPacks": ["judicial-cases"]`,
			`"requiredPacks": ["judicial-cases", "judicial-cases"]`,
		),
	}

	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load([]byte(value)); err == nil {
				t.Fatal("SOT-ENG-023: 不正な requiredPacks を受理しました")
			}
		})
	}
}

func TestLoadRejectsMissingOversizedAndTrailingDatasets(t *testing.T) {
	t.Parallel()

	tests := map[string][]byte{
		"empty":         nil,
		"oversized":     make([]byte, maxDatasetBytes+1),
		"invalid UTF-8": {0xff},
		"trailing JSON": append(append([]byte{}, []byte(validFixture)...), []byte(` {}`)...),
		"empty entries": []byte(`{
		  "schemaVersion": 1,
		  "lexiconVersion": "legal-concept-2026-07-28",
		  "generatedAt": "2026-07-28T00:00:00Z",
		  "entries": []
		}`),
	}

	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load(value); err == nil {
				t.Fatal("SOT-ENG-023: 不正な dataset 境界を受理しました")
			}
		})
	}
}
