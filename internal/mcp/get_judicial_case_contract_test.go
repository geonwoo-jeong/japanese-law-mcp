package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetJudicialCaseRejectsInvalidInputBeforePort(t *testing.T) {
	t.Parallel()

	invalidUTF8 := append(
		[]byte(`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"`),
		append(
			[]byte{0xff},
			[]byte(`","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`)...,
		)...,
	)
	tests := []struct {
		name      string
		arguments json.RawMessage
	}{
		{name: "入力なし", arguments: nil},
		{name: "null object", arguments: json.RawMessage(`null`)},
		{name: "配列", arguments: json.RawMessage(`[]`)},
		{name: "不正 JSON", arguments: json.RawMessage(`{"ref":`)},
		{name: "ref 欠落", arguments: json.RawMessage(`{}`)},
		{name: "ref null", arguments: json.RawMessage(`{"ref":null}`)},
		{name: "ref 型", arguments: json.RawMessage(`{"ref":"courts"}`)},
		{
			name:      "最上位の未知項目",
			arguments: json.RawMessage(`{"ref":{},"unknown":true}`),
		},
		{
			name: "ref の未知項目",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":{},"unknown":true}}`,
			),
		},
		{
			name: "providerId 欠落",
			arguments: json.RawMessage(
				`{"ref":{"key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`,
			),
		},
		{
			name: "providerId null",
			arguments: json.RawMessage(
				`{"ref":{"providerId":null,"key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`,
			),
		},
		{
			name: "providerId 型",
			arguments: json.RawMessage(
				`{"ref":{"providerId":1,"key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`,
			),
		},
		{
			name: "providerId 単独 surrogate",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"\ud800","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`,
			),
		},
		{
			name: "key 欠落",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html"}}`,
			),
		},
		{
			name: "key null",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":null}}`,
			),
		},
		{
			name: "key の未知項目",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2","unknown":true}}}`,
			),
		},
		{
			name: "sourceId 欠落",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":{"resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`,
			),
		},
		{
			name: "sourceId 空",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"","resourceType":"judicial-decision","resourceId":"95570/detail2"}}}`,
			),
		},
		{
			name: "resourceType 不一致",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"law","resourceId":"95570/detail2"}}}`,
			),
		},
		{
			name: "resourceId 欠落",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision"}}}`,
			),
		},
		{
			name: "resourceId 空",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":""}}}`,
			),
		},
		{
			name: "versionId 指定",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2","versionId":"v1"}}}`,
			),
		},
		{
			name: "versionId 空文字指定",
			arguments: json.RawMessage(
				`{"ref":{"providerId":"courts-hanrei-html","key":{"sourceId":"courts-hanrei","resourceType":"judicial-decision","resourceId":"95570/detail2","versionId":""}}}`,
			),
		},
		{name: "不正 UTF-8", arguments: json.RawMessage(invalidUTF8)},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			port := &recordingGetJudicialCasePort{}
			result, err := callGetJudicialCase(
				context.Background(),
				port,
				testCase.arguments,
			)
			if err != nil {
				t.Fatalf("callGetJudicialCase() error = %v", err)
			}
			payload := assertJudicialSearchErrorCode(
				t,
				result,
				"invalid_argument",
			)
			if payload.Details["field"] != "ref" {
				t.Fatalf("SOT-IF-027/048: details = %#v", payload.Details)
			}
			if port.calls != 0 {
				t.Fatalf("SOT-IF-048: 不正入力で port を %d 回呼び出した", port.calls)
			}
		})
	}
}

