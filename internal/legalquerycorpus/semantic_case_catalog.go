package legalquerycorpus

import (
	"fmt"
	"sort"
)

const (
	semanticCategoryAmbiguity                 = "ambiguity"
	semanticCategoryBudgetBoundary            = "budget-boundary"
	semanticCategoryCapabilityIntent          = "capability-intent"
	semanticCategoryInputBoundary             = "input-boundary"
	semanticCategoryLawNameAndConcept         = "law-name-and-concept"
	semanticCategoryOfficialReference         = "official-reference"
	semanticCategoryPackState                 = "pack-state"
	semanticCategorySafetyExecutionBoundary   = "safety-execution-boundary"
	semanticCategoryStructuredLocationAndDate = "structured-location-and-date"
	semanticCategorySurfaceVariation          = "surface-variation"
	semanticCategoryTypoVariation             = "typo-variation"
	semanticCategoryUnsupportedScope          = "unsupported-scope"
)

type semanticCoverageDefinition struct {
	id                        string
	categoryID                string
	minimumHoldoutCount       int
	requiresSafetyVariantPair bool
}

func validateSemanticCoverage(
	schemaVersion int,
	coverageIDs []string,
	safetyVariant *SafetyVariant,
) error {
	if len(coverageIDs) < 1 || len(coverageIDs) > maximumSemanticCaseListItems {
		return fmt.Errorf("coverageIds は一件以上六十四件以下でなければなりません")
	}
	previous := ""
	requiresSafetyVariant := false
	for index, coverageID := range coverageIDs {
		if !isSemanticCoverageIDForSchemaVersion(schemaVersion, coverageID) {
			return fmt.Errorf("coverageIds に未定義の値があります")
		}
		if index > 0 && previous >= coverageID {
			return fmt.Errorf("coverageIds は昇順で重複なく保持してください")
		}
		previous = coverageID
		requiresSafetyVariant = requiresSafetyVariant ||
			isSemanticSafetyCoverageIDForSchemaVersion(schemaVersion, coverageID)
	}
	if requiresSafetyVariant != (safetyVariant != nil) {
		return fmt.Errorf("safetyVariant の存在が safety coverage と一致しません")
	}
	if safetyVariant != nil &&
		*safetyVariant != SafetyVariantOrdinary &&
		*safetyVariant != SafetyVariantAdversarial {
		return fmt.Errorf("safetyVariant が定義されていません")
	}
	return nil
}

func validateSemanticEnabledPacks(values []string) error {
	if len(values) > maximumSemanticCaseListItems {
		return fmt.Errorf("enabledPacks は六十四件以下でなければなりません")
	}
	previous := ""
	for index, value := range values {
		if err := validateExpectedIdentifier("enabledPacks", value); err != nil {
			return err
		}
		if index > 0 && previous >= value {
			return fmt.Errorf("enabledPacks は昇順で重複なく保持してください")
		}
		previous = value
	}
	return nil
}

func isSemanticCoverageID(value string) bool {
	_, exists := semanticCoverageDefinitionForSchemaVersion(
		corpusSchemaVersionV1,
		value,
	)
	return exists
}

func isSemanticSafetyCoverageID(value string) bool {
	definition, exists := semanticCoverageDefinitionForSchemaVersion(
		corpusSchemaVersionV1,
		value,
	)
	return exists && definition.requiresSafetyVariantPair
}

func isSemanticCoverageIDForSchemaVersion(version int, value string) bool {
	_, exists := semanticCoverageDefinitionForSchemaVersion(version, value)
	return exists
}

func isSemanticSafetyCoverageIDForSchemaVersion(version int, value string) bool {
	definition, exists := semanticCoverageDefinitionForSchemaVersion(version, value)
	return exists && definition.requiresSafetyVariantPair
}

func semanticCoverageIDs() []string {
	return semanticCoverageIDsForSchemaVersion(corpusSchemaVersionV1)
}

func semanticCoverageIDsForSchemaVersion(version int) []string {
	definitions := semanticCoverageDefinitionsForSchemaVersion(version)
	values := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		values = append(values, definition.id)
	}
	return values
}

func semanticCategoryIDs() []string {
	return []string{
		semanticCategoryAmbiguity,
		semanticCategoryBudgetBoundary,
		semanticCategoryCapabilityIntent,
		semanticCategoryInputBoundary,
		semanticCategoryLawNameAndConcept,
		semanticCategoryOfficialReference,
		semanticCategoryPackState,
		semanticCategorySafetyExecutionBoundary,
		semanticCategoryStructuredLocationAndDate,
		semanticCategorySurfaceVariation,
		semanticCategoryTypoVariation,
		semanticCategoryUnsupportedScope,
	}
}

