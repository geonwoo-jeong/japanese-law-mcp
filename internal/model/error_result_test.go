package model_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestErrorResultMapsEverySourceErrorWithoutLosingCode(t *testing.T) {
	t.Parallel()

	for _, expectation := range sourceErrorExpectations() {
		expectation := expectation
		t.Run(string(expectation.code), func(t *testing.T) {
			t.Parallel()

			values := validSourceErrorValues(t, expectation.code)
			if expectation.code == model.SourceErrorCodeRateLimited ||
				expectation.code == model.SourceErrorCodeSourceUnavailable {
				values.RetryAfter = "120"
			}
			sourceError, err := model.NewSourceError(values)
			if err != nil {
				t.Fatalf("SourceError の作成エラー = %v", err)
			}
			result, err := model.NewErrorResultFromSourceError(sourceError)
			if err != nil {
				t.Fatalf("SOT-IF-027: SourceError の変換エラー = %v", err)
			}
			if string(result.Code()) != string(expectation.code) ||
				result.Retryable() != expectation.retryable ||
				result.Message() != sourceError.Message() {
				t.Fatalf("SOT-IF-027: ErrorResult = %#v", result)
			}
			details, exists := result.Details()
			if !exists {
				t.Fatal("SOT-IF-027: 情報源エラーの details がない")
			}
			if details["providerId"] != "e-gov-law-api-v2" ||
				details["sourceId"] != "e-gov-law-api-v2" ||
				details["capabilityId"] != "law.search" {
				t.Fatalf("SOT-IF-027: details = %#v", details)
			}
			_, hasOperation := details["operation"]
			wantOperation := expectation.code != model.SourceErrorCodeUnsupportedCapability &&
				expectation.code != model.SourceErrorCodeUnsupportedQuery &&
				expectation.code != model.SourceErrorCodeConfigurationRequired
			if hasOperation != wantOperation {
				t.Fatalf("SOT-IF-027: operation = %#v", details["operation"])
			}
			_, hasRetryAfter := details["retryAfter"]
			wantRetryAfter := expectation.code == model.SourceErrorCodeRateLimited ||
				expectation.code == model.SourceErrorCodeSourceUnavailable
			if hasRetryAfter != wantRetryAfter {
				t.Fatalf("SOT-IF-027: retryAfter = %#v", details["retryAfter"])
			}
		})
	}
}

func TestErrorResultProvidesSafeInputAndInternalErrors(t *testing.T) {
	t.Parallel()

	invalid, err := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeInvalidArgument,
		Details: map[string]any{
			"field":  "query",
			"reason": "一文字以上で指定してください",
		},
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-005: invalid_argument の作成エラー = %v", err)
	}
	if invalid.Retryable() ||
		!strings.Contains(invalid.Message(), "入力") {
		t.Fatalf("SOT-IF-027: invalid_argument = %#v", invalid)
	}
	details, exists := invalid.Details()
	if !exists {
		t.Fatal("SOT-MODEL-005: details がない")
	}
	details["field"] = "secret"
	got, _ := invalid.Details()
	if got["field"] != "query" {
		t.Fatal("SOT-MODEL-005: details が getter 経由で変更された")
	}

	internal, err := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeInternalError,
	})
	if err != nil {
		t.Fatalf("SOT-IF-027: internal_error の作成エラー = %v", err)
	}
	if _, exists := internal.Details(); exists ||
		internal.Retryable() {
		t.Fatalf("SOT-IF-027: internal_error = %#v", internal)
	}
}

func TestErrorResultJSONAndValidation(t *testing.T) {
	t.Parallel()

	result, err := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeInvalidArgument,
		Details: map[string]any{
			"field":  "offset",
			"reason": "0 以上で指定してください",
		},
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-005: NewErrorResult() のエラー = %v", err)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("SOT-MODEL-005/009: JSON 変換のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-005: JSON を再解析できない: %v", err)
	}
	if object["code"] != "invalid_argument" ||
		object["retryable"] != false ||
		!reflect.DeepEqual(object["details"], map[string]any{
			"field":  "offset",
			"reason": "0 以上で指定してください",
		}) {
		t.Fatalf("SOT-MODEL-005: JSON = %#v", object)
	}

	tests := []model.ErrorResultValues{
		{Code: model.ErrorCode("unknown")},
		{
			Code: model.ErrorCodeInternalError,
			Details: map[string]any{
				"field": "query",
			},
		},
		{
			Code: model.ErrorCodeInvalidArgument,
			Details: map[string]any{
				"unsafeKey": "漏らしてはならない値",
			},
		},
		{
			Code: model.ErrorCodeInvalidArgument,
			Details: map[string]any{
				"field": map[string]any{"nested": true},
			},
		},
	}
	for _, values := range tests {
		if _, err := model.NewErrorResult(values); err == nil {
			t.Fatalf("SOT-MODEL-005/IF-027: 不正な値を受理した: %#v", values)
		}
	}
	if _, err := json.Marshal(model.ErrorResult{}); err == nil {
		t.Fatal("SOT-MODEL-005: zero value を JSON に変換できた")
	}
	var decoded model.ErrorResult
	if err := json.Unmarshal([]byte(`{"code":"internal_error"}`), &decoded); err == nil {
		t.Fatal("SOT-MODEL-005: JSON から直接復元できた")
	}
}
