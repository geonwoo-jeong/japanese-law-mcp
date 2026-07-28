package legalquery

import "fmt"

// ExecutedStepError は、provider port を実際に呼び出した後の失敗を表す。
type ExecutedStepError struct {
	cause error
}

// NewExecutedStepError は、実行済み step の原因を保持する分類可能な error を返す。
func NewExecutedStepError(cause error) (ExecutedStepError, error) {
	result := ExecutedStepError{cause: cause}
	if err := result.Validate(); err != nil {
		return ExecutedStepError{}, err
	}
	return result, nil
}

// Error は、元の provider port error を変更せず表示する。
func (e ExecutedStepError) Error() string {
	if err := e.Validate(); err != nil {
		return "実行済み step error が初期化されていません"
	}
	return e.cause.Error()
}

// Unwrap は、公開 error への変換に使用する元の失敗を返す。
func (e ExecutedStepError) Unwrap() error {
	return e.cause
}

// Validate は、実行後の失敗原因が実在することを確認する。
func (e ExecutedStepError) Validate() error {
	if isNilInterfaceValue(e.cause) {
		return fmt.Errorf("実行済み step error には原因が必要です")
	}
	return nil
}
