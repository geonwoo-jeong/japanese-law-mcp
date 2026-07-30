package core

import (
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func (p *Profile) retainRelationV2SupportedDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	drafts []candidateDraft,
	signals []legalquery.CandidateGenerationSignal,
) []candidateDraft {
	if !hasUnsupportedTaskOrResourceSignal(signals) {
		return retainSupportedDraftsForUnsupportedRequest(
			drafts,
			cues,
			signals,
		)
	}
	result := make([]candidateDraft, 0, len(drafts))
	for _, draft := range drafts {
		if hasDraftEvidence(
			draft,
			legalquery.EvidenceUniqueTypoCorrection,
		) || !p.relationV2DraftHasStrongScopedSteps(input, cues, draft) {
			continue
		}
		if p.relationV2HasUnrepresentedSeparatedSubject(
			input,
			cues,
			draft,
		) {
			continue
		}
		result = append(result, draft)
	}
	return result
}

func (p *Profile) relationV2DraftHasStrongScopedSteps(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
) bool {
	if len(draft.steps) == 0 {
		return false
	}
	for _, step := range draft.steps {
		if !p.relationV2StepHasStrongScope(input, cues, draft, step) {
			return false
		}
	}
	return true
}

func (p *Profile) relationV2StepHasStrongScope(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	step stepDraft,
) bool {
	for _, relation := range input.CueTaskRelations() {
		if !p.relationV2RelationSupportsStep(relation, step) ||
			!relationV2PositionInSpan(
				step.startByte,
				relation.ClauseSpan(),
			) {
			continue
		}
		if p.relationV2StepHasStrongAnchor(
			input,
			cues,
			draft,
			step,
			relation.ClauseSpan(),
		) {
			return true
		}
	}
	return false
}

func (p *Profile) relationV2RelationSupportsStep(
	relation legalquery.CueTaskRelation,
	step stepDraft,
) bool {
	subject := relation.Subject()
	if subject.ProfileID() != p.metadata.ProfileID() {
		return false
	}
	definition, exists := p.cueByID[subject.CueID()]
	if !exists {
		return false
	}
	if definition.category == "unsupported" &&
		definition.value == "task_or_resource" {
		return true
	}
	return relation.Kind() == legalquery.CueTaskRelationDirectTask &&
		definition.category == "task" &&
		definition.value == relationV2TaskValue(step.input)
}

func relationV2TaskValue(input legalquery.LogicalInput) string {
	switch input.InputKind() {
	case legalquery.InputKindLawSearch,
		legalquery.InputKindLawContentSearch:
		return "search"
	case legalquery.InputKindLawRead,
		legalquery.InputKindLawArticleRead:
		return "read"
	case legalquery.InputKindLawUpdates:
		return "list_updates"
	default:
		return ""
	}
}

func (p *Profile) relationV2StepHasStrongAnchor(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	step stepDraft,
	clause legalquery.QuerySpan,
) bool {
	if hasDraftEvidence(draft, legalquery.EvidenceOfficialIdentifier) &&
		relationV2StepUsesRef(step.input) {
		return true
	}
	if hasDraftEvidence(draft, legalquery.EvidenceExplicitResource) &&
		relationV2HasStepResourceCue(cues, step.input, clause) {
		return true
	}
	if hasDraftEvidence(draft, legalquery.EvidenceOfficialIdentifier) {
		for _, mention := range input.IdentifierMentions() {
			if mention.Kind() != legalquery.IdentifierMentionLawNumber &&
				relationV2MentionStartsStep(
					mention.Span(),
					step,
					clause,
				) {
				return true
			}
		}
	}
	if hasDraftEvidence(draft, legalquery.EvidenceStructuredReference) &&
		relationV2StepHasStructuredAnchor(input, step, clause) {
		return true
	}
	if hasDraftEvidence(draft, legalquery.EvidenceOfficialAlias) {
		for _, mention := range input.LawNameMentions() {
			if relationV2MentionStartsStep(
				mention.Span(),
				step,
				clause,
			) {
				return true
			}
		}
	}
	if hasDraftEvidence(draft, legalquery.EvidenceLegalConcept) {
		for _, mention := range input.LegalConceptMentions() {
			if relationV2MentionStartsStep(
				mention.Span(),
				step,
				clause,
			) {
				return true
			}
		}
	}
	return false
}

