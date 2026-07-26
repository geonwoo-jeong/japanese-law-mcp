package model

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrorCode は、公開するツールエラーの分類を表す。
type ErrorCode string

const (
	ErrorCodeInvalidArgument        ErrorCode = "invalid_argument"
	ErrorCodeNotFound               ErrorCode = "not_found"
	ErrorCodeAmbiguousLocation      ErrorCode = "ambiguous_location"
	ErrorCodeUnsupportedCapability  ErrorCode = "unsupported_capability"
	ErrorCodeUnsupportedQuery       ErrorCode = "unsupported_query"
	ErrorCodeConfigurationRequired  ErrorCode = "configuration_required"
	ErrorCodeSourceAuthFailed       ErrorCode = "source_auth_failed"
	ErrorCodeRateLimited            ErrorCode = "rate_limited"
	ErrorCodeSourceTimeout          ErrorCode = "source_timeout"
	ErrorCodeSourceUnavailable      ErrorCode = "source_unavailable"
	ErrorCodeSourceBusy             ErrorCode = "source_busy"
	ErrorCodeSourceContractChanged  ErrorCode = "source_contract_changed"
	ErrorCodeInvalidSourceResponse  ErrorCode = "invalid_source_response"
	ErrorCodeSourceResponseTooLarge ErrorCode = "source_response_too_large"
	ErrorCodeSourceProcessingLimit  ErrorCode = "source_processing_limit"
	ErrorCodeUnsafeSourceContent    ErrorCode = "unsafe_source_content"
	ErrorCodeInternalError          ErrorCode = "internal_error"
)

// ErrorResultValues は、ErrorResult の作成に必要な安全な値を保持する。
type ErrorResultValues struct {
	Code    ErrorCode
	Details map[string]any
}

// ErrorResult は、公開可能な固定メッセージと追加情報を不変に保持する。
type ErrorResult struct {
	code        ErrorCode
	message     string
	retryable   bool
	details     map[string]any
	initialized bool
}

// NewErrorResult は、コードごとの固定メッセージと許可済み details を返す。
func NewErrorResult(values ErrorResultValues) (ErrorResult, error) {
	message, exists := publicErrorMessage(values.Code)
	if !exists {
		return ErrorResult{}, fmt.Errorf("ErrorResult の code が定義されていません")
	}
	details, err := cloneAndValidateDetails(values.Code, values.Details)
	if err != nil {
		return ErrorResult{}, err
	}
	result := ErrorResult{
		code:        values.Code,
		message:     message,
		retryable:   publicErrorRetryable(values.Code),
		details:     details,
		initialized: true,
	}
	if err := result.Validate(); err != nil {
		return ErrorResult{}, err
	}
	return result, nil
}

// NewErrorResultFromSourceError は、情報源エラーを同名の公開コードへ一対一で変換する。
func NewErrorResultFromSourceError(source SourceError) (ErrorResult, error) {
	if err := source.Validate(); err != nil {
		return ErrorResult{}, fmt.Errorf("SourceError が有効ではありません: %w", err)
	}
	code := ErrorCode(source.Code())
	details := map[string]any{
		"providerId":   source.ProviderID(),
		"sourceId":     source.SourceID(),
		"capabilityId": source.CapabilityID(),
	}
	if sourceErrorPublishesOperation(code) {
		details["operation"] = source.Operation()
	}
	if retryAfter, exists := source.RetryAfter(); exists {
		details["retryAfter"] = retryAfter
	}
	return NewErrorResult(ErrorResultValues{
		Code:    code,
		Details: details,
	})
}

func (r ErrorResult) Code() ErrorCode {
	return r.code
}

func (r ErrorResult) Message() string {
	return r.message
}

func (r ErrorResult) Retryable() bool {
	return r.retryable
}

