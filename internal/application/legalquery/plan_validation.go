package legalquery

import (
	"fmt"
	"slices"
)

type rankedCandidateReference struct {
	index     int
	candidate LegalQueryCandidate
}

func validatePlanDefinition(plan LegalQueryPlan) error {
	if plan.language != legalQueryLanguage {
		return fmt.Errorf("language は ja でなければなりません")
	}
	if err := validateProfileVersion(plan.profileVersion); err != nil {
		return err
	}
	if !isPlanDecision(plan.decision) {
		return fmt.Errorf("decision が定義されていません")
	}
	references, err := validateRankedCandidates(plan.rankedCandidates)
	if err != nil {
		return err
	}
	if err := validatePlanSelections(plan.selected, references); err != nil {
		return err
	}
	if err := validateDecisionShape(plan.decision, plan.selected); err != nil {
		return err
	}
	if err := validateReasonCodes(plan.decision, plan.reasonCodes); err != nil {
		return err
	}
	if slices.Equal(
		plan.reasonCodes,
		[]ReasonCode{ReasonCodeStandaloneStructuredQuery},
	) && len(plan.rankedCandidates) != 0 {
		return fmt.Errorf("standalone structured query は rankedCandidates を持てません")
	}
	return nil
}

func validateRankedCandidates(
	values []LegalQueryCandidate,
) (map[string]rankedCandidateReference, error) {
	if len(values) > MaxRankedCandidates {
		return nil, fmt.Errorf("rankedCandidates は十六件以下でなければなりません")
	}
	references := make(
		map[string]rankedCandidateReference,
		len(values),
	)
	stepIDs := make(map[string]struct{})
	for index, candidate := range values {
		if err := candidate.Validate(); err != nil {
			return nil, fmt.Errorf(
				"rankedCandidates[%d] が有効ではありません: %w",
				index,
				err,
			)
		}
		if index > 0 &&
			values[index-1].SemanticScore() < candidate.SemanticScore() {
			return nil, fmt.Errorf("rankedCandidates は semanticScore の非増加順でなければなりません")
		}
		if _, exists := references[candidate.CandidateID()]; exists {
			return nil, fmt.Errorf("rankedCandidates の candidateId を重複させることはできません")
		}
		references[candidate.CandidateID()] = rankedCandidateReference{
			index:     index,
			candidate: candidate,
		}
		for _, step := range candidate.Steps() {
			if _, exists := stepIDs[step.StepID()]; exists {
				return nil, fmt.Errorf("rankedCandidates 全体で stepId を重複させることはできません")
			}
			stepIDs[step.StepID()] = struct{}{}
		}
	}
	return references, nil
}

func validatePlanSelections(
	values []LegalQueryPlanSelection,
	references map[string]rankedCandidateReference,
) error {
	if len(values) > MaxSelectedCandidates {
		return fmt.Errorf("selected は二件以下でなければなりません")
	}
	seen := make(map[string]struct{}, len(values))
	previousRank := -1
	selectedSteps := 0
	for index, selection := range values {
		if err := selection.Validate(); err != nil {
			return fmt.Errorf("selected[%d] が有効ではありません: %w", index, err)
		}
		if _, exists := seen[selection.CandidateID()]; exists {
			return fmt.Errorf("selected の candidateId を重複させることはできません")
		}
		seen[selection.CandidateID()] = struct{}{}
		reference, exists := references[selection.CandidateID()]
		if !exists {
			return fmt.Errorf("selected は rankedCandidates の候補だけを参照できます")
		}
		if reference.index <= previousRank {
			return fmt.Errorf("selected は rankedCandidates の意味順位を保持しなければなりません")
		}
		previousRank = reference.index
		if !slices.Equal(
			selection.RequiredPacks(),
			reference.candidate.RequiredPacks(),
		) {
			return fmt.Errorf("selected の requiredPacks が参照先候補と一致しません")
		}
		selectedSteps += len(reference.candidate.Steps())
		if selectedSteps > MaxCapabilityCalls {
			return fmt.Errorf("選択した全候補の step は四つ以下でなければなりません")
		}
	}
	return nil
}

