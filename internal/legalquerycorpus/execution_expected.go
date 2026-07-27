package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	maximumExecutionExpectedAttempts  = 4
	maximumExecutionReturnedItemCount = 40
)

// ExecutionExpectedTerminal は、execution fixture の終端 variant を表す。
type ExecutionExpectedTerminal string

const (
	// ExecutionExpectedTerminalResult は、型付き結果の返却を表す。
	ExecutionExpectedTerminalResult ExecutionExpectedTerminal = "result"
	// ExecutionExpectedTerminalError は、全 action 失敗による error を表す。
	ExecutionExpectedTerminalError ExecutionExpectedTerminal = "error"
)

// ExecutionExpected は、execution fixture が許可する二つの終端期待値である。
type ExecutionExpected interface {
	Terminal() ExecutionExpectedTerminal
	Attempts() []ExpectedAttempt
	Validate() error
	executionExpected()
}

// ExecutionExpectedResultValues は、結果終端の期待投影値を保持する。
type ExecutionExpectedResultValues struct {
	Status            legalquery.LegalQueryResultStatus
	ReturnedItemCount int
	Attempts          []ExpectedAttempt
}

// ExecutionExpectedResult は、status、公開総件数および plan 順 attempt を表す。
type ExecutionExpectedResult struct {
	status            legalquery.LegalQueryResultStatus
	returnedItemCount int
	attempts          []ExpectedAttempt
	initialized       bool
}

// NewExecutionExpectedResult は、結果終端内で完結する整合を検証して返す。
func NewExecutionExpectedResult(
	values ExecutionExpectedResultValues,
) (ExecutionExpectedResult, error) {
	attempts, err := cloneExpectedAttempts(values.Attempts)
	if err != nil {
		return ExecutionExpectedResult{}, err
	}
	expected := ExecutionExpectedResult{
		status:            values.Status,
		returnedItemCount: values.ReturnedItemCount,
		attempts:          attempts,
		initialized:       true,
	}
	if err := expected.Validate(); err != nil {
		return ExecutionExpectedResult{}, err
	}
	return expected, nil
}

// Terminal は、result を返す。
func (e ExecutionExpectedResult) Terminal() ExecutionExpectedTerminal {
	return ExecutionExpectedTerminalResult
}

// Status は、completed、empty または partial を返す。
func (e ExecutionExpectedResult) Status() legalquery.LegalQueryResultStatus {
	return e.status
}

// ReturnedItemCount は、成功 attempt が公開する item 合計を返す。
func (e ExecutionExpectedResult) ReturnedItemCount() int {
	return e.returnedItemCount
}

// Attempts は、plan 順の期待 attempt の複製を返す。
func (e ExecutionExpectedResult) Attempts() []ExpectedAttempt {
	return mustCloneExecutionExpectedAttempts(e.attempts)
}

