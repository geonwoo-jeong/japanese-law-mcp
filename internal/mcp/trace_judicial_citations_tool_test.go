package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationtrace"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCallTraceJudicialCitationsReturnsExactStructuredResultAndDefaults(t *testing.T) {
	t.Parallel()

	want := mustJudicialCitationGraphResult(t)
	port := &stubTraceJudicialCitationsPort{result: want}
	result, err := callTraceJudicialCitations(
		context.Background(),
		port,
		canonicalTraceJudicialCitationsInput(),
	)
	if err != nil {
		t.Fatalf("callTraceJudicialCitations() error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("result = %#v", result)
	}
	if port.calls != 1 ||
		port.request.Direction() != model.JudicialCitationRequestedDirectionBoth ||
		port.request.IncomingLimit() != judicialcitationtrace.DefaultIncomingLimit {
		t.Fatalf("request = %#v, calls = %d", port.request, port.calls)
	}
	structured, ok := result.StructuredContent.(json.RawMessage)
	if !ok {
		t.Fatalf("structuredContent type = %T", result.StructuredContent)
	}
	assertSameJudicialJSONValue(t, structured, want)
	if text := firstToolText(t, result); !jsonValuesEqual(t, []byte(text), structured) {
		t.Fatal("content と structuredContent が一致しません")
	}
}

func TestTraceJudicialCitationsToolRoundTripsThroughMCP(t *testing.T) {
	t.Parallel()

	want := mustJudicialCitationGraphResult(t)
	port := &stubTraceJudicialCitationsPort{result: want}
	dependencies, err := NewJudicialCitationsDependencies(port)
	if err != nil {
		t.Fatalf("dependencies = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	judicialCases := mustJudicialCitationCasesDependencies(t)
	go func() {
		serverResult <- NewServerWithDependencies(
			"test-version",
			Dependencies{
				JudicialCases:     judicialCases,
				JudicialCitations: dependencies,
			},
		).Run(ctx, serverTransport)
	}()
	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("MCP connect = %v", err)
	}
	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "trace_judicial_citations",
		Arguments: map[string]any{
			"ref": map[string]any{
				"providerId": "courts-hanrei-html",
				"key": map[string]any{
					"sourceId":     "courts-hanrei",
					"resourceType": "judicial-decision",
					"resourceId":   "95570/detail2",
				},
			},
		},
	})
	if err != nil || result == nil || result.IsError {
		t.Fatalf("MCP result = %#v, error = %v", result, err)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("structuredContent marshal = %v", err)
	}
	assertSameJudicialJSONValue(t, structured, want)
	if port.calls != 1 {
		t.Fatalf("trace port calls = %d", port.calls)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("MCP close = %v", err)
	}
	select {
	case err := <-serverResult:
		if err != nil {
			t.Fatalf("MCP server = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("MCP server close timeout = %v", ctx.Err())
	}
}

func TestTraceJudicialCitationsStreamableHTTPMatchesStatefulSchemaAndResult(t *testing.T) {
	t.Parallel()

	want := mustJudicialCitationGraphResult(t)
	statefulPort := &stubTraceJudicialCitationsPort{result: want}
	statefulCitations, err := NewJudicialCitationsDependencies(statefulPort)
	if err != nil {
		t.Fatalf("stateful dependencies = %v", err)
	}
	statefulSchema := traceJudicialCitationsSchemaFromServer(
		t,
		NewServerWithDependencies("test-version", Dependencies{
			JudicialCases:     mustJudicialCitationCasesDependencies(t),
			JudicialCitations: statefulCitations,
		}),
	)

	httpPort := &stubTraceJudicialCitationsPort{result: want}
	httpCitations, err := NewJudicialCitationsDependencies(httpPort)
	if err != nil {
		t.Fatalf("HTTP dependencies = %v", err)
	}
	httpCases := mustJudicialCitationCasesDependencies(t)
	handler := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server {
			return NewSessionlessServerWithDependencies(
				"test-version",
				Dependencies{
					JudicialCases:     httpCases,
					JudicialCitations: httpCitations,
				},
			)
		},
		&sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)
	toolsPayload := callTraceJudicialCitationsHTTP(
		t,
		handler,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
	)
	var toolsEnvelope struct {
		Result struct {
			Tools []*sdk.Tool `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(toolsPayload, &toolsEnvelope); err != nil {
		t.Fatalf("HTTP tools/list decode = %v", err)
	}
	httpSchema := traceJudicialCitationsSchemaFromTools(t, toolsEnvelope.Result.Tools)
	if string(httpSchema.input) != string(statefulSchema.input) ||
		string(httpSchema.output) != string(statefulSchema.output) {
		t.Fatal("stateful と Streamable HTTP の schema が一致しません")
	}
	callPayload := callTraceJudicialCitationsHTTP(
		t,
		handler,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"trace_judicial_citations","arguments":{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}}}`,
	)
	var callEnvelope struct {
		Result struct {
			StructuredContent json.RawMessage `json:"structuredContent"`
			IsError           bool            `json:"isError"`
		} `json:"result"`
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(callPayload, &callEnvelope); err != nil {
		t.Fatalf("HTTP tools/call decode = %v", err)
	}
	if len(callEnvelope.Error) != 0 || callEnvelope.Result.IsError {
		t.Fatalf("HTTP tools/call = %s", callPayload)
	}
	assertSameJudicialJSONValue(t, callEnvelope.Result.StructuredContent, want)
	if httpPort.calls != 1 {
		t.Fatalf("HTTP trace port calls = %d", httpPort.calls)
	}
}

func callTraceJudicialCitationsHTTP(
	t *testing.T,
	handler http.Handler,
	body string,
) []byte {
	t.Helper()

	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/mcp",
		strings.NewReader(body),
	)
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("HTTP status = %d, body = %s", recorder.Code, recorder.Body.Bytes())
	}
	return append([]byte(nil), recorder.Body.Bytes()...)
}

