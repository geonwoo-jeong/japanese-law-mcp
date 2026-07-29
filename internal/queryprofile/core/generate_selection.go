package core

import (
	"fmt"
	"strconv"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

func (p *Profile) selectionMode(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	candidates []legalquery.LegalQueryCandidate,
) legalquery.QuerySelectionMode {
	switch {
	case hasCollidingLawAlias(input.LawNameMentions()):
		return legalquery.QuerySelectionModeClarificationRequired
	case len(candidates) > 0 && p.hasUnresolvedNoAutoExecuteConcept(
		p.selectedCoreConceptMentions(input, cues),
		cues,
	):
		return legalquery.QuerySelectionModeClarificationRequired
	case p.hasWeakGeneralAmbiguity(input, cues):
		return legalquery.QuerySelectionModeClarificationRequired
	case isCoreResourceChoice(input, cues):
		return legalquery.QuerySelectionModeClarificationRequired
	case hasTooManySeparatedSubjects(input, cues):
		return legalquery.QuerySelectionModeClarificationRequired
	case len(candidates) == 0 && hasSeparatedSubjectEvidence(input, cues):
		return legalquery.QuerySelectionModeClarificationRequired
	default:
		return legalquery.QuerySelectionModeAutomatic
	}
}

func (p *Profile) hedgePairs(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	candidates []legalquery.LegalQueryCandidate,
	mode legalquery.QuerySelectionMode,
) ([]legalquery.CandidateHedgePair, error) {
	if mode != legalquery.QuerySelectionModeAutomatic ||
		len(candidates) != 2 {
		return nil, nil
	}
	if !cues.has("operator", "dual_candidate") &&
		!p.hasTwoIndependentSingleConceptCandidates(candidates) {
		return nil, nil
	}
	pair, err := legalquery.NewCandidateHedgePair(
		legalquery.CandidateHedgePairValues{
			FirstCandidateID:  candidates[0].CandidateID(),
			SecondCandidateID: candidates[1].CandidateID(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("core profile の hedge pair を作成できません: %w", err)
	}
	return []legalquery.CandidateHedgePair{pair}, nil
}

func hasCollidingLawAlias(
	mentions []legalquery.LawNameMention,
) bool {
	bySpan := make(map[string]map[string]struct{})
	for _, mention := range mentions {
		span := mention.Span()
		key := strconv.Itoa(span.StartByte()) + ":" +
			strconv.Itoa(span.EndByte())
		if bySpan[key] == nil {
			bySpan[key] = make(map[string]struct{})
		}
		bySpan[key][mention.LawID()] = struct{}{}
		if len(bySpan[key]) > 1 {
			return true
		}
	}
	return false
}

func (p *Profile) hasUnresolvedNoAutoExecuteConcept(
	mentions []legalquery.LegalConceptMention,
	cues resolvedCues,
) bool {
	for _, mention := range mentions {
		definition, exists := p.concepts[mention.ConceptID()]
		if exists && definition.entry.SelectionPolicy ==
			legalconceptlexicon.SelectionPolicyAmbiguousNoAutoExecute {
			if cues.has("resource", "law_provision") &&
				coreCandidateCount(definition) == 1 {
				continue
			}
			return true
		}
	}
	return false
}

func coreCandidateCount(definition conceptDefinition) int {
	count := 0
	for _, candidate := range definition.entry.Candidates {
		if isCoreConceptCandidate(candidate) {
			count++
		}
	}
	return count
}

func (p *Profile) hasWeakGeneralAmbiguity(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	return isWeakLawResourceAmbiguity(
		cues,
		len(buildLawTargets(input)) > 0,
		p.coreConceptCandidateCount(input.LegalConceptMentions()),
		len(coreContentQueryTerms(input, cues)),
	)
}

func (p *Profile) coreConceptCandidateCount(
	mentions []legalquery.LegalConceptMention,
) int {
	count := 0
	for _, mention := range mentions {
		definition, exists := p.concepts[mention.ConceptID()]
		if !exists {
			continue
		}
		for _, candidate := range definition.entry.Candidates {
			if isCoreConceptCandidate(candidate) {
				count++
			}
		}
	}
	return count
}

func (p *Profile) hasTwoIndependentSingleConceptCandidates(
	candidates []legalquery.LegalQueryCandidate,
) bool {
	if len(candidates) != 2 {
		return false
	}
	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		sources := candidate.ConceptSources()
		if len(sources) == 0 {
			return false
		}
		for _, source := range sources {
			definition, exists := p.concepts[source.ConceptID()]
			if !exists ||
				definition.entry.SelectionPolicy !=
					legalconceptlexicon.SelectionPolicySingleCandidate {
				return false
			}
			seen[source.ConceptID()] = struct{}{}
		}
	}
	return len(seen) >= 2
}

func hasTooManySeparatedSubjects(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	if !separatesSubjects(cues) {
		return false
	}
	return separatedSubjectCount(input) > 4
}

func separatedSubjectCount(
	input legalquery.CandidateGenerationInput,
) int {
	positions := make(map[string]struct{})
	for _, mention := range input.LegalConceptMentions() {
		addSubjectSpan(positions, mention.Span())
	}
	for _, mention := range input.QueryTermMentions() {
		addSubjectSpan(positions, mention.Span())
	}
	for _, mention := range input.LawNameMentions() {
		addSubjectSpan(positions, mention.Span())
	}
	for _, mention := range input.IdentifierMentions() {
		addSubjectSpan(positions, mention.Span())
	}
	return len(positions)
}

func addSubjectSpan(
	positions map[string]struct{},
	span legalquery.QuerySpan,
) {
	key := strconv.Itoa(span.StartByte()) + ":" +
		strconv.Itoa(span.EndByte())
	positions[key] = struct{}{}
}

func hasSeparatedSubjectEvidence(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	if !separatesSubjects(cues) {
		return false
	}
	return separatedSubjectCount(input) >= 2
}
