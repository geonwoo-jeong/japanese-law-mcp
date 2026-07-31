package legalquerycorpus

import (
	"encoding/json"
	"fmt"
)

type executionCaseDTO struct {
	ArtifactKind   *string            `json:"artifactKind"`
	SchemaVersion  *int               `json:"schemaVersion"`
	CaseID         *string            `json:"caseId"`
	ScenarioIDs    *[]string          `json:"scenarioIds"`
	SemanticCaseID *string            `json:"semanticCaseId"`
	Actions        *[]json.RawMessage `json:"actions"`
	Expected       *json.RawMessage   `json:"expected"`
}

func decodeExecutionCaseV1(data []byte) (ExecutionCase, error) {
	return decodeExecutionCase(data)
}

func decodeExecutionCaseV2(data []byte) (ExecutionCase, error) {
	return decodeExecutionCase(data)
}

func decodeExecutionCase(data []byte) (ExecutionCase, error) {
	var dto executionCaseDTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return ExecutionCase{}, err
	}
	if dto.ArtifactKind == nil ||
		dto.SchemaVersion == nil ||
		dto.CaseID == nil ||
		dto.ScenarioIDs == nil ||
		dto.SemanticCaseID == nil ||
		dto.Actions == nil ||
		dto.Expected == nil {
		return ExecutionCase{}, fmt.Errorf("execution case の必須項目が不足しています")
	}
	actions, err := decodeExecutionActionsV1(*dto.Actions)
	if err != nil {
		return ExecutionCase{}, err
	}
	expected, err := decodeExecutionExpectedV1(*dto.Expected)
	if err != nil {
		return ExecutionCase{},
			fmt.Errorf("execution case の expected を復元できません: %w", err)
	}
	return NewExecutionCase(ExecutionCaseValues{
		ArtifactKind:   ArtifactKind(*dto.ArtifactKind),
		SchemaVersion:  *dto.SchemaVersion,
		CaseID:         *dto.CaseID,
		ScenarioIDs:    *dto.ScenarioIDs,
		SemanticCaseID: *dto.SemanticCaseID,
		Actions:        actions,
		Expected:       expected,
	})
}

func decodeExecutionActionsV1(
	values []json.RawMessage,
) ([]ExecutionAction, error) {
	actions := make([]ExecutionAction, 0, len(values))
	for _, value := range values {
		action, err := decodeExecutionActionV1(value)
		if err != nil {
			return nil, fmt.Errorf("execution case の action を復元できません: %w", err)
		}
		actions = append(actions, action)
	}
	return actions, nil
}
