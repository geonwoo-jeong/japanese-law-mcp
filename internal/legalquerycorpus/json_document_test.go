package legalquerycorpus

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestJSONDocument検査はrootHeaderと空白を受理する(t *testing.T) {
	t.Parallel()

	data := []byte(`
		{
			"schemaVersion" : 1,
			"metadata" : {"enabled": true},
			"artifactKind" : "semantic_case"
		}
	`)
	header, err := inspectJSONDocument(data)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 有効な JSON document の検査 error = %v", err)
	}
	if header.artifactKind != ArtifactKindSemanticCase ||
		header.schemaVersion != 1 {
		t.Fatalf("SOT-ENG-026: document header = %#v", header)
	}
}

func TestJSONDocument検査は未知のheader値を後段へ渡す(t *testing.T) {
	t.Parallel()

	header, err := inspectJSONDocument([]byte(
		`{"artifactKind":"future_kind","schemaVersion":999}`,
	))
	if err != nil {
		t.Fatalf("SOT-ENG-026: 構造上有効な未知 header を拒否した: %v", err)
	}
	if header.artifactKind != ArtifactKind("future_kind") ||
		header.schemaVersion != 999 {
		t.Fatalf("SOT-ENG-026: 未知 header = %#v", header)
	}
}

func TestJSONDocument検査は有効でないUTF8を拒否する(t *testing.T) {
	t.Parallel()

	invalidString := append(
		[]byte(`{"artifactKind":"semantic_case","schemaVersion":1,"value":"`),
		0xff,
	)
	invalidString = append(invalidString, []byte(`"}`)...)
	validPrefix := []byte(
		`{"artifactKind":"semantic_case","schemaVersion":1}`,
	)
	invalidTrailing := append(append([]byte{}, validPrefix...), 0xff)

	for name, data := range map[string][]byte{
		"文字列": invalidString,
		"末尾":  invalidTrailing,
	} {
		name, data := name, data
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := inspectJSONDocument(data); err == nil {
				t.Fatal("SOT-ENG-026: 有効でない UTF-8 を受理した")
			}
		})
	}
}

func TestJSONDocument検査は不正なUnicodeEscapeを拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"high surrogate単独": `"\uD800"`,
		"low surrogate単独":  `"\uDC00"`,
		"surrogate組不一致":    `"\uD800\u0041"`,
		"hex不足":            `"\u123"`,
	}
	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data := []byte(
				`{"artifactKind":"semantic_case","schemaVersion":1,"value":` +
					value + `}`,
			)
			if _, err := inspectJSONDocument(data); err == nil {
				t.Fatal("SOT-ENG-026: 不正な Unicode escape を受理した")
			}
		})
	}
}

func TestJSONDocument検査は正しいsurrogate組を受理する(t *testing.T) {
	t.Parallel()

	data := []byte(
		`{"artifactKind":"semantic_case","schemaVersion":1,"value":"\uD83D\uDE00"}`,
	)
	if _, err := inspectJSONDocument(data); err != nil {
		t.Fatalf("SOT-ENG-026: 正しい surrogate 組を拒否した: %v", err)
	}
}

func TestJSONDocument検査はroot以外と不完全な入力を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string][]byte{
		"array":      []byte(`[{"artifactKind":"semantic_case","schemaVersion":1}]`),
		"string":     []byte(`"value"`),
		"number":     []byte(`1`),
		"boolean":    []byte(`true`),
		"null":       []byte(`null`),
		"empty":      nil,
		"whitespace": []byte(" \n\t "),
		"object開始":   []byte(`{`),
		"value途中":    []byte(`{"artifactKind":`),
	}
	for name, data := range tests {
		name, data := name, data
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := inspectJSONDocument(data); err == nil {
				t.Fatal("SOT-ENG-026: root object ではないか不完全な入力を受理した")
			}
		})
	}
}