func TestCallTraceJudicialCitationsForwardsExplicitDirectionAndLimit(t *testing.T) {
	t.Parallel()

	port := &stubTraceJudicialCitationsPort{
		result: mustJudicialCitationGraphResultForDirection(
			t,
			model.JudicialCitationRequestedDirectionOutgoing,
		),
	}
	result, err := callTraceJudicialCitations(
		context.Background(),
		port,
		json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}},"direction":"outgoing","incomingLimit":10}`),
	)
	if err != nil || result == nil || result.IsError {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
	if port.calls != 1 ||
		port.request.Direction() != model.JudicialCitationRequestedDirectionOutgoing ||
		port.request.IncomingLimit() != 10 {
		t.Fatalf("request = %#v, calls = %d", port.request, port.calls)
	}
}

func TestCallTraceJudicialCitationsRejectsInvalidInputBeforePort(t *testing.T) {
	t.Parallel()

	invalidUTF8 := append(
		[]byte(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"`),
		append([]byte{0xff}, []byte(`"}}}`)...)...,
	)
	tests := []struct {
		name      string
		arguments json.RawMessage
		field     string
	}{
		{name: "入力なし", arguments: nil, field: "ref"},
		{name: "null", arguments: json.RawMessage(`null`), field: "ref"},
		{name: "配列", arguments: json.RawMessage(`[]`), field: "ref"},
		{name: "不正 JSON", arguments: json.RawMessage(`{"ref":`), field: "ref"},
		{name: "ref 欠落", arguments: json.RawMessage(`{}`), field: "ref"},
		{name: "ref null", arguments: json.RawMessage(`{"ref":null}`), field: "ref"},
		{name: "ref 型", arguments: json.RawMessage(`{"ref":"case"}`), field: "ref"},
		{name: "未知項目", arguments: withTraceInputSuffix(`,"unknown":true`), field: "ref"},
		{name: "maxDepth", arguments: withTraceInputSuffix(`,"maxDepth":1`), field: "ref"},
		{name: "ref 未知項目", arguments: json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"},"unknown":true}}`), field: "ref"},
		{name: "provider null", arguments: json.RawMessage(`{"ref":{"providerId":null,"key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`), field: "ref"},
		{name: "provider 型", arguments: json.RawMessage(`{"ref":{"providerId":1,"key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`), field: "ref"},
		{name: "key null", arguments: json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":null}}`), field: "ref"},
		{name: "key 未知項目", arguments: json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2","unknown":true}}}`), field: "ref"},
		{name: "source null", arguments: json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":null,"resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`), field: "ref"},
		{name: "source 型", arguments: json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":1,"resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`), field: "ref"},
		{name: "resource type 型", arguments: json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":1,"resourceId":"95570/detail2"}}}`), field: "ref"},
		{name: "resource ID 型", arguments: json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":95570}}}`), field: "ref"},
		{name: "provider 不一致", arguments: replaceTraceInput(t, `courts-hanrei-html`, `other-provider`), field: "ref"},
		{name: "source 不一致", arguments: replaceTraceInput(t, `courts-hanrei","resourceType`, `other-source","resourceType`), field: "ref"},
		{name: "resource type 不一致", arguments: replaceTraceInput(t, `judicial-decision","resourceId`, `law","resourceId`), field: "ref"},
		{name: "canonical resource ID でない", arguments: replaceTraceInput(t, `95570/detail2`, `case-2024-001`), field: "ref"},
		{name: "detail 範囲外", arguments: replaceTraceInput(t, `95570/detail2`, `95570/detail9`), field: "ref"},
		{name: "versionId", arguments: json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2","versionId":"v1"}}}`), field: "ref"},
		{name: "空 versionId", arguments: json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2","versionId":""}}}`), field: "ref"},
		{name: "direction null", arguments: withTraceInputSuffix(`,"direction":null`), field: "direction"},
		{name: "direction 型", arguments: withTraceInputSuffix(`,"direction":1`), field: "direction"},
		{name: "direction 不正 surrogate", arguments: withTraceInputSuffix(`,"direction":"\ud800"`), field: "direction"},
		{name: "direction 列挙外", arguments: withTraceInputSuffix(`,"direction":"sideways"`), field: "direction"},
		{name: "incomingLimit null", arguments: withTraceInputSuffix(`,"incomingLimit":null`), field: "incomingLimit"},
		{name: "incomingLimit 型", arguments: withTraceInputSuffix(`,"incomingLimit":"5"`), field: "incomingLimit"},
		{name: "incomingLimit 真偽値", arguments: withTraceInputSuffix(`,"incomingLimit":true`), field: "incomingLimit"},
		{name: "incomingLimit 小数", arguments: withTraceInputSuffix(`,"incomingLimit":1.5`), field: "incomingLimit"},
		{name: "incomingLimit 下限", arguments: withTraceInputSuffix(`,"incomingLimit":0`), field: "incomingLimit"},
		{name: "incomingLimit 上限", arguments: withTraceInputSuffix(`,"incomingLimit":11`), field: "incomingLimit"},
		{name: "不正 UTF-8", arguments: invalidUTF8, field: "ref"},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			port := &stubTraceJudicialCitationsPort{result: mustJudicialCitationGraphResult(t)}
			result, err := callTraceJudicialCitations(
				context.Background(),
				port,
				testCase.arguments,
			)
			if err != nil {
				t.Fatalf("callTraceJudicialCitations() error = %v", err)
			}
			payload := assertJudicialSearchErrorCode(t, result, "invalid_argument")
			if payload.Details["field"] != testCase.field {
				t.Fatalf("details = %#v, want field %q", payload.Details, testCase.field)
			}
			if port.calls != 0 {
				t.Fatalf("不正入力で port を %d 回呼び出しました", port.calls)
			}
		})
	}
}

func TestCallTraceJudicialCitationsReturnsSafeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		port judicialcitationtrace.Port
	}{
		{name: "nil port", port: nil},
		{
			name: "internal failure",
			port: &stubTraceJudicialCitationsPort{
				err: errors.New("秘密の検索語 95570/detail2"),
			},
		},
		{
			name: "invalid application result",
			port: &stubTraceJudicialCitationsPort{
				result: model.JudicialCitationGraphResult{},
			},
		},
		{
			name: "request と coverage の不一致",
			port: &stubTraceJudicialCitationsPort{
				result: mustJudicialCitationGraphResultForDirection(
					t,
					model.JudicialCitationRequestedDirectionOutgoing,
				),
			},
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := callTraceJudicialCitations(
				context.Background(),
				testCase.port,
				canonicalTraceJudicialCitationsInput(),
			)
			if err != nil {
				t.Fatalf("callTraceJudicialCitations() error = %v", err)
			}
			assertJudicialSearchErrorCode(t, result, "internal_error")
			if text := firstToolText(t, result); strings.Contains(text, "秘密") ||
				strings.Contains(text, "95570/detail2") {
				t.Fatalf("error text が原文を公開しました: %q", text)
			}
		})
	}
}

