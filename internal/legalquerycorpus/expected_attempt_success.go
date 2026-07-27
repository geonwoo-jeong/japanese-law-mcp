package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const maximumExpectedPublishedCollectionItems = 20

// ExpectedCompletedReadAttemptValues は、read 成功の期待投影値を保持する。
type ExpectedCompletedReadAttemptValues struct {
	MeaningID          string
	StepOrdinal        int
	PublishedItemCount int
}

// ExpectedCompletedReadAttempt は、一 item を公開する read 成功を表す。
type ExpectedCompletedReadAttempt struct {
	header             expectedAttemptHeader
	publishedItemCount int
}

// NewExpectedCompletedReadAttempt は、固定一 item の read 成功を返す。
func NewExpectedCompletedReadAttempt(
	values ExpectedCompletedReadAttemptValues,
) (ExpectedCompletedReadAttempt, error) {
	header, err := newExpectedAttemptHeader(values.MeaningID, values.StepOrdinal)
	if err != nil {
		return ExpectedCompletedReadAttempt{}, err
	}
	attempt := ExpectedCompletedReadAttempt{
		header:             header,
		publishedItemCount: values.PublishedItemCount,
	}
	if err := attempt.Validate(); err != nil {
		return ExpectedCompletedReadAttempt{}, err
	}
	return attempt, nil
}

// MeaningID は、semantic case の期待意味参照を返す。
func (a ExpectedCompletedReadAttempt) MeaningID() string {
	return a.header.meaningID
}

// StepOrdinal は、意味内で一から始まる step 順を返す。
func (a ExpectedCompletedReadAttempt) StepOrdinal() int {
	return a.header.stepOrdinal
}

// Outcome は、completed を返す。
func (a ExpectedCompletedReadAttempt) Outcome() legalquery.LegalQueryAttemptOutcome {
	return legalquery.LegalQueryAttemptOutcomeCompleted
}

// PublishedItemCount は、固定値一を返す。
func (a ExpectedCompletedReadAttempt) PublishedItemCount() int {
	return a.publishedItemCount
}