func TestJSONDocument検査は全階層の重複keyを拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string][]byte{
		"root": []byte(
			`{"artifactKind":"semantic_case","schemaVersion":1,"name":1,"name":2}`,
		),
		"nested object": []byte(
			`{"artifactKind":"semantic_case","schemaVersion":1,"value":{"name":1,"name":2}}`,
		),
		"array内object": []byte(
			`{"artifactKind":"semantic_case","schemaVersion":1,"value":[{"name":1,"name":2}]}`,
		),
		"escape同値": []byte(
			`{"artifactKind":"semantic_case","schemaVersion":1,"name":1,"\u006eame":2}`,
		),
	}
	for name, data := range tests {
		name, data := name, data
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := inspectJSONDocument(data); err == nil {
				t.Fatal("SOT-ENG-026: 同一 object 内の重複 key を受理した")
			}
		})
	}
}

func TestJSONDocument検査は別objectの同じkeyを受理する(t *testing.T) {
	t.Parallel()

	data := []byte(
		`{"artifactKind":"semantic_case","schemaVersion":1,` +
			`"left":{"same":1},"right":{"same":2},` +
			`"items":[{"same":3},{"same":4}]}`,
	)
	if _, err := inspectJSONDocument(data); err != nil {
		t.Fatalf("SOT-ENG-026: 別 object の同じ key を拒否した: %v", err)
	}
}

func TestJSONDocument検査はtrailingValueを拒否する(t *testing.T) {
	t.Parallel()

	valid := []byte(`{"artifactKind":"semantic_case","schemaVersion":1}`)
	for name, suffix := range map[string]string{
		"object": " \n {}",
		"scalar": " 0",
	} {
		name, suffix := name, suffix
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data := append(append([]byte{}, valid...), []byte(suffix)...)
			if _, err := inspectJSONDocument(data); err == nil {
				t.Fatal("SOT-ENG-026: root 後の trailing JSON value を受理した")
			}
		})
	}
}

func TestJSONDocument検査はJSON文法違反を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"object末尾comma": `{"value":0,}`,
		"array末尾comma":  `{"value":[0,]}`,
		"不正escape":      `{"value":"\x20"}`,
		"先頭零":           `{"value":01}`,
		"指数符号重複":        `{"value":1e+-2}`,
	}
	for name, fragment := range tests {
		name, fragment := name, fragment
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data := []byte(
				`{"artifactKind":"semantic_case","schemaVersion":1,"payload":` +
					fragment + `}`,
			)
			if _, err := inspectJSONDocument(data); err == nil {
				t.Fatal("SOT-ENG-026: JSON 文法違反を受理した")
			}
		})
	}
}

func TestJSONDocument検査はrootHeaderの欠落と型違反を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string][]byte{
		"artifactKind欠落":  []byte(`{"schemaVersion":1}`),
		"schemaVersion欠落": []byte(`{"artifactKind":"semantic_case"}`),
		"headerがnestedだけ": []byte(
			`{"metadata":{"artifactKind":"semantic_case","schemaVersion":1}}`,
		),
	}
	for _, invalid := range []string{"null", "true", "1", "[]", "{}"} {
		tests["artifactKind型 "+invalid] = []byte(fmt.Sprintf(
			`{"artifactKind":%s,"schemaVersion":1}`,
			invalid,
		))
	}
	for _, invalid := range []string{`"1"`, "null", "true", "[]", "{}"} {
		tests["schemaVersion型 "+invalid] = []byte(fmt.Sprintf(
			`{"artifactKind":"semantic_case","schemaVersion":%s}`,
			invalid,
		))
	}

	for name, data := range tests {
		name, data := name, data
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := inspectJSONDocument(data); err == nil {
				t.Fatal("SOT-ENG-026: root header の欠落または型違反を受理した")
			}
		})
	}
}

