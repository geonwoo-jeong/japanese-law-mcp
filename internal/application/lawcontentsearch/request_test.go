package lawcontentsearch_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequestConstants(t *testing.T) {
	t.Parallel()

	if lawcontentsearch.CapabilityID != "law.content.search" ||
		lawcontentsearch.MajorVersion != 1 ||
		lawcontentsearch.DefaultLimit != 20 ||
		lawcontentsearch.MaxLimit != 100 ||
		lawcontentsearch.MaxTokenBytes != 4096 {
		t.Fatal("SOT-IF-023: capability 定数が契約と一致しない")
	}
}

func TestNewRequestNormalizesAndCopiesInput(t *testing.T) {
	t.Parallel()

	allTerms := []string{"  行政  ", "手続"}
	anyTerms := []string{"申請"}
	excludeTerms := []string{"却下"}
	asOf := contentDate(t, "2025-01-02")
	limit := 50
	opaqueToken := strings.Repeat("t", 16)
	request, err := lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
		AllTerms:          allTerms,
		AnyTerms:          anyTerms,
		ExcludeTerms:      excludeTerms,
		AsOf:              &asOf,
		Limit:             &limit,
		ContinuationToken: opaqueToken,
	})
	if err != nil {
		t.Fatalf("SOT-IF-023: NewRequest() のエラー = %v", err)
	}

	allTerms[0] = "変更"
	anyTerms[0] = "変更"
	excludeTerms[0] = "変更"
	asOf = contentDate(t, "2030-01-02")
	limit = 1

	if got := request.AllTerms(); !sameStrings(got, []string{"行政", "手続"}) {
		t.Fatalf("SOT-IF-023: AllTerms() = %#v", got)
	}
	if got := request.AnyTerms(); !sameStrings(got, []string{"申請"}) {
		t.Fatalf("SOT-IF-023: AnyTerms() = %#v", got)
	}
	if got := request.ExcludeTerms(); !sameStrings(got, []string{"却下"}) {
		t.Fatalf("SOT-IF-023: ExcludeTerms() = %#v", got)
	}
	got := request.AllTerms()
	got[0] = "外部変更"
	if request.AllTerms()[0] != "行政" {
		t.Fatal("SOT-IF-015/023: getter から terms が変更された")
	}
	if date, ok := request.AsOf(); !ok || date.String() != "2025-01-02" {
		t.Fatalf("SOT-IF-023: AsOf() = %q, %t", date.String(), ok)
	}
	if request.Limit() != 50 {
		t.Fatalf("SOT-IF-023: Limit() = %d", request.Limit())
	}
	if token, ok := request.ContinuationToken(); !ok || token != opaqueToken {
		t.Fatalf("SOT-IF-016/023: ContinuationToken() = %q, %t", token, ok)
	}
}

func TestNewRequestNormalizesNilAndEmptyArrays(t *testing.T) {
	t.Parallel()

	request, err := lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
		AllTerms: []string{"民法"},
	})
	if err != nil {
		t.Fatalf("SOT-IF-023: NewRequest() のエラー = %v", err)
	}
	if request.AnyTerms() == nil || request.ExcludeTerms() == nil {
		t.Fatal("SOT-IF-023: 省略配列が non-nil 空配列へ正規化されていない")
	}
	if request.Limit() != lawcontentsearch.DefaultLimit {
		t.Fatalf("SOT-IF-023: 既定 limit = %d", request.Limit())
	}
	if _, ok := request.AsOf(); ok {
		t.Fatal("SOT-IF-023: 省略した asOf が存在する")
	}
	if _, ok := request.ContinuationToken(); ok {
		t.Fatal("SOT-IF-023: 省略した continuationToken が存在する")
	}
}

func TestNewRequestRejectsInvalidTerms(t *testing.T) {
	t.Parallel()

	invalidUTF8 := string([]byte{'a', 0xff})
	tests := map[string]lawcontentsearch.RequestValues{
		"正の条件なし": {
			ExcludeTerms: []string{"廃止"},
		},
		"trim 後に空": {
			AllTerms: []string{"   "},
		},
		"UTF-8 不正": {
			AllTerms: []string{invalidUTF8},
		},
		"128 byte 超": {
			AllTerms: []string{strings.Repeat("法", 43)},
		},
		"内部 U+0020": {
			AllTerms: []string{"行政 手続"},
		},
		"ASCII 制御文字": {
			AllTerms: []string{"行政\t手続"},
		},
		"DEL": {
			AllTerms: []string{"行政\x7f手続"},
		},
		"禁止記号": {
			AllTerms: []string{"行政*"},
		},
		"同じ配列内の重複": {
			AllTerms: []string{"行政", " 行政 "},
		},
		"配列間の重複": {
			AllTerms:     []string{"行政"},
			ExcludeTerms: []string{"行政"},
		},
		"一配列 8 件超": {
			AllTerms: numberedTerms("a", 9),
		},
		"合計 16 件超": {
			AllTerms:     numberedTerms("a", 8),
			AnyTerms:     numberedTerms("b", 8),
			ExcludeTerms: []string{"c0"},
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := lawcontentsearch.NewRequest(values); err == nil {
				t.Fatalf("SOT-IF-023: 不正な terms を受理した: %#v", values)
			}
		})
	}
}

