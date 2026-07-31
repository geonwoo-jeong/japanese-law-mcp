package core

import (
	"fmt"
	"slices"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type coreSharedTerminalBinding struct {
	relation            legalquery.CueTaskRelation
	taskFactID          string
	resourceFactIDs     []string
	lawProvisionBinding coreLawProvisionBindingKind
}

type coreSharedTerminalTopic struct {
	span    legalquery.QuerySpan
	options []coreTopicOption
}

type coreSharedTerminalBuild struct {
	drafts            []coreSharedTerminalDraft
	handled           bool
	stepLimitExceeded bool
}

// buildCoreSharedTerminalDrafts は、検証済み sidecar だけから次版 core の
// topic-local draft と限定代替列を作る。active profile からは到達しない。
func (p *Profile) buildCoreSharedTerminalDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) (coreSharedTerminalBuild, error) {
	sequences := input.SharedTerminalSequences()
	if len(sequences) == 0 || hasExplicitContentOperator(cues) {
		return coreSharedTerminalBuild{}, nil
	}

	topics := make([]coreSharedTerminalTopic, 0)
	for _, sequence := range sequences {
		binding, valid := p.coreSharedTerminalBinding(input, cues, sequence)
		if !valid {
			return coreSharedTerminalBuild{}, nil
		}
		for _, span := range sequence.TopicSpans() {
			options, err := p.coreSharedTerminalTopicOptions(
				input,
				cues,
				span,
				binding,
			)
			if err != nil {
				return coreSharedTerminalBuild{}, err
			}
			if len(options) == 0 {
				return coreSharedTerminalBuild{}, nil
			}
			topics = append(topics, coreSharedTerminalTopic{
				span:    span,
				options: options,
			})
		}
	}
	for index := 1; index < len(topics); index++ {
		if topics[index-1].span.EndByte() > topics[index].span.StartByte() {
			// 複数 sidecar 間の重複を配列順で解消しない。
			return coreSharedTerminalBuild{}, nil
		}
	}

	optionSets := make([][]coreTopicOption, 0, len(topics))
	for _, topic := range topics {
		optionSets = append(optionSets, topic.options)
	}
	choices := coreBoundedTopicChoices(optionSets)
	drafts := make([]coreSharedTerminalDraft, 0, len(choices))
	for _, choice := range choices {
		draft, err := buildCoreSharedTerminalDraft(choice)
		if err != nil {
			return coreSharedTerminalBuild{}, err
		}
		if len(draft.draft.steps) > legalquery.MaxCapabilityCalls {
			return coreSharedTerminalBuild{
				handled:           true,
				stepLimitExceeded: true,
			}, nil
		}
		drafts = append(drafts, draft)
	}
	return coreSharedTerminalBuild{
		drafts:  drafts,
		handled: true,
	}, nil
}

func (p *Profile) coreSharedTerminalBinding(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	sequence legalquery.SharedTerminalSequence,
) (coreSharedTerminalBinding, bool) {
	relation := sequence.TerminalTaskRelation()
	if !coreInputHasExactTerminalRelation(input, relation) ||
		relation.Kind() != legalquery.CueTaskRelationDirectTask ||
		relation.Subject() != relation.Predicate() {
		return coreSharedTerminalBinding{}, false
	}

	subject := relation.Subject()
	if subject.ProfileID() != profileID {
		return coreSharedTerminalBinding{}, false
	}
	definition, exists := p.cueByID[subject.CueID()]
	if !exists || definition.category != "task" || definition.value != "search" ||
		!coreResolvedCueRefMatches(
			cues.mentions[cueMeaningKey("task", "search")],
			subject,
		) {
		return coreSharedTerminalBinding{}, false
	}
	taskFactID, exists := coreCueFactID(input, subject)
	if !exists || !p.coreSharedTerminalRelationsCompatible(input, cues, relation) {
		return coreSharedTerminalBinding{}, false
	}

	resourceFactIDs, compatible := coreSharedTerminalResourceFacts(
		input,
		cues,
		relation.ClauseSpan(),
	)
	if !compatible {
		return coreSharedTerminalBinding{}, false
	}
	if len(resourceFactIDs) > 0 {
		return coreSharedTerminalBinding{
			relation:            relation,
			taskFactID:          taskFactID,
			resourceFactIDs:     slices.Clone(resourceFactIDs),
			lawProvisionBinding: coreLawProvisionBindingExplicitResource,
		}, true
	}

	surface, exists := coreExactCueSurface(input, subject)
	if !exists || (surface != "教えて" && surface != "教えてください") {
		return coreSharedTerminalBinding{}, false
	}
	return coreSharedTerminalBinding{
		relation:            relation,
		taskFactID:          taskFactID,
		lawProvisionBinding: coreLawProvisionBindingTerminalTask,
	}, true
}

