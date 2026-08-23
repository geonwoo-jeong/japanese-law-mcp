package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/comparelawversions"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlaws"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewSessionlessServerDoesNotIssueSessionID(t *testing.T) {
	t.Parallel()

	server := NewSessionlessServerWithDependencies("test-version", Dependencies{})
	handler := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server {
			return server
		},
		&sdk.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		},
	)
	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/mcp",
		strings.NewReader(
			`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		),
	)
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if values := recorder.Header().Values("Mcp-Session-Id"); len(values) != 0 {
		t.Fatalf("Mcp-Session-Id = %#v, want absent", values)
	}
}

func TestNewServerAdvertisesInitialContract(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	compareLawVersions := &stubCompareLawVersionsPort{}
	listLawUpdates := &recordingListLawUpdatesPort{
		result: mustListLawUpdatesResult(t),
	}
	listLawRevisions := &recordingListLawRevisionsPort{
		result: mustMCPListLawRevisionsResult(t, "law-1"),
	}
	queryLegalInformation := &recordingQueryLegalInformationPort{
		result: queryLegalInformationResultFixture(
			t,
			newQuerySchemaModelFixture(t),
		)["unsupported"],
	}
	go func() {
		serverResult <- NewServerWithDependencies(
			"test-version",
			Dependencies{
				SearchLaws:            stubSearchLawsPort{},
				SearchLawContent:      stubSearchLawContentPort{},
				GetLaw:                stubGetLawPort{},
				CompareLawVersions:    compareLawVersions,
				GetArticle:            &recordingGetArticlePort{},
				ListLawRevisions:      listLawRevisions,
				ListLawUpdates:        listLawUpdates,
				QueryLegalInformation: queryLegalInformation,
			},
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
	if len(tools.Tools) != 8 {
		t.Fatalf("ツール数 = %d, want 8", len(tools.Tools))
	}
	if tools.Tools[0].Name != "compare_law_versions" ||
		tools.Tools[1].Name != "get_article" ||
		tools.Tools[2].Name != "get_law" ||
		tools.Tools[3].Name != "list_law_revisions" ||
		tools.Tools[4].Name != "list_law_updates" ||
		tools.Tools[5].Name != "query_legal_information" ||
		tools.Tools[6].Name != "search_law_content" ||
		tools.Tools[7].Name != "search_laws" {
		t.Fatalf(
			"tool names = %q, %q, %q, %q, %q, %q, %q, %q",
			tools.Tools[0].Name,
			tools.Tools[1].Name,
			tools.Tools[2].Name,
			tools.Tools[3].Name,
			tools.Tools[4].Name,
			tools.Tools[5].Name,
			tools.Tools[6].Name,
			tools.Tools[7].Name,
		)
	}
	for _, tool := range tools.Tools {
		if tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatalf("%s の schema がありません", tool.Name)
		}
	}
	callResult, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "query_legal_information",
		Arguments: map[string]any{
			"query": "永住許可について教えてください",
		},
	})
	if err != nil {
		t.Fatalf("query_legal_information を呼び出せません: %v", err)
	}
	if callResult == nil || callResult.IsError {
		t.Fatalf(
			"query_legal_information の結果 = %#v, want success",
			callResult,
		)
	}
	if calls, _, request := queryLegalInformation.snapshot(); calls != 1 ||
		request.Query() != "永住許可について教えてください" {
		t.Fatalf(
			"query_legal_information の application 呼出し = %d, query = %q",
			calls,
			request.Query(),
		)
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

func TestJudicialCasesToolsAreRegisteredAtomically(t *testing.T) {
	t.Parallel()

	complete, err := NewJudicialCasesDependencies(
		stubSearchJudicialCasesPort{},
		&recordingGetJudicialCasePort{
			item: mustJudicialDecisionDetailsResource(),
		},
	)
	if err != nil {
		t.Fatalf("judicial-cases dependencies を作成できません: %v", err)
	}
	tests := []struct {
		name         string
		dependencies JudicialCasesDependencies
		wantNames    []string
	}{
		{
			name:      "disabled",
			wantNames: []string{},
		},
		{
			name:         "complete",
			dependencies: complete,
			wantNames: []string{
				"get_judicial_case",
				"search_judicial_cases",
			},
		},
		{
			name: "partial internal value",
			dependencies: JudicialCasesDependencies{
				search:      stubSearchJudicialCasesPort{},
				initialized: true,
			},
			wantNames: []string{},
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			serverTransport, clientTransport := sdk.NewInMemoryTransports()
			serverResult := make(chan error, 1)
			go func() {
				serverResult <- NewServerWithDependencies(
					"test-version",
					Dependencies{JudicialCases: testCase.dependencies},
				).Run(ctx, serverTransport)
			}()

			client := sdk.NewClient(
				&sdk.Implementation{Name: "test-client", Version: "test-version"},
				nil,
			)
			session, connectErr := client.Connect(ctx, clientTransport, nil)
			if connectErr != nil {
				t.Fatalf("MCP セッションを初期化できません: %v", connectErr)
			}
			tools, listErr := session.ListTools(ctx, nil)
			if listErr != nil {
				t.Fatalf("tools/list error = %v", listErr)
			}
			names := make([]string, len(tools.Tools))
			for index, tool := range tools.Tools {
				names[index] = tool.Name
			}
			if !reflect.DeepEqual(names, testCase.wantNames) {
				t.Fatalf(
					"SOT-IF-040: tool names = %#v, want %#v",
					names,
					testCase.wantNames,
				)
			}
			if closeErr := session.Close(); closeErr != nil {
				t.Fatalf("MCP セッションを終了できません: %v", closeErr)
			}
			select {
			case runErr := <-serverResult:
				if runErr != nil {
					t.Fatalf("MCP サーバーが正常終了しませんでした: %v", runErr)
				}
			case <-ctx.Done():
				t.Fatalf("MCP サーバーの終了を待機できません: %v", ctx.Err())
			}
		})
	}
}

func TestNewJudicialCasesDependenciesRejectsMissingPort(t *testing.T) {
	t.Parallel()

	var typedNilSearch *recordingJudicialSearchPort
	var typedNilRead *recordingGetJudicialCasePort
	tests := []struct {
		name   string
		search judicialdecisionsearch.Port
		read   judicialdecisionread.Port
	}{
		{
			name: "nil search",
			read: &recordingGetJudicialCasePort{
				item: mustJudicialDecisionDetailsResource(),
			},
		},
		{
			name:   "typed nil search",
			search: typedNilSearch,
			read: &recordingGetJudicialCasePort{
				item: mustJudicialDecisionDetailsResource(),
			},
		},
		{
			name:   "nil read",
			search: stubSearchJudicialCasesPort{},
		},
		{
			name:   "typed nil read",
			search: stubSearchJudicialCasesPort{},
			read:   typedNilRead,
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewJudicialCasesDependencies(
				testCase.search,
				testCase.read,
			); err == nil {
				t.Fatal("不完全な judicial-cases dependencies を受理しました")
			}
		})
	}
}

func TestLegislativeHistoryToolIsRegisteredAtomically(t *testing.T) {
	t.Parallel()

	complete, err := NewLegislativeHistoryDependencies(
		stubSearchDietSpeechesPort{},
	)
	if err != nil {
		t.Fatalf("legislative-history dependencies を作成できません: %v", err)
	}
	tests := []struct {
		name         string
		dependencies LegislativeHistoryDependencies
		wantNames    []string
	}{
		{name: "disabled", wantNames: []string{}},
		{
			name:         "complete",
			dependencies: complete,
			wantNames:    []string{"search_diet_speeches"},
		},
		{
			name: "partial internal value",
			dependencies: LegislativeHistoryDependencies{
				initialized: true,
			},
			wantNames: []string{},
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			serverTransport, clientTransport := sdk.NewInMemoryTransports()
			serverResult := make(chan error, 1)
			go func() {
				serverResult <- NewServerWithDependencies(
					"test-version",
					Dependencies{LegislativeHistory: testCase.dependencies},
				).Run(ctx, serverTransport)
			}()
			client := sdk.NewClient(
				&sdk.Implementation{Name: "test-client", Version: "test-version"},
				nil,
			)
			session, connectErr := client.Connect(ctx, clientTransport, nil)
			if connectErr != nil {
				t.Fatalf("MCP セッションを初期化できません: %v", connectErr)
			}
			tools, listErr := session.ListTools(ctx, nil)
			if listErr != nil {
				t.Fatalf("tools/list error = %v", listErr)
			}
			names := make([]string, len(tools.Tools))
			for index, tool := range tools.Tools {
				names[index] = tool.Name
			}
			if !reflect.DeepEqual(names, testCase.wantNames) {
				t.Fatalf("SOT-IF-061: tool names = %#v, want %#v", names, testCase.wantNames)
			}
			if closeErr := session.Close(); closeErr != nil {
				t.Fatalf("MCP セッションを終了できません: %v", closeErr)
			}
			select {
			case runErr := <-serverResult:
				if runErr != nil {
					t.Fatalf("MCP サーバーが正常終了しませんでした: %v", runErr)
				}
			case <-ctx.Done():
				t.Fatalf("MCP サーバーの終了を待機できません: %v", ctx.Err())
			}
		})
	}
}

func TestNewLegislativeHistoryDependenciesRejectsMissingPort(t *testing.T) {
	t.Parallel()

	var typedNil *recordingDietSpeechSearchPort
	tests := []parliamentspeechsearch.Port{
		nil,
		typedNil,
	}
	for _, port := range tests {
		if _, err := NewLegislativeHistoryDependencies(port); err == nil {
			t.Fatal("不完全な legislative-history dependencies を受理しました")
		}
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

type stubCompareLawVersionsPort struct{}

func (*stubCompareLawVersionsPort) Compare(
	context.Context,
	comparelawversions.Request,
) (model.LawVersionComparison, error) {
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/redoc/",
	})
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	beforeLaw, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "rev-1",
		Title:      "民法",
		Source:     source,
	})
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	afterLaw, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "rev-2",
		Title:      "民法",
		Source:     source,
	})
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	beforeCitation, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      "law-1",
		RevisionID: "rev-1",
		URL:        "https://laws.e-gov.go.jp/document?lawid=law-1&rev=rev-1",
	})
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	afterCitation, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      "law-1",
		RevisionID: "rev-2",
		URL:        "https://laws.e-gov.go.jp/document?lawid=law-1&rev=rev-2",
	})
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	beforeSnapshot, err := model.NewLawVersionSnapshot(model.LawVersionSnapshotValues{
		Law:      beforeLaw,
		Citation: beforeCitation,
	})
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	afterSnapshot, err := model.NewLawVersionSnapshot(model.LawVersionSnapshotValues{
		Law:      afterLaw,
		Citation: afterCitation,
	})
	if err != nil {
		return model.LawVersionComparison{}, err
	}
	return model.NewLawVersionComparison(model.LawVersionComparisonValues{
		LawID:              "law-1",
		Scope:              model.LawVersionComparisonScopeMainAndOriginalSupplementaryArticles,
		Before:             beforeSnapshot,
		After:              afterSnapshot,
		BeforeArticleCount: 0,
		AfterArticleCount:  0,
		AddedCount:         0,
		RemovedCount:       0,
		ModifiedCount:      0,
		UnchangedCount:     0,
		TotalCount:         0,
		Items:              nil,
	})
}

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
