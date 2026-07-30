package core

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func (p *Profile) buildRelationV2MentionTargetDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, error) {
	relationSubjects := make(map[cueRelationRefKey]struct{})
	for _, relation := range input.CueTaskRelations() {
		subject := relation.Subject()
		if subject.ProfileID() != p.metadata.ProfileID() {
			continue
		}
		relationSubjects[cueRelationRefKeyFromRelationRef(subject)] = struct{}{}
	}

	result := make([]candidateDraft, 0)
	seenClauses := make(map[[2]int]struct{})
	for _, relation := range input.CueTaskRelations() {
		if !p.isRelationV2DirectSearch(relation) {
			continue
		}
		clause := relation.ClauseSpan()
		clauseKey := [2]int{clause.StartByte(), clause.EndByte()}
		if _, exists := seenClauses[clauseKey]; exists {
			continue
		}
		seenClauses[clauseKey] = struct{}{}

		values, firstSpan := p.relationV2MentionTargetValues(
			input,
			cues,
			relationSubjects,
			relation,
		)
		if len(values) == 0 {
			continue
		}
		allTerms, anyTerms, excludeTerms :=
			partitionSearchValues(values, cues)
		contentInput, err := newContentInput(
			allTerms,
			anyTerms,
			excludeTerms,
			selectedAsOfDate(input, cues, false),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"非 task cue の検索対象から本文検索条件を構築できません: %w",
				err,
			)
		}
		draft := newCandidateDraft()
		addExplicitSearchEvidence(&draft, cues, true)
		draft.evidence[legalquery.EvidenceGeneralTerm] =
			struct{}{}
		draft.steps = append(draft.steps, stepDraft{
			startByte: contentSubjectStartByte(firstSpan, cues),
			input:     contentInput,
		})
		result = append(result, draft)
	}
	return result, nil
}

func (p *Profile) isRelationV2DirectSearch(
	relation legalquery.CueTaskRelation,
) bool {
	if relation.Kind() != legalquery.CueTaskRelationDirectTask {
		return false
	}
	subject := relation.Subject()
	if subject.ProfileID() != p.metadata.ProfileID() {
		return false
	}
	definition, exists := p.cueByID[subject.CueID()]
	return exists &&
		definition.category == "task" &&
		definition.value == "search"
}

func (p *Profile) relationV2MentionTargetValues(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	relationSubjects map[cueRelationRefKey]struct{},
	searchRelation legalquery.CueTaskRelation,
) ([]operatedContentValue, legalquery.QuerySpan) {
	clause := searchRelation.ClauseSpan()
	predicateStart := searchRelation.Predicate().Span().StartByte()
	resources := cues.mentions[cueMeaningKey("resource", "law_provision")]
	queryTerms := input.QueryTermMentions()
	values := make([]operatedContentValue, 0)
	var firstSpan legalquery.QuerySpan
	for _, mention := range input.CueMentions() {
		if mention.ProfileID() != p.metadata.ProfileID() ||
			!relationV2SpanContains(clause, mention.Span()) {
			continue
		}
		definition, exists := p.cueByID[mention.CueID()]
		if !exists ||
			(definition.category != "unsupported" &&
				definition.category != "reserved_pack") {
			continue
		}
		if _, bound := relationSubjects[cueRelationRefKeyFromMention(mention)]; bound {
			continue
		}
		if relationV2CueOverlapsQueryTerm(mention, queryTerms) ||
			!relationV2ProvisionFollowsTarget(
				mention.Span(),
				predicateStart,
				clause,
				resources,
			) {
			continue
		}
		if len(values) == 0 {
			firstSpan = mention.Span()
		}
		values = append(values, operatedContentValue{
			value:     mention.Surface(),
			startByte: mention.Span().StartByte(),
		})
	}
	return values, firstSpan
}

