package core

import (
	"strconv"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

const broadLegalInformationSurface = "法情報"

// selectedCoreConceptMentions は、SOT-ENG-023 の resource 範囲に従って
// 同じ表記の法概念候補を選ぶ。
func (p *Profile) selectedCoreConceptMentions(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) []legalquery.LegalConceptMention {
	values := coreConceptMentionsForResources(input, cues)

	ambiguousGroups := make(map[string]struct{})
	for _, mention := range values {
		definition, exists := p.concepts[mention.ConceptID()]
		if !usesBroadLegalInformationScope(input, cues, mention) ||
			!exists ||
			definition.entry.ConflictGroupID == "" ||
			definition.entry.SelectionPolicy !=
				legalconceptlexicon.SelectionPolicyAmbiguousNoAutoExecute ||
			conceptResourceCount(definition.entry) < 2 {
			continue
		}
		ambiguousGroups[conceptMentionGroupKey(
			mention,
			definition.entry.ConflictGroupID,
		)] = struct{}{}
	}

	result := make([]legalquery.LegalConceptMention, 0, len(values))
	for _, mention := range values {
		definition, exists := p.concepts[mention.ConceptID()]
		if !exists || definition.entry.ConflictGroupID == "" {
			result = append(result, mention)
			continue
		}
		_, ambiguous := ambiguousGroups[conceptMentionGroupKey(
			mention,
			definition.entry.ConflictGroupID,
		)]
		if ambiguous &&
			definition.entry.SelectionPolicy !=
				legalconceptlexicon.SelectionPolicyAmbiguousNoAutoExecute {
			continue
		}
		result = append(result, mention)
	}
	return result
}

func usesBroadLegalInformationScope(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	mention legalquery.LegalConceptMention,
) bool {
	for _, resource := range cues.mentions[cueMeaningKey("resource", "law")] {
		if resource.Surface() == broadLegalInformationSurface &&
			conceptResourceAssociated(
				input,
				cues,
				mention.Span(),
				resource.Span(),
			) {
			return usesUnresolvedConceptResourceScope(input, cues, mention)
		}
	}
	return false
}

func usesUnresolvedConceptResourceScope(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	mention legalquery.LegalConceptMention,
) bool {
	for _, resource := range contentCoreResources(cues) {
		if conceptResourceAssociated(
			input,
			cues,
			mention.Span(),
			resource.Span(),
		) {
			return false
		}
	}
	for _, resource := range contentJudicialResources(cues) {
		if conceptResourceAssociated(
			input,
			cues,
			mention.Span(),
			resource.Span(),
		) {
			return false
		}
	}
	return true
}

func conceptResourceAssociated(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	subject legalquery.QuerySpan,
	resource legalquery.QuerySpan,
) bool {
	_, associated := contentSubjectResourceDistance(subject, resource)
	if !associated {
		return false
	}
	startByte := subject.EndByte()
	endByte := resource.StartByte()
	if resource.EndByte() <= subject.StartByte() {
		startByte = resource.EndByte()
		endByte = subject.StartByte()
	}
	return !hasInterveningRefSubject(
		input,
		cues,
		startByte,
		endByte,
	)
}

func conceptMentionGroupKey(
	mention legalquery.LegalConceptMention,
	conflictGroupID string,
) string {
	span := mention.Span()
	return strconv.Itoa(span.StartByte()) + ":" +
		strconv.Itoa(span.EndByte()) + ":" +
		conflictGroupID
}

func conceptResourceCount(entry legalconceptlexicon.Entry) int {
	resources := make(map[legalquery.Resource]struct{})
	for _, candidate := range entry.Candidates {
		resources[candidate.Resource] = struct{}{}
	}
	return len(resources)
}
