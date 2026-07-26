package providerconformance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

const (
	yamlMapTag    = "!!map"
	yamlSeqTag    = "!!seq"
	yamlStringTag = "!!str"
	yamlBoolTag   = "!!bool"
	yamlIntTag    = "!!int"
	yamlFloatTag  = "!!float"
	yamlNullTag   = "!!null"
	yamlMergeTag  = "!!merge"
)

func decodeStrictYAML(data []byte) (any, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("YAML は UTF-8 で記述してください")
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("YAML document がありません")
		}
		return nil, fmt.Errorf("YAML を解釈できません: %w", err)
	}

	var extra yaml.Node
	switch err := decoder.Decode(&extra); {
	case err == nil:
		return nil, fmt.Errorf("YAML document は一つだけ記述してください")
	case !errors.Is(err, io.EOF):
		return nil, fmt.Errorf("YAML の第二 document を確認できません: %w", err)
	}

	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return nil, fmt.Errorf("YAML の最上位 document が不正です")
	}
	value, err := yamlNodeToJSON(document.Content[0], "$")
	if err != nil {
		return nil, err
	}
	return value, nil
}

func yamlNodeToJSON(node *yaml.Node, path string) (any, error) {
	if node.Anchor != "" {
		return nil, fmt.Errorf("%s: anchor は使用できません", path)
	}
	if node.Kind == yaml.AliasNode || node.Alias != nil {
		return nil, fmt.Errorf("%s: alias は使用できません", path)
	}

	switch node.Kind {
	case yaml.MappingNode:
		if node.ShortTag() != yamlMapTag {
			return nil, fmt.Errorf("%s: custom tag %q は使用できません", path, node.Tag)
		}
		if len(node.Content)%2 != 0 {
			return nil, fmt.Errorf("%s: mapping の key と value が対応していません", path)
		}
		result := make(map[string]any, len(node.Content)/2)
		keys := make(map[string]struct{}, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			if keyNode.Anchor != "" || keyNode.Kind == yaml.AliasNode || keyNode.Alias != nil {
				return nil, fmt.Errorf("%s: mapping key に anchor または alias は使用できません", path)
			}
			if keyNode.Kind != yaml.ScalarNode || keyNode.ShortTag() != yamlStringTag {
				if keyNode.ShortTag() == yamlMergeTag {
					return nil, fmt.Errorf("%s: merge key は使用できません", path)
				}
				return nil, fmt.Errorf("%s: mapping key は文字列で記述してください", path)
			}
			key := keyNode.Value
			if _, exists := keys[key]; exists {
				return nil, fmt.Errorf("%s: key %q が重複しています", path, key)
			}
			keys[key] = struct{}{}
			value, err := yamlNodeToJSON(node.Content[i+1], path+"."+key)
			if err != nil {
				return nil, err
			}
			result[key] = value
		}
		return result, nil

	case yaml.SequenceNode:
		if node.ShortTag() != yamlSeqTag {
			return nil, fmt.Errorf("%s: custom tag %q は使用できません", path, node.Tag)
		}
		result := make([]any, len(node.Content))
		for i, child := range node.Content {
			value, err := yamlNodeToJSON(child, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			result[i] = value
		}
		return result, nil

	case yaml.ScalarNode:
		return yamlScalarToJSON(node, path)

	default:
		return nil, fmt.Errorf("%s: YAML node 種別 %d は使用できません", path, node.Kind)
	}
}

func yamlScalarToJSON(node *yaml.Node, path string) (any, error) {
	switch node.ShortTag() {
	case yamlStringTag:
		return node.Value, nil
	case yamlBoolTag, yamlIntTag, yamlFloatTag, yamlNullTag:
		var value any
		decoder := json.NewDecoder(bytes.NewBufferString(node.Value))
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("%s: JSON と互換性のない scalar %q は使用できません", path, node.Value)
		}
		if err := ensureJSONEnd(decoder); err != nil {
			return nil, fmt.Errorf("%s: JSON と互換性のない scalar %q は使用できません", path, node.Value)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("%s: custom tag または JSON と互換性のない tag %q は使用できません", path, node.Tag)
	}
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("scalar の後ろに値があります")
		}
		return err
	}
	return nil
}
