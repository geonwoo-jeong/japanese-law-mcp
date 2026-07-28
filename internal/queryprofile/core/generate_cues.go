package core

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

type resolvedCues struct {
	mentions map[string][]legalquery.CueMention
}

func (p *Profile) resolveCues(
	values []legalquery.CueMention,
) (resolvedCues, error) {
	result := resolvedCues{mentions: make(map[string][]legalquery.CueMention)}
	for _, mention := range values {
		if mention.ProfileID() != p.metadata.ProfileID() {
			continue
		}
		definition, exists := p.cueByID[mention.CueID()]
		if !exists {
			return resolvedCues{}, fmt.Errorf(
				"core profile に未登録の cueId %q があります",
				mention.CueID(),
			)
		}
		key := cueMeaningKey(definition.category, definition.value)
		result.mentions[key] = append(result.mentions[key], mention)
	}
	return result, nil
}

func (c resolvedCues) has(category string, value string) bool {
	return len(c.mentions[cueMeaningKey(category, value)]) > 0
}

func cueMeaningKey(category string, value string) string {
	return category + "\x00" + value
}

func (p *Profile) generationSignals(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) []legalquery.CandidateGenerationSignal {
	present := make(map[legalquery.CandidateGenerationSignal]struct{})
	if input.Language() == legalquery.QueryLanguageNonJapanese {
		present[legalquery.CandidateSignalNonJapaneseQuery] = struct{}{}
	}
	if cues.has("unsupported", "legal_advice") {
		present[legalquery.CandidateSignalUnsupportedLegalAdvice] = struct{}{}
	}
	if cues.has("unsupported", "translation") {
		present[legalquery.CandidateSignalUnsupportedTranslation] = struct{}{}
	}
	if cues.has("unsupported", "task_or_resource") {
		present[legalquery.CandidateSignalUnsupportedTaskOrResource] = struct{}{}
	}
	if cues.has("reserved_pack", "judicial-cases") {
		present[legalquery.CandidateSignalReservedPackRequest] = struct{}{}
	}
	if ref, exists := input.Ref(); exists &&
		ref.Key().ResourceType() == "judicial-decision" {
		present[legalquery.CandidateSignalReservedPackRequest] = struct{}{}
	}
	order := []legalquery.CandidateGenerationSignal{
		legalquery.CandidateSignalNonJapaneseQuery,
		legalquery.CandidateSignalUnsupportedLegalAdvice,
		legalquery.CandidateSignalUnsupportedTranslation,
		legalquery.CandidateSignalUnsupportedTaskOrResource,
		legalquery.CandidateSignalReservedPackRequest,
	}
	result := make([]legalquery.CandidateGenerationSignal, 0, len(present))
	for _, signal := range order {
		if _, exists := present[signal]; exists {
			result = append(result, signal)
		}
	}
	return result
}
