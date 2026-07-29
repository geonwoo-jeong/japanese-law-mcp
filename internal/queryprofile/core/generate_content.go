package core

import (
	"fmt"
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func (p *Profile) buildContentCandidates(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	hasLawTargets bool,
) ([]candidateDraft, error) {
	conceptDrafts, err := p.buildConceptCandidates(input, cues)
	if err != nil {
		return nil, err
	}
	terms := coreContentQueryTerms(input, cues)
	operated, handled, err := buildOperatedContentCandidates(
		input,
		cues,
		conceptDrafts,
		terms,
	)
	if err != nil {
		return nil, err
	}
	if handled {
		return operated, nil
	}
	separated, handled, err := buildSeparatedContentCandidates(
		input,
		cues,
		conceptDrafts,
		terms,
	)
	if err != nil {
		return nil, err
	}
	if handled {
		return separated, nil
	}
	if len(terms) == 0 {
		return conceptDrafts, nil
	}
	if hasLawTargets && !cues.has("resource", "law_provision") {
		return conceptDrafts, nil
	}

	if isWeakLawResourceAmbiguity(cues, hasLawTargets, len(conceptDrafts)) {
		ambiguous, ambiguousErr := buildWeakGeneralDrafts(input, cues, terms)
		if ambiguousErr != nil {
			return nil, ambiguousErr
		}
		return append(conceptDrafts, ambiguous...), nil
	}

	termDraft, err := buildTermContentDraft(input, cues, terms)
	if err != nil {
		return nil, err
	}
	if termDraft == nil {
		return conceptDrafts, nil
	}
	return append(conceptDrafts, *termDraft), nil
}

func coreContentQueryTerms(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) []legalquery.QueryTermMention {
	terms := append(
		[]legalquery.QueryTermMention(nil),
		input.QueryTermMentions()...,
	)
	ref, exists := input.Ref()
	if exists && cues.has("task", "read") {
		readCues := cues.mentions[cueMeaningKey("task", "read")]
		searchCues := cues.mentions[cueMeaningKey("task", "search")]
		resourceCues := refReadResourceCues(ref, cues)
		filtered := make([]legalquery.QueryTermMention, 0, len(terms))
		for _, term := range terms {
			if term.Kind() ==
				legalquery.QueryTermMentionMorphologicalPhrase &&
				isRefReadTerm(
					term,
					resourceCues,
					readCues,
					searchCues,
				) {
				continue
			}
			filtered = append(filtered, term)
		}
		terms = filtered
	}
	if cues.has("task", "read") {
		terms = excludeLawTargetReadTerms(
			input,
			terms,
			cues.mentions[cueMeaningKey("task", "read")],
			cues.mentions[cueMeaningKey("task", "search")],
		)
	}
	return queryTermsForCoreResources(input, terms, cues)
}

func excludeLawTargetReadTerms(
	input legalquery.CandidateGenerationInput,
	terms []legalquery.QueryTermMention,
	readCues []legalquery.CueMention,
	searchCues []legalquery.CueMention,
) []legalquery.QueryTermMention {
	filtered := make([]legalquery.QueryTermMention, 0, len(terms))
	for _, term := range terms {
		if isLawTargetSpan(input, term.Span()) &&
			termPrecedesReadWithoutSearchBetween(
				term,
				readCues,
				searchCues,
			) {
			continue
		}
		filtered = append(filtered, term)
	}
	return filtered
}

func isLawTargetSpan(
	input legalquery.CandidateGenerationInput,
	span legalquery.QuerySpan,
) bool {
	for _, mention := range input.LawNameMentions() {
		if sameQuerySpan(span, mention.Span()) {
			return true
		}
	}
	for _, mention := range input.IdentifierMentions() {
		if sameQuerySpan(span, mention.Span()) {
			return true
		}
	}
	return false
}

func sameQuerySpan(left legalquery.QuerySpan, right legalquery.QuerySpan) bool {
	return left.StartByte() == right.StartByte() &&
		left.EndByte() == right.EndByte()
}

func refReadResourceCues(
	ref model.SourceResourceRef,
	cues resolvedCues,
) []legalquery.CueMention {
	switch ref.Key().ResourceType() {
	case "law":
		return nil
	case "judicial-decision":
		if !cues.has("reserved_pack", "judicial-cases") {
			return nil
		}
		return cues.mentions[cueMeaningKey("reserved_pack", "judicial-cases")]
	default:
		return nil
	}
}

func isRefReadTerm(
	term legalquery.QueryTermMention,
	resourceCues []legalquery.CueMention,
	readCues []legalquery.CueMention,
	searchCues []legalquery.CueMention,
) bool {
	if term.Surface() == "参照" &&
		termPrecedesReadWithoutSearchBetween(
			term,
			readCues,
			searchCues,
		) {
		return true
	}
	for _, resourceCue := range resourceCues {
		for _, readCue := range readCues {
			if resourceCue.Span().EndByte() <=
				term.Span().StartByte() &&
				term.Span().EndByte() <=
					readCue.Span().StartByte() &&
				!cueStartsBetween(
					searchCues,
					term.Span().EndByte(),
					readCue.Span().StartByte(),
				) {
				return true
			}
		}
	}
	return false
}

func termPrecedesReadWithoutSearchBetween(
	term legalquery.QueryTermMention,
	readCues []legalquery.CueMention,
	searchCues []legalquery.CueMention,
) bool {
	for _, readCue := range readCues {
		if term.Span().EndByte() <= readCue.Span().StartByte() &&
			!cueStartsBetween(
				searchCues,
				term.Span().EndByte(),
				readCue.Span().StartByte(),
			) {
			return true
		}
	}
	return false
}

func cueStartsBetween(
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

func (p *Profile) buildConceptCandidates(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, error) {
	result := make([]candidateDraft, 0)
	asOf := selectedAsOfDate(input, cues, false)
	for _, mention := range p.selectedCoreConceptMentions(input, cues) {
		unresolvedConceptResource := usesUnresolvedConceptResourceScope(
			input,
			cues,
			mention,
		)
		definition, exists := p.concepts[mention.ConceptID()]
		if !exists {
			return nil, fmt.Errorf(
				"core profile に未登録の conceptId %q があります",
				mention.ConceptID(),
			)
		}
		if mention.Canonical() != definition.entry.Canonical {
			return nil, fmt.Errorf(
				"conceptId %q の canonical が profile snapshot と一致しません",
				mention.ConceptID(),
			)
		}
		for _, candidate := range definition.entry.Candidates {
			if !isCoreConceptCandidate(candidate) {
				continue
			}
			contentInput, err := legalquery.NewLawContentSearchIntentV1(
				legalquery.LawContentSearchIntentV1Values{
					AllTerms: []string{candidate.OfficialTerm},
					AsOf:     asOf,
				},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"conceptId %q の本文検索条件を構築できません: %w",
					mention.ConceptID(),
					err,
				)
			}
			draft := newCandidateDraft()
			draft.evidence[legalquery.EvidenceLegalConcept] = struct{}{}
			addExplicitSearchEvidence(
				&draft,
				cues,
				hasLegalResourceCue(cues) &&
					!unresolvedConceptResource,
			)
			if unresolvedConceptResource {
				draft.evidence[legalquery.EvidenceMorphologicalContext] =
					struct{}{}
				draft.preserveMorphologicalContext = true
			}
			if mention.MatchKind() ==
				legalquery.PreprocessMatchUniqueTypoCorrection {
				draft.evidence[legalquery.EvidenceUniqueTypoCorrection] =
					struct{}{}
			}
			draft.concepts = append(draft.concepts, definition.source)
			draft.steps = append(draft.steps, stepDraft{
				startByte: contentSubjectStartByte(
					mention.Span(),
					cues,
				),
				input: contentInput,
			})
			result = append(result, draft)
		}
	}
	return result, nil
}

func buildSeparatedContentCandidates(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	concepts []candidateDraft,
	terms []legalquery.QueryTermMention,
) ([]candidateDraft, bool, error) {
	if !separatesSubjects(cues) {
		return nil, false, nil
	}
	options, err := buildContentSubjectOptions(
		input,
		cues,
		concepts,
		terms,
	)
	if err != nil {
		return nil, false, err
	}
	if len(options) < 2 {
		return nil, false, nil
	}
	if len(options) > 4 {
		return nil, true, nil
	}

	positions := sortedSubjectOptionPositions(options)
	result := []candidateDraft{newCandidateDraft()}
	for _, position := range positions {
		next := make([]candidateDraft, 0, len(result)*len(options[position]))
		for _, base := range result {
			for _, option := range options[position] {
				current := cloneDraft(base)
				mergeDraft(&current, option.draft)
				next = append(next, current)
				if len(next) > maximumGeneratedCandidates {
					return nil, true, nil
				}
			}
		}
		result = next
	}
	return result, true, nil
}

func buildOperatedContentCandidates(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	concepts []candidateDraft,
	terms []legalquery.QueryTermMention,
) ([]candidateDraft, bool, error) {
	if !hasExplicitContentOperator(cues) {
		return nil, false, nil
	}
	options, err := buildContentSubjectOptions(
		input,
		cues,
		concepts,
		terms,
	)
	if err != nil {
		return nil, false, err
	}
	if len(options) < 2 {
		return nil, false, nil
	}
	positions := sortedSubjectOptionPositions(options)
	combinations := []operatedContentCombination{{
		draft: newCandidateDraft(),
	}}
	for _, position := range positions {
		next := make(
			[]operatedContentCombination,
			0,
			len(combinations)*len(options[position]),
		)
		for _, base := range combinations {
			for _, option := range options[position] {
				value, valueErr := singleContentSubjectValue(
					option.draft,
				)
				if valueErr != nil {
					return nil, false, valueErr
				}
				current := operatedContentCombination{
					draft: cloneDraft(base.draft),
					values: append(
						[]operatedContentValue(nil),
						base.values...,
					),
				}
				optionMetadata := cloneDraft(option.draft)
				optionMetadata.steps = nil
				mergeDraft(&current.draft, optionMetadata)
				current.values = appendUniqueContentValue(
					current.values,
					operatedContentValue{
						value:     value,
						startByte: position,
					},
				)
				next = append(next, current)
				if len(next) > maximumGeneratedCandidates {
					return nil, true, nil
				}
			}
		}
		combinations = next
	}
	asOf := selectedAsOfDate(input, cues, false)
	result := make([]candidateDraft, 0, len(combinations))
	for _, combination := range combinations {
		allTerms, anyTerms, excludeTerms := partitionSearchValues(
			combination.values,
			cues,
		)
		contentInput, inputErr := newContentInput(
			allTerms,
			anyTerms,
			excludeTerms,
			asOf,
		)
		if inputErr != nil {
			return nil, false, inputErr
		}
		current := cloneDraft(combination.draft)
		current.steps = []stepDraft{{
			startByte: positions[0],
			input:     contentInput,
		}}
		result = append(result, current)
	}
	return result, true, nil
}

type operatedContentCombination struct {
	draft  candidateDraft
	values []operatedContentValue
}

type operatedContentValue struct {
	value     string
	startByte int
}

func singleContentSubjectValue(
	draft candidateDraft,
) (string, error) {
	if len(draft.steps) != 1 {
		return "", fmt.Errorf(
			"検索演算子の一主題は一つの logical step を必要とします",
		)
	}
	contentInput, ok := draft.steps[0].input.(legalquery.LawContentSearchIntentV1)
	if !ok {
		return "", fmt.Errorf(
			"検索演算子の主題は law_content_search でなければなりません",
		)
	}
	values := append(contentInput.AllTerms(), contentInput.AnyTerms()...)
	values = append(values, contentInput.ExcludeTerms()...)
	if len(values) != 1 {
		return "", fmt.Errorf(
			"検索演算子の一主題は一つの検索語を必要とします",
		)
	}
	return values[0], nil
}

func appendUniqueContentValue(
	values []operatedContentValue,
	value operatedContentValue,
) []operatedContentValue {
	for _, current := range values {
		if current.value == value.value &&
			current.startByte == value.startByte {
			return values
		}
	}
	return append(values, value)
}

func hasExplicitContentOperator(cues resolvedCues) bool {
	return cues.has("operator", "all") ||
		cues.has("operator", "any") ||
		cues.has("operator", "exclude")
}

func buildContentSubjectOptions(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	concepts []candidateDraft,
	terms []legalquery.QueryTermMention,
) (map[int][]subjectOption, error) {
	options := make(map[int][]subjectOption)
	for _, concept := range concepts {
		if err := addSubjectOption(options, concept); err != nil {
			return nil, err
		}
	}
	asOf := selectedAsOfDate(input, cues, false)
	for _, term := range terms {
		contentInput, err := newContentInput(
			[]string{term.Surface()},
			nil,
			nil,
			asOf,
		)
		if err != nil {
			return nil, err
		}
		draft := newCandidateDraft()
		addExplicitSearchEvidence(
			&draft,
			cues,
			hasLegalResourceCue(cues),
		)
		draft.evidence[legalquery.EvidenceMorphologicalContext] = struct{}{}
		draft.steps = append(draft.steps, stepDraft{
			startByte: contentSubjectStartByte(term.Span(), cues),
			input:     contentInput,
		})
		if err := addSubjectOption(options, draft); err != nil {
			return nil, err
		}
	}
	return options, nil
}

func sortedSubjectOptionPositions(
	options map[int][]subjectOption,
) []int {
	positions := make([]int, 0, len(options))
	for position := range options {
		positions = append(positions, position)
	}
	sort.Ints(positions)
	return positions
}

type subjectOption struct {
	draft     candidateDraft
	signature string
}

func addSubjectOption(
	options map[int][]subjectOption,
	value candidateDraft,
) error {
	if len(value.steps) != 1 {
		return fmt.Errorf("複数主題の一選択肢は一つの logical step を必要とします")
	}
	position := value.steps[0].startByte
	signature, err := draftMeaningSignature(value)
	if err != nil {
		return err
	}
	for index, option := range options[position] {
		if option.signature == signature {
			mergeEquivalentDraft(&options[position][index].draft, value)
			return nil
		}
	}
	options[position] = append(options[position], subjectOption{
		draft:     cloneDraft(value),
		signature: signature,
	})
	sort.SliceStable(options[position], func(left, right int) bool {
		return options[position][left].signature <
			options[position][right].signature
	})
	return nil
}

func isCoreConceptCandidate(
	candidate legalconceptlexicon.Candidate,
) bool {
	return candidate.Task == legalquery.TaskSearch &&
		candidate.Resource == legalquery.ResourceLawProvision &&
		candidate.InputKind == legalquery.InputKindLawContentSearch &&
		len(candidate.RequiredPacks) == 0
}

func buildTermContentDraft(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	terms []legalquery.QueryTermMention,
) (*candidateDraft, error) {
	if len(terms) == 0 {
		return nil, nil
	}
	ordered := append([]legalquery.QueryTermMention(nil), terms...)
	sort.SliceStable(ordered, func(left, right int) bool {
		return ordered[left].Span().StartByte() <
			ordered[right].Span().StartByte()
	})
	asOf := selectedAsOfDate(input, cues, false)
	draft := newCandidateDraft()
	addExplicitSearchEvidence(
		&draft,
		cues,
		cues.has("resource", "law_provision"),
	)
	switch {
	case cues.has("resource", "law_provision"):
		draft.evidence[legalquery.EvidenceMorphologicalContext] = struct{}{}
	case separatesSubjects(cues) && len(ordered) > 1:
		draft.evidence[legalquery.EvidenceMorphologicalContext] = struct{}{}
	default:
		draft.evidence[legalquery.EvidenceGeneralTerm] = struct{}{}
	}

	if separatesSubjects(cues) && len(ordered) > 1 {
		if len(ordered) > 4 {
			return nil, nil
		}
		for _, term := range ordered {
			contentInput, err := newContentInput(
				[]string{term.Surface()},
				nil,
				nil,
				asOf,
			)
			if err != nil {
				return nil, err
			}
			draft.steps = append(draft.steps, stepDraft{
				startByte: contentSubjectStartByte(
					term.Span(),
					cues,
				),
				input: contentInput,
			})
		}
		return &draft, nil
	}

	allTerms, anyTerms, excludeTerms := partitionSearchTerms(ordered, cues)
	contentInput, err := newContentInput(
		allTerms,
		anyTerms,
		excludeTerms,
		asOf,
	)
	if err != nil {
		return nil, err
	}
	draft.steps = append(draft.steps, stepDraft{
		startByte: contentSubjectStartByte(
			ordered[0].Span(),
			cues,
		),
		input: contentInput,
	})
	return &draft, nil
}

func partitionSearchTerms(
	terms []legalquery.QueryTermMention,
	cues resolvedCues,
) ([]string, []string, []string) {
	values := make([]operatedContentValue, 0, len(terms))
	for _, term := range terms {
		values = append(values, operatedContentValue{
			value:     term.Surface(),
			startByte: term.Span().StartByte(),
		})
	}
	return partitionSearchValues(values, cues)
}

func partitionSearchValues(
	values []operatedContentValue,
	cues resolvedCues,
) ([]string, []string, []string) {
	surfaces := contentValueSurfaces(values)
	if !cues.has("operator", "exclude") || len(values) <= 1 {
		if cues.has("operator", "any") {
			return nil, surfaces, nil
		}
		return surfaces, nil, nil
	}

	excluded := excludeContentValueIndexes(values, cues)
	positiveTerms := make([]string, 0, len(values)-len(excluded))
	excludeTerms := make([]string, 0, len(excluded))
	for index, value := range values {
		if _, exists := excluded[index]; exists {
			excludeTerms = appendUniqueContentSurface(
				excludeTerms,
				value.value,
			)
			continue
		}
		positiveTerms = appendUniqueContentSurface(
			positiveTerms,
			value.value,
		)
	}
	if cues.has("operator", "any") {
		return nil, positiveTerms, excludeTerms
	}
	return positiveTerms, nil, excludeTerms
}

func contentValueSurfaces(values []operatedContentValue) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = appendUniqueContentSurface(result, value.value)
	}
	return result
}

