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
	terms := input.QueryTermMentions()
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

func (p *Profile) buildConceptCandidates(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, error) {
	result := make([]candidateDraft, 0)
	asOf := selectedAsOfDate(input, cues, false)
	for _, mention := range input.LegalConceptMentions() {
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
				hasLegalResourceCue(cues),
			)
			if mention.MatchKind() ==
				legalquery.PreprocessMatchUniqueTypoCorrection {
				draft.evidence[legalquery.EvidenceUniqueTypoCorrection] =
					struct{}{}
			}
			draft.concepts = append(draft.concepts, definition.source)
			draft.steps = append(draft.steps, stepDraft{
				startByte: mention.Span().StartByte(),
				input:     contentInput,
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
	options := make(map[int][]subjectOption)
	for _, concept := range concepts {
		if err := addSubjectOption(options, concept); err != nil {
			return nil, false, err
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
			return nil, false, err
		}
		draft := newCandidateDraft()
		addExplicitSearchEvidence(
			&draft,
			cues,
			hasLegalResourceCue(cues),
		)
		draft.evidence[legalquery.EvidenceMorphologicalContext] = struct{}{}
		draft.steps = append(draft.steps, stepDraft{
			startByte: term.Span().StartByte(),
			input:     contentInput,
		})
		if err := addSubjectOption(options, draft); err != nil {
			return nil, false, err
		}
	}
	if len(options) < 2 {
		return nil, false, nil
	}
	if len(options) > 4 {
		return nil, true, nil
	}

	positions := make([]int, 0, len(options))
	for position := range options {
		positions = append(positions, position)
	}
	sort.Ints(positions)
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
				startByte: term.Span().StartByte(),
				input:     contentInput,
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
		startByte: ordered[0].Span().StartByte(),
		input:     contentInput,
	})
	return &draft, nil
}

func partitionSearchTerms(
	terms []legalquery.QueryTermMention,
	cues resolvedCues,
) ([]string, []string, []string) {
	values := make([]string, 0, len(terms))
	for _, term := range terms {
		values = append(values, term.Surface())
	}
	if cues.has("operator", "any") {
		return nil, values, nil
	}
	if cues.has("operator", "exclude") && len(values) > 1 {
		return values[:len(values)-1], nil, values[len(values)-1:]
	}
	return values, nil, nil
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
