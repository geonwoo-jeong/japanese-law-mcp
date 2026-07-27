package legalquerycorpus

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type executionActionV1DTO struct {
	MeaningID    *string          `json:"meaningId"`
	StepOrdinal  *int             `json:"stepOrdinal"`
	ReleaseOrder *int             `json:"releaseOrder"`
	Outcome      *json.RawMessage `json:"outcome"`
}

type executionOutcomeKindV1DTO struct {
	Kind *string `json:"kind"`
}

type collectionSuccessOutcomeV1DTO struct {
	Kind            *string `json:"kind"`
	SourceItemCount *int    `json:"sourceItemCount"`
}

type readSuccessOutcomeV1DTO struct {
	Kind *string `json:"kind"`
}

type failureOutcomeV1DTO struct {
	Kind      *string `json:"kind"`
	ErrorCode *string `json:"errorCode"`
}

type timeoutOutcomeV1DTO struct {
	Kind *string `json:"kind"`
}

func decodeExecutionActionV1(data []byte) (ExecutionAction, error) {
	var dto executionActionV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExecutionAction{}, err
	}
	if dto.MeaningID == nil ||
		dto.StepOrdinal == nil ||
		dto.ReleaseOrder == nil ||
		dto.Outcome == nil {
		return ExecutionAction{}, fmt.Errorf("execution action の必須項目が不足しています")
	}
	outcome, err := decodeExecutionOutcomeV1(*dto.Outcome)
	if err != nil {
		return ExecutionAction{}, fmt.Errorf("execution action の outcome を復元できません: %w", err)
	}
	return NewExecutionAction(ExecutionActionValues{
		MeaningID:    *dto.MeaningID,
		StepOrdinal:  *dto.StepOrdinal,
		ReleaseOrder: *dto.ReleaseOrder,
		Outcome:      outcome,
	})
}

func decodeExecutionOutcomeV1(data []byte) (ExecutionOutcome, error) {
	var discriminator executionOutcomeKindV1DTO
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return nil, fmt.Errorf("execution outcome の kind を判定できません")
	}
	if discriminator.Kind == nil {
		return nil, fmt.Errorf("execution outcome の kind は必須です")
	}
	switch ExecutionOutcomeKind(*discriminator.Kind) {
	case ExecutionOutcomeKindCollectionSuccess:
		return decodeCollectionSuccessOutcomeV1(data)
	case ExecutionOutcomeKindReadSuccess:
		return decodeReadSuccessOutcomeV1(data)
	case ExecutionOutcomeKindFailure:
		return decodeFailureOutcomeV1(data)
	case ExecutionOutcomeKindTimeout:
		return decodeTimeoutOutcomeV1(data)
	default:
		return nil, fmt.Errorf("execution outcome の kind が定義されていません")
	}
}

func decodeCollectionSuccessOutcomeV1(
	data []byte,
) (CollectionSuccessOutcome, error) {
	var dto collectionSuccessOutcomeV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return CollectionSuccessOutcome{}, err
	}
	if dto.Kind == nil || dto.SourceItemCount == nil {
		return CollectionSuccessOutcome{}, fmt.Errorf(
			"collection_success outcome の必須項目が不足しています",
		)
	}
	if ExecutionOutcomeKind(*dto.Kind) != ExecutionOutcomeKindCollectionSuccess {
		return CollectionSuccessOutcome{}, fmt.Errorf(
			"collection_success outcome の kind が一致しません",
		)
	}
	return NewCollectionSuccessOutcome(*dto.SourceItemCount)
}

func decodeReadSuccessOutcomeV1(data []byte) (ReadSuccessOutcome, error) {
	var dto readSuccessOutcomeV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ReadSuccessOutcome{}, err
	}
	if dto.Kind == nil {
		return ReadSuccessOutcome{}, fmt.Errorf("read_success outcome の kind は必須です")
	}
	if ExecutionOutcomeKind(*dto.Kind) != ExecutionOutcomeKindReadSuccess {
		return ReadSuccessOutcome{}, fmt.Errorf("read_success outcome の kind が一致しません")
	}
	return NewReadSuccessOutcome()
}

func decodeFailureOutcomeV1(data []byte) (FailureOutcome, error) {
	var dto failureOutcomeV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return FailureOutcome{}, err
	}
	if dto.Kind == nil || dto.ErrorCode == nil {
		return FailureOutcome{}, fmt.Errorf("failure outcome の必須項目が不足しています")
	}
	if ExecutionOutcomeKind(*dto.Kind) != ExecutionOutcomeKindFailure {
		return FailureOutcome{}, fmt.Errorf("failure outcome の kind が一致しません")
	}
	return NewFailureOutcome(model.ErrorCode(*dto.ErrorCode))
}

func decodeTimeoutOutcomeV1(data []byte) (TimeoutOutcome, error) {
	var dto timeoutOutcomeV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return TimeoutOutcome{}, err
	}
	if dto.Kind == nil {
		return TimeoutOutcome{}, fmt.Errorf("timeout outcome の kind は必須です")
	}
	if ExecutionOutcomeKind(*dto.Kind) != ExecutionOutcomeKindTimeout {
		return TimeoutOutcome{}, fmt.Errorf("timeout outcome の kind が一致しません")
	}
	return NewTimeoutOutcome()
}
