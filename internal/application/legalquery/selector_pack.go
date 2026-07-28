package legalquery

import "fmt"

func validateCandidatePackAdoption(
	candidates []LegalQueryCandidate,
	state PackState,
) error {
	for _, candidate := range candidates {
		for _, packID := range candidate.RequiredPacks() {
			_, adopted := state.State(packID)
			if !adopted {
				return fmt.Errorf(
					"candidate %q が未採用の required pack %q を要求しています",
					candidate.CandidateID(),
					packID,
				)
			}
		}
	}
	return nil
}

func applyPackAvailability(
	semantic semanticSelection,
	state PackState,
) (
	PlanDecision,
	[]LegalQueryPlanSelection,
	[]ReasonCode,
	error,
) {
	if semantic.decision == PlanDecisionUnsupported {
		return semantic.decision, nil, semantic.reasons, nil
	}
	selections := make(
		[]LegalQueryPlanSelection,
		0,
		len(semantic.candidates),
	)
	disabled := make(
		[]LegalQueryPlanSelection,
		0,
		len(semantic.candidates),
	)
	for _, candidate := range semantic.candidates {
		selection, err := candidatePlanSelection(candidate, state)
		if err != nil {
			return "", nil, nil, err
		}
		selections = append(selections, selection)
		if selection.Availability() ==
			SelectionAvailabilityPackDisabled {
			disabled = append(disabled, selection)
		}
	}
	if len(disabled) > 0 {
		return PlanDecisionCapabilityUnavailable,
			disabled,
			[]ReasonCode{ReasonCodeRequiredPackDisabled},
			nil
	}
	return semantic.decision, selections, semantic.reasons, nil
}

func candidatePlanSelection(
	candidate LegalQueryCandidate,
	state PackState,
) (LegalQueryPlanSelection, error) {
	availability := SelectionAvailabilityAvailable
	for _, packID := range candidate.RequiredPacks() {
		enabled, adopted := state.State(packID)
		if !adopted {
			return LegalQueryPlanSelection{}, fmt.Errorf(
				"未採用の required pack %q です",
				packID,
			)
		}
		if !enabled {
			availability = SelectionAvailabilityPackDisabled
		}
	}
	return NewLegalQueryPlanSelection(LegalQueryPlanSelectionValues{
		CandidateID:   candidate.CandidateID(),
		Availability:  availability,
		RequiredPacks: candidate.RequiredPacks(),
	})
}
