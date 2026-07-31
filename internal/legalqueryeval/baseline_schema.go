package legalqueryeval

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
	"github.com/google/jsonschema-go/jsonschema"
)

const baselineSchemaDraft202012 = "https://json-schema.org/draft/2020-12/schema"

type baselineSchemaV1 struct {
	resolved *jsonschema.Resolved
}

func newBaselineSchemaV1(data []byte) (baselineSchemaV1, error) {
	if err := legalqueryartifact.InspectJSONObject(
		data,
		legalqueryartifact.JSONLimits{Depth: 32, Values: 16384, RejectNull: true},
	); err != nil {
		return baselineSchemaV1{}, fmt.Errorf("baseline schema の JSON が不正です: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return baselineSchemaV1{}, fmt.Errorf("baseline schema を解釈できません")
	}
	if raw["$schema"] != baselineSchemaDraft202012 {
		return baselineSchemaV1{}, fmt.Errorf("baseline schema の draft が不正です")
	}
	if err := validateBaselineSchemaReferences(raw); err != nil {
		return baselineSchemaV1{}, err
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return baselineSchemaV1{}, fmt.Errorf("baseline schema を解釈できません")
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return baselineSchemaV1{}, fmt.Errorf("baseline schema を自己解決できません")
	}
	return baselineSchemaV1{resolved: resolved}, nil
}

func validateBaselineSchemaReferences(value any) error {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			switch key {
			case "$ref":
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#/") {
					return fmt.Errorf("baseline schema の $ref は内部参照に限ります")
				}
			case "$dynamicRef":
				return fmt.Errorf("baseline schema で $dynamicRef は使用できません")
			}
			if err := validateBaselineSchemaReferences(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range current {
			if err := validateBaselineSchemaReferences(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s baselineSchemaV1) validate(data []byte) error {
	if s.resolved == nil {
		return fmt.Errorf("baseline schema が初期化されていません")
	}
	var instance any
	if err := json.Unmarshal(data, &instance); err != nil {
		return fmt.Errorf("baseline を schema 検証用に解釈できません")
	}
	if err := s.resolved.Validate(instance); err != nil {
		return fmt.Errorf("baseline が schema v1 に適合しません")
	}
	return nil
}
