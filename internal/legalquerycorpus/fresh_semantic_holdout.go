package legalquerycorpus

import (
	"fmt"
	"slices"
	"strings"
)

// validateFreshSemanticHoldout は、SOT-ENG-046 の空入力分離を版と集合に限定する。
func (c SemanticCase) validateFreshSemanticHoldout() error {
	if c.schemaVersion != corpusSchemaVersionV3 ||
		!strings.HasPrefix(c.caseID, "holdout-") {
		return nil
	}
	if slices.Contains(c.coverageIDs, "input-query-empty") ||
		strings.TrimSpace(c.request.Query()) == "" {
		return fmt.Errorf("schema version 3 の holdout は空入力の意味家族を保持できません")
	}
	return nil
}

func semanticHoldoutCoverageDefinitions(version int) []semanticCoverageDefinition {
	definitions := semanticCoverageDefinitionsForSchemaVersion(version)
	if version != corpusSchemaVersionV3 {
		return definitions
	}
	filtered := make([]semanticCoverageDefinition, 0, len(definitions)-1)
	for _, definition := range definitions {
		if definition.id != "input-query-empty" {
			filtered = append(filtered, definition)
		}
	}
	return filtered
}
