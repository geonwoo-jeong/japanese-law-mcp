package judicialdecisionsearch

import (
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestNewArgumentErrorKeepsSafeClassification(t *testing.T) {
	t.Parallel()
	got, err := NewArgumentError(
		"continuationToken",
		"このプロバイダーでは使用できません",
	)
	if err != nil {
		t.Fatal(err)
	}
	var argumentError ArgumentError
	if !errors.As(error(got), &argumentError) {
		t.Fatalf("ArgumentError ではない: %T", got)
	}
	if argumentError.Code() != model.ErrorCodeInvalidArgument ||
		argumentError.Field() != "continuationToken" ||
		argumentError.Reason() != "このプロバイダーでは使用できません" {
		t.Fatalf("ArgumentError = %#v", argumentError)
	}
}

func TestNewArgumentErrorRejectsUnsafeValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		field  string
		reason string
	}{
		{"unknown", "使用できません"},
		{"query", ""},
		{"query", "改行\nは禁止です"},
	}
	for _, testCase := range cases {
		if _, err := NewArgumentError(testCase.field, testCase.reason); err == nil {
			t.Fatalf("不正な argument error を受理した: %#v", testCase)
		}
	}
}
