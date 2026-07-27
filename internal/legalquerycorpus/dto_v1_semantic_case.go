package legalquerycorpus

import (
	"encoding/json"
	"fmt"
)

type semanticCaseV1DTO struct {
	ArtifactKind   *string          `json:"artifactKind"`
	SchemaVersion  *int             `json:"schemaVersion"`
	CaseID         *string          `json:"caseId"`
	LeakageGroupID *string          `json:"leakageGroupId"`
	CoverageIDs    *[]string        `json:"coverageIds"`
	SafetyVariant  *string          `json:"safetyVariant,omitempty"`
	EnabledPacks   *[]string        `json:"enabledPacks"`
	Request        *rawRequestV1DTO `json:"request"`
	Expected       *json.RawMessage `json:"expected"`
}

func decodeSemanticCaseV1(data []byte) (SemanticCase, error) {
	var dto semanticCaseV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return SemanticCase{}, err
	}
	return convertSemanticCaseV1(dto)
}

func convertSemanticCaseV1(dto semanticCaseV1DTO) (SemanticCase, error) {
	if dto.ArtifactKind == nil ||
		dto.SchemaVersion == nil ||
		dto.CaseID == nil ||
		dto.LeakageGroupID == nil ||
		dto.CoverageIDs == nil ||
		dto.EnabledPacks == nil ||
		dto.Request == nil ||
		dto.Expected == nil {
		return SemanticCase{}, fmt.Errorf("semantic case の必須項目が不足しています")
	}
	request, err := convertRawRequestV1(*dto.Request)
	if err != nil {
		return SemanticCase{}, fmt.Errorf("semantic case の request を復元できません: %w", err)
	}
	expected, err := decodeSemanticExpectedV1(*dto.Expected)
	if err != nil {
		return SemanticCase{}, fmt.Errorf("semantic case の expected を復元できません: %w", err)
	}
	return NewSemanticCase(SemanticCaseValues{
		ArtifactKind:   ArtifactKind(*dto.ArtifactKind),
		SchemaVersion:  *dto.SchemaVersion,
		CaseID:         *dto.CaseID,
		LeakageGroupID: *dto.LeakageGroupID,
		CoverageIDs:    *dto.CoverageIDs,
		SafetyVariant:  convertSafetyVariantV1(dto.SafetyVariant),
		EnabledPacks:   *dto.EnabledPacks,
		Request:        request,
		Expected:       expected,
	})
}

func convertSafetyVariantV1(value *string) *SafetyVariant {
	if value == nil {
		return nil
	}
	converted := SafetyVariant(*value)
	return &converted
}
