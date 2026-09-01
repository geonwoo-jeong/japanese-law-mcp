package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	discoverLegalToolsMaxArgumentsBytes = 16 * 1024
	executeLegalToolMaxArgumentsBytes   = 64 * 1024
	discoverLegalToolsDefaultLimit      = 5
	discoverLegalToolsMaxLimit          = 16
	discoverLegalToolsMaxQueryBytes     = 256
)

type metaToolInputError struct {
	field  string
	reason string
}

func (err metaToolInputError) Error() string {
	return err.field + " " + err.reason
}

func newMetaToolInputError(field, reason string) metaToolInputError {
	return metaToolInputError{field: field, reason: reason}
}

func decodeStrictMetaToolObject(
	arguments json.RawMessage,
	maxBytes int,
	allowed map[string]struct{},
) (map[string]json.RawMessage, error) {
	if len(arguments) > maxBytes {
		return nil, newMetaToolInputError(
			"arguments",
			fmt.Sprintf("は UTF-8 JSON で %d byte 以下でなければなりません", maxBytes),
		)
	}
	value := bytes.TrimSpace(arguments)
	if len(value) == 0 || !utf8.Valid(value) || !hasValidJSONSurrogatePairs(value) {
		return nil, newMetaToolInputError(
			"arguments",
			"は有効な UTF-8 の JSON object でなければなりません",
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	start, err := decoder.Token()
	delimiter, isDelimiter := start.(json.Delim)
	if err != nil || !isDelimiter || delimiter != '{' {
		return nil, newMetaToolInputError(
			"arguments",
			"は JSON object でなければなりません",
		)
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		nameToken, nameErr := decoder.Token()
		name, isString := nameToken.(string)
		if nameErr != nil || !isString {
			return nil, newMetaToolInputError(
				"arguments",
				"は JSON object でなければなりません",
			)
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, newMetaToolInputError(
				"arguments",
				"に同じ項目を複数指定できません",
			)
		}
		if _, exists := allowed[name]; !exists {
			return nil, newMetaToolInputError(
				"arguments",
				"に定義していない項目は使用できません",
			)
		}
		var raw json.RawMessage
		if decodeErr := decoder.Decode(&raw); decodeErr != nil {
			return nil, newMetaToolInputError(
				"arguments",
				"は JSON object でなければなりません",
			)
		}
		fields[name] = append(json.RawMessage(nil), raw...)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, newMetaToolInputError(
			"arguments",
			"は JSON object でなければなりません",
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, newMetaToolInputError(
			"arguments",
			"は一つの JSON object でなければなりません",
		)
	}
	return fields, nil
}

func validateJSONObjectWithoutDuplicateKeys(raw json.RawMessage) error {
	value := bytes.TrimSpace(raw)
	if len(value) == 0 || !utf8.Valid(value) {
		return newMetaToolInputError(
			"arguments",
			"は有効な UTF-8 の JSON object でなければなりません",
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.UseNumber()
	if err := scanJSONValueWithoutDuplicateKeys(decoder, true); err != nil {
		return newMetaToolInputError(
			"arguments",
			"は重複項目のない JSON object でなければなりません",
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return newMetaToolInputError(
			"arguments",
			"は一つの JSON object でなければなりません",
		)
	}
	return nil
}

func scanJSONValueWithoutDuplicateKeys(
	decoder *json.Decoder,
	requireObject bool,
) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if requireObject && (!isDelimiter || delimiter != '{') {
		return fmt.Errorf("JSON object ではありません")
	}
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		names := make(map[string]struct{})
		for decoder.More() {
			nameToken, nameErr := decoder.Token()
			name, isString := nameToken.(string)
			if nameErr != nil || !isString {
				return fmt.Errorf("JSON object の項目名が有効ではありません")
			}
			if _, duplicate := names[name]; duplicate {
				return fmt.Errorf("JSON object の項目 %q が重複しています", name)
			}
			names[name] = struct{}{}
			if err := scanJSONValueWithoutDuplicateKeys(decoder, false); err != nil {
				return err
			}
		}
		end, endErr := decoder.Token()
		if endErr != nil || end != json.Delim('}') {
			return fmt.Errorf("JSON object が閉じていません")
		}
		return nil
	case '[':
		for decoder.More() {
			if err := scanJSONValueWithoutDuplicateKeys(decoder, false); err != nil {
				return err
			}
		}
		end, endErr := decoder.Token()
		if endErr != nil || end != json.Delim(']') {
			return fmt.Errorf("JSON array が閉じていません")
		}
		return nil
	default:
		return fmt.Errorf("JSON delimiter が有効ではありません")
	}
}

func decodeMetaToolString(
	fields map[string]json.RawMessage,
	name string,
	required bool,
) (string, bool, error) {
	raw, exists := fields[name]
	if !exists {
		if required {
			return "", false, newMetaToolInputError(name, "は必須です")
		}
		return "", false, nil
	}
	if queryLegalInformationJSONNull(raw) {
		return "", false, newMetaToolInputError(name, "に null は使用できません")
	}
	var value string
	if err := decodeStrictJSONString(raw, &value); err != nil {
		return "", false, newMetaToolInputError(name, "は文字列でなければなりません")
	}
	return value, true, nil
}

func decodeDiscoverLegalToolsLimit(
	fields map[string]json.RawMessage,
) (int, error) {
	raw, exists := fields["limit"]
	if !exists {
		return discoverLegalToolsDefaultLimit, nil
	}
	if queryLegalInformationJSONNull(raw) {
		return 0, newMetaToolInputError("limit", "に null は使用できません")
	}
	value := bytes.TrimSpace(raw)
	if len(value) == 0 ||
		(value[0] != '-' && (value[0] < '0' || value[0] > '9')) {
		return 0, newMetaToolInputError("limit", "は 1 以上 16 以下の整数でなければなりません")
	}
	var number json.Number
	if err := json.Unmarshal(value, &number); err != nil {
		return 0, newMetaToolInputError("limit", "は 1 以上 16 以下の整数でなければなりません")
	}
	limit, err := exactSmallInteger(number.String())
	if err != nil || limit < 1 || limit > discoverLegalToolsMaxLimit {
		return 0, newMetaToolInputError("limit", "は 1 以上 16 以下の整数でなければなりません")
	}
	return limit, nil
}

func validateDiscoverLegalToolsQuery(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 || len(trimmed) > discoverLegalToolsMaxQueryBytes {
		return "", newMetaToolInputError(
			"query",
			"は trim 後に UTF-8 で 1 byte 以上 256 byte 以下でなければなりません",
		)
	}
	for _, character := range trimmed {
		if character <= '\u001f' || character == '\u007f' {
			return "", newMetaToolInputError(
				"query",
				"に ASCII 制御文字を含めることはできません",
			)
		}
	}
	return trimmed, nil
}

func metaToolInvalidArgumentResult(err error) (*sdk.CallToolResult, error) {
	var inputError metaToolInputError
	if !errors.As(err, &inputError) {
		return errorToolResult(newInternalErrorResult())
	}
	result, buildErr := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeInvalidArgument,
		Details: map[string]any{
			"field":  inputError.field,
			"reason": inputError.reason,
		},
	})
	if buildErr != nil {
		return errorToolResult(newInternalErrorResult())
	}
	return errorToolResult(result)
}
