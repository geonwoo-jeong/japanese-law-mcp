package legalconceptlexicon

import (
	"slices"
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

func TestLoadAcceptsEquivalentSurfaceFormsWithOneComparisonTerm(t *testing.T) {
	t.Parallel()

	value := replaceOnce(
		t,
		validFixture,
		`"terms": ["永住権"],`,
		`"terms": ["クーリングオフ", "クーリング・オフ"],`,
	)
	value = replaceOnce(
		t,
		value,
		`"comparisonTerms": ["永住権"],`,
		`"comparisonTerms": ["くーりんぐおふ"],`,
	)

	lexicon, err := Load([]byte(value))
	if err != nil {
		t.Fatalf("SOT-ENG-023: 同じ比較語へ正規化される表記を受理できません: %v", err)
	}
	entry := lexicon.Entries()[0]
	if len(entry.Terms) != 2 ||
		len(entry.ComparisonTerms) != 1 ||
		entry.ComparisonTerms[0] != "くーりんぐおふ" {
		t.Fatalf("SOT-ENG-023: entry = %#v", entry)
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
	if lexicon.Version() != "legal-concept-2026-07-28-2" {
		t.Fatalf("SOT-ENG-023: embedded version = %q", lexicon.Version())
	}
	entries := lexicon.Entries()
	if len(entries) != 11 {
		t.Fatalf("SOT-ENG-023: entry count = %d, want 11", len(entries))
	}

	expected := []expectedEntry{
		{
			conceptID: "adult-guardianship",
			term:      "成年後見",
			policy:    SelectionPolicyAmbiguousNoAutoExecute,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "成年後見",
				},
				{
					resource:      "judicial_decision",
					inputKind:     "judicial_decision_search",
					officialTerm:  "成年後見",
					requiredPacks: []string{"judicial-cases"},
				},
			},
		},
		{
			conceptID: "annual-paid-leave",
			term:      "有休",
			policy:    SelectionPolicySingleCandidate,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "年次有給休暇",
				},
			},
		},
		{
			conceptID: "child-support",
			term:      "養育費",
			policy:    SelectionPolicyAmbiguousNoAutoExecute,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "養育費",
				},
				{
					resource:      "judicial_decision",
					inputKind:     "judicial_decision_search",
					officialTerm:  "養育費",
					requiredPacks: []string{"judicial-cases"},
				},
			},
		},
		{
			conceptID:       "childcare-leave",
			term:            "育休",
			conflictGroupID: "childcare-leave-group",
			policy:          SelectionPolicySingleCandidate,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "育児休業",
				},
			},
		},
		{
			conceptID:       "childcare-leave-benefit",
			term:            "育休",
			conflictGroupID: "childcare-leave-group",
			policy:          SelectionPolicySingleCandidate,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "育児休業給付",
				},
			},
		},
		{
			conceptID: "cooling-off",
			term:      "クーリング・オフ",
			policy:    SelectionPolicySingleCandidate,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "申込みの撤回",
				},
			},
		},
		{
			conceptID: "online-defamation",
			term:      "ネット中傷",
			policy:    SelectionPolicyAmbiguousNoAutoExecute,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "名誉毀損",
				},
				{
					resource:      "judicial_decision",
					inputKind:     "judicial_decision_search",
					officialTerm:  "名誉毀損",
					requiredPacks: []string{"judicial-cases"},
				},
			},
		},
		{
			conceptID: "overtime-premium-pay",
			term:      "残業代",
			policy:    SelectionPolicySingleCandidate,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "割増賃金",
				},
			},
		},
		{
			conceptID:       "permanent-residence",
			term:            "永住権",
			conflictGroupID: "permanent-residence-group",
			policy:          SelectionPolicyAmbiguousNoAutoExecute,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "永住許可",
				},
				{
					resource:      "judicial_decision",
					inputKind:     "judicial_decision_search",
					officialTerm:  "永住許可",
					requiredPacks: []string{"judicial-cases"},
				},
			},
		},
		{
			conceptID:       "permanent-residence-permission",
			term:            "永住権",
			conflictGroupID: "permanent-residence-group",
			policy:          SelectionPolicySingleCandidate,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "永住許可",
				},
			},
		},
		{
			conceptID: "unemployment-basic-allowance",
			term:      "失業手当",
			policy:    SelectionPolicySingleCandidate,
			candidates: []expectedCandidate{
				{
					resource:     "law_provision",
					inputKind:    "law_content_search",
					officialTerm: "基本手当",
				},
			},
		},
	}
	for _, value := range expected {
		assertEntry(t, entries, value)
	}
}

type expectedEntry struct {
	conceptID       string
	term            string
	conflictGroupID string
	policy          SelectionPolicy
	candidates      []expectedCandidate
}

type expectedCandidate struct {
	resource      string
	inputKind     string
	officialTerm  string
	requiredPacks []string
}

func assertEntry(t *testing.T, entries []Entry, want expectedEntry) {
	t.Helper()
	for _, entry := range entries {
		if entry.ConceptID != want.conceptID {
			continue
		}
		if !contains(entry.Terms, want.term) {
			t.Fatalf("SOT-ENG-023: terms = %#v", entry.Terms)
		}
		if entry.SelectionPolicy != want.policy {
			t.Fatalf("SOT-ENG-023: selectionPolicy = %q", entry.SelectionPolicy)
		}
		if entry.ConflictGroupID != want.conflictGroupID {
			t.Fatalf(
				"SOT-ENG-023: conflictGroupId = %q, want %q",
				entry.ConflictGroupID,
				want.conflictGroupID,
			)
		}
		if len(entry.Candidates) != len(want.candidates) {
			t.Fatalf("SOT-ENG-023: candidates = %#v", entry.Candidates)
		}
		for index, candidate := range entry.Candidates {
			expectedCandidate := want.candidates[index]
			if string(candidate.Task) != "search" ||
				string(candidate.Resource) != expectedCandidate.resource ||
				string(candidate.InputKind) != expectedCandidate.inputKind ||
				candidate.OfficialTerm != expectedCandidate.officialTerm ||
				!slices.Equal(candidate.RequiredPacks, expectedCandidate.requiredPacks) {
				t.Fatalf("SOT-ENG-023: candidate = %#v", candidate)
			}
		}
		return
	}
	t.Fatalf("SOT-ENG-023: conceptId %q がありません", want.conceptID)
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
