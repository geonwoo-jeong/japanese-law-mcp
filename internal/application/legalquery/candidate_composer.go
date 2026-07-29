package legalquery

import (
	"fmt"
	"slices"
	"sort"
	"unicode/utf8"
)

const defaultCandidateCompositionVersion = "candidate-composition-2026-07-29-3"

// CandidateComposer は、異なる profile の必須構成員を一つの意味候補へ結合する。
type CandidateComposer struct {
	version string
}

// NewCandidateComposer は、不透明な合成規則版を持つ composer を返す。
func NewCandidateComposer(version string) (CandidateComposer, error) {
	composer := CandidateComposer{version: version}
	if err := composer.Validate(); err != nil {
		return CandidateComposer{}, err
	}
	return composer, nil
}

// Version は、候補合成規則の不透明な版を返す。
func (c CandidateComposer) Version() string {
	return c.version
}

// Validate は、候補合成規則版を確認する。
func (c CandidateComposer) Validate() error {
	if err := validateProfileVersion(c.version); err != nil {
		return fmt.Errorf("compositionVersion が有効ではありません: %w", err)
	}
	return nil
}

type candidateCompositionContribution struct {
	profileOrdinal int
	candidates     []LegalQueryCandidate
	members        []QueryCandidateCompositionMember
	hedgePairs     []CandidateHedgePair
	selectionMode  QuerySelectionMode
}

type candidateCompositionResult struct {
	candidates []LegalQueryCandidate
	hedgePairs []CandidateHedgePair
	constraint QueryCompositionConstraint
}

type candidateCompositionSource struct {
	profileOrdinal int
	candidate      LegalQueryCandidate
	member         QueryCandidateCompositionMember
}

type candidateCompositionSourceCollection struct {
	sources    []candidateCompositionSource
	ineligible bool
}

type positionedCompositionStep struct {
	sourceStartByte int
	profileOrdinal  int
	stepOrdinal     int
	input           LogicalInput
}

func (c CandidateComposer) compose(
	query string,
	contributions []candidateCompositionContribution,
	candidates []LegalQueryCandidate,
	hedgePairs []CandidateHedgePair,
	scorePolicy QueryScorePolicy,
	scope CandidateIDScope,
) (candidateCompositionResult, error) {
	if err := c.Validate(); err != nil {
		return candidateCompositionResult{}, err
	}
	collection, err := collectCandidateCompositionSources(query, contributions)
	if err != nil {
		return candidateCompositionResult{}, err
	}
	constraint := QueryCompositionConstraintNone
	if collection.ineligible {
		constraint = QueryCompositionConstraintIneligible
	}
	sources := collection.sources
	if len(sources) < 2 {
		return newCandidateCompositionResult(
			candidates,
			hedgePairs,
			constraint,
		)
	}

	steps := positionedCandidateCompositionSteps(sources)
	consumed := compositionSourceCandidateIDs(sources)
	if len(steps) > MaxCapabilityCalls {
		return newCandidateCompositionResult(
			removeCandidateIDs(candidates, consumed),
			removeHedgePairsForCandidates(hedgePairs, consumed),
			QueryCompositionConstraintStepLimitExceeded,
		)
	}

	composed, err := assembleComposedCandidate(
		sources,
		steps,
		scorePolicy,
		scope,
	)
	if err != nil {
		return candidateCompositionResult{}, err
	}
	filtered := removeCandidateIDs(candidates, consumed)
	filtered = append(filtered, composed)
	return newCandidateCompositionResult(
		filtered,
		removeHedgePairsForCandidates(hedgePairs, consumed),
		constraint,
	)
}

