package legalquerycandidateeval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
	"github.com/google/jsonschema-go/jsonschema"
)

const schemaDraft202012 = "https://json-schema.org/draft/2020-12/schema"

// SchemaV2 は外部参照を持たない解決済み JSON Schema である。
type SchemaV2 struct {
	resolved *jsonschema.Resolved
}

// ParseSchemaV2 は Draft 2020-12 schema を同一 document 内だけで解決する。
func ParseSchemaV2(raw []byte) (SchemaV2, error) {
	if len(raw) == 0 || len(raw) > maximumSchemaBytes {
		return SchemaV2{}, fmt.Errorf("candidate evaluation schema の size が不正です")
	}
	if err := legalqueryartifact.InspectJSONObject(raw, legalqueryartifact.JSONLimits{
		Depth: 64, Values: 65536, RejectNull: true,
	}); err != nil {
		return SchemaV2{}, fmt.Errorf("candidate evaluation schema の JSON が不正です: %w", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return SchemaV2{}, fmt.Errorf("candidate evaluation schema を解釈できません")
	}
	if generic["$schema"] != schemaDraft202012 {
		return SchemaV2{}, fmt.Errorf("candidate evaluation schema の draft が不正です")
	}
	if err := validateSchemaReferences(generic); err != nil {
		return SchemaV2{}, err
	}
	var document jsonschema.Schema
	if err := json.Unmarshal(raw, &document); err != nil {
		return SchemaV2{}, fmt.Errorf("candidate evaluation schema を解釈できません")
	}
	resolved, err := document.Resolve(nil)
	if err != nil {
		return SchemaV2{}, fmt.Errorf("candidate evaluation schema を自己解決できません")
	}
	return SchemaV2{resolved: resolved}, nil
}

// Validate は一 artifact を解決済み schema に照合する。
func (s SchemaV2) Validate(ctx context.Context, raw []byte) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	return s.validateRaw(raw)
}

func (s SchemaV2) validateRaw(raw []byte) error {
	if s.resolved == nil {
		return fmt.Errorf("candidate evaluation schema が初期化されていません")
	}
	var instance any
	if err := json.Unmarshal(raw, &instance); err != nil {
		return fmt.Errorf("candidate evaluation 成果物を schema 用に解釈できません")
	}
	if err := s.resolved.Validate(instance); err != nil {
		return fmt.Errorf("candidate evaluation 成果物が schema v2 に適合しません")
	}
	return nil
}

func validateSchemaReferences(value any) error {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			if key == "$ref" {
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#/") {
					return fmt.Errorf("candidate evaluation schema の $ref は内部参照に限ります")
				}
			}
			if key == "$dynamicRef" || key == "$recursiveRef" {
				return fmt.Errorf("candidate evaluation schema で動的参照は使用できません")
			}
			if err := validateSchemaReferences(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range current {
			if err := validateSchemaReferences(child); err != nil {
				return err
			}
		}
	}
	return nil
}
