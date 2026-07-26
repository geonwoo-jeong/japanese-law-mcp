package config

import "errors"

// ValidationError は、起動時設定の構造または意味が成立しないことを表す。
type ValidationError struct {
	cause error
}

// NewValidationError は、composition root を含む起動時設定検証のエラーを分類する。
func NewValidationError(cause error) error {
	if cause == nil {
		return nil
	}
	if IsValidationError(cause) {
		return cause
	}
	return &ValidationError{cause: cause}
}

// Error は、利用者向けの原因をそのまま返す。
func (e *ValidationError) Error() string {
	return e.cause.Error()
}

// Unwrap は、設定エラーの具体的な原因を返す。
func (e *ValidationError) Unwrap() error {
	return e.cause
}

// IsValidationError は、エラー連鎖に起動時設定検証エラーが含まれるかを返す。
func IsValidationError(err error) bool {
	var target *ValidationError
	return errors.As(err, &target)
}
