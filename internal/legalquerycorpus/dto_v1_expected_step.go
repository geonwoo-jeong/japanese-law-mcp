package legalquerycorpus

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

type expectedStepV1DTO struct {
	Task         *string         `json:"task"`
	Resource     *string         `json:"resource"`
	InputKind    *string         `json:"inputKind"`
	LogicalInput json.RawMessage `json:"logicalInput"`
}

func decodeExpectedStepV1(data []byte) (ExpectedStep, error) {
	var dto expectedStepV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExpectedStep{}, err
	}
	return convertExpectedStepV1(dto)
}

func convertExpectedStepV1(dto expectedStepV1DTO) (ExpectedStep, error) {
	if dto.Task == nil ||
		dto.Resource == nil ||
		dto.InputKind == nil ||
		dto.LogicalInput == nil {
		return ExpectedStep{}, fmt.Errorf("expected step の必須項目が不足しています")
	}
	inputKind := legalquery.LogicalInputKind(*dto.InputKind)
	input, err := convertExpectedLogicalInputV1(inputKind, dto.LogicalInput)
	if err != nil {
		return ExpectedStep{}, err
	}
	return NewExpectedStep(ExpectedStepValues{
		Task:         legalquery.Task(*dto.Task),
		Resource:     legalquery.Resource(*dto.Resource),
		InputKind:    inputKind,
		LogicalInput: input,
	})
}
