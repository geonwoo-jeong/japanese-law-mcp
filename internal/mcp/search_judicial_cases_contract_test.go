package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSearchJudicialCasesAcceptsExactlyIntegralJSONNumbers(t *testing.T) {
	t.Parallel()

	for name, arguments := range map[string]json.RawMessage{
		"integer":          json.RawMessage(`{"query":"民法","limit":20}`),
		"decimal integer":  json.RawMessage(`{"query":"民法","limit":20.0}`),
		"exponent integer": json.RawMessage(`{"query":"民法","limit":2e1}`),
	} {
		name, arguments := name, arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			port := &recordingJudicialSearchPort{page: mustJudicialSearchPage()}
			result, err := callSearchJudicialCases(
				context.Background(),
				port,
				arguments,
			)
			if err != nil {
				t.Fatalf("SOT-IF-047: callSearchJudicialCases() のエラー = %v", err)
			}
			if result.IsError {
				t.Fatalf("SOT-IF-047: 正確な整数を拒否した: %#v", result)
			}
			if port.calls != 1 || port.request.Limit() != 20 {
				t.Fatalf(
					"SOT-IF-047: port 呼出し = %d, limit = %d",
					port.calls,
					port.request.Limit(),
				)
			}
		})
	}
}

func TestSearchJudicialCasesNormalizesRequestAtPublicBoundary(t *testing.T) {
	t.Parallel()

	port := &recordingJudicialSearchPort{page: mustJudicialSearchPage()}
	result, err := callSearchJudicialCases(
		context.Background(),
		port,
		json.RawMessage(
			`{"query":"　民  法　","continuationToken":"resume-cursor"}`,
		),
	)
	if err != nil || result.IsError {
		t.Fatalf("SOT-IF-047: 有効な入力を拒否した: result=%#v, err=%v", result, err)
	}
	continuationValue, exists := port.request.ContinuationToken()
	if port.calls != 1 ||
		port.request.Query() != "民  法" ||
		port.request.Limit() != judicialdecisionsearch.DefaultLimit ||
		!exists ||
		continuationValue != "resume-cursor" {
		t.Fatalf("SOT-IF-041/047: request = %#v", port.request)
	}
}

func TestSearchJudicialCasesAcceptsValidSurrogatePair(t *testing.T) {
	t.Parallel()

	port := &recordingJudicialSearchPort{page: mustJudicialSearchPage()}
	result, err := callSearchJudicialCases(
		context.Background(),
		port,
		json.RawMessage(`{"query":"\ud83d\ude00"}`),
	)
	if err != nil || result.IsError || port.calls != 1 || port.request.Query() != "😀" {
		t.Fatalf(
			"SOT-IF-041/047: 有効な surrogate pair を拒否した: request=%#v, result=%#v, err=%v",
			port.request,
			result,
			err,
		)
	}
}

