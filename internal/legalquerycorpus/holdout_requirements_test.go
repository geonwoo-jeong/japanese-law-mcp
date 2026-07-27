package legalquerycorpus

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type holdoutRequirementsTestCategory struct {
	id        string
	coverages []string
}

func TestHoldoutRequirementsは全最小条件を満たす240件を受理する(
	t *testing.T,
) {
	holdout := holdoutRequirementsTestValidHoldout(t)
	before, err := cloneCorpusSemanticCases(holdout)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 入力比較用 holdout を複製できない: %v", err)
	}

	if err := validateHoldoutRequirements(holdout); err != nil {
		t.Fatalf("SOT-ENG-024/026: 正常な holdout を拒否した: %v", err)
	}
	if !reflect.DeepEqual(holdout, before) {
		t.Fatal("SOT-ENG-026: holdout 検証が入力または manifest 順を変更した")
	}
}

func TestManifestIntegrityはHoldoutRequirementsを必ず実行する(
	t *testing.T,
) {
	fixtures := manifestIntegrityTestBaseFixtures(t)
	layout, schema, manifest := manifestIntegrityTestPrepare(
		t,
		fixtures,
		"corpus-v1",
		"",
	)
	filesystem := filesystemReadTestOpen(t, layout)

	got, err := validateManifestIntegrity(
		context.Background(),
		filesystem,
		schema,
		manifest,
	)
	if err == nil {
		t.Fatal("SOT-ENG-024/026: loader orchestration が少数 holdout を受理した")
	}
	if !reflect.DeepEqual(got, integrityCheckedCorpus{}) {
		t.Fatalf("SOT-ENG-026: 失敗時に部分的な semantic integrity を返した: %#v", got)
	}
}

func TestIntegrityHoldoutRequirementsは検証済みHoldoutだけを返す(
	t *testing.T,
) {
	holdout := holdoutRequirementsTestValidHoldout(t)
	got, err := validateIntegrityHoldoutRequirements(integrityCheckedCorpus{
		holdout: holdout,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-024/026: 正常な semantic integrity error = %v", err)
	}
	if len(got.holdout) != len(holdout) {
		t.Fatalf(
			"SOT-ENG-026: holdout 件数 = %d, want %d",
			len(got.holdout),
			len(holdout),
		)
	}
}

func TestIntegrityHoldoutRequirementsは失敗時に部分結果を返さない(
	t *testing.T,
) {
	tests := map[string]func(*testing.T, []SemanticCase) []SemanticCase{
		"239件": func(_ *testing.T, holdout []SemanticCase) []SemanticCase {
			return append([]SemanticCase{}, holdout[:239]...)
		},
		"category不足": func(t *testing.T, holdout []SemanticCase) []SemanticCase {
			values := append([]SemanticCase{}, holdout...)
			values[19] = holdoutRequirementsTestCase(
				t,
				19,
				[]string{"budget-capability-call-limit"},
				"",
			)
			return values
		},
		"safety pair不足": func(t *testing.T, holdout []SemanticCase) []SemanticCase {
			values := append([]SemanticCase{}, holdout...)
			for index, semanticCase := range values {
				if holdoutRequirementsTestContains(
					semanticCase.CoverageIDs(),
					"boundary-budget-limit",
				) {
					values[index] = holdoutRequirementsTestCase(
						t,
						index,
						semanticCase.CoverageIDs(),
						SafetyVariantOrdinary,
					)
				}
			}
			return values
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			checked := integrityCheckedCorpus{
				holdout: mutate(t, holdoutRequirementsTestValidHoldout(t)),
			}
			got, err := validateIntegrityHoldoutRequirements(checked)
			if err == nil {
				t.Fatal("SOT-ENG-024/026: 不正な semantic integrity を受理した")
			}
			if !reflect.DeepEqual(got, integrityCheckedCorpus{}) {
				t.Fatalf("SOT-ENG-026: 失敗時に部分結果を返した: %#v", got)
			}
		})
	}
}

