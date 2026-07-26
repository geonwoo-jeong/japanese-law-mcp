package getlaw

import (
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestNewRequestNormalizesPublicInput(t *testing.T) {
	t.Parallel()

	asOf := mustRequestDate(t, "2026-07-26")
	request, err := NewRequest(RequestValues{
		LawID: "  令和法律第一号  ",
		AsOf:  &asOf,
	})
	if err != nil {
		t.Fatalf("NewRequest() のエラー = %v", err)
	}
	if request.LawID() != "令和法律第一号" {
		t.Fatalf("LawID() = %q", request.LawID())
	}
	gotAsOf, exists := request.AsOf()
	if !exists || gotAsOf.String() != "2026-07-26" {
		t.Fatalf("AsOf() = %q, %v", gotAsOf.String(), exists)
	}
}

func TestNewRequestRejectsInvalidPublicInput(t *testing.T) {
	t.Parallel()

	oldDate := mustRequestDate(t, "2017-03-31")
	tests := map[string]RequestValues{
		"空値":       {LawID: "   "},
		"byte 数超過": {LawID: strings.Repeat("a", 257)},
		"制御文字":     {LawID: "law\n1"},
		"収録開始日前":   {LawID: "law-1", AsOf: &oldDate},
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := NewRequest(values); err == nil {
				t.Fatal("NewRequest() が入力を受理しました")
			}
		})
	}
}

func TestRequestCannotBeRestoredDirectlyFromJSON(t *testing.T) {
	t.Parallel()

	var request Request
	if err := request.UnmarshalJSON([]byte(`{"lawId":"law-1"}`)); err == nil {
		t.Fatal("UnmarshalJSON() が直接復元を受理しました")
	}
}

func mustRequestDate(t *testing.T, value string) model.Date {
	t.Helper()

	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("日付を作成できません: %v", err)
	}
	return date
}
