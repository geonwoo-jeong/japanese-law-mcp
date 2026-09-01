package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestExecuteLegalToolMatchesFullDirectResultForEverySpecialist(t *testing.T) {
	dependencies := newAllTestDependencies(t)
	full, err := NewServerWithDependenciesAndExposure(
		"test-version",
		dependencies,
		ToolExposureFull,
	)
	if err != nil {
		t.Fatalf("full server = %v", err)
	}
	compact, err := NewServerWithDependenciesAndExposure(
		"test-version",
		dependencies,
		ToolExposureCompact,
	)
	if err != nil {
		t.Fatalf("compact server = %v", err)
	}
	tests := []struct {
		name      string
		arguments json.RawMessage
	}{
		{
			name: "compare_law_versions",
			arguments: json.RawMessage(
				`{"lawId":"law-1","before":{"revisionId":"rev-1"},"after":{"revisionId":"rev-2"}}`,
			),
		},
		{name: "get_article", arguments: json.RawMessage(`{"lawId":"law-1","article":"1"}`)},
		{name: "get_judicial_case", arguments: canonicalJudicialCaseInput()},
		{name: "get_law", arguments: json.RawMessage(`{"lawId":"law-1"}`)},
		{name: "list_law_revisions", arguments: json.RawMessage(`{"lawIdOrNumber":"law-1"}`)},
		{name: "list_law_updates", arguments: json.RawMessage(`{"date":"2026-07-26"}`)},
		{name: "search_diet_speeches", arguments: json.RawMessage(`{"query":"永住許可","limit":1}`)},
		{name: "search_judicial_cases", arguments: json.RawMessage(`{"query":"民法"}`)},
		{name: "search_law_content", arguments: json.RawMessage(`{"query":"情報 公開"}`)},
		{name: "search_laws", arguments: json.RawMessage(`{"query":"民法"}`)},
		{name: "trace_judicial_citations", arguments: canonicalTraceJudicialCitationsInput()},
	}
	withMCPClientSession(t, full, func(fullContext context.Context, fullSession *sdk.ClientSession) {
		withMCPClientSession(t, compact, func(compactContext context.Context, compactSession *sdk.ClientSession) {
			for _, testCase := range tests {
				fullResult, fullErr := fullSession.CallTool(
					fullContext,
					&sdk.CallToolParams{
						Name:      testCase.name,
						Arguments: decodeExecuteParityArguments(t, testCase.arguments),
					},
				)
				compactResult, compactErr := compactSession.CallTool(
					compactContext,
					&sdk.CallToolParams{
						Name: executeLegalToolToolName,
						Arguments: map[string]any{
							"toolName":  testCase.name,
							"arguments": decodeExecuteParityArguments(t, testCase.arguments),
						},
					},
				)
				if (fullErr == nil) != (compactErr == nil) {
					t.Fatalf("%s errors = %v / %v", testCase.name, fullErr, compactErr)
				}
				if fullErr != nil {
					continue
				}
				fullJSON, marshalErr := json.Marshal(fullResult)
				if marshalErr != nil {
					t.Fatalf("%s full result marshal = %v", testCase.name, marshalErr)
				}
				compactJSON, marshalErr := json.Marshal(compactResult)
				if marshalErr != nil {
					t.Fatalf("%s compact result marshal = %v", testCase.name, marshalErr)
				}
				if string(fullJSON) != string(compactJSON) {
					t.Fatalf("%s result mismatch:\nfull=%s\ncompact=%s", testCase.name, fullJSON, compactJSON)
				}
			}
		})
	})
}

