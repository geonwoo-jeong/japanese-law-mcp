package legalquerycorpus

import (
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestExpectedAttemptV1Decodeは四つの形を復元する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source map[string]any
		target any
	}{
		{
			name: "completed read",
			source: map[string]any{
				"meaningId":          "law-read",
				"stepOrdinal":        float64(1),
				"outcome":            "completed",
				"publishedItemCount": float64(1),
			},
			target: ExpectedCompletedReadAttempt{},
		},
		{
			name: "completed collection",
			source: map[string]any{
				"meaningId":          "law-search",
				"stepOrdinal":        float64(1),
				"outcome":            "completed",
				"publishedItemCount": float64(20),
				"hasMore":            true,
			},
			target: ExpectedCompletedCollectionAttempt{},
		},
		{
			name: "empty",
			source: map[string]any{
				"meaningId":          "law-search",
				"stepOrdinal":        float64(1),
				"outcome":            "empty",
				"publishedItemCount": float64(0),
				"hasMore":            false,
			},
			target: ExpectedEmptyAttempt{},
		},
		{
			name: "failed",
			source: map[string]any{
				"meaningId":   "law-search",
				"stepOrdinal": float64(1),
				"outcome":     "failed",
				"errorCode":   "source_busy",
			},
			target: ExpectedFailedAttempt{},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			attempt, err := decodeExpectedAttemptV1(mustJSONBytes(t, test.source))
			if err != nil {
				t.Fatalf("SOT-ENG-026: expected attempt decode error = %v", err)
			}
			if reflect.TypeOf(attempt) != reflect.TypeOf(test.target) {
				t.Fatalf("SOT-ENG-026: attempt type = %T", attempt)
			}
		})
	}
}

func TestExpectedAttemptV1DecodeはcompletedをhasMoreの存在で形判定する(t *testing.T) {
	t.Parallel()

	collection, err := decodeExpectedAttemptV1(mustJSONBytes(t, map[string]any{
		"meaningId":          "meaning-one",
		"stepOrdinal":        float64(1),
		"outcome":            "completed",
		"publishedItemCount": float64(1),
		"hasMore":            false,
	}))
	if err != nil {
		t.Fatalf("SOT-ENG-026: collection shape decode error = %v", err)
	}
	read, err := decodeExpectedAttemptV1(mustJSONBytes(t, map[string]any{
		"meaningId":          "meaning-two",
		"stepOrdinal":        float64(1),
		"outcome":            "completed",
		"publishedItemCount": float64(1),
	}))
	if err != nil {
		t.Fatalf("SOT-ENG-026: read shape decode error = %v", err)
	}
	if _, ok := collection.(ExpectedCompletedCollectionAttempt); !ok {
		t.Fatalf("SOT-ENG-026: hasMore ありの型 = %T", collection)
	}
	if _, ok := read.(ExpectedCompletedReadAttempt); !ok {
		t.Fatalf("SOT-ENG-026: hasMore なしの型 = %T", read)
	}
}

func TestExpectedAttemptV1Decodeはvariantを厳密に閉じる(t *testing.T) {
	t.Parallel()

	tests := []map[string]any{
		{},
		{
			"meaningId": "law-read", "stepOrdinal": float64(1),
			"outcome": "completed",
		},
		{
			"meaningId": "law-read", "stepOrdinal": float64(1),
			"outcome": "completed", "publishedItemCount": float64(2),
		},
		{
			"meaningId": "law-read", "stepOrdinal": float64(1),
			"outcome": "completed", "publishedItemCount": float64(1),
			"errorCode": "source_busy",
		},
		{
			"meaningId": "law-search", "stepOrdinal": float64(1),
			"outcome": "empty", "publishedItemCount": float64(0),
		},
		{
			"meaningId": "law-search", "stepOrdinal": float64(1),
			"outcome": "empty", "publishedItemCount": float64(0), "hasMore": true,
		},
		{
			"meaningId": "law-search", "stepOrdinal": float64(1),
			"outcome": "failed",
		},
		{
			"meaningId": "law-search", "stepOrdinal": float64(1),
			"outcome": "failed", "errorCode": "invalid_argument",
		},
		{
			"meaningId": "law-search", "stepOrdinal": float64(1),
			"outcome": "unknown", "errorCode": "source_busy",
		},
	}
	for _, source := range tests {
		if _, err := decodeExpectedAttemptV1(mustJSONBytes(t, source)); err == nil {
			t.Fatalf("SOT-ENG-026: 不正な expected attempt を受理した: %#v", source)
		}
	}
	trailing := append(
		mustJSONBytes(t, map[string]any{
			"meaningId": "law-read", "stepOrdinal": float64(1),
			"outcome": "completed", "publishedItemCount": float64(1),
		}),
		[]byte(` {}`)...,
	)
	if _, err := decodeExpectedAttemptV1(trailing); err == nil {
		t.Fatal("SOT-ENG-026: 末尾の別 JSON 値を受理した")
	}
}