func coreInputHasExactTerminalRelation(
	input legalquery.CandidateGenerationInput,
	target legalquery.CueTaskRelation,
) bool {
	count := 0
	for _, relation := range input.CueTaskRelations() {
		if relation == target {
			count++
		}
	}
	return count == 1
}

func (p *Profile) coreSharedTerminalRelationsCompatible(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	terminal legalquery.CueTaskRelation,
) bool {
	for _, relation := range input.CueTaskRelations() {
		if !sameQuerySpan(relation.ClauseSpan(), terminal.ClauseSpan()) ||
			relation == terminal {
			continue
		}
		// 同じ節の別 task relation は、配列順や距離では弱めない。
		return false
	}

	for meaning, mentions := range cues.mentions {
		category, value, valid := strings.Cut(meaning, "\x00")
		if !valid || category != "task" {
			continue
		}
		for _, mention := range mentions {
			if !coreSpanContains(terminal.ClauseSpan(), mention.Span()) {
				continue
			}
			if value != "search" ||
				mention.ProfileID() != terminal.Subject().ProfileID() ||
				mention.CueID() != terminal.Subject().CueID() ||
				!sameQuerySpan(mention.Span(), terminal.Subject().Span()) {
				return false
			}
		}
	}
	return true
}

func coreSharedTerminalResourceFacts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	clause legalquery.QuerySpan,
) ([]string, bool) {
	result := make([]string, 0, 1)
	for meaning, mentions := range cues.mentions {
		category, value, valid := strings.Cut(meaning, "\x00")
		if !valid || category != "resource" {
			continue
		}
		for _, mention := range mentions {
			if !coreSpanContains(clause, mention.Span()) {
				continue
			}
			if value != "law_provision" {
				return nil, false
			}
			factID, exists := coreCueMentionFactID(input, mention)
			if !exists {
				return nil, false
			}
			if !slices.Contains(result, factID) {
				result = append(result, factID)
			}
		}
	}
	slices.Sort(result)
	return result, true
}

func coreCueMentionFactID(
	input legalquery.CandidateGenerationInput,
	target legalquery.CueMention,
) (string, bool) {
	for index, mention := range input.CueMentions() {
		if mention.ProfileID() == target.ProfileID() &&
			mention.CueID() == target.CueID() &&
			sameQuerySpan(mention.Span(), target.Span()) {
			return fmt.Sprintf("cue-%d", index+1), true
		}
	}
	return "", false
}

func coreExactCueSurface(
	input legalquery.CandidateGenerationInput,
	ref legalquery.CueTaskRelationRef,
) (string, bool) {
	var surface string
	count := 0
	for _, mention := range input.CueMentions() {
		if mention.ProfileID() == ref.ProfileID() &&
			mention.CueID() == ref.CueID() &&
			sameQuerySpan(mention.Span(), ref.Span()) {
			surface = mention.Surface()
			count++
		}
	}
	return surface, count == 1
}

