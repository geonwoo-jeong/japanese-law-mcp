package judicialcases

import "github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"

func isJudicialResourceChoice(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	return cues.has("operator", "resource_choice") &&
		cues.has("resource", "judicial_decision") &&
		len(foreignResourceSpans(input, cues)) > 0
}
