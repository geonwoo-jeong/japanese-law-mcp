package judicialdecisionsearch_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
)

func TestRequestConstants(t *testing.T) {
	t.Parallel()

	if judicialdecisionsearch.CapabilityID != "judicial-decision.search" ||
		judicialdecisionsearch.MajorVersion != 1 ||
		judicialdecisionsearch.DefaultLimit != 20 ||
		judicialdecisionsearch.MaxLimit != 30 ||
		judicialdecisionsearch.MaxTokenBytes != 4096 {
		t.Fatal("SOT-IF-041: capability 定数が契約と一致しない")
	}
}

func TestNewRequestNormalizesOnlyOuterUnicodeWhitespaceAndCopiesLimit(t *testing.T) {
	t.Parallel()

	limit := 12
	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{
			Query: "\u3000\t  民  法  \n\u00a0",
			Limit: &limit,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-041: NewRequest() のエラー = %v", err)
	}
	limit = 1

	if request.Query() != "民  法" {
		t.Fatalf("SOT-IF-041: Query() = %q", request.Query())
	}
	if request.Limit() != 12 {
		t.Fatalf("SOT-IF-041: Limit() = %d", request.Limit())
	}
	if token, exists := request.ContinuationToken(); exists || token != "" {
		t.Fatalf("SOT-IF-016/SOT-IF-041: ContinuationToken() = %q, %t", token, exists)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("SOT-IF-041: Validate() のエラー = %v", err)
	}
}

func TestNewRequestAppliesDefaultLimit(t *testing.T) {
	t.Parallel()

	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{Query: "民法"},
	)
	if err != nil {
		t.Fatalf("SOT-IF-041: NewRequest() のエラー = %v", err)
	}
	if request.Limit() != judicialdecisionsearch.DefaultLimit {
		t.Fatalf("SOT-IF-041: 既定 limit = %d", request.Limit())
	}
	if _, exists := request.ContinuationToken(); exists {
		t.Fatal("SOT-IF-041: 省略した continuationToken が存在すると判定された")
	}
}

func TestNewRequestAcceptsQueryBoundary(t *testing.T) {
	t.Parallel()

	query := strings.Repeat("法", 170) + "ab"
	limit := judicialdecisionsearch.MaxLimit
	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{
			Query: query,
			Limit: &limit,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-041: 境界値を拒否した: %v", err)
	}
	if len(request.Query()) != 512 {
		t.Fatalf("SOT-IF-041: query の byte 数 = %d", len(request.Query()))
	}
}

func TestNewRequestRejectsInvalidQuery(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"空文字":          "",
		"Unicode 空白だけ": "\u3000 \t\n",
		"内部タブ":         "民\t法",
		"内部改行":         "民\n法",
		"内部 NUL":       "民\x00法",
		"DEL":          "民\x7f法",
		"不正な UTF-8":    string([]byte{'a', 0xff}),
		"512 byte 超":   strings.Repeat("法", 171),
	}
	for name, query := range tests {
		name, query := name, query
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := judicialdecisionsearch.NewRequest(
				judicialdecisionsearch.RequestValues{Query: query},
			); err == nil {
				t.Fatalf("SOT-IF-041: 不正な query %q を受理した", query)
			}
		})
	}
}

func TestNewRequestRejectsInvalidLimit(t *testing.T) {
	t.Parallel()

	for _, value := range []int{-1, 0, judicialdecisionsearch.MaxLimit + 1} {
		limit := value
		t.Run("limit", func(t *testing.T) {
			t.Parallel()

			if _, err := judicialdecisionsearch.NewRequest(
				judicialdecisionsearch.RequestValues{
					Query: "民法",
					Limit: &limit,
				},
			); err == nil {
				t.Fatalf("SOT-IF-041: limit=%d を受理した", limit)
			}
		})
	}
}

func TestNewRequestRejectsInvalidContinuationToken(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"不正な UTF-8":   string([]byte{'v', '1', '.', 0xff}),
		"4096 byte 超": strings.Repeat("x", judicialdecisionsearch.MaxTokenBytes+1),
	}
	for name, token := range tests {
		name, token := name, token
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := judicialdecisionsearch.NewRequest(
				judicialdecisionsearch.RequestValues{
					Query:             "民法",
					ContinuationToken: token,
				},
			); err == nil {
				t.Fatal("SOT-IF-016/SOT-IF-041: 不正な continuationToken を受理した")
			}
		})
	}
}

func TestNewRequestAcceptsOpaqueContinuationToken(t *testing.T) {
	t.Parallel()

	token := string([]byte{
		'o', 'p', 'a', 'q', 'u', 'e', '-', 't', 'o', 'k', 'e', 'n',
	})
	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{
			Query:             "民法",
			ContinuationToken: token,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-016/SOT-IF-041: NewRequest() のエラー = %v", err)
	}
	got, exists := request.ContinuationToken()
	if !exists || got != token {
		t.Fatalf("SOT-IF-016/SOT-IF-041: ContinuationToken() = %q, %t", got, exists)
	}
}

func TestRequestConditionObjectUsesNormalizedConditionOnly(t *testing.T) {
	t.Parallel()

	limit := 7
	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{
			Query: "\u3000民  法\u3000",
			Limit: &limit,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-041: NewRequest() のエラー = %v", err)
	}
	object, err := request.ConditionObject()
	if err != nil {
		t.Fatalf("SOT-IF-016/SOT-IF-041: ConditionObject() のエラー = %v", err)
	}
	want := []byte(`{"limit":7,"query":"民  法"}`)
	if !bytes.Equal(object.Bytes(), want) {
		t.Fatalf("SOT-IF-016/SOT-IF-041: condition = %s、期待値 = %s", object.Bytes(), want)
	}

	got := object.Bytes()
	got[0] = '['
	if !bytes.Equal(object.Bytes(), want) {
		t.Fatal("SOT-IF-016: condition object が外部から変更された")
	}
}

func TestRequestRejectsDirectJSONDecodeAndInvalidZeroValue(t *testing.T) {
	t.Parallel()

	var request judicialdecisionsearch.Request
	if err := json.Unmarshal([]byte(`{"query":"民法"}`), &request); err == nil {
		t.Fatal("SOT-ENG-002/SOT-IF-041: Request を JSON から直接復元できた")
	}
	if err := request.Validate(); err == nil {
		t.Fatal("SOT-IF-041: Request のゼロ値を受理した")
	}
	if _, err := request.ConditionObject(); err == nil {
		t.Fatal("SOT-IF-041: ゼロ値から condition object を作成できた")
	}
}
