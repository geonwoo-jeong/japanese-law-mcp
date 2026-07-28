package legalqueryeval

import (
	"reflect"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestCompareMeaningは署名不一致時にassertionを適用しない(t *testing.T) {
	t.Parallel()

	fixtures := comparisonStepFixtures(t)
	expected := mustExpectedMeaning(
		t,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
		},
		[]string{"concept-a"},
		nil,
		fixtures[0],
	)
	actual := mustCandidate(
		t,
		"candidate-signature-mismatch",
		500,
		legalquery.ConfidenceMedium,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
		},
		[]legalquery.LegalConceptSource{
			mustComparisonConceptSource(
				t,
				"concept-a",
				"資料甲",
				"https://example.go.jp/concept/a",
				"2026-01-01",
			),
		},
		nil,
		fixtures[1],
	)

	comparison, err := CompareMeaning(expected, actual)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 署名不一致の比較に失敗しました: %v", err)
	}
	if comparison.SignatureMatched() || comparison.AllMatched() {
		t.Fatal("SOT-ENG-026: 異なる意味署名を一致と判定しました")
	}
	assertComparisonAssertions(t, comparison, false, false, false, false)
}

func TestCompareMeaningは未初期化入力を拒否する(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	validExpected := mustExpectedMeaning(
		t,
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)
	validCandidate := mustCandidate(
		t,
		"candidate-valid",
		500,
		legalquery.ConfidenceMedium,
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)

	tests := []struct {
		name      string
		expected  legalquerycorpus.ExpectedMeaning
		candidate legalquery.LegalQueryCandidate
	}{
		{
			name:      "expected",
			expected:  legalquerycorpus.ExpectedMeaning{},
			candidate: validCandidate,
		},
		{
			name:      "candidate",
			expected:  validExpected,
			candidate: legalquery.LegalQueryCandidate{},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := CompareMeaning(test.expected, test.candidate); err == nil {
				t.Fatal("SOT-ENG-026: 未初期化の比較入力を受理しました")
			}
		})
	}
}

func TestCompareMeaningは入力を変更せず並列に再現できる(t *testing.T) {
	t.Parallel()

	fixtures := comparisonStepFixtures(t)
	expected := mustExpectedMeaning(
		t,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
		},
		[]string{"concept-a", "concept-b"},
		nil,
		fixtures[0],
		fixtures[1],
	)
	actual := mustCandidate(
		t,
		"candidate-shared",
		750,
		legalquery.ConfidenceHigh,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
		},
		[]legalquery.LegalConceptSource{
			mustComparisonConceptSource(
				t,
				"concept-b",
				"資料乙",
				"https://example.go.jp/concept/b",
				"2026-01-02",
			),
			mustComparisonConceptSource(
				t,
				"concept-a",
				"資料甲",
				"https://example.go.jp/concept/a",
				"2026-01-01",
			),
		},
		nil,
		fixtures[0],
		fixtures[1],
	)
	expectedBefore := snapshotExpectedMeaning(expected)
	actualBefore := snapshotCandidate(actual)

	const workers = 64
	errors := make(chan string, workers)
	var group sync.WaitGroup
	group.Add(workers)
	for range workers {
		go func() {
			defer group.Done()
			comparison, err := CompareMeaning(expected, actual)
			if err != nil {
				errors <- err.Error()
				return
			}
			if !comparison.AllMatched() {
				errors <- "並列比較の結果が完全一致ではありません"
			}
		}()
	}
	group.Wait()
	close(errors)
	for message := range errors {
		t.Errorf("SOT-ENG-024: %s", message)
	}

	if got := snapshotExpectedMeaning(expected); !reflect.DeepEqual(got, expectedBefore) {
		t.Fatal("SOT-ENG-026: CompareMeaning が ExpectedMeaning を変更しました")
	}
	if got := snapshotCandidate(actual); !reflect.DeepEqual(got, actualBefore) {
		t.Fatal("SOT-ENG-026: CompareMeaning が LegalQueryCandidate を変更しました")
	}
}

type expectedMeaningSnapshot struct {
	meaningID     string
	evidenceCodes []legalquery.EvidenceCode
	conceptIDs    []string
	requiredPacks []string
	steps         []legalquerycorpus.ExpectedStep
}

func snapshotExpectedMeaning(
	value legalquerycorpus.ExpectedMeaning,
) expectedMeaningSnapshot {
	return expectedMeaningSnapshot{
		meaningID:     value.MeaningID(),
		evidenceCodes: value.EvidenceCodes(),
		conceptIDs:    value.ConceptIDs(),
		requiredPacks: value.RequiredPacks(),
		steps:         value.Steps(),
	}
}

type candidateSnapshot struct {
	candidateID    string
	semanticScore  int
	confidence     legalquery.Confidence
	evidenceCodes  []legalquery.EvidenceCode
	conceptSources []legalquery.LegalConceptSource
	requiredPacks  []string
	steps          []legalquery.LegalQueryCandidateStep
}

func snapshotCandidate(value legalquery.LegalQueryCandidate) candidateSnapshot {
	return candidateSnapshot{
		candidateID:    value.CandidateID(),
		semanticScore:  value.SemanticScore(),
		confidence:     value.Confidence(),
		evidenceCodes:  value.EvidenceCodes(),
		conceptSources: value.ConceptSources(),
		requiredPacks:  value.RequiredPacks(),
		steps:          value.Steps(),
	}
}