func (p *Profile) relationV2CueIsContentTarget(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	mention legalquery.CueMention,
) bool {
	resources := cues.mentions[cueMeaningKey("resource", "law_provision")]
	for _, relation := range input.CueTaskRelations() {
		if !p.isRelationV2DirectSearch(relation) ||
			!relationV2SpanContains(relation.ClauseSpan(), mention.Span()) {
			continue
		}
		if relationV2ProvisionFollowsTarget(
			mention.Span(),
			relation.Predicate().Span().StartByte(),
			relation.ClauseSpan(),
			resources,
		) {
			return true
		}
	}
	return false
}

func relationV2ProvisionFollowsTarget(
	target legalquery.QuerySpan,
	predicateStart int,
	clause legalquery.QuerySpan,
	resources []legalquery.CueMention,
) bool {
	for _, resource := range resources {
		span := resource.Span()
		if relationV2SpanContains(clause, span) &&
			target.EndByte() <= span.StartByte() &&
			span.EndByte() <= predicateStart {
			return true
		}
	}
	return false
}

func relationV2CueOverlapsQueryTerm(
	cue legalquery.CueMention,
	terms []legalquery.QueryTermMention,
) bool {
	for _, term := range terms {
		if cue.Span().StartByte() < term.Span().EndByte() &&
			term.Span().StartByte() < cue.Span().EndByte() {
			return true
		}
	}
	return false
}

func relationV2CueOverlapsQuotedQueryTerm(
	cue legalquery.CueMention,
	terms []legalquery.QueryTermMention,
) bool {
	for _, term := range terms {
		if term.Kind() == legalquery.QueryTermMentionQuotedPhrase &&
			cue.Span().StartByte() < term.Span().EndByte() &&
			term.Span().StartByte() < cue.Span().EndByte() {
			return true
		}
	}
	return false
}

func relationV2SpanContains(
	container legalquery.QuerySpan,
	value legalquery.QuerySpan,
) bool {
	return container.StartByte() <= value.StartByte() &&
		value.EndByte() <= container.EndByte()
}

func (p *Profile) withoutRelationV2MentionNounDrafts(
	input legalquery.CandidateGenerationInput,
	drafts []candidateDraft,
	mentionTargets []candidateDraft,
) []candidateDraft {
	clauses := p.relationV2MentionTargetClauses(input, mentionTargets)
	result := make([]candidateDraft, 0, len(drafts))
	for _, draft := range drafts {
		if relationV2IsMentionNounDraft(draft) &&
			relationV2DraftStartsInAnyClause(draft, clauses) {
			continue
		}
		result = append(result, draft)
	}
	return result
}

func (p *Profile) relationV2MentionTargetClauses(
	input legalquery.CandidateGenerationInput,
	targets []candidateDraft,
) []legalquery.QuerySpan {
	result := make([]legalquery.QuerySpan, 0, len(targets))
	seen := make(map[[2]int]struct{})
	for _, target := range targets {
		if len(target.steps) != 1 {
			continue
		}
		for _, relation := range input.CueTaskRelations() {
			if !p.isRelationV2DirectSearch(relation) ||
				!relationV2PositionInSpan(
					target.steps[0].startByte,
					relation.ClauseSpan(),
				) {
				continue
			}
			clause := relation.ClauseSpan()
			key := [2]int{clause.StartByte(), clause.EndByte()}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, clause)
		}
	}
	return result
}

func relationV2DraftStartsInAnyClause(
	draft candidateDraft,
	clauses []legalquery.QuerySpan,
) bool {
	if len(draft.steps) != 1 {
		return false
	}
	for _, clause := range clauses {
		if relationV2PositionInSpan(draft.steps[0].startByte, clause) {
			return true
		}
	}
	return false
}

func relationV2IsMentionNounDraft(draft candidateDraft) bool {
	if len(draft.steps) != 1 {
		return false
	}
	input, ok := draft.steps[0].input.(legalquery.LawContentSearchIntentV1)
	if !ok ||
		len(input.AllTerms()) != 1 ||
		len(input.AnyTerms()) != 0 ||
		len(input.ExcludeTerms()) != 0 {
		return false
	}
	switch input.AllTerms()[0] {
	case "語", "言葉", "表現", "用語", "文字列", "文言":
		return true
	default:
		return false
	}
}