func TestSanitizeTraceJudicialCitationsOperationResultIsFailClosed(t *testing.T) {
	t.Parallel()

	base := map[string]any{
		"providerId":   traceJudicialCitationsProviderID,
		"sourceId":     traceJudicialCitationsSourceID,
		"capabilityId": "judicial-decision.read",
	}
	with := func(values map[string]any) map[string]any {
		cloned := make(map[string]any, len(base)+len(values))
		for key, value := range base {
			cloned[key] = value
		}
		for key, value := range values {
			cloned[key] = value
		}
		return cloned
	}
	tests := []struct {
		name    string
		code    model.ErrorCode
		details map[string]any
		want    bool
	}{
		{name: "not found", code: model.ErrorCodeNotFound, want: true},
		{
			name:    "safe unsupported capability",
			code:    model.ErrorCodeUnsupportedCapability,
			details: with(nil),
			want:    true,
		},
		{
			name: string(model.ErrorCodeSourceTimeout),
			code: model.ErrorCodeSourceTimeout,
			details: with(map[string]any{
				"operation": traceJudicialCitationsReadOperation,
			}),
			want: true,
		},
		{
			name: "safe retry after",
			code: model.ErrorCodeRateLimited,
			details: with(map[string]any{
				"operation":  traceJudicialCitationsReadOperation,
				"retryAfter": "3",
			}),
			want: true,
		},
		{
			name: "invalid argument from application",
			code: model.ErrorCodeInvalidArgument,
			details: map[string]any{
				"field":  "ref",
				"reason": "秘密の入力 95570/detail2",
			},
		},
		{
			name: "unlisted error code",
			code: model.ErrorCodeAmbiguousLocation,
			details: map[string]any{
				"candidates": []any{"秘密"},
			},
		},
		{
			name: "unknown operation",
			code: model.ErrorCodeSourceTimeout,
			details: with(map[string]any{
				"operation": "秘密の検索語",
			}),
		},
		{
			name: "unsafe retry after",
			code: model.ErrorCodeRateLimited,
			details: with(map[string]any{
				"operation":  traceJudicialCitationsReadOperation,
				"retryAfter": "秘密の待機値",
			}),
		},
		{
			name: "extra details",
			code: model.ErrorCodeUnsupportedQuery,
			details: with(map[string]any{
				"field":      "ref",
				"constraint": "秘密の検索語",
			}),
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			input, err := model.NewErrorResult(model.ErrorResultValues{
				Code:    testCase.code,
				Details: testCase.details,
			})
			if err != nil {
				t.Fatalf("test ErrorResult = %v", err)
			}
			got, safe := sanitizeTraceJudicialCitationsOperationResult(input)
			if safe != testCase.want {
				t.Fatalf("safe = %t, want %t", safe, testCase.want)
			}
			if safe && (got.Validate() != nil || got.Code() != testCase.code) {
				t.Fatalf("sanitized result = %#v", got)
			}
			if safe {
				gotDetails, gotHasDetails := got.Details()
				wantDetails, wantHasDetails := input.Details()
				if gotHasDetails != wantHasDetails ||
					!reflect.DeepEqual(gotDetails, wantDetails) {
					t.Fatalf(
						"sanitized details = %#v, want %#v",
						gotDetails,
						wantDetails,
					)
				}
			}
		})
	}
}

