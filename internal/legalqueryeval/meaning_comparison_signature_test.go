package legalqueryeval

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestCompareMeaningは七つのlogicalInputを完全一致させる(t *testing.T) {
	t.Parallel()

	for _, fixture := range comparisonStepFixtures(t) {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()

			expected := mustExpectedMeaning(
				t,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				fixture.requiredPacks,
				fixture,
			)
			actual := mustCandidate(
				t,
				"candidate-exact",
				850,
				legalquery.ConfidenceHigh,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				fixture.requiredPacks,
				fixture,
			)

			comparison, err := CompareMeaning(expected, actual)
			if err != nil {
				t.Fatalf("SOT-ENG-026: 完全一致の比較に失敗しました: %v", err)
			}
			if !comparison.SignatureMatched() || !comparison.AllMatched() {
				t.Fatal("SOT-ENG-026: 完全一致した意味署名を不一致と判定しました")
			}
			assertComparisonAssertions(t, comparison, true, true, false, false)
		})
	}
}

func TestCompareMeaningは署名の主要要素の差を検出する(t *testing.T) {
	t.Parallel()

	fixtures := comparisonStepFixtures(t)
	tests := []struct {
		name     string
		expected comparisonStepFixture
		actual   comparisonStepFixture
	}{
		{
			name:     "task",
			expected: fixtures[0],
			actual:   fixtures[2],
		},
		{
			name:     "resource",
			expected: fixtures[0],
			actual:   fixtures[1],
		},
		{
			name:     "inputKind",
			expected: fixtures[5],
			actual:   fixtures[6],
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			expected := mustExpectedMeaning(
				t,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				test.expected,
			)
			actual := mustCandidate(
				t,
				"candidate-major-field",
				500,
				legalquery.ConfidenceMedium,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				nil,
				test.actual,
			)

			assertSignatureMismatch(t, expected, actual)
		})
	}
}

func TestCompareMeaningは各logicalInputの値の差を検出する(t *testing.T) {
	t.Parallel()

	for _, fixture := range comparisonStepFixtures(t) {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			changed := changedLogicalInputFixture(t, fixture)
			expected := mustExpectedMeaning(
				t,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				fixture.requiredPacks,
				fixture,
			)
			actual := mustCandidate(
				t,
				"candidate-input-difference",
				500,
				legalquery.ConfidenceMedium,
				[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
				nil,
				fixture.requiredPacks,
				changed,
			)

			assertSignatureMismatch(t, expected, actual)
		})
	}
}

func TestCompareMeaningはlawReadのID形とref形の差分を検出する(t *testing.T) {
	t.Parallel()

	asOf := mustComparisonDate(t, "2025-04-01")
	expectedFixture := comparisonStepFixture{
		name:      "法令読取り-ID形",
		task:      legalquery.TaskRead,
		resource:  legalquery.ResourceLaw,
		inputKind: legalquery.InputKindLawRead,
		logicalInput: mustLawReadInputByID(
			t,
			"503AC0000000037",
			"",
			&asOf,
		),
		capabilityID:           "law.document.read",
		capabilityMajorVersion: 1,
	}

	t.Run("ID形のasOf差", func(t *testing.T) {
		t.Parallel()
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			expectedFixture,
		)
		actualFixture := expectedFixture
		actualFixture.logicalInput = mustLawReadInputByID(
			t,
			"503AC0000000037",
			"",
			nil,
		)
		actual := mustCandidate(
			t,
			"candidate-law-read-asof",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			actualFixture,
		)

		assertSignatureMismatch(t, expected, actual)
	})

	t.Run("ID形のrevisionId差", func(t *testing.T) {
		t.Parallel()
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			comparisonStepFixture{
				name:      "法令読取り-revision",
				task:      legalquery.TaskRead,
				resource:  legalquery.ResourceLaw,
				inputKind: legalquery.InputKindLawRead,
				logicalInput: mustLawReadInputByID(
					t,
					"503AC0000000037",
					"505AC0000000011_20240401_506AC0000000012",
					nil,
				),
				capabilityID:           "law.document.read",
				capabilityMajorVersion: 1,
			},
		)
		actual := mustCandidate(
			t,
			"candidate-law-read-revision",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			comparisonStepFixture{
				name:      "法令読取り-revision-different",
				task:      legalquery.TaskRead,
				resource:  legalquery.ResourceLaw,
				inputKind: legalquery.InputKindLawRead,
				logicalInput: mustLawReadInputByID(
					t,
					"503AC0000000037",
					"505AC0000000011_20240401_507AC0000000013",
					nil,
				),
				capabilityID:           "law.document.read",
				capabilityMajorVersion: 1,
			},
		)

		assertSignatureMismatch(t, expected, actual)
	})

	t.Run("ref形のversionId差", func(t *testing.T) {
		t.Parallel()
		expectedFixture := comparisonStepFixture{
			name:      "法令読取り-ref-version",
			task:      legalquery.TaskRead,
			resource:  legalquery.ResourceLaw,
			inputKind: legalquery.InputKindLawRead,
			logicalInput: mustLawReadInput(
				t,
				mustComparisonRefWithVersion(
					t,
					"e-gov-law-api-v2",
					"e-gov-law-api-v2",
					"law",
					"503AC0000000037",
					"505AC0000000011_20240401_506AC0000000012",
				),
			),
			capabilityID:           "law.document.read",
			capabilityMajorVersion: 1,
		}
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			expectedFixture,
		)
		actualFixture := expectedFixture
		actualFixture.logicalInput = mustLawReadInput(
			t,
			mustComparisonRefWithVersion(
				t,
				"e-gov-law-api-v2",
				"e-gov-law-api-v2",
				"law",
				"503AC0000000037",
				"505AC0000000011_20240401_507AC0000000013",
			),
		)
		actual := mustCandidate(
			t,
			"candidate-law-read-ref-version",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			actualFixture,
		)

		assertSignatureMismatch(t, expected, actual)
	})
}