func TestExecuteLegalToolMatchesFullDirectInputErrorForEverySpecialist(t *testing.T) {
	dependencies := newAllTestDependencies(t)
	full := mustNewFullServer(t, dependencies)
	compact := mustNewCompactServer(t, dependencies)
	names := []string{
		"compare_law_versions",
		"get_article",
		"get_judicial_case",
		"get_law",
		"list_law_revisions",
		"list_law_updates",
		"search_diet_speeches",
		"search_judicial_cases",
		"search_law_content",
		"search_laws",
		"trace_judicial_citations",
	}
	withMCPClientSession(t, full, func(fullContext context.Context, fullSession *sdk.ClientSession) {
		withMCPClientSession(t, compact, func(compactContext context.Context, compactSession *sdk.ClientSession) {
			for _, name := range names {
				fullResult, fullErr := fullSession.CallTool(
					fullContext,
					&sdk.CallToolParams{Name: name, Arguments: map[string]any{}},
				)
				compactResult, compactErr := compactSession.CallTool(
					compactContext,
					&sdk.CallToolParams{
						Name: executeLegalToolToolName,
						Arguments: map[string]any{
							"toolName":  name,
							"arguments": map[string]any{},
						},
					},
				)
				if fullErr != nil || compactErr != nil {
					t.Fatalf("%s errors = %v / %v", name, fullErr, compactErr)
				}
				fullJSON, marshalErr := json.Marshal(fullResult)
				if marshalErr != nil {
					t.Fatalf("%s full result marshal = %v", name, marshalErr)
				}
				compactJSON, marshalErr := json.Marshal(compactResult)
				if marshalErr != nil {
					t.Fatalf("%s compact result marshal = %v", name, marshalErr)
				}
				if string(fullJSON) != string(compactJSON) {
					t.Fatalf("%s input error mismatch:\nfull=%s\ncompact=%s", name, fullJSON, compactJSON)
				}
			}
		})
	})
}

func TestExecuteLegalToolClonesRequestAndPreservesRawArgumentsAndResult(t *testing.T) {
	const nested = `{"alpha":[1,{ "beta":true}],"escaped":"\\uD800"}`
	var received *sdk.CallToolRequest
	want := &sdk.CallToolResult{
		Content: []sdk.Content{&sdk.TextContent{Text: `{"ok":true}`}},
		StructuredContent: map[string]any{
			"ok": true,
		},
	}
	registry := mustSyntheticOperationRegistry(
		t,
		"synthetic_operation",
		func(_ context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			received = request
			request.Params.Meta["progressToken"] = "changed"
			request.Params.Meta["nested"].(map[string]any)["value"] = "changed"
			return want, nil
		},
	)
	extra := &sdk.RequestExtra{}
	session := &sdk.ServerSession{}
	outerRaw := json.RawMessage(
		`{"toolName":"synthetic_operation","arguments":` + nested + `}`,
	)
	request := &sdk.CallToolRequest{
		Session: session,
		Params: &sdk.CallToolParamsRaw{
			Meta: sdk.Meta{
				"progressToken": "token-1",
				"nested": map[string]any{
					"value": "original",
				},
			},
			Name:      executeLegalToolToolName,
			Arguments: outerRaw,
		},
		Extra: extra,
	}
	result, err := callExecuteLegalTool(context.Background(), registry, request)
	if err != nil {
		t.Fatalf("callExecuteLegalTool() error = %v", err)
	}
	if result != want {
		t.Fatalf("result pointer = %p, want %p", result, want)
	}
	if received == nil || received == request || received.Params == request.Params {
		t.Fatalf("forwarded request = %#v", received)
	}
	if received.Session != session || received.Extra != extra ||
		received.Params.Name != "synthetic_operation" ||
		string(received.Params.Arguments) != nested {
		t.Fatalf("forwarded request = %#v", received)
	}
	if request.Params.Meta["progressToken"] != "token-1" ||
		request.Params.Meta["nested"].(map[string]any)["value"] != "original" {
		t.Fatalf("outer request meta was mutated = %#v", request.Params.Meta)
	}
	if request.Params.Name != executeLegalToolToolName ||
		string(request.Params.Arguments) != string(outerRaw) {
		t.Fatalf("outer request was mutated = %#v", request.Params)
	}
}