func TestTraceJudicialCitationsSchemasAreClosedAndValidateResult(t *testing.T) {
	t.Parallel()

	input := newTraceJudicialCitationsInputSchema()
	assertTraceSchemaRejects(t, input, map[string]any{
		"ref": map[string]any{
			"providerId": "courts-hanrei-html",
			"key": map[string]any{
				"sourceId":     "courts-hanrei",
				"resourceType": "judicial-decision",
				"resourceId":   "95570/detail2",
			},
		},
		"maxDepth": 1,
	})
	assertTraceSchemaAccepts(t, input, map[string]any{
		"ref": map[string]any{
			"providerId": "courts-hanrei-html",
			"key": map[string]any{
				"sourceId":     "courts-hanrei",
				"resourceType": "judicial-decision",
				"resourceId":   "95570/detail2",
			},
		},
		"direction":     "both",
		"incomingLimit": 5,
	})

	output := newTraceJudicialCitationsOutputSchema()
	payload, err := json.Marshal(mustJudicialCitationGraphResult(t))
	if err != nil {
		t.Fatalf("result marshal = %v", err)
	}
	var instance any
	if err := json.Unmarshal(payload, &instance); err != nil {
		t.Fatalf("result decode = %v", err)
	}
	assertTraceSchemaAccepts(t, output, instance)
	expanded := cloneTraceJSONValue(t, payload)
	appendTraceUnresolvedMentions(t, expanded, 257)
	assertTraceSchemaAccepts(t, output, expanded)
	assertNoOpaqueTraceObjectSchema(t, output, "output", map[*jsonschema.Schema]bool{})

	invalidResults := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{
			name: "空 node",
			mutate: func(value map[string]any) {
				value["graph"].(map[string]any)["nodes"] = []any{map[string]any{}}
			},
		},
		{
			name: "空 edge",
			mutate: func(value map[string]any) {
				value["graph"].(map[string]any)["edges"] = []any{map[string]any{}}
			},
		},
		{
			name: "空 unresolved mention",
			mutate: func(value map[string]any) {
				value["graph"].(map[string]any)["unresolvedMentions"] = []any{map[string]any{}}
			},
		},
		{
			name: "空 issue",
			mutate: func(value map[string]any) {
				value["issues"] = []any{map[string]any{}}
			},
		},
		{
			name: "非採用 impactScore",
			mutate: func(value map[string]any) {
				value["graph"].(map[string]any)["impactScore"] = 1
			},
		},
	}
	for _, testCase := range invalidResults {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cloned := cloneTraceJSONValue(t, payload)
			testCase.mutate(cloned)
			assertTraceSchemaRejects(t, output, cloned)
		})
	}
}

