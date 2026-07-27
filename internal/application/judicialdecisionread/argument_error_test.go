package judicialdecisionread

import (
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestNewArgumentErrorKeepsSafeClassification(t *testing.T) {
	t.Parallel()
	got, err := NewArgumentError("ref", "courts-hanrei-html では使用できません")
	if err != nil {
		t.Fatal(err)
	}
	var argumentError ArgumentError
	if !errors.As(got, &argumentError) {
		t.Fatalf("ArgumentError ではない: %T", got)
	}
	if argumentError.Code() != model.ErrorCodeInvalidArgument ||
		argumentError.Field() != "ref" ||
		argumentError.Reason() != "courts-hanrei-html では使用できません" {
		t.Fatalf("ArgumentError = %#v", argumentError)
	}
}

func TestNewArgumentErrorRejectsUnsafeValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		field  string
		reason string
	}{
		{"query", "理由"},
		{"ref", ""},
		{"ref", " 前後空白 "},
		{"ref", "bad\nreason"},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.field+"/"+testCase.reason, func(t *testing.T) {
			t.Parallel()
			if _, err := NewArgumentError(testCase.field, testCase.reason); err == nil {
				t.Fatal("不正な入力を受理した")
			}
		})
	}
}
