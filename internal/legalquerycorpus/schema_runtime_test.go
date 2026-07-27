package legalquerycorpus

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestCorpusSchemaRuntimeは固定V1Schemaをcompileする(t *testing.T) {
	t.Parallel()

	if _, err := newCorpusSchemaV1(schemaRuntimeTestFixedSchema(t)); err != nil {
		t.Fatalf("SOT-ENG-026: 固定 v1 schema を compile できない: %v", err)
	}
}

func TestCorpusSchemaRuntimeは安全でないJSONSchema文書を拒否する(t *testing.T) {
	t.Parallel()

	invalidUTF8 := append(
		[]byte(`{"$schema":"`+corpusDraft202012+`","title":"`),
		0xff,
	)
	invalidUTF8 = append(invalidUTF8, []byte(`"}`)...)
	valid := schemaRuntimeTestMinimalSchema(`"type":"object"`)

	tests := map[string][]byte{
		"不正UTF-8": invalidUTF8,
		"重複key": []byte(
			`{"$schema":"` + corpusDraft202012 + `",` +
				`"$defs":{"value":{"type":"string","type":"number"}}}`,
		),
		"trailing value": append(append([]byte{}, valid...), []byte(` {}`)...),
		"root非object":    []byte(`[]`),
		"depth上限超過":      schemaRuntimeTestTooDeepSchema(),
		"value上限超過":      schemaRuntimeTestTooManyValuesSchema(),
	}
	for name, data := range tests {
		name, data := name, data
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := newCorpusSchemaV1(data); err == nil {
				t.Fatal("SOT-ENG-026: 安全境界に違反する schema を受理した")
			}
		})
	}
}

func TestCorpusSchemaRuntimeはDraft202012以外を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string][]byte{
		"draft-07": []byte(
			`{"$schema":"http://json-schema.org/draft-07/schema#","type":"object"}`,
		),
		"$schema欠落": []byte(`{"type":"object"}`),
		"$schema型不一致": []byte(
			`{"$schema":202012,"type":"object"}`,
		),
	}
	for name, data := range tests {
		name, data := name, data
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := newCorpusSchemaV1(data); err == nil {
				t.Fatal("SOT-ENG-026: Draft 2020-12 ではない schema を受理した")
			}
		})
	}
}

func TestCorpusSchemaRuntimeは外部RefとDynamicRefを拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		keyword   string
		reference string
	}{
		{
			name:      "https ref",
			keyword:   "$ref",
			reference: "https://example.invalid/schema.json",
		},
		{
			name:      "relative ref",
			keyword:   "$ref",
			reference: "other-schema.json#/$defs/value",
		},
		{
			name:      "file ref",
			keyword:   "$ref",
			reference: "file:///tmp/schema.json",
		},
		{
			name:      "https dynamicRef",
			keyword:   "$dynamicRef",
			reference: "https://example.invalid/schema.json",
		},
		{
			name:      "relative dynamicRef",
			keyword:   "$dynamicRef",
			reference: "other-schema.json#value",
		},
		{
			name:      "file dynamicRef",
			keyword:   "$dynamicRef",
			reference: "file:///tmp/schema.json#value",
		},
		{
			name:      "local dynamicRef",
			keyword:   "$dynamicRef",
			reference: "#value",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			data := schemaRuntimeTestSchemaWithReference(
				test.keyword,
				test.reference,
			)
			if _, err := newCorpusSchemaV1(data); err == nil {
				t.Fatal("SOT-ENG-026: 禁止された schema reference を受理した")
			}
		})
	}
}

func TestCorpusSchemaRuntimeはLocalDefsRefを受理する(t *testing.T) {
	t.Parallel()

	data := schemaRuntimeTestSchemaWithReference(
		"$ref",
		"#/$defs/value",
	)
	if _, err := newCorpusSchemaV1(data); err != nil {
		t.Fatalf("SOT-ENG-026: local $defs reference を拒否した: %v", err)
	}
}

func TestCorpusSchemaRuntimeのzeroValueはdecodeを拒否する(t *testing.T) {
	t.Parallel()

	data := mustJSONBytes(t, validManifest())
	header, err := inspectJSONDocument(data)
	if err != nil {
		t.Fatalf("SOT-ENG-026: manifest header を検査できない: %v", err)
	}
	if _, err := (corpusSchemaV1{}).validateAndDecode(data, header); err == nil {
		t.Fatal("SOT-ENG-026: corpusSchemaV1 の zero value で decode した")
	}
}

func schemaRuntimeTestFixedSchema(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile(corpusSchemaV1Path(t))
	if err != nil {
		t.Fatalf("SOT-ENG-026: 固定 v1 schema を読めない: %v", err)
	}
	return data
}

func schemaRuntimeTestMinimalSchema(body string) []byte {
	return []byte(
		`{"$schema":"` + corpusDraft202012 + `",` + body + `}`,
	)
}

func schemaRuntimeTestSchemaWithReference(
	keyword string,
	reference string,
) []byte {
	return []byte(fmt.Sprintf(
		`{"$schema":%q,"$defs":{"value":{"$anchor":"value","type":"object"}},%q:%q}`,
		corpusDraft202012,
		keyword,
		reference,
	))
}

func schemaRuntimeTestTooDeepSchema() []byte {
	const nestedContainers = maxJSONDocumentDepth

	return []byte(
		`{"$schema":"` + corpusDraft202012 + `","x-depth":` +
			strings.Repeat("[", nestedContainers) +
			`true` +
			strings.Repeat("]", nestedContainers) +
			`}`,
	)
}

func schemaRuntimeTestTooManyValuesSchema() []byte {
	var builder strings.Builder
	builder.Grow(maxJSONDocumentValues*2 + 128)
	builder.WriteString(`{"$schema":"`)
	builder.WriteString(corpusDraft202012)
	builder.WriteString(`","x-values":[`)
	for index := 0; index <= maxJSONDocumentValues; index++ {
		if index > 0 {
			builder.WriteByte(',')
		}
		builder.WriteByte('0')
	}
	builder.WriteString(`]}`)
	return []byte(builder.String())
}