func TestSearchJudicialCasesRejectsInvalidInputBeforePort(t *testing.T) {
	t.Parallel()

	tooLongQuery := strings.Repeat("法", 171)
	maxQuery := strings.Repeat("法", 170) + "ab"
	tooLongToken := strings.Repeat(
		"x",
		judicialdecisionsearch.MaxTokenBytes+1,
	)
	invalidUTF8 := append(
		[]byte(`{"query":"`),
		append([]byte{0xff}, []byte(`"}`)...)...,
	)
	tests := []struct {
		name      string
		arguments json.RawMessage
	}{
		{name: "入力なし", arguments: nil},
		{name: "null object", arguments: json.RawMessage(`null`)},
		{name: "配列", arguments: json.RawMessage(`[]`)},
		{name: "文字列", arguments: json.RawMessage(`"民法"`)},
		{name: "不正 JSON", arguments: json.RawMessage(`{"query":`)},
		{name: "query 欠落", arguments: json.RawMessage(`{}`)},
		{name: "query null", arguments: json.RawMessage(`{"query":null}`)},
		{name: "query 型", arguments: json.RawMessage(`{"query":1}`)},
		{name: "query 空", arguments: json.RawMessage(`{"query":""}`)},
		{name: "query 制御文字", arguments: json.RawMessage(`{"query":"民\u0000法"}`)},
		{name: "query 単独 surrogate", arguments: json.RawMessage(`{"query":"\ud800"}`)},
		{
			name:      "query 512 byte 超",
			arguments: json.RawMessage(fmt.Sprintf(`{"query":%q}`, tooLongQuery)),
		},
		{name: "query 不正 UTF-8", arguments: json.RawMessage(invalidUTF8)},
		{name: "limit null", arguments: json.RawMessage(`{"query":"民法","limit":null}`)},
		{name: "limit 文字列", arguments: json.RawMessage(`{"query":"民法","limit":"20"}`)},
		{name: "limit 非整数", arguments: json.RawMessage(`{"query":"民法","limit":20.5}`)},
		{
			name:      "limit 指数過大",
			arguments: json.RawMessage(`{"query":"民法","limit":1e9223372036854775807}`),
		},
		{name: "limit 下限未満", arguments: json.RawMessage(`{"query":"民法","limit":0}`)},
		{name: "limit 上限超", arguments: json.RawMessage(`{"query":"民法","limit":31}`)},
		{
			name:      "continuationToken null",
			arguments: json.RawMessage(`{"query":"民法","continuationToken":null}`),
		},
		{
			name:      "continuationToken 型",
			arguments: json.RawMessage(`{"query":"民法","continuationToken":1}`),
		},
		{
			name:      "continuationToken 単独 surrogate",
			arguments: json.RawMessage(`{"query":"民法","continuationToken":"\udfff"}`),
		},
		{
			name: "continuationToken 上限超",
			arguments: json.RawMessage(
				fmt.Sprintf(
					`{"query":"民法","continuationToken":%q}`,
					tooLongToken,
				),
			),
		},
		{
			name:      "未知項目",
			arguments: json.RawMessage(`{"query":"民法","unknown":true}`),
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			port := &recordingJudicialSearchPort{}
			result, err := callSearchJudicialCases(
				context.Background(),
				port,
				testCase.arguments,
			)
			if err != nil {
				t.Fatalf("callSearchJudicialCases() のエラー = %v", err)
			}
			assertJudicialSearchErrorCode(t, result, "invalid_argument")
			if port.calls != 0 {
				t.Fatalf("SOT-IF-047: 不正入力で port を %d 回呼び出した", port.calls)
			}
		})
	}

	port := &recordingJudicialSearchPort{page: mustJudicialSearchPage()}
	result, err := callSearchJudicialCases(
		context.Background(),
		port,
		json.RawMessage(fmt.Sprintf(`{"query":%q,"limit":30}`, maxQuery)),
	)
	if err != nil || result.IsError || port.calls != 1 {
		t.Fatalf("SOT-IF-041/047: 境界値を拒否した: result=%#v, err=%v", result, err)
	}
}

func TestSearchJudicialCasesMapsArgumentAndSourceErrors(t *testing.T) {
	t.Parallel()

	argumentError, err := judicialdecisionsearch.NewArgumentError(
		"continuationToken",
		"このプロバイダーでは使用できません",
	)
	if err != nil {
		t.Fatalf("試験用 ArgumentError を作成できません: %v", err)
	}
	result, err := callSearchJudicialCases(
		context.Background(),
		&recordingJudicialSearchPort{
			err: fmt.Errorf("provider validation: %w", argumentError),
		},
		json.RawMessage(`{"query":"民法","continuationToken":"opaque"}`),
	)
	if err != nil {
		t.Fatalf("callSearchJudicialCases() のエラー = %v", err)
	}
	payload := assertJudicialSearchErrorCode(t, result, "invalid_argument")
	if payload.Details["field"] != "continuationToken" ||
		payload.Details["reason"] != "このプロバイダーでは使用できません" {
		t.Fatalf("SOT-IF-027/047: details = %#v", payload.Details)
	}

	sourceCodes := []model.SourceErrorCode{
		model.SourceErrorCodeUnsupportedCapability,
		model.SourceErrorCodeUnsupportedQuery,
		model.SourceErrorCodeConfigurationRequired,
		model.SourceErrorCodeSourceAuthFailed,
		model.SourceErrorCodeRateLimited,
		model.SourceErrorCodeSourceTimeout,
		model.SourceErrorCodeSourceUnavailable,
		model.SourceErrorCodeSourceBusy,
		model.SourceErrorCodeSourceContractChanged,
		model.SourceErrorCodeInvalidSourceResponse,
		model.SourceErrorCodeSourceResponseTooLarge,
		model.SourceErrorCodeSourceProcessingLimit,
		model.SourceErrorCodeUnsafeSourceContent,
	}
	for _, code := range sourceCodes {
		code := code
		t.Run(string(code), func(t *testing.T) {
			t.Parallel()

			result, callErr := callSearchJudicialCases(
				context.Background(),
				&recordingJudicialSearchPort{
					err: mustJudicialSearchSourceError(code),
				},
				json.RawMessage(`{"query":"民法"}`),
			)
			if callErr != nil {
				t.Fatalf("callSearchJudicialCases() のエラー = %v", callErr)
			}
			got := assertJudicialSearchErrorCode(t, result, string(code))
			wantRetryable := code == model.SourceErrorCodeRateLimited ||
				code == model.SourceErrorCodeSourceTimeout ||
				code == model.SourceErrorCodeSourceUnavailable ||
				code == model.SourceErrorCodeSourceBusy
			if got.Retryable != wantRetryable {
				t.Fatalf("SOT-IF-027: retryable = %t, want %t", got.Retryable, wantRetryable)
			}
		})
	}
}