func TestExecuteLegalToolPropagatesToolErrorGoErrorAndCancellation(t *testing.T) {
	t.Run("tool error", func(t *testing.T) {
		want := &sdk.CallToolResult{
			Content: []sdk.Content{&sdk.TextContent{Text: `{"code":"invalid_argument"}`}},
			IsError: true,
		}
		registry := mustSyntheticOperationRegistry(
			t,
			"synthetic_operation",
			func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
				return want, nil
			},
		)
		result, err := callSyntheticExecute(t, context.Background(), registry)
		if err != nil || result != want || !result.IsError {
			t.Fatalf("result = %#v, error = %v", result, err)
		}
	})

	t.Run("Go error", func(t *testing.T) {
		wantErr := errors.New("handler failed")
		registry := mustSyntheticOperationRegistry(
			t,
			"synthetic_operation",
			func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
				return nil, wantErr
			},
		)
		result, err := callSyntheticExecute(t, context.Background(), registry)
		if result != nil || !errors.Is(err, wantErr) {
			t.Fatalf("result = %#v, error = %v", result, err)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		registry := mustSyntheticOperationRegistry(
			t,
			"synthetic_operation",
			func(ctx context.Context, _ *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		result, err := callSyntheticExecute(t, ctx, registry)
		if result != nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("result = %#v, error = %v", result, err)
		}
	})
}

func TestExecuteLegalToolRejectsUnknownDisabledAndRecursiveNamesUniformly(t *testing.T) {
	registry := mustSpecialistRegistry(t, newCoreTestDependencies(t))
	names := []string{
		"unknown_operation",
		"search_diet_speeches",
		queryLegalInformationToolName,
		discoverLegalToolsToolName,
		executeLegalToolToolName,
	}
	var wantPayload string
	for _, name := range names {
		result, err := callExecuteLegalTool(
			context.Background(),
			registry,
			&sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{
				Name: executeLegalToolToolName,
				Arguments: json.RawMessage(
					`{"toolName":"` + name + `","arguments":{}}`,
				),
			}},
		)
		if err != nil {
			t.Fatalf("%s error = %v", name, err)
		}
		assertMetaInvalidArgumentField(t, result, "toolName")
		payload := result.Content[0].(*sdk.TextContent).Text
		if wantPayload == "" {
			wantPayload = payload
		} else if payload != wantPayload {
			t.Fatalf("%s payload = %s, want %s", name, payload, wantPayload)
		}
	}
}

func TestExecuteLegalToolStrictAndRecursiveInputBoundaries(t *testing.T) {
	var calls atomic.Int32
	registry := mustSyntheticOperationRegistry(
		t,
		"synthetic_operation",
		func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			calls.Add(1)
			return successMetaToolResult(map[string]any{"ok": true})
		},
	)
	invalidUTF8 := append(
		[]byte(`{"toolName":"synthetic_operation","arguments":{"value":"`),
		append([]byte{0xff}, []byte(`"}}`)...)...,
	)
	tests := []struct {
		name      string
		arguments []byte
		field     string
	}{
		{name: "non-object", arguments: []byte(`[]`), field: "arguments"},
		{name: "unknown outer field", arguments: []byte(`{"toolName":"synthetic_operation","arguments":{},"unknown":true}`), field: "arguments"},
		{name: "duplicate outer field", arguments: []byte(`{"toolName":"synthetic_operation","toolName":"synthetic_operation","arguments":{}}`), field: "arguments"},
		{name: "missing toolName", arguments: []byte(`{"arguments":{}}`), field: "toolName"},
		{name: "null toolName", arguments: []byte(`{"toolName":null,"arguments":{}}`), field: "toolName"},
		{name: "toolName type", arguments: []byte(`{"toolName":1,"arguments":{}}`), field: "toolName"},
		{name: "empty toolName", arguments: []byte(`{"toolName":"  ","arguments":{}}`), field: "toolName"},
		{name: "toolName too large", arguments: []byte(`{"toolName":"` + strings.Repeat("a", 129) + `","arguments":{}}`), field: "toolName"},
		{name: "toolName control", arguments: []byte(`{"toolName":"synthetic\u0000operation","arguments":{}}`), field: "toolName"},
		{name: "missing arguments", arguments: []byte(`{"toolName":"synthetic_operation"}`), field: "arguments"},
		{name: "null arguments", arguments: []byte(`{"toolName":"synthetic_operation","arguments":null}`), field: "arguments"},
		{name: "array arguments", arguments: []byte(`{"toolName":"synthetic_operation","arguments":[]}`), field: "arguments"},
		{name: "duplicate nested root", arguments: []byte(`{"toolName":"synthetic_operation","arguments":{"key":1,"key":2}}`), field: "arguments"},
		{name: "duplicate nested object", arguments: []byte(`{"toolName":"synthetic_operation","arguments":{"outer":{"key":1,"key":2}}}`), field: "arguments"},
		{name: "duplicate object in array", arguments: []byte(`{"toolName":"synthetic_operation","arguments":{"items":[{"key":1,"key":2}]}}`), field: "arguments"},
		{name: "invalid UTF-8", arguments: invalidUTF8, field: "arguments"},
		{name: "high surrogate", arguments: []byte(`{"toolName":"synthetic_operation","arguments":{"value":"\uD800"}}`), field: "arguments"},
		{name: "low surrogate", arguments: []byte(`{"toolName":"synthetic_operation","arguments":{"value":"\uDC00"}}`), field: "arguments"},
		{
			name:      "outer arguments too large",
			arguments: []byte(`{"toolName":"synthetic_operation","arguments":{"padding":"` + strings.Repeat("a", executeLegalToolMaxArgumentsBytes) + `"}}`),
			field:     "arguments",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := callExecuteLegalTool(
				context.Background(),
				registry,
				&sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{
					Name:      executeLegalToolToolName,
					Arguments: testCase.arguments,
				}},
			)
			if err != nil {
				t.Fatalf("callExecuteLegalTool() error = %v", err)
			}
			assertMetaInvalidArgumentField(t, result, testCase.field)
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("rejected input handler calls = %d", calls.Load())
	}
}

