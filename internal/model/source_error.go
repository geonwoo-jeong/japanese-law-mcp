package model

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// SourceErrorCode は、正規化済みの情報源エラー分類を表す。
type SourceErrorCode string

const (
	// SourceErrorCodeUnsupportedCapability は、プロバイダーが能力を宣言していないことを表す。
	SourceErrorCodeUnsupportedCapability SourceErrorCode = "unsupported_capability"
	// SourceErrorCodeUnsupportedQuery は、有効な条件がプロバイダーの公式な対象範囲外であることを表す。
	SourceErrorCodeUnsupportedQuery SourceErrorCode = "unsupported_query"
	// SourceErrorCodeConfigurationRequired は、必要なプロバイダー設定がないことを表す。
	SourceErrorCodeConfigurationRequired SourceErrorCode = "configuration_required"
	// SourceErrorCodeSourceAuthFailed は、情報源の認証または認可に失敗したことを表す。
	SourceErrorCodeSourceAuthFailed SourceErrorCode = "source_auth_failed"
	// SourceErrorCodeRateLimited は、情報源が呼出し頻度を制限したことを表す。
	SourceErrorCodeRateLimited SourceErrorCode = "rate_limited"
	// SourceErrorCodeSourceTimeout は、情報源との通信が期限を超えたことを表す。
	SourceErrorCodeSourceTimeout SourceErrorCode = "source_timeout"
	// SourceErrorCodeSourceUnavailable は、情報源へ一時的に到達できないことを表す。
	SourceErrorCodeSourceUnavailable SourceErrorCode = "source_unavailable"
	// SourceErrorCodeSourceBusy は、同時実行 group が上限に達したことを表す。
	SourceErrorCodeSourceBusy SourceErrorCode = "source_busy"
	// SourceErrorCodeSourceContractChanged は、情報源の構造が確認済みの契約と一致しないことを表す。
	SourceErrorCodeSourceContractChanged SourceErrorCode = "source_contract_changed"
	// SourceErrorCodeInvalidSourceResponse は、情報源の応答値または形式が契約を満たさないことを表す。
	SourceErrorCodeInvalidSourceResponse SourceErrorCode = "invalid_source_response"
	// SourceErrorCodeSourceResponseTooLarge は、応答または展開結果が安全上の上限を超えたことを表す。
	SourceErrorCodeSourceResponseTooLarge SourceErrorCode = "source_response_too_large"
	// SourceErrorCodeSourceProcessingLimit は、展開または解析が処理時間上限を超えたことを表す。
	SourceErrorCodeSourceProcessingLimit SourceErrorCode = "source_processing_limit"
	// SourceErrorCodeUnsafeSourceContent は、情報源の内容を安全に処理できないことを表す。
	SourceErrorCodeUnsafeSourceContent SourceErrorCode = "unsafe_source_content"
)

const (
	unsupportedCapabilityMessage  = "選択したプロバイダーは、この機能に対応していません。別の情報源または機能を選択してください。"
	unsupportedQueryMessage       = "指定した条件は、選択した情報源の公式な対象範囲では処理できません。条件または情報源を変更してください。"
	configurationRequiredMessage  = "情報源を利用するための設定が不足しています。実行環境の設定を確認してください。"
	sourceAuthFailedMessage       = "情報源の認証または利用権限を確認してください。"
	rateLimitedMessage            = "情報源の呼出し頻度が制限されています。しばらく待ってから再試行してください。"
	sourceTimeoutMessage          = "情報源との通信が期限を超えました。しばらく待ってから再試行してください。"
	sourceUnavailableMessage      = "情報源へ一時的に接続できません。しばらく待ってから再試行してください。"
	sourceBusyMessage             = "このプロバイダーで実行中の処理が上限に達しています。完了を待ってから再試行してください。"
	sourceContractChangedMessage  = "情報源の仕様が確認済みの契約と一致しません。実装または情報源の更新を確認してください。"
	invalidSourceResponseMessage  = "情報源から契約を満たさない応答を受信しました。入力を変えず、情報源または実装の更新を確認してください。"
	sourceResponseTooLargeMessage = "情報源の応答が安全に処理できる大きさの上限を超えました。"
	sourceProcessingLimitMessage  = "情報源の内容を安全な処理時間内に解析できませんでした。"
	unsafeSourceContentMessage    = "情報源の内容を安全に処理できませんでした。"
	invalidSourceErrorMessage     = "情報源エラーが有効ではありません。"
)