func TestSearchJudicialCasesRejectsInvalidPortResult(t *testing.T) {
	t.Parallel()

	var typedNil *recordingJudicialSearchPort
	for name, port := range map[string]judicialdecisionsearch.Port{
		"nil":       nil,
		"typed nil": typedNil,
		"unknown error": &recordingJudicialSearchPort{
			err: errors.New("非公開の内部エラー"),
		},
		"invalid page": &recordingJudicialSearchPort{},
	} {
		name, port := name, port
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callSearchJudicialCases(
				context.Background(),
				port,
				json.RawMessage(`{"query":"秘密検索語"}`),
			)
			if err != nil {
				t.Fatalf("callSearchJudicialCases() のエラー = %v", err)
			}
			assertJudicialSearchErrorCode(t, result, "internal_error")
			if strings.Contains(
				result.Content[0].(*sdk.TextContent).Text,
				"秘密検索語",
			) {
				t.Fatal("SOT-IF-027: ErrorResult に検索語が含まれています")
			}
		})
	}
}

func TestSearchJudicialCasesToolPublishesBoundedSchemas(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	server := NewServer("test-version")
	addSearchJudicialCasesTool(server, &recordingJudicialSearchPort{
		page: mustJudicialSearchPage(),
	})
	go func() {
		serverResult <- server.Run(ctx, serverTransport)
	}()

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("MCP セッションを初期化できません: %v", err)
	}
	defer func() { _ = session.Close() }()
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("tools/list のエラー = %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "search_judicial_cases" {
		t.Fatalf("SOT-IF-047: tools = %#v", tools.Tools)
	}
	tool := tools.Tools[0]
	if tool.InputSchema == nil || tool.OutputSchema == nil {
		t.Fatal("SOT-IF-007/047: inputSchema または outputSchema がありません")
	}
	inputSchema := decodeJudicialSearchSchema(t, tool.InputSchema)
	outputSchema := decodeJudicialSearchSchema(t, tool.OutputSchema)
	limit := inputSchema.Properties["limit"]
	if limit == nil ||
		limit.Minimum == nil ||
		*limit.Minimum != 1 ||
		limit.Maximum == nil ||
		*limit.Maximum != 30 ||
		string(limit.Default) != "20" {
		t.Fatalf("SOT-IF-047: limit schema = %#v", limit)
	}
	query := inputSchema.Properties["query"]
	if query == nil || query.MinLength == nil || *query.MinLength != 1 {
		t.Fatalf("SOT-IF-041/047: query schema = %#v", query)
	}
	assertUTF8ByteLimitSchema(
		t,
		query,
		judicialdecisionsearch.MaxQueryBytes,
	)
	token := inputSchema.Properties["continuationToken"]
	assertUTF8ByteLimitSchema(
		t,
		token,
		judicialdecisionsearch.MaxTokenBytes,
	)
	if !containsString(inputSchema.Required, "query") ||
		!containsString(outputSchema.Required, "coverageNotice") ||
		!containsString(outputSchema.Required, "items") ||
		!containsString(outputSchema.Required, "page") {
		t.Fatalf(
			"SOT-IF-007/047: required = input %#v, output %#v",
			inputSchema.Required,
			outputSchema.Required,
		)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("MCP セッションを終了できません: %v", err)
	}
	select {
	case runErr := <-serverResult:
		if runErr != nil {
			t.Fatalf("MCP サーバーが正常終了しませんでした: %v", runErr)
		}
	case <-ctx.Done():
		t.Fatalf("MCP サーバーの終了を待機できません: %v", ctx.Err())
	}
}

