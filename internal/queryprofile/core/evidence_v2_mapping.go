package core

import (
	"fmt"
	"sort"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type coreEvidenceEvaluation struct {
	mapping profileevidence.Mapping
	drafts  []coreEvidenceDraftRef
	facts   coreEvidenceFactSet
}

type coreEvidenceDraftRef struct {
	draftID string
	stepIDs []string
}

type coreEvidenceFactKind uint8

const (
	coreEvidenceFactInputRef coreEvidenceFactKind = iota + 1
	coreEvidenceFactLawName
	coreEvidenceFactLegalConcept
	coreEvidenceFactCue
	coreEvidenceFactIdentifier
	coreEvidenceFactDate
	coreEvidenceFactArticle
	coreEvidenceFactParagraph
	coreEvidenceFactQueryTerm
)

type coreEvidenceFact struct {
	values         profileevidence.FactValues
	kind           coreEvidenceFactKind
	surface        string
	conceptID      string
	cueCategory    string
	cueValue       string
	directTask     bool
	lawRef         bool
	identifierKind legalquery.IdentifierMentionKind
	matchKind      legalquery.PreprocessMatchKind
	queryTermKind  legalquery.QueryTermMentionKind
}

type coreEvidenceFactSet struct {
	values []profileevidence.FactValues
	byID   map[string]coreEvidenceFact
}

type coreCueFactMetadata struct {
	category   string
	value      string
	directTask bool
}

func buildCoreEvidenceEvaluation(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	drafts []candidateDraft,
) (coreEvidenceEvaluation, error) {
	if err := input.Validate(); err != nil {
		return coreEvidenceEvaluation{}, fmt.Errorf(
			"core evidence input が有効ではありません: %w",
			err,
		)
	}
	if err := validateCoreDraftInputs(input, drafts); err != nil {
		return coreEvidenceEvaluation{}, err
	}
	boundDrafts, err := withCoreEvidenceBindings(input, cues, drafts)
	if err != nil {
		return coreEvidenceEvaluation{}, fmt.Errorf(
			"core evidence binding を構築できません: %w",
			err,
		)
	}
	drafts = boundDrafts
	facts, err := buildCoreEvidenceFacts(input, cues)
	if err != nil {
		return coreEvidenceEvaluation{}, err
	}

	values := profileevidence.MappingValues{
		ProfileID: profileID,
		Facts:     facts.values,
	}
	references := make([]coreEvidenceDraftRef, 0, len(drafts))
	for index, draft := range drafts {
		draftValue, reference, retained, buildErr :=
			buildCoreEvidenceDraftValue(index, draft, facts)
		if buildErr != nil {
			return coreEvidenceEvaluation{}, buildErr
		}
		if !retained {
			continue
		}
		values.Drafts = append(values.Drafts, draftValue)
		references = append(references, reference)
	}
	mapping, err := profileevidence.NewMapping(values)
	if err != nil {
		return coreEvidenceEvaluation{}, fmt.Errorf(
			"core evidence mapping を構築できません: %w",
			err,
		)
	}
	return coreEvidenceEvaluation{
		mapping: mapping,
		drafts:  references,
		facts:   facts,
	}, nil
}

func buildCoreEvidenceFacts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) (coreEvidenceFactSet, error) {
	result := coreEvidenceFactSet{
		byID: make(map[string]coreEvidenceFact),
	}
	if ref, exists := input.Ref(); exists {
		result.add(coreEvidenceFact{
			values: profileevidence.FactValues{FactID: "input-ref"},
			kind:   coreEvidenceFactInputRef,
			lawRef: ref.Key().ResourceType() == "law",
		})
	}
	for index, mention := range input.LawNameMentions() {
		fact := coreEvidenceMentionFact(
			"law-name", index, mention.Span(), coreEvidenceFactLawName,
			mention.MatchKind(),
		)
		fact.surface = strings.TrimSpace(mention.Surface())
		result.add(fact)
	}
	for index, mention := range input.LegalConceptMentions() {
		fact := coreEvidenceMentionFact(
			"legal-concept", index, mention.Span(),
			coreEvidenceFactLegalConcept, mention.MatchKind(),
		)
		fact.surface = mention.Surface()
		fact.conceptID = mention.ConceptID()
		result.add(fact)
	}
	if err := result.addCueFacts(input, cues); err != nil {
		return coreEvidenceFactSet{}, err
	}
	for index, mention := range input.IdentifierMentions() {
		fact := coreEvidenceMentionFact(
			"identifier", index, mention.Span(),
			coreEvidenceFactIdentifier, "",
		)
		fact.identifierKind = mention.Kind()
		fact.surface = mention.Surface()
		result.add(fact)
	}
	for index, mention := range input.DateMentions() {
		result.add(coreEvidenceMentionFact(
			"date", index, mention.Span(), coreEvidenceFactDate, "",
		))
	}
	for index, mention := range input.ArticleMentions() {
		result.add(coreEvidenceMentionFact(
			"article", index, mention.Span(), coreEvidenceFactArticle, "",
		))
	}
	for index, mention := range input.ParagraphMentions() {
		result.add(coreEvidenceMentionFact(
			"paragraph", index, mention.Span(), coreEvidenceFactParagraph, "",
		))
	}
	for index, mention := range input.QueryTermMentions() {
		fact := coreEvidenceMentionFact(
			"query-term", index, mention.Span(),
			coreEvidenceFactQueryTerm, "",
		)
		fact.queryTermKind = mention.Kind()
		fact.surface = mention.Surface()
		result.add(fact)
	}
	return result, nil
}

