package json

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	segmentjson "github.com/segmentio/encoding/json"
)

// candidate21OldUnmarshal は、候補の関数を呼ばない固定v1.6.1の独立な比較経路。
func candidate21OldUnmarshal(data []byte, value any) error {
	decoder := segmentjson.NewDecoder(bytes.NewReader(data))
	decoder.DontMatchCaseInsensitiveStructFields()
	return decoder.Decode(value)
}

type candidate21Fields struct {
	Field  string             `json:"field"`
	Number int                `json:"number"`
	Quoted int                `json:"quoted,string"`
	Nested *candidate21Fields `json:"nested"`
	Raw    stdjson.RawMessage `json:"raw"`
	Values []any              `json:"values"`
	Custom candidate21Custom  `json:"custom"`
}

var candidate21CallbackError = errors.New("診断用の callback error")

type candidate21Custom struct {
	Calls    int
	Observed []byte
	Retained []byte
	Mutate   bool
	Fail     bool
}

func (value *candidate21Custom) UnmarshalJSON(data []byte) error {
	value.Calls++
	value.Observed = bytes.Clone(data)
	// 所有権の差を観測する診断専用の保持。公開APIの容量を要求する意図はない。
	value.Retained = data
	if value.Mutate && len(data) != 0 {
		data[0] = '!'
	}
	if value.Fail {
		return candidate21CallbackError
	}
	return nil
}

type candidate21Input struct {
	name string
	data []byte
}

