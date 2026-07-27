package legalquerycorpus

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type executionExpectedTerminalV1DTO struct {
	Terminal *string `json:"terminal"`
}

type executionExpectedResultV1DTO struct {
	Terminal          *string            `json:"terminal"`
	Status            *string            `json:"status"`
	ReturnedItemCount *int               `json:"returnedItemCount"`
	Attempts          *[]json.RawMessage `json:"attempts"`
}

type executionExpectedErrorV1DTO struct {
	Terminal  *string            `json:"terminal"`
	ErrorCode *string            `json:"errorCode"`
	Attempts  *[]json.RawMessage `json:"attempts"`
}

func decodeExecutionExpectedV1(data []byte) (ExecutionExpected, error) {
	var discriminator executionExpectedTerminalV1DTO
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return nil, fmt.Errorf("execution expected の terminal を判定できません")
	}
	if discriminator.Terminal == nil {
		return nil, fmt.Errorf("execution expected の terminal は必須です")
	}
	switch ExecutionExpectedTerminal(*discriminator.Terminal) {
	case ExecutionExpectedTerminalResult:
		return decodeExecutionExpectedResultV1(data)
	case ExecutionExpectedTerminalError:
		return decodeExecutionExpectedErrorV1(data)
	default:
		return nil, fmt.Errorf("execution expected の terminal が定義されていません")
	}
}

func decodeExecutionExpectedResultV1(
	data []byte,
) (ExecutionExpectedResult, error) {
	var dto executionExpectedResultV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExecutionExpectedResult{}, err
	}
	if dto.Terminal == nil ||
		dto.Status == nil ||
		dto.ReturnedItemCount == nil ||
		dto.Attempts == nil {
		return ExecutionExpectedResult{},
			fmt.Errorf("expected result の必須項目が不足しています")
	}
	if ExecutionExpectedTerminal(*dto.Terminal) !=
		ExecutionExpectedTerminalResult {
		return ExecutionExpectedResult{},
			fmt.Errorf("expected result の terminal が一致しません")
	}
	attempts, err := decodeExpectedAttemptsV1(*dto.Attempts)
	if err != nil {
		return ExecutionExpectedResult{}, err
	}
	return NewExecutionExpectedResult(ExecutionExpectedResultValues{
		Status:            legalquery.LegalQueryResultStatus(*dto.Status),
		ReturnedItemCount: *dto.ReturnedItemCount,
		Attempts:          attempts,
	})
}

func decodeExecutionExpectedErrorV1(
	data []byte,
) (ExecutionExpectedError, error) {
	var dto executionExpectedErrorV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExecutionExpectedError{}, err
	}
	if dto.Terminal == nil || dto.ErrorCode == nil || dto.Attempts == nil {
		return ExecutionExpectedError{},
			fmt.Errorf("expected error の必須項目が不足しています")
	}
	if ExecutionExpectedTerminal(*dto.Terminal) !=
		ExecutionExpectedTerminalError {
		return ExecutionExpectedError{},
			fmt.Errorf("expected error の terminal が一致しません")
	}
	attempts, err := decodeExpectedAttemptsV1(*dto.Attempts)
	if err != nil {
		return ExecutionExpectedError{}, err
	}
	return NewExecutionExpectedError(ExecutionExpectedErrorValues{
		ErrorCode: model.ErrorCode(*dto.ErrorCode),
		Attempts:  attempts,
	})
}

func decodeExpectedAttemptsV1(
	values []json.RawMessage,
) ([]ExpectedAttempt, error) {
	attempts := make([]ExpectedAttempt, 0, len(values))
	for _, value := range values {
		attempt, err := decodeExpectedAttemptV1(value)
		if err != nil {
			return nil, fmt.Errorf("execution expected の attempt を復元できません: %w", err)
		}
		attempts = append(attempts, attempt)
	}
	return attempts, nil
}
