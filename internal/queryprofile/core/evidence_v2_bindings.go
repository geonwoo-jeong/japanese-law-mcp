package core

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

func withCoreEvidenceBindings(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	values []candidateDraft,
) ([]candidateDraft, error) {
	result := make([]candidateDraft, 0, len(values))
	for _, value := range values {
		current := cloneDraft(value)
		if err := populateCoreDraftBindings(input, cues, &current); err != nil {
			return nil, err
		}
		result = append(result, current)
	}
	return result, nil
}

func populateCoreDraftBindings(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft *candidateDraft,
) error {
	for index := range draft.steps {
		step := &draft.steps[index]
		if coreStepUsesRef(step.input) && step.startByte == 0 {
			if startByte, exists := coreTaskCueStartByte(
				cues,
				step.input.InputKind(),
			); exists {
				step.startByte = startByte
			}
		}
	}
	order := make([]int, len(draft.steps))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(left int, right int) bool {
		return draft.steps[order[left]].startByte <
			draft.steps[order[right]].startByte
	})
	topicOrdinals := make(map[int]int, len(order))
	if !separatesSubjects(cues) {
		for _, index := range order {
			topicOrdinals[index] = 1
		}
	} else {
		ordinal := 0
		previousStart := -1
		for _, index := range order {
			currentStart := draft.steps[index].startByte
			if ordinal == 0 || currentStart != previousStart {
				ordinal++
				previousStart = currentStart
			}
			topicOrdinals[index] = ordinal
		}
	}

	for index := range draft.steps {
		step := &draft.steps[index]
		if step.topicOrdinal == 0 {
			step.topicOrdinal = topicOrdinals[index]
		}
		if len(step.evidenceBindings) > 0 {
			continue
		}
		bindings, err := coreBindingsForStep(input, cues, *draft, *step)
		if err != nil {
			return err
		}
		step.evidenceBindings = bindings
	}
	return nil
}

func coreStepUsesRef(input legalquery.LogicalInput) bool {
	switch value := input.(type) {
	case legalquery.LawReadIntentV1:
		_, exists := value.Ref()
		return exists
	case legalquery.LawArticleReadIntentV1:
		_, exists := value.Ref()
		return exists
	default:
		return false
	}
}

func coreTaskCueStartByte(
	cues resolvedCues,
	kind legalquery.LogicalInputKind,
) (int, bool) {
	taskValue := "read"
	switch kind {
	case legalquery.InputKindLawUpdates:
		taskValue = "list_updates"
	case legalquery.InputKindLawSearch,
		legalquery.InputKindLawContentSearch:
		taskValue = "search"
	}
	mentions := cues.mentions[cueMeaningKey("task", taskValue)]
	if len(mentions) == 0 {
		return 0, false
	}
	return mentions[0].Span().StartByte(), true
}

func coreBindingsForStep(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	step stepDraft,
) ([]profileevidence.EvidenceValues, error) {
	targets, err := coreStepTargetBindings(input, cues, draft, step)
	if err != nil {
		return nil, err
	}
	result := coreStepCueBindings(
		input,
		cues,
		step.input.InputKind(),
		coreEvidenceFactSpans(input, targets),
		coreStepUsesRef(step.input),
	)
	result = append(result, targets...)
	return dedupeCoreEvidenceValues(result), nil
}

func coreStepCueBindings(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	kind legalquery.LogicalInputKind,
	targetSpans []legalquery.QuerySpan,
	requireCompatibleResources bool,
) []profileevidence.EvidenceValues {
	var result []profileevidence.EvidenceValues
	taskValue := "search"
	switch kind {
	case legalquery.InputKindLawRead, legalquery.InputKindLawArticleRead:
		taskValue = "read"
	case legalquery.InputKindLawUpdates:
		taskValue = "list_updates"
	}
	scope, scopeExists := coreTaskScopeFor(
		input,
		cues,
		taskValue,
		targetSpans,
	)
	if scopeExists {
		if requireCompatibleResources &&
			!coreResourceCuesCompatible(
				cues,
				scope.relation.ClauseSpan(),
				coreResourceMeanings(kind),
			) {
			return nil
		}
		result = append(result, profileevidence.EvidenceValues{
			FactID:      scope.taskFactID,
			Layer:       profileevidence.LayerExplicitTaskResource,
			Code:        legalquery.EvidenceExplicitTask,
			ClusterSpan: true,
		})
	}
	if !scopeExists {
		return result
	}
	for _, resource := range coreResourceMeanings(kind) {
		for _, factID := range coreResourceFactIDsInClause(
			input,
			cues,
			resource,
			scope.relation.ClauseSpan(),
		) {
			result = append(result, profileevidence.EvidenceValues{
				FactID: factID,
				Layer:  profileevidence.LayerExplicitTaskResource,
				Code:   legalquery.EvidenceExplicitResource,
			})
		}
	}
	return result
}