func TestCompareMeaningはlawArticleReadの参照形と位置差を検出する(t *testing.T) {
	t.Parallel()

	ref := mustComparisonRef(
		t,
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"law",
		"503AC0000000037",
	)
	asOf := mustComparisonDate(t, "2025-04-01")

	t.Run("ref形のasOf差", func(t *testing.T) {
		t.Parallel()
		expectedFixture := comparisonStepFixture{
			name:      "条文読取り-ref",
			task:      legalquery.TaskRead,
			resource:  legalquery.ResourceLawProvision,
			inputKind: legalquery.InputKindLawArticleRead,
			logicalInput: mustLawArticleReadInputByRef(
				t,
				ref,
				mustComparisonLocationWithoutParagraph(
					t,
					model.LawArticleProvisionMain,
					"22",
				),
				&asOf,
			),
			capabilityID:           "law.article.read",
			capabilityMajorVersion: 1,
		}
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			expectedFixture,
		)
		actualFixture := expectedFixture
		actualFixture.logicalInput = mustLawArticleReadInputByRef(
			t,
			ref,
			mustComparisonLocationWithoutParagraph(
				t,
				model.LawArticleProvisionMain,
				"22",
			),
			nil,
		)
		actual := mustCandidate(
			t,
			"candidate-law-article-ref-asof",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			actualFixture,
		)

		assertSignatureMismatch(t, expected, actual)
	})

	t.Run("paragraph有無の差", func(t *testing.T) {
		t.Parallel()
		expectedFixture := comparisonStepFixture{
			name:      "条文読取り-no-paragraph",
			task:      legalquery.TaskRead,
			resource:  legalquery.ResourceLawProvision,
			inputKind: legalquery.InputKindLawArticleRead,
			logicalInput: mustLawArticleReadInputByRef(
				t,
				ref,
				mustComparisonLocationWithoutParagraph(
					t,
					model.LawArticleProvisionMain,
					"22",
				),
				nil,
			),
			capabilityID:           "law.article.read",
			capabilityMajorVersion: 1,
		}
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			expectedFixture,
		)
		actualFixture := expectedFixture
		actualFixture.logicalInput = mustLawArticleReadInputByRef(
			t,
			ref,
			mustComparisonLocation(t, "22", 1),
			nil,
		)
		actual := mustCandidate(
			t,
			"candidate-law-article-paragraph",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			actualFixture,
		)

		assertSignatureMismatch(t, expected, actual)
	})

	t.Run("provision差", func(t *testing.T) {
		t.Parallel()
		expectedFixture := comparisonStepFixture{
			name:      "条文読取り-main",
			task:      legalquery.TaskRead,
			resource:  legalquery.ResourceLawProvision,
			inputKind: legalquery.InputKindLawArticleRead,
			logicalInput: mustLawArticleReadInputByRef(
				t,
				ref,
				mustComparisonLocationWithoutParagraph(
					t,
					model.LawArticleProvisionMain,
					"22",
				),
				nil,
			),
			capabilityID:           "law.article.read",
			capabilityMajorVersion: 1,
		}
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			expectedFixture,
		)
		actualFixture := expectedFixture
		actualFixture.logicalInput = mustLawArticleReadInputByRef(
			t,
			ref,
			mustComparisonLocationWithoutParagraph(
				t,
				model.LawArticleProvisionSupplementary,
				"22",
			),
			nil,
		)
		actual := mustCandidate(
			t,
			"candidate-law-article-provision",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			actualFixture,
		)

		assertSignatureMismatch(t, expected, actual)
	})
}

