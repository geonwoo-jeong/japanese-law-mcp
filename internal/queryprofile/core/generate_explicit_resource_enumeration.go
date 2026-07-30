package core

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func buildExplicitLawAndContentSearchCandidate(
	cues resolvedCues,
	hasLawTargets bool,
	content []candidateDraft,
) ([]candidateDraft, bool, error) {
	if hasLawTargets ||
		!explicitLawAndContentResourcesRequested(cues) ||
		len(content) != 1 ||
		len(content[0].steps) != 1 {
		return nil, false, nil
	}

	contentInput, ok := content[0].steps[0].input.(legalquery.LawContentSearchIntentV1)
	if !ok ||
		len(contentInput.AllTerms()) != 1 ||
		len(contentInput.AnyTerms()) != 0 ||
		len(contentInput.ExcludeTerms()) != 0 {
		return nil, false, nil
	}
	asOf, hasAsOf := contentInput.AsOf()
	var asOfPointer *model.Date
	if hasAsOf {
		asOfPointer = &asOf
	}
	searchInput, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{
			Query: contentInput.AllTerms()[0],
			AsOf:  asOfPointer,
		},
	)
	if err != nil {
		return nil, false, fmt.Errorf(
			"明示された法令名検索条件を構築できません: %w",
			err,
		)
	}

	combined := cloneDraft(content[0])
	combined.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
	combined.evidence[legalquery.EvidenceExplicitResource] = struct{}{}
	combined.evidence[legalquery.EvidenceGeneralTerm] = struct{}{}
	combined.steps = append(
		[]stepDraft{{
			startByte: content[0].steps[0].startByte,
			input:     searchInput,
		}},
		combined.steps...,
	)
	return []candidateDraft{combined}, true, nil
}

func explicitLawAndContentResourcesRequested(cues resolvedCues) bool {
	return cues.has("task", "search") &&
		cues.has("resource", "law") &&
		cues.has("resource", "law_provision") &&
		!cues.has("operator", "dual_candidate") &&
		!cues.has("operator", "resource_choice") &&
		!cues.has("operator", "any") &&
		!cues.has("operator", "exclude") &&
		!cues.has("operator", "single_choice")
}
