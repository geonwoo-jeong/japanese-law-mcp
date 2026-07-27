package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// ExpectedFailedAttemptValues は、失敗 attempt の期待投影値を保持する。
type ExpectedFailedAttemptValues struct {
	MeaningID   string
	StepOrdinal int
	ErrorCode   model.ErrorCode
}

// ExpectedFailedAttempt は、公開可能な failed attempt の分類を表す。
type ExpectedFailedAttempt struct {
	header    expectedAttemptHeader
	errorCode model.ErrorCode
}

// NewExpectedFailedAttempt は、許可した情報源エラーだけを持つ期待失敗を返す。
func NewExpectedFailedAttempt(
	values ExpectedFailedAttemptValues,
) (ExpectedFailedAttempt, error) {
	header, err := newExpectedAttemptHeader(values.MeaningID, values.StepOrdinal)
	if err != nil {
		return ExpectedFailedAttempt{}, err
	}
	attempt := ExpectedFailedAttempt{
		header:    header,
		errorCode: values.ErrorCode,
	}
	if err := attempt.Validate(); err != nil {
		return ExpectedFailedAttempt{}, err
	}
	return attempt, nil
}

// MeaningID は、semantic case の期待意味参照を返す。
func (a ExpectedFailedAttempt) MeaningID() string {
	return a.header.meaningID
}

// StepOrdinal は、意味内で一から始まる step 順を返す。
func (a ExpectedFailedAttempt) StepOrdinal() int {
	return a.header.stepOrdinal
}

// Outcome は、failed を返す。
func (a ExpectedFailedAttempt) Outcome() legalquery.LegalQueryAttemptOutcome {
	return legalquery.LegalQueryAttemptOutcomeFailed
}

// ErrorCode は、公開結果へ投影する error code を返す。
func (a ExpectedFailedAttempt) ErrorCode() model.ErrorCode {
	return a.errorCode
}

// Validate は、header と failed attempt の許可 code を確認する。
func (a ExpectedFailedAttempt) Validate() error {
	if err := a.header.validate(); err != nil {
		return err
	}
	if !isAllowedExecutionFailureCode(a.errorCode) {
		return fmt.Errorf(
			"failed attempt の errorCode は公開可能な実行時エラーでなければなりません",
		)
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExpectedFailedAttempt) UnmarshalJSON(_ []byte) error {
	return directExpectedAttemptRestoreError("ExpectedFailedAttempt")
}

func (ExpectedFailedAttempt) expectedAttempt() {}
