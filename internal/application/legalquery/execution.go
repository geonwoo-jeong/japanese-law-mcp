package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// LegalQueryExecutionValues は、確定済み plan と実行済み attempt の作成値を保持する。
type LegalQueryExecutionValues struct {
	Plan     LegalQueryPlan
	Attempts []LegalQueryAttempt
}

// LegalQueryExecution は、計画順の実行済み attempt を不変に保持する。
type LegalQueryExecution struct {
	attempts    []LegalQueryAttempt
	initialized bool
}

// NewLegalQueryExecution は、attempt が plan 順の部分列であることを確認して複製する。
func NewLegalQueryExecution(
	values LegalQueryExecutionValues,
) (LegalQueryExecution, error) {
	if err := values.Plan.Validate(); err != nil {
		return LegalQueryExecution{}, fmt.Errorf(
			"plan が有効ではありません: %w",
			err,
		)
	}
	interpretations, err := NewLegalQueryInterpretations(values.Plan)
	if err != nil {
		return LegalQueryExecution{}, fmt.Errorf(
			"interpretations を作成できません: %w",
			err,
		)
	}
	attempts, err := cloneLegalQueryAttempts(values.Attempts)
	if err != nil {
		return LegalQueryExecution{}, err
	}
	if err := validateAttemptPlanOrder(interpretations, attempts); err != nil {
		return LegalQueryExecution{}, err
	}
	execution := LegalQueryExecution{
		attempts:    attempts,
		initialized: true,
	}
	if err := execution.Validate(); err != nil {
		return LegalQueryExecution{}, err
	}
	return execution, nil
}

// Attempts は、計画順の attempt の複製を返す。
func (e LegalQueryExecution) Attempts() []LegalQueryAttempt {
	attempts, err := cloneLegalQueryAttempts(e.attempts)
	if err != nil {
		panic(fmt.Sprintf("検証済み execution の複製に失敗しました: %v", err))
	}
	return attempts
}

// Validate は、attempt 自体の件数、interpretation 順、重複および全失敗禁止を確認する。
func (e LegalQueryExecution) Validate() error {
	if !e.initialized {
		return fmt.Errorf(
			"LegalQueryExecution は NewLegalQueryExecution で作成しなければなりません",
		)
	}
	if len(e.attempts) < 1 || len(e.attempts) > MaxCapabilityCalls {
		return fmt.Errorf("attempts は一件以上四件以下でなければなりません")
	}
	seenSteps := make(map[string]struct{}, len(e.attempts))
	previousInterpretation := ""
	successes := 0
	for index, attempt := range e.attempts {
		if isNilLegalQueryAttempt(attempt) {
			return fmt.Errorf("attempts[%d] は nil にできません", index)
		}
		if err := attempt.Validate(); err != nil {
			return fmt.Errorf(
				"attempts[%d] が有効ではありません: %w",
				index,
				err,
			)
		}
		if _, exists := seenSteps[attempt.StepID()]; exists {
			return fmt.Errorf("同じ step の attempt を重複させることはできません")
		}
		seenSteps[attempt.StepID()] = struct{}{}
		if previousInterpretation == "interpretation-2" &&
			attempt.InterpretationID() == "interpretation-1" {
			return fmt.Errorf("attempts は interpretation の計画順でなければなりません")
		}
		previousInterpretation = attempt.InterpretationID()
		switch attempt.Outcome() {
		case LegalQueryAttemptOutcomeCompleted, LegalQueryAttemptOutcomeEmpty:
			successes++
		case LegalQueryAttemptOutcomeFailed:
		default:
			return fmt.Errorf("attempts[%d].outcome が定義されていません", index)
		}
	}
	if successes == 0 {
		return fmt.Errorf("全 step 失敗を execution artifact にできません")
	}
	return nil
}

// UnmarshalJSON は、executor を介さない直接復元を拒否する。
func (*LegalQueryExecution) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryExecution は JSON から直接復元できません。NewLegalQueryExecution を使用してください",
	)
}

// LegalQueryAllFailedError は、全 step 失敗時の最初の公開エラーを保持する。
type LegalQueryAllFailedError struct {
	result      model.ErrorResult
	initialized bool
}

func newLegalQueryAllFailedError(
	result model.ErrorResult,
) (LegalQueryAllFailedError, error) {
	allFailed := LegalQueryAllFailedError{
		result:      result,
		initialized: true,
	}
	if err := allFailed.Validate(); err != nil {
		return LegalQueryAllFailedError{}, err
	}
	return allFailed, nil
}

// Error は、情報源の生エラーを含まない固定の公開メッセージを返す。
func (e LegalQueryAllFailedError) Error() string {
	if err := e.Validate(); err != nil {
		return "統合法情報照会の全 step が失敗しました"
	}
	return e.result.Message()
}

// ErrorResult は、計画順で最初の失敗に対応する公開エラーを返す。
func (e LegalQueryAllFailedError) ErrorResult() model.ErrorResult {
	return e.result
}

// Validate は、公開エラーが初期化済みであることを確認する。
func (e LegalQueryAllFailedError) Validate() error {
	if !e.initialized {
		return fmt.Errorf("LegalQueryAllFailedError が初期化されていません")
	}
	if err := e.result.Validate(); err != nil {
		return fmt.Errorf("errorResult が有効ではありません: %w", err)
	}
	if !isAllowedFailedAttemptError(e.result.Code()) {
		return fmt.Errorf("errorResult.code は全失敗で公開できません")
	}
	return nil
}