func TestHoldoutRequirementsは239件を拒否する(t *testing.T) {
	holdout := holdoutRequirementsTestValidHoldout(t)
	if err := validateHoldoutRequirements(holdout[:239]); err == nil {
		t.Fatal("SOT-ENG-024: 239件の holdout を受理した")
	}
}

func TestHoldoutRequirementsは各CategoryをCase単位で20件要求する(
	t *testing.T,
) {
	catalog := holdoutRequirementsTestCatalog()
	for categoryIndex, category := range catalog {
		categoryIndex, category := categoryIndex, category
		t.Run(category.id, func(t *testing.T) {
			holdout := holdoutRequirementsTestValidHoldout(t)
			firstIndex := categoryIndex * 20
			lastIndex := firstIndex + 19

			// 同じ category の coverage を二件持っても、一 case と数える。
			holdout[firstIndex] = holdoutRequirementsTestCase(
				t,
				firstIndex,
				category.coverages[:2],
				"",
			)
			donor := catalog[(categoryIndex+1)%len(catalog)]
			holdout[lastIndex] = holdoutRequirementsTestCase(
				t,
				lastIndex,
				[]string{holdoutRequirementsTestNonTargetCoverage(
					donor.coverages,
					"",
				)},
				"",
			)

			err := validateHoldoutRequirements(holdout)
			if err == nil {
				t.Fatalf(
					"SOT-ENG-024: category %s が19件でも受理した",
					category.id,
				)
			}
			if !strings.Contains(err.Error(), category.id) {
				t.Fatalf(
					"SOT-ENG-024: category 不足 error に ID がない: %v",
					err,
				)
			}
		})
	}
}

func TestHoldoutRequirementsは全Coverageの最小件数を要求する(
	t *testing.T,
) {
	for _, category := range holdoutRequirementsTestCatalog() {
		for _, coverageID := range category.coverages {
			coverageID := coverageID
			t.Run(coverageID, func(t *testing.T) {
				holdout := holdoutRequirementsTestValidHoldout(t)
				minimum := 1
				if isSemanticSafetyCoverageID(coverageID) {
					minimum = 2
				}
				holdout = holdoutRequirementsTestKeepCoverageCount(
					t,
					holdout,
					category.coverages,
					coverageID,
					minimum-1,
				)

				err := validateHoldoutRequirements(holdout)
				if err == nil {
					t.Fatalf(
						"SOT-ENG-026: coverage %s の最小件数不足を受理した",
						coverageID,
					)
				}
				if !strings.Contains(err.Error(), coverageID) {
					t.Fatalf(
						"SOT-ENG-026: coverage 不足 error に ID がない: %v",
						err,
					)
				}
			})
		}
	}
}

func TestHoldoutRequirementsは全Coverageの下限値を受理する(
	t *testing.T,
) {
	for _, category := range holdoutRequirementsTestCatalog() {
		for _, coverageID := range category.coverages {
			coverageID := coverageID
			t.Run(coverageID, func(t *testing.T) {
				minimum := 1
				if isSemanticSafetyCoverageID(coverageID) {
					minimum = 2
				}
				holdout := holdoutRequirementsTestKeepCoverageCount(
					t,
					holdoutRequirementsTestValidHoldout(t),
					category.coverages,
					coverageID,
					minimum,
				)
				if err := validateHoldoutRequirements(holdout); err != nil {
					t.Fatalf(
						"SOT-ENG-026: coverage %s の下限値を拒否した: %v",
						coverageID,
						err,
					)
				}
			})
		}
	}
}