func TestGetJudicialCaseRoundTripsPublishedSearchRef(t *testing.T) {
	t.Parallel()

	searchResult, err := callSearchJudicialCases(
		context.Background(),
		stubSearchJudicialCasesPort{},
		json.RawMessage(`{"query":"民法"}`),
	)
	if err != nil || searchResult.IsError {
		t.Fatalf("search_judicial_cases result = %#v, err = %v", searchResult, err)
	}
	var searchPayload struct {
		Items []struct {
			Ref json.RawMessage `json:"ref"`
		} `json:"items"`
	}
	if err := json.Unmarshal(
		[]byte(searchResult.Content[0].(*sdk.TextContent).Text),
		&searchPayload,
	); err != nil || len(searchPayload.Items) != 1 {
		t.Fatalf("search payload = %#v, err = %v", searchPayload, err)
	}
	arguments, err := json.Marshal(map[string]json.RawMessage{
		"ref": searchPayload.Items[0].Ref,
	})
	if err != nil {
		t.Fatalf("get arguments を作成できません: %v", err)
	}

	port := &recordingGetJudicialCasePort{
		item: mustJudicialDecisionDetailsResource(),
	}
	getResult, err := callGetJudicialCase(
		context.Background(),
		port,
		arguments,
	)
	if err != nil || getResult.IsError || port.calls != 1 {
		t.Fatalf(
			"SOT-IF-047/048: get result = %#v, calls = %d, err = %v",
			getResult,
			port.calls,
			err,
		)
	}
	if port.request.Ref().Key().ResourceID() != "95570/detail2" {
		t.Fatalf("SOT-IF-044/048: ref = %#v", port.request.Ref())
	}
}

func TestGetJudicialCaseMapsArgumentAndResolutionErrors(t *testing.T) {
	t.Parallel()

	argumentError, err := judicialdecisionread.NewArgumentError(
		"ref",
		"resourceId は {decisionId}/detail{2..8} でなければなりません",
	)
	if err != nil {
		t.Fatalf("ArgumentError を作成できません: %v", err)
	}
	result, err := callGetJudicialCase(
		context.Background(),
		&recordingGetJudicialCasePort{
			err: fmt.Errorf("provider validation: %w", argumentError),
		},
		canonicalJudicialCaseInput(),
	)
	if err != nil {
		t.Fatalf("ArgumentError result error = %v", err)
	}
	payload := assertJudicialSearchErrorCode(t, result, "invalid_argument")
	if payload.Details["field"] != "ref" ||
		payload.Details["reason"] !=
			"resourceId は {decisionId}/detail{2..8} でなければなりません" {
		t.Fatalf("SOT-IF-027/048: details = %#v", payload.Details)
	}

	t.Run("unknown provider", func(t *testing.T) {
		underlying := &recordingGetJudicialCasePort{}
		service := mustJudicialReadService(t, judicialdecisionread.ProviderBinding{
			Descriptor: hanreiTestProviderDescriptor(),
			Enabled:    true,
			Port:       underlying,
		})
		result, callErr := callGetJudicialCase(
			context.Background(),
			service,
			judicialCaseInput(
				"unknown-provider",
				"courts-hanrei",
				"95570/detail2",
			),
		)
		if callErr != nil {
			t.Fatalf("unknown provider result error = %v", callErr)
		}
		payload := assertJudicialSearchErrorCode(t, result, "invalid_argument")
		if payload.Details["field"] != "ref" || underlying.calls != 0 {
			t.Fatalf("SOT-IF-042/048: payload = %#v, calls = %d", payload, underlying.calls)
		}
	})

	t.Run("provider source mismatch", func(t *testing.T) {
		underlying := &recordingGetJudicialCasePort{}
		service := mustJudicialReadService(t, judicialdecisionread.ProviderBinding{
			Descriptor: hanreiTestProviderDescriptor(),
			Enabled:    true,
			Port:       underlying,
		})
		result, callErr := callGetJudicialCase(
			context.Background(),
			service,
			judicialCaseInput(
				"courts-hanrei-html",
				"other-source",
				"95570/detail2",
			),
		)
		if callErr != nil {
			t.Fatalf("source mismatch result error = %v", callErr)
		}
		assertJudicialSearchErrorCode(t, result, "invalid_argument")
		if underlying.calls != 0 {
			t.Fatalf("SOT-IF-042: source 不一致で provider を呼び出しました")
		}
	})

	t.Run("configuration required", func(t *testing.T) {
		underlying := &recordingGetJudicialCasePort{}
		service := mustJudicialReadService(t, judicialdecisionread.ProviderBinding{
			Descriptor: hanreiTestProviderDescriptor(),
			Enabled:    false,
			Port:       underlying,
		})
		result, callErr := callGetJudicialCase(
			context.Background(),
			service,
			canonicalJudicialCaseInput(),
		)
		if callErr != nil {
			t.Fatalf("configuration result error = %v", callErr)
		}
		assertJudicialSearchErrorCode(t, result, "configuration_required")
		if underlying.calls != 0 {
			t.Fatalf("SOT-IF-042: 無効 provider を呼び出しました")
		}
	})

	t.Run("unsupported capability", func(t *testing.T) {
		service := mustJudicialReadService(t, judicialdecisionread.ProviderBinding{
			Descriptor: hanreiSearchOnlyTestProviderDescriptor(),
			Enabled:    true,
		})
		result, callErr := callGetJudicialCase(
			context.Background(),
			service,
			canonicalJudicialCaseInput(),
		)
		if callErr != nil {
			t.Fatalf("unsupported result error = %v", callErr)
		}
		assertJudicialSearchErrorCode(t, result, "unsupported_capability")
	})
}