func (s *coreEvidenceFactSet) addCueFacts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) error {
	metadata, err := buildCoreCueFactMetadata(input, cues)
	if err != nil {
		return err
	}
	for index, mention := range input.CueMentions() {
		if mention.ProfileID() != profileID {
			continue
		}
		current := metadata[cueRelationRefKeyFromMention(mention)]
		fact := coreEvidenceMentionFact(
			"cue", index, mention.Span(), coreEvidenceFactCue, "",
		)
		fact.cueCategory = current.category
		fact.cueValue = current.value
		fact.directTask = current.directTask
		s.add(fact)
	}
	return nil
}

func buildCoreCueFactMetadata(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) (map[cueRelationRefKey]coreCueFactMetadata, error) {
	result := make(map[cueRelationRefKey]coreCueFactMetadata)
	directTasks := coreDirectTaskCueKeys(input)
	meanings := make([]string, 0, len(cues.mentions))
	for meaning := range cues.mentions {
		meanings = append(meanings, meaning)
	}
	sort.Strings(meanings)
	for _, meaning := range meanings {
		category, value, valid := strings.Cut(meaning, "\x00")
		if !valid {
			return nil, fmt.Errorf("core cue の意味 key が有効ではありません")
		}
		for _, mention := range cues.mentions[meaning] {
			key := cueRelationRefKeyFromMention(mention)
			previous, exists := result[key]
			if exists &&
				(previous.category != category || previous.value != value) {
				return nil, fmt.Errorf(
					"同じ core cue 出現に複数の意味があります",
				)
			}
			result[key] = coreCueFactMetadata{
				category:   category,
				value:      value,
				directTask: directTasks[key],
			}
		}
	}
	return result, nil
}

func coreDirectTaskCueKeys(
	input legalquery.CandidateGenerationInput,
) map[cueRelationRefKey]bool {
	result := make(map[cueRelationRefKey]bool)
	for _, relation := range input.CueTaskRelations() {
		if relation.Kind() != legalquery.CueTaskRelationDirectTask {
			continue
		}
		subject := relation.Subject()
		if subject.ProfileID() == profileID {
			result[cueRelationRefKeyFromRelationRef(subject)] = true
		}
	}
	return result
}

func buildCoreEvidenceDraftValue(
	index int,
	draft candidateDraft,
	facts coreEvidenceFactSet,
) (
	profileevidence.DraftValues,
	coreEvidenceDraftRef,
	bool,
	error,
) {
	draftID := fmt.Sprintf("draft-%d", index+1)
	raw, reference, err := coreEvidenceRawDraftValue(draftID, draft)
	if err != nil {
		return profileevidence.DraftValues{}, coreEvidenceDraftRef{}, false, fmt.Errorf(
			"%s: %w",
			draftID,
			err,
		)
	}
	if err := validateCoreRawDraft(raw, facts.values); err != nil {
		return profileevidence.DraftValues{}, coreEvidenceDraftRef{}, false, fmt.Errorf(
			"%s: %w",
			draftID,
			err,
		)
	}
	for stepIndex, step := range raw.Steps {
		if err := validateCoreEvidenceBindings(
			draft.steps[stepIndex].input,
			step.Evidence,
			facts.byID,
		); err != nil {
			return profileevidence.DraftValues{}, coreEvidenceDraftRef{}, false, fmt.Errorf(
				"%s/%s: %w",
				draftID,
				step.StepID,
				err,
			)
		}
	}
	filtered, retained := removeAmbiguousCoreEvidence(raw, draft, facts.byID)
	if !retained {
		return profileevidence.DraftValues{}, coreEvidenceDraftRef{}, false, nil
	}
	filtered, err = withCoreNormalizationGroups(filtered, draft, facts.byID)
	if err != nil {
		return profileevidence.DraftValues{}, coreEvidenceDraftRef{}, false, fmt.Errorf(
			"%s: %w",
			draftID,
			err,
		)
	}
	if !coreRequiredStepEvidencePresent(filtered, draft, facts.byID) {
		return profileevidence.DraftValues{}, coreEvidenceDraftRef{}, false, nil
	}
	if !coreEvidenceDraftHasPositiveTopics(filtered) {
		return profileevidence.DraftValues{}, coreEvidenceDraftRef{}, false, nil
	}
	return filtered, reference, true, nil
}

