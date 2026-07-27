package legalquerycorpus

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func (s *jsonDocumentScanner) parseString() ([]byte, error) {
	if !s.hasByte() || s.currentByte() != '"' {
		return nil, invalidJSONStructure()
	}
	start := s.offset
	s.offset++
	for s.hasByte() {
		switch current := s.currentByte(); {
		case current == '"':
			s.offset++
			return s.data[start:s.offset], nil
		case current == '\\':
			if err := s.consumeStringEscape(); err != nil {
				return nil, err
			}
		case current < 0x20:
			return nil, invalidJSONStructure()
		default:
			s.offset++
		}
	}
	return nil, invalidJSONStructure()
}

func (s *jsonDocumentScanner) consumeStringEscape() error {
	if s.offset+1 >= len(s.data) {
		return invalidJSONStructure()
	}
	switch s.data[s.offset+1] {
	case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
		s.offset += 2
		return nil
	case 'u':
		return s.consumeUnicodeEscape()
	default:
		return invalidJSONStructure()
	}
}

func (s *jsonDocumentScanner) consumeUnicodeEscape() error {
	codePoint, valid := decodeJSONHexQuad(s.data, s.offset+2)
	if !valid {
		return invalidJSONStructure()
	}
	s.offset += 6
	switch {
	case codePoint >= 0xd800 && codePoint <= 0xdbff:
		return s.consumeLowSurrogate()
	case codePoint >= 0xdc00 && codePoint <= 0xdfff:
		return invalidJSONStructure()
	default:
		return nil
	}
}

func (s *jsonDocumentScanner) consumeLowSurrogate() error {
	if s.offset+6 > len(s.data) ||
		s.data[s.offset] != '\\' ||
		s.data[s.offset+1] != 'u' {
		return invalidJSONStructure()
	}
	codePoint, valid := decodeJSONHexQuad(s.data, s.offset+2)
	if !valid || codePoint < 0xdc00 || codePoint > 0xdfff {
		return invalidJSONStructure()
	}
	s.offset += 6
	return nil
}

func decodeJSONHexQuad(data []byte, start int) (uint16, bool) {
	if start+4 > len(data) {
		return 0, false
	}
	var value uint16
	for _, character := range data[start : start+4] {
		digit, valid := jsonHexDigit(character)
		if !valid {
			return 0, false
		}
		value = value*16 + uint16(digit)
	}
	return value, true
}

func jsonHexDigit(character byte) (byte, bool) {
	switch {
	case character >= '0' && character <= '9':
		return character - '0', true
	case character >= 'a' && character <= 'f':
		return character - 'a' + 10, true
	case character >= 'A' && character <= 'F':
		return character - 'A' + 10, true
	default:
		return 0, false
	}
}

func (s *jsonDocumentScanner) parseNumber() ([]byte, error) {
	start := s.offset
	s.consumeByte('-')
	if !s.hasByte() {
		return nil, invalidJSONStructure()
	}
	switch {
	case s.consumeByte('0'):
		if s.hasByte() && isJSONDigit(s.currentByte()) {
			return nil, invalidJSONStructure()
		}
	case s.currentByte() >= '1' && s.currentByte() <= '9':
		s.consumeJSONDigits()
	default:
		return nil, invalidJSONStructure()
	}
	if s.consumeByte('.') {
		if !s.consumeJSONDigits() {
			return nil, invalidJSONStructure()
		}
	}
	if s.consumeByte('e') || s.consumeByte('E') {
		if !s.consumeJSONExponent() {
			return nil, invalidJSONStructure()
		}
	}
	return s.data[start:s.offset], nil
}

func (s *jsonDocumentScanner) consumeJSONExponent() bool {
	if !s.consumeByte('+') {
		s.consumeByte('-')
	}
	return s.consumeJSONDigits()
}

func (s *jsonDocumentScanner) consumeJSONDigits() bool {
	start := s.offset
	for s.hasByte() && isJSONDigit(s.currentByte()) {
		s.offset++
	}
	return s.offset > start
}

func isJSONDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

func (s *jsonDocumentScanner) parseLiteral(
	literal string,
	kind jsonValueKind,
) (scannedJSONValue, error) {
	if len(s.data)-s.offset < len(literal) ||
		string(s.data[s.offset:s.offset+len(literal)]) != literal {
		return scannedJSONValue{}, invalidJSONStructure()
	}
	s.offset += len(literal)
	return scannedJSONValue{kind: kind}, nil
}

func (s *jsonDocumentScanner) captureHeader(
	key string,
	value scannedJSONValue,
) error {
	switch key {
	case "artifactKind":
		return s.captureArtifactKind(value)
	case "schemaVersion":
		return s.captureSchemaVersion(value)
	default:
		return nil
	}
}

func (s *jsonDocumentScanner) captureArtifactKind(
	value scannedJSONValue,
) error {
	if value.kind != jsonValueString {
		return fmt.Errorf("JSON 成果物の artifactKind は string でなければなりません")
	}
	artifactKind, err := decodeJSONString(value.raw)
	if err != nil {
		return err
	}
	s.header.artifactKind = ArtifactKind(artifactKind)
	s.artifactKindFound = true
	return nil
}

func (s *jsonDocumentScanner) captureSchemaVersion(
	value scannedJSONValue,
) error {
	if value.kind != jsonValueNumber {
		return fmt.Errorf("JSON 成果物の schemaVersion は整数でなければなりません")
	}
	version, err := strconv.Atoi(string(value.raw))
	if err != nil {
		return fmt.Errorf("JSON 成果物の schemaVersion は整数でなければなりません")
	}
	s.header.schemaVersion = version
	s.schemaVersionFound = true
	return nil
}

func (s *jsonDocumentScanner) validatedHeader() (jsonDocumentHeader, error) {
	if !s.artifactKindFound || !s.schemaVersionFound {
		return jsonDocumentHeader{}, fmt.Errorf(
			"JSON 成果物の root header に必須項目がありません",
		)
	}
	return s.header, nil
}

func decodeJSONString(raw []byte) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", invalidJSONStructure()
	}
	return value, nil
}

func (s *jsonDocumentScanner) skipWhitespace() {
	for s.hasByte() {
		switch s.currentByte() {
		case ' ', '\t', '\n', '\r':
			s.offset++
		default:
			return
		}
	}
}

func (s *jsonDocumentScanner) consumeSeparatedByte(expected byte) bool {
	s.skipWhitespace()
	return s.consumeByte(expected)
}

func (s *jsonDocumentScanner) consumeByte(expected byte) bool {
	if !s.hasByte() || s.currentByte() != expected {
		return false
	}
	s.offset++
	return true
}

func (s *jsonDocumentScanner) hasByte() bool {
	return s.offset < len(s.data)
}

func (s *jsonDocumentScanner) currentByte() byte {
	return s.data[s.offset]
}