// SourceErrorValues は、SourceError の作成に必要な安全な文脈を保持する。
type SourceErrorValues struct {
	Code       SourceErrorCode
	Provider   ProviderDescriptor
	Capability ProviderCapability
	Operation  string
	RetryAfter string
}

// SourceError は、プロバイダー固有の失敗を共通分類へ正規化した内部エラーである。
type SourceError struct {
	code       SourceErrorCode
	provider   ProviderDescriptor
	capability ProviderCapability
	operation  string
	retryAfter string
}

// NewSourceError は、安全な固定メッセージと再試行規則を持つ SourceError を返す。
func NewSourceError(values SourceErrorValues) (SourceError, error) {
	sourceError := SourceError{
		code:       values.Code,
		provider:   values.Provider,
		capability: values.Capability,
		operation:  values.Operation,
		retryAfter: values.RetryAfter,
	}
	if err := sourceError.Validate(); err != nil {
		return SourceError{}, err
	}
	return sourceError, nil
}

// Error は、外部本文や内部原因を含まない利用者向けの日本語メッセージを返す。
func (e SourceError) Error() string {
	message, exists := sourceErrorMessage(e.code)
	if !exists {
		return invalidSourceErrorMessage
	}
	return message
}

// Code は、正規化済みの情報源エラーコードを返す。
func (e SourceError) Code() SourceErrorCode {
	return e.code
}

// Message は、コードごとに固定した安全な日本語メッセージを返す。
func (e SourceError) Message() string {
	return e.Error()
}

// ProviderID は、失敗したプロバイダーの識別子を返す。
func (e SourceError) ProviderID() string {
	return e.provider.ProviderID()
}

// SourceID は、失敗した情報源の識別子を返す。
func (e SourceError) SourceID() string {
	return e.provider.Source().ID()
}

// CapabilityID は、実行しようとした能力の識別子を返す。
func (e SourceError) CapabilityID() string {
	return e.capability.ID()
}

// Operation は、失敗したプロバイダー内 operation を返す。
func (e SourceError) Operation() string {
	return e.operation
}

// Retryable は、入力を変えずに再試行する意味があるかを返す。
func (e SourceError) Retryable() bool {
	return e.code == SourceErrorCodeRateLimited ||
		e.code == SourceErrorCodeSourceTimeout ||
		e.code == SourceErrorCodeSourceUnavailable ||
		e.code == SourceErrorCodeSourceBusy
}

// RetryAfter は、情報源が明示した待機値と有無を返す。
func (e SourceError) RetryAfter() (string, bool) {
	return e.retryAfter, e.retryAfter != ""
}

// Validate は、分類、安全な文脈および retryAfter の従属制約を確認する。
func (e SourceError) Validate() error {
	if _, exists := sourceErrorMessage(e.code); !exists {
		return fmt.Errorf("code が定義されていません")
	}
	if err := e.provider.Validate(); err != nil {
		return fmt.Errorf("provider が有効ではありません: %w", err)
	}
	if err := e.capability.Validate(); err != nil {
		return fmt.Errorf("capability が有効ではありません: %w", err)
	}
	if err := validateCapabilityNamespace(e.provider.ProviderID(), e.capability); err != nil {
		return fmt.Errorf("capability が provider と一致しません: %w", err)
	}
	if err := validateSourceErrorCapabilityDeclaration(
		e.code,
		e.provider,
		e.capability,
	); err != nil {
		return err
	}
	if err := validateSourceErrorOperation(e.operation); err != nil {
		return err
	}
	return validateSourceErrorRetryAfter(e.code, e.retryAfter)
}