func appendUniqueContentSurface(values []string, value string) []string {
	for _, current := range values {
		if current == value {
			return values
		}
	}
	return append(values, value)
}

func excludeContentValueIndexes(
	values []operatedContentValue,
	cues resolvedCues,
) map[int]struct{} {
	result := make(map[int]struct{})
	for _, cue := range cues.mentions[cueMeaningKey("operator", "exclude")] {
		best := -1
		for index, value := range values {
			if value.startByte >= cue.Span().StartByte() {
				continue
			}
			if best < 0 ||
				values[best].startByte < value.startByte {
				best = index
			}
		}
		if best >= 0 {
			result[best] = struct{}{}
		}
	}
	if len(result) == 0 {
		result[len(values)-1] = struct{}{}
	}
	return result
}

func separatesSubjects(cues resolvedCues) bool {
	return cues.has("operator", "individual") &&
		!cues.has("operator", "all") &&
		!cues.has("operator", "any") &&
		!cues.has("operator", "exclude")
}

func hasLegalResourceCue(cues resolvedCues) bool {
	return cues.has("resource", "law") ||
		cues.has("resource", "law_provision")
}

func newContentInput(
	allTerms []string,
	anyTerms []string,
	excludeTerms []string,
	asOf *model.Date,
) (legalquery.LawContentSearchIntentV1, error) {
	input, err := legalquery.NewLawContentSearchIntentV1(
		legalquery.LawContentSearchIntentV1Values{
			AllTerms:     allTerms,
			AnyTerms:     anyTerms,
			ExcludeTerms: excludeTerms,
			AsOf:         asOf,
		},
	)
	if err != nil {
		return legalquery.LawContentSearchIntentV1{}, fmt.Errorf(
			"一般検索語から法令本文検索条件を構築できません: %w",
			err,
		)
	}
	return input, nil
}

