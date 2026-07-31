package core

import (
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func (p *Profile) buildCoreProjectedLawNameDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, bool, error) {
	if hasExplicitContentOperator(cues) {
		return nil, false, nil
	}

	individual := cues.has("operator", "individual")
	seen := make(map[legalquery.QuerySpan]struct{})
	options := make([]coreTopicOption, 0, len(input.LawNameMentions()))
	for _, mention := range input.LawNameMentions() {
		span := mention.Span()
		if _, exists := seen[span]; exists {
			continue
		}
		seen[span] = struct{}{}

		var individualTopic *legalquery.QuerySpan
		if individual {
			current := span
			individualTopic = &current
		}
		projectionContext, err := newCoreLawNameProjectionContext(
			span,
			individualTopic,
			nil,
		)
		if err != nil {
			return nil, false, err
		}
		option, exists, err := p.projectCoreLawName(
			input,
			cues,
			projectionContext,
		)
		if err != nil {
			return nil, false, err
		}
		if exists {
			options = append(options, option)
		}
	}

	if len(options) == 0 {
		return nil, false, nil
	}
	sort.SliceStable(options, func(left int, right int) bool {
		return options[left].startByte < options[right].startByte
	})
	draft, err := buildCoreProjectedContentDraft(options)
	if err != nil {
		return nil, false, err
	}
	return []candidateDraft{draft}, true, nil
}

func buildCoreProjectedContentDraft(
	options []coreTopicOption,
) (candidateDraft, error) {
	draft := newCandidateDraft()
	for index, option := range options {
		draft.steps = append(draft.steps, stepDraft{
			startByte:        option.startByte,
			topicOrdinal:     index + 1,
			input:            option.input,
			evidenceBindings: option.evidence,
		})
		draft.concepts = append(draft.concepts, option.concepts...)
		for _, evidence := range option.evidence {
			draft.evidence[evidence.Code] = struct{}{}
		}
	}
	return draft, nil
}
