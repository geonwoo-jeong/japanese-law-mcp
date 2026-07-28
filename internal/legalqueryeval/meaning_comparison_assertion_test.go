package legalqueryeval

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestCompareMeaningは根拠差だけを別assertionとして失敗させる(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expected := mustExpectedMeaning(
		t,
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)
	actual := mustCandidate(
		t,
		"candidate-evidence",
		500,
		legalquery.ConfidenceMedium,
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitResource},
		nil,
		nil,
		fixture,
	)

	comparison, err := CompareMeaning(expected, actual)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 根拠差の比較に失敗しました: %v", err)
	}
	if !comparison.SignatureMatched() {
		t.Fatal("SOT-ENG-026: 根拠差を意味署名の不一致へ混在させました")
	}
	assertComparisonAssertions(t, comparison, false, true, false, false)
	if comparison.AllMatched() {
		t.Fatal("SOT-ENG-026: 根拠が異なる候補を完全一致と判定しました")
	}
}

func TestCompareMeaningはconceptIdを整列済み集合として比較する(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expected := mustExpectedMeaning(
		t,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
		},
		[]string{"concept-a", "concept-b"},
		nil,
		fixture,
	)

	firstSources := []legalquery.LegalConceptSource{
		mustComparisonConceptSource(
			t,
			"concept-b",
			"資料乙",
			"https://example.go.jp/concept/b/first",
			"2026-01-02",
		),
		mustComparisonConceptSource(
			t,
			"concept-a",
			"資料甲",
			"https://example.go.jp/concept/a/first",
			"2026-01-01",
		),
	}
	secondSources := []legalquery.LegalConceptSource{
		mustComparisonConceptSource(
			t,
			"concept-a",
			"差し替え後の資料甲",
			"https://example.go.jp/concept/a/revised",
			"2026-07-01",
		),
		mustComparisonConceptSource(
			t,
			"concept-b",
			"差し替え後の資料乙",
			"https://example.go.jp/concept/b/revised",
			"2026-07-02",
		),
	}

	for index, sources := range [][]legalquery.LegalConceptSource{
		firstSources,
		secondSources,
	} {
		actual := mustCandidate(
			t,
			"candidate-concept-"+string(rune('a'+index)),
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceLegalConcept,
			},
			sources,
			nil,
			fixture,
		)
		comparison, err := CompareMeaning(expected, actual)
		if err != nil {
			t.Fatalf("SOT-ENG-026: conceptId 集合の比較に失敗しました: %v", err)
		}
		if !comparison.AllMatched() {
			t.Fatal("SOT-ENG-026: conceptId の順序または資料 metadata を一致条件へ混在させました")
		}
		assertComparisonAssertions(t, comparison, true, true, true, true)
	}
}

func TestCompareMeaningはconceptId差だけを別assertionとして失敗させる(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expected := mustExpectedMeaning(
		t,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceLegalConcept,
		},
		[]string{"concept-a", "concept-b"},
		nil,
		fixture,
	)
	actual := mustCandidate(
		t,
		"candidate-concept-difference",
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
			mustComparisonConceptSource(
				t,
				"concept-c",
				"資料丙",
				"https://example.go.jp/concept/c",
				"2026-01-03",
			),
		},
		nil,
		fixture,
	)

	comparison, err := CompareMeaning(expected, actual)
	if err != nil {
		t.Fatalf("SOT-ENG-026: conceptId 差の比較に失敗しました: %v", err)
	}
	if !comparison.SignatureMatched() {
		t.Fatal("SOT-ENG-026: conceptId 差を意味署名の不一致へ混在させました")
	}
	assertComparisonAssertions(t, comparison, true, true, false, true)
	if comparison.AllMatched() {
		t.Fatal("SOT-ENG-026: conceptId が異なる候補を完全一致と判定しました")
	}
}

func TestCompareMeaningは法概念根拠がない場合conceptAssertionを適用しない(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expected := mustExpectedMeaning(
		t,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceOfficialAlias,
		},
		nil,
		nil,
		fixture,
	)
	actual := mustCandidate(
		t,
		"candidate-no-concept-assertion",
		500,
		legalquery.ConfidenceMedium,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceOfficialAlias,
		},
		nil,
		nil,
		fixture,
	)

	comparison, err := CompareMeaning(expected, actual)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 非適用の concept assertion 比較に失敗しました: %v", err)
	}
	if !comparison.AllMatched() {
		t.Fatal("SOT-ENG-026: 非適用の concept assertion を完全一致失敗として扱いました")
	}
	assertComparisonAssertions(t, comparison, true, true, false, false)
}

func assertComparisonAssertions(
	t *testing.T,
	comparison interface {
		EvidenceAssertion() (bool, bool)
		ConceptAssertion() (bool, bool)
	},
	wantEvidenceMatched bool,
	wantEvidenceApplicable bool,
	wantConceptMatched bool,
	wantConceptApplicable bool,
) {
	t.Helper()

	evidenceMatched, evidenceApplicable := comparison.EvidenceAssertion()
	if evidenceMatched != wantEvidenceMatched ||
		evidenceApplicable != wantEvidenceApplicable {
		t.Fatalf(
			"SOT-ENG-026: EvidenceAssertion() = (%t, %t)、期待値 = (%t, %t)",
			evidenceMatched,
			evidenceApplicable,
			wantEvidenceMatched,
			wantEvidenceApplicable,
		)
	}
	conceptMatched, conceptApplicable := comparison.ConceptAssertion()
	if conceptMatched != wantConceptMatched ||
		conceptApplicable != wantConceptApplicable {
		t.Fatalf(
			"SOT-ENG-026: ConceptAssertion() = (%t, %t)、期待値 = (%t, %t)",
			conceptMatched,
			conceptApplicable,
			wantConceptMatched,
			wantConceptApplicable,
		)
	}
}