func coreRequiredStepEvidencePresent(
	value profileevidence.DraftValues,
	draft candidateDraft,
	facts map[string]coreEvidenceFact,
) bool {
	if len(value.Steps) != len(draft.steps) {
		return false
	}
	for index, step := range value.Steps {
		kind := draft.steps[index].input.InputKind()
		requireTask := kind == legalquery.InputKindLawRead ||
			kind == legalquery.InputKindLawArticleRead ||
			kind == legalquery.InputKindLawUpdates
		requireResource := kind == legalquery.InputKindLawUpdates
		var hasTask bool
		var hasResource bool
		var hasPositive bool
		var hasLawTarget bool
		var hasArticleTarget bool
		var hasUpdateDate bool
		for _, evidence := range step.Evidence {
			fact := facts[evidence.FactID]
			if evidence.IndependentPositive {
				hasPositive = true
			}
			if evidence.Layer != profileevidence.LayerExplicitTaskResource {
				if evidence.IndependentPositive {
					hasLawTarget = hasLawTarget ||
						fact.kind == coreEvidenceFactInputRef ||
						fact.kind == coreEvidenceFactLawName ||
						fact.kind == coreEvidenceFactIdentifier
					hasArticleTarget = hasArticleTarget ||
						fact.kind == coreEvidenceFactArticle
					hasUpdateDate = hasUpdateDate ||
						fact.kind == coreEvidenceFactDate
				}
				continue
			}
			hasTask = hasTask ||
				evidence.Code == legalquery.EvidenceExplicitTask
			hasResource = hasResource ||
				evidence.Code == legalquery.EvidenceExplicitResource
		}
		if !hasPositive ||
			requireTask && !hasTask ||
			requireResource && !hasResource {
			return false
		}
		switch kind {
		case legalquery.InputKindLawRead:
			if !hasLawTarget {
				return false
			}
		case legalquery.InputKindLawArticleRead:
			if !hasLawTarget || !hasArticleTarget {
				return false
			}
		case legalquery.InputKindLawUpdates:
			if !hasUpdateDate {
				return false
			}
		}
	}
	return true
}

func coreEvidenceRawDraftValue(
	draftID string,
	draft candidateDraft,
) (profileevidence.DraftValues, coreEvidenceDraftRef, error) {
	value := profileevidence.DraftValues{
		DraftID: draftID,
		Steps:   make([]profileevidence.StepValues, 0, len(draft.steps)),
	}
	reference := coreEvidenceDraftRef{
		draftID: draftID,
		stepIDs: make([]string, 0, len(draft.steps)),
	}
	for index, step := range draft.steps {
		signature, err := logicalInputSignature(step.input)
		if err != nil {
			return profileevidence.DraftValues{}, coreEvidenceDraftRef{}, err
		}
		stepID := fmt.Sprintf("step-%d", index+1)
		value.Steps = append(value.Steps, profileevidence.StepValues{
			StepID:               stepID,
			SourceOrdinal:        index + 1,
			TopicOrdinal:         step.topicOrdinal,
			StepMeaningSignature: signature,
			Evidence: append(
				[]profileevidence.EvidenceValues(nil),
				step.evidenceBindings...,
			),
		})
		reference.stepIDs = append(reference.stepIDs, stepID)
	}
	return value, reference, nil
}

func validateCoreRawDraft(
	value profileevidence.DraftValues,
	facts []profileevidence.FactValues,
) error {
	allStepsHaveEvidence := len(value.Steps) > 0
	for _, step := range value.Steps {
		if len(step.Evidence) == 0 {
			allStepsHaveEvidence = false
			continue
		}
		validation := profileevidence.DraftValues{
			DraftID: "draft",
			Steps: []profileevidence.StepValues{{
				StepID:               "step",
				SourceOrdinal:        1,
				TopicOrdinal:         1,
				StepMeaningSignature: step.StepMeaningSignature,
				Evidence:             step.Evidence,
			}},
		}
		if err := newCoreValidationMapping(facts, validation); err != nil {
			return err
		}
	}
	if allStepsHaveEvidence ||
		len(value.Steps) > legalquery.MaxCapabilityCalls {
		return newCoreValidationMapping(facts, value)
	}
	return nil
}

