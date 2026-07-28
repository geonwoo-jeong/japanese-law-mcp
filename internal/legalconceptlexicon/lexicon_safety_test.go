package legalconceptlexicon

import (
	"strings"
	"sync"
	"testing"
)

func TestLoadRejectsMissingOrOversizedDataset(t *testing.T) {
	t.Parallel()

	if _, err := Load(nil); err == nil {
		t.Fatal("SOT-ENG-023: 空の辞書を受理しました")
	}
	if _, err := Load(make([]byte, maxDatasetBytes+1)); err == nil {
		t.Fatal("SOT-ENG-023: 上限超過の辞書を受理しました")
	}
}

func TestLoadRejectsNormalizedCollisionWithoutConflictGroup(t *testing.T) {
	t.Parallel()

	value := `{
  "schemaVersion": 1,
  "lexiconVersion": "collision-test",
  "generatedAt": "2026-07-28T00:00:00Z",
  "entries": [
    {
      "conceptId": "cooling-off-katakana",
      "canonical": "クーリングオフ",
      "terms": ["クーリングオフ"],
      "comparisonTerms": ["くーりんぐおふ"],
      "sourceName": "消費者庁 クーリング・オフ Q&A",
      "sourceUrl": "https://www.no-trouble.caa.go.jp/qa/coolingoff.html",
      "confirmedAt": "2026-07-28",
      "mappingNote": "カタカナ表記です。",
      "selectionPolicy": "single_candidate",
      "candidates": [
        {
          "task": "search",
          "resource": "law_provision",
          "inputKind": "law_content_search",
          "officialTerm": "申込みの撤回",
          "requiredPacks": []
        }
      ]
    },
    {
      "conceptId": "cooling-off-hiragana",
      "canonical": "くーりんぐおふ",
      "terms": ["くーりんぐおふ"],
      "comparisonTerms": ["くーりんぐおふ"],
      "sourceName": "消費者庁 クーリング・オフ Q&A",
      "sourceUrl": "https://www.no-trouble.caa.go.jp/qa/coolingoff.html",
      "confirmedAt": "2026-07-28",
      "mappingNote": "ひらがな表記です。",
      "selectionPolicy": "single_candidate",
      "candidates": [
        {
          "task": "search",
          "resource": "law_provision",
          "inputKind": "law_content_search",
          "officialTerm": "申込みの撤回",
          "requiredPacks": []
        }
      ]
    }
  ]
}`
	if _, err := Load([]byte(value)); err == nil {
		t.Fatal("SOT-ENG-023: 正規化衝突する辞書を受理しました")
	}
}

func TestLoadEmbeddedConcurrentAccessDoesNotMutate(t *testing.T) {
	t.Parallel()

	lexicon, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-023: LoadEmbedded() error = %v", err)
	}

	var waitGroup sync.WaitGroup
	for iteration := 0; iteration < 32; iteration++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			entries := lexicon.Entries()
			terms := lexicon.Terms()
			comparisonTerms := lexicon.ComparisonTerms()
			if len(entries) == 0 || len(terms) == 0 || len(comparisonTerms) == 0 {
				t.Error("SOT-ENG-023: 並行読取で空配列が返りました")
				return
			}
			entries[0].Canonical = "変更後"
			terms[0] = "変更後"
			comparisonTerms[0] = "変更後"
		}()
	}
	waitGroup.Wait()

	if strings.Contains(strings.Join(lexicon.Terms(), ","), "変更後") {
		t.Fatal("SOT-ENG-023: Terms が並行読取で変更されました")
	}
	if strings.Contains(strings.Join(lexicon.ComparisonTerms(), ","), "変更後") {
		t.Fatal("SOT-ENG-023: ComparisonTerms が並行読取で変更されました")
	}
}