func TestJSONDocument検査はschemaVersionの非正規整数を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"fraction": "1.5",
		"exponent": "1e0",
		"overflow": "9223372036854775808",
	}
	for name, version := range tests {
		name, version := name, version
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data := []byte(fmt.Sprintf(
				`{"artifactKind":"semantic_case","schemaVersion":%s}`,
				version,
			))
			if _, err := inspectJSONDocument(data); err == nil {
				t.Fatal("SOT-ENG-026: schemaVersion の非正規整数を受理した")
			}
		})
	}
}

func TestJSONDocument検査はdepth上限をroot基準で適用する(t *testing.T) {
	t.Parallel()

	// root object を depth 1 とし、子 container ごとに一段増える。
	if _, err := inspectJSONDocument(jsonDocumentAtDepthForTest(16)); err != nil {
		t.Fatalf("SOT-ENG-026: depth 16 を拒否した: %v", err)
	}
	if _, err := inspectJSONDocument(jsonDocumentAtDepthForTest(17)); err == nil {
		t.Fatal("SOT-ENG-026: depth 17 を受理した")
	}
}

func TestJSONDocument検査はvalue数上限をcontainer込みで適用する(t *testing.T) {
	t.Parallel()

	// root object、header の二 scalar、payload array と、その scalar 要素を
	// それぞれ一 JSON value として数える。object key 自体は数えない。
	if _, err := inspectJSONDocument(
		jsonDocumentWithValueCountForTest(100000),
	); err != nil {
		t.Fatalf("SOT-ENG-026: 100000 JSON value を拒否した: %v", err)
	}
	if _, err := inspectJSONDocument(
		jsonDocumentWithValueCountForTest(100001),
	); err == nil {
		t.Fatal("SOT-ENG-026: 100001 JSON value を受理した")
	}
}

func TestJSONDocument検査errorは入力中の秘密値を含まない(t *testing.T) {
	t.Parallel()

	secret := "secret-value-that-must-not-be-logged"
	data := []byte(
		`{"artifactKind":"semantic_case","schemaVersion":1,` +
			`"token":"` + secret + `","token":"other"}`,
	)
	_, err := inspectJSONDocument(data)
	if err == nil {
		t.Fatal("SOT-ENG-026: 秘密値を含む重複 key 入力を受理した")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("SOT-ENG-026: error が入力中の秘密値を含む: %v", err)
	}
}

func FuzzJSONDocument検査は任意byteでpanicしない(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte(`{"artifactKind":"semantic_case","schemaVersion":1}`))
	f.Add([]byte(`{"artifactKind":"semantic_case","schemaVersion":1`))
	f.Add([]byte{0xff, 0xfe, 0xfd})
	f.Add(jsonDocumentAtDepthForTest(17))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := inspectJSONDocument(data)
		if err == nil && !json.Valid(data) {
			t.Fatal("SOT-ENG-026: 標準 JSON 文法で無効な document を受理した")
		}
	})
}

func jsonDocumentAtDepthForTest(depth int) []byte {
	if depth < 1 {
		panic("テスト用 JSON の depth は一以上でなければなりません")
	}
	var builder strings.Builder
	builder.WriteString(
		`{"artifactKind":"semantic_case","schemaVersion":1,"payload":`,
	)
	for current := 1; current < depth; current++ {
		builder.WriteByte('[')
	}
	builder.WriteByte('0')
	for current := 1; current < depth; current++ {
		builder.WriteByte(']')
	}
	builder.WriteByte('}')
	return []byte(builder.String())
}

func jsonDocumentWithValueCountForTest(total int) []byte {
	const fixedValueCount = 4
	if total < fixedValueCount {
		panic("テスト用 JSON の value 数が header 最小値を下回っています")
	}
	scalarCount := total - fixedValueCount
	var builder strings.Builder
	builder.Grow(scalarCount*2 + 80)
	builder.WriteString(
		`{"artifactKind":"semantic_case","schemaVersion":1,"payload":[`,
	)
	for index := 0; index < scalarCount; index++ {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteByte('0')
	}
	builder.WriteString(`]}`)
	return []byte(builder.String())
}