func newCoreValidationMapping(
	facts []profileevidence.FactValues,
	draft profileevidence.DraftValues,
) error {
	_, err := profileevidence.NewMapping(profileevidence.MappingValues{
		ProfileID: profileID,
		Facts:     facts,
		Drafts:    []profileevidence.DraftValues{draft},
	})
	return err
}

func validateCoreEvidenceBindings(
	input legalquery.LogicalInput,
	values []profileevidence.EvidenceValues,
	facts map[string]coreEvidenceFact,
) error {
	if input == nil {
		return fmt.Errorf("logical input は必須です")
	}
	kind := input.InputKind()
	for _, value := range values {
		fact, exists := facts[value.FactID]
		if !exists {
			continue
		}
		if !coreEvidenceBindingAllowed(kind, value, fact) {
			return fmt.Errorf(
				"input kind %q に fact %q の %s/%s は使用できません",
				kind,
				value.FactID,
				value.Layer,
				value.Code,
			)
		}
	}
	return nil
}

func coreEvidenceDraftHasPositiveTopics(
	value profileevidence.DraftValues,
) bool {
	if len(value.Steps) == 0 {
		return true
	}
	required := make(map[int]struct{}, len(value.Steps))
	positive := make(map[int]bool, len(value.Steps))
	for _, step := range value.Steps {
		if step.TopicOrdinal < 1 || len(step.Evidence) == 0 {
			return false
		}
		required[step.TopicOrdinal] = struct{}{}
		for _, evidence := range step.Evidence {
			if evidence.IndependentPositive {
				positive[step.TopicOrdinal] = true
			}
		}
	}
	for topic := range required {
		if !positive[topic] {
			return false
		}
	}
	return true
}

func coreEvidenceBindingAllowed(
	kind legalquery.LogicalInputKind,
	value profileevidence.EvidenceValues,
	fact coreEvidenceFact,
) bool {
	switch value.Layer {
	case profileevidence.LayerBoundary:
		return (kind == legalquery.InputKindLawRead ||
			kind == legalquery.InputKindLawArticleRead) &&
			value.Code == legalquery.EvidenceOfficialIdentifier &&
			fact.kind == coreEvidenceFactInputRef &&
			fact.lawRef
	case profileevidence.LayerExplicitTaskResource:
		return coreExplicitBindingAllowed(kind, value.Code, fact)
	case profileevidence.LayerTargetAnchor:
		return coreTargetBindingAllowed(kind, value, fact)
	case profileevidence.LayerSemanticExpansion:
		return coreSemanticBindingAllowed(kind, value.Code, fact)
	default:
		return false
	}
}

func coreExplicitBindingAllowed(
	kind legalquery.LogicalInputKind,
	code legalquery.EvidenceCode,
	fact coreEvidenceFact,
) bool {
	if fact.kind != coreEvidenceFactCue {
		return false
	}
	if fact.cueCategory == "" {
		return false
	}
	switch code {
	case legalquery.EvidenceExplicitTask:
		return fact.directTask &&
			fact.cueCategory == "task" &&
			fact.cueValue == coreEvidenceTaskValue(kind)
	case legalquery.EvidenceExplicitResource:
		return fact.cueCategory == "resource" &&
			fact.cueValue == coreEvidenceResourceValue(kind)
	default:
		return false
	}
}

func coreTargetBindingAllowed(
	kind legalquery.LogicalInputKind,
	value profileevidence.EvidenceValues,
	fact coreEvidenceFact,
) bool {
	switch value.Code {
	case legalquery.EvidenceOfficialIdentifier:
		return coreIdentifierTargetAllowed(kind, fact)
	case legalquery.EvidenceStructuredReference:
		return coreStructuredTargetAllowed(kind, value, fact)
	case legalquery.EvidenceOfficialAlias:
		return coreAliasTargetAllowed(kind, fact, false)
	case legalquery.EvidenceUniqueTypoCorrection:
		return coreAliasTargetAllowed(kind, fact, true)
	case legalquery.EvidenceGeneralTerm:
		return (kind == legalquery.InputKindLawSearch ||
			kind == legalquery.InputKindLawContentSearch) &&
			fact.kind == coreEvidenceFactQueryTerm &&
			fact.queryTermKind == legalquery.QueryTermMentionQuotedPhrase
	default:
		return false
	}
}