func TestCompareMeaningはstep順序とrequiredPacksを署名へ含める(t *testing.T) {
	t.Parallel()

	fixtures := comparisonStepFixtures(t)
	first := fixtures[0]
	second := fixtures[1]

	t.Run("step順序", func(t *testing.T) {
		t.Parallel()
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			first,
			second,
		)
		actual := mustCandidate(
			t,
			"candidate-step-order",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			second,
			first,
		)

		assertSignatureMismatch(t, expected, actual)
	})

	t.Run("requiredPacks", func(t *testing.T) {
		t.Parallel()
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			[]string{"judicial-cases"},
			first,
		)
		actual := mustCandidate(
			t,
			"candidate-pack",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			first,
		)

		assertSignatureMismatch(t, expected, actual)
	})
}

func TestCompareMeaningは任意項目の有無と有効な別表現を署名へ含める(t *testing.T) {
	t.Parallel()

	t.Run("lawReadのid形", func(t *testing.T) {
		t.Parallel()

		asOf := mustComparisonDate(t, "2025-04-01")
		expectedStep := comparisonStepFixture{
			name:      "法令読取り-ID",
			task:      legalquery.TaskRead,
			resource:  legalquery.ResourceLaw,
			inputKind: legalquery.InputKindLawRead,
			logicalInput: mustLawReadInputByID(
				t,
				"503AC0000000037",
				"",
				&asOf,
			),
			capabilityID:           "law.document.read",
			capabilityMajorVersion: 1,
		}
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			expectedStep,
		)
		actual := mustCandidate(
			t,
			"candidate-law-read-id",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			expectedStep,
		)

		comparison, err := CompareMeaning(expected, actual)
		if err != nil {
			t.Fatalf("SOT-ENG-026: law read id形の比較に失敗しました: %v", err)
		}
		if !comparison.AllMatched() {
			t.Fatal("SOT-ENG-026: law read id形の一致を誤判定しました")
		}
	})

	t.Run("lawReadのref形とid形は区別する", func(t *testing.T) {
		t.Parallel()

		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			comparisonStepFixture{
				name:      "法令読取り-ID",
				task:      legalquery.TaskRead,
				resource:  legalquery.ResourceLaw,
				inputKind: legalquery.InputKindLawRead,
				logicalInput: mustLawReadInputByID(
					t,
					"503AC0000000037",
					"",
					nil,
				),
				capabilityID:           "law.document.read",
				capabilityMajorVersion: 1,
			},
		)
		actual := mustCandidate(
			t,
			"candidate-law-read-ref",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			comparisonStepFixture{
				name:      "法令読取り-ref",
				task:      legalquery.TaskRead,
				resource:  legalquery.ResourceLaw,
				inputKind: legalquery.InputKindLawRead,
				logicalInput: mustLawReadInput(
					t,
					mustComparisonRef(
						t,
						"e-gov-law-api-v2",
						"e-gov-law-api-v2",
						"law",
						"503AC0000000037",
					),
				),
				capabilityID:           "law.document.read",
				capabilityMajorVersion: 1,
			},
		)

		assertSignatureMismatch(t, expected, actual)
	})

	t.Run("lawArticleReadのref形と項なし位置", func(t *testing.T) {
		t.Parallel()

		location := mustComparisonLocationWithoutParagraph(
			t,
			model.LawArticleProvisionSupplementary,
			"9_2",
		)
		ref := mustComparisonRef(
			t,
			"e-gov-law-api-v2",
			"e-gov-law-api-v2",
			"law",
			"503AC0000000037",
		)
		step := comparisonStepFixture{
			name:      "条文読取り-ref",
			task:      legalquery.TaskRead,
			resource:  legalquery.ResourceLawProvision,
			inputKind: legalquery.InputKindLawArticleRead,
			logicalInput: mustLawArticleReadInputByRef(
				t,
				ref,
				location,
				nil,
			),
			capabilityID:           "law.article.read",
			capabilityMajorVersion: 1,
		}
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			step,
		)
		actual := mustCandidate(
			t,
			"candidate-law-article-ref",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			step,
		)

		comparison, err := CompareMeaning(expected, actual)
		if err != nil {
			t.Fatalf("SOT-ENG-026: law article ref形の比較に失敗しました: %v", err)
		}
		if !comparison.AllMatched() {
			t.Fatal("SOT-ENG-026: law article ref形の一致を誤判定しました")
		}
	})

	t.Run("lawArticleReadの項の有無を区別する", func(t *testing.T) {
		t.Parallel()

		ref := mustComparisonRef(
			t,
			"e-gov-law-api-v2",
			"e-gov-law-api-v2",
			"law",
			"503AC0000000037",
		)
		expected := mustExpectedMeaning(
			t,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			comparisonStepFixture{
				name:      "条文読取り-項なし",
				task:      legalquery.TaskRead,
				resource:  legalquery.ResourceLawProvision,
				inputKind: legalquery.InputKindLawArticleRead,
				logicalInput: mustLawArticleReadInputByRef(
					t,
					ref,
					mustComparisonLocationWithoutParagraph(
						t,
						model.LawArticleProvisionMain,
						"22",
					),
					nil,
				),
				capabilityID:           "law.article.read",
				capabilityMajorVersion: 1,
			},
		)
		actual := mustCandidate(
			t,
			"candidate-law-article-paragraph",
			500,
			legalquery.ConfidenceMedium,
			[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			nil,
			nil,
			comparisonStepFixture{
				name:      "条文読取り-項あり",
				task:      legalquery.TaskRead,
				resource:  legalquery.ResourceLawProvision,
				inputKind: legalquery.InputKindLawArticleRead,
				logicalInput: mustLawArticleReadInputByRef(
					t,
					ref,
					mustComparisonLocation(t, "22", 1),
					nil,
				),
				capabilityID:           "law.article.read",
				capabilityMajorVersion: 1,
			},
		)

		assertSignatureMismatch(t, expected, actual)
	})
}

