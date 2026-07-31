package judicialcases

import (
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

func buildJudicialEvidenceReadDraft(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) (*candidateDraft, error) {
	if !cues.has("task", "read") {
		return nil, nil
	}
	ref, exists := input.Ref()
	if !exists || ref.Key().ResourceType() != "judicial-decision" {
		return nil, nil
	}
	if _, hasVersion := ref.Key().VersionID(); hasVersion {
		return nil, nil
	}
	relation, ok, err := singleJudicialReadRelation(input)
	if err != nil || !ok {
		return nil, err
	}
	if !judicialClauseResourceCompatible(cues, relation.ClauseSpan()) {
		return nil, nil
	}
	readInput, err := legalquery.NewJudicialDecisionReadIntentV1(
		legalquery.JudicialDecisionReadIntentV1Values{Ref: ref},
	)
	if err != nil {
		return nil, err
	}
	taskFactID := judicialCueFactID(input, relation.Subject())
	if taskFactID == "" {
		return nil, fmt.Errorf("read cue fact を特定できません")
	}
	return &candidateDraft{
		evidence: []legalquery.EvidenceCode{
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceExplicitTask,
		},
		steps: []stepDraft{{
			startByte:    relation.Subject().Span().StartByte(),
			topicOrdinal: 1,
			input:        readInput,
			evidenceCodes: []legalquery.EvidenceCode{
				legalquery.EvidenceOfficialIdentifier,
				legalquery.EvidenceExplicitTask,
			},
			evidenceBindings: []profileevidence.EvidenceValues{
				{
					FactID:              "input-ref",
					Layer:               profileevidence.LayerBoundary,
					Code:                legalquery.EvidenceOfficialIdentifier,
					IndependentPositive: true,
				},
				{
					FactID:      taskFactID,
					Layer:       profileevidence.LayerExplicitTaskResource,
					Code:        legalquery.EvidenceExplicitTask,
					ClusterSpan: true,
				},
			},
		}},
	}, nil
}

func withJudicialEvidenceBindings(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	drafts []candidateDraft,
) ([]candidateDraft, error) {
	facts, err := buildJudicialEvidenceFacts(input, cues)
	if err != nil {
		return nil, err
	}
	result := make([]candidateDraft, 0, len(drafts))
	for _, draft := range drafts {
		if len(draft.steps) == 0 {
			continue
		}
		bound := draft
		bound.steps = append([]stepDraft(nil), draft.steps...)
		for stepIndex := range bound.steps {
			bindings := append(
				[]profileevidence.EvidenceValues(nil),
				bound.steps[stepIndex].evidenceBindings...,
			)
			ok := len(bindings) > 0
			var bindErr error
			if !ok {
				bindings, ok, bindErr = bindJudicialStep(
					input,
					cues,
					bound,
					bound.steps[stepIndex],
				)
			}
			if bindErr != nil {
				return nil, bindErr
			}
			if !ok {
				bound.steps = nil
				break
			}
			bound.steps[stepIndex].evidenceBindings = bindings
			if bound.steps[stepIndex].input.InputKind() ==
				legalquery.InputKindJudicialDecisionSearch {
				if start, exists := judicialSearchBindingStart(bindings, facts); exists {
					bound.steps[stepIndex].startByte = start
				}
			}
		}
		if len(bound.steps) == 0 {
			continue
		}
		result = append(result, bound)
	}
	return result, nil
}

func bindJudicialStep(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	step stepDraft,
) ([]profileevidence.EvidenceValues, bool, error) {
	switch current := step.input.(type) {
	case legalquery.JudicialDecisionReadIntentV1:
		read, err := buildJudicialEvidenceReadDraft(input, cues)
		if err != nil || read == nil || len(read.steps) != 1 {
			return nil, false, err
		}
		if current.Ref() != read.steps[0].input.(legalquery.JudicialDecisionReadIntentV1).Ref() {
			return nil, false, nil
		}
		return append([]profileevidence.EvidenceValues(nil), read.steps[0].evidenceBindings...), true, nil
	case legalquery.JudicialDecisionSearchIntentV1:
		return bindJudicialSearchStep(input, cues, draft, step, current)
	default:
		return nil, false, fmt.Errorf("judicial evidence binding が未対応の logical input を受け取りました")
	}
}

func bindJudicialSearchStep(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	step stepDraft,
	search legalquery.JudicialDecisionSearchIntentV1,
) ([]profileevidence.EvidenceValues, bool, error) {
	target, targetSpan, ok := judicialSearchTargetBinding(
		input,
		draft,
		step,
		search.Query(),
	)
	if !ok {
		return nil, false, nil
	}
	taskFactID, resourceFactID, ok := singleJudicialSearchCueFacts(
		input,
		cues,
		targetSpan,
	)
	if !ok || resourceFactID == "" {
		return nil, false, nil
	}
	bindings := []profileevidence.EvidenceValues{
		{
			FactID:      taskFactID,
			Layer:       profileevidence.LayerExplicitTaskResource,
			Code:        legalquery.EvidenceExplicitTask,
			ClusterSpan: true,
		},
	}
	if resourceFactID != "" {
		bindings = append(bindings, profileevidence.EvidenceValues{
			FactID:      resourceFactID,
			Layer:       profileevidence.LayerExplicitTaskResource,
			Code:        legalquery.EvidenceExplicitResource,
			ClusterSpan: true,
		})
	}
	bindings = append(bindings, target...)
	return bindings, true, nil
}

func singleJudicialReadRelation(
	input legalquery.CandidateGenerationInput,
) (legalquery.CueTaskRelation, bool, error) {
	var result legalquery.CueTaskRelation
	count := 0
	for _, relation := range input.CueTaskRelations() {
		if relation.Kind() != legalquery.CueTaskRelationDirectTask ||
			relation.Subject().ProfileID() != profileID ||
			relation.Subject().CueID() != "task-read" {
			continue
		}
		result = relation
		count++
	}
	if count == 1 {
		return result, true, nil
	}
	return legalquery.CueTaskRelation{}, false, nil
}

func judicialClauseResourceCompatible(
	cues resolvedCues,
	clause legalquery.QuerySpan,
) bool {
	for _, mention := range cues.mentions[cueMeaningKey("resource", "judicial_decision")] {
		if querySpanContains(clause, mention.Span()) {
			return true
		}
	}
	for meaning, mentions := range cues.mentions {
		category, _, _ := strings.Cut(meaning, "\x00")
		if category != "resource" {
			continue
		}
		for _, mention := range mentions {
			if querySpanContains(clause, mention.Span()) {
				return false
			}
		}
	}
	return true
}

func judicialCueFactID(
	input legalquery.CandidateGenerationInput,
	ref legalquery.CueTaskRelationRef,
) string {
	for index, mention := range input.CueMentions() {
		if mention.ProfileID() == ref.ProfileID() &&
			mention.CueID() == ref.CueID() &&
			mention.Span() == ref.Span() {
			return fmt.Sprintf("cue-%03d", index+1)
		}
	}
	return ""
}

func singleJudicialSearchCueFacts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	targetSpan legalquery.QuerySpan,
) (string, string, bool) {
	var taskFactID string
	var resourceFactID string
	for _, relation := range input.CueTaskRelations() {
		if relation.Kind() != legalquery.CueTaskRelationDirectTask ||
			relation.Subject().ProfileID() != profileID ||
			relation.Subject().CueID() != "task-search" {
			continue
		}
		if !querySpanContains(relation.ClauseSpan(), targetSpan) {
			continue
		}
		if taskFactID != "" {
			return "", "", false
		}
		taskFactID = judicialCueFactID(input, relation.Subject())
		resourceFactID = judicialClauseJudicialResourceFactID(input, cues, relation.ClauseSpan())
	}
	return taskFactID, resourceFactID, taskFactID != ""
}