// MarshalJSON は、公開 ErrorResult を介さない JSON 変換を拒否する。
func (SourceError) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf(
		"SourceError は JSON に直接変換できません。MCP 境界で ErrorResult へ変換してください",
	)
}

// UnmarshalJSON は、情報源アダプターによる正規化を介さない直接復元を拒否する。
func (*SourceError) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"SourceError は JSON から直接復元できません。情報源アダプターで NewSourceError を使用してください",
	)
}

func sourceErrorMessage(code SourceErrorCode) (string, bool) {
	switch code {
	case SourceErrorCodeUnsupportedCapability:
		return unsupportedCapabilityMessage, true
	case SourceErrorCodeUnsupportedQuery:
		return unsupportedQueryMessage, true
	case SourceErrorCodeConfigurationRequired:
		return configurationRequiredMessage, true
	case SourceErrorCodeSourceAuthFailed:
		return sourceAuthFailedMessage, true
	case SourceErrorCodeRateLimited:
		return rateLimitedMessage, true
	case SourceErrorCodeSourceTimeout:
		return sourceTimeoutMessage, true
	case SourceErrorCodeSourceUnavailable:
		return sourceUnavailableMessage, true
	case SourceErrorCodeSourceBusy:
		return sourceBusyMessage, true
	case SourceErrorCodeSourceContractChanged:
		return sourceContractChangedMessage, true
	case SourceErrorCodeInvalidSourceResponse:
		return invalidSourceResponseMessage, true
	case SourceErrorCodeSourceResponseTooLarge:
		return sourceResponseTooLargeMessage, true
	case SourceErrorCodeSourceProcessingLimit:
		return sourceProcessingLimitMessage, true
	case SourceErrorCodeUnsafeSourceContent:
		return unsafeSourceContentMessage, true
	default:
		return "", false
	}
}

func validateSourceErrorCapabilityDeclaration(
	code SourceErrorCode,
	provider ProviderDescriptor,
	capability ProviderCapability,
) error {
	var declared ProviderCapability
	exists := false
	for _, candidate := range provider.Capabilities() {
		if candidate.ID() == capability.ID() &&
			candidate.MajorVersion() == capability.MajorVersion() {
			declared = candidate
			exists = true
			break
		}
	}
	if code == SourceErrorCodeUnsupportedCapability {
		if exists {
			return fmt.Errorf("unsupported_capability の capability が provider に宣言されています")
		}
		return nil
	}
	if !exists {
		return fmt.Errorf("capability が provider に宣言されていません")
	}
	if declared != capability {
		return fmt.Errorf("capability が provider の宣言と一致しません")
	}
	return nil
}

func validateSourceErrorOperation(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("operation は必須です")
	}
	if !utf8.ValidString(value) ||
		strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return fmt.Errorf("operation は制御文字を含まない有効な UTF-8 でなければなりません")
	}
	if strings.ContainsAny(value, "?#") {
		return fmt.Errorf("operation に query または fragment を含めることはできません")
	}
	return nil
}

func validateSourceErrorRetryAfter(
	code SourceErrorCode,
	retryAfter string,
) error {
	if retryAfter == "" {
		return nil
	}
	if code != SourceErrorCodeRateLimited &&
		code != SourceErrorCodeSourceUnavailable {
		return fmt.Errorf("retryAfter は rate_limited または source_unavailable だけで使用できます")
	}
	if !utf8.ValidString(retryAfter) ||
		strings.IndexFunc(retryAfter, unicode.IsControl) >= 0 {
		return fmt.Errorf("retryAfter は制御文字を含まない有効な UTF-8 でなければなりません")
	}
	return nil
}