func TestExecuteLegalToolTrimsNameAndPreservesValidEscapedLiteral(t *testing.T) {
	var got json.RawMessage
	registry := mustSyntheticOperationRegistry(
		t,
		"synthetic_operation",
		func(_ context.Context, request *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			got = append(json.RawMessage(nil), request.Params.Arguments...)
			return successMetaToolResult(map[string]any{"ok": true})
		},
	)
	want := `{"literal":"\\uD800"}`
	result, err := callExecuteLegalTool(
		context.Background(),
		registry,
		&sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{
			Name: executeLegalToolToolName,
			Arguments: json.RawMessage(
				`{"toolName":"  synthetic_operation  ","arguments":` + want + `}`,
			),
		}},
	)
	if err != nil || result == nil || result.IsError {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
	if string(got) != want {
		t.Fatalf("raw arguments = %q, want %q", got, want)
	}
}

func TestMetaToolSchemasExposeJSONAndUTF8ByteLimits(t *testing.T) {
	tests := []struct {
		name        string
		schema      *jsonschema.Schema
		maxJSON     int
		stringField string
		maxUTF8     int
	}{
		{
			name:        "discover",
			schema:      newDiscoverLegalToolsInputSchema(),
			maxJSON:     discoverLegalToolsMaxArgumentsBytes,
			stringField: "query",
			maxUTF8:     discoverLegalToolsMaxQueryBytes,
		},
		{
			name:        "execute",
			schema:      newExecuteLegalToolInputSchema(),
			maxJSON:     executeLegalToolMaxArgumentsBytes,
			stringField: "toolName",
			maxUTF8:     128,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			payload, err := json.Marshal(testCase.schema)
			if err != nil {
				t.Fatalf("schema marshal = %v", err)
			}
			var object map[string]any
			if err := json.Unmarshal(payload, &object); err != nil {
				t.Fatalf("schema decode = %v", err)
			}
			if got := int(object["x-maxJsonBytes"].(float64)); got != testCase.maxJSON {
				t.Fatalf("x-maxJsonBytes = %d, want %d", got, testCase.maxJSON)
			}
			properties := object["properties"].(map[string]any)
			field := properties[testCase.stringField].(map[string]any)
			if got := int(field["x-maxUTF8Bytes"].(float64)); got != testCase.maxUTF8 {
				t.Fatalf("x-maxUTF8Bytes = %d, want %d", got, testCase.maxUTF8)
			}
		})
	}
}