// Details は、安全な追加情報の複製と有無を返す。
func (r ErrorResult) Details() (map[string]any, bool) {
	if len(r.details) == 0 {
		return nil, false
	}
	cloned, _ := cloneAndValidateDetails(r.code, r.details)
	return cloned, true
}

// Validate は、コード、固定メッセージ、再試行可否および details を確認する。
func (r ErrorResult) Validate() error {
	if !r.initialized {
		return fmt.Errorf("ErrorResult は NewErrorResult で作成しなければなりません")
	}
	message, exists := publicErrorMessage(r.code)
	if !exists || r.message != message {
		return fmt.Errorf("code と message が一致しません")
	}
	if r.retryable != publicErrorRetryable(r.code) {
		return fmt.Errorf("code と retryable が一致しません")
	}
	_, err := cloneAndValidateDetails(r.code, r.details)
	return err
}

// MarshalJSON は、SOT-MODEL-005 の項目名で公開エラーを表す。
func (r ErrorResult) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	details, _ := r.Details()
	return json.Marshal(struct {
		Code      ErrorCode      `json:"code"`
		Message   string         `json:"message"`
		Retryable bool           `json:"retryable"`
		Details   map[string]any `json:"details,omitempty"`
	}{
		Code:      r.code,
		Message:   r.message,
		Retryable: r.retryable,
		Details:   details,
	})
}

// UnmarshalJSON は、公開境界で検証しない直接復元を拒否する。
func (*ErrorResult) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ErrorResult は JSON から直接復元できません。NewErrorResult を使用してください",
	)
}

func publicErrorMessage(code ErrorCode) (string, bool) {
	switch code {
	case ErrorCodeInvalidArgument:
		return "入力値が契約を満たしていません。field と reason を確認して修正してください。", true
	case ErrorCodeNotFound:
		return "指定した条件に該当する情報がありません。", true
	case ErrorCodeAmbiguousLocation:
		return "指定した位置を一意に決定できません。候補を確認してください。", true
	case ErrorCodeUnsupportedCapability:
		return unsupportedCapabilityMessage, true
	case ErrorCodeUnsupportedQuery:
		return unsupportedQueryMessage, true
	case ErrorCodeConfigurationRequired:
		return configurationRequiredMessage, true
	case ErrorCodeSourceAuthFailed:
		return sourceAuthFailedMessage, true
	case ErrorCodeRateLimited:
		return rateLimitedMessage, true
	case ErrorCodeSourceTimeout:
		return sourceTimeoutMessage, true
	case ErrorCodeSourceUnavailable:
		return sourceUnavailableMessage, true
	case ErrorCodeSourceBusy:
		return sourceBusyMessage, true
	case ErrorCodeSourceContractChanged:
		return sourceContractChangedMessage, true
	case ErrorCodeInvalidSourceResponse:
		return invalidSourceResponseMessage, true
	case ErrorCodeSourceResponseTooLarge:
		return sourceResponseTooLargeMessage, true
	case ErrorCodeSourceProcessingLimit:
		return sourceProcessingLimitMessage, true
	case ErrorCodeUnsafeSourceContent:
		return unsafeSourceContentMessage, true
	case ErrorCodeInternalError:
		return "内部処理を完了できませんでした。入力を変えずに繰り返さず、実装または実行環境を確認してください。", true
	default:
		return "", false
	}
}

func publicErrorRetryable(code ErrorCode) bool {
	return code == ErrorCodeRateLimited ||
		code == ErrorCodeSourceTimeout ||
		code == ErrorCodeSourceUnavailable ||
		code == ErrorCodeSourceBusy
}

func sourceErrorPublishesOperation(code ErrorCode) bool {
	switch code {
	case ErrorCodeSourceAuthFailed,
		ErrorCodeRateLimited,
		ErrorCodeSourceTimeout,
		ErrorCodeSourceUnavailable,
		ErrorCodeSourceBusy,
		ErrorCodeSourceContractChanged,
		ErrorCodeInvalidSourceResponse,
		ErrorCodeSourceResponseTooLarge,
		ErrorCodeSourceProcessingLimit,
		ErrorCodeUnsafeSourceContent:
		return true
	default:
		return false
	}
}