func addExplicitSearchEvidence(
	draft *candidateDraft,
	cues resolvedCues,
	explicitResource bool,
) {
	if cues.has("task", "search") {
		draft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
	}
	if explicitResource {
		draft.evidence[legalquery.EvidenceExplicitResource] = struct{}{}
	}
}

func isWeakLawResourceAmbiguity(
	cues resolvedCues,
	hasLawTargets bool,
	conceptCandidateCount int,
) bool {
	return !hasLawTargets &&
		conceptCandidateCount == 0 &&
		cues.has("task", "search") &&
		cues.has("resource", "law") &&
		!cues.has("resource", "law_provision")
}

func buildWeakGeneralDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	terms []legalquery.QueryTermMention,
) ([]candidateDraft, error) {
	if len(terms) != 1 {
		return nil, nil
	}
	term := terms[0]
	asOf := selectedAsOfDate(input, cues, false)
	searchInput, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{
			Query: term.Surface(),
			AsOf:  asOf,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"一般検索語から法令名検索条件を構築できません: %w",
			err,
		)
	}
	contentInput, err := newContentInput(
		[]string{term.Surface()},
		nil,
		nil,
		asOf,
	)
	if err != nil {
		return nil, err
	}

	lawDraft := newCandidateDraft()
	lawDraft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
	lawDraft.evidence[legalquery.EvidenceExplicitResource] = struct{}{}
	lawDraft.evidence[legalquery.EvidenceMorphologicalContext] = struct{}{}
	lawDraft.steps = append(lawDraft.steps, stepDraft{
		startByte: term.Span().StartByte(),
		input:     searchInput,
	})

	contentDraft := newCandidateDraft()
	contentDraft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
	contentDraft.evidence[legalquery.EvidenceGeneralTerm] = struct{}{}
	contentDraft.steps = append(contentDraft.steps, stepDraft{
		startByte: term.Span().StartByte(),
		input:     contentInput,
	})
	return []candidateDraft{lawDraft, contentDraft}, nil
}

func buildDualCandidateDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, error) {
	terms := input.QueryTermMentions()
	if len(terms) != 1 {
		return nil, nil
	}
	term := terms[0]
	asOf := selectedAsOfDate(input, cues, false)
	searchInput, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{
			Query: term.Surface(),
			AsOf:  asOf,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"二候補の法令名検索条件を構築できません: %w",
			err,
		)
	}
	contentInput, err := newContentInput(
		[]string{term.Surface()},
		nil,
		nil,
		asOf,
	)
	if err != nil {
		return nil, err
	}
	baseEvidence := []legalquery.EvidenceCode{
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceGeneralTerm,
	}
	lawDraft := newCandidateDraft()
	contentDraft := newCandidateDraft()
	for _, evidence := range baseEvidence {
		lawDraft.evidence[evidence] = struct{}{}
		contentDraft.evidence[evidence] = struct{}{}
	}
	lawDraft.steps = append(lawDraft.steps, stepDraft{
		startByte: term.Span().StartByte(),
		input:     searchInput,
	})
	contentDraft.steps = append(contentDraft.steps, stepDraft{
		startByte: term.Span().StartByte(),
		input:     contentInput,
	})
	return []candidateDraft{lawDraft, contentDraft}, nil
}
