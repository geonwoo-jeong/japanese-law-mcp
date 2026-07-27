package legalquerycorpus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

const corpusSchemaDraft202012 = "https://json-schema.org/draft/2020-12/schema"

// corpusSchemaV1 は、外部参照を持たない解決済みの v1 schema を保持する。
type corpusSchemaV1 struct {
	resolved *jsonschema.Resolved
}

// newCorpusSchemaV1 は、安全境界を満たす固定 v1 schema を解決する。
func newCorpusSchemaV1(data []byte) (corpusSchemaV1, error) {
	if len(data) > int(corpusSchemaMaximumBytes) {
		return corpusSchemaV1{}, fmt.Errorf(
			"corpus schema v1 は size 上限以下でなければなりません",
		)
	}
	if _, err := inspectJSONObject(data, false); err != nil {
		return corpusSchemaV1{}, fmt.Errorf(
			"corpus schema v1 の JSON 安全境界が有効ではありません",
		)
	}
	raw, err := decodeCorpusSchemaDocument(data)
	if err != nil {
		return corpusSchemaV1{}, err
	}
	if err := validateCorpusSchemaDraft(raw); err != nil {
		return corpusSchemaV1{}, err
	}
	if err := validateCorpusSchemaReferences(raw); err != nil {
		return corpusSchemaV1{}, err
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return corpusSchemaV1{}, fmt.Errorf(
			"corpus schema v1 を解釈できません",
		)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return corpusSchemaV1{}, fmt.Errorf(
			"corpus schema v1 を自己解決できません",
		)
	}
	return corpusSchemaV1{resolved: resolved}, nil
}

func decodeCorpusSchemaDocument(data []byte) (map[string]any, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("corpus schema v1 を解釈できません")
	}
	if raw == nil {
		return nil, fmt.Errorf("corpus schema v1 は object でなければなりません")
	}
	return raw, nil
}

func validateCorpusSchemaDraft(raw map[string]any) error {
	draft, ok := raw["$schema"].(string)
	if !ok || draft != corpusSchemaDraft202012 {
		return fmt.Errorf(
			"corpus schema v1 は JSON Schema Draft 2020-12 でなければなりません",
		)
	}
	return nil
}

func validateCorpusSchemaReferences(value any) error {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			switch key {
			case "$ref":
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#/") {
					return fmt.Errorf(
						"corpus schema v1 の $ref は local fragment でなければなりません",
					)
				}
			case "$dynamicRef":
				return fmt.Errorf(
					"corpus schema v1 では $dynamicRef を使用できません",
				)
			}
			if err := validateCorpusSchemaReferences(child); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range current {
			if err := validateCorpusSchemaReferences(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s corpusSchemaV1) validate(data []byte) error {
	if s.resolved == nil {
		return fmt.Errorf("corpus schema v1 が初期化されていません")
	}
	var instance any
	if err := json.Unmarshal(data, &instance); err != nil {
		return fmt.Errorf("JSON 成果物を schema 検証用に解釈できません")
	}
	if err := s.resolved.Validate(instance); err != nil {
		return fmt.Errorf("JSON 成果物が corpus schema v1 に適合しません")
	}
	return nil
}