func semanticCategoryIDsForCoverageIDs(
	schemaVersion int,
	coverageIDs []string,
) []string {
	present := make(map[string]struct{}, len(coverageIDs))
	for _, coverageID := range coverageIDs {
		definition, exists := semanticCoverageDefinitionForSchemaVersion(
			schemaVersion,
			coverageID,
		)
		if exists {
			present[definition.categoryID] = struct{}{}
		}
	}
	values := make([]string, 0, len(present))
	for _, categoryID := range semanticCategoryIDs() {
		if _, exists := present[categoryID]; exists {
			values = append(values, categoryID)
		}
	}
	return values
}

func semanticCoverageDefinitionFor(
	coverageID string,
) (semanticCoverageDefinition, bool) {
	return semanticCoverageDefinitionForSchemaVersion(
		corpusSchemaVersionV1,
		coverageID,
	)
}

func semanticCoverageDefinitionForSchemaVersion(
	schemaVersion int,
	coverageID string,
) (semanticCoverageDefinition, bool) {
	catalog := semanticCoverageDefinitionsForSchemaVersion(schemaVersion)
	low, high := 0, len(catalog)
	for low < high {
		middle := low + (high-low)/2
		if catalog[middle].id < coverageID {
			low = middle + 1
		} else {
			high = middle
		}
	}
	if low < len(catalog) && catalog[low].id == coverageID {
		return catalog[low], true
	}
	return semanticCoverageDefinition{}, false
}

func semanticCoverageDefinitions() []semanticCoverageDefinition {
	return append([]semanticCoverageDefinition{}, semanticCoverageCatalog[:]...)
}

func semanticCoverageDefinitionsForSchemaVersion(
	schemaVersion int,
) []semanticCoverageDefinition {
	switch schemaVersion {
	case corpusSchemaVersionV1:
		return semanticCoverageDefinitions()
	case corpusSchemaVersionV2:
		return append([]semanticCoverageDefinition{}, semanticCoverageCatalogV2...)
	default:
		return nil
	}
}