func (p *Profile) coreSharedTerminalTopicOptions(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	span legalquery.QuerySpan,
	binding coreSharedTerminalBinding,
) ([]coreTopicOption, error) {
	baseEvidence := []profileevidence.EvidenceValues{{
		FactID:      binding.taskFactID,
		Layer:       profileevidence.LayerExplicitTaskResource,
		Code:        legalquery.EvidenceExplicitTask,
		ClusterSpan: true,
	}}
	if binding.lawProvisionBinding == coreLawProvisionBindingExplicitResource {
		for _, factID := range binding.resourceFactIDs {
			baseEvidence = append(baseEvidence, profileevidence.EvidenceValues{
				FactID:      factID,
				Layer:       profileevidence.LayerExplicitTaskResource,
				Code:        legalquery.EvidenceExplicitResource,
				ClusterSpan: true,
			})
		}
	}
	validated, err := newCoreValidatedContentTopic(
		span,
		baseEvidence,
		binding.lawProvisionBinding,
	)
	if err != nil {
		return nil, err
	}

	result := make([]coreTopicOption, 0)
	quoted, quotedFactID, quotedExists := exactQuotedTerm(input, span)
	if quotedExists {
		option, optionErr := newCoreTopicOption(
			quoted.Surface(),
			span.StartByte(),
			append(slices.Clone(baseEvidence), profileevidence.EvidenceValues{
				FactID:              quotedFactID,
				Layer:               profileevidence.LayerTargetAnchor,
				Code:                legalquery.EvidenceGeneralTerm,
				IndependentPositive: true,
				ClusterSpan:         true,
			}),
			nil,
			"quoted_phrase",
		)
		if optionErr != nil {
			return nil, optionErr
		}
		result = append(result, option)
	}

	if hasLawNameAtSpan(input.LawNameMentions(), span) && !quotedExists {
		context, contextErr := newCoreLawNameProjectionContext(
			span,
			nil,
			&validated,
		)
		if contextErr != nil {
			return nil, contextErr
		}
		option, exists, projectionErr := p.projectCoreLawName(input, cues, context)
		if projectionErr != nil {
			return nil, projectionErr
		}
		if exists {
			option.meaningKey = "law_name_projection"
			result = append(result, option)
		}
	}

	for index, mention := range input.LegalConceptMentions() {
		if !sameQuerySpan(mention.Span(), span) {
			continue
		}
		definition, definitionErr := p.conceptDefinitionForMention(mention)
		if definitionErr != nil {
			return nil, definitionErr
		}
		for _, candidate := range definition.entry.Candidates {
			if !isCoreConceptCandidate(candidate) {
				continue
			}
			factID := fmt.Sprintf("legal-concept-%d", index+1)
			evidence := append(
				slices.Clone(baseEvidence),
				profileevidence.EvidenceValues{
					FactID:              factID,
					Layer:               profileevidence.LayerSemanticExpansion,
					Code:                legalquery.EvidenceLegalConcept,
					IndependentPositive: true,
					ClusterSpan:         true,
				},
			)
			if mention.MatchKind() == legalquery.PreprocessMatchUniqueTypoCorrection {
				evidence = append(evidence, profileevidence.EvidenceValues{
					FactID: factID,
					Layer:  profileevidence.LayerSemanticExpansion,
					Code:   legalquery.EvidenceUniqueTypoCorrection,
				})
			}
			option, optionErr := newCoreTopicOption(
				candidate.OfficialTermFor(mention.Surface()),
				span.StartByte(),
				evidence,
				[]legalquery.LegalConceptSource{definition.source},
				coreConceptMeaningKey(mention.ConceptID(), candidate),
			)
			if optionErr != nil {
				return nil, optionErr
			}
			result = append(result, option)
		}
	}

	for index, mention := range input.QueryTermMentions() {
		if !sameQuerySpan(mention.Span(), span) ||
			mention.Kind() != legalquery.QueryTermMentionMorphologicalPhrase {
			continue
		}
		option, optionErr := newCoreTopicOption(
			mention.Surface(),
			span.StartByte(),
			append(slices.Clone(baseEvidence), profileevidence.EvidenceValues{
				FactID:              fmt.Sprintf("query-term-%d", index+1),
				Layer:               profileevidence.LayerSemanticExpansion,
				Code:                legalquery.EvidenceMorphologicalContext,
				IndependentPositive: true,
				ClusterSpan:         true,
			}),
			nil,
			"morphological_phrase",
		)
		if optionErr != nil {
			return nil, optionErr
		}
		result = append(result, option)
	}
	return p.orderCoreTopicOptions(result)
}

func coreConceptMeaningKey(
	conceptID string,
	candidate legalconceptlexicon.Candidate,
) string {
	packs := slices.Clone(candidate.RequiredPacks)
	slices.Sort(packs)
	return fmt.Sprintf(
		"legal_concept:%s:%s:%s:%s:%s:%s",
		conceptID,
		candidate.Task,
		candidate.Resource,
		candidate.InputKind,
		candidate.OfficialTerm,
		strings.Join(packs, ","),
	)
}