type recordingJudicialSearchPort struct {
	page    judicialdecisionsearch.Page
	err     error
	request judicialdecisionsearch.Request
	calls   int
}

func (p *recordingJudicialSearchPort) Search(
	_ context.Context,
	request judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	p.calls++
	p.request = request
	return p.page, p.err
}

type judicialSearchErrorPayload struct {
	Code      string         `json:"code"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details"`
}

func assertJudicialSearchErrorCode(
	t *testing.T,
	result *sdk.CallToolResult,
	want string,
) judicialSearchErrorPayload {
	t.Helper()
	if result == nil || !result.IsError || result.StructuredContent != nil {
		t.Fatalf("SOT-IF-007: tool error = %#v", result)
	}
	var payload judicialSearchErrorPayload
	if err := json.Unmarshal(
		[]byte(result.Content[0].(*sdk.TextContent).Text),
		&payload,
	); err != nil {
		t.Fatalf("ErrorResult を解析できません: %v", err)
	}
	if payload.Code != want {
		t.Fatalf("ErrorResult.code = %q, want %q", payload.Code, want)
	}
	return payload
}

func mustJudicialSearchSourceError(code model.SourceErrorCode) error {
	provider := hanreiTestProviderDescriptor()
	capability := provider.Capabilities()[1]
	if code == model.SourceErrorCodeUnsupportedCapability {
		provider = hanreiReadOnlyTestProviderDescriptor()
		capability = mustJudicialSearchCapability()
	}
	return mustSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   provider,
		Capability: capability,
		Operation:  hanreiTestOperation("search_html"),
	})
}

func hanreiReadOnlyTestProviderDescriptor() model.ProviderDescriptor {
	full := hanreiTestProviderDescriptor()
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             full.ProviderID(),
		Source:                 full.Source(),
		AdapterContractVersion: full.AdapterContractVersion(),
		UpstreamSpecVersion:    "2026-07-27",
		VerifiedAt:             full.VerifiedAt(),
		InterfaceType:          full.InterfaceType(),
		CredentialRequired:     false,
		Capabilities:           full.Capabilities()[:1],
	})
	if err != nil {
		panic(err)
	}
	return descriptor
}

func mustJudicialSearchCapability() model.ProviderCapability {
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           judicialdecisionsearch.CapabilityID,
		MajorVersion: judicialdecisionsearch.MajorVersion,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(err)
	}
	return capability
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func decodeJudicialSearchSchema(
	t *testing.T,
	value any,
) jsonschema.Schema {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("schema を JSON に変換できません: %v", err)
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(encoded, &schema); err != nil {
		t.Fatalf("schema を解析できません: %v", err)
	}
	return schema
}

func assertUTF8ByteLimitSchema(
	t *testing.T,
	schema *jsonschema.Schema,
	want int,
) {
	t.Helper()
	if schema == nil {
		t.Fatal("UTF-8 byte 制約を持つ schema がありません")
	}
	if schema.MaxLength != nil {
		t.Fatalf(
			"UTF-8 byte 制約を Unicode 文字数の maxLength として公開しています: %#v",
			schema,
		)
	}
	rawLimit, exists := schema.Extra["x-maxUtf8Bytes"]
	limit, number := rawLimit.(float64)
	if !exists || !number || limit != float64(want) {
		t.Fatalf(
			"x-maxUtf8Bytes = %#v, want %d",
			rawLimit,
			want,
		)
	}
	if !strings.Contains(schema.Description, fmt.Sprintf("%d byte", want)) {
		t.Fatalf(
			"UTF-8 byte 制約の説明 = %q, want %d byte",
			schema.Description,
			want,
		)
	}
}