func cloneAndValidateDetails(
	code ErrorCode,
	details map[string]any,
) (map[string]any, error) {
	if len(details) == 0 {
		return nil, nil
	}
	allowed, exists := publicErrorDetailKeys(code)
	if !exists {
		return nil, fmt.Errorf("code が定義されていません")
	}
	cloned := make(map[string]any, len(details))
	for key, value := range details {
		if _, permitted := allowed[key]; !permitted {
			return nil, fmt.Errorf("details.%s は code で許可されていません", key)
		}
		clonedValue, err := cloneDetailValue(value, false)
		if err != nil {
			return nil, fmt.Errorf("details.%s が有効ではありません: %w", key, err)
		}
		cloned[key] = clonedValue
	}
	return cloned, nil
}

func publicErrorDetailKeys(code ErrorCode) (map[string]struct{}, bool) {
	keys := func(values ...string) map[string]struct{} {
		result := make(map[string]struct{}, len(values))
		for _, value := range values {
			result[value] = struct{}{}
		}
		return result
	}
	switch code {
	case ErrorCodeInvalidArgument:
		return keys("field", "reason"), true
	case ErrorCodeNotFound, ErrorCodeInternalError:
		return keys(), true
	case ErrorCodeAmbiguousLocation:
		return keys("candidates"), true
	case ErrorCodeUnsupportedCapability:
		return keys("providerId", "sourceId", "capabilityId"), true
	case ErrorCodeUnsupportedQuery:
		return keys("providerId", "sourceId", "capabilityId", "field", "constraint"), true
	case ErrorCodeConfigurationRequired:
		return keys("providerId", "sourceId", "capabilityId", "missing"), true
	case ErrorCodeSourceAuthFailed,
		ErrorCodeSourceTimeout,
		ErrorCodeSourceBusy,
		ErrorCodeSourceContractChanged,
		ErrorCodeInvalidSourceResponse,
		ErrorCodeUnsafeSourceContent:
		return keys("providerId", "sourceId", "capabilityId", "operation"), true
	case ErrorCodeRateLimited, ErrorCodeSourceUnavailable:
		return keys(
			"providerId",
			"sourceId",
			"capabilityId",
			"operation",
			"retryAfter",
		), true
	case ErrorCodeSourceResponseTooLarge, ErrorCodeSourceProcessingLimit:
		return keys(
			"providerId",
			"sourceId",
			"capabilityId",
			"operation",
			"limitName",
		), true
	default:
		return nil, false
	}
}

func cloneDetailValue(value any, nested bool) (any, error) {
	if value == nil {
		return nil, fmt.Errorf("null は使用できません")
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.String:
		text := reflected.String()
		if !utf8.ValidString(text) ||
			strings.IndexFunc(text, unicode.IsControl) >= 0 {
			return nil, fmt.Errorf("制御文字を含まない UTF-8 文字列が必要です")
		}
		return text, nil
	case reflect.Bool:
		return reflected.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflected.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflected.Uint(), nil
	case reflect.Float32, reflect.Float64:
		number := reflected.Float()
		if math.IsInf(number, 0) || math.IsNaN(number) {
			return nil, fmt.Errorf("有限の数値が必要です")
		}
		return number, nil
	case reflect.Array, reflect.Slice:
		if nested {
			return nil, fmt.Errorf("配列を入れ子にできません")
		}
		values := make([]any, reflected.Len())
		for index := 0; index < reflected.Len(); index++ {
			cloned, err := cloneDetailValue(reflected.Index(index).Interface(), true)
			if err != nil {
				return nil, err
			}
			values[index] = cloned
		}
		return values, nil
	default:
		return nil, fmt.Errorf("JSON の文字列、数値、真偽値またはその配列が必要です")
	}
}
