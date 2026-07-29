package judicialcases

import (
	"fmt"
	"sort"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

func (p *Profile) buildSearchDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, bool, bool, error) {
	var (
		drafts    []candidateDraft
		tooMany   bool
		ambiguous bool
		err       error
	)
	switch {
	case !cues.has("resource", "judicial_decision"):
		drafts, tooMany, ambiguous, err =
			p.buildBroadLegalInformationSearchDrafts(input, cues)
	case cues.has("task", "search"):
		drafts, ambiguous, err = p.buildSearchSubjects(input, cues)
	default:
		concepts := p.judicialReadFallbackConcepts(input, cues)
		if len(concepts) == 0 {
			return nil, false, false, nil
		}
		drafts, ambiguous, err = p.buildConceptSearchSubjects(
			concepts,
			true,
		)
	}
	if err != nil {
		return nil, false, false, err
	}
	if tooMany {
		return nil, true, ambiguous, nil
	}
	if !cues.has("operator", "individual") || len(drafts) < 2 {
		return drafts, false, ambiguous, nil
	}
	if len(drafts) > 4 {
		return nil, true, ambiguous, nil
	}
	sort.SliceStable(drafts, func(left, right int) bool {
		return drafts[left].steps[0].startByte <
			drafts[right].steps[0].startByte
	})
	combined := candidateDraft{
		evidence: []legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
		},
	}
	for _, draft := range drafts {
		combined.evidence = append(combined.evidence, draft.evidence...)
		combined.concepts = append(combined.concepts, draft.concepts...)
		combined.steps = append(combined.steps, draft.steps...)
		combined.preserveMorphologicalContext =
			combined.preserveMorphologicalContext ||
				draft.preserveMorphologicalContext
	}
	return []candidateDraft{combined}, false, ambiguous, nil
}

func (p *Profile) judicialReadFallbackConcepts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) []legalquery.LegalConceptMention {
	if !cues.has("task", "read") {
		return nil
	}
	ref, exists := input.Ref()
	if !exists || ref.Key().ResourceType() != "law" {
		return nil
	}
	result := make([]legalquery.LegalConceptMention, 0)
	for _, mention := range input.LegalConceptMentions() {
		definition, registered := p.concepts[mention.ConceptID()]
		if registered &&
			definition.entry.Canonical == mention.Canonical() &&
			judicialConceptCandidateCount(definition) > 0 &&
			conceptHasJudicialReadFallbackTarget(
				mention,
				input,
				cues,
			) {
			result = append(result, mention)
		}
	}
	return result
}

func conceptHasJudicialReadFallbackTarget(
	mention legalquery.LegalConceptMention,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	for _, resource := range cues.mentions[cueMeaningKey("resource", "judicial_decision")] {
		if mention.Span().EndByte() > resource.Span().StartByte() ||
			!isNearestConceptBeforeResource(
				mention,
				input.LegalConceptMentions(),
				resource,
			) ||
			hasInterveningFallbackSubject(
				input,
				mention.Span().EndByte(),
				resource.Span().StartByte(),
			) {
			continue
		}
		for _, read := range cues.mentions[cueMeaningKey("task", "read")] {
			if resource.Span().EndByte() > read.Span().StartByte() ||
				hasInterveningFallbackSubject(
					input,
					resource.Span().EndByte(),
					read.Span().StartByte(),
				) {
				continue
			}
			return true
		}
	}
	return false
}

func isNearestConceptBeforeResource(
	mention legalquery.LegalConceptMention,
	mentions []legalquery.LegalConceptMention,
	resource legalquery.CueMention,
) bool {
	for _, other := range mentions {
		if other.Span().StartByte() > mention.Span().StartByte() &&
			other.Span().EndByte() <= resource.Span().StartByte() {
			return false
		}
	}
	return true
}