func TestExecutionExpectedV1Decodeはresultとerrorを復元する(t *testing.T) {
	t.Parallel()

	resultSource := map[string]any{
		"terminal":          "result",
		"status":            "completed",
		"returnedItemCount": float64(1),
		"attempts": []any{map[string]any{
			"meaningId": "law-read", "stepOrdinal": float64(1),
			"outcome": "completed", "publishedItemCount": float64(1),
		}},
	}
	errorSource := map[string]any{
		"terminal":  "error",
		"errorCode": "source_busy",
		"attempts": []any{map[string]any{
			"meaningId": "law-search", "stepOrdinal": float64(1),
			"outcome": "failed", "errorCode": "source_busy",
		}},
	}
	result, err := decodeExecutionExpectedV1(mustJSONBytes(t, resultSource))
	if err != nil {
		t.Fatalf("SOT-ENG-026: result decode error = %v", err)
	}
	failure, err := decodeExecutionExpectedV1(mustJSONBytes(t, errorSource))
	if err != nil {
		t.Fatalf("SOT-ENG-026: error decode error = %v", err)
	}
	if result.(ExecutionExpectedResult).Status() !=
		legalquery.LegalQueryResultStatusCompleted ||
		failure.(ExecutionExpectedError).ErrorCode() != model.ErrorCodeSourceBusy {
		t.Fatal("SOT-ENG-026: execution expected の値が一致しない")
	}
}

func TestExecutionExpectedV1Decodeは必須項目とvariant未知項目を拒否する(t *testing.T) {
	t.Parallel()

	validResult := func() map[string]any {
		return map[string]any{
			"terminal":          "result",
			"status":            "empty",
			"returnedItemCount": float64(0),
			"attempts": []any{map[string]any{
				"meaningId": "law-search", "stepOrdinal": float64(1),
				"outcome": "empty", "publishedItemCount": float64(0),
				"hasMore": false,
			}},
		}
	}
	for _, field := range []string{
		"terminal", "status", "returnedItemCount", "attempts",
	} {
		source := validResult()
		delete(source, field)
		if _, err := decodeExecutionExpectedV1(mustJSONBytes(t, source)); err == nil {
			t.Fatalf("SOT-ENG-026: result.%s 欠落を受理した", field)
		}
	}

	validError := func() map[string]any {
		return map[string]any{
			"terminal":  "error",
			"errorCode": "source_busy",
			"attempts": []any{map[string]any{
				"meaningId": "law-search", "stepOrdinal": float64(1),
				"outcome": "failed", "errorCode": "source_busy",
			}},
		}
	}
	for _, field := range []string{"terminal", "errorCode", "attempts"} {
		source := validError()
		delete(source, field)
		if _, err := decodeExecutionExpectedV1(mustJSONBytes(t, source)); err == nil {
			t.Fatalf("SOT-ENG-026: error.%s 欠落を受理した", field)
		}
	}

	unknown := validResult()
	unknown["notices"] = []any{}
	errorWithStatus := map[string]any{
		"terminal": "error", "errorCode": "source_busy",
		"status": "partial",
		"attempts": []any{map[string]any{
			"meaningId": "law-search", "stepOrdinal": float64(1),
			"outcome": "failed", "errorCode": "source_busy",
		}},
	}
	unknownTerminal := validResult()
	unknownTerminal["terminal"] = "unknown"
	for _, source := range []map[string]any{unknown, errorWithStatus, unknownTerminal} {
		if _, err := decodeExecutionExpectedV1(mustJSONBytes(t, source)); err == nil {
			t.Fatalf("SOT-ENG-026: 不正な execution expected を受理した: %#v", source)
		}
	}
	trailing := append(mustJSONBytes(t, validResult()), []byte(` {}`)...)
	if _, err := decodeExecutionExpectedV1(trailing); err == nil {
		t.Fatal("SOT-ENG-026: execution expected の末尾値を受理した")
	}
}

func TestExecutionExpected公開型は動的JSONを保持しない(t *testing.T) {
	t.Parallel()

	for _, current := range []reflect.Type{
		reflect.TypeOf(ExpectedCompletedReadAttempt{}),
		reflect.TypeOf(ExpectedCompletedCollectionAttempt{}),
		reflect.TypeOf(ExpectedEmptyAttempt{}),
		reflect.TypeOf(ExpectedFailedAttempt{}),
		reflect.TypeOf(ExecutionExpectedResult{}),
		reflect.TypeOf(ExecutionExpectedError{}),
		reflect.TypeOf(ExecutionExpectedResultValues{}),
		reflect.TypeOf(ExecutionExpectedErrorValues{}),
	} {
		for index := 0; index < current.NumField(); index++ {
			field := current.Field(index)
			if field.Type.Kind() == reflect.Map ||
				field.Type.String() == "json.RawMessage" ||
				field.Type.String() == "[]json.RawMessage" {
				t.Fatalf(
					"SOT-ENG-026: 公開型 %s.%s が動的 JSON を保持する",
					current.Name(),
					field.Name,
				)
			}
		}
	}
}
