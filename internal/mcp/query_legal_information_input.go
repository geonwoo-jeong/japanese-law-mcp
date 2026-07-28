package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const queryLegalInformationMaxArgumentsBytes = 16 * 1024

func decodeQueryLegalInformationInput(
	arguments json.RawMessage,
) (legalquery.Request, error) {
	if len(arguments) > queryLegalInformationMaxArgumentsBytes {
		return legalquery.Request{}, newQueryLegalInformationInputError(
			"arguments",
			"は UTF-8 JSON で 16384 byte 以下でなければなりません",
		)
	}
	fields, err := decodeQueryLegalInformationObject(
		arguments,
		map[string]struct{}{
			"query":           {},
			"ref":             {},
			"limitPerAttempt": {},
		},
		"arguments",
	)
	if err != nil {
		return legalquery.Request{}, err
	}

	query, err := decodeQueryLegalInformationRequiredString(
		fields,
		"query",
		"query",
	)
	if err != nil {
		return legalquery.Request{}, err
	}
	values := legalquery.RequestValues{Query: query}
	if raw, exists := fields["limitPerAttempt"]; exists {
		if queryLegalInformationJSONNull(raw) {
			return legalquery.Request{}, newQueryLegalInformationInputError(
				"limitPerAttempt",
				"に null は使用できません",
			)
		}
		limit, limitErr := decodeQueryLegalInformationLimit(raw)
		if limitErr != nil {
			return legalquery.Request{}, newQueryLegalInformationInputError(
				"limitPerAttempt",
				"は 1 以上 20 以下の整数でなければなりません",
			)
		}
		values.LimitPerAttempt = &limit
	}
	if raw, exists := fields["ref"]; exists {
		if queryLegalInformationJSONNull(raw) {
			return legalquery.Request{}, newQueryLegalInformationInputError(
				"ref",
				"に null は使用できません",
			)
		}
		ref, refErr := decodeQueryLegalInformationRef(raw)
		if refErr != nil {
			return legalquery.Request{}, refErr
		}
		values.Ref = &ref
	}
	return legalquery.NewRequest(values)
}

func decodeQueryLegalInformationRef(
	raw json.RawMessage,
) (model.SourceResourceRef, error) {
	fields, err := decodeQueryLegalInformationObject(
		raw,
		map[string]struct{}{
			"providerId": {},
			"key":        {},
		},
		"ref",
	)
	if err != nil {
		return model.SourceResourceRef{}, err
	}
	providerID, err := decodeQueryLegalInformationRequiredString(
		fields,
		"providerId",
		"ref",
	)
	if err != nil {
		return model.SourceResourceRef{}, err
	}
	keyRaw, exists := fields["key"]
	if !exists || queryLegalInformationJSONNull(keyRaw) {
		return model.SourceResourceRef{}, newQueryLegalInformationInputError(
			"ref",
			"の key は必須です",
		)
	}
	key, err := decodeQueryLegalInformationResourceKey(keyRaw)
	if err != nil {
		return model.SourceResourceRef{}, err
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return model.SourceResourceRef{}, newQueryLegalInformationInputError(
			"ref",
			"は有効な SourceResourceRef でなければなりません",
		)
	}
	return ref, nil
}

func decodeQueryLegalInformationResourceKey(
	raw json.RawMessage,
) (model.SourceResourceKey, error) {
	fields, err := decodeQueryLegalInformationObject(
		raw,
		map[string]struct{}{
			"sourceId":     {},
			"resourceType": {},
			"resourceId":   {},
			"versionId":    {},
		},
		"ref",
	)
	if err != nil {
		return model.SourceResourceKey{}, err
	}
	sourceID, err := decodeQueryLegalInformationRequiredString(
		fields,
		"sourceId",
		"ref",
	)
	if err != nil {
		return model.SourceResourceKey{}, err
	}
	resourceType, err := decodeQueryLegalInformationRequiredString(
		fields,
		"resourceType",
		"ref",
	)
	if err != nil {
		return model.SourceResourceKey{}, err
	}
	resourceID, err := decodeQueryLegalInformationRequiredString(
		fields,
		"resourceId",
		"ref",
	)
	if err != nil {
		return model.SourceResourceKey{}, err
	}
	versionID := ""
	if _, exists := fields["versionId"]; exists {
		versionID, err = decodeQueryLegalInformationRequiredString(
			fields,
			"versionId",
			"ref",
		)
		if err != nil {
			return model.SourceResourceKey{}, err
		}
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		return model.SourceResourceKey{}, newQueryLegalInformationInputError(
			"ref",
			"の key が有効ではありません",
		)
	}
	return key, nil
}

