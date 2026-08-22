package legalqueryeval

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

const repositoryCorpusV16DerivedObservationPopulationVerificationID = "legal-query-corpus-derived-observation-population"

func TestRepositoryCorpusV16は四種の派生観測ごとに母集団を持つ(t *testing.T) {
	repositoryRoot := repositoryCorpusV16Root(t)
	corpus, err := legalquerycorpus.Load(
		t.Context(),
		repositoryRoot,
		"testdata/legalquery/corpus-v16",
	)
	if err != nil {
		t.Fatalf(
			"%s: corpus-v16 を読めません: %v",
			repositoryCorpusV16DerivedObservationPopulationVerificationID,
			err,
		)
	}

	observationIDs := derivedObservationIDs()
	if len(observationIDs) != 4 {
		t.Fatalf(
			"%s: 派生観測の定義件数 = %d",
			repositoryCorpusV16DerivedObservationPopulationVerificationID,
			len(observationIDs),
		)
	}
	counts := make(map[string]int, len(observationIDs))
	for _, observationID := range observationIDs {
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
				repositoryCorpusV16DerivedObservationPopulationVerificationID,
			)
		}
		for _, observationID := range applicable {
			counts[observationID]++
		}
	}

	for _, observationID := range observationIDs {
		t.Run(observationID, func(t *testing.T) {
			if counts[observationID] < 1 {
				t.Fatalf(
					"%s: 対象 case がありません",
					repositoryCorpusV16DerivedObservationPopulationVerificationID,
				)
			}
		})
	}
}

func repositoryCorpusV16Root(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf(
			"%s: test file path を取得できません",
			repositoryCorpusV16DerivedObservationPopulationVerificationID,
		)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