func collectCandidateCompositionSources(
	query string,
	contributions []candidateCompositionContribution,
) (candidateCompositionSourceCollection, error) {
	result := candidateCompositionSourceCollection{
		sources: make([]candidateCompositionSource, 0, len(contributions)),
	}
	memberProfileCount := 0
	for _, contribution := range contributions {
		if contribution.selectionMode == QuerySelectionModeAutomatic &&
			len(contribution.members) > 0 {
			memberProfileCount++
		}
	}
	compositionRequested := memberProfileCount >= 2
	for _, contribution := range contributions {
		if contribution.selectionMode ==
			QuerySelectionModeClarificationRequired {
			continue
		}
		if len(contribution.members) == 0 {
			continue
		}
		if len(contribution.members) != 1 ||
			len(contribution.candidates) != 1 {
			if compositionRequested {
				result.ineligible = true
			}
			continue
		}
		member := contribution.members[0]
		candidate, exists := candidateByID(
			contribution.candidates,
			member.CandidateID(),
		)
		if !exists {
			return candidateCompositionSourceCollection{}, fmt.Errorf(
				"composition member が同じ contribution の候補を参照していません",
			)
		}
		if compositionMemberBelongsToHedge(
			member,
			contribution.hedgePairs,
		) || slices.Contains(
			candidate.EvidenceCodes(),
			EvidenceGeneralTerm,
		) {
			if compositionRequested {
				result.ineligible = true
			}
			continue
		}
		if err := validateCompositionMemberQueryOrigins(query, member); err != nil {
			if compositionRequested {
				result.ineligible = true
			}
			continue
		}
		result.sources = append(result.sources, candidateCompositionSource{
			profileOrdinal: contribution.profileOrdinal,
			candidate:      candidate,
			member:         member,
		})
	}
	return result, nil
}

func compositionMemberBelongsToHedge(
	member QueryCandidateCompositionMember,
	pairs []CandidateHedgePair,
) bool {
	for _, pair := range pairs {
		if member.CandidateID() == pair.FirstCandidateID() ||
			member.CandidateID() == pair.SecondCandidateID() {
			return true
		}
	}
	return false
}

func candidateByID(
	candidates []LegalQueryCandidate,
	candidateID string,
) (LegalQueryCandidate, bool) {
	for _, candidate := range candidates {
		if candidate.CandidateID() == candidateID {
			return candidate, true
		}
	}
	return LegalQueryCandidate{}, false
}

func validateCompositionMemberQueryOrigins(
	query string,
	member QueryCandidateCompositionMember,
) error {
	for _, origin := range member.StepOrigins() {
		start := origin.SourceStartByte()
		if start < 0 || start >= len(query) {
			return fmt.Errorf("sourceStartByte は照会文の byte 範囲内でなければなりません")
		}
		if !utf8.RuneStart(query[start]) {
			return fmt.Errorf("sourceStartByte は UTF-8 rune の開始境界でなければなりません")
		}
	}
	return nil
}

func positionedCandidateCompositionSteps(
	sources []candidateCompositionSource,
) []positionedCompositionStep {
	result := make([]positionedCompositionStep, 0)
	for _, source := range sources {
		steps := source.candidate.Steps()
		origins := source.member.StepOrigins()
		for index, step := range steps {
			result = append(result, positionedCompositionStep{
				sourceStartByte: origins[index].SourceStartByte(),
				profileOrdinal:  source.profileOrdinal,
				stepOrdinal:     index,
				input:           step.LogicalInput(),
			})
		}
	}
	sort.SliceStable(result, func(left int, right int) bool {
		if result[left].sourceStartByte != result[right].sourceStartByte {
			return result[left].sourceStartByte < result[right].sourceStartByte
		}
		if result[left].profileOrdinal != result[right].profileOrdinal {
			return result[left].profileOrdinal < result[right].profileOrdinal
		}
		return result[left].stepOrdinal < result[right].stepOrdinal
	})
	return result
}

func compositionSourceCandidateIDs(
	sources []candidateCompositionSource,
) map[string]struct{} {
	result := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		result[source.candidate.CandidateID()] = struct{}{}
	}
	return result
}

func assembleComposedCandidate(
	sources []candidateCompositionSource,
	steps []positionedCompositionStep,
	scorePolicy QueryScorePolicy,
	scope CandidateIDScope,
) (LegalQueryCandidate, error) {
	evidence := composedEvidenceCodes(sources)
	score, err := scorePolicy.Score(evidence)
	if err != nil {
		return LegalQueryCandidate{}, fmt.Errorf("合成候補の score を再計算できません: %w", err)
	}
	confidence, err := scorePolicy.ConfidenceFor(score)
	if err != nil {
		return LegalQueryCandidate{}, fmt.Errorf("合成候補の confidence を再計算できません: %w", err)
	}
	concepts, err := composedConceptSources(sources)
	if err != nil {
		return LegalQueryCandidate{}, err
	}
	inputs := make([]LogicalInput, 0, len(steps))
	for _, step := range steps {
		inputs = append(inputs, step.input)
	}
	return AssembleLegalQueryCandidate(CandidateAssemblyValues{
		IDScope:          scope,
		CandidateOrdinal: 1,
		SemanticScore:    score,
		Confidence:       confidence,
		EvidenceCodes:    evidence,
		ConceptSources:   concepts,
		RequiredPacks:    composedRequiredPacks(sources),
		LogicalInputs:    inputs,
	})
}

