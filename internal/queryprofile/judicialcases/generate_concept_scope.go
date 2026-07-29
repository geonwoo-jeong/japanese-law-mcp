package judicialcases

import (
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

func (p *Profile) buildBroadLegalInformationSearchDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, bool, bool, error) {
	if !cues.has("resource_scope", "legal_information") ||
		!cues.has("task", "search") {
		return nil, false, false, nil
	}

	mentions := p.ambiguousCrossResourceConceptMentions(
		input,
		cues,
	)
	drafts, _, err := p.buildConceptSearchSubjects(mentions, false)
	if err != nil {
		return nil, false, false, err
	}
	return drafts, false, len(drafts) > 0, nil
}

func (p *Profile) ambiguousCrossResourceConceptMentions(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) []legalquery.LegalConceptMention {
	values := input.LegalConceptMentions()
	result := make([]legalquery.LegalConceptMention, 0, len(values))
	for _, mention := range values {
		definition, exists := p.concepts[mention.ConceptID()]
		if !exists ||
			!broadLegalInformationAppliesToConcept(
				input,
				cues,
				mention,
			) ||
			definition.entry.SelectionPolicy !=
				legalconceptlexicon.SelectionPolicyAmbiguousNoAutoExecute ||
			judicialConceptCandidateCount(definition) == 0 ||
			judicialConceptResourceCount(definition) < 2 {
			continue
		}
		result = append(result, mention)
	}
	return result
}

func judicialConceptResourceCount(definition conceptDefinition) int {
	resources := make(map[legalquery.Resource]struct{})
	for _, candidate := range definition.entry.Candidates {
		resources[candidate.Resource] = struct{}{}
	}
	return len(resources)
}

func broadLegalInformationAppliesToConcept(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	mention legalquery.LegalConceptMention,
) bool {
	broadSpans := make(map[[2]int]struct{})
	broadAssociated := false
	for _, broad := range cues.mentions[cueMeaningKey(
		"resource_scope",
		"legal_information",
	)] {
		broadSpans[querySpanKey(broad.Span())] = struct{}{}
		if broadScopeResourceAssociated(
			input,
			mention.Span(),
			broad.Span(),
		) {
			broadAssociated = true
		}
	}
	if !broadAssociated {
		return false
	}
	for _, resource := range input.CueMentions() {
		if resource.ProfileID() == profileID ||
			!strings.HasPrefix(resource.CueID(), "resource-") {
			continue
		}
		if _, broad := broadSpans[querySpanKey(resource.Span())]; broad {
			continue
		}
		if broadScopeResourceAssociated(
			input,
			mention.Span(),
			resource.Span(),
		) {
			return false
		}
	}
	return true
}

func broadScopeResourceAssociated(
	input legalquery.CandidateGenerationInput,
	subject legalquery.QuerySpan,
	resource legalquery.QuerySpan,
) bool {
	_, associated := subjectResourceDistance(subject, resource)
	if !associated {
		return false
	}
	startByte := subject.EndByte()
	endByte := resource.StartByte()
	if resource.EndByte() <= subject.StartByte() {
		startByte = resource.EndByte()
		endByte = subject.StartByte()
	}
	return !hasInterveningBroadScopeSubject(
		input,
		startByte,
		endByte,
	)
}

func hasInterveningBroadScopeSubject(
	input legalquery.CandidateGenerationInput,
	startByte int,
	endByte int,
) bool {
	spans := make([]legalquery.QuerySpan, 0)
	for _, value := range input.QueryTermMentions() {
		spans = append(spans, value.Span())
	}
	for _, value := range input.LegalConceptMentions() {
		spans = append(spans, value.Span())
	}
	for _, value := range input.LawNameMentions() {
		spans = append(spans, value.Span())
	}
	for _, value := range input.IdentifierMentions() {
		spans = append(spans, value.Span())
	}
	for _, value := range input.ArticleMentions() {
		spans = append(spans, value.Span())
	}
	for _, value := range input.ParagraphMentions() {
		spans = append(spans, value.Span())
	}
	for _, value := range input.CaseNumberMentions() {
		spans = append(spans, value.Span())
	}
	for _, value := range input.DateMentions() {
		spans = append(spans, value.Span())
	}
	for _, span := range spans {
		if startByte <= span.StartByte() &&
			span.EndByte() <= endByte {
			return true
		}
	}
	return false
}