func judicialClauseJudicialResourceFactID(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	clause legalquery.QuerySpan,
) string {
	for _, mention := range cues.mentions[cueMeaningKey("resource", "judicial_decision")] {
		if querySpanContains(clause, mention.Span()) {
			for index, current := range input.CueMentions() {
				if current.ProfileID() == mention.ProfileID() &&
					current.CueID() == mention.CueID() &&
					current.Span() == mention.Span() {
					return fmt.Sprintf("cue-%03d", index+1)
				}
			}
		}
	}
	return ""
}

func judicialSearchTargetBinding(
	input legalquery.CandidateGenerationInput,
	draft candidateDraft,
	step stepDraft,
	query string,
) ([]profileevidence.EvidenceValues, legalquery.QuerySpan, bool) {
	for index, mention := range input.CaseNumberMentions() {
		if mention.SearchText() == query &&
			judicialMentionMatchesStep(
				mention.Span(),
				step.startByte,
				judicialCaseNumberMatchCount(input, query),
			) {
			return []profileevidence.EvidenceValues{judicialTargetEvidence(
				fmt.Sprintf("case-number-%03d", index+1),
				legalquery.EvidenceStructuredReference,
			)}, mention.Span(), true
		}
	}
	for index, mention := range input.DateMentions() {
		if mention.Surface() == query &&
			judicialMentionMatchesStep(
				mention.Span(),
				step.startByte,
				judicialDateMatchCount(input, query),
			) {
			return []profileevidence.EvidenceValues{judicialTargetEvidence(
				fmt.Sprintf("date-%03d", index+1),
				legalquery.EvidenceStructuredReference,
			)}, mention.Span(), true
		}
	}
	for index, mention := range input.QueryTermMentions() {
		if mention.Surface() != query ||
			!judicialMentionMatchesStep(
				mention.Span(),
				step.startByte,
				judicialQueryTermMatchCount(input, query),
			) {
			continue
		}
		code := legalquery.EvidenceMorphologicalContext
		if mention.Kind() == legalquery.QueryTermMentionQuotedPhrase {
			code = legalquery.EvidenceGeneralTerm
		}
		return []profileevidence.EvidenceValues{judicialTargetEvidence(
			fmt.Sprintf("query-term-%03d", index+1),
			code,
		)}, mention.Span(), true
	}
	for _, source := range draft.concepts {
		matchCount := judicialConceptMatchCount(input, source.ConceptID())
		for index, mention := range input.LegalConceptMentions() {
			if source.ConceptID() != mention.ConceptID() ||
				!judicialMentionMatchesStep(
					mention.Span(),
					step.startByte,
					matchCount,
				) {
				continue
			}
			bindings := []profileevidence.EvidenceValues{judicialTargetEvidence(
				fmt.Sprintf("legal-concept-%03d", index+1),
				legalquery.EvidenceLegalConcept,
			)}
			if mention.MatchKind() ==
				legalquery.PreprocessMatchUniqueTypoCorrection {
				bindings = append(bindings, profileevidence.EvidenceValues{
					FactID: fmt.Sprintf("legal-concept-%03d", index+1),
					Layer:  profileevidence.LayerSemanticExpansion,
					Code:   legalquery.EvidenceUniqueTypoCorrection,
				})
			}
			return bindings, mention.Span(), true
		}
	}
	return nil, legalquery.QuerySpan{}, false
}

