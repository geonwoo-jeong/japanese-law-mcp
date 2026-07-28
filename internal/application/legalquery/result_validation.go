package legalquery

import (
	"fmt"
)

type plannedResultStep struct {
	interpretationID string
	summary          LegalQueryStepSummary
}

type resultInterpretationShape struct {
	minimum      int
	maximum      int
	availability SelectionAvailability
}

func resultDecisionFor(
	planDecision PlanDecision,
	status LegalQueryResultStatus,
) (LegalQueryResultDecision, error) {
	switch planDecision {
	case PlanDecisionSingle:
		if !isExecutionResultStatus(status) {
			break
		}
		return LegalQueryResultDecisionSingle, nil
	case PlanDecisionHedged:
		if !isExecutionResultStatus(status) {
			break
		}
		return LegalQueryResultDecisionHedged, nil
	case PlanDecisionNeedsClarification:
		if status == LegalQueryResultStatusNeedsClarification {
			return LegalQueryResultDecisionNoExecution, nil
		}
	case PlanDecisionCapabilityUnavailable:
		if status == LegalQueryResultStatusCapabilityUnavailable {
			return LegalQueryResultDecisionNoExecution, nil
		}
	case PlanDecisionUnsupported:
		if status == LegalQueryResultStatusUnsupported {
			return LegalQueryResultDecisionNoExecution, nil
		}
	}
	return "", fmt.Errorf("plan.decision と result.status の組合せが許可されていません")
}

func isExecutionResultStatus(status LegalQueryResultStatus) bool {
	return status == LegalQueryResultStatusCompleted ||
		status == LegalQueryResultStatusEmpty ||
		status == LegalQueryResultStatusPartial
}

func validateResultInterpretations(
	status LegalQueryResultStatus,
	decision LegalQueryResultDecision,
	values []LegalQueryInterpretation,
) error {
	shape, err := interpretationShape(
		status,
		decision,
	)
	if err != nil {
		return err
	}
	if len(values) < shape.minimum || len(values) > shape.maximum {
		return fmt.Errorf("interpretations の件数が status と decision に一致しません")
	}
	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		if err := value.Validate(); err != nil {
			return fmt.Errorf(
				"interpretations[%d] が有効ではありません: %w",
				index,
				err,
			)
		}
		expectedID := interpretationIDForIndex(index)
		if value.InterpretationID() != expectedID {
			return fmt.Errorf("interpretationId は selection 順の固定値でなければなりません")
		}
		if _, exists := seen[value.InterpretationID()]; exists {
			return fmt.Errorf("interpretationId を重複させることはできません")
		}
		seen[value.InterpretationID()] = struct{}{}
		if value.Availability() != shape.availability {
			return fmt.Errorf("interpretation.availability が result.status と一致しません")
		}
	}
	return nil
}

func interpretationShape(
	status LegalQueryResultStatus,
	decision LegalQueryResultDecision,
) (resultInterpretationShape, error) {
	switch status {
	case LegalQueryResultStatusCompleted,
		LegalQueryResultStatusEmpty,
		LegalQueryResultStatusPartial:
		switch decision {
		case LegalQueryResultDecisionSingle:
			return resultInterpretationShape{
				minimum:      1,
				maximum:      1,
				availability: SelectionAvailabilityAvailable,
			}, nil
		case LegalQueryResultDecisionHedged:
			return resultInterpretationShape{
				minimum:      2,
				maximum:      2,
				availability: SelectionAvailabilityAvailable,
			}, nil
		default:
			return resultInterpretationShape{},
				fmt.Errorf("実行結果の decision が定義されていません")
		}
	case LegalQueryResultStatusNeedsClarification:
		if decision == LegalQueryResultDecisionNoExecution {
			return resultInterpretationShape{
				minimum:      0,
				maximum:      2,
				availability: SelectionAvailabilityAvailable,
			}, nil
		}
	case LegalQueryResultStatusCapabilityUnavailable:
		if decision == LegalQueryResultDecisionNoExecution {
			return resultInterpretationShape{
				minimum:      1,
				maximum:      2,
				availability: SelectionAvailabilityPackDisabled,
			}, nil
		}
	case LegalQueryResultStatusUnsupported:
		if decision == LegalQueryResultDecisionNoExecution {
			return resultInterpretationShape{
				minimum:      0,
				maximum:      0,
				availability: SelectionAvailabilityAvailable,
			}, nil
		}
	}
	return resultInterpretationShape{},
		fmt.Errorf("status と decision の組合せが許可されていません")
}

