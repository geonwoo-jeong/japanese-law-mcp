package legalquery

import (
	"fmt"
	"slices"
)

// SelectorInput は、意味順位、pack 状態および確定済み取得上限を保持する。
type SelectorInput struct {
	ProfileSetResult QueryProfileSetResult
	PackState        PackState
	LimitPerAttempt  int
}

type semanticSelection struct {
	decision   PlanDecision
	candidates []LegalQueryCandidate
	reasons    []ReasonCode
}

// SelectLegalQueryPlan は、意味選択後に pack 可否と固定予算を確定する。
func SelectLegalQueryPlan(input SelectorInput) (LegalQueryPlan, error) {
	if err := input.ProfileSetResult.validate(); err != nil {
		return LegalQueryPlan{}, fmt.Errorf(
			"profile set result が有効ではありません: %w",
			err,
		)
	}
	if isNilInterfaceValue(input.PackState) {
		return LegalQueryPlan{}, fmt.Errorf("packState は必須です")
	}
	candidates := input.ProfileSetResult.RankedCandidates()
	if err := validateCandidatePackAdoption(candidates, input.PackState); err != nil {
		return LegalQueryPlan{}, err
	}
	semantic := selectUnsupported(
		candidates,
		input.ProfileSetResult.Signals(),
	)
	if semantic == nil {
		value := selectSemanticCandidates(input.ProfileSetResult)
		semantic = &value
	}
	decision, selections, reasons, err := applyPackAvailability(
		*semantic,
		input.PackState,
	)
	if err != nil {
		return LegalQueryPlan{}, err
	}
	return NewLegalQueryPlan(LegalQueryPlanValues{
		ProfileVersion:   input.ProfileSetResult.ProfileVersion(),
		Decision:         decision,
		RankedCandidates: candidates,
		Selected:         selections,
		ReasonCodes:      reasons,
		LimitPerAttempt:  input.LimitPerAttempt,
	})
}

func selectUnsupported(
	candidates []LegalQueryCandidate,
	signals []CandidateGenerationSignal,
) *semanticSelection {
	if slices.Contains(
		signals,
		CandidateSignalStandaloneStructuredQuery,
	) {
		return &semanticSelection{
			decision: PlanDecisionUnsupported,
			reasons: []ReasonCode{
				ReasonCodeStandaloneStructuredQuery,
			},
		}
	}
	if slices.Contains(signals, CandidateSignalNonJapaneseQuery) {
		return &semanticSelection{
			decision: PlanDecisionUnsupported,
			reasons:  []ReasonCode{ReasonCodeNonJapaneseQuery},
		}
	}
	hasAdvice := slices.Contains(
		signals,
		CandidateSignalUnsupportedLegalAdvice,
	)
	hasTranslation := slices.Contains(
		signals,
		CandidateSignalUnsupportedTranslation,
	)
	hasUnsupportedTarget := slices.Contains(
		signals,
		CandidateSignalUnsupportedTaskOrResource,
	)
	if !hasAdvice && !hasTranslation && !hasUnsupportedTarget {
		return nil
	}
	reasons := make([]ReasonCode, 0, 2)
	if len(candidates) > 0 {
		reasons = append(reasons, ReasonCodeMixedUnsupportedIntent)
	} else {
		reasons = append(reasons, ReasonCodeUnsupportedTaskOrResource)
	}
	if len(candidates) > 0 && hasUnsupportedTarget {
		reasons = append(reasons, ReasonCodeUnsupportedTaskOrResource)
	}
	return &semanticSelection{
		decision: PlanDecisionUnsupported,
		reasons:  reasons,
	}
}

func selectSemanticCandidates(
	result QueryProfileSetResult,
) semanticSelection {
	candidates := result.RankedCandidates()
	policy := result.selectionPolicy()
	if result.SelectionMode() ==
		QuerySelectionModeClarificationRequired {
		return clarificationSelection(candidates, policy, true)
	}
	if len(candidates) == 0 {
		return clarificationSelection(candidates, policy, false)
	}
	if isSingleCandidate(candidates, policy) {
		return semanticSelection{
			decision:   PlanDecisionSingle,
			candidates: candidates[:1],
			reasons:    []ReasonCode{ReasonCodeSingleClearCandidate},
		}
	}
	if isHedgedCandidatePair(
		candidates,
		result.HedgePairs(),
		policy,
	) {
		return semanticSelection{
			decision:   PlanDecisionHedged,
			candidates: candidates[:2],
			reasons:    []ReasonCode{ReasonCodeHedgedCloseCandidates},
		}
	}
	return clarificationSelection(candidates, policy, false)
}

func isSingleCandidate(
	candidates []LegalQueryCandidate,
	policy QuerySelectionPolicy,
) bool {
	if len(candidates) == 0 ||
		candidates[0].SemanticScore() < policy.SingleThreshold() {
		return false
	}
	return len(candidates) == 1 ||
		candidates[0].SemanticScore()-candidates[1].SemanticScore() >=
			policy.SingleMargin()
}

func isHedgedCandidatePair(
	candidates []LegalQueryCandidate,
	pairs []CandidateHedgePair,
	policy QuerySelectionPolicy,
) bool {
	if len(candidates) < 2 ||
		candidates[0].SemanticScore() <
			policy.MinimumExecutionThreshold() ||
		candidates[1].SemanticScore() <
			policy.MinimumExecutionThreshold() ||
		candidates[0].SemanticScore()-candidates[1].SemanticScore() >
			policy.HedgeMargin() {
		return false
	}
	if len(candidates) > 2 &&
		candidates[2].SemanticScore() >=
			policy.MinimumExecutionThreshold() {
		return false
	}
	for _, pair := range pairs {
		if pair.FirstCandidateID() == candidates[0].CandidateID() &&
			pair.SecondCandidateID() == candidates[1].CandidateID() {
			return true
		}
	}
	return false
}

func clarificationSelection(
	candidates []LegalQueryCandidate,
	policy QuerySelectionPolicy,
	forced bool,
) semanticSelection {
	reasons := clarificationReasonCodes(candidates, policy, forced)
	return semanticSelection{
		decision: PlanDecisionNeedsClarification,
		candidates: clarificationDisplayCandidates(
			candidates,
			policy.MinimumExecutionThreshold(),
		),
		reasons: reasons,
	}
}

func clarificationReasonCodes(
	candidates []LegalQueryCandidate,
	policy QuerySelectionPolicy,
	forced bool,
) []ReasonCode {
	reasons := make([]ReasonCode, 0, 2)
	below := len(candidates) == 0 ||
		candidates[0].SemanticScore() <
			policy.MinimumExecutionThreshold() ||
		len(candidates) > 1 &&
			candidates[1].SemanticScore() <
				policy.MinimumExecutionThreshold()
	if below {
		reasons = append(reasons, ReasonCodeBelowExecutionThreshold)
	}
	if forced || len(candidates) > 0 &&
		candidates[0].SemanticScore() >=
			policy.MinimumExecutionThreshold() {
		reasons = append(reasons, ReasonCodeAmbiguousCandidates)
	}
	return reasons
}

func clarificationDisplayCandidates(
	candidates []LegalQueryCandidate,
	minimum int,
) []LegalQueryCandidate {
	if len(candidates) == 0 ||
		candidates[0].SemanticScore() < minimum {
		return nil
	}
	if len(candidates) == 1 {
		return candidates[:1]
	}
	if candidates[1].SemanticScore() < minimum {
		return nil
	}
	return candidates[:min(len(candidates), MaxSelectedCandidates)]
}
