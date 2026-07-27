package legalquerycorpus

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type semanticExpectedKindV1DTO struct {
	Kind *string `json:"kind"`
}

type expectedMeaningV1DTO struct {
	MeaningID     *string            `json:"meaningId"`
	EvidenceCodes *[]string          `json:"evidenceCodes"`
	ConceptIDs    *[]string          `json:"conceptIds"`
	RequiredPacks *[]string          `json:"requiredPacks"`
	Steps         *[]json.RawMessage `json:"steps"`
}

type expectedPlanV1DTO struct {
	Kind               *string                 `json:"kind"`
	Decision           *string                 `json:"decision"`
	ReasonCodes        *[]string               `json:"reasonCodes"`
	Meanings           *[]expectedMeaningV1DTO `json:"meanings"`
	SelectedMeaningIDs *[]string               `json:"selectedMeaningIds"`
}

type expectedRequestErrorV1DTO struct {
	Kind      *string `json:"kind"`
	ErrorCode *string `json:"errorCode"`
	Field     *string `json:"field"`
}

func decodeSemanticExpectedV1(data []byte) (SemanticExpected, error) {
	var discriminator semanticExpectedKindV1DTO
	if err := json.Unmarshal(data, &discriminator); err != nil {
		return nil, fmt.Errorf("semantic expected の kind を判定できません")
	}
	if discriminator.Kind == nil {
		return nil, fmt.Errorf("semantic expected の kind は必須です")
	}
	switch SemanticExpectedKind(*discriminator.Kind) {
	case SemanticExpectedKindPlan:
		return decodeExpectedPlanV1(data)
	case SemanticExpectedKindRequestError:
		return decodeExpectedRequestErrorV1(data)
	default:
		return nil, fmt.Errorf("semantic expected の kind が定義されていません")
	}
}

func decodeExpectedPlanV1(data []byte) (ExpectedPlan, error) {
	var dto expectedPlanV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExpectedPlan{}, err
	}
	return convertExpectedPlanV1(dto)
}

func convertExpectedPlanV1(dto expectedPlanV1DTO) (ExpectedPlan, error) {
	if dto.Kind == nil ||
		dto.Decision == nil ||
		dto.ReasonCodes == nil ||
		dto.Meanings == nil ||
		dto.SelectedMeaningIDs == nil {
		return ExpectedPlan{}, fmt.Errorf("expected plan の必須項目が不足しています")
	}
	if SemanticExpectedKind(*dto.Kind) != SemanticExpectedKindPlan {
		return ExpectedPlan{}, fmt.Errorf("expected plan の kind が一致しません")
	}
	meanings, err := convertExpectedMeaningsV1(*dto.Meanings)
	if err != nil {
		return ExpectedPlan{}, err
	}
	return NewExpectedPlan(ExpectedPlanValues{
		Decision:           legalquery.PlanDecision(*dto.Decision),
		ReasonCodes:        convertExpectedReasonCodes(*dto.ReasonCodes),
		Meanings:           meanings,
		SelectedMeaningIDs: *dto.SelectedMeaningIDs,
	})
}

func convertExpectedMeaningsV1(
	values []expectedMeaningV1DTO,
) ([]ExpectedMeaning, error) {
	meanings := make([]ExpectedMeaning, 0, len(values))
	for _, value := range values {
		meaning, err := convertExpectedMeaningV1(value)
		if err != nil {
			return nil, err
		}
		meanings = append(meanings, meaning)
	}
	return meanings, nil
}

func convertExpectedMeaningV1(
	dto expectedMeaningV1DTO,
) (ExpectedMeaning, error) {
	if dto.MeaningID == nil ||
		dto.EvidenceCodes == nil ||
		dto.ConceptIDs == nil ||
		dto.RequiredPacks == nil ||
		dto.Steps == nil {
		return ExpectedMeaning{}, fmt.Errorf("expected meaning の必須項目が不足しています")
	}
	steps, err := convertExpectedStepsV1(*dto.Steps)
	if err != nil {
		return ExpectedMeaning{}, err
	}
	return NewExpectedMeaning(ExpectedMeaningValues{
		MeaningID:     *dto.MeaningID,
		EvidenceCodes: convertExpectedEvidenceCodes(*dto.EvidenceCodes),
		ConceptIDs:    *dto.ConceptIDs,
		RequiredPacks: *dto.RequiredPacks,
		Steps:         steps,
	})
}

func convertExpectedStepsV1(values []json.RawMessage) ([]ExpectedStep, error) {
	steps := make([]ExpectedStep, 0, len(values))
	for _, value := range values {
		step, err := decodeExpectedStepV1(value)
		if err != nil {
			return nil, fmt.Errorf("expected meaning の step を復元できません: %w", err)
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func decodeExpectedRequestErrorV1(
	data []byte,
) (ExpectedRequestError, error) {
	var dto expectedRequestErrorV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExpectedRequestError{}, err
	}
	if dto.Kind == nil || dto.ErrorCode == nil || dto.Field == nil {
		return ExpectedRequestError{}, fmt.Errorf("expected request error の必須項目が不足しています")
	}
	if SemanticExpectedKind(*dto.Kind) != SemanticExpectedKindRequestError {
		return ExpectedRequestError{}, fmt.Errorf("expected request error の kind が一致しません")
	}
	return NewExpectedRequestError(ExpectedRequestErrorValues{
		ErrorCode: model.ErrorCode(*dto.ErrorCode),
		Field:     RequestErrorField(*dto.Field),
	})
}

func convertExpectedEvidenceCodes(
	values []string,
) []legalquery.EvidenceCode {
	converted := make([]legalquery.EvidenceCode, 0, len(values))
	for _, value := range values {
		converted = append(converted, legalquery.EvidenceCode(value))
	}
	return converted
}

func convertExpectedReasonCodes(values []string) []legalquery.ReasonCode {
	converted := make([]legalquery.ReasonCode, 0, len(values))
	for _, value := range values {
		converted = append(converted, legalquery.ReasonCode(value))
	}
	return converted
}