func TestGetJudicialCaseMapsNotFoundAndSourceErrors(t *testing.T) {
	t.Parallel()

	result, err := callGetJudicialCase(
		context.Background(),
		&recordingGetJudicialCasePort{
			err: fmt.Errorf("read failed: %w", judicialdecisionread.ErrNotFound),
		},
		canonicalJudicialCaseInput(),
	)
	if err != nil {
		t.Fatalf("not found result error = %v", err)
	}
	assertJudicialSearchErrorCode(t, result, "not_found")

	for _, code := range []model.SourceErrorCode{
		model.SourceErrorCodeUnsupportedCapability,
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
	} {
		code := code
		t.Run(string(code), func(t *testing.T) {
			t.Parallel()

			result, callErr := callGetJudicialCase(
				context.Background(),
				&recordingGetJudicialCasePort{
					err: mustJudicialReadSourceError(code),
				},
				canonicalJudicialCaseInput(),
			)
			if callErr != nil {
				t.Fatalf("%s result error = %v", code, callErr)
			}
			payload := assertJudicialSearchErrorCode(t, result, string(code))
			wantRetryable := code == model.SourceErrorCodeRateLimited ||
				code == model.SourceErrorCodeSourceTimeout ||
				code == model.SourceErrorCodeSourceUnavailable ||
				code == model.SourceErrorCodeSourceBusy
			if payload.Retryable != wantRetryable {
				t.Fatalf("%s retryable = %t", code, payload.Retryable)
			}
		})
	}
}

func TestGetJudicialCaseFailsClosed(t *testing.T) {
	t.Parallel()

	var typedNil *recordingGetJudicialCasePort
	for name, port := range map[string]judicialdecisionread.Port{
		"nil":       nil,
		"typed nil": typedNil,
	} {
		name, port := name, port
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result, err := callGetJudicialCase(
				context.Background(),
				port,
				canonicalJudicialCaseInput(),
			)
			if err != nil {
				t.Fatalf("nil port result error = %v", err)
			}
			assertJudicialSearchErrorCode(t, result, "internal_error")
		})
	}

	for name, port := range map[string]judicialdecisionread.Port{
		"unknown error": &recordingGetJudicialCasePort{
			err: errors.New("raw HTML /private/secret token=abc"),
		},
		"invalid item": &recordingGetJudicialCasePort{},
		"mismatched ref": &recordingGetJudicialCasePort{
			item: mustJudicialDecisionDetailsResource(),
		},
		"unsupported query is not public": &recordingGetJudicialCasePort{
			err: mustJudicialReadSourceError(
				model.SourceErrorCodeUnsupportedQuery,
			),
		},
	} {
		name, port := name, port
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			arguments := canonicalJudicialCaseInput()
			if name == "mismatched ref" {
				arguments = judicialCaseInput(
					"courts-hanrei-html",
					"courts-hanrei",
					"95571/detail2",
				)
			}
			result, err := callGetJudicialCase(
				context.Background(),
				port,
				arguments,
			)
			if err != nil {
				t.Fatalf("fail-close result error = %v", err)
			}
			assertJudicialSearchErrorCode(t, result, "internal_error")
			errorText := result.Content[0].(*sdk.TextContent).Text
			for _, secret := range []string{"raw HTML", "/private/secret", "token=abc"} {
				if strings.Contains(errorText, secret) {
					t.Fatalf("SOT-IF-027: error payload に秘密情報があります: %q", errorText)
				}
			}
		})
	}
}