func TestCompareMeaningは内部候補metadataを署名から除外する(t *testing.T) {
	t.Parallel()

	fixture := comparisonStepFixtures(t)[0]
	expected := mustExpectedMeaning(
		t,
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)
	first := mustCandidate(
		t,
		"candidate-low",
		1,
		legalquery.ConfidenceLow,
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)
	second := mustCandidate(
		t,
		"candidate-high",
		999,
		legalquery.ConfidenceHigh,
		[]legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
		nil,
		nil,
		fixture,
	)

	for _, actual := range []legalquery.LegalQueryCandidate{first, second} {
		comparison, err := CompareMeaning(expected, actual)
		if err != nil {
			t.Fatalf("SOT-ENG-026: metadata を変えた候補の比較に失敗しました: %v", err)
		}
		if !comparison.SignatureMatched() || !comparison.AllMatched() {
			t.Fatal("SOT-ENG-026: candidateId、stepId、score または confidence を意味署名へ混在させました")
		}
		assertComparisonAssertions(t, comparison, true, true, false, false)
	}
}

func assertSignatureMismatch(
	t *testing.T,
	expected legalquerycorpus.ExpectedMeaning,
	actual legalquery.LegalQueryCandidate,
) {
	t.Helper()

	comparison, err := CompareMeaning(expected, actual)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 意味署名差の比較に失敗しました: %v", err)
	}
	if comparison.SignatureMatched() || comparison.AllMatched() {
		t.Fatal("SOT-ENG-026: 異なる意味署名を一致と判定しました")
	}
	assertComparisonAssertions(t, comparison, false, false, false, false)
}
