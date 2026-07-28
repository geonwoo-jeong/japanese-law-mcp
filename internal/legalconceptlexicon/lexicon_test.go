package legalconceptlexicon

import (
	"strings"
	"testing"
)

const validFixture = `{
  "schemaVersion": 1,
  "lexiconVersion": "legal-concept-2026-07-28",
  "generatedAt": "2026-07-28T00:00:00Z",
  "entries": [
    {
      "conceptId": "permanent-residence",
      "canonical": "永住権",
      "terms": ["永住権"],
      "comparisonTerms": ["永住権"],
      "sourceName": "出入国在留管理庁 永住許可申請",
      "sourceUrl": "https://www.moj.go.jp/isa/applications/procedures/16-4.html",
      "confirmedAt": "2026-07-28",
      "mappingNote": "一般に永住権と呼ばれる内容を、法情報検索では永住許可の語で確認する。",
      "conflictGroupId": "permanent-residence-group",
      "selectionPolicy": "ambiguous_no_auto_execute",
      "candidates": [
        {
          "task": "search",
          "resource": "law_provision",
          "inputKind": "law_content_search",
          "officialTerm": "永住許可",
          "requiredPacks": []
        },
        {
          "task": "search",
          "resource": "judicial_decision",
          "inputKind": "judicial_decision_search",
          "officialTerm": "永住許可",
          "requiredPacks": ["judicial-cases"]
        }
      ]
    },
    {
      "conceptId": "permanent-residence-permission",
      "canonical": "永住権",
      "terms": ["永住権"],
      "comparisonTerms": ["永住権"],
      "sourceName": "出入国在留管理庁 永住許可申請",
      "sourceUrl": "https://www.moj.go.jp/isa/applications/procedures/16-4.html",
      "confirmedAt": "2026-07-28",
      "mappingNote": "条文検索の明示意図がある場合は永住許可の法令本文検索へ寄せる。",
      "conflictGroupId": "permanent-residence-group",
      "selectionPolicy": "single_candidate",
      "candidates": [
        {
          "task": "search",
          "resource": "law_provision",
          "inputKind": "law_content_search",
          "officialTerm": "永住許可",
          "requiredPacks": []
        }
      ]
    }
  ]
}`

func TestLoadBuildsImmutableLexicon(t *testing.T) {
	t.Parallel()

	lexicon, err := Load([]byte(validFixture))
	if err != nil {
		t.Fatalf("SOT-ENG-023: Load() error = %v", err)
	}
	if lexicon.Version() != "legal-concept-2026-07-28" {
		t.Fatalf("SOT-ENG-023: version = %q", lexicon.Version())
	}
	entries := lexicon.Entries()
	if len(entries) != 2 {
		t.Fatalf("SOT-ENG-023: entries = %d", len(entries))
	}
	if entries[0].Candidates[1].RequiredPacks[0] != "judicial-cases" {
		t.Fatalf("SOT-ENG-023: candidates = %#v", entries[0].Candidates)
	}
	if len(lexicon.Terms()) != 1 || lexicon.Terms()[0] != "永住権" {
		t.Fatalf("SOT-ENG-023: terms = %#v", lexicon.Terms())
	}
	if len(lexicon.ComparisonTerms()) != 1 ||
		lexicon.ComparisonTerms()[0] != "永住権" {
		t.Fatalf(
			"SOT-ENG-023: comparison terms = %#v",
			lexicon.ComparisonTerms(),
		)
	}

	entries[0].Canonical = "変更後"
	entries[0].Terms[0] = "変更後"
	entries[0].Candidates[0].RequiredPacks = []string{"changed"}
	reloaded := lexicon.Entries()
	if reloaded[0].Canonical != "永住権" || reloaded[0].Terms[0] != "永住権" {
		t.Fatalf("SOT-ENG-023: lexicon mutated = %#v", reloaded[0])
	}
	if len(reloaded[0].Candidates[0].RequiredPacks) != 0 {
		t.Fatalf("SOT-ENG-023: candidates mutated = %#v", reloaded[0].Candidates[0])
	}
	comparisonTerms := lexicon.ComparisonTerms()
	comparisonTerms[0] = "変更後"
	if lexicon.ComparisonTerms()[0] != "永住権" {
		t.Fatalf("SOT-ENG-023: comparison terms mutated = %#v", lexicon.ComparisonTerms())
	}

	var nilLexicon *Lexicon
	if nilLexicon.Version() != "" ||
		nilLexicon.Entries() != nil ||
		nilLexicon.Terms() != nil ||
		nilLexicon.ComparisonTerms() != nil {
		t.Fatal("nil Lexicon が値を返しました")
	}
}