func coreResourceCuesCompatible(
	cues resolvedCues,
	clause legalquery.QuerySpan,
	allowed []string,
) bool {
	meanings := make([]string, 0, len(cues.mentions))
	for meaning := range cues.mentions {
		meanings = append(meanings, meaning)
	}
	sort.Strings(meanings)
	for _, meaning := range meanings {
		category, value, valid := strings.Cut(meaning, "\x00")
		if !valid || category != "resource" {
			continue
		}
		for _, mention := range cues.mentions[meaning] {
			if coreSpanContains(clause, mention.Span()) &&
				!slices.Contains(allowed, value) {
				return false
			}
		}
	}
	return true
}

func coreResourceMeanings(kind legalquery.LogicalInputKind) []string {
	switch kind {
	case legalquery.InputKindLawSearch:
		return []string{"law"}
	case legalquery.InputKindLawContentSearch:
		return []string{"law_provision"}
	case legalquery.InputKindLawRead:
		return []string{"law"}
	case legalquery.InputKindLawArticleRead:
		return []string{"law_provision"}
	case legalquery.InputKindLawUpdates:
		return []string{"updates"}
	default:
		return nil
	}
}

func coreStepTargetBindings(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	step stepDraft,
) ([]profileevidence.EvidenceValues, error) {
	switch value := step.input.(type) {
	case legalquery.LawSearchIntentV1:
		return coreLawSearchBindings(input, cues, draft, step, value), nil
	case legalquery.LawContentSearchIntentV1:
		return coreContentSearchBindings(input, cues, draft, step, value), nil
	case legalquery.LawReadIntentV1:
		return coreLawReadBindings(input, value), nil
	case legalquery.LawArticleReadIntentV1:
		return coreArticleReadBindings(input, value), nil
	case legalquery.LawUpdateListIntentV1:
		return coreUpdateBindings(input, value), nil
	default:
		return nil, fmt.Errorf("core evidence mapping が未対応の input kind を受け取りました")
	}
}

func coreLawSearchBindings(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	step stepDraft,
	intent legalquery.LawSearchIntentV1,
) []profileevidence.EvidenceValues {
	var result []profileevidence.EvidenceValues
	for index, mention := range input.IdentifierMentions() {
		if !identifierMatchesSearch(mention, intent.Query()) {
			continue
		}
		code := legalquery.EvidenceOfficialIdentifier
		if mention.Kind() == legalquery.IdentifierMentionLawNumber {
			code = legalquery.EvidenceStructuredReference
		}
		result = append(result, targetEvidence(
			fmt.Sprintf("identifier-%d", index+1),
			code,
			true,
		))
	}
	for index, mention := range input.LawNameMentions() {
		if searchTermForLawMention(mention) != intent.Query() {
			continue
		}
		result = append(result, lawNameEvidence(index, mention, true)...)
	}
	for index, mention := range input.QueryTermMentions() {
		if mention.Surface() != intent.Query() {
			continue
		}
		result = append(result, queryTermEvidence(index, mention, true))
	}
	result = restrictCoreRepeatedTargetBindings(
		input,
		cues,
		draft,
		step,
		result,
	)
	if date, exists := intent.AsOf(); exists {
		result = append(result, coreAsOfBindings(input, date)...)
	}
	return result
}

func identifierMatchesSearch(
	mention legalquery.IdentifierMention,
	query string,
) bool {
	if mention.Surface() == query || mention.LawID() == query {
		return true
	}
	if revisionID, exists := mention.RevisionID(); exists && revisionID == query {
		return true
	}
	if lawNumber, exists := mention.LawNumber(); exists && lawNumber == query {
		return true
	}
	return false
}

