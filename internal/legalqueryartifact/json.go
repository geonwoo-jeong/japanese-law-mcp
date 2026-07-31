package legalqueryartifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// JSONLimits は、一 JSON object の構造検査上限を保持する。
type JSONLimits struct {
	Depth      int
	Values     int
	RejectNull bool
}

// InspectJSONObject は、閉じた JSON object の構造と資源上限を検査する。
func InspectJSONObject(data []byte, limits JSONLimits) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("JSON 成果物は有効な UTF-8 でなければなりません")
	}
	if limits.Depth <= 0 {
		return fmt.Errorf("JSON depth 上限が不正です")
	}
	if limits.Values <= 0 {
		return fmt.Errorf("JSON value 上限が不正です")
	}
	scanner := jsonInspector{
		data:   data,
		limits: limits,
	}
	scanner.skipWhitespace()
	if !scanner.hasByte() || scanner.currentByte() != '{' {
		return fmt.Errorf("JSON 成果物の最上位は一つの object でなければなりません")
	}
	kind, err := scanner.parseValue(1)
	if err != nil {
		return err
	}
	if kind != jsonValueObject {
		return fmt.Errorf("JSON 成果物の最上位は一つの object でなければなりません")
	}
	scanner.skipWhitespace()
	if scanner.hasByte() {
		return fmt.Errorf("JSON 成果物の root 後には空白以外を置けません")
	}
	return nil
}

// DecodeClosed は、未知 field と二個目の値を拒否して typed value へ decode する。
func DecodeClosed(data []byte, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("JSON 成果物を読み込めません: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("JSON 成果物の後に別の値があります")
		}
		return fmt.Errorf("JSON 成果物の終端が不正です: %w", err)
	}
	return nil
}

type jsonValueKind uint8

const (
	jsonValueObject jsonValueKind = iota + 1
	jsonValueArray
	jsonValueString
	jsonValueNumber
	jsonValueBoolean
	jsonValueNull
)

type jsonInspector struct {
	data       []byte
	offset     int
	valueCount int
	limits     JSONLimits
}

func (i *jsonInspector) parseValue(depth int) (jsonValueKind, error) {
	i.skipWhitespace()
	if err := i.countValue(); err != nil {
		return 0, err
	}
	if !i.hasByte() {
		return 0, invalidJSONStructure()
	}
	switch i.currentByte() {
	case '{':
		return i.parseObject(depth)
	case '[':
		return i.parseArray(depth)
	case '"':
		if _, err := i.parseString(); err != nil {
			return 0, err
		}
		return jsonValueString, nil
	case 't':
		return i.parseLiteral("true", jsonValueBoolean)
	case 'f':
		return i.parseLiteral("false", jsonValueBoolean)
	case 'n':
		if i.limits.RejectNull {
			return 0, fmt.Errorf("JSON 成果物に null は使用できません")
		}
		return i.parseLiteral("null", jsonValueNull)
	default:
		if _, err := i.parseNumber(); err != nil {
			return 0, err
		}
		return jsonValueNumber, nil
	}
}

func (i *jsonInspector) parseObject(depth int) (jsonValueKind, error) {
	if depth > i.limits.Depth {
		return 0, fmt.Errorf("JSON の入れ子が上限を超えています")
	}
	i.offset++
	i.skipWhitespace()
	if i.consumeByte('}') {
		return jsonValueObject, nil
	}
	keys := make(map[string]struct{})
	for {
		_, err := i.parseObjectKey(keys)
		if err != nil {
			return 0, err
		}
		if !i.consumeSeparatedByte(':') {
			return 0, invalidJSONStructure()
		}
		if _, err := i.parseValue(depth + 1); err != nil {
			return 0, fmt.Errorf("JSON object の値が不正です: %w", err)
		}
		closed, err := i.consumeCollectionSeparator('}')
		if err != nil {
			return 0, err
		}
		if closed {
			return jsonValueObject, nil
		}
	}
}

func (i *jsonInspector) parseArray(depth int) (jsonValueKind, error) {
	if depth > i.limits.Depth {
		return 0, fmt.Errorf("JSON の入れ子が上限を超えています")
	}
	i.offset++
	i.skipWhitespace()
	if i.consumeByte(']') {
		return jsonValueArray, nil
	}
	for {
		if _, err := i.parseValue(depth + 1); err != nil {
			return 0, err
		}
		closed, err := i.consumeCollectionSeparator(']')
		if err != nil {
			return 0, err
		}
		if closed {
			return jsonValueArray, nil
		}
	}
}

