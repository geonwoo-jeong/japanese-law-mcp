package legalquerycorpus

import (
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestExecutionActionV1Decodeは四outcomeを復元する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		outcome map[string]any
		kind    ExecutionOutcomeKind
	}{
		{
			name: "collection success",
			outcome: map[string]any{
				"kind":            "collection_success",
				"sourceItemCount": float64(1000),
			},
			kind: ExecutionOutcomeKindCollectionSuccess,
		},
		{
			name:    "read success",
			outcome: map[string]any{"kind": "read_success"},
			kind:    ExecutionOutcomeKindReadSuccess,
		},
		{
			name: "failure",
			outcome: map[string]any{
				"kind":      "failure",
				"errorCode": "source_unavailable",
			},
			kind: ExecutionOutcomeKindFailure,
		},
		{
			name:    "timeout",
			outcome: map[string]any{"kind": "timeout"},
			kind:    ExecutionOutcomeKindTimeout,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			action, err := decodeExecutionActionV1(
				mustJSONBytes(t, validExecutionActionForTest(test.outcome)),
			)
			if err != nil {
				t.Fatalf("SOT-ENG-026: execution action decode error = %v", err)
			}
			if action.Outcome().Kind() != test.kind {
				t.Fatalf("SOT-ENG-026: outcome kind = %q", action.Outcome().Kind())
			}
		})
	}
}

func TestExecutionActionV1Decodeは必須項目と未知項目を拒否する(t *testing.T) {
	t.Parallel()

	for _, field := range []string{
		"meaningId",
		"stepOrdinal",
		"releaseOrder",
		"outcome",
	} {
		field := field
		t.Run(field+"欠落", func(t *testing.T) {
			t.Parallel()
			source := validExecutionActionForTest(
				map[string]any{"kind": "read_success"},
			)
			delete(source, field)
			if _, err := decodeExecutionActionV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 必須項目欠落を受理した")
			}
		})
	}

	tests := map[string]func(map[string]any){
		"action未知項目": func(source map[string]any) {
			source["providerId"] = "forbidden"
		},
		"outcome未知項目": func(source map[string]any) {
			source["outcome"].(map[string]any)["response"] = "forbidden"
		},
		"outcome kind欠落": func(source map[string]any) {
			delete(source["outcome"].(map[string]any), "kind")
		},
		"outcome kind未知": func(source map[string]any) {
			source["outcome"].(map[string]any)["kind"] = "unknown"
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			source := validExecutionActionForTest(
				map[string]any{"kind": "read_success"},
			)
			mutate(source)
			if _, err := decodeExecutionActionV1(mustJSONBytes(t, source)); err == nil {
				t.Fatal("SOT-ENG-026: 不正な execution action を受理した")
			}
		})
	}
	trailing := append(
		mustJSONBytes(
			t,
			validExecutionActionForTest(map[string]any{"kind": "read_success"}),
		),
		[]byte(` {}`)...,
	)
	if _, err := decodeExecutionActionV1(trailing); err == nil {
		t.Fatal("SOT-ENG-026: 末尾の別 JSON 値を受理した")
	}
}

func TestExecutionOutcomeV1Decodeはvariant固有項目を閉じる(t *testing.T) {
	t.Parallel()

	tests := []map[string]any{
		{"kind": "collection_success"},
		{"kind": "collection_success", "sourceItemCount": float64(-1)},
		{"kind": "collection_success", "sourceItemCount": float64(1001)},
		{"kind": "read_success", "sourceItemCount": float64(1)},
		{"kind": "failure"},
		{"kind": "failure", "errorCode": "invalid_argument"},
		{"kind": "timeout", "errorCode": "source_timeout"},
	}
	for _, outcome := range tests {
		if _, err := decodeExecutionOutcomeV1(mustJSONBytes(t, outcome)); err == nil {
			t.Fatalf("SOT-ENG-026: 不正な outcome を受理した: %#v", outcome)
		}
	}

	for _, code := range allowedExecutionFailureCodesForTest() {
		outcome, err := decodeExecutionOutcomeV1(mustJSONBytes(t, map[string]any{
			"kind":      "failure",
			"errorCode": string(code),
		}))
		if err != nil {
			t.Fatalf("SOT-ENG-026: errorCode=%q decode error = %v", code, err)
		}
		if outcome.(FailureOutcome).ErrorCode() != code {
			t.Fatalf("SOT-ENG-026: errorCode = %q", outcome.(FailureOutcome).ErrorCode())
		}
	}
}

func TestExecutionAction公開型は動的JSONを保持しない(t *testing.T) {
	t.Parallel()

	for _, current := range []reflect.Type{
		reflect.TypeOf(ExecutionAction{}),
		reflect.TypeOf(ExecutionActionValues{}),
		reflect.TypeOf(CollectionSuccessOutcome{}),
		reflect.TypeOf(ReadSuccessOutcome{}),
		reflect.TypeOf(FailureOutcome{}),
		reflect.TypeOf(TimeoutOutcome{}),
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

func TestTimeoutOutcomeはsourceTimeoutへ投影できる分類を持つ(t *testing.T) {
	t.Parallel()

	timeout, err := decodeExecutionOutcomeV1(
		mustJSONBytes(t, map[string]any{"kind": "timeout"}),
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: timeout decode error = %v", err)
	}
	if timeout.Kind() != ExecutionOutcomeKindTimeout ||
		model.ErrorCodeSourceTimeout != "source_timeout" {
		t.Fatal("SOT-ENG-026: timeout の固定投影前提が一致しない")
	}
}

func validExecutionActionForTest(outcome map[string]any) map[string]any {
	return map[string]any{
		"meaningId":    "law-search",
		"stepOrdinal":  float64(1),
		"releaseOrder": float64(1),
		"outcome":      outcome,
	}
}
