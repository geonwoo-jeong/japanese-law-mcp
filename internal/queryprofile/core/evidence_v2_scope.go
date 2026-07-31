package core

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type coreTaskScope struct {
	relation   legalquery.CueTaskRelation
	taskFactID string
}

func coreTaskScopeFor(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	taskValue string,
	targetSpans []legalquery.QuerySpan,
) (coreTaskScope, bool) {
	var result coreTaskScope
	found := false
	for _, relation := range input.CueTaskRelations() {
		if relation.Kind() != legalquery.CueTaskRelationDirectTask {
			continue
		}
		subject := relation.Subject()
		if !coreResolvedCueRefMatches(
			cues.mentions[cueMeaningKey("task", taskValue)],
			subject,
		) || !coreClauseContainsAll(relation.ClauseSpan(), targetSpans) {
			continue
		}
		factID, exists := coreCueFactID(input, subject)
		if !exists || found {
			return coreTaskScope{}, false
		}
		result = coreTaskScope{
			relation:   relation,
			taskFactID: factID,
		}
		found = true
	}
	return result, found
}

func coreTaskClauseForStart(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	taskValue string,
	startByte int,
) (legalquery.QuerySpan, bool) {
	var result legalquery.QuerySpan
	found := false
	for _, relation := range input.CueTaskRelations() {
		if relation.Kind() != legalquery.CueTaskRelationDirectTask ||
			!coreSpanContainsByte(relation.ClauseSpan(), startByte) {
			continue
		}
		if !coreResolvedCueRefMatches(
			cues.mentions[cueMeaningKey("task", taskValue)],
			relation.Subject(),
		) || found {
			return legalquery.QuerySpan{}, false
		}
		result = relation.ClauseSpan()
		found = true
	}
	return result, found
}

func coreResolvedCueRefMatches(
	mentions []legalquery.CueMention,
	ref legalquery.CueTaskRelationRef,
) bool {
	for _, mention := range mentions {
		if mention.ProfileID() == ref.ProfileID() &&
			mention.CueID() == ref.CueID() &&
			sameQuerySpan(mention.Span(), ref.Span()) {
			return true
		}
	}
	return false
}

func coreCueFactID(
	input legalquery.CandidateGenerationInput,
	ref legalquery.CueTaskRelationRef,
) (string, bool) {
	for index, mention := range input.CueMentions() {
		if mention.ProfileID() == ref.ProfileID() &&
			mention.CueID() == ref.CueID() &&
			sameQuerySpan(mention.Span(), ref.Span()) {
			return fmt.Sprintf("cue-%d", index+1), true
		}
	}
	return "", false
}

func coreClauseContainsAll(
	clause legalquery.QuerySpan,
	spans []legalquery.QuerySpan,
) bool {
	for _, span := range spans {
		if !coreSpanContains(clause, span) {
			return false
		}
	}
	return true
}

func coreSpanContains(
	container legalquery.QuerySpan,
	target legalquery.QuerySpan,
) bool {
	return container.StartByte() <= target.StartByte() &&
		target.EndByte() <= container.EndByte()
}

func coreSpanContainsByte(
	container legalquery.QuerySpan,
	position int,
) bool {
	return container.StartByte() <= position &&
		position < container.EndByte()
}

func coreResourceFactIDsInClause(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	resourceValue string,
	clause legalquery.QuerySpan,
) []string {
	var result []string
	for _, resolved := range cues.mentions[cueMeaningKey("resource", resourceValue)] {
		if !coreSpanContains(clause, resolved.Span()) {
			continue
		}
		for index, mention := range input.CueMentions() {
			if mention.ProfileID() != resolved.ProfileID() ||
				mention.CueID() != resolved.CueID() ||
				!sameQuerySpan(mention.Span(), resolved.Span()) {
				continue
			}
			result = append(result, fmt.Sprintf("cue-%d", index+1))
			break
		}
	}
	return result
}

func coreEvidenceFactSpans(
	input legalquery.CandidateGenerationInput,
	values []profileevidence.EvidenceValues,
) []legalquery.QuerySpan {
	result := make([]legalquery.QuerySpan, 0, len(values))
	for _, value := range values {
		span, exists := coreEvidenceFactSpan(input, value.FactID)
		if !exists {
			continue
		}
		duplicate := false
		for _, current := range result {
			if sameQuerySpan(current, span) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, span)
		}
	}
	return result
}

func coreEvidenceFactSpan(
	input legalquery.CandidateGenerationInput,
	factID string,
) (legalquery.QuerySpan, bool) {
	prefix, ordinal, valid := coreFactIDParts(factID)
	if !valid {
		return legalquery.QuerySpan{}, false
	}
	index := ordinal - 1
	switch prefix {
	case "law-name":
		values := input.LawNameMentions()
		if index < len(values) {
			return values[index].Span(), true
		}
	case "legal-concept":
		values := input.LegalConceptMentions()
		if index < len(values) {
			return values[index].Span(), true
		}
	case "cue":
		values := input.CueMentions()
		if index < len(values) {
			return values[index].Span(), true
		}
	case "identifier":
		values := input.IdentifierMentions()
		if index < len(values) {
			return values[index].Span(), true
		}
	case "date":
		values := input.DateMentions()
		if index < len(values) {
			return values[index].Span(), true
		}
	case "article":
		values := input.ArticleMentions()
		if index < len(values) {
			return values[index].Span(), true
		}
	case "paragraph":
		values := input.ParagraphMentions()
		if index < len(values) {
			return values[index].Span(), true
		}
	case "query-term":
		values := input.QueryTermMentions()
		if index < len(values) {
			return values[index].Span(), true
		}
	}
	return legalquery.QuerySpan{}, false
}

func coreFactIDParts(value string) (string, int, bool) {
	for index := len(value) - 1; index >= 0; index-- {
		if value[index] != '-' {
			continue
		}
		ordinal := 0
		for _, current := range value[index+1:] {
			if current < '0' || current > '9' {
				return "", 0, false
			}
			ordinal = ordinal*10 + int(current-'0')
		}
		if ordinal < 1 {
			return "", 0, false
		}
		return value[:index], ordinal, true
	}
	return "", 0, false
}
