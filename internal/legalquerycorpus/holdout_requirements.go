package legalquerycorpus

import (
	"fmt"
)

const (
	minimumHoldoutCaseCount         = 240
	minimumHoldoutCategoryCaseCount = 20
)

type holdoutSafetyVariantKey struct {
	coverageID string
	variant    SafetyVariant
}

type holdoutRequirementCounts struct {
	categories     map[string]int
	coverages      map[string]int
	safetyVariants map[holdoutSafetyVariantKey]int
}

func validateIntegrityHoldoutRequirements(
	checked integrityCheckedCorpus,
) (integrityCheckedCorpus, error) {
	schemaVersion := checked.manifest.SchemaVersion()
	if !isSupportedCorpusSchemaVersion(schemaVersion) && len(checked.holdout) > 0 {
		schemaVersion = checked.holdout[0].SchemaVersion()
	}
	if err := validateHoldoutRequirementsForSchemaVersion(
		schemaVersion,
		checked.holdout,
	); err != nil {
		return integrityCheckedCorpus{}, err
	}
	return checked, nil
}

// validateHoldoutRequirements は、holdout の規模、網羅性および安全対を確認する。
func validateHoldoutRequirements(holdout []SemanticCase) error {
	return validateHoldoutRequirementsForSchemaVersion(
		corpusSchemaVersionV1,
		holdout,
	)
}

func validateHoldoutRequirementsForSchemaVersion(
	schemaVersion int,
	holdout []SemanticCase,
) error {
	if len(holdout) < minimumHoldoutCaseCount {
		return fmt.Errorf(
			"holdout は%d件以上でなければなりません",
			minimumHoldoutCaseCount,
		)
	}
	counts := collectHoldoutRequirementCounts(holdout)
	if err := validateHoldoutCategoryCounts(counts.categories); err != nil {
		return err
	}
	if err := validateHoldoutCoverageCounts(
		schemaVersion,
		counts.coverages,
	); err != nil {
		return err
	}
	return validateHoldoutSafetyVariantPairs(
		schemaVersion,
		counts.safetyVariants,
	)
}

func collectHoldoutRequirementCounts(
	holdout []SemanticCase,
) holdoutRequirementCounts {
	counts := holdoutRequirementCounts{
		categories:     make(map[string]int),
		coverages:      make(map[string]int),
		safetyVariants: make(map[holdoutSafetyVariantKey]int),
	}
	for _, semanticCase := range holdout {
		for _, categoryID := range semanticCase.CategoryIDs() {
			counts.categories[categoryID]++
		}
		variant, hasVariant := semanticCase.SafetyVariant()
		for _, coverageID := range semanticCase.CoverageIDs() {
			counts.coverages[coverageID]++
			definition, exists := semanticCoverageDefinitionForSchemaVersion(
				semanticCase.SchemaVersion(),
				coverageID,
			)
			if exists && definition.requiresSafetyVariantPair && hasVariant {
				counts.safetyVariants[holdoutSafetyVariantKey{
					coverageID: coverageID,
					variant:    variant,
				}]++
			}
		}
	}
	return counts
}

func validateHoldoutCategoryCounts(counts map[string]int) error {
	for _, categoryID := range semanticCategoryIDs() {
		if counts[categoryID] < minimumHoldoutCategoryCaseCount {
			return fmt.Errorf(
				"holdout の categoryId %q は%d件以上でなければなりません",
				categoryID,
				minimumHoldoutCategoryCaseCount,
			)
		}
	}
	return nil
}

func validateHoldoutCoverageCounts(
	schemaVersion int,
	counts map[string]int,
) error {
	for _, definition := range semanticCoverageDefinitionsForSchemaVersion(
		schemaVersion,
	) {
		if counts[definition.id] < definition.minimumHoldoutCount {
			return fmt.Errorf(
				"holdout の coverageId %q は%d件以上でなければなりません",
				definition.id,
				definition.minimumHoldoutCount,
			)
		}
	}
	return nil
}

func validateHoldoutSafetyVariantPairs(
	schemaVersion int,
	counts map[holdoutSafetyVariantKey]int,
) error {
	for _, definition := range semanticCoverageDefinitionsForSchemaVersion(
		schemaVersion,
	) {
		if !definition.requiresSafetyVariantPair {
			continue
		}
		ordinary := holdoutSafetyVariantKey{
			coverageID: definition.id,
			variant:    SafetyVariantOrdinary,
		}
		adversarial := holdoutSafetyVariantKey{
			coverageID: definition.id,
			variant:    SafetyVariantAdversarial,
		}
		if counts[ordinary] < 1 || counts[adversarial] < 1 {
			return fmt.Errorf(
				"holdout の safety coverage %q は ordinary と adversarial を一件以上ずつ必要とします",
				definition.id,
			)
		}
	}
	return nil
}
