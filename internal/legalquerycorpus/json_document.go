package legalquerycorpus

import (
	"fmt"
	"unicode/utf8"
)

const (
	maxJSONDocumentDepth  = 16
	maxJSONDocumentValues = 100000
)

type jsonDocumentHeader struct {
	artifactKind  ArtifactKind
	schemaVersion int
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

type scannedJSONValue struct {
	kind jsonValueKind
	raw  []byte
}

type jsonDocumentScanner struct {
	data               []byte
	offset             int
	valueCount         int
	header             jsonDocumentHeader
	artifactKindFound  bool
	schemaVersionFound bool
}

// inspectJSONDocument は、JSON 文書の安全境界を検査して root header だけを返す。
func inspectJSONDocument(data []byte) (jsonDocumentHeader, error) {
	if !utf8.Valid(data) {
		return jsonDocumentHeader{}, fmt.Errorf(
			"JSON 成果物は有効な UTF-8 でなければなりません",
		)
	}
	scanner := jsonDocumentScanner{data: data}
	scanner.skipWhitespace()
	if !scanner.hasByte() || scanner.currentByte() != '{' {
		return jsonDocumentHeader{}, fmt.Errorf(
			"JSON 成果物の最上位は一つの object でなければなりません",
		)
	}
	value, err := scanner.parseValue(1, true)
	if err != nil {
		return jsonDocumentHeader{}, err
	}
	if value.kind != jsonValueObject {
		return jsonDocumentHeader{}, fmt.Errorf(
			"JSON 成果物の最上位は一つの object でなければなりません",
		)
	}
	scanner.skipWhitespace()
	if scanner.hasByte() {
		return jsonDocumentHeader{}, fmt.Errorf(
			"JSON 成果物の root 後には空白以外を置けません",
		)
	}
	return scanner.validatedHeader()
}

func (s *jsonDocumentScanner) parseValue(
	depth int,
	captureRootHeader bool,
) (scannedJSONValue, error) {
	s.skipWhitespace()
	if err := s.countValue(); err != nil {
		return scannedJSONValue{}, err
	}
	if !s.hasByte() {
		return scannedJSONValue{}, invalidJSONStructure()
	}
	switch s.currentByte() {
	case '{':
		return s.parseObject(depth, captureRootHeader)
	case '[':
		return s.parseArray(depth)
	case '"':
		raw, err := s.parseString()
		return scannedJSONValue{kind: jsonValueString, raw: raw}, err
	case 't':
		return s.parseLiteral("true", jsonValueBoolean)
	case 'f':
		return s.parseLiteral("false", jsonValueBoolean)
	case 'n':
		return s.parseLiteral("null", jsonValueNull)
	default:
		raw, err := s.parseNumber()
		return scannedJSONValue{kind: jsonValueNumber, raw: raw}, err
	}
}

func (s *jsonDocumentScanner) parseObject(
	depth int,
	captureRootHeader bool,
) (scannedJSONValue, error) {
	if err := validateJSONDepth(depth); err != nil {
		return scannedJSONValue{}, err
	}
	s.offset++
	keys := make(map[string]struct{})
	s.skipWhitespace()
	if s.consumeByte('}') {
		return scannedJSONValue{kind: jsonValueObject}, nil
	}
	for {
		key, err := s.parseObjectKey(keys)
		if err != nil {
			return scannedJSONValue{}, err
		}
		if !s.consumeSeparatedByte(':') {
			return scannedJSONValue{}, invalidJSONStructure()
		}
		value, err := s.parseValue(depth+1, false)
		if err != nil {
			return scannedJSONValue{}, err
		}
		if captureRootHeader {
			if err := s.captureHeader(key, value); err != nil {
				return scannedJSONValue{}, err
			}
		}
		if closed, err := s.consumeCollectionSeparator('}'); err != nil {
			return scannedJSONValue{}, err
		} else if closed {
			return scannedJSONValue{kind: jsonValueObject}, nil
		}
	}
}

func (s *jsonDocumentScanner) parseArray(
	depth int,
) (scannedJSONValue, error) {
	if err := validateJSONDepth(depth); err != nil {
		return scannedJSONValue{}, err
	}
	s.offset++
	s.skipWhitespace()
	if s.consumeByte(']') {
		return scannedJSONValue{kind: jsonValueArray}, nil
	}
	for {
		if _, err := s.parseValue(depth+1, false); err != nil {
			return scannedJSONValue{}, err
		}
		if closed, err := s.consumeCollectionSeparator(']'); err != nil {
			return scannedJSONValue{}, err
		} else if closed {
			return scannedJSONValue{kind: jsonValueArray}, nil
		}
	}
}

func (s *jsonDocumentScanner) parseObjectKey(
	keys map[string]struct{},
) (string, error) {
	s.skipWhitespace()
	raw, err := s.parseString()
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

func (s *jsonDocumentScanner) consumeCollectionSeparator(
	closing byte,
) (bool, error) {
	s.skipWhitespace()
	if s.consumeByte(closing) {
		return true, nil
	}
	if !s.consumeByte(',') {
		return false, invalidJSONStructure()
	}
	return false, nil
}

func (s *jsonDocumentScanner) countValue() error {
	s.valueCount++
	if s.valueCount > maxJSONDocumentValues {
		return fmt.Errorf(
			"一つの JSON 成果物は %d value 以下でなければなりません",
			maxJSONDocumentValues,
		)
	}
	return nil
}

func validateJSONDepth(depth int) error {
	if depth > maxJSONDocumentDepth {
		return fmt.Errorf(
			"JSON 成果物の nesting depth は %d 以下でなければなりません",
			maxJSONDocumentDepth,
		)
	}
	return nil
}

func invalidJSONStructure() error {
	return fmt.Errorf("JSON 成果物の構造が有効ではありません")
}
