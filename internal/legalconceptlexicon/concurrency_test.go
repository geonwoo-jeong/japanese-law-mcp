package legalconceptlexicon

import (
	"sync"
	"testing"
)

func TestLexiconSupportsConcurrentImmutableReads(t *testing.T) {
	t.Parallel()

	lexicon, err := Load([]byte(validFixture))
	if err != nil {
		t.Fatalf("SOT-ENG-023: Load() error = %v", err)
	}

	const (
		readerCount = 24
		iterations  = 50
	)
	var waitGroup sync.WaitGroup
	waitGroup.Add(readerCount)
	for reader := 0; reader < readerCount; reader++ {
		go func() {
			defer waitGroup.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				entries := lexicon.Entries()
				terms := lexicon.Terms()
				comparisonTerms := lexicon.ComparisonTerms()
				if len(entries) != 2 ||
					len(terms) != 1 ||
					len(comparisonTerms) != 1 {
					t.Errorf(
						"SOT-ENG-023: concurrent read = entries:%d terms:%d comparison:%d",
						len(entries),
						len(terms),
						len(comparisonTerms),
					)
					return
				}
				entries[0].Terms[0] = "変更後"
				entries[0].Candidates[0].RequiredPacks = []string{"changed"}
				terms[0] = "変更後"
				comparisonTerms[0] = "変更後"
			}
		}()
	}
	waitGroup.Wait()

	entries := lexicon.Entries()
	if entries[0].Terms[0] != "永住権" ||
		len(entries[0].Candidates[0].RequiredPacks) != 0 ||
		lexicon.Terms()[0] != "永住権" ||
		lexicon.ComparisonTerms()[0] != "永住権" {
		t.Fatal("SOT-ENG-023: concurrent reader が辞書を変更しました")
	}
}