// semanticCoverageCatalog は、ID 昇順の固定 catalog であり初期化後に変更しない。
var semanticCoverageCatalog = [...]semanticCoverageDefinition{
	standardSemanticCoverage("ambiguity-alias-collision", semanticCategoryAmbiguity),
	standardSemanticCoverage("ambiguity-multiple-concepts", semanticCategoryAmbiguity),
	standardSemanticCoverage("ambiguity-three-or-more-candidates", semanticCategoryAmbiguity),
	standardSemanticCoverage("ambiguity-weak-general-term", semanticCategoryAmbiguity),
	safetySemanticCoverage("boundary-budget-limit", semanticCategoryBudgetBoundary),
	safetySemanticCoverage("boundary-mixed-unsupported", semanticCategorySafetyExecutionBoundary),
	safetySemanticCoverage("boundary-no-implicit-first-read", semanticCategorySafetyExecutionBoundary),
	safetySemanticCoverage("boundary-non-japanese", semanticCategorySafetyExecutionBoundary),
	safetySemanticCoverage("boundary-pack-disabled", semanticCategoryPackState),
	standardSemanticCoverage("budget-capability-call-limit", semanticCategoryBudgetBoundary),
	standardSemanticCoverage("budget-item-limit", semanticCategoryBudgetBoundary),
	standardSemanticCoverage("budget-page-limit", semanticCategoryBudgetBoundary),
	standardSemanticCoverage("budget-ranked-candidate-limit", semanticCategoryBudgetBoundary),
	standardSemanticCoverage("budget-step-limit", semanticCategoryBudgetBoundary),
	standardSemanticCoverage("concept-single", semanticCategoryLawNameAndConcept),
	standardSemanticCoverage("input-invalid-ref", semanticCategoryInputBoundary),
	standardSemanticCoverage("input-limit-above-maximum", semanticCategoryInputBoundary),
	standardSemanticCoverage("input-limit-below-minimum", semanticCategoryInputBoundary),
	standardSemanticCoverage("input-limit-maximum-accepted", semanticCategoryInputBoundary),
	standardSemanticCoverage("input-limit-minimum-accepted", semanticCategoryInputBoundary),
	standardSemanticCoverage("input-query-ascii-control", semanticCategoryInputBoundary),
	standardSemanticCoverage("input-query-empty", semanticCategoryInputBoundary),
	standardSemanticCoverage("input-query-maximum-accepted", semanticCategoryInputBoundary),
	standardSemanticCoverage("input-query-too-long", semanticCategoryInputBoundary),
	standardSemanticCoverage("intent-judicial-decision-read", semanticCategoryCapabilityIntent),
	standardSemanticCoverage("intent-judicial-decision-search", semanticCategoryCapabilityIntent),
	standardSemanticCoverage("intent-law-article-read", semanticCategoryCapabilityIntent),
	standardSemanticCoverage("intent-law-content-search", semanticCategoryCapabilityIntent),
	standardSemanticCoverage("intent-law-read", semanticCategoryCapabilityIntent),
	standardSemanticCoverage("intent-law-search", semanticCategoryCapabilityIntent),
	standardSemanticCoverage("intent-law-updates", semanticCategoryCapabilityIntent),
	standardSemanticCoverage("name-official", semanticCategoryLawNameAndConcept),
	standardSemanticCoverage("name-official-abbreviation", semanticCategoryLawNameAndConcept),
	standardSemanticCoverage("name-sourced-alias", semanticCategoryLawNameAndConcept),
	standardSemanticCoverage("pack-judicial-enabled", semanticCategoryPackState),
	standardSemanticCoverage("reference-case-reference", semanticCategoryOfficialReference),
	standardSemanticCoverage("reference-law-id", semanticCategoryOfficialReference),
	standardSemanticCoverage("reference-law-number", semanticCategoryOfficialReference),
	standardSemanticCoverage("reference-revision-id", semanticCategoryOfficialReference),
	standardSemanticCoverage("reference-source-resource-ref", semanticCategoryOfficialReference),
	standardSemanticCoverage("structure-article", semanticCategoryStructuredLocationAndDate),
	standardSemanticCoverage("structure-complete-date", semanticCategoryStructuredLocationAndDate),
	standardSemanticCoverage("structure-multiple-explicit-intents", semanticCategoryStructuredLocationAndDate),
	standardSemanticCoverage("structure-paragraph", semanticCategoryStructuredLocationAndDate),
	standardSemanticCoverage("surface-orthographic-variation", semanticCategorySurfaceVariation),
	standardSemanticCoverage("surface-whitespace-variation", semanticCategorySurfaceVariation),
	standardSemanticCoverage("typo-adjacent-transposition", semanticCategoryTypoVariation),
	standardSemanticCoverage("typo-deletion", semanticCategoryTypoVariation),
	standardSemanticCoverage("typo-insertion", semanticCategoryTypoVariation),
	standardSemanticCoverage("typo-substitution", semanticCategoryTypoVariation),
	standardSemanticCoverage("unsupported-legal-advice", semanticCategoryUnsupportedScope),
	standardSemanticCoverage("unsupported-resource", semanticCategoryUnsupportedScope),
	standardSemanticCoverage("unsupported-translation", semanticCategoryUnsupportedScope),
	standardSemanticCoverage("unsupported-unadopted-pack", semanticCategoryUnsupportedScope),
}

var semanticCoverageCatalogV2 = func() []semanticCoverageDefinition {
	values := semanticCoverageDefinitions()
	values = append(values,
		safetySemanticCoverage(
			"boundary-no-unbounded-fanout",
			semanticCategorySafetyExecutionBoundary,
		),
		safetySemanticCoverage(
			"boundary-unmarked-enumeration",
			semanticCategorySafetyExecutionBoundary,
		),
		safetySemanticCoverage(
			"boundary-unsupported-candidate-scope",
			semanticCategorySafetyExecutionBoundary,
		),
		safetySemanticCoverage(
			"boundary-unsupported-cue-context",
			semanticCategorySafetyExecutionBoundary,
		),
		standardSemanticCoverage(
			"structure-shared-terminal-cue",
			semanticCategoryStructuredLocationAndDate,
		),
		standardSemanticCoverage(
			"unsupported-relationship-analysis",
			semanticCategoryUnsupportedScope,
		),
		standardSemanticCoverage(
			"unsupported-version-comparison",
			semanticCategoryUnsupportedScope,
		),
	)
	sort.Slice(values, func(left int, right int) bool {
		return values[left].id < values[right].id
	})
	return values
}()

func standardSemanticCoverage(
	id string,
	categoryID string,
) semanticCoverageDefinition {
	return semanticCoverageDefinition{
		id:                  id,
		categoryID:          categoryID,
		minimumHoldoutCount: 1,
	}
}

func safetySemanticCoverage(
	id string,
	categoryID string,
) semanticCoverageDefinition {
	return semanticCoverageDefinition{
		id:                        id,
		categoryID:                categoryID,
		minimumHoldoutCount:       2,
		requiresSafetyVariantPair: true,
	}
}
