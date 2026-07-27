package legalquerycorpus

import "fmt"

func validateSemanticCoverage(
	coverageIDs []string,
	safetyVariant *SafetyVariant,
) error {
	if len(coverageIDs) < 1 || len(coverageIDs) > maximumSemanticCaseListItems {
		return fmt.Errorf("coverageIds は一件以上六十四件以下でなければなりません")
	}
	previous := ""
	requiresSafetyVariant := false
	for index, coverageID := range coverageIDs {
		if !isSemanticCoverageID(coverageID) {
			return fmt.Errorf("coverageIds に未定義の値があります")
		}
		if index > 0 && previous >= coverageID {
			return fmt.Errorf("coverageIds は昇順で重複なく保持してください")
		}
		previous = coverageID
		requiresSafetyVariant = requiresSafetyVariant ||
			isSemanticSafetyCoverageID(coverageID)
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
	for _, coverageID := range semanticCoverageIDs() {
		if value == coverageID {
			return true
		}
	}
	return false
}

func isSemanticSafetyCoverageID(value string) bool {
	switch value {
	case "boundary-budget-limit",
		"boundary-mixed-unsupported",
		"boundary-no-implicit-first-read",
		"boundary-non-japanese",
		"boundary-pack-disabled":
		return true
	default:
		return false
	}
}

func semanticCoverageIDs() []string {
	return []string{
		"ambiguity-alias-collision",
		"ambiguity-multiple-concepts",
		"ambiguity-three-or-more-candidates",
		"ambiguity-weak-general-term",
		"boundary-budget-limit",
		"boundary-mixed-unsupported",
		"boundary-no-implicit-first-read",
		"boundary-non-japanese",
		"boundary-pack-disabled",
		"budget-capability-call-limit",
		"budget-item-limit",
		"budget-page-limit",
		"budget-ranked-candidate-limit",
		"budget-step-limit",
		"concept-single",
		"input-invalid-ref",
		"input-limit-above-maximum",
		"input-limit-below-minimum",
		"input-limit-maximum-accepted",
		"input-limit-minimum-accepted",
		"input-query-ascii-control",
		"input-query-empty",
		"input-query-maximum-accepted",
		"input-query-too-long",
		"intent-judicial-decision-read",
		"intent-judicial-decision-search",
		"intent-law-article-read",
		"intent-law-content-search",
		"intent-law-read",
		"intent-law-search",
		"intent-law-updates",
		"name-official",
		"name-official-abbreviation",
		"name-sourced-alias",
		"pack-judicial-enabled",
		"reference-case-reference",
		"reference-law-id",
		"reference-law-number",
		"reference-revision-id",
		"reference-source-resource-ref",
		"structure-article",
		"structure-complete-date",
		"structure-multiple-explicit-intents",
		"structure-paragraph",
		"surface-orthographic-variation",
		"surface-whitespace-variation",
		"typo-adjacent-transposition",
		"typo-deletion",
		"typo-insertion",
		"typo-substitution",
		"unsupported-legal-advice",
		"unsupported-resource",
		"unsupported-translation",
		"unsupported-unadopted-pack",
	}
}
