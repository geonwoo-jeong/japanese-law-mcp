package continuation

import (
	"bytes"
	"encoding/json"
	"unicode/utf8"

	"github.com/gowebpki/jcs"
)

// JSONValue は、RFC 8785 で canonicalize した不変な JSON 値を保持する。
type JSONValue struct {
	canonical []byte
}

// JSONObject は、RFC 8785 で canonicalize した不変な JSON object を保持する。
type JSONObject struct {
	value JSONValue
}

// NewJSONValue は、I-JSON として有効な一つの JSON 値を canonicalize する。
func NewJSONValue(raw []byte) (JSONValue, error) {
	canonical, err := canonicalizeJSON(raw)
	if err != nil {
		return JSONValue{}, errInvalidJSONValue
	}
	return JSONValue{canonical: canonical}, nil
}

// NewJSONObject は、I-JSON として有効な一つの JSON object を canonicalize する。
func NewJSONObject(raw []byte) (JSONObject, error) {
	value, err := NewJSONValue(raw)
	if err != nil || !value.isObject() {
		return JSONObject{}, errInvalidJSONObject
	}
	return JSONObject{value: value}, nil
}

// Bytes は、canonical JSON byte 列の複製を返す。
func (v JSONValue) Bytes() []byte {
	return bytes.Clone(v.canonical)
}

// String は、内容を公開しない表現を返す。
func (JSONValue) String() string {
	return "continuation.JSONValue"
}

// GoString は、内容を公開しない Go 構文表現を返す。
func (JSONValue) GoString() string {
	return "continuation.JSONValue"
}

// Bytes は、canonical JSON object byte 列の複製を返す。
func (o JSONObject) Bytes() []byte {
	return o.value.Bytes()
}

// String は、内容を公開しない表現を返す。
func (JSONObject) String() string {
	return "continuation.JSONObject"
}

// GoString は、内容を公開しない Go 構文表現を返す。
func (JSONObject) GoString() string {
	return "continuation.JSONObject"
}

func (v JSONValue) valid() bool {
	return len(v.canonical) != 0
}

func (v JSONValue) isObject() bool {
	return v.valid() && v.canonical[0] == '{'
}

func (o JSONObject) valid() bool {
	return o.value.isObject()
}

func canonicalizeJSON(raw []byte) ([]byte, error) {
	if !utf8.Valid(raw) || !json.Valid(raw) || !validUnicodeEscapes(raw) {
		return nil, errInvalidJSONValue
	}

	canonical, err := jcs.Transform(bytes.TrimSpace(raw))
	if err != nil || !utf8.Valid(canonical) {
		return nil, errInvalidJSONValue
	}
	return bytes.Clone(canonical), nil
}

func validUnicodeEscapes(raw []byte) bool {
	inString := false
	for index := 0; index < len(raw); index++ {
		switch raw[index] {
		case '"':
			inString = !inString
		case '\\':
			if !inString {
				continue
			}
			index++
			if index >= len(raw) {
				return false
			}
			if raw[index] != 'u' {
				continue
			}

			codeUnit, next, ok := readUTF16CodeUnit(raw, index+1)
			if !ok {
				return false
			}
			index = next - 1
			switch {
			case codeUnit >= 0xd800 && codeUnit <= 0xdbff:
				if index+6 >= len(raw) ||
					raw[index+1] != '\\' ||
					raw[index+2] != 'u' {
					return false
				}
				low, afterLow, valid := readUTF16CodeUnit(raw, index+3)
				if !valid || low < 0xdc00 || low > 0xdfff {
					return false
				}
				index = afterLow - 1
			case codeUnit >= 0xdc00 && codeUnit <= 0xdfff:
				return false
			}
		}
	}
	return true
}

func readUTF16CodeUnit(raw []byte, start int) (uint16, int, bool) {
	const hexadecimalDigits = 4
	if start < 0 || start+hexadecimalDigits > len(raw) {
		return 0, start, false
	}

	var value uint16
	for index := start; index < start+hexadecimalDigits; index++ {
		digit, ok := hexadecimalValue(raw[index])
		if !ok {
			return 0, start, false
		}
		value = value*16 + uint16(digit)
	}
	return value, start + hexadecimalDigits, true
}

func hexadecimalValue(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}