func validateDecisionShape(
	decision PlanDecision,
	selected []LegalQueryPlanSelection,
) error {
	switch decision {
	case PlanDecisionSingle:
		return requireSelectionShape(
			selected,
			1,
			1,
			SelectionAvailabilityAvailable,
		)
	case PlanDecisionHedged:
		return requireSelectionShape(
			selected,
			2,
			2,
			SelectionAvailabilityAvailable,
		)
	case PlanDecisionNeedsClarification:
		return requireSelectionShape(
			selected,
			0,
			2,
			SelectionAvailabilityAvailable,
		)
	case PlanDecisionCapabilityUnavailable:
		return requireSelectionShape(
			selected,
			1,
			2,
			SelectionAvailabilityPackDisabled,
		)
	case PlanDecisionUnsupported:
		return requireSelectionShape(
			selected,
			0,
			0,
			SelectionAvailabilityAvailable,
		)
	default:
		return fmt.Errorf("decision が定義されていません")
	}
}

func requireSelectionShape(
	selected []LegalQueryPlanSelection,
	minimum int,
	maximum int,
	availability SelectionAvailability,
) error {
	if len(selected) < minimum || len(selected) > maximum {
		return fmt.Errorf("selected の件数が decision と一致しません")
	}
	for _, selection := range selected {
		if selection.Availability() != availability {
			return fmt.Errorf("selected の availability が decision と一致しません")
		}
	}
	return nil
}

func validateReasonCodes(
	decision PlanDecision,
	values []ReasonCode,
) error {
	previousRank := -1
	for _, value := range values {
		rank, exists := reasonCodeRank(value)
		if !exists {
			return fmt.Errorf("reasonCodes に定義されていない値があります")
		}
		if rank <= previousRank {
			return fmt.Errorf("reasonCodes は重複させず規定の順序で並べなければなりません")
		}
		previousRank = rank
	}
	switch decision {
	case PlanDecisionSingle:
		return requireExactReason(values, ReasonCodeSingleClearCandidate)
	case PlanDecisionHedged:
		return requireExactReason(values, ReasonCodeHedgedCloseCandidates)
	case PlanDecisionNeedsClarification:
		return requireReasonsFrom(
			values,
			1,
			2,
			ReasonCodeBelowExecutionThreshold,
			ReasonCodeAmbiguousCandidates,
		)
	case PlanDecisionCapabilityUnavailable:
		return requireExactReason(values, ReasonCodeRequiredPackDisabled)
	case PlanDecisionUnsupported:
		if len(values) == 1 &&
			values[0] == ReasonCodeStandaloneStructuredQuery {
			return nil
		}
		return requireReasonsFrom(
			values,
			1,
			3,
			ReasonCodeNonJapaneseQuery,
			ReasonCodeMixedUnsupportedIntent,
			ReasonCodeUnsupportedTaskOrResource,
		)
	default:
		return fmt.Errorf("decision が定義されていません")
	}
}

func requireExactReason(values []ReasonCode, expected ReasonCode) error {
	if len(values) != 1 || values[0] != expected {
		return fmt.Errorf("reasonCodes が decision と一致しません")
	}
	return nil
}

func requireReasonsFrom(
	values []ReasonCode,
	minimum int,
	maximum int,
	allowed ...ReasonCode,
) error {
	if len(values) < minimum || len(values) > maximum {
		return fmt.Errorf("reasonCodes の件数が decision と一致しません")
	}
	for _, value := range values {
		if !slices.Contains(allowed, value) {
			return fmt.Errorf("reasonCodes が decision と一致しません")
		}
	}
	return nil
}

func reasonCodeRank(value ReasonCode) (int, bool) {
	switch value {
	case ReasonCodeSingleClearCandidate:
		return 0, true
	case ReasonCodeHedgedCloseCandidates:
		return 1, true
	case ReasonCodeBelowExecutionThreshold:
		return 2, true
	case ReasonCodeAmbiguousCandidates:
		return 3, true
	case ReasonCodeRequiredPackDisabled:
		return 4, true
	case ReasonCodeNonJapaneseQuery:
		return 5, true
	case ReasonCodeStandaloneStructuredQuery:
		return 6, true
	case ReasonCodeMixedUnsupportedIntent:
		return 7, true
	case ReasonCodeUnsupportedTaskOrResource:
		return 8, true
	default:
		return 0, false
	}
}

func isPlanDecision(value PlanDecision) bool {
	return value == PlanDecisionSingle ||
		value == PlanDecisionHedged ||
		value == PlanDecisionNeedsClarification ||
		value == PlanDecisionCapabilityUnavailable ||
		value == PlanDecisionUnsupported
}
