package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

// ExpectedAttempt は、execution fixture が許可する四つの attempt 投影である。
type ExpectedAttempt interface {
	MeaningID() string
	StepOrdinal() int
	Outcome() legalquery.LegalQueryAttemptOutcome
	Validate() error
	expectedAttempt()
}

type expectedAttemptHeader struct {
	meaningID   string
	stepOrdinal int
	initialized bool
}

func newExpectedAttemptHeader(
	meaningID string,
	stepOrdinal int,
) (expectedAttemptHeader, error) {
	header := expectedAttemptHeader{
		meaningID:   meaningID,
		stepOrdinal: stepOrdinal,
		initialized: true,
	}
	if err := header.validate(); err != nil {
		return expectedAttemptHeader{}, err
	}
	return header, nil
}

func (h expectedAttemptHeader) validate() error {
	if !h.initialized {
		return fmt.Errorf("expected attempt header が初期化されていません")
	}
	if err := validateExpectedIdentifier("meaningId", h.meaningID); err != nil {
		return err
	}
	if h.stepOrdinal < 1 || h.stepOrdinal > maximumExecutionActionOrdinal {
		return fmt.Errorf("stepOrdinal は 1 以上 4 以下でなければなりません")
	}
	return nil
}

func cloneExpectedAttempt(value ExpectedAttempt) (ExpectedAttempt, error) {
	switch typed := value.(type) {
	case ExpectedCompletedReadAttempt:
		if err := typed.Validate(); err != nil {
			return nil, err
		}
		return NewExpectedCompletedReadAttempt(
			ExpectedCompletedReadAttemptValues{
				MeaningID:          typed.MeaningID(),
				StepOrdinal:        typed.StepOrdinal(),
				PublishedItemCount: typed.PublishedItemCount(),
			},
		)
	case ExpectedCompletedCollectionAttempt:
		if err := typed.Validate(); err != nil {
			return nil, err
		}
		return NewExpectedCompletedCollectionAttempt(
			ExpectedCompletedCollectionAttemptValues{
				MeaningID:          typed.MeaningID(),
				StepOrdinal:        typed.StepOrdinal(),
				PublishedItemCount: typed.PublishedItemCount(),
				HasMore:            typed.HasMore(),
			},
		)
	case ExpectedEmptyAttempt:
		if err := typed.Validate(); err != nil {
			return nil, err
		}
		return NewExpectedEmptyAttempt(ExpectedEmptyAttemptValues{
			MeaningID:          typed.MeaningID(),
			StepOrdinal:        typed.StepOrdinal(),
			PublishedItemCount: typed.PublishedItemCount(),
			HasMore:            typed.HasMore(),
		})
	case ExpectedFailedAttempt:
		if err := typed.Validate(); err != nil {
			return nil, err
		}
		return NewExpectedFailedAttempt(ExpectedFailedAttemptValues{
			MeaningID:   typed.MeaningID(),
			StepOrdinal: typed.StepOrdinal(),
			ErrorCode:   typed.ErrorCode(),
		})
	default:
		return nil, fmt.Errorf(
			"expected attempt は検証済みの値 variant でなければなりません",
		)
	}
}

func cloneExpectedAttempts(values []ExpectedAttempt) ([]ExpectedAttempt, error) {
	cloned := make([]ExpectedAttempt, 0, len(values))
	for _, value := range values {
		attempt, err := cloneExpectedAttempt(value)
		if err != nil {
			return nil, fmt.Errorf("expected attempt が有効ではありません: %w", err)
		}
		cloned = append(cloned, attempt)
	}
	return cloned, nil
}

func directExpectedAttemptRestoreError(typeName string) error {
	return fmt.Errorf(
		"%s は JSON から直接復元できません。version 別 DTO を使用してください",
		typeName,
	)
}