func decodeQueryLegalInformationRequiredString(
	fields map[string]json.RawMessage,
	name string,
	errorField string,
) (string, error) {
	raw, exists := fields[name]
	if !exists || queryLegalInformationJSONNull(raw) {
		return "", newQueryLegalInformationInputError(
			errorField,
			"の "+name+" は必須です",
		)
	}
	var value string
	if err := decodeStrictJSONString(raw, &value); err != nil {
		return "", newQueryLegalInformationInputError(
			errorField,
			"の "+name+" は文字列でなければなりません",
		)
	}
	if value == "" {
		return "", newQueryLegalInformationInputError(
			errorField,
			"の "+name+" は空にできません",
		)
	}
	return value, nil
}

func decodeQueryLegalInformationLimit(raw json.RawMessage) (int, error) {
	value := bytes.TrimSpace(raw)
	if len(value) == 0 ||
		(value[0] != '-' && (value[0] < '0' || value[0] > '9')) {
		return 0, fmt.Errorf("JSON number ではありません")
	}
	var number json.Number
	if err := json.Unmarshal(value, &number); err != nil {
		return 0, fmt.Errorf("JSON number ではありません")
	}
	limit, err := exactSmallInteger(number.String())
	if err != nil ||
		limit < 1 ||
		limit > legalquery.MaxLimitPerAttempt {
		return 0, fmt.Errorf("1 以上 20 以下の整数ではありません")
	}
	return limit, nil
}

func decodeQueryLegalInformationObject(
	raw json.RawMessage,
	allowed map[string]struct{},
	errorField string,
) (map[string]json.RawMessage, error) {
	value := bytes.TrimSpace(raw)
	if len(value) == 0 || !utf8.Valid(value) {
		return nil, newQueryLegalInformationInputError(
			errorField,
			"は JSON object でなければなりません",
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	token, err := decoder.Token()
	if err != nil {
		return nil, newQueryLegalInformationInputError(
			errorField,
			"は JSON object でなければなりません",
		)
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter || delimiter != '{' {
		return nil, newQueryLegalInformationInputError(
			errorField,
			"は JSON object でなければなりません",
		)
	}

	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		nameToken, nameErr := decoder.Token()
		name, isString := nameToken.(string)
		if nameErr != nil || !isString {
			return nil, newQueryLegalInformationInputError(
				errorField,
				"は JSON object でなければなりません",
			)
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, newQueryLegalInformationInputError(
				errorField,
				"に同じ項目を複数指定できません",
			)
		}
		if _, exists := allowed[name]; !exists {
			return nil, newQueryLegalInformationInputError(
				errorField,
				"に定義していない項目は使用できません",
			)
		}
		var field json.RawMessage
		if err := decoder.Decode(&field); err != nil {
			return nil, newQueryLegalInformationInputError(
				errorField,
				"は JSON object でなければなりません",
			)
		}
		fields[name] = append(json.RawMessage{}, field...)
	}
	if _, err := decoder.Token(); err != nil {
		return nil, newQueryLegalInformationInputError(
			errorField,
			"は JSON object でなければなりません",
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, newQueryLegalInformationInputError(
			errorField,
			"は一つの JSON object でなければなりません",
		)
	}
	return fields, nil
}

type queryLegalInformationInputError struct {
	field  string
	reason string
}

func (err queryLegalInformationInputError) Error() string {
	return err.field + " " + err.reason
}

func newQueryLegalInformationInputError(
	field string,
	reason string,
) queryLegalInformationInputError {
	return queryLegalInformationInputError{
		field:  field,
		reason: reason,
	}
}

func queryLegalInformationJSONNull(value json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}
