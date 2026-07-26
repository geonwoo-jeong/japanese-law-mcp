package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/searchlaws"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewServerAdvertisesInitialContract(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- NewServerWithDependencies(
			"test-version",
			Dependencies{SearchLaws: stubSearchLawsPort{}},
		).Run(ctx, serverTransport)
	}()

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("MCP セッションを初期化できません: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	result := session.InitializeResult()
	if result == nil {
		t.Fatal("初期化結果がありません")
	}
	if result.ProtocolVersion != "2025-11-25" {
		t.Fatalf("プロトコルバージョン = %q, want %q", result.ProtocolVersion, "2025-11-25")
	}
	if result.ServerInfo == nil {
		t.Fatal("サーバー情報がありません")
	}
	if result.ServerInfo.Name != "japanese-law-mcp" {
		t.Fatalf("サーバー名 = %q, want %q", result.ServerInfo.Name, "japanese-law-mcp")
	}
	if result.ServerInfo.Title != "Japanese Law MCP" {
		t.Fatalf("表示名 = %q, want %q", result.ServerInfo.Title, "Japanese Law MCP")
	}
	if result.ServerInfo.Version != "test-version" {
		t.Fatalf("サーバーバージョン = %q, want %q", result.ServerInfo.Version, "test-version")
	}

	assertInitialCapabilities(t, result.Capabilities)

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ツール一覧を取得できません: %v", err)
	}
	if tools.Tools == nil {
		t.Fatal("ツール一覧が null です")
	}
	if len(tools.Tools) != 1 {
		t.Fatalf("ツール数 = %d, want 1", len(tools.Tools))
	}
	if tools.Tools[0].Name != "search_laws" {
		t.Fatalf("tool name = %q, want search_laws", tools.Tools[0].Name)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("MCP セッションを終了できません: %v", err)
	}
	select {
	case err := <-serverResult:
		if err != nil {
			t.Fatalf("MCP サーバーが正常終了しませんでした: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("MCP サーバーの終了を待機できません: %v", ctx.Err())
	}
}

func assertInitialCapabilities(t *testing.T, capabilities *sdk.ServerCapabilities) {
	t.Helper()

	if capabilities == nil {
		t.Fatal("サーバー capability がありません")
	}
	if capabilities.Tools == nil {
		t.Fatal("tools capability がありません")
	}
	if capabilities.Tools.ListChanged {
		t.Fatal("変更通知を提供しない初期ツール一覧で listChanged が有効です")
	}
	if capabilities.Completions != nil {
		t.Fatal("未定義の completions capability が提示されています")
	}
	if capabilities.Logging != nil {
		t.Fatal("未定義の logging capability が提示されています")
	}
	if capabilities.Prompts != nil {
		t.Fatal("未定義の prompts capability が提示されています")
	}
	if capabilities.Resources != nil {
		t.Fatal("未定義の resources capability が提示されています")
	}
}

func TestSearchLawsToolReturnsStructuredSuccess(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- NewServerWithDependencies(
			"test-version",
			Dependencies{SearchLaws: stubSearchLawsPort{}},
		).Run(ctx, serverTransport)
	}()

	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "test-version"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = session.Close() }()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "search_laws",
		Arguments: map[string]any{"query": "民法"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("result.IsError = true, want false: %#v", result)
	}
	var payload searchLawsOutput
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &payload); err != nil {
		t.Fatalf("content JSON error = %v", err)
	}
	if payload.TotalCount != 1 || len(payload.Items) != 1 || payload.Items[0].LawID != "law-1" {
		t.Fatalf("payload = %#v", payload)
	}
	contentObject := map[string]any{}
	if err := json.Unmarshal(
		[]byte(result.Content[0].(*sdk.TextContent).Text),
		&contentObject,
	); err != nil {
		t.Fatalf("content を比較用に解析できません: %v", err)
	}
	structuredJSON, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("structuredContent を JSON に変換できません: %v", err)
	}
	structuredObject := map[string]any{}
	if err := json.Unmarshal(structuredJSON, &structuredObject); err != nil {
		t.Fatalf("structuredContent を比較用に解析できません: %v", err)
	}
	if !mapsEqual(contentObject, structuredObject) {
		t.Fatalf(
			"content と structuredContent が一致しません: %#v != %#v",
			contentObject,
			structuredObject,
		)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case runErr := <-serverResult:
		if runErr != nil {
			t.Fatalf("server run error = %v", runErr)
		}
	case <-ctx.Done():
		t.Fatalf("server shutdown wait error = %v", ctx.Err())
	}
}