func TestLoadRejectsMalformedDataset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "unknown schema version",
			value: replaceOnce(t, validFixture, `"schemaVersion": 1`, `"schemaVersion": 2`),
		},
		{
			name: "unknown field",
			value: replaceOnce(
				t,
				validFixture,
				`"lexiconVersion": "legal-concept-2026-07-28",`,
				`"lexiconVersion": "legal-concept-2026-07-28", "unknown": true,`,
			),
		},
		{
			name:  "invalid generatedAt offset",
			value: replaceOnce(t, validFixture, `2026-07-28T00:00:00Z`, `2026-07-28T09:00:00+09:00`),
		},
		{
			name:  "trailing json",
			value: validFixture + `{}`,
		},
		{
			name:  "unsorted terms",
			value: replaceOnce(t, validFixture, `"terms": ["永住権"]`, `"terms": ["永住権", "あ"]`),
		},
		{
			name:  "comparison mismatch",
			value: replaceOnce(t, validFixture, `"comparisonTerms": ["永住権"]`, `"comparisonTerms": ["えいじゅうけん"]`),
		},
		{
			name:  "duplicate concept id",
			value: replaceOnce(t, validFixture, `"permanent-residence-permission"`, `"permanent-residence"`),
		},
		{
			name:  "invalid http source",
			value: replaceOnce(t, validFixture, `https://www.moj.go.jp/isa/applications/procedures/16-4.html`, `http://www.moj.go.jp/isa/applications/procedures/16-4.html`),
		},
		{
			name: "userinfo in source url",
			value: replaceOnce(
				t,
				validFixture,
				`https://www.moj.go.jp/isa/applications/procedures/16-4.html`,
				`https://user@example.com/isa/applications/procedures/16-4.html`,
			),
		},
		{
			name: "invalid confirmed date",
			value: replaceOnce(
				t,
				validFixture,
				`"confirmedAt": "2026-07-28"`,
				`"confirmedAt": "2026-02-30"`,
			),
		},
		{
			name:  "single candidate with two candidates",
			value: replaceOnce(t, validFixture, `"selectionPolicy": "single_candidate"`, `"selectionPolicy": "ambiguous_no_auto_execute"`),
		},
		{
			name: "unsupported candidate variant",
			value: replaceOnce(
				t,
				validFixture,
				`"resource": "law_provision"`,
				`"resource": "law"`,
			),
		},
		{
			name: "duplicate required packs",
			value: replaceOnce(
				t,
				validFixture,
				`"requiredPacks": ["judicial-cases"]`,
				`"requiredPacks": ["judicial-cases", "judicial-cases"]`,
			),
		},
		{
			name: "term collision without group",
			value: replaceOnce(
				t,
				validFixture,
				`"conflictGroupId": "permanent-residence-group",`,
				``,
			),
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Load([]byte(testCase.value)); err == nil {
				t.Fatal("SOT-ENG-023: 不正な法概念辞書を受理しました")
			}
		})
	}
}

func TestLoadEmbeddedContainsExpectedOfficialConcepts(t *testing.T) {
	t.Parallel()

	lexicon, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-023: LoadEmbedded() error = %v", err)
	}
	entries := lexicon.Entries()
	if len(entries) != 8 {
		t.Fatalf("SOT-ENG-023: entry count = %d, want 8", len(entries))
	}
	assertEntry(t, entries, "cooling-off", "クーリング・オフ", "申込みの撤回")
	assertEntry(t, entries, "annual-paid-leave", "有休", "年次有給休暇")
	assertEntry(t, entries, "unemployment-basic-allowance", "失業手当", "基本手当")
	assertEntry(t, entries, "adult-guardianship", "成年後見", "成年後見")
	assertEntry(t, entries, "permanent-residence", "永住権", "永住許可")
}

func assertEntry(
	t *testing.T,
	entries []Entry,
	conceptID string,
	term string,
	officialTerm string,
) {
	t.Helper()
	for _, entry := range entries {
		if entry.ConceptID != conceptID {
			continue
		}
		if !contains(entry.Terms, term) {
			t.Fatalf("SOT-ENG-023: terms = %#v", entry.Terms)
		}
		for _, candidate := range entry.Candidates {
			if candidate.OfficialTerm == officialTerm {
				return
			}
		}
		t.Fatalf("SOT-ENG-023: entry = %#v", entry)
	}
	t.Fatalf("SOT-ENG-023: conceptId %q がありません", conceptID)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func replaceOnce(t *testing.T, value, old, new string) string {
	t.Helper()
	replaced := strings.Replace(value, old, new, 1)
	if replaced == value {
		t.Fatalf("replace target not found: %q", old)
	}
	return replaced
}