// Validate は、header と read の固定公開件数を確認する。
func (a ExpectedCompletedReadAttempt) Validate() error {
	if err := a.header.validate(); err != nil {
		return err
	}
	if a.publishedItemCount != 1 {
		return fmt.Errorf("completed read の publishedItemCount は 1 でなければなりません")
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExpectedCompletedReadAttempt) UnmarshalJSON(_ []byte) error {
	return directExpectedAttemptRestoreError("ExpectedCompletedReadAttempt")
}

func (ExpectedCompletedReadAttempt) expectedAttempt() {}

// ExpectedCompletedCollectionAttemptValues は、collection 成功の期待値を保持する。
type ExpectedCompletedCollectionAttemptValues struct {
	MeaningID          string
	StepOrdinal        int
	PublishedItemCount int
	HasMore            bool
}

// ExpectedCompletedCollectionAttempt は、非空 preview の件数と継続有無を表す。
type ExpectedCompletedCollectionAttempt struct {
	header             expectedAttemptHeader
	publishedItemCount int
	hasMore            bool
}

// NewExpectedCompletedCollectionAttempt は、非空 collection 成功を返す。
func NewExpectedCompletedCollectionAttempt(
	values ExpectedCompletedCollectionAttemptValues,
) (ExpectedCompletedCollectionAttempt, error) {
	header, err := newExpectedAttemptHeader(values.MeaningID, values.StepOrdinal)
	if err != nil {
		return ExpectedCompletedCollectionAttempt{}, err
	}
	attempt := ExpectedCompletedCollectionAttempt{
		header:             header,
		publishedItemCount: values.PublishedItemCount,
		hasMore:            values.HasMore,
	}
	if err := attempt.Validate(); err != nil {
		return ExpectedCompletedCollectionAttempt{}, err
	}
	return attempt, nil
}

// MeaningID は、semantic case の期待意味参照を返す。
func (a ExpectedCompletedCollectionAttempt) MeaningID() string {
	return a.header.meaningID
}

// StepOrdinal は、意味内で一から始まる step 順を返す。
func (a ExpectedCompletedCollectionAttempt) StepOrdinal() int {
	return a.header.stepOrdinal
}

// Outcome は、completed を返す。
func (a ExpectedCompletedCollectionAttempt) Outcome() legalquery.LegalQueryAttemptOutcome {
	return legalquery.LegalQueryAttemptOutcomeCompleted
}

// PublishedItemCount は、公開する preview 件数を返す。
func (a ExpectedCompletedCollectionAttempt) PublishedItemCount() int {
	return a.publishedItemCount
}

// HasMore は、情報源に公開上限を超える残件があるか返す。
func (a ExpectedCompletedCollectionAttempt) HasMore() bool {
	return a.hasMore
}

// Validate は、header と collection の非空公開件数を確認する。
func (a ExpectedCompletedCollectionAttempt) Validate() error {
	if err := a.header.validate(); err != nil {
		return err
	}
	if a.publishedItemCount < 1 ||
		a.publishedItemCount > maximumExpectedPublishedCollectionItems {
		return fmt.Errorf(
			"completed collection の publishedItemCount は 1 以上 20 以下でなければなりません",
		)
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExpectedCompletedCollectionAttempt) UnmarshalJSON(_ []byte) error {
	return directExpectedAttemptRestoreError("ExpectedCompletedCollectionAttempt")
}

func (ExpectedCompletedCollectionAttempt) expectedAttempt() {}

// ExpectedEmptyAttemptValues は、collection 空結果の期待値を保持する。
type ExpectedEmptyAttemptValues struct {
	MeaningID          string
	StepOrdinal        int
	PublishedItemCount int
	HasMore            bool
}

// ExpectedEmptyAttempt は、零件で継続なしの正常な collection 結果を表す。
type ExpectedEmptyAttempt struct {
	header             expectedAttemptHeader
	publishedItemCount int
	hasMore            bool
}

// NewExpectedEmptyAttempt は、固定零件の collection 空結果を返す。
func NewExpectedEmptyAttempt(
	values ExpectedEmptyAttemptValues,
) (ExpectedEmptyAttempt, error) {
	header, err := newExpectedAttemptHeader(values.MeaningID, values.StepOrdinal)
	if err != nil {
		return ExpectedEmptyAttempt{}, err
	}
	attempt := ExpectedEmptyAttempt{
		header:             header,
		publishedItemCount: values.PublishedItemCount,
		hasMore:            values.HasMore,
	}
	if err := attempt.Validate(); err != nil {
		return ExpectedEmptyAttempt{}, err
	}
	return attempt, nil
}

// MeaningID は、semantic case の期待意味参照を返す。
func (a ExpectedEmptyAttempt) MeaningID() string {
	return a.header.meaningID
}

// StepOrdinal は、意味内で一から始まる step 順を返す。
func (a ExpectedEmptyAttempt) StepOrdinal() int {
	return a.header.stepOrdinal
}

// Outcome は、empty を返す。
func (a ExpectedEmptyAttempt) Outcome() legalquery.LegalQueryAttemptOutcome {
	return legalquery.LegalQueryAttemptOutcomeEmpty
}

// PublishedItemCount は、固定値零を返す。
func (a ExpectedEmptyAttempt) PublishedItemCount() int {
	return a.publishedItemCount
}

// HasMore は、固定値 false を返す。
func (a ExpectedEmptyAttempt) HasMore() bool {
	return a.hasMore
}

// Validate は、header、零件および継続なしを確認する。
func (a ExpectedEmptyAttempt) Validate() error {
	if err := a.header.validate(); err != nil {
		return err
	}
	if a.publishedItemCount != 0 || a.hasMore {
		return fmt.Errorf(
			"empty の publishedItemCount は 0、hasMore は false でなければなりません",
		)
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExpectedEmptyAttempt) UnmarshalJSON(_ []byte) error {
	return directExpectedAttemptRestoreError("ExpectedEmptyAttempt")
}

func (ExpectedEmptyAttempt) expectedAttempt() {}