func composedEvidenceCodes(
	sources []candidateCompositionSource,
) []EvidenceCode {
	present := make(map[EvidenceCode]struct{})
	for _, source := range sources {
		for _, code := range source.candidate.EvidenceCodes() {
			present[code] = struct{}{}
		}
	}
	result := make([]EvidenceCode, 0, len(present))
	for _, code := range []EvidenceCode{
		EvidenceOfficialIdentifier,
		EvidenceStructuredReference,
		EvidenceExplicitTask,
		EvidenceExplicitResource,
		EvidenceOfficialAlias,
		EvidenceLegalConcept,
		EvidenceMorphologicalContext,
		EvidenceUniqueTypoCorrection,
		EvidenceGeneralTerm,
	} {
		if _, exists := present[code]; exists {
			result = append(result, code)
		}
	}
	return result
}

func composedConceptSources(
	sources []candidateCompositionSource,
) ([]LegalConceptSource, error) {
	byID := make(map[string]LegalConceptSource)
	for _, source := range sources {
		for _, current := range source.candidate.ConceptSources() {
			existing, exists := byID[current.ConceptID()]
			if exists && !sameLegalConceptSource(existing, current) {
				return nil, fmt.Errorf(
					"conceptId %q の資料が profile 間で一致しません",
					current.ConceptID(),
				)
			}
			byID[current.ConceptID()] = current
		}
	}
	ids := make([]string, 0, len(byID))
	for conceptID := range byID {
		ids = append(ids, conceptID)
	}
	slices.Sort(ids)
	result := make([]LegalConceptSource, 0, len(ids))
	for _, conceptID := range ids {
		result = append(result, byID[conceptID])
	}
	return result, nil
}

func sameLegalConceptSource(
	left LegalConceptSource,
	right LegalConceptSource,
) bool {
	return left.ConceptID() == right.ConceptID() &&
		left.Title() == right.Title() &&
		left.URL() == right.URL() &&
		left.ConfirmedOn().String() == right.ConfirmedOn().String()
}

func composedRequiredPacks(
	sources []candidateCompositionSource,
) []string {
	present := make(map[string]struct{})
	for _, source := range sources {
		for _, packID := range source.candidate.RequiredPacks() {
			present[packID] = struct{}{}
		}
	}
	result := make([]string, 0, len(present))
	for packID := range present {
		result = append(result, packID)
	}
	slices.Sort(result)
	return result
}

func removeCandidateIDs(
	candidates []LegalQueryCandidate,
	removed map[string]struct{},
) []LegalQueryCandidate {
	result := make([]LegalQueryCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if _, exists := removed[candidate.CandidateID()]; !exists {
			result = append(result, candidate)
		}
	}
	return result
}

func removeHedgePairsForCandidates(
	pairs []CandidateHedgePair,
	removed map[string]struct{},
) []CandidateHedgePair {
	result := make([]CandidateHedgePair, 0, len(pairs))
	for _, pair := range pairs {
		_, firstRemoved := removed[pair.FirstCandidateID()]
		_, secondRemoved := removed[pair.SecondCandidateID()]
		if !firstRemoved && !secondRemoved {
			result = append(result, pair)
		}
	}
	return result
}

func newCandidateCompositionResult(
	candidates []LegalQueryCandidate,
	hedgePairs []CandidateHedgePair,
	constraint QueryCompositionConstraint,
) (candidateCompositionResult, error) {
	if len(candidates) > MaxRankedCandidates {
		return candidateCompositionResult{}, fmt.Errorf(
			"合成後の candidates は %d 件以下でなければなりません",
			MaxRankedCandidates,
		)
	}
	if err := constraint.Validate(); err != nil {
		return candidateCompositionResult{}, err
	}
	return candidateCompositionResult{
		candidates: append([]LegalQueryCandidate(nil), candidates...),
		hedgePairs: append([]CandidateHedgePair(nil), hedgePairs...),
		constraint: constraint,
	}, nil
}