func coreContentSearchBindings(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	step stepDraft,
	intent legalquery.LawContentSearchIntentV1,
) []profileevidence.EvidenceValues {
	terms := append(intent.AllTerms(), intent.AnyTerms()...)
	terms = append(terms, intent.ExcludeTerms()...)
	var result []profileevidence.EvidenceValues
	for index, mention := range input.QueryTermMentions() {
		if slices.Contains(terms, mention.Surface()) {
			result = append(result, queryTermEvidence(index, mention, true))
		}
	}
	conceptIDs := make(map[string]struct{}, len(draft.concepts))
	for _, source := range draft.concepts {
		conceptIDs[source.ConceptID()] = struct{}{}
	}
	for index, mention := range input.LegalConceptMentions() {
		if _, exists := conceptIDs[mention.ConceptID()]; !exists {
			continue
		}
		if len(draft.steps) > 1 &&
			mention.Span().StartByte() != step.startByte {
			continue
		}
		result = append(result, conceptEvidence(index, mention, true)...)
	}
	result = restrictCoreRepeatedTargetBindings(
		input,
		cues,
		draft,
		step,
		result,
	)
	if date, exists := intent.AsOf(); exists {
		result = append(result, coreAsOfBindings(input, date)...)
	}
	return result
}

func coreLawReadBindings(
	input legalquery.CandidateGenerationInput,
	intent legalquery.LawReadIntentV1,
) []profileevidence.EvidenceValues {
	if _, exists := intent.Ref(); exists {
		if _, hasRef := input.Ref(); hasRef {
			return []profileevidence.EvidenceValues{{
				FactID:              "input-ref",
				Layer:               profileevidence.LayerBoundary,
				Code:                legalquery.EvidenceOfficialIdentifier,
				IndependentPositive: true,
			}}
		}
		return nil
	}
	lawID, _ := intent.LawID()
	revisionID, _ := intent.RevisionID()
	result := coreLawIdentityBindings(input, lawID, revisionID)
	if date, exists := intent.AsOf(); exists {
		result = append(result, coreAsOfBindings(input, date)...)
	}
	return result
}

func coreArticleReadBindings(
	input legalquery.CandidateGenerationInput,
	intent legalquery.LawArticleReadIntentV1,
) []profileevidence.EvidenceValues {
	var result []profileevidence.EvidenceValues
	if _, exists := intent.Ref(); exists {
		if _, hasRef := input.Ref(); hasRef {
			result = append(result, profileevidence.EvidenceValues{
				FactID:              "input-ref",
				Layer:               profileevidence.LayerBoundary,
				Code:                legalquery.EvidenceOfficialIdentifier,
				IndependentPositive: true,
			})
		}
	} else {
		lawID, _ := intent.LawID()
		result = append(result, coreLawIdentityBindings(input, lawID, "")...)
	}
	location := intent.Location()
	for index, mention := range input.ArticleMentions() {
		if mention.Provision() == location.Provision() &&
			mention.ArticleNumber() == location.ArticleNumber() {
			result = append(result, targetEvidence(
				fmt.Sprintf("article-%d", index+1),
				legalquery.EvidenceStructuredReference,
				true,
			))
		}
	}
	if paragraph, exists := location.ParagraphNumber(); exists {
		for index, mention := range input.ParagraphMentions() {
			if mention.ParagraphNumber() == paragraph {
				result = append(result, targetEvidence(
					fmt.Sprintf("paragraph-%d", index+1),
					legalquery.EvidenceStructuredReference,
					true,
				))
			}
		}
	}
	if date, exists := intent.AsOf(); exists {
		result = append(result, coreAsOfBindings(input, date)...)
	}
	return result
}

func coreUpdateBindings(
	input legalquery.CandidateGenerationInput,
	intent legalquery.LawUpdateListIntentV1,
) []profileevidence.EvidenceValues {
	var result []profileevidence.EvidenceValues
	for index, mention := range input.DateMentions() {
		if mention.Date() == intent.Date() {
			result = append(result, targetEvidence(
				fmt.Sprintf("date-%d", index+1),
				legalquery.EvidenceStructuredReference,
				true,
			))
		}
	}
	if len(result) != 1 {
		return nil
	}
	return result
}

func coreLawIdentityBindings(
	input legalquery.CandidateGenerationInput,
	lawID string,
	revisionID string,
) []profileevidence.EvidenceValues {
	var result []profileevidence.EvidenceValues
	for index, mention := range input.IdentifierMentions() {
		mentionRevision, _ := mention.RevisionID()
		if mention.LawID() == lawID &&
			(revisionID == "" || mentionRevision == revisionID) {
			code := legalquery.EvidenceOfficialIdentifier
			if mention.Kind() == legalquery.IdentifierMentionLawNumber {
				code = legalquery.EvidenceStructuredReference
			}
			result = append(result, targetEvidence(
				fmt.Sprintf("identifier-%d", index+1),
				code,
				true,
			))
		}
	}
	for index, mention := range input.LawNameMentions() {
		if mention.LawID() == lawID {
			result = append(result, lawNameEvidence(index, mention, true)...)
		}
	}
	return result
}