func TestSearchLawsToolReturnsErrorResult(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- NewServerWithDependencies(
			"test-version",
			Dependencies{SearchLaws: errorSearchLawsPort{}},
		).Run(ctx, serverTransport)
	}()

	client := sdk.NewClient(&sdk.Implementation{Name: "test-client", Version: "test-version"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = session.Close() }()

	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "search_laws",
		Arguments: map[string]any{"query": "民法"},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("result.IsError = false, want true: %#v", result)
	}
	var payload struct {
		Code      string `json:"code"`
		Retryable bool   `json:"retryable"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &payload); err != nil {
		t.Fatalf("error JSON error = %v", err)
	}
	if payload.Code != "source_busy" || !payload.Retryable {
		t.Fatalf("payload = %#v", payload)
	}
	if result.StructuredContent != nil {
		t.Fatalf("エラー結果に structuredContent があります: %#v", result.StructuredContent)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case runErr := <-serverResult:
		if runErr != nil {
			t.Fatalf("server run error = %v", runErr)
		}
	case <-ctx.Done():
		t.Fatalf("server shutdown wait error = %v", ctx.Err())
	}
}

func TestSearchLawsToolRejectsInvalidInputAsToolError(t *testing.T) {
	t.Parallel()

	tests := map[string]json.RawMessage{
		"query の欠落":    json.RawMessage(`{}`),
		"query の null": json.RawMessage(`{"query":null}`),
		"未定義の入力項目":     json.RawMessage(`{"query":"民法","unknown":true}`),
		"配列形式":         json.RawMessage(`[]`),
	}
	for name, arguments := range tests {
		name := name
		arguments := arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callSearchLaws(
				context.Background(),
				stubSearchLawsPort{},
				arguments,
			)
			if err != nil {
				t.Fatalf("callSearchLaws() のエラー = %v", err)
			}
			if !result.IsError || result.StructuredContent != nil {
				t.Fatalf("入力エラー結果 = %#v", result)
			}
			var payload struct {
				Code      string `json:"code"`
				Retryable bool   `json:"retryable"`
			}
			if err := json.Unmarshal(
				[]byte(result.Content[0].(*sdk.TextContent).Text),
				&payload,
			); err != nil {
				t.Fatalf("ErrorResult を解析できません: %v", err)
			}
			if payload.Code != "invalid_argument" || payload.Retryable {
				t.Fatalf("ErrorResult = %#v", payload)
			}
		})
	}
}

func mapsEqual(left, right map[string]any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil &&
		string(leftJSON) == string(rightJSON)
}

type stubSearchLawsPort struct{}

func (stubSearchLawsPort) Search(
	context.Context,
	searchlaws.Request,
) (model.LawSearchResult, error) {
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/redoc/",
	})
	if err != nil {
		return model.LawSearchResult{}, err
	}
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		return model.LawSearchResult{}, err
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "rev-1",
		Title:      "民法",
		Source:     source,
	})
	if err != nil {
		return model.LawSearchResult{}, err
	}
	return model.NewLawSearchResult(model.LawSearchResultValues{
		TotalCount: 1,
		Items:      []model.LawSummary{summary},
	})
}

type errorSearchLawsPort struct{}

func (errorSearchLawsPort) Search(
	context.Context,
	searchlaws.Request,
) (model.LawSearchResult, error) {
	provider := mustSearchLawsTestProviderDescriptor()
	return model.LawSearchResult{}, mustSourceError(
		model.SourceErrorValues{
			Code:       model.SourceErrorCodeSourceBusy,
			Provider:   provider,
			Capability: provider.Capabilities()[0],
			Operation:  testOperation("search_laws"),
		},
	)
}

type testOperation string

func (o testOperation) SourceOperationProviderID() string { return "e-gov-law-api-v2" }
func (o testOperation) SourceOperationName() string       { return string(o) }
func (o testOperation) ValidateSourceOperation() error    { return nil }

func mustSourceError(values model.SourceErrorValues) error {
	err, buildErr := model.NewSourceError(values)
	if buildErr != nil {
		return buildErr
	}
	return err
}

func mustSearchLawsTestProviderDescriptor() model.ProviderDescriptor {
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/redoc/",
	})
	if err != nil {
		panic(err)
	}
	verifiedAt, err := model.NewDate("2026-07-26")
	if err != nil {
		panic(err)
	}
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           "law.search",
		MajorVersion: 1,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(err)
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "e-gov-law-api-v2",
		Source:                 informationSource,
		AdapterContractVersion: "1.0.0",
		UpstreamSpecVersion:    "2026-07-26",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		CredentialRequired:     false,
		Capabilities:           []model.ProviderCapability{capability},
	})
	if err != nil {
		panic(err)
	}
	return descriptor
}