type stubTraceJudicialCitationsPort struct {
	result  model.JudicialCitationGraphResult
	err     error
	calls   int
	request judicialcitationtrace.Request
}

func (p *stubTraceJudicialCitationsPort) Trace(
	_ context.Context,
	request judicialcitationtrace.Request,
) (model.JudicialCitationGraphResult, error) {
	p.calls++
	p.request = request
	return p.result, p.err
}

func mustJudicialCitationGraphResult(t *testing.T) model.JudicialCitationGraphResult {
	t.Helper()
	return mustJudicialCitationGraphResultForDirection(
		t,
		model.JudicialCitationRequestedDirectionBoth,
	)
}

func mustJudicialCitationGraphResultForDirection(
	t *testing.T,
	direction model.JudicialCitationRequestedDirection,
) model.JudicialCitationGraphResult {
	t.Helper()
	root := mustJudicialDecisionDetailsResource()
	ref := root.Ref()
	summary := root.Data().Summary()
	rootNode, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:          "node-1",
		NodeType:        model.JudicialCitationNodeTypeDecision,
		Label:           summary.CaseNumber(),
		Ref:             &ref,
		DecisionSummary: &summary,
	})
	if err != nil {
		t.Fatalf("root node = %v", err)
	}
	citedText := "最高裁判所令和3年(受)第100号判決"
	citedNode, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:        "node-2",
		NodeType:      model.JudicialCitationNodeTypeDecisionReference,
		Label:         citedText,
		ReferenceText: &citedText,
	})
	if err != nil {
		t.Fatalf("cited node = %v", err)
	}
	location, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: "709",
	})
	if err != nil {
		t.Fatalf("law location = %v", err)
	}
	lawReference, err := model.NewJudicialCitationLawReference(
		model.JudicialCitationLawReferenceValues{
			LawID:    "129AC0000000089",
			LawTitle: "民法",
			Location: location,
		},
	)
	if err != nil {
		t.Fatalf("law reference = %v", err)
	}
	lawNode, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:       "node-3",
		NodeType:     model.JudicialCitationNodeTypeLawProvision,
		Label:        "民法第709条",
		LawReference: &lawReference,
	})
	if err != nil {
		t.Fatalf("law node = %v", err)
	}
	lowerText := "東京高等裁判所令和2年(ネ)第200号判決"
	lowerNode, err := model.NewJudicialCitationNode(model.JudicialCitationNodeValues{
		NodeID:        "node-4",
		NodeType:      model.JudicialCitationNodeTypeDecisionReference,
		Label:         lowerText,
		ReferenceText: &lowerText,
	})
	if err != nil {
		t.Fatalf("lower node = %v", err)
	}
	provenance := root.Provenance()[0]
	excerpt := "同判決を参照する。"
	exactEvidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelExactTextMatch,
		Provenance:    provenance,
		Excerpt:       &excerpt,
	})
	if err != nil {
		t.Fatalf("exact evidence = %v", err)
	}
	metadataEvidence, err := model.NewJudicialCitationEvidence(model.JudicialCitationEvidenceValues{
		EvidenceLevel: model.JudicialCitationEvidenceLevelOfficialMetadata,
		Provenance:    provenance,
	})
	if err != nil {
		t.Fatalf("metadata evidence = %v", err)
	}
	edges := make([]model.JudicialCitationEdge, 0, 3)
	for _, values := range []model.JudicialCitationEdgeValues{
		{
			EdgeID:       "edge-1",
			FromNodeID:   "node-1",
			ToNodeID:     "node-2",
			RelationType: model.JudicialCitationRelationTypeCitesDecision,
			Evidence:     []model.JudicialCitationEvidence{exactEvidence},
		},
		{
			EdgeID:       "edge-2",
			FromNodeID:   "node-1",
			ToNodeID:     "node-3",
			RelationType: model.JudicialCitationRelationTypeReferencesLawProvision,
			Evidence:     []model.JudicialCitationEvidence{metadataEvidence},
		},
		{
			EdgeID:       "edge-3",
			FromNodeID:   "node-1",
			ToNodeID:     "node-4",
			RelationType: model.JudicialCitationRelationTypeHasLowerCourtDecision,
			Evidence:     []model.JudicialCitationEvidence{metadataEvidence},
		},
	} {
		edge, edgeErr := model.NewJudicialCitationEdge(values)
		if edgeErr != nil {
			t.Fatalf("edge = %v", edgeErr)
		}
		edges = append(edges, edge)
	}
	unresolved, err := model.NewJudicialCitationUnresolvedMention(
		model.JudicialCitationUnresolvedMentionValues{
			MentionType: model.JudicialCitationMentionTypeDecision,
			MentionText: "同年の最高裁判決",
			Reason:      model.JudicialCitationUnresolvedReasonInsufficientIdentity,
			Provenance:  provenance,
		},
	)
	if err != nil {
		t.Fatalf("unresolved mention = %v", err)
	}
	outgoing, err := model.NewJudicialCitationDirectionCoverage(model.JudicialCitationDirectionCoverageValues{
		Status: model.JudicialCitationDirectionStatusComplete,
		Methods: []model.JudicialCitationMethod{
			model.JudicialCitationMethodOfficialDetailMetadata,
			model.JudicialCitationMethodOfficialPDFText,
		},
	})
	if err != nil {
		t.Fatalf("outgoing coverage = %v", err)
	}
	incomingValues := model.JudicialCitationDirectionCoverageValues{
		Status:  model.JudicialCitationDirectionStatusNotRequested,
		Methods: []model.JudicialCitationMethod{},
	}
	if direction != model.JudicialCitationRequestedDirectionOutgoing {
		limit := judicialcitationtrace.DefaultIncomingLimit
		attempted := 1
		completed := 1
		incomingValues = model.JudicialCitationDirectionCoverageValues{
			Status: model.JudicialCitationDirectionStatusComplete,
			Methods: []model.JudicialCitationMethod{
				model.JudicialCitationMethodOfficialDetailMetadata,
				model.JudicialCitationMethodOfficialCaseSearch,
			},
			Limit:             &limit,
			AttemptedSearches: &attempted,
			CompletedSearches: &completed,
		}
	}
	incoming, err := model.NewJudicialCitationDirectionCoverage(incomingValues)
	if err != nil {
		t.Fatalf("incoming coverage = %v", err)
	}
	coverage, err := model.NewJudicialCitationCoverage(model.JudicialCitationCoverageValues{
		RequestedDirection: direction,
		HopDepth:           1,
		Outgoing:           outgoing,
		Incoming:           incoming,
	})
	if err != nil {
		t.Fatalf("coverage = %v", err)
	}
	graphSummary, err := model.NewJudicialCitationSummary(model.JudicialCitationSummaryValues{
		ConfirmedOutgoingDecisionCount:  1,
		ReferencedProvisionCount:        1,
		LowerCourtRelationCount:         1,
		UnresolvedMentionCount:          1,
		IncomingObservedYearBuckets:     []model.JudicialCitationYearBucket{},
		IncomingObservedCategoryBuckets: []model.JudicialCitationCategoryBucket{},
	})
	if err != nil {
		t.Fatalf("summary = %v", err)
	}
	graph, err := model.NewJudicialCitationGraph(model.JudicialCitationGraphValues{
		RootNodeID: "node-1",
		Nodes: []model.JudicialCitationNode{
			rootNode,
			citedNode,
			lawNode,
			lowerNode,
		},
		Edges:              edges,
		UnresolvedMentions: []model.JudicialCitationUnresolvedMention{unresolved},
		Summary:            graphSummary,
		Coverage:           coverage,
	})
	if err != nil {
		t.Fatalf("graph = %v", err)
	}
	result, err := model.NewJudicialCitationGraphResult(model.JudicialCitationGraphResultValues{
		Status:         model.JudicialCitationResultStatusComplete,
		CoverageNotice: judicialcitationtrace.CoverageNotice,
		Graph:          graph,
		Issues:         []model.JudicialCitationIssue{},
	})
	if err != nil {
		t.Fatalf("result = %v", err)
	}
	return result
}

