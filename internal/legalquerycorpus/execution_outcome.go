package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const maximumExecutionSourceItemCount = 1000

// ExecutionOutcomeKind は、fake capability action の宣言的結果種別を表す。
type ExecutionOutcomeKind string

const (
	// ExecutionOutcomeKindCollectionSuccess は、collection の成功を表す。
	ExecutionOutcomeKindCollectionSuccess ExecutionOutcomeKind = "collection_success"
	// ExecutionOutcomeKindReadSuccess は、read の成功を表す。
	ExecutionOutcomeKindReadSuccess ExecutionOutcomeKind = "read_success"
	// ExecutionOutcomeKindFailure は、公開可能な情報源エラーを表す。
	ExecutionOutcomeKindFailure ExecutionOutcomeKind = "failure"
	// ExecutionOutcomeKindTimeout は、fake clock の deadline event を表す。
	ExecutionOutcomeKindTimeout ExecutionOutcomeKind = "timeout"
)

// ExecutionOutcome は、execution action が許可する四つの宣言的結果である。
type ExecutionOutcome interface {
	Kind() ExecutionOutcomeKind
	Validate() error
	executionOutcome()
}

// CollectionSuccessOutcome は、collection 情報源の取得件数を保持する。
type CollectionSuccessOutcome struct {
	sourceItemCount int
	initialized     bool
}

// NewCollectionSuccessOutcome は、取得件数を検証した collection 成功を返す。
func NewCollectionSuccessOutcome(
	sourceItemCount int,
) (CollectionSuccessOutcome, error) {
	outcome := CollectionSuccessOutcome{
		sourceItemCount: sourceItemCount,
		initialized:     true,
	}
	if err := outcome.Validate(); err != nil {
		return CollectionSuccessOutcome{}, err
	}
	return outcome, nil
}

// Kind は、collection_success を返す。
func (o CollectionSuccessOutcome) Kind() ExecutionOutcomeKind {
	return ExecutionOutcomeKindCollectionSuccess
}

// SourceItemCount は、情報源が返すと宣言した item 件数を返す。
func (o CollectionSuccessOutcome) SourceItemCount() int {
	return o.sourceItemCount
}