func coreIdentifierTargetAllowed(
	kind legalquery.LogicalInputKind,
	fact coreEvidenceFact,
) bool {
	if fact.kind != coreEvidenceFactIdentifier ||
		(fact.identifierKind != legalquery.IdentifierMentionLawID &&
			fact.identifierKind !=
				legalquery.IdentifierMentionLawRevisionID) {
		return false
	}
	return kind == legalquery.InputKindLawSearch ||
		kind == legalquery.InputKindLawRead ||
		kind == legalquery.InputKindLawArticleRead
}

func coreStructuredTargetAllowed(
	kind legalquery.LogicalInputKind,
	value profileevidence.EvidenceValues,
	fact coreEvidenceFact,
) bool {
	switch kind {
	case legalquery.InputKindLawSearch,
		legalquery.InputKindLawContentSearch:
		return coreLawNumberOrAsOf(value, fact)
	case legalquery.InputKindLawRead:
		return coreLawNumberOrAsOf(value, fact)
	case legalquery.InputKindLawArticleRead:
		return coreLawNumberOrAsOf(value, fact) ||
			fact.kind == coreEvidenceFactArticle ||
			fact.kind == coreEvidenceFactParagraph
	case legalquery.InputKindLawUpdates:
		return fact.kind == coreEvidenceFactDate
	default:
		return false
	}
}

func coreLawNumberOrAsOf(
	value profileevidence.EvidenceValues,
	fact coreEvidenceFact,
) bool {
	if fact.kind == coreEvidenceFactIdentifier {
		return fact.identifierKind == legalquery.IdentifierMentionLawNumber
	}
	return fact.kind == coreEvidenceFactDate &&
		!value.IndependentPositive &&
		!value.ClusterSpan
}

func coreAliasTargetAllowed(
	kind legalquery.LogicalInputKind,
	fact coreEvidenceFact,
	typo bool,
) bool {
	if fact.kind != coreEvidenceFactLawName ||
		(typo &&
			fact.matchKind != legalquery.PreprocessMatchUniqueTypoCorrection) {
		return false
	}
	return kind == legalquery.InputKindLawSearch ||
		kind == legalquery.InputKindLawContentSearch ||
		kind == legalquery.InputKindLawRead ||
		kind == legalquery.InputKindLawArticleRead
}

func coreSemanticBindingAllowed(
	kind legalquery.LogicalInputKind,
	code legalquery.EvidenceCode,
	fact coreEvidenceFact,
) bool {
	if kind != legalquery.InputKindLawSearch &&
		kind != legalquery.InputKindLawContentSearch {
		return false
	}
	switch code {
	case legalquery.EvidenceLegalConcept:
		return kind == legalquery.InputKindLawContentSearch &&
			fact.kind == coreEvidenceFactLegalConcept
	case legalquery.EvidenceUniqueTypoCorrection:
		return kind == legalquery.InputKindLawContentSearch &&
			fact.kind == coreEvidenceFactLegalConcept &&
			fact.matchKind ==
				legalquery.PreprocessMatchUniqueTypoCorrection
	case legalquery.EvidenceMorphologicalContext:
		return fact.kind == coreEvidenceFactQueryTerm &&
			fact.queryTermKind ==
				legalquery.QueryTermMentionMorphologicalPhrase
	default:
		return false
	}
}

func coreEvidenceTaskValue(
	kind legalquery.LogicalInputKind,
) string {
	switch kind {
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

func coreEvidenceResourceValue(
	kind legalquery.LogicalInputKind,
) string {
	switch kind {
	case legalquery.InputKindLawSearch,
		legalquery.InputKindLawRead:
		return "law"
	case legalquery.InputKindLawContentSearch,
		legalquery.InputKindLawArticleRead:
		return "law_provision"
	case legalquery.InputKindLawUpdates:
		return "updates"
	default:
		return ""
	}
}

func coreEvidenceMentionFact(
	prefix string,
	index int,
	span legalquery.QuerySpan,
	kind coreEvidenceFactKind,
	matchKind legalquery.PreprocessMatchKind,
) coreEvidenceFact {
	cloned := span
	return coreEvidenceFact{
		values: profileevidence.FactValues{
			FactID: fmt.Sprintf("%s-%d", prefix, index+1),
			Span:   &cloned,
		},
		kind:      kind,
		matchKind: matchKind,
	}
}

func (s *coreEvidenceFactSet) add(value coreEvidenceFact) {
	s.values = append(s.values, value.values)
	s.byID[value.values.FactID] = value
}
