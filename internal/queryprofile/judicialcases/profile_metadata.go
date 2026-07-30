package judicialcases

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func validateJudicialMetadata(
	metadata legalquery.QueryProfileMetadata,
	conditionalTieBreaksPresent bool,
) error {
	if !isExactJudicialTargets(metadata.Targets()) {
		return fmt.Errorf(
			"judicial-cases profile targets は裁判例二能力の固定順でなければなりません",
		)
	}
	if conditionalTieBreaksPresent ||
		len(metadata.ConditionalTieBreaks()) != 0 {
		return fmt.Errorf(
			"judicial-cases profile に conditionalTieBreaks は指定できません",
		)
	}
	return nil
}

func isExactJudicialTargets(
	values []legalquery.QueryProfileTarget,
) bool {
	expected := []legalquery.LogicalInputKind{
		legalquery.InputKindJudicialDecisionSearch,
		legalquery.InputKindJudicialDecisionRead,
	}
	if len(values) != len(expected) {
		return false
	}
	for index, target := range values {
		if target.InputKind() != expected[index] {
			return false
		}
	}
	return true
}
