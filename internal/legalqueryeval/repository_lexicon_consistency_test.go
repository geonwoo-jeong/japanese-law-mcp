package legalqueryeval_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func Test組込み辞書とCorpusV4の法概念IDは相互に対応する(t *testing.T) {
	repositoryRoot := legalConceptRepositoryRoot(t)
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot,
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("SOT-ENG-023, SOT-ENG-026: corpus-v4 Load() error = %v", err)
	}
	lexicon, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ENG-023: LoadEmbedded() error = %v", err)
	}
	knownConceptIDs := make(map[string]struct{}, len(lexicon.Entries()))
	for _, entry := range lexicon.Entries() {
		knownConceptIDs[entry.ConceptID] = struct{}{}
	}

	fixtures := append(corpus.Development(), corpus.Holdout()...)
	referencedConceptIDs := make(map[string]struct{}, len(knownConceptIDs))
	for _, fixture := range fixtures {
		plan, ok := fixture.Expected().(legalquerycorpus.ExpectedPlan)
		if !ok {
			continue
		}
		for _, meaning := range plan.Meanings() {
			for _, conceptID := range meaning.ConceptIDs() {
				referencedConceptIDs[conceptID] = struct{}{}
				if _, exists := knownConceptIDs[conceptID]; exists {
					continue
				}
				t.Errorf(
					"SOT-ENG-023: %s が未知の conceptId %q を参照しています",
					fixture.CaseID(),
					conceptID,
				)
			}
		}
	}
	for conceptID := range knownConceptIDs {
		if _, exists := referencedConceptIDs[conceptID]; exists {
			continue
		}
		t.Errorf(
			"SOT-ENG-023: 組込み辞書の conceptId %q に corpus-v4 の fixture がありません",
			conceptID,
		)
	}
}

func legalConceptRepositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("SOT-ENG-023: test file path を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