func TestHoldoutRequirementsはSafetyCoverageの両Variantを要求する(
	t *testing.T,
) {
	for _, coverageID := range holdoutRequirementsTestSafetyCoverageIDs() {
		coverageID := coverageID
		for _, onlyVariant := range []SafetyVariant{
			SafetyVariantOrdinary,
			SafetyVariantAdversarial,
		} {
			onlyVariant := onlyVariant
			t.Run(
				coverageID+"/"+string(onlyVariant)+"のみ",
				func(t *testing.T) {
					holdout := holdoutRequirementsTestValidHoldout(t)
					for index, semanticCase := range holdout {
						if !holdoutRequirementsTestContains(
							semanticCase.CoverageIDs(),
							coverageID,
						) {
							continue
						}
						holdout[index] = holdoutRequirementsTestCase(
							t,
							index,
							semanticCase.CoverageIDs(),
							onlyVariant,
						)
					}

					err := validateHoldoutRequirements(holdout)
					if err == nil {
						t.Fatalf(
							"SOT-ENG-026: safety coverage %s の片側 variant だけを受理した",
							coverageID,
						)
					}
					if !strings.Contains(err.Error(), coverageID) {
						t.Fatalf(
							"SOT-ENG-026: safety pair error に ID がない: %v",
							err,
						)
					}
				},
			)
		}
	}
}

func TestSafetyCoverageCatalogは五件だけを対象にする(t *testing.T) {
	safetyIDs := holdoutRequirementsTestSafetyCoverageIDs()
	for _, coverageID := range semanticCoverageIDs() {
		want := holdoutRequirementsTestContains(safetyIDs, coverageID)
		if got := isSemanticSafetyCoverageID(coverageID); got != want {
			t.Fatalf(
				"SOT-ENG-026: safety coverage %q = %t, want %t",
				coverageID,
				got,
				want,
			)
		}
	}
}

func TestCategoryIDsはCoverageを正確に一意Categoryへ写像する(
	t *testing.T,
) {
	catalog := holdoutRequirementsTestCatalog()
	coverageCount := 0
	seen := make(map[string]struct{}, 54)
	for _, category := range catalog {
		category := category
		for _, coverageID := range category.coverages {
			coverageID := coverageID
			coverageCount++
			if _, duplicated := seen[coverageID]; duplicated {
				t.Fatalf("SOT-ENG-026: coverage catalog が重複した: %s", coverageID)
			}
			seen[coverageID] = struct{}{}
			t.Run(coverageID, func(t *testing.T) {
				semanticCase := holdoutRequirementsTestCase(
					t,
					0,
					[]string{coverageID},
					"",
				)
				if got := semanticCase.CategoryIDs(); !reflect.DeepEqual(
					got,
					[]string{category.id},
				) {
					t.Fatalf(
						"SOT-ENG-026: CategoryIDs() = %#v, want %q",
						got,
						category.id,
					)
				}
			})
		}
	}
	if coverageCount != 54 || len(semanticCoverageIDs()) != 54 {
		t.Fatalf(
			"SOT-ENG-026: coverage catalog 件数 = (%d, %d), want 54",
			coverageCount,
			len(semanticCoverageIDs()),
		)
	}
	for _, coverageID := range semanticCoverageIDs() {
		if _, exists := seen[coverageID]; !exists {
			t.Fatalf("SOT-ENG-026: coverage mapping が不足した: %s", coverageID)
		}
	}
}

func TestCategoryIDsはManifest順のUnique値を複製して返す(t *testing.T) {
	allCoverages := append([]string{}, semanticCoverageIDs()...)
	semanticCase := holdoutRequirementsTestCase(
		t,
		0,
		allCoverages,
		SafetyVariantOrdinary,
	)
	want := manifestRequiredCategoryIDs()
	got := semanticCase.CategoryIDs()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SOT-ENG-026: CategoryIDs() = %#v, want %#v", got, want)
	}
	got[0] = "changed"
	if again := semanticCase.CategoryIDs(); !reflect.DeepEqual(again, want) {
		t.Fatal("SOT-ENG-026: CategoryIDs getter から内部 slice が変更された")
	}
}

