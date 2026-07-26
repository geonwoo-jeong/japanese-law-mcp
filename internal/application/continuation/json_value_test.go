package continuation

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestJSONValueCanonicalizesRFC8785NumberVector(t *testing.T) {
	t.Parallel()

	input := []byte(
		`{"numbers":[333333333.33333329,1E30,4.50,2e-3,0.000000000000000000000000001],` +
			`"string":"\u20ac$\u000F\nA'B\"\\\"/","literals":[null,true,false]}`,
	)
	want := []byte(
		`{"literals":[null,true,false],"numbers":[333333333.3333333,1e+30,4.5,0.002,1e-27],` +
			`"string":"€$\u000f\nA'B\"\\\"/"}`,
	)

	got, err := NewJSONValue(input)
	if err != nil {
		t.Fatalf("SOT-IF-016: RFC 8785 の number vector の canonicalize エラー = %v", err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("SOT-IF-016: canonical JSON = %s、期待値 = %s", got.Bytes(), want)
	}
}

func TestJSONValueUsesRFC8785UTF16PropertyOrder(t *testing.T) {
	t.Parallel()

	input := []byte(
		`{"\u20ac":"Euro Sign","\r":"Carriage Return",` +
			`"\ufb33":"Hebrew Letter Dalet With Dagesh","1":"One",` +
			`"\ud83d\ude00":"Emoji: Grinning Face","\u0080":"Control",` +
			`"\u00f6":"Latin Small Letter O With Diaeresis"}`,
	)
	want := []byte(
		`{"\r":"Carriage Return","1":"One","` + "\u0080" + `":"Control",` +
			`"ö":"Latin Small Letter O With Diaeresis","€":"Euro Sign",` +
			`"😀":"Emoji: Grinning Face","דּ":"Hebrew Letter Dalet With Dagesh"}`,
	)

	got, err := NewJSONObject(input)
	if err != nil {
		t.Fatalf("SOT-IF-016: RFC 8785 の UTF-16 key 順序の canonicalize エラー = %v", err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("SOT-IF-016: canonical JSON = %s、期待値 = %s", got.Bytes(), want)
	}
}

func TestJSONValueAcceptsEveryJSONValueAndObjectRequiresObject(t *testing.T) {
	t.Parallel()

	values := map[string]struct {
		input string
		want  string
	}{
		"配列":   {input: `[3,2,1]`, want: `[3,2,1]`},
		"文字列":  {input: `"値"`, want: `"値"`},
		"整数":   {input: `1E0`, want: `1`},
		"真偽値":  {input: `true`, want: `true`},
		"null": {input: `null`, want: `null`},
	}
	for name, test := range values {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := NewJSONValue([]byte(test.input))
			if err != nil {
				t.Fatalf("SOT-IF-016: NewJSONValue() のエラー = %v", err)
			}
			if string(got.Bytes()) != test.want {
				t.Fatalf("SOT-IF-016: canonical JSON = %s、期待値 = %s", got.Bytes(), test.want)
			}
			if _, err := NewJSONObject([]byte(test.input)); err == nil {
				t.Fatalf("SOT-IF-016/SOT-IF-026: object 以外を JSONObject が受理した")
			}
		})
	}

	object, err := NewJSONObject([]byte(`{ "b":2, "a":1 }`))
	if err != nil {
		t.Fatalf("SOT-IF-016: NewJSONObject() のエラー = %v", err)
	}
	if string(object.Bytes()) != `{"a":1,"b":2}` {
		t.Fatalf("SOT-IF-016: JSONObject = %s", object.Bytes())
	}
}

func TestJSONValueRejectsDuplicateAndNonIJSONInput(t *testing.T) {
	t.Parallel()

	tests := map[string][]byte{
		"同名 key":           []byte(`{"a":1,"a":2}`),
		"escape 後に同名の key": []byte(`{"a":1,"\u0061":2}`),
		"上位 surrogate だけ":  []byte(`{"text":"\ud800"}`),
		"下位 surrogate だけ":  []byte(`{"text":"\udc00"}`),
		"有限でない数値":          []byte(`{"number":1e999}`),
		"不正な UTF-8":        {'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'},
		"末尾に別の値":           []byte(`{"a":1}{"b":2}`),
	}
	for name, input := range tests {
		input := bytes.Clone(input)
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewJSONValue(input); err == nil {
				t.Fatalf("SOT-IF-016: 非 I-JSON または重複 key を受理した: %q", input)
			}
			if _, err := NewJSONObject(input); err == nil {
				t.Fatalf("SOT-IF-016: 非 I-JSON または重複 key を object として受理した: %q", input)
			}
		})
	}
}

func TestJSONValuesAreImmutableAndFormattingDoesNotRevealContent(t *testing.T) {
	t.Parallel()

	input := []byte(`{"query":"漏らしてはいけない検索語","page":1}`)
	value, err := NewJSONValue(input)
	if err != nil {
		t.Fatalf("SOT-IF-016: NewJSONValue() のエラー = %v", err)
	}
	object, err := NewJSONObject(input)
	if err != nil {
		t.Fatalf("SOT-IF-016: NewJSONObject() のエラー = %v", err)
	}
	want := []byte(`{"page":1,"query":"漏らしてはいけない検索語"}`)

	for index := range input {
		input[index] = 'x'
	}
	first := value.Bytes()
	first[0] = '['
	second := object.Bytes()
	second[0] = '['
	if !bytes.Equal(value.Bytes(), want) || !bytes.Equal(object.Bytes(), want) {
		t.Fatalf("SOT-IF-016: JSON value が外部の変更を受けた")
	}

	for _, candidate := range []any{value, object} {
		for _, format := range []string{"%s", "%v", "%+v", "%#v"} {
			formatted := fmt.Sprintf(format, candidate)
			if strings.Contains(formatted, "漏らしてはいけない検索語") {
				t.Fatalf("SOT-IF-016: safe formatting が内容を公開した: %s", formatted)
			}
		}
	}
}
