package legalqueryadoption

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
	"github.com/google/jsonschema-go/jsonschema"
)

const adoptionSchemaDraft202012 = "https://json-schema.org/draft/2020-12/schema"

type adoptionSchemaV1 struct {
	resolved *jsonschema.Resolved
}

func newAdoptionSchemaV1(data []byte) (adoptionSchemaV1, error) {
	if err := legalqueryartifact.InspectJSONObject(
		data,
		legalqueryartifact.JSONLimits{Depth: 32, Values: 16384, RejectNull: true},
	); err != nil {
		return adoptionSchemaV1{}, fmt.Errorf("adoption schema の JSON が不正です: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return adoptionSchemaV1{}, fmt.Errorf("adoption schema を解釈できません")
	}
	if raw["$schema"] != adoptionSchemaDraft202012 {
		return adoptionSchemaV1{}, fmt.Errorf("adoption schema の draft が不正です")
	}
	if err := validateAdoptionSchemaReferences(raw); err != nil {
		return adoptionSchemaV1{}, err
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return adoptionSchemaV1{}, fmt.Errorf("adoption schema を解釈できません")
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return adoptionSchemaV1{}, fmt.Errorf("adoption schema を自己解決できません")
	}
	return adoptionSchemaV1{resolved: resolved}, nil
}

func validateAdoptionSchemaReferences(value any) error {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			switch key {
			case "$ref":
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#/") {
					return fmt.Errorf("adoption schema の $ref は内部参照に限ります")
				}
			case "$dynamicRef":
				return fmt.Errorf("adoption schema で $dynamicRef は使用できません")
			}
			if err := validateAdoptionSchemaReferences(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range current {
			if err := validateAdoptionSchemaReferences(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s adoptionSchemaV1) validate(data []byte) error {
	if s.resolved == nil {
		return fmt.Errorf("adoption schema が初期化されていません")
	}
	var instance any
	if err := json.Unmarshal(data, &instance); err != nil {
		return fmt.Errorf("adoption 成果物を schema 検証用に解釈できません")
	}
	if err := s.resolved.Validate(instance); err != nil {
		return fmt.Errorf("adoption 成果物が schema v1 に適合しません")
	}
	return nil
}
