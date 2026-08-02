package legalqueryeval

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

const repositoryCorpusV13DerivedObservationPopulationVerificationID = "legal-query-corpus-derived-observation-population"

func TestRepositoryCorpusV13は全派生観測の母集団を持つ(t *testing.T) {
	repositoryRoot := repositoryCorpusV13Root(t)
	corpus, err := legalquerycorpus.Load(
		t.Context(),
		repositoryRoot,
		"testdata/legalquery/corpus-v13",
	)
	if err != nil {
		t.Fatalf(
			"%s: corpus-v13 を読めません: %v",
			repositoryCorpusV13DerivedObservationPopulationVerificationID,
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
				"%s: case %q の派生観測を計算できません: %v",
				repositoryCorpusV13DerivedObservationPopulationVerificationID,
				semanticCase.CaseID(),
				err,
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
				repositoryCorpusV13DerivedObservationPopulationVerificationID,
				observationID,
			)
		}
	}
}

func repositoryCorpusV13Root(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf(
			"%s: test file path を取得できません",
			repositoryCorpusV13DerivedObservationPopulationVerificationID,
		)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