func coreAsOfBindings(
	input legalquery.CandidateGenerationInput,
	date model.Date,
) []profileevidence.EvidenceValues {
	var result []profileevidence.EvidenceValues
	for index, mention := range input.DateMentions() {
		if mention.Date() != date {
			continue
		}
		result = append(result, profileevidence.EvidenceValues{
			FactID: fmt.Sprintf("date-%d", index+1),
			Layer:  profileevidence.LayerTargetAnchor,
			Code:   legalquery.EvidenceStructuredReference,
		})
	}
	if len(result) != 1 {
		return nil
	}
	return result
}

func restrictCoreRepeatedTargetBindings(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	step stepDraft,
	values []profileevidence.EvidenceValues,
) []profileevidence.EvidenceValues {
	if len(values) < 2 {
		return values
	}
	if len(draft.steps) > 1 {
		exact := make([]profileevidence.EvidenceValues, 0, len(values))
		for _, value := range values {
			span, exists := coreEvidenceFactSpan(input, value.FactID)
			if exists && span.StartByte() == step.startByte {
				exact = append(exact, value)
			}
		}
		if len(exact) > 0 {
			return exact
		}
	}
	clause, exists := coreTaskClauseForStart(
		input,
		cues,
		coreEvidenceTaskValue(step.input.InputKind()),
		step.startByte,
	)
	if !exists {
		if len(draft.steps) > 1 {
			return nil
		}
		return values
	}
	scoped := make([]profileevidence.EvidenceValues, 0, len(values))
	for _, value := range values {
		span, hasSpan := coreEvidenceFactSpan(input, value.FactID)
		if hasSpan && coreSpanContains(clause, span) {
			scoped = append(scoped, value)
		}
	}
	return scoped
}

func targetEvidence(
	factID string,
	code legalquery.EvidenceCode,
	positive bool,
) profileevidence.EvidenceValues {
	return profileevidence.EvidenceValues{
		FactID:              factID,
		Layer:               profileevidence.LayerTargetAnchor,
		Code:                code,
		IndependentPositive: positive,
		ClusterSpan:         true,
	}
}

func lawNameEvidence(
	index int,
	mention legalquery.LawNameMention,
	positive bool,
) []profileevidence.EvidenceValues {
	factID := fmt.Sprintf("law-name-%d", index+1)
	result := []profileevidence.EvidenceValues{targetEvidence(
		factID,
		legalquery.EvidenceOfficialAlias,
		positive,
	)}
	if mention.MatchKind() ==
		legalquery.PreprocessMatchUniqueTypoCorrection {
		result = append(result, profileevidence.EvidenceValues{
			FactID: factID,
			Layer:  profileevidence.LayerTargetAnchor,
			Code:   legalquery.EvidenceUniqueTypoCorrection,
		})
	}
	return result
}

func queryTermEvidence(
	index int,
	mention legalquery.QueryTermMention,
	positive bool,
) profileevidence.EvidenceValues {
	layer := profileevidence.LayerSemanticExpansion
	code := legalquery.EvidenceMorphologicalContext
	if mention.Kind() == legalquery.QueryTermMentionQuotedPhrase {
		layer = profileevidence.LayerTargetAnchor
		code = legalquery.EvidenceGeneralTerm
	}
	return profileevidence.EvidenceValues{
		FactID:              fmt.Sprintf("query-term-%d", index+1),
		Layer:               layer,
		Code:                code,
		IndependentPositive: positive,
		ClusterSpan:         true,
	}
}

func conceptEvidence(
	index int,
	mention legalquery.LegalConceptMention,
	positive bool,
) []profileevidence.EvidenceValues {
	factID := fmt.Sprintf("legal-concept-%d", index+1)
	result := []profileevidence.EvidenceValues{{
		FactID:              factID,
		Layer:               profileevidence.LayerSemanticExpansion,
		Code:                legalquery.EvidenceLegalConcept,
		IndependentPositive: positive,
		ClusterSpan:         true,
	}}
	if mention.MatchKind() ==
		legalquery.PreprocessMatchUniqueTypoCorrection {
		result = append(result, profileevidence.EvidenceValues{
			FactID: factID,
			Layer:  profileevidence.LayerSemanticExpansion,
			Code:   legalquery.EvidenceUniqueTypoCorrection,
		})
	}
	return result
}

func dedupeCoreEvidenceValues(
	values []profileevidence.EvidenceValues,
) []profileevidence.EvidenceValues {
	result := make([]profileevidence.EvidenceValues, 0, len(values))
	for _, value := range values {
		if slices.Contains(result, value) {
			continue
		}
		result = append(result, value)
	}
	return result
}
