package legalqueryeval

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

const repositoryCorpusV14DerivedObservationPopulationVerificationID = "legal-query-corpus-derived-observation-population"

func TestRepositoryCorpusV14は全派生観測の母集団を持つ(t *testing.T) {
	repositoryRoot := repositoryCorpusV14Root(t)
	corpus, err := legalquerycorpus.Load(
		t.Context(),
		repositoryRoot,
		"testdata/legalquery/corpus-v14",
	)
	if err != nil {
		t.Fatalf(
			"%s: corpus-v14 を読めません: %v",
			repositoryCorpusV14DerivedObservationPopulationVerificationID,
			err,
		)
	}

	counts := make(map[string]int, len(derivedObservationIDs()))
	for _, observationID := range derivedObservationIDs() {
		counts[observationID] = 0
	}
	for _, semanticCase := range corpus.Holdout() {
		expected, ok := semanticCase.Expected().(legalquerycorpus.ExpectedPlan)
		if !ok {
			continue
		}
		applicable, err := applicableDerivedObservations(expected)
		if err != nil {
			t.Fatalf(
				"%s: 派生観測を計算できません",
				repositoryCorpusV14DerivedObservationPopulationVerificationID,
			)
		}
		for _, observationID := range applicable {
			counts[observationID]++
		}
	}

	for _, observationID := range derivedObservationIDs() {
		if counts[observationID] < 1 {
			t.Fatalf(
				"%s: 派生観測 %s の対象 case がありません",
				repositoryCorpusV14DerivedObservationPopulationVerificationID,
				observationID,
			)
		}
	}
}

func repositoryCorpusV14Root(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf(
			"%s: test file path を取得できません",
			repositoryCorpusV14DerivedObservationPopulationVerificationID,
		)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