func TestCandidate21JSONEquivalentToOriginal(t *testing.T) {
	inputs := []candidate21Input{
		{"nil", nil}, {"空", []byte{}}, {"空白", []byte(" \t\r\n")},
		{"空object", []byte(`{}`)}, {"空array", []byte(`[]`)},
		{"null", []byte(`null`)}, {"true", []byte(`true`)}, {"false", []byte(`false`)},
		{"string", []byte(`"法令"`)}, {"前後空白", []byte(" \t\n{\"field\":\"法令\"}\r\n")},
		{"型errorの後方空白", []byte("[] \t\r\n")},
		{"数値の前後空白", []byte(" \t1.25e2\r\n")},
		{"overflowの前後空白", []byte(" \t1e309\r\n")},
		{"stringの前後空白", []byte(" \t\"法令\"\r\n")},
		{"JSON外の前方空白", []byte("\u00a0{}")},
		{"JSON外の後方空白", []byte("{}\u00a0")},
		{"一致field", []byte(`{"field":"法令","number":3}`)},
		{"case相違", []byte(`{"Field":"無視","NUMBER":3,"nested":{"Field":"無視"}}`)},
		{"NUL付きkey", []byte(`{"field\u0000":"無視","fi\u0000eld":"無視","field":"一致"}`)},
		{"未知field", []byte(`{"unknown":{"a":[1,2,3]},"field":"維持"}`)},
		{"重複field", []byte(`{"field":"前","field":"後","number":1,"number":2}`)},
		{"重複nested", []byte(`{"nested":{"field":"前"},"nested":{"number":2}}`)},
		{"nested", []byte(`{"nested":{"field":"法令","number":4},"values":[null,true,1.5,"条文"]}`)},
		{"raw", []byte(`{"raw": { "x" : [1,2,null] } }`)},
		{"stringtag", []byte(`{"quoted":"42"}`)},
		{"stringtag型相違", []byte(`{"field":"変更済み","quoted":42}`)},
		{"型相違で途中変更", []byte(`{"field":"変更済み","number":"不一致","nested":{"field":"後"}}`)},
		{"callback", []byte(`{"field":"先","custom":{"x":1},"number":2}`)},
		{"重複callback", []byte(`{"custom":{"x":1},"custom":{"x":2}}`)},
		{"構文不正で変更なし", []byte(`{"field":"変更不可","number":`)},
		{"切断object", []byte(`{"field":"法令"`)}, {"切断array", []byte(`[1,2`)},
		{"切断string", []byte(`"法令`)}, {"切断escape", []byte(`"\u12`)},
		{"不正escape", []byte(`"\q"`)}, {"不正unicode", []byte(`"\uQQQQ"`)},
		{"不正制御文字", []byte{'"', 1, '"'}},
		{"不正UTF8", []byte{'"', 0xff, '"'}},
		{"不正UTF8field", []byte{'{', '"', 'f', 'i', 'e', 'l', 'd', '"', ':', '"', 0xff, '"', '}'}},
		{"surrogate単独", []byte(`"\ud800"`)}, {"surrogate対", []byte(`"\ud83d\ude00"`)},
		{"surrogate不一致", []byte(`"\ud800\u0041"`)},
		{"BOM", append([]byte{0xef, 0xbb, 0xbf}, []byte(`{}`)...)},
		{"二値", []byte(`{"field":"先"} {"field":"後"}`)},
		{"後方不正token", []byte(`{"field":"先"}!`)},
		{"後方切断", []byte(`{"field":"先"}{`)},
		{"token連結", []byte(`truefalse`)},
		{"前方不正token", []byte(`!{}`)},
		{"深いarray", []byte(strings.Repeat("[", 64) + "0" + strings.Repeat("]", 64))},
		{"深い切断", []byte(strings.Repeat("[", 64) + "0")},
	}
	for _, number := range []string{"0", "-0", "1", "-1", "0.0", "1e2", "1e-300", "1e309", "9007199254740993", "18446744073709551616", "01", "1 2", "1e", "1e+", "1.", "-", "+1", "NaN", "Infinity"} {
		inputs = append(inputs, candidate21Input{"数値" + number, []byte(number)})
	}
	for _, size := range []int{32767, 32768, 32769} {
		inputs = append(inputs,
			candidate21Input{fmt.Sprintf("string境界%d", size), []byte(`"` + strings.Repeat("a", size-2) + `"`)},
			candidate21Input{fmt.Sprintf("object境界%d", size), []byte(`{"field":"` + strings.Repeat("a", size-len(`{"field":""}`)) + `"}`)},
			candidate21Input{fmt.Sprintf("数値chunk境界%d", size), []byte("1" + strings.Repeat("0", size-2) + "1")},
			candidate21Input{fmt.Sprintf("前空白境界%d", size), []byte(strings.Repeat(" ", size-4) + "null")},
		)
	}
	factories := []struct {
		name string
		new  func() any
	}{
		{"struct", func() any {
			return &candidate21Fields{Field: "既存", Number: 7, Raw: stdjson.RawMessage(`{"以前":true}`)}
		}},
		{"generic", func() any { var value any = map[string]any{"既存": float64(7)}; return &value }},
		{"map", func() any { value := map[string]any{"既存": "維持"}; return &value }},
		{"number", func() any { value := stdjson.Number("7"); return &value }},
		{"raw", func() any { value := stdjson.RawMessage(`{"既存":true}`); return &value }},
		{"custom", func() any { return &candidate21Custom{} }},
		{"custom書換え", func() any { return &candidate21Custom{Mutate: true} }},
		{"customerror", func() any { return &candidate21Custom{Fail: true} }},
		{"custom書換えerror", func() any { return &candidate21Custom{Mutate: true, Fail: true} }},
		{"nestedcustomerror", func() any {
			return &candidate21Fields{Field: "既存", Custom: candidate21Custom{Mutate: true, Fail: true}}
		}},
		{"nilTarget", func() any { return nil }},
		{"nilPointer", func() any { return (*candidate21Fields)(nil) }},
		{"非pointer", func() any { return candidate21Fields{Field: "既存"} }},
	}
	for _, input := range inputs {
		for _, factory := range factories {
			t.Run(input.name+"/"+factory.name, func(t *testing.T) {
				oldInput, newInput := bytes.Clone(input.data), bytes.Clone(input.data)
				oldValue, newValue := factory.new(), factory.new()
				oldErr := candidate21OldUnmarshal(oldInput, oldValue)
				newErr := Unmarshal(newInput, newValue)
				candidate21CompareErrors(t, oldErr, newErr)
				if !reflect.DeepEqual(oldValue, newValue) {
					t.Fatalf("値・部分変更・callback副作用が旧経路と一致しません: 型 %T", newValue)
				}
				if !bytes.Equal(oldInput, input.data) || !bytes.Equal(newInput, input.data) {
					t.Fatal("呼出し元の入力 byte が変更されました")
				}
			})
		}
	}
}