func TestHoldoutRequirementsのErrorはQueryとRef原文を含まない(
	t *testing.T,
) {
	const (
		query      = "secret-holdout-query"
		providerID = "secret-holdout-provider"
		sourceID   = "secret-holdout-source"
		resourceID = "secret-holdout-resource"
		versionID  = "secret-holdout-version"
	)
	holdout := holdoutRequirementsTestValidHoldout(t)
	request := setSeparationTestRequest(t, setSeparationTestRequestValues{
		query:        query,
		hasRef:       true,
		providerID:   providerID,
		sourceID:     sourceID,
		resourceType: "law",
		resourceID:   resourceID,
		hasVersion:   true,
		versionID:    versionID,
	})
	holdout[0] = holdoutRequirementsTestCaseWithRequest(
		t,
		holdout[0],
		request,
	)
	holdout[19] = holdoutRequirementsTestCase(
		t,
		19,
		[]string{"budget-capability-call-limit"},
		"",
	)

	err := validateHoldoutRequirements(holdout)
	if err == nil {
		t.Fatal("SOT-ENG-024: category 不足を受理した")
	}
	for _, secret := range []string{
		query,
		providerID,
		sourceID,
		resourceID,
		versionID,
	} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("SOT-ENG-026: error が query/ref 原文を含む: %v", err)
		}
	}
}

func holdoutRequirementsTestValidHoldout(t *testing.T) []SemanticCase {
	t.Helper()

	cases := make([]SemanticCase, 0, 240)
	safetyCounts := make(map[string]int)
	for _, category := range holdoutRequirementsTestCatalog() {
		for offset := 0; offset < 20; offset++ {
			coverageID := category.coverages[offset%len(category.coverages)]
			variant := SafetyVariant("")
			if isSemanticSafetyCoverageID(coverageID) {
				if safetyCounts[coverageID]%2 == 0 {
					variant = SafetyVariantOrdinary
				} else {
					variant = SafetyVariantAdversarial
				}
				safetyCounts[coverageID]++
			}
			cases = append(cases, holdoutRequirementsTestCase(
				t,
				len(cases),
				[]string{coverageID},
				variant,
			))
		}
	}
	if len(cases) != 240 {
		t.Fatalf("SOT-ENG-024: test holdout 件数 = %d", len(cases))
	}
	return cases
}

func holdoutRequirementsTestCase(
	t *testing.T,
	index int,
	coverageIDs []string,
	variant SafetyVariant,
) SemanticCase {
	t.Helper()

	hasSafety := false
	for _, coverageID := range coverageIDs {
		hasSafety = hasSafety || isSemanticSafetyCoverageID(coverageID)
	}
	var safetyVariant *SafetyVariant
	if hasSafety {
		if variant == "" {
			variant = SafetyVariantOrdinary
		}
		value := variant
		safetyVariant = &value
	}
	values := validSemanticCaseValues(t)
	values.CaseID = fmt.Sprintf("holdout-case-%03d", index)
	values.LeakageGroupID = fmt.Sprintf("holdout-group-%03d", index)
	values.CoverageIDs = append([]string{}, coverageIDs...)
	values.SafetyVariant = safetyVariant
	values.Request = mustRawRequest(
		t,
		fmt.Sprintf("法令照会%d", index),
		nil,
		holdoutRequirementsTestIntPointer(10),
	)
	semanticCase, err := NewSemanticCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: holdout test case error = %v", err)
	}
	return semanticCase
}