func (i *jsonInspector) parseObjectKey(keys map[string]struct{}) (string, error) {
	i.skipWhitespace()
	raw, err := i.parseString()
	if err != nil {
		return "", err
	}
	key, err := decodeJSONString(raw)
	if err != nil {
		return "", err
	}
	if _, exists := keys[key]; exists {
		return "", fmt.Errorf("JSON object の key を重複させることはできません")
	}
	keys[key] = struct{}{}
	return key, nil
}

func (i *jsonInspector) parseString() ([]byte, error) {
	start := i.offset
	if !i.consumeByte('"') {
		return nil, invalidJSONStructure()
	}
	escaped := false
	for i.hasByte() {
		current := i.currentByte()
		i.offset++
		if escaped {
			escaped = false
			continue
		}
		switch current {
		case '\\':
			escaped = true
		case '"':
			return i.data[start:i.offset], nil
		case '\n', '\r', '\t':
			return nil, invalidJSONStructure()
		}
	}
	return nil, invalidJSONStructure()
}

func (i *jsonInspector) parseLiteral(literal string, kind jsonValueKind) (jsonValueKind, error) {
	if len(i.data)-i.offset < len(literal) || string(i.data[i.offset:i.offset+len(literal)]) != literal {
		return 0, invalidJSONStructure()
	}
	i.offset += len(literal)
	return kind, nil
}

func (i *jsonInspector) parseNumber() ([]byte, error) {
	start := i.offset
	if i.currentByte() == '-' {
		i.offset++
	}
	if !i.hasByte() {
		return nil, invalidJSONStructure()
	}
	if i.currentByte() == '0' {
		i.offset++
	} else {
		if !isDigit19(i.currentByte()) {
			return nil, invalidJSONStructure()
		}
		for i.hasByte() && isDigit(i.currentByte()) {
			i.offset++
		}
	}
	if i.hasByte() && i.currentByte() == '.' {
		i.offset++
		if !i.hasByte() || !isDigit(i.currentByte()) {
			return nil, invalidJSONStructure()
		}
		for i.hasByte() && isDigit(i.currentByte()) {
			i.offset++
		}
	}
	if i.hasByte() && (i.currentByte() == 'e' || i.currentByte() == 'E') {
		i.offset++
		if i.hasByte() && (i.currentByte() == '+' || i.currentByte() == '-') {
			i.offset++
		}
		if !i.hasByte() || !isDigit(i.currentByte()) {
			return nil, invalidJSONStructure()
		}
		for i.hasByte() && isDigit(i.currentByte()) {
			i.offset++
		}
	}
	return i.data[start:i.offset], nil
}

func (i *jsonInspector) countValue() error {
	i.valueCount++
	if i.valueCount > i.limits.Values {
		return fmt.Errorf("JSON value 数が上限を超えています")
	}
	return nil
}

func (i *jsonInspector) consumeCollectionSeparator(closing byte) (bool, error) {
	i.skipWhitespace()
	if i.consumeByte(closing) {
		return true, nil
	}
	if !i.consumeByte(',') {
		return false, invalidJSONStructure()
	}
	return false, nil
}

func (i *jsonInspector) consumeSeparatedByte(want byte) bool {
	i.skipWhitespace()
	return i.consumeByte(want)
}

func (i *jsonInspector) skipWhitespace() {
	for i.hasByte() {
		switch i.currentByte() {
		case ' ', '\n', '\r', '\t':
			i.offset++
		default:
			return
		}
	}
}

func (i *jsonInspector) consumeByte(want byte) bool {
	if !i.hasByte() || i.currentByte() != want {
		return false
	}
	i.offset++
	return true
}

func (i *jsonInspector) hasByte() bool {
	return i.offset < len(i.data)
}

func (i *jsonInspector) currentByte() byte {
	return i.data[i.offset]
}

func decodeJSONString(raw []byte) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", invalidJSONStructure()
	}
	return value, nil
}

func invalidJSONStructure() error {
	return fmt.Errorf("JSON 構造が不正です")
}

func isDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isDigit19(value byte) bool {
	return value >= '1' && value <= '9'
}
