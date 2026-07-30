package core

import (
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func buildRefFollowupLawSearchDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	read []candidateDraft,
	content []candidateDraft,
	update *candidateDraft,
	asOf *model.Date,
) ([]candidateDraft, bool, error) {
	ref, hasRef := input.Ref()
	if !hasRef ||
		ref.Key().ResourceType() != "law" ||
		!cues.has("task", "read") ||
		!cues.has("task", "search") ||
		!cues.has("resource", "law") ||
		len(read) != 1 ||
		len(content) != 1 ||
		update != nil ||
		hasRefFollowupSearchAlternative(cues) {
		return nil, false, nil
	}

	targets := refFollowupLawSearchTargets(input, cues)
	if len(targets) != 1 {
		return nil, false, nil
	}
	search, err := buildLawSearchDrafts(
		targets,
		true,
		false,
		asOf,
	)
	if err != nil {
		return nil, false, err
	}
	if len(search) != 1 ||
		len(read[0].steps)+len(search[0].steps)+len(content[0].steps) > 4 {
		return nil, false, nil
	}

	readAndSearch := combineDraftSets(read, search, nil)
	return combineDraftSets(readAndSearch, content, nil), true, nil
}

func hasRefFollowupSearchAlternative(cues resolvedCues) bool {
	return cues.has("operator", "all") ||
		cues.has("operator", "any") ||
		cues.has("operator", "dual_candidate") ||
		cues.has("operator", "exclude") ||
		cues.has("operator", "resource_choice") ||
		cues.has("operator", "single_choice")
}

func refFollowupLawSearchTargets(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) []lawTarget {
	readCues := cues.mentions[cueMeaningKey("task", "read")]
	searchCues := cues.mentions[cueMeaningKey("task", "search")]
	result := make([]lawTarget, 0)
	for _, target := range buildMentionLawTargets(input) {
		if targetFollowsReadAndPrecedesSearch(
			target,
			readCues,
			searchCues,
		) {
			result = append(result, target)
		}
	}
	return result
}

func targetFollowsReadAndPrecedesSearch(
	target lawTarget,
	readCues []legalquery.CueMention,
	searchCues []legalquery.CueMention,
) bool {
	followsRead := false
	for _, read := range readCues {
		if read.Span().EndByte() <= target.startByte {
			followsRead = true
			break
		}
	}
	if !followsRead {
		return false
	}
	for _, search := range searchCues {
		if target.endByte <= search.Span().StartByte() &&
			!cueStartsBetween(
				readCues,
				target.endByte,
				search.Span().StartByte(),
			) {
			return true
		}
	}
	return false
}