func judicialSearchBindingStart(
	bindings []profileevidence.EvidenceValues,
	facts judicialEvidenceFactSet,
) (int, bool) {
	for _, binding := range bindings {
		if !binding.IndependentPositive ||
			(binding.Layer != profileevidence.LayerTargetAnchor &&
				binding.Layer != profileevidence.LayerSemanticExpansion) {
			continue
		}
		fact, exists := facts.byID[binding.FactID]
		if !exists || fact.values.Span == nil {
			continue
		}
		return fact.values.Span.StartByte(), true
	}
	return 0, false
}

func judicialMentionMatchesStep(
	span legalquery.QuerySpan,
	stepStart int,
	matchCount int,
) bool {
	return matchCount == 1 || span.StartByte() == stepStart
}

func judicialCaseNumberMatchCount(
	input legalquery.CandidateGenerationInput,
	query string,
) int {
	count := 0
	for _, mention := range input.CaseNumberMentions() {
		if mention.SearchText() == query {
			count++
		}
	}
	return count
}

func judicialDateMatchCount(
	input legalquery.CandidateGenerationInput,
	query string,
) int {
	count := 0
	for _, mention := range input.DateMentions() {
		if mention.Surface() == query {
			count++
		}
	}
	return count
}

func judicialQueryTermMatchCount(
	input legalquery.CandidateGenerationInput,
	query string,
) int {
	count := 0
	for _, mention := range input.QueryTermMentions() {
		if mention.Surface() == query {
			count++
		}
	}
	return count
}

func judicialConceptMatchCount(
	input legalquery.CandidateGenerationInput,
	conceptID string,
) int {
	count := 0
	for _, mention := range input.LegalConceptMentions() {
		if mention.ConceptID() == conceptID {
			count++
		}
	}
	return count
}

func judicialTargetEvidence(
	factID string,
	code legalquery.EvidenceCode,
) profileevidence.EvidenceValues {
	return profileevidence.EvidenceValues{
		FactID:              factID,
		Layer:               judicialTargetLayer(code),
		Code:                code,
		IndependentPositive: true,
		ClusterSpan:         true,
	}
}

func judicialTargetLayer(code legalquery.EvidenceCode) profileevidence.Layer {
	switch code {
	case legalquery.EvidenceLegalConcept,
		legalquery.EvidenceMorphologicalContext:
		return profileevidence.LayerSemanticExpansion
	default:
		return profileevidence.LayerTargetAnchor
	}
}

func querySpanContains(container, inner legalquery.QuerySpan) bool {
	return container.StartByte() <= inner.StartByte() &&
		inner.EndByte() <= container.EndByte()
}