func TestNewRequestAcceptsTermBoundariesAndLiteralWords(t *testing.T) {
	t.Parallel()

	allTerms := make([]string, 8)
	anyTerms := make([]string, 8)
	for index := range allTerms {
		allTerms[index] = strings.Repeat("a", 127) + string(rune('A'+index))
		anyTerms[index] = strings.Repeat("b", 127) + string(rune('A'+index))
	}
	request, err := lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
		AllTerms: allTerms,
		AnyTerms: anyTerms,
	})
	if err != nil {
		t.Fatalf("SOT-IF-023: 16 件・合計 2048 byte の境界値を拒否した: %v", err)
	}
	if len(request.AllTerms()) != 8 || len(request.AnyTerms()) != 8 {
		t.Fatal("SOT-IF-023: 境界値の語が失われた")
	}

	for _, term := range []string{"AND", "OR", "NOT", "\u3000行政\u3000"} {
		term := term
		t.Run(term, func(t *testing.T) {
			t.Parallel()

			if _, err := lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
				AllTerms: []string{term},
			}); err != nil {
				t.Fatalf("SOT-IF-023: 通常の検索語 %q を拒否した: %v", term, err)
			}
		})
	}
}

func TestNewRequestRejectsInvalidDateLimitAndToken(t *testing.T) {
	t.Parallel()

	zeroDate := model.Date{}
	for name, values := range map[string]lawcontentsearch.RequestValues{
		"asOf のゼロ値": {
			AllTerms: []string{"民法"},
			AsOf:     &zeroDate,
		},
		"limit 0": {
			AllTerms: []string{"民法"},
			Limit:    intPointer(0),
		},
		"limit 101": {
			AllTerms: []string{"民法"},
			Limit:    intPointer(101),
		},
		"token の UTF-8 不正": {
			AllTerms:          []string{"民法"},
			ContinuationToken: string([]byte{'v', 0xff}),
		},
		"token の 4096 byte 超": {
			AllTerms:          []string{"民法"},
			ContinuationToken: strings.Repeat("x", lawcontentsearch.MaxTokenBytes+1),
		},
	} {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := lawcontentsearch.NewRequest(values); err == nil {
				t.Fatalf("SOT-IF-016/023: 不正な入力を受理した: %#v", values)
			}
		})
	}
}

func TestRequestConditionObjectHasExactFiveKeysAndKeepsOrder(t *testing.T) {
	t.Parallel()

	request, err := lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
		AllTerms: []string{" 第二 ", "第一"},
		AnyTerms: []string{},
	})
	if err != nil {
		t.Fatalf("SOT-IF-023: NewRequest() のエラー = %v", err)
	}
	object, err := request.ConditionObject()
	if err != nil {
		t.Fatalf("SOT-IF-016/023: ConditionObject() のエラー = %v", err)
	}
	want := []byte(
		`{"allTerms":["第二","第一"],"anyTerms":[],"asOf":null,"excludeTerms":[],"limit":20}`,
	)
	if !bytes.Equal(object.Bytes(), want) {
		t.Fatalf("SOT-IF-023: condition = %s、期待値 = %s", object.Bytes(), want)
	}

	asOf := contentDate(t, "2025-01-02")
	limit := 40
	withDate, err := lawcontentsearch.NewRequest(lawcontentsearch.RequestValues{
		AllTerms:     []string{"行政"},
		AnyTerms:     []string{"手続"},
		ExcludeTerms: []string{"廃止"},
		AsOf:         &asOf,
		Limit:        &limit,
	})
	if err != nil {
		t.Fatalf("SOT-IF-023: NewRequest() のエラー = %v", err)
	}
	object, err = withDate.ConditionObject()
	if err != nil {
		t.Fatalf("SOT-IF-016/023: ConditionObject() のエラー = %v", err)
	}
	want = []byte(
		`{"allTerms":["行政"],"anyTerms":["手続"],"asOf":"2025-01-02","excludeTerms":["廃止"],"limit":40}`,
	)
	if !bytes.Equal(object.Bytes(), want) {
		t.Fatalf("SOT-IF-023: condition = %s、期待値 = %s", object.Bytes(), want)
	}
}

func TestRequestRejectsDirectJSONDecodeAndInvalidZeroValue(t *testing.T) {
	t.Parallel()

	var request lawcontentsearch.Request
	if err := json.Unmarshal([]byte(`{"allTerms":["民法"]}`), &request); err == nil {
		t.Fatal("SOT-IF-023: Request を JSON から直接復元できた")
	}
	if err := request.Validate(); err == nil {
		t.Fatal("SOT-IF-023: Request のゼロ値を受理した")
	}
	if _, err := request.ConditionObject(); err == nil {
		t.Fatal("SOT-IF-023: ゼロ値から condition object を作成できた")
	}
}

func numberedTerms(prefix string, count int) []string {
	terms := make([]string, count)
	for index := range terms {
		terms[index] = prefix + string(rune('a'+index))
	}
	return terms
}

func contentDate(t *testing.T, value string) model.Date {
	t.Helper()

	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("Date を作成できない: %v", err)
	}
	return date
}

func intPointer(value int) *int {
	return &value
}

func sameStrings(left, right []string) bool {
	return slices.Equal(left, right)
}