func TestMetaToolArgumentByteLimitsAcceptExactBoundary(t *testing.T) {
	discoverRaw := append(
		[]byte(`{}`),
		[]byte(strings.Repeat(" ", discoverLegalToolsMaxArgumentsBytes-2))...,
	)
	if len(discoverRaw) != discoverLegalToolsMaxArgumentsBytes {
		t.Fatalf("discover boundary bytes = %d", len(discoverRaw))
	}
	registry := mustSyntheticOperationRegistry(
		t,
		"synthetic_operation",
		func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			return successMetaToolResult(map[string]any{"ok": true})
		},
	)
	discoverResult, err := callDiscoverLegalTools(
		context.Background(),
		registry,
		&sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{
			Name:      discoverLegalToolsToolName,
			Arguments: discoverRaw,
		}},
	)
	if err != nil || discoverResult == nil || discoverResult.IsError {
		t.Fatalf("discover boundary result = %#v, error = %v", discoverResult, err)
	}

	executeBase := []byte(`{"toolName":"synthetic_operation","arguments":{}}`)
	executeRaw := append(
		executeBase,
		[]byte(strings.Repeat(" ", executeLegalToolMaxArgumentsBytes-len(executeBase)))...,
	)
	if len(executeRaw) != executeLegalToolMaxArgumentsBytes {
		t.Fatalf("execute boundary bytes = %d", len(executeRaw))
	}
	executeResult, err := callExecuteLegalTool(
		context.Background(),
		registry,
		&sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{
			Name:      executeLegalToolToolName,
			Arguments: executeRaw,
		}},
	)
	if err != nil || executeResult == nil || executeResult.IsError {
		t.Fatalf("execute boundary result = %#v, error = %v", executeResult, err)
	}
}

func callSyntheticExecute(
	t *testing.T,
	ctx context.Context,
	registry operationRegistry,
) (*sdk.CallToolResult, error) {
	t.Helper()
	return callExecuteLegalTool(
		ctx,
		registry,
		&sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{
			Name: executeLegalToolToolName,
			Arguments: json.RawMessage(
				`{"toolName":"synthetic_operation","arguments":{}}`,
			),
		}},
	)
}

func decodeExecuteParityArguments(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	var arguments any
	if err := json.Unmarshal(raw, &arguments); err != nil {
		t.Fatalf("test arguments decode = %v", err)
	}
	return arguments
}

func mustSyntheticOperationRegistry(
	t *testing.T,
	name string,
	handler sdk.ToolHandler,
) operationRegistry {
	t.Helper()
	builder := newOperationRegistryBuilder()
	builder.AddTool(&sdk.Tool{
		Name:         name,
		Description:  "テスト用の読み取り専用専門操作です。",
		InputSchema:  &jsonschema.Schema{Type: "object"},
		OutputSchema: &jsonschema.Schema{Type: "object"},
	}, handler)
	registry, err := builder.build()
	if err != nil {
		t.Fatalf("synthetic registry = %v", err)
	}
	return registry
}

func assertMetaInvalidArgumentField(
	t *testing.T,
	result *sdk.CallToolResult,
	wantField string,
) {
	t.Helper()
	if result == nil || !result.IsError || result.StructuredContent != nil ||
		len(result.Content) != 1 {
		t.Fatalf("invalid argument result = %#v", result)
	}
	text, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("error content type = %T", result.Content[0])
	}
	var payload struct {
		Code    string `json:"code"`
		Details struct {
			Field  string `json:"field"`
			Reason string `json:"reason"`
		} `json:"details"`
	}
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		t.Fatalf("ErrorResult decode = %v", err)
	}
	if payload.Code != "invalid_argument" || payload.Details.Field != wantField ||
		payload.Details.Reason == "" {
		t.Fatalf("ErrorResult = %#v", payload)
	}
}
