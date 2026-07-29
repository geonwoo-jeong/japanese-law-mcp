package core

import (
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func queryTermsForCoreResources(
	input legalquery.CandidateGenerationInput,
	terms []legalquery.QueryTermMention,
	cues resolvedCues,
) []legalquery.QueryTermMention {
	if !cues.has("reserved_pack", "judicial-cases") {
		return append([]legalquery.QueryTermMention(nil), terms...)
	}
	resources := contentCoreResources(cues)
	if len(resources) == 0 {
		return nil
	}
	selected := make(map[[2]int]struct{})
	assigned := make(map[[2]int]struct{})
	for _, term := range terms {
		resourceKey, core, unique := nearestUninterruptedContentResource(
			input,
			term.Span(),
			cues,
		)
		if !unique || !core {
			continue
		}
		selected[contentSpanKey(term.Span())] = struct{}{}
		assigned[resourceKey] = struct{}{}
	}
	for key := range coreResourceKeysForConcepts(
		input,
		input.LegalConceptMentions(),
		cues,
	) {
		assigned[key] = struct{}{}
	}
	for _, resource := range resources {
		key := contentSpanKey(resource.Span())
		if _, exists := assigned[key]; exists {
			continue
		}
		if span, found := nearestSharedQueryTerm(
			input,
			terms,
			resource.Span(),
			cues,
		); found {
			selected[span] = struct{}{}
		}
	}
	result := make([]legalquery.QueryTermMention, 0, len(selected))
	for _, term := range terms {
		if _, exists := selected[contentSpanKey(term.Span())]; exists {
			result = append(result, term)
		}
	}
	return result
}

func coreConceptMentionsForResources(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) []legalquery.LegalConceptMention {
	mentions := input.LegalConceptMentions()
	if !cues.has("reserved_pack", "judicial-cases") {
		return append([]legalquery.LegalConceptMention(nil), mentions...)
	}
	resources := contentCoreResources(cues)
	if len(resources) == 0 {
		return nil
	}
	selected := make(map[[2]int]struct{})
	assigned := make(map[[2]int]struct{})
	for _, mention := range mentions {
		resourceKey, core, unique := nearestUninterruptedContentResource(
			input,
			mention.Span(),
			cues,
		)
		if !unique || !core {
			continue
		}
		selected[contentSpanKey(mention.Span())] = struct{}{}
		assigned[resourceKey] = struct{}{}
	}
	for key := range coreResourceKeysForTerms(
		input,
		input.QueryTermMentions(),
		cues,
	) {
		assigned[key] = struct{}{}
	}
	for _, resource := range resources {
		key := contentSpanKey(resource.Span())
		if _, exists := assigned[key]; exists {
			continue
		}
		if span, found := nearestSharedLegalConcept(
			input,
			mentions,
			resource.Span(),
			cues,
		); found {
			selected[span] = struct{}{}
		}
	}
	result := make([]legalquery.LegalConceptMention, 0, len(selected))
	for _, mention := range mentions {
		if _, exists := selected[contentSpanKey(mention.Span())]; exists {
			result = append(result, mention)
		}
	}
	return result
}

func coreResourceKeysForConcepts(
	input legalquery.CandidateGenerationInput,
	mentions []legalquery.LegalConceptMention,
	cues resolvedCues,
) map[[2]int]struct{} {
	result := make(map[[2]int]struct{})
	for _, mention := range mentions {
		resourceKey, core, unique := nearestUninterruptedContentResource(
			input,
			mention.Span(),
			cues,
		)
		if unique && core {
			result[resourceKey] = struct{}{}
		}
	}
	return result
}

func coreResourceKeysForTerms(
	input legalquery.CandidateGenerationInput,
	terms []legalquery.QueryTermMention,
	cues resolvedCues,
) map[[2]int]struct{} {
	result := make(map[[2]int]struct{})
	for _, term := range terms {
		resourceKey, core, unique := nearestUninterruptedContentResource(
			input,
			term.Span(),
			cues,
		)
		if unique && core {
			result[resourceKey] = struct{}{}
		}
	}
	return result
}

func nearestSharedQueryTerm(
	input legalquery.CandidateGenerationInput,
	terms []legalquery.QueryTermMention,
	resource legalquery.QuerySpan,
	cues resolvedCues,
) ([2]int, bool) {
	bestDistance := -1
	bestSpan := [2]int{}
	tied := false
	for _, term := range terms {
		_, core, unique := nearestUninterruptedContentResource(
			input,
			term.Span(),
			cues,
		)
		if !unique || core {
			continue
		}
		if hasInterveningContentSubject(
			input,
			term.Span(),
			contentSpanKey(resource),
		) && !groupsContentSubjects(cues) {
			continue
		}
		distance, associated := contentSubjectResourceDistance(
			term.Span(),
			resource,
		)
		if !associated {
			continue
		}
		span := contentSpanKey(term.Span())
		switch {
		case bestDistance < 0 || distance < bestDistance:
			bestDistance = distance
			bestSpan = span
			tied = false
		case distance == bestDistance && span != bestSpan:
			tied = true
		}
	}
	return bestSpan, bestDistance >= 0 && !tied
}

func nearestSharedLegalConcept(
	input legalquery.CandidateGenerationInput,
	mentions []legalquery.LegalConceptMention,
	resource legalquery.QuerySpan,
	cues resolvedCues,
) ([2]int, bool) {
	bestDistance := -1
	bestSpan := [2]int{}
	tied := false
	for _, mention := range mentions {
		_, core, unique := nearestUninterruptedContentResource(
			input,
			mention.Span(),
			cues,
		)
		if !unique || core {
			continue
		}
		if hasInterveningContentSubject(
			input,
			mention.Span(),
			contentSpanKey(resource),
		) && !groupsContentSubjects(cues) {
			continue
		}
		distance, associated := contentSubjectResourceDistance(
			mention.Span(),
			resource,
		)
		if !associated {
			continue
		}
		span := contentSpanKey(mention.Span())
		switch {
		case bestDistance < 0 || distance < bestDistance:
			bestDistance = distance
			bestSpan = span
			tied = false
		case distance == bestDistance && span != bestSpan:
			tied = true
		}
	}
	return bestSpan, bestDistance >= 0 && !tied
}

func nearestUninterruptedContentResource(
	input legalquery.CandidateGenerationInput,
	subject legalquery.QuerySpan,
	cues resolvedCues,
) ([2]int, bool, bool) {
	resourceKey, core, unique := nearestContentResource(subject, cues)
	println(
		"nearest-resource",
		subject.StartByte(),
		subject.EndByte(),
		resourceKey[0],
		resourceKey[1],
		core,
		unique,
	)
	if !unique ||
		precedesLeadingRefContentCluster(input, subject, resourceKey, cues) ||
		hasInterveningContentSubject(input, subject, resourceKey) &&
			!groupsContentSubjects(cues) {
		return [2]int{}, false, false
	}
	return resourceKey, core, true
}

func groupsContentSubjects(cues resolvedCues) bool {
	return cues.has("operator", "all") ||
		cues.has("operator", "any") ||
		cues.has("operator", "exclude")
}

func precedesLeadingRefContentCluster(
	input legalquery.CandidateGenerationInput,
	subject legalquery.QuerySpan,
	resourceKey [2]int,
	cues resolvedCues,
) bool {
	ref, exists := input.Ref()
	if !exists || ref.Key().ResourceType() != "law" {
		return false
	}
	resources := contentCoreResources(cues)
	if len(resources) == 0 {
		return false
	}
	leading := resources[0].Span()
	if contentSpanKey(leading) != resourceKey {
		return false
	}
	return subject.EndByte() <= leading.StartByte()
}

func hasInterveningContentSubject(
	input legalquery.CandidateGenerationInput,
	subject legalquery.QuerySpan,
	resource [2]int,
) bool {
	for _, term := range input.QueryTermMentions() {
		if contentSpanIntervenes(subject, resource, term.Span()) {
			println(
				"intervening-term",
				subject.StartByte(),
				subject.EndByte(),
				resource[0],
				resource[1],
				term.Surface(),
				term.Span().StartByte(),
				term.Span().EndByte(),
			)
			return true
		}
	}
	for _, concept := range input.LegalConceptMentions() {
		if contentSpanIntervenes(subject, resource, concept.Span()) {
			println(
				"intervening-concept",
				subject.StartByte(),
				subject.EndByte(),
				resource[0],
				resource[1],
				concept.Surface(),
				concept.Span().StartByte(),
				concept.Span().EndByte(),
			)
			return true
		}
	}
	return false
}

func contentSpanIntervenes(
	subject legalquery.QuerySpan,
	resource [2]int,
	other legalquery.QuerySpan,
) bool {
	switch {
	case subject.EndByte() <= resource[0]:
		return subject.EndByte() <= other.StartByte() &&
			other.EndByte() <= resource[0]
	case resource[1] <= subject.StartByte():
		return resource[1] <= other.StartByte() &&
			other.EndByte() <= subject.StartByte()
	default:
		return false
	}
}

func nearestContentResource(
	subject legalquery.QuerySpan,
	cues resolvedCues,
) ([2]int, bool, bool) {
	bestDistance := -1
	bestSpan := [2]int{}
	bestCore := false
	bestFollowsSubject := false
	tied := false
	consider := func(resource legalquery.QuerySpan, core bool) {
		distance, associated := contentSubjectResourceDistance(
			subject,
			resource,
		)
		if !associated {
			return
		}
		span := contentSpanKey(resource)
		followsSubject := subject.EndByte() <= resource.StartByte()
		switch {
		case bestDistance < 0 || distance < bestDistance:
			bestDistance = distance
			bestSpan = span
			bestCore = core
			bestFollowsSubject = followsSubject
			tied = false
		case distance == bestDistance &&
			followsSubject != bestFollowsSubject:
			if followsSubject {
				bestSpan = span
				bestCore = core
				bestFollowsSubject = true
			}
			tied = false
		case distance == bestDistance && span != bestSpan:
			tied = true
		}
	}
	for _, resource := range contentCoreResources(cues) {
		consider(resource.Span(), true)
	}
	for _, resource := range contentJudicialResources(cues) {
		consider(resource.Span(), false)
	}
	return bestSpan, bestCore, bestDistance >= 0 && !tied
}

func contentCoreResources(
	cues resolvedCues,
) []legalquery.CueMention {
	values := cues.mentions[cueMeaningKey("resource", "law_provision")]
	result := make([]legalquery.CueMention, 0, len(values))
	for _, resource := range values {
		if !overlapsReservedJudicialCue(resource, cues) {
			result = append(result, resource)
		}
	}
	return collapseOverlappingResourceMentions(result)
}

func contentJudicialResources(
	cues resolvedCues,
) []legalquery.CueMention {
	return collapseOverlappingResourceMentions(
		cues.mentions[cueMeaningKey(
			"reserved_pack",
			"judicial-cases",
		)],
	)
}

func collapseOverlappingResourceMentions(
	values []legalquery.CueMention,
) []legalquery.CueMention {
	ordered := append([]legalquery.CueMention(nil), values...)
	sort.SliceStable(ordered, func(left int, right int) bool {
		leftSpan := ordered[left].Span()
		rightSpan := ordered[right].Span()
		if leftSpan.StartByte() != rightSpan.StartByte() {
			return leftSpan.StartByte() < rightSpan.StartByte()
		}
		return leftSpan.EndByte() > rightSpan.EndByte()
	})
	result := make([]legalquery.CueMention, 0, len(ordered))
	for _, current := range ordered {
		if len(result) == 0 {
			result = append(result, current)
			continue
		}
		last := result[len(result)-1]
		if current.Span().StartByte() < last.Span().EndByte() {
			if current.Span().EndByte()-current.Span().StartByte() >
				last.Span().EndByte()-last.Span().StartByte() {
				result[len(result)-1] = current
			}
			continue
		}
		result = append(result, current)
	}
	return result
}

func contentSubjectResourceDistance(
	subject legalquery.QuerySpan,
	resource legalquery.QuerySpan,
) (int, bool) {
	switch {
	case subject.EndByte() <= resource.StartByte():
		return resource.StartByte() - subject.EndByte(), true
	case resource.EndByte() <= subject.StartByte():
		return subject.StartByte() - resource.EndByte(), true
	default:
		return 0, false
	}
}

func contentSubjectStartByte(
	subject legalquery.QuerySpan,
	cues resolvedCues,
) int {
	_, core, unique := nearestContentResource(subject, cues)
	if !unique || core {
		return subject.StartByte()
	}
	bestDistance := -1
	bestStart := subject.StartByte()
	tied := false
	for _, resource := range contentCoreResources(cues) {
		distance, associated := contentSubjectResourceDistance(
			subject,
			resource.Span(),
		)
		if !associated {
			continue
		}
		switch {
		case bestDistance < 0 || distance < bestDistance:
			bestDistance = distance
			bestStart = resource.Span().StartByte()
			tied = false
		case distance == bestDistance &&
			resource.Span().StartByte() != bestStart:
			tied = true
		}
	}
	if bestDistance >= 0 && !tied {
		return bestStart
	}
	return subject.StartByte()
}

func contentSpanKey(span legalquery.QuerySpan) [2]int {
	return [2]int{
		span.StartByte(),
		span.EndByte(),
	}
}

func overlapsReservedJudicialCue(
	resource legalquery.CueMention,
	cues resolvedCues,
) bool {
	for _, judicial := range contentJudicialResources(cues) {
		if resource.Span().StartByte() < judicial.Span().EndByte() &&
			judicial.Span().StartByte() < resource.Span().EndByte() {
			return true
		}
	}
	return false
}
