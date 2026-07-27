package legalquery_test

import (
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestArgumentErrorKeepsSafePublicDetails(t *testing.T) {
	t.Parallel()

	argumentError, err := legalquery.NewArgumentError(
		"query",
		"は有効な UTF-8 でなければなりません",
	)
	if err != nil {
		t.Fatalf("SOT-IF-027/SOT-IF-051: NewArgumentError() のエラー = %v", err)
	}
	if argumentError.Code() != model.ErrorCodeInvalidArgument ||
		argumentError.Field() != "query" ||
		argumentError.Reason() != "は有効な UTF-8 でなければなりません" {
		t.Fatalf("SOT-IF-027: ArgumentError = %#v", argumentError)
	}
	if argumentError.Error() != "query は有効な UTF-8 でなければなりません" {
		t.Fatalf("SOT-IF-027: Error() = %q", argumentError.Error())
	}
	if err := argumentError.Validate(); err != nil {
		t.Fatalf("SOT-IF-027: Validate() のエラー = %v", err)
	}
}

func TestNewArgumentErrorAcceptsOnlyDefinedSafeDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		reason string
	}{
		{name: "未知の field", field: "providerId", reason: "は使用できません"},
		{name: "空の reason", field: "query", reason: ""},
		{name: "外側の空白", field: "ref", reason: " は有効ではありません"},
		{name: "ASCII 制御文字", field: "limitPerAttempt", reason: "は\n範囲外です"},
		{
			name:   "reason の上限超過",
			field:  "query",
			reason: strings.Repeat("法", 86),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewArgumentError(test.field, test.reason); err == nil {
				t.Fatal("SOT-IF-027: 不正な ArgumentError を受理した")
			}
		})
	}
}

func TestArgumentErrorZeroValueIsInvalidAndSafe(t *testing.T) {
	t.Parallel()

	var argumentError legalquery.ArgumentError
	if err := argumentError.Validate(); err == nil {
		t.Fatal("SOT-IF-027: ArgumentError のゼロ値を受理した")
	}
	if argumentError.Error() != "統合法情報照会の入力値が契約を満たしていません" {
		t.Fatalf("SOT-IF-027: ゼロ値 Error() = %q", argumentError.Error())
	}
}