func hasInterveningFallbackSubject(
	input legalquery.CandidateGenerationInput,
	startByte int,
	endByte int,
) bool {
	for _, term := range input.QueryTermMentions() {
		start := term.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	for _, article := range input.ArticleMentions() {
		start := article.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	return false
}

func explicitJudicialSubjectSelection(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) map[[2]int]struct{} {
	if cues.has("operator", "individual") {
		foreignResources := foreignResourceSpans(input, cues)
		if len(foreignResources) == 0 {
			return nil
		}
		return individualJudicialSubjectSelection(
			judicialSubjectSpans(input),
			judicialResourceMentions(cues),
			foreignResources,
		)
	}
	subjects := judicialSubjectSpans(input)
	selected := make(map[[2]int]struct{})
	for _, resource := range judicialResourceMentions(cues) {
		selectNearestJudicialSubject(
			selected,
			subjects,
			resource.Span(),
		)
	}
	return selected
}

func foreignResourceSpans(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) []legalquery.QuerySpan {
	judicialResources := judicialResourceMentions(cues)
	result := make([]legalquery.QuerySpan, 0)
	for _, mention := range input.CueMentions() {
		if mention.ProfileID() == profileID ||
			!strings.HasPrefix(mention.CueID(), "resource-") ||
			overlapsJudicialResource(
				mention.Span(),
				judicialResources,
			) {
			continue
		}
		result = append(result, mention.Span())
	}
	return collapseJudicialResourceSpans(result)
}

func judicialResourceMentions(
	cues resolvedCues,
) []legalquery.CueMention {
	values := append(
		[]legalquery.CueMention(nil),
		cues.mentions[cueMeaningKey(
			"resource",
			"judicial_decision",
		)]...,
	)
	sort.SliceStable(values, func(left int, right int) bool {
		leftSpan := values[left].Span()
		rightSpan := values[right].Span()
		if leftSpan.StartByte() != rightSpan.StartByte() {
			return leftSpan.StartByte() < rightSpan.StartByte()
		}
		return leftSpan.EndByte() > rightSpan.EndByte()
	})
	result := make([]legalquery.CueMention, 0, len(values))
	for _, current := range values {
		if len(result) == 0 ||
			current.Span().StartByte() >=
				result[len(result)-1].Span().EndByte() {
			result = append(result, current)
			continue
		}
		last := result[len(result)-1]
		if judicialSpanWidth(current.Span()) >
			judicialSpanWidth(last.Span()) {
			result[len(result)-1] = current
		}
	}
	return result
}

func collapseJudicialResourceSpans(
	values []legalquery.QuerySpan,
) []legalquery.QuerySpan {
	ordered := append([]legalquery.QuerySpan(nil), values...)
	sort.SliceStable(ordered, func(left int, right int) bool {
		if ordered[left].StartByte() != ordered[right].StartByte() {
			return ordered[left].StartByte() <
				ordered[right].StartByte()
		}
		return ordered[left].EndByte() > ordered[right].EndByte()
	})
	result := make([]legalquery.QuerySpan, 0, len(ordered))
	for _, current := range ordered {
		if len(result) == 0 ||
			current.StartByte() >= result[len(result)-1].EndByte() {
			result = append(result, current)
			continue
		}
		if judicialSpanWidth(current) >
			judicialSpanWidth(result[len(result)-1]) {
			result[len(result)-1] = current
		}
	}
	return result
}

func judicialSpanWidth(span legalquery.QuerySpan) int {
	return span.EndByte() - span.StartByte()
}

func overlapsJudicialResource(
	span legalquery.QuerySpan,
	resources []legalquery.CueMention,
) bool {
	for _, resource := range resources {
		if span.StartByte() < resource.Span().EndByte() &&
			resource.Span().StartByte() < span.EndByte() {
			return true
		}
	}
	return false
}

func individualJudicialSubjectSelection(
	subjects []legalquery.QuerySpan,
	resources []legalquery.CueMention,
	foreignResources []legalquery.QuerySpan,
) map[[2]int]struct{} {
	selected := make(map[[2]int]struct{})
	assignedResources := make(map[[2]int]struct{})
	for _, subject := range subjects {
		resourceKey, judicial, unique := nearestResourceForSubject(
			subject,
			resources,
			foreignResources,
		)
		if !unique || !judicial {
			continue
		}
		selected[[2]int{
			subject.StartByte(),
			subject.EndByte(),
		}] = struct{}{}
		assignedResources[resourceKey] = struct{}{}
	}
	for _, resource := range resources {
		key := querySpanKey(resource.Span())
		if _, exists := assignedResources[key]; exists {
			continue
		}
		selectNearestSharedSubject(
			selected,
			subjects,
			resource.Span(),
			resources,
			foreignResources,
		)
	}
	return selected
}

func selectNearestSharedSubject(
	selected map[[2]int]struct{},
	subjects []legalquery.QuerySpan,
	resource legalquery.QuerySpan,
	judicialResources []legalquery.CueMention,
	foreignResources []legalquery.QuerySpan,
) {
	bestDistance := -1
	bestKey := [2]int{}
	tied := false
	for _, subject := range subjects {
		_, judicial, unique := nearestResourceForSubject(
			subject,
			judicialResources,
			foreignResources,
		)
		if !unique || judicial {
			continue
		}
		distance, associated := subjectResourceDistance(subject, resource)
		if !associated {
			continue
		}
		key := querySpanKey(subject)
		switch {
		case bestDistance < 0 || distance < bestDistance:
			bestDistance = distance
			bestKey = key
			tied = false
		case distance == bestDistance && key != bestKey:
			tied = true
		}
	}
	if bestDistance >= 0 && !tied {
		selected[bestKey] = struct{}{}
	}
}

func nearestResourceForSubject(
	subject legalquery.QuerySpan,
	judicialResources []legalquery.CueMention,
	foreignResources []legalquery.QuerySpan,
) ([2]int, bool, bool) {
	bestDistance := -1
	bestKey := [2]int{}
	bestJudicial := false
	bestFollowsSubject := false
	tied := false
	consider := func(resource legalquery.QuerySpan, judicial bool) {
		distance, associated := subjectResourceDistance(subject, resource)
		if !associated {
			return
		}
		key := querySpanKey(resource)
		followsSubject := subject.EndByte() <= resource.StartByte()
		switch {
		case bestDistance < 0 || distance < bestDistance:
			bestDistance = distance
			bestKey = key
			bestJudicial = judicial
			bestFollowsSubject = followsSubject
			tied = false
		case distance == bestDistance &&
			followsSubject != bestFollowsSubject:
			if followsSubject {
				bestKey = key
				bestJudicial = judicial
				bestFollowsSubject = true
			}
			tied = false
		case distance == bestDistance && key != bestKey:
			tied = true
		}
	}
	for _, resource := range judicialResources {
		consider(resource.Span(), true)
	}
	for _, resource := range foreignResources {
		consider(resource, false)
	}
	return bestKey, bestJudicial, bestDistance >= 0 && !tied
}

func querySpanKey(span legalquery.QuerySpan) [2]int {
	return [2]int{
		span.StartByte(),
		span.EndByte(),
	}
}

func selectNearestJudicialSubject(
	selected map[[2]int]struct{},
	subjects []legalquery.QuerySpan,
	resource legalquery.QuerySpan,
) {
	bestDistance := -1
	bestKey := [2]int{}
	tied := false
	for _, subject := range subjects {
		distance, associated := subjectResourceDistance(subject, resource)
		if !associated {
			continue
		}
		key := [2]int{
			subject.StartByte(),
			subject.EndByte(),
		}
		switch {
		case bestDistance < 0 || distance < bestDistance:
			bestDistance = distance
			bestKey = key
			tied = false
		case distance == bestDistance && key != bestKey:
			tied = true
		}
	}
	if bestDistance >= 0 && !tied {
		selected[bestKey] = struct{}{}
	}
}

func judicialSubjectSpans(
	input legalquery.CandidateGenerationInput,
) []legalquery.QuerySpan {
	result := make([]legalquery.QuerySpan, 0)
	for _, caseNumber := range input.CaseNumberMentions() {
		result = append(result, caseNumber.Span())
	}
	for _, term := range input.QueryTermMentions() {
		result = append(result, term.Span())
	}
	for _, date := range input.DateMentions() {
		result = append(result, date.Span())
	}
	for _, concept := range input.LegalConceptMentions() {
		result = append(result, concept.Span())
	}
	return result
}

func subjectResourceDistance(
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

func judicialSubjectSelected(
	span legalquery.QuerySpan,
	selected map[[2]int]struct{},
) bool {
	if selected == nil {
		return true
	}
	_, exists := selected[[2]int{
		span.StartByte(),
		span.EndByte(),
	}]
	return exists
}

func selectedJudicialConceptMentions(
	mentions []legalquery.LegalConceptMention,
	selected map[[2]int]struct{},
) []legalquery.LegalConceptMention {
	result := make([]legalquery.LegalConceptMention, 0, len(mentions))
	for _, mention := range mentions {
		if judicialSubjectSelected(mention.Span(), selected) {
			result = append(result, mention)
		}
	}
	return result
}

func (p *Profile) buildSearchSubjects(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, bool, error) {
	result := make([]candidateDraft, 0)
	selectedSubjects := explicitJudicialSubjectSelection(input, cues)
	caseNumbers := input.CaseNumberMentions()
	for _, caseNumber := range caseNumbers {
		if !judicialSubjectSelected(
			caseNumber.Span(),
			selectedSubjects,
		) {
			continue
		}
		searchInput, err := legalquery.NewJudicialDecisionSearchIntentV1(
			legalquery.JudicialDecisionSearchIntentV1Values{
				Query: caseNumber.SearchText(),
			},
		)
		if err != nil {
			return nil, false, err
		}
		result = append(result, candidateDraft{
			evidence: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			steps: []stepDraft{{
				startByte: caseNumber.Span().StartByte(),
				input:     searchInput,
			}},
		})
	}
	for _, term := range input.QueryTermMentions() {
		if queryTermDuplicatesCaseNumber(term, caseNumbers) ||
			queryTermBelongsToReadReference(term, input, cues) ||
			!judicialSubjectSelected(term.Span(), selectedSubjects) {
			continue
		}
		searchInput, err := legalquery.NewJudicialDecisionSearchIntentV1(
			legalquery.JudicialDecisionSearchIntentV1Values{
				Query: term.Surface(),
			},
		)
		if err != nil {
			return nil, false, err
		}
		result = append(result, candidateDraft{
			evidence: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceMorphologicalContext,
			},
			steps: []stepDraft{{
				startByte: term.Span().StartByte(),
				input:     searchInput,
			}},
		})
	}
	for _, date := range input.DateMentions() {
		if !judicialSubjectSelected(date.Span(), selectedSubjects) {
			continue
		}
		searchInput, err := legalquery.NewJudicialDecisionSearchIntentV1(
			legalquery.JudicialDecisionSearchIntentV1Values{
				Query: date.Surface(),
			},
		)
		if err != nil {
			return nil, false, err
		}
		result = append(result, candidateDraft{
			evidence: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			steps: []stepDraft{{
				startByte: date.Span().StartByte(),
				input:     searchInput,
			}},
		})
	}

	concepts, ambiguous, err := p.buildConceptSearchSubjects(
		selectedJudicialConceptMentions(
			input.LegalConceptMentions(),
			selectedSubjects,
		),
		true,
	)
	if err != nil {
		return nil, false, err
	}
	result = append(result, concepts...)
	sort.SliceStable(result, func(left, right int) bool {
		return result[left].steps[0].startByte <
			result[right].steps[0].startByte
	})
	return result, ambiguous, nil
}

func (p *Profile) buildConceptSearchSubjects(
	mentions []legalquery.LegalConceptMention,
	explicitResource bool,
) ([]candidateDraft, bool, error) {
	result := make([]candidateDraft, 0, len(mentions))
	ambiguous := false
	for _, mention := range mentions {
		definition, exists := p.concepts[mention.ConceptID()]
		if !exists {
			return nil, false, fmt.Errorf(
				"judicial-cases profile に未登録の conceptId %q があります",
				mention.ConceptID(),
			)
		}
		if mention.Canonical() != definition.entry.Canonical {
			return nil, false, fmt.Errorf(
				"conceptId %q の canonical が profile snapshot と一致しません",
				mention.ConceptID(),
			)
		}
		for _, candidate := range definition.entry.Candidates {
			if !isJudicialConceptCandidate(candidate) {
				continue
			}
			searchInput, err := legalquery.NewJudicialDecisionSearchIntentV1(
				legalquery.JudicialDecisionSearchIntentV1Values{
					Query: candidate.OfficialTerm,
				},
			)
			if err != nil {
				return nil, false, fmt.Errorf(
					"conceptId %q の裁判例検索条件を構築できません: %w",
					mention.ConceptID(),
					err,
				)
			}
			evidence := []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceLegalConcept,
			}
			if explicitResource {
				evidence = append(
					evidence,
					legalquery.EvidenceExplicitResource,
				)
			} else {
				evidence = append(
					evidence,
					legalquery.EvidenceMorphologicalContext,
				)
			}
			if mention.MatchKind() ==
				legalquery.PreprocessMatchUniqueTypoCorrection {
				evidence = append(
					evidence,
					legalquery.EvidenceUniqueTypoCorrection,
				)
			}
			result = append(result, candidateDraft{
				evidence: evidence,
				concepts: []legalquery.LegalConceptSource{
					definition.source,
				},
				preserveMorphologicalContext: !explicitResource,
				steps: []stepDraft{{
					startByte: mention.Span().StartByte(),
					input:     searchInput,
				}},
			})
			if definition.entry.SelectionPolicy ==
				legalconceptlexicon.SelectionPolicyAmbiguousNoAutoExecute &&
				judicialConceptCandidateCount(definition) > 1 {
				ambiguous = true
			}
		}
	}
	sort.SliceStable(result, func(left, right int) bool {
		return result[left].steps[0].startByte <
			result[right].steps[0].startByte
	})
	return result, ambiguous, nil
}

func judicialConceptCandidateCount(definition conceptDefinition) int {
	count := 0
	for _, candidate := range definition.entry.Candidates {
		if isJudicialConceptCandidate(candidate) {
			count++
		}
	}
	return count
}

func queryTermBelongsToReadReference(
	term legalquery.QueryTermMention,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	ref, hasRef := input.Ref()
	if !hasRef ||
		term.Kind() != legalquery.QueryTermMentionMorphologicalPhrase ||
		!cues.has("task", "read") {
		return false
	}
	if term.Surface() == "参照" &&
		judicialTermPrecedesReadWithoutSearchBetween(
			term,
			cues.mentions[cueMeaningKey("task", "read")],
			cues.mentions[cueMeaningKey("task", "search")],
		) {
		return true
	}
	if ref.Key().ResourceType() != "judicial-decision" {
		return false
	}
	span := term.Span()
	for _, resource := range cues.mentions[cueMeaningKey("resource", "judicial_decision")] {
		for _, read := range cues.mentions[cueMeaningKey("task", "read")] {
			if resource.Span().EndByte() <= span.StartByte() &&
				span.EndByte() <= read.Span().StartByte() &&
				!judicialCueStartsBetween(
					cues.mentions[cueMeaningKey("task", "search")],
					span.EndByte(),
					read.Span().StartByte(),
				) {
				return true
			}
		}
	}
	return false
}

func judicialTermPrecedesReadWithoutSearchBetween(
	term legalquery.QueryTermMention,
	readCues []legalquery.CueMention,
	searchCues []legalquery.CueMention,
) bool {
	for _, readCue := range readCues {
		if term.Span().EndByte() <= readCue.Span().StartByte() &&
			!judicialCueStartsBetween(
				searchCues,
				term.Span().EndByte(),
				readCue.Span().StartByte(),
			) {
			return true
		}
	}
	return false
}

func judicialCueStartsBetween(
	cues []legalquery.CueMention,
	startByte int,
	endByte int,
) bool {
	for _, cue := range cues {
		start := cue.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	return false
}

func queryTermDuplicatesCaseNumber(
	term legalquery.QueryTermMention,
	caseNumbers []legalquery.JudicialCaseNumberMention,
) bool {
	for _, caseNumber := range caseNumbers {
		if term.Span() == caseNumber.Span() {
			return true
		}
	}
	return false
}

func isJudicialConceptCandidate(
	candidate legalconceptlexicon.Candidate,
) bool {
	return candidate.Task == legalquery.TaskSearch &&
		candidate.Resource == legalquery.ResourceJudicialDecision &&
		candidate.InputKind == legalquery.InputKindJudicialDecisionSearch &&
		len(candidate.RequiredPacks) == 1 &&
		candidate.RequiredPacks[0] == requiredPackID
}