func candidate21CompareErrors(t *testing.T, oldErr, newErr error) {
	t.Helper()
	if (oldErr == nil) != (newErr == nil) {
		t.Fatalf("errorの有無が異なります: 旧=%v 新=%v", oldErr, newErr)
	}
	if oldErr == nil {
		return
	}
	if reflect.TypeOf(oldErr) != reflect.TypeOf(newErr) || oldErr.Error() != newErr.Error() || !reflect.DeepEqual(oldErr, newErr) {
		t.Fatalf("errorの型・本文・内容が異なります: 旧=%T %v 新=%T %v", oldErr, oldErr, newErr, newErr)
	}
	if errors.Is(oldErr, candidate21CallbackError) != errors.Is(newErr, candidate21CallbackError) {
		t.Fatal("callback error の対応が変わりました")
	}
}

func TestCandidate21CustomCallbackIsolation(t *testing.T) {
	for _, size := range []int{48, 32767, 32768, 32769} {
		for _, mutate := range []bool{false, true} {
			for _, fail := range []bool{false, true} {
				t.Run(fmt.Sprintf("%d/書換え%t/error%t", size, mutate, fail), func(t *testing.T) {
					// callbackへ渡るraw tokenからは前後空白が除かれる。
					original := []byte(" \t" + `{"x":"` + strings.Repeat("a", size-len(" \t"+`{"x":""}`+"\n")) + `"}` + "\n")
					oldInput, newInput := bytes.Clone(original), bytes.Clone(original)
					oldValue := &candidate21Custom{Mutate: mutate, Fail: fail}
					newValue := &candidate21Custom{Mutate: mutate, Fail: fail}
					candidate21CompareErrors(t, candidate21OldUnmarshal(oldInput, oldValue), Unmarshal(newInput, newValue))
					if oldValue.Calls != 1 || newValue.Calls != 1 || !reflect.DeepEqual(oldValue, newValue) {
						t.Fatal("callbackの呼出し回数・raw token・副作用が一致しません")
					}
					if !bytes.Equal(oldInput, original) || !bytes.Equal(newInput, original) {
						t.Fatal("custom callbackの書換えが呼出し元に漏れました")
					}
					oldSaved, newSaved := bytes.Clone(oldValue.Retained), bytes.Clone(newValue.Retained)
					for index := range original {
						oldInput[index], newInput[index] = 0, 0
					}
					if !bytes.Equal(oldValue.Retained, oldSaved) || !bytes.Equal(newValue.Retained, newSaved) {
						t.Fatal("呼出し元による後続書換えがcallback側の入力へ到達しました")
					}
				})
			}
		}
	}
}

func TestCandidate21OrdinaryValueCopyIsolation(t *testing.T) {
	for _, decode := range []struct {
		name string
		call func([]byte, any) error
	}{{"旧", candidate21OldUnmarshal}, {"候補", Unmarshal}} {
		t.Run(decode.name, func(t *testing.T) {
			input := []byte(`{"field":"保存","raw":{"値":1},"values":["保持",42]}`)
			var value candidate21Fields
			if err := decode.call(input, &value); err != nil {
				t.Fatal(err)
			}
			before, err := stdjson.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			for index := range input {
				input[index] = 0
			}
			after, err := stdjson.Marshal(value)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("復元したstring、RawMessage、generic値が原入力を参照しています")
			}
		})
	}
}