func holdoutRequirementsTestCaseWithRequest(
	t *testing.T,
	source SemanticCase,
	request Request,
) SemanticCase {
	t.Helper()

	var safetyVariant *SafetyVariant
	if variant, exists := source.SafetyVariant(); exists {
		safetyVariant = &variant
	}
	semanticCase, err := NewSemanticCase(SemanticCaseValues{
		ArtifactKind:   source.ArtifactKind(),
		SchemaVersion:  source.SchemaVersion(),
		CaseID:         source.CaseID(),
		LeakageGroupID: source.LeakageGroupID(),
		CoverageIDs:    source.CoverageIDs(),
		SafetyVariant:  safetyVariant,
		EnabledPacks:   source.EnabledPacks(),
		Request:        request,
		Expected:       source.Expected(),
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: request 置換 case error = %v", err)
	}
	return semanticCase
}

func holdoutRequirementsTestNonTargetCoverage(
	coverages []string,
	target string,
) string {
	for _, coverageID := range coverages {
		if coverageID != target {
			return coverageID
		}
	}
	panic("同じ category に代替 coverage がありません")
}

func holdoutRequirementsTestKeepCoverageCount(
	t *testing.T,
	holdout []SemanticCase,
	categoryCoverages []string,
	coverageID string,
	keep int,
) []SemanticCase {
	t.Helper()

	values := append([]SemanticCase{}, holdout...)
	replacement := holdoutRequirementsTestNonTargetCoverage(
		categoryCoverages,
		coverageID,
	)
	for index, semanticCase := range values {
		if !holdoutRequirementsTestContains(
			semanticCase.CoverageIDs(),
			coverageID,
		) {
			continue
		}
		if keep > 0 {
			keep--
			continue
		}
		variant, _ := semanticCase.SafetyVariant()
		values[index] = holdoutRequirementsTestCase(
			t,
			index,
			[]string{replacement},
			variant,
		)
	}
	return values
}

func holdoutRequirementsTestContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func holdoutRequirementsTestIntPointer(value int) *int {
	return &value
}

func holdoutRequirementsTestSafetyCoverageIDs() []string {
	return []string{
		"boundary-budget-limit",
		"boundary-mixed-unsupported",
		"boundary-no-implicit-first-read",
		"boundary-non-japanese",
		"boundary-pack-disabled",
	}
}

func holdoutRequirementsTestCatalog() []holdoutRequirementsTestCategory {
	return []holdoutRequirementsTestCategory{
		{
			id: "ambiguity",
			coverages: []string{
				"ambiguity-alias-collision",
				"ambiguity-multiple-concepts",
				"ambiguity-three-or-more-candidates",
				"ambiguity-weak-general-term",
			},
		},
		{
			id: "budget-boundary",
			coverages: []string{
				"boundary-budget-limit",
				"budget-capability-call-limit",
				"budget-item-limit",
				"budget-page-limit",
				"budget-ranked-candidate-limit",
				"budget-step-limit",
			},
		},
		{
			id: "capability-intent",
			coverages: []string{
				"intent-judicial-decision-read",
				"intent-judicial-decision-search",
				"intent-law-article-read",
				"intent-law-content-search",
				"intent-law-read",
				"intent-law-search",
				"intent-law-updates",
			},
		},
		{
			id: "input-boundary",
			coverages: []string{
				"input-invalid-ref",
				"input-limit-above-maximum",
				"input-limit-below-minimum",
				"input-limit-maximum-accepted",
				"input-limit-minimum-accepted",
				"input-query-ascii-control",
				"input-query-empty",
				"input-query-maximum-accepted",
				"input-query-too-long",
			},
		},
		{
			id: "law-name-and-concept",
			coverages: []string{
				"concept-single",
				"name-official",
				"name-official-abbreviation",
				"name-sourced-alias",
			},
		},
		{
			id: "official-reference",
			coverages: []string{
				"reference-case-reference",
				"reference-law-id",
				"reference-law-number",
				"reference-revision-id",
				"reference-source-resource-ref",
			},
		},
		{
			id: "pack-state",
			coverages: []string{
				"boundary-pack-disabled",
				"pack-judicial-enabled",
			},
		},
		{
			id: "safety-execution-boundary",
			coverages: []string{
				"boundary-mixed-unsupported",
				"boundary-no-implicit-first-read",
				"boundary-non-japanese",
			},
		},
		{
			id: "structured-location-and-date",
			coverages: []string{
				"structure-article",
				"structure-complete-date",
				"structure-multiple-explicit-intents",
				"structure-paragraph",
			},
		},
		{
			id: "surface-variation",
			coverages: []string{
				"surface-orthographic-variation",
				"surface-whitespace-variation",
			},
		},
		{
			id: "typo-variation",
			coverages: []string{
				"typo-adjacent-transposition",
				"typo-deletion",
				"typo-insertion",
				"typo-substitution",
			},
		},
		{
			id: "unsupported-scope",
			coverages: []string{
				"unsupported-legal-advice",
				"unsupported-resource",
				"unsupported-translation",
				"unsupported-unadopted-pack",
			},
		},
	}
}