// Validate は、status、attempt 組合せおよび公開件数合計を確認する。
func (e ExecutionExpectedResult) Validate() error {
	if !e.initialized {
		return fmt.Errorf(
			"ExecutionExpectedResult は NewExecutionExpectedResult で作成しなければなりません",
		)
	}
	if e.returnedItemCount < 0 ||
		e.returnedItemCount > maximumExecutionReturnedItemCount {
		return fmt.Errorf("returnedItemCount は 0 以上 40 以下でなければなりません")
	}
	counts, err := summarizeExecutionExpectedAttempts(e.attempts)
	if err != nil {
		return err
	}
	if counts.published != e.returnedItemCount {
		return fmt.Errorf("returnedItemCount は成功 attempt の公開件数合計と一致しなければなりません")
	}
	switch e.status {
	case legalquery.LegalQueryResultStatusCompleted:
		if counts.completed < 1 || counts.failed != 0 {
			return fmt.Errorf("completed には非空成功が一件以上あり、失敗を含められません")
		}
	case legalquery.LegalQueryResultStatusEmpty:
		if counts.empty != len(e.attempts) {
			return fmt.Errorf("empty の全 attempt は正常な空でなければなりません")
		}
	case legalquery.LegalQueryResultStatusPartial:
		if len(e.attempts) < 2 ||
			counts.failed < 1 ||
			counts.completed+counts.empty < 1 {
			return fmt.Errorf("partial には成功と失敗が一件以上ずつ必要です")
		}
	default:
		return fmt.Errorf("expected result の status が定義されていません")
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExecutionExpectedResult) UnmarshalJSON(_ []byte) error {
	return directExecutionExpectedRestoreError("ExecutionExpectedResult")
}

func (ExecutionExpectedResult) executionExpected() {}

// ExecutionExpectedErrorValues は、全 action 失敗時の期待値を保持する。
type ExecutionExpectedErrorValues struct {
	ErrorCode model.ErrorCode
	Attempts  []ExpectedAttempt
}

// ExecutionExpectedError は、最初の plan 順失敗 code と全 failed attempt を表す。
type ExecutionExpectedError struct {
	errorCode   model.ErrorCode
	attempts    []ExpectedAttempt
	initialized bool
}

// NewExecutionExpectedError は、error 終端内の整合を検証して返す。
func NewExecutionExpectedError(
	values ExecutionExpectedErrorValues,
) (ExecutionExpectedError, error) {
	attempts, err := cloneExpectedAttempts(values.Attempts)
	if err != nil {
		return ExecutionExpectedError{}, err
	}
	expected := ExecutionExpectedError{
		errorCode:   values.ErrorCode,
		attempts:    attempts,
		initialized: true,
	}
	if err := expected.Validate(); err != nil {
		return ExecutionExpectedError{}, err
	}
	return expected, nil
}

// Terminal は、error を返す。
func (e ExecutionExpectedError) Terminal() ExecutionExpectedTerminal {
	return ExecutionExpectedTerminalError
}

// ErrorCode は、plan 順で最初の failed attempt の code を返す。
func (e ExecutionExpectedError) ErrorCode() model.ErrorCode {
	return e.errorCode
}

// Attempts は、plan 順の failed attempt の複製を返す。
func (e ExecutionExpectedError) Attempts() []ExpectedAttempt {
	return mustCloneExecutionExpectedAttempts(e.attempts)
}

// Validate は、全 attempt の失敗と先頭 code の一致を確認する。
func (e ExecutionExpectedError) Validate() error {
	if !e.initialized {
		return fmt.Errorf(
			"ExecutionExpectedError は NewExecutionExpectedError で作成しなければなりません",
		)
	}
	if !isAllowedExecutionFailureCode(e.errorCode) {
		return fmt.Errorf("expected error の errorCode は公開可能な実行時エラーでなければなりません")
	}
	if len(e.attempts) < 1 || len(e.attempts) > maximumExecutionExpectedAttempts {
		return fmt.Errorf("expected error の attempts は一件以上四件以下でなければなりません")
	}
	for _, attempt := range e.attempts {
		failed, ok := attempt.(ExpectedFailedAttempt)
		if !ok {
			return fmt.Errorf("expected error の attempts は全件 failed でなければなりません")
		}
		if err := failed.Validate(); err != nil {
			return fmt.Errorf("expected error の failed attempt が有効ではありません: %w", err)
		}
	}
	first := e.attempts[0].(ExpectedFailedAttempt)
	if e.errorCode != first.ErrorCode() {
		return fmt.Errorf("expected error の errorCode は先頭 attempt と一致しなければなりません")
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExecutionExpectedError) UnmarshalJSON(_ []byte) error {
	return directExecutionExpectedRestoreError("ExecutionExpectedError")
}

func (ExecutionExpectedError) executionExpected() {}

type executionExpectedAttemptCounts struct {
	completed int
	empty     int
	failed    int
	published int
}

func summarizeExecutionExpectedAttempts(
	attempts []ExpectedAttempt,
) (executionExpectedAttemptCounts, error) {
	if len(attempts) < 1 || len(attempts) > maximumExecutionExpectedAttempts {
		return executionExpectedAttemptCounts{},
			fmt.Errorf("expected result の attempts は一件以上四件以下でなければなりません")
	}
	counts := executionExpectedAttemptCounts{}
	for _, attempt := range attempts {
		if _, err := cloneExpectedAttempt(attempt); err != nil {
			return executionExpectedAttemptCounts{},
				fmt.Errorf("expected result の attempt が有効ではありません: %w", err)
		}
		switch typed := attempt.(type) {
		case ExpectedCompletedReadAttempt:
			counts.completed++
			counts.published += typed.PublishedItemCount()
		case ExpectedCompletedCollectionAttempt:
			counts.completed++
			counts.published += typed.PublishedItemCount()
		case ExpectedEmptyAttempt:
			counts.empty++
		case ExpectedFailedAttempt:
			counts.failed++
		default:
			return executionExpectedAttemptCounts{},
				fmt.Errorf("expected result の attempt variant が定義されていません")
		}
	}
	return counts, nil
}

func mustCloneExecutionExpectedAttempts(
	values []ExpectedAttempt,
) []ExpectedAttempt {
	cloned, err := cloneExpectedAttempts(values)
	if err != nil {
		panic(fmt.Sprintf("検証済み execution expected の attempts 複製に失敗しました: %v", err))
	}
	return cloned
}

func directExecutionExpectedRestoreError(typeName string) error {
	return fmt.Errorf(
		"%s は JSON から直接復元できません。version 別 DTO を使用してください",
		typeName,
	)
}