// Validate は、初期化と取得件数の固定上限を確認する。
func (o CollectionSuccessOutcome) Validate() error {
	if !o.initialized {
		return fmt.Errorf(
			"CollectionSuccessOutcome は NewCollectionSuccessOutcome で作成しなければなりません",
		)
	}
	if o.sourceItemCount < 0 || o.sourceItemCount > maximumExecutionSourceItemCount {
		return fmt.Errorf("sourceItemCount は 0 以上 1000 以下でなければなりません")
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*CollectionSuccessOutcome) UnmarshalJSON(_ []byte) error {
	return directExecutionOutcomeRestoreError("CollectionSuccessOutcome")
}

func (CollectionSuccessOutcome) executionOutcome() {}

// ReadSuccessOutcome は、追加 payload を持たない read 成功を表す。
type ReadSuccessOutcome struct {
	initialized bool
}

// NewReadSuccessOutcome は、初期化済みの read 成功を返す。
func NewReadSuccessOutcome() (ReadSuccessOutcome, error) {
	outcome := ReadSuccessOutcome{initialized: true}
	if err := outcome.Validate(); err != nil {
		return ReadSuccessOutcome{}, err
	}
	return outcome, nil
}

// Kind は、read_success を返す。
func (o ReadSuccessOutcome) Kind() ExecutionOutcomeKind {
	return ExecutionOutcomeKindReadSuccess
}

// Validate は、constructor を介して作成されたことを確認する。
func (o ReadSuccessOutcome) Validate() error {
	if !o.initialized {
		return fmt.Errorf(
			"ReadSuccessOutcome は NewReadSuccessOutcome で作成しなければなりません",
		)
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ReadSuccessOutcome) UnmarshalJSON(_ []byte) error {
	return directExecutionOutcomeRestoreError("ReadSuccessOutcome")
}

func (ReadSuccessOutcome) executionOutcome() {}

// FailureOutcome は、failed attempt で公開可能な error code を保持する。
type FailureOutcome struct {
	errorCode   model.ErrorCode
	initialized bool
}

// NewFailureOutcome は、許可した情報源エラーだけを持つ失敗を返す。
func NewFailureOutcome(errorCode model.ErrorCode) (FailureOutcome, error) {
	outcome := FailureOutcome{
		errorCode:   errorCode,
		initialized: true,
	}
	if err := outcome.Validate(); err != nil {
		return FailureOutcome{}, err
	}
	return outcome, nil
}

// Kind は、failure を返す。
func (o FailureOutcome) Kind() ExecutionOutcomeKind {
	return ExecutionOutcomeKindFailure
}

// ErrorCode は、failed attempt へ投影する公開 error code を返す。
func (o FailureOutcome) ErrorCode() model.ErrorCode {
	return o.errorCode
}

// Validate は、failed attempt に許可した error code だけか確認する。
func (o FailureOutcome) Validate() error {
	if !o.initialized {
		return fmt.Errorf("FailureOutcome は NewFailureOutcome で作成しなければなりません")
	}
	if !isAllowedExecutionFailureCode(o.errorCode) {
		return fmt.Errorf("failure の errorCode は failed attempt に許可された値でなければなりません")
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*FailureOutcome) UnmarshalJSON(_ []byte) error {
	return directExecutionOutcomeRestoreError("FailureOutcome")
}

func (FailureOutcome) executionOutcome() {}

// TimeoutOutcome は、fake clock が解放する deadline event を表す。
type TimeoutOutcome struct {
	initialized bool
}

// NewTimeoutOutcome は、初期化済みの timeout を返す。
func NewTimeoutOutcome() (TimeoutOutcome, error) {
	outcome := TimeoutOutcome{initialized: true}
	if err := outcome.Validate(); err != nil {
		return TimeoutOutcome{}, err
	}
	return outcome, nil
}

// Kind は、timeout を返す。
func (o TimeoutOutcome) Kind() ExecutionOutcomeKind {
	return ExecutionOutcomeKindTimeout
}

// Validate は、constructor を介して作成されたことを確認する。
func (o TimeoutOutcome) Validate() error {
	if !o.initialized {
		return fmt.Errorf("TimeoutOutcome は NewTimeoutOutcome で作成しなければなりません")
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*TimeoutOutcome) UnmarshalJSON(_ []byte) error {
	return directExecutionOutcomeRestoreError("TimeoutOutcome")
}

func (TimeoutOutcome) executionOutcome() {}

func cloneExecutionOutcome(value ExecutionOutcome) (ExecutionOutcome, error) {
	switch typed := value.(type) {
	case CollectionSuccessOutcome:
		if err := typed.Validate(); err != nil {
			return nil, err
		}
		return NewCollectionSuccessOutcome(typed.SourceItemCount())
	case ReadSuccessOutcome:
		if err := typed.Validate(); err != nil {
			return nil, err
		}
		return NewReadSuccessOutcome()
	case FailureOutcome:
		if err := typed.Validate(); err != nil {
			return nil, err
		}
		return NewFailureOutcome(typed.ErrorCode())
	case TimeoutOutcome:
		if err := typed.Validate(); err != nil {
			return nil, err
		}
		return NewTimeoutOutcome()
	default:
		return nil, fmt.Errorf(
			"execution outcome は検証済みの値 variant でなければなりません",
		)
	}
}

func isAllowedExecutionFailureCode(value model.ErrorCode) bool {
	switch value {
	case model.ErrorCodeNotFound,
		model.ErrorCodeAmbiguousLocation,
		model.ErrorCodeUnsupportedQuery,
		model.ErrorCodeSourceAuthFailed,
		model.ErrorCodeRateLimited,
		model.ErrorCodeSourceTimeout,
		model.ErrorCodeSourceUnavailable,
		model.ErrorCodeSourceBusy,
		model.ErrorCodeSourceContractChanged,
		model.ErrorCodeInvalidSourceResponse,
		model.ErrorCodeSourceResponseTooLarge,
		model.ErrorCodeSourceProcessingLimit,
		model.ErrorCodeUnsafeSourceContent,
		model.ErrorCodeInternalError:
		return true
	default:
		return false
	}
}

func directExecutionOutcomeRestoreError(typeName string) error {
	return fmt.Errorf(
		"%s は JSON から直接復元できません。version 別 DTO を使用してください",
		typeName,
	)
}