func TestGetJudicialCaseToolPublishesSchemas(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	server := NewServer("test-version")
	addGetJudicialCaseTool(server, &recordingGetJudicialCasePort{
		item: mustJudicialDecisionDetailsResource(),
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
		t.Fatalf("tools/list error = %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "get_judicial_case" {
		t.Fatalf("SOT-IF-048: tools = %#v", tools.Tools)
	}
	inputSchema := decodeGetJudicialCaseSchema(t, tools.Tools[0].InputSchema)
	outputSchema := decodeGetJudicialCaseSchema(t, tools.Tools[0].OutputSchema)
	if !containsString(inputSchema.Required, "ref") ||
		!containsString(outputSchema.Required, "coverageNotice") ||
		!containsString(outputSchema.Required, "item") {
		t.Fatalf(
			"SOT-IF-007/048: required = input %#v, output %#v",
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

func judicialCaseInput(
	providerID string,
	sourceID string,
	resourceID string,
) json.RawMessage {
	input, err := json.Marshal(map[string]any{
		"ref": map[string]any{
			"providerId": providerID,
			"key": map[string]any{
				"sourceId":     sourceID,
				"resourceType": "judicial-decision",
				"resourceId":   resourceID,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	return input
}

func mustJudicialReadService(
	t *testing.T,
	binding judicialdecisionread.ProviderBinding,
) judicialdecisionread.Port {
	t.Helper()
	resolver, err := judicialdecisionread.NewResolver(
		[]judicialdecisionread.ProviderBinding{binding},
	)
	if err != nil {
		t.Fatalf("judicial read resolver を作成できません: %v", err)
	}
	service, err := judicialdecisionread.NewService(resolver, time.Second)
	if err != nil {
		t.Fatalf("judicial read service を作成できません: %v", err)
	}
	return service
}

func hanreiSearchOnlyTestProviderDescriptor() model.ProviderDescriptor {
	full := hanreiTestProviderDescriptor()
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             full.ProviderID(),
		Source:                 full.Source(),
		AdapterContractVersion: full.AdapterContractVersion(),
		UpstreamSpecVersion:    "2026-07-27",
		VerifiedAt:             full.VerifiedAt(),
		InterfaceType:          full.InterfaceType(),
		CredentialRequired:     false,
		Capabilities:           full.Capabilities()[1:],
	})
	if err != nil {
		panic(err)
	}
	return descriptor
}

func mustJudicialReadSourceError(code model.SourceErrorCode) error {
	provider := hanreiTestProviderDescriptor()
	capability := provider.Capabilities()[0]
	if code == model.SourceErrorCodeUnsupportedCapability {
		provider = hanreiSearchOnlyTestProviderDescriptor()
		capability = mustJudicialReadCapability()
	}
	return mustSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   provider,
		Capability: capability,
		Operation:  hanreiTestOperation("read_html"),
	})
}

func mustJudicialReadCapability() model.ProviderCapability {
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           judicialdecisionread.CapabilityID,
		MajorVersion: judicialdecisionread.MajorVersion,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(err)
	}
	return capability
}

func decodeGetJudicialCaseSchema(
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
