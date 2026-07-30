package core

import "github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"

func isCoreResourceChoice(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	return cues.has("operator", "resource_choice") &&
		cues.has("resource", "law") &&
		cues.has("resource", "law_provision") &&
		!cues.has("resource", "updates") &&
		len(buildLawTargets(input, cues)) == 0 &&
		len(coreContentQueryTerms(input, cues)) == 1
}