func relationV2StepHasStructuredAnchor(
	input legalquery.CandidateGenerationInput,
	step stepDraft,
	clause legalquery.QuerySpan,
) bool {
	for _, mention := range input.IdentifierMentions() {
		if mention.Kind() == legalquery.IdentifierMentionLawNumber &&
			relationV2MentionStartsStep(mention.Span(), step, clause) {
			return true
		}
	}
	for _, mention := range input.ArticleMentions() {
		if relationV2MentionStartsStep(mention.Span(), step, clause) {
			return true
		}
	}
	for _, mention := range input.ParagraphMentions() {
		if relationV2MentionStartsStep(mention.Span(), step, clause) {
			return true
		}
	}
	for _, mention := range input.DateMentions() {
		if relationV2MentionStartsStep(mention.Span(), step, clause) {
			return true
		}
	}
	return false
}

func relationV2MentionStartsStep(
	span legalquery.QuerySpan,
	step stepDraft,
	clause legalquery.QuerySpan,
) bool {
	return span.StartByte() == step.startByte &&
		relationV2SpanContains(clause, span)
}

func relationV2HasStepResourceCue(
	cues resolvedCues,
	input legalquery.LogicalInput,
	clause legalquery.QuerySpan,
) bool {
	values := relationV2StepResourceValues(input)
	for _, value := range values {
		for _, mention := range cues.mentions[cueMeaningKey("resource", value)] {
			if relationV2SpanContains(clause, mention.Span()) {
				return true
			}
		}
	}
	return false
}

func relationV2StepResourceValues(
	input legalquery.LogicalInput,
) []string {
	switch input.InputKind() {
	case legalquery.InputKindLawSearch,
		legalquery.InputKindLawRead:
		return []string{"law"}
	case legalquery.InputKindLawContentSearch,
		legalquery.InputKindLawArticleRead:
		return []string{"law_provision"}
	case legalquery.InputKindLawUpdates:
		return []string{"updates", "law"}
	default:
		return nil
	}
}

func relationV2StepUsesRef(input legalquery.LogicalInput) bool {
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

func (p *Profile) relationV2HasUnrepresentedSeparatedSubject(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
) bool {
	if !separatesSubjects(cues) {
		return false
	}
	taskValues := make(map[string]struct{}, len(draft.steps))
	for _, step := range draft.steps {
		taskValues[relationV2TaskValue(step.input)] = struct{}{}
	}
	clauses := make([]legalquery.QuerySpan, 0)
	for _, relation := range input.CueTaskRelations() {
		if relation.Kind() != legalquery.CueTaskRelationDirectTask {
			continue
		}
		subject := relation.Subject()
		if subject.ProfileID() != p.metadata.ProfileID() {
			continue
		}
		definition, exists := p.cueByID[subject.CueID()]
		if !exists || definition.category != "task" {
			continue
		}
		if _, represented := taskValues[definition.value]; represented {
			clauses = append(clauses, relation.ClauseSpan())
		}
	}
	if len(clauses) == 0 {
		return false
	}
	required := p.relationV2SeparatedSubjectSpans(input, cues, clauses)
	for _, span := range required {
		represented := false
		for _, step := range draft.steps {
			if step.startByte == span.StartByte() ||
				step.startByte == contentSubjectStartByte(span, cues) {
				represented = true
				break
			}
		}
		if !represented {
			return true
		}
	}
	return false
}

func (p *Profile) relationV2SeparatedSubjectSpans(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	clauses []legalquery.QuerySpan,
) []legalquery.QuerySpan {
	result := make([]legalquery.QuerySpan, 0)
	seen := make(map[[2]int]struct{})
	appendSpan := func(span legalquery.QuerySpan) {
		if !relationV2SpanInAnyClause(span, clauses) {
			return
		}
		key := [2]int{span.StartByte(), span.EndByte()}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		result = append(result, span)
	}
	for _, mention := range input.IdentifierMentions() {
		appendSpan(mention.Span())
	}
	for _, mention := range input.LawNameMentions() {
		appendSpan(mention.Span())
	}
	for _, mention := range p.selectedCoreConceptMentions(input, cues) {
		appendSpan(mention.Span())
	}
	for _, mention := range input.QueryTermMentions() {
		appendSpan(mention.Span())
	}
	return result
}

func relationV2SpanInAnyClause(
	span legalquery.QuerySpan,
	clauses []legalquery.QuerySpan,
) bool {
	for _, clause := range clauses {
		if relationV2SpanContains(clause, span) {
			return true
		}
	}
	return false
}

func relationV2PositionInSpan(
	position int,
	span legalquery.QuerySpan,
) bool {
	return span.StartByte() <= position && position < span.EndByte()
}