func validateResultAttempts(
	status LegalQueryResultStatus,
	interpretations []LegalQueryInterpretation,
	attempts []LegalQueryAttempt,
) error {
	if len(attempts) > MaxCapabilityCalls {
		return fmt.Errorf("attempts は四件以下でなければなりません")
	}
	if !isExecutionResultStatus(status) {
		if len(attempts) != 0 {
			return fmt.Errorf("非実行結果は attempts を持てません")
		}
		return nil
	}
	if len(attempts) == 0 {
		return fmt.Errorf("実行結果には attempt が一件以上必要です")
	}
	if err := validateAttemptPlanOrder(interpretations, attempts); err != nil {
		return err
	}
	var completed int
	var empty int
	var failed int
	for _, attempt := range attempts {
		switch attempt.Outcome() {
		case LegalQueryAttemptOutcomeCompleted:
			completed++
		case LegalQueryAttemptOutcomeEmpty:
			empty++
		case LegalQueryAttemptOutcomeFailed:
			failed++
		default:
			return fmt.Errorf("attempt.outcome が定義されていません")
		}
	}
	switch status {
	case LegalQueryResultStatusCompleted:
		if completed < 1 || failed != 0 {
			return fmt.Errorf("completed には非空成功が一件以上あり、失敗があってはなりません")
		}
	case LegalQueryResultStatusEmpty:
		if empty != len(attempts) {
			return fmt.Errorf("empty の全 attempt は正常な空でなければなりません")
		}
	case LegalQueryResultStatusPartial:
		if len(attempts) < 2 || failed < 1 || completed+empty < 1 {
			return fmt.Errorf("partial には成功と失敗が一件以上ずつ必要です")
		}
	default:
		return fmt.Errorf("実行結果の status が定義されていません")
	}
	return nil
}

func validateAttemptPlanOrder(
	interpretations []LegalQueryInterpretation,
	attempts []LegalQueryAttempt,
) error {
	planned := make([]plannedResultStep, 0, MaxCapabilityCalls)
	for _, interpretation := range interpretations {
		for _, summary := range interpretation.steps {
			planned = append(planned, plannedResultStep{
				interpretationID: interpretation.InterpretationID(),
				summary:          summary,
			})
		}
	}
	nextPlanned := 0
	seenSteps := make(map[string]struct{}, len(attempts))
	for index, attempt := range attempts {
		if isNilLegalQueryAttempt(attempt) {
			return fmt.Errorf("attempts[%d] は nil にできません", index)
		}
		if err := attempt.Validate(); err != nil {
			return fmt.Errorf("attempts[%d] が有効ではありません: %w", index, err)
		}
		if _, exists := seenSteps[attempt.StepID()]; exists {
			return fmt.Errorf("同じ step の attempt を重複させることはできません")
		}
		seenSteps[attempt.StepID()] = struct{}{}
		matched := false
		for nextPlanned < len(planned) {
			current := planned[nextPlanned]
			nextPlanned++
			if attempt.InterpretationID() == current.interpretationID &&
				attempt.StepID() == current.summary.StepID() &&
				attempt.CapabilityID() == current.summary.CapabilityID() &&
				attempt.CapabilityMajorVersion() == current.summary.CapabilityMajorVersion() {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf(
				"attempts[%d] は interpretation と step の計画順の部分列ではありません",
				index,
			)
		}
	}
	return nil
}

func deriveLegalQueryNotices(
	status LegalQueryResultStatus,
	attempts []LegalQueryAttempt,
	reasonCodes []ReasonCode,
) []string {
	notices := make([]string, 0, 3)
	if isExecutionResultStatus(status) {
		if hasMoreAttempt(attempts) {
			notices = append(notices, legalQueryFirstPageNotice)
		}
		if len(attempts) >= 2 {
			notices = append(notices, legalQuerySeparateAttemptsNotice)
		}
		if status == LegalQueryResultStatusPartial {
			notices = append(notices, legalQueryPartialFailureNotice)
		}
		return notices
	}
	if status == LegalQueryResultStatusCapabilityUnavailable {
		return append(notices, legalQueryPackDisabledNotice)
	}
	if status != LegalQueryResultStatusUnsupported {
		return notices
	}
	for _, reasonCode := range reasonCodes {
		switch reasonCode {
		case ReasonCodeNonJapaneseQuery:
			notices = append(notices, legalQueryNonJapaneseNotice)
		case ReasonCodeStandaloneStructuredQuery:
			notices = append(notices, legalQueryStandaloneStructuredNotice)
		case ReasonCodeMixedUnsupportedIntent:
			notices = append(notices, legalQueryMixedUnsupportedNotice)
		case ReasonCodeUnsupportedTaskOrResource:
			notices = append(notices, legalQueryUnsupportedScopeNotice)
		}
	}
	return notices
}

func hasMoreAttempt(attempts []LegalQueryAttempt) bool {
	for _, attempt := range attempts {
		var page LegalQueryPagePreview
		switch typed := attempt.(type) {
		case LegalQueryLawSearchAttempt:
			page = typed.Page()
		case LegalQueryLawContentSearchAttempt:
			page = typed.Page()
		case LegalQueryLawUpdatesAttempt:
			page = typed.Page()
		case LegalQueryJudicialSearchAttempt:
			page = typed.Page()
		default:
			continue
		}
		hasMore, known := page.HasMore()
		if known && hasMore {
			return true
		}
	}
	return false
}