func canonicalTraceJudicialCitationsInput() json.RawMessage {
	return json.RawMessage(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`)
}

func mustJudicialCitationCasesDependencies(t *testing.T) JudicialCasesDependencies {
	t.Helper()
	dependencies, err := NewJudicialCasesDependencies(
		stubSearchJudicialCasesPort{},
		&recordingGetJudicialCasePort{item: mustJudicialDecisionDetailsResource()},
	)
	if err != nil {
		t.Fatalf("judicial-cases dependencies = %v", err)
	}
	return dependencies
}

func withTraceInputSuffix(suffix string) json.RawMessage {
	base := string(canonicalTraceJudicialCitationsInput())
	return json.RawMessage(base[:len(base)-1] + suffix + `}`)
}

func replaceTraceInput(t *testing.T, old, replacement string) json.RawMessage {
	t.Helper()
	value := strings.Replace(string(canonicalTraceJudicialCitationsInput()), old, replacement, 1)
	if value == string(canonicalTraceJudicialCitationsInput()) {
		t.Fatalf("test input replacement %q was not applied", old)
	}
	return json.RawMessage(value)
}

func firstToolText(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	text, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("content[0] = %#v", result.Content[0])
	}
	return text.Text
}

func jsonValuesEqual(t *testing.T, left, right []byte) bool {
	t.Helper()
	var leftValue, rightValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		t.Fatalf("left JSON = %v", err)
	}
	if err := json.Unmarshal(right, &rightValue); err != nil {
		t.Fatalf("right JSON = %v", err)
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func cloneTraceJSONValue(t *testing.T, payload []byte) map[string]any {
	t.Helper()
	var cloned map[string]any
	if err := json.Unmarshal(payload, &cloned); err != nil {
		t.Fatalf("JSON clone = %v", err)
	}
	return cloned
}

func appendTraceUnresolvedMentions(t *testing.T, payload map[string]any, count int) {
	t.Helper()
	graph, ok := payload["graph"].(map[string]any)
	if !ok {
		t.Fatal("graph が object ではありません")
	}
	mentions, ok := graph["unresolvedMentions"].([]any)
	if !ok {
		t.Fatal("unresolvedMentions が配列ではありません")
	}
	base, ok := mentions[0].(map[string]any)
	if !ok {
		t.Fatal("基準 unresolvedMention が object ではありません")
	}
	expanded := make([]any, 0, count)
	for index := 0; index < count; index++ {
		cloned := make(map[string]any, len(base))
		for key, value := range base {
			cloned[key] = value
		}
		cloned["mentionText"] = fmt.Sprintf("未解決言及-%03d", index+1)
		expanded = append(expanded, cloned)
	}
	graph["unresolvedMentions"] = expanded
	summary, ok := graph["summary"].(map[string]any)
	if !ok {
		t.Fatal("summary が object ではありません")
	}
	summary["unresolvedMentionCount"] = float64(count)
}

type traceJudicialCitationsSchemaSnapshot struct {
	input  json.RawMessage
	output json.RawMessage
}

func traceJudicialCitationsSchemaFromServer(
	t *testing.T,
	server *sdk.Server,
) traceJudicialCitationsSchemaSnapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- server.Run(ctx, serverTransport)
	}()
	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("schema MCP connect = %v", err)
	}
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("schema tools/list = %v", err)
	}
	snapshot := traceJudicialCitationsSchemaFromTools(t, tools.Tools)
	if err := session.Close(); err != nil {
		t.Fatalf("schema MCP close = %v", err)
	}
	select {
	case err := <-serverResult:
		if err != nil {
			t.Fatalf("schema MCP server = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("schema MCP server close timeout = %v", ctx.Err())
	}
	return snapshot
}

func traceJudicialCitationsSchemaFromTools(
	t *testing.T,
	tools []*sdk.Tool,
) traceJudicialCitationsSchemaSnapshot {
	t.Helper()
	var target *sdk.Tool
	for _, tool := range tools {
		if tool.Name == "trace_judicial_citations" {
			target = tool
			break
		}
	}
	if target == nil {
		t.Fatal("trace_judicial_citations が登録されていません")
	}
	input, err := json.Marshal(target.InputSchema)
	if err != nil {
		t.Fatalf("input schema marshal = %v", err)
	}
	output, err := json.Marshal(target.OutputSchema)
	if err != nil {
		t.Fatalf("output schema marshal = %v", err)
	}
	return traceJudicialCitationsSchemaSnapshot{input: input, output: output}
}

func assertTraceSchemaAccepts(t *testing.T, schema *jsonschema.Schema, instance any) {
	t.Helper()
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("schema.Resolve() error = %v", err)
	}
	if err := resolved.Validate(instance); err != nil {
		t.Fatalf("schema rejected valid instance: %v", err)
	}
}

func assertTraceSchemaRejects(t *testing.T, schema *jsonschema.Schema, instance any) {
	t.Helper()
	resolved, err := schema.Resolve(nil)
	if err != nil {
		t.Fatalf("schema.Resolve() error = %v", err)
	}
	if err := resolved.Validate(instance); err == nil {
		t.Fatal("schema accepted invalid instance")
	}
}

func assertNoOpaqueTraceObjectSchema(
	t *testing.T,
	schema *jsonschema.Schema,
	path string,
	visited map[*jsonschema.Schema]bool,
) {
	t.Helper()
	if schema == nil || visited[schema] {
		return
	}
	visited[schema] = true
	if schema.Type == "object" && schema.Ref == "" && len(schema.OneOf) == 0 &&
		len(schema.Properties) == 0 {
		t.Fatalf("%s is an opaque object schema", path)
	}
	for name, child := range schema.Properties {
		assertNoOpaqueTraceObjectSchema(t, child, path+"."+name, visited)
	}
	for _, child := range schema.OneOf {
		assertNoOpaqueTraceObjectSchema(t, child, path+".oneOf", visited)
	}
	assertNoOpaqueTraceObjectSchema(t, schema.Items, path+"[]", visited)
	for name, child := range schema.Defs {
		assertNoOpaqueTraceObjectSchema(t, child, path+".$defs."+name, visited)
	}
}
