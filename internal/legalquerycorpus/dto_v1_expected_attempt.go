package legalquerycorpus

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type expectedAttemptKindV1DTO struct {
	Outcome *string `json:"outcome"`
	HasMore *bool   `json:"hasMore,omitempty"`
}

type expectedCompletedReadAttemptV1DTO struct {
	MeaningID          *string `json:"meaningId"`
	StepOrdinal        *int    `json:"stepOrdinal"`
	Outcome            *string `json:"outcome"`
	PublishedItemCount *int    `json:"publishedItemCount"`
}

type expectedCompletedCollectionAttemptV1DTO struct {
	MeaningID          *string `json:"meaningId"`
	StepOrdinal        *int    `json:"stepOrdinal"`
	Outcome            *string `json:"outcome"`
	PublishedItemCount *int    `json:"publishedItemCount"`
	HasMore            *bool   `json:"hasMore"`
}

type expectedEmptyAttemptV1DTO struct {
	MeaningID          *string `json:"meaningId"`
	StepOrdinal        *int    `json:"stepOrdinal"`
	Outcome            *string `json:"outcome"`
	PublishedItemCount *int    `json:"publishedItemCount"`
	HasMore            *bool   `json:"hasMore"`
}

type expectedFailedAttemptV1DTO struct {
	MeaningID   *string `json:"meaningId"`
	StepOrdinal *int    `json:"stepOrdinal"`
	Outcome     *string `json:"outcome"`
	ErrorCode   *string `json:"errorCode"`
}

func decodeExpectedAttemptV1(data []byte) (ExpectedAttempt, error) {
	var discriminator expectedAttemptKindV1DTO
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return nil, fmt.Errorf("expected attempt の outcome を判定できません")
	}
	if discriminator.Outcome == nil {
		return nil, fmt.Errorf("expected attempt の outcome は必須です")
	}
	switch legalquery.LegalQueryAttemptOutcome(*discriminator.Outcome) {
	case legalquery.LegalQueryAttemptOutcomeCompleted:
		if discriminator.HasMore != nil {
			return decodeExpectedCompletedCollectionAttemptV1(data)
		}
		return decodeExpectedCompletedReadAttemptV1(data)
	case legalquery.LegalQueryAttemptOutcomeEmpty:
		return decodeExpectedEmptyAttemptV1(data)
	case legalquery.LegalQueryAttemptOutcomeFailed:
		return decodeExpectedFailedAttemptV1(data)
	default:
		return nil, fmt.Errorf("expected attempt の outcome が定義されていません")
	}
}

func decodeExpectedCompletedReadAttemptV1(
	data []byte,
) (ExpectedCompletedReadAttempt, error) {
	var dto expectedCompletedReadAttemptV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExpectedCompletedReadAttempt{}, err
	}
	if dto.MeaningID == nil ||
		dto.StepOrdinal == nil ||
		dto.Outcome == nil ||
		dto.PublishedItemCount == nil {
		return ExpectedCompletedReadAttempt{},
			fmt.Errorf("completed read attempt の必須項目が不足しています")
	}
	if legalquery.LegalQueryAttemptOutcome(*dto.Outcome) !=
		legalquery.LegalQueryAttemptOutcomeCompleted {
		return ExpectedCompletedReadAttempt{},
			fmt.Errorf("completed read attempt の outcome が一致しません")
	}
	return NewExpectedCompletedReadAttempt(
		ExpectedCompletedReadAttemptValues{
			MeaningID:          *dto.MeaningID,
			StepOrdinal:        *dto.StepOrdinal,
			PublishedItemCount: *dto.PublishedItemCount,
		},
	)
}

func decodeExpectedCompletedCollectionAttemptV1(
	data []byte,
) (ExpectedCompletedCollectionAttempt, error) {
	var dto expectedCompletedCollectionAttemptV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExpectedCompletedCollectionAttempt{}, err
	}
	if dto.MeaningID == nil ||
		dto.StepOrdinal == nil ||
		dto.Outcome == nil ||
		dto.PublishedItemCount == nil ||
		dto.HasMore == nil {
		return ExpectedCompletedCollectionAttempt{},
			fmt.Errorf("completed collection attempt の必須項目が不足しています")
	}
	if legalquery.LegalQueryAttemptOutcome(*dto.Outcome) !=
		legalquery.LegalQueryAttemptOutcomeCompleted {
		return ExpectedCompletedCollectionAttempt{},
			fmt.Errorf("completed collection attempt の outcome が一致しません")
	}
	return NewExpectedCompletedCollectionAttempt(
		ExpectedCompletedCollectionAttemptValues{
			MeaningID:          *dto.MeaningID,
			StepOrdinal:        *dto.StepOrdinal,
			PublishedItemCount: *dto.PublishedItemCount,
			HasMore:            *dto.HasMore,
		},
	)
}

func decodeExpectedEmptyAttemptV1(
	data []byte,
) (ExpectedEmptyAttempt, error) {
	var dto expectedEmptyAttemptV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExpectedEmptyAttempt{}, err
	}
	if dto.MeaningID == nil ||
		dto.StepOrdinal == nil ||
		dto.Outcome == nil ||
		dto.PublishedItemCount == nil ||
		dto.HasMore == nil {
		return ExpectedEmptyAttempt{},
			fmt.Errorf("empty attempt の必須項目が不足しています")
	}
	if legalquery.LegalQueryAttemptOutcome(*dto.Outcome) !=
		legalquery.LegalQueryAttemptOutcomeEmpty {
		return ExpectedEmptyAttempt{}, fmt.Errorf("empty attempt の outcome が一致しません")
	}
	return NewExpectedEmptyAttempt(ExpectedEmptyAttemptValues{
		MeaningID:          *dto.MeaningID,
		StepOrdinal:        *dto.StepOrdinal,
		PublishedItemCount: *dto.PublishedItemCount,
		HasMore:            *dto.HasMore,
	})
}

func decodeExpectedFailedAttemptV1(
	data []byte,
) (ExpectedFailedAttempt, error) {
	var dto expectedFailedAttemptV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExpectedFailedAttempt{}, err
	}
	if dto.MeaningID == nil ||
		dto.StepOrdinal == nil ||
		dto.Outcome == nil ||
		dto.ErrorCode == nil {
		return ExpectedFailedAttempt{},
			fmt.Errorf("failed attempt の必須項目が不足しています")
	}
	if legalquery.LegalQueryAttemptOutcome(*dto.Outcome) !=
		legalquery.LegalQueryAttemptOutcomeFailed {
		return ExpectedFailedAttempt{}, fmt.Errorf("failed attempt の outcome が一致しません")
	}
	return NewExpectedFailedAttempt(ExpectedFailedAttemptValues{
		MeaningID:   *dto.MeaningID,
		StepOrdinal: *dto.StepOrdinal,
		ErrorCode:   model.ErrorCode(*dto.ErrorCode),
	})
}
