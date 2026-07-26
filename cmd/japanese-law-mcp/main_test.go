package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/cli"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/config"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/transport/stdio"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/transport/streamablehttp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	testServerProcessEnvironment   = "TEST_JAPANESE_LAW_MCP_SERVER_PROCESS"
	testConfigDirectoryEnvironment = "TEST_JAPANESE_LAW_MCP_CONFIG_DIRECTORY"
)

func TestServerRunnerUsesStdio(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	called := false
	run := func(gotContext context.Context, server stdio.Server) error {
		called = true
		if gotContext != ctx {
			t.Fatal("サーバー実行コンテキストが一致しません")
		}
		if server == nil {
			t.Fatal("MCP サーバーがありません")
		}
		return nil
	}

	err := newServerRunner("test-version", run)(ctx, config.Default())
	if err != nil {
		t.Fatalf("サーバー実行エラー = %v", err)
	}
	if !called {
		t.Fatal("stdio サーバーが実行されていません")
	}
}

func TestServerRunnerUsesStreamableHTTP(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	values := config.Values{
		Transport:      string(config.TransportStreamableHTTP),
		RequestTimeout: config.Default().RequestTimeout(),
		ListenAddress:  config.Default().ListenAddress(),
		AllowedOrigins: []string{"https://example.com"},
		Diagnostics:    false,
	}
	cfg, err := config.New(values)
	if err != nil {
		t.Fatalf("テスト設定を生成できません: %v", err)
	}

	stdioCalled := false
	httpCalled := false
	err = newServerRunnerWithTransports(
		"test-version",
		func(context.Context, stdio.Server) error {
			stdioCalled = true
			return nil
		},
		func(
			gotContext context.Context,
			server *sdk.Server,
			options streamablehttp.Options,
		) error {
			httpCalled = true
			if gotContext != ctx {
				t.Fatal("サーバー実行コンテキストが一致しません")
			}
			if options.ListenAddress != cfg.ListenAddress() {
				t.Fatalf(
					"listenAddress = %q, want %q",
					options.ListenAddress,
					cfg.ListenAddress(),
				)
			}
			if len(options.AllowedOrigins) != 1 ||
				options.AllowedOrigins[0] != "https://example.com" {
				t.Fatalf("allowedOrigins = %#v", options.AllowedOrigins)
			}

			handler := streamablehttp.NewHandler(server, options)
			request := httptest.NewRequest(
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
				t.Fatalf(
					"initialize status = %d, body = %q",
					recorder.Code,
					recorder.Body.String(),
				)
			}
			if values := recorder.Header().Values("Mcp-Session-Id"); len(values) != 0 {
				t.Fatalf("Mcp-Session-Id = %#v, want absent", values)
			}
			return nil
		},
	)(ctx, cfg)
	if err != nil {
		t.Fatalf("サーバー実行エラー = %v", err)
	}
	if stdioCalled {
		t.Fatal("HTTP 設定で stdio が実行されました")
	}
	if !httpCalled {
		t.Fatal("Streamable HTTP が実行されていません")
	}
}

func TestDefaultProviderRoutesActivateFiveEGovBindings(t *testing.T) {
	t.Parallel()

	registry, routes, err := newProviderRoutes(config.Default())
	if err != nil {
		t.Fatalf("provider runtime を初期化できません: %v", err)
	}
	if descriptor, exists := registry.Descriptor("e-gov-law-api-v2"); !exists ||
		len(descriptor.Capabilities()) != 4 {
		t.Fatalf("e-Gov v2 descriptor = %#v, %t", descriptor, exists)
	}
	if descriptor, exists := registry.Descriptor("e-gov-law-api-v1"); !exists ||
		len(descriptor.Capabilities()) != 1 {
		t.Fatalf("e-Gov v1 descriptor = %#v, %t", descriptor, exists)
	}
	for _, capability := range []struct {
		id         string
		version    int
		providerID string
	}{
		{
			lawarticleread.CapabilityID,
			lawarticleread.MajorVersion,
			"e-gov-law-api-v2",
		},
		{
			lawcontentsearch.CapabilityID,
			lawcontentsearch.MajorVersion,
			"e-gov-law-api-v2",
		},
		{
			lawdocumentread.CapabilityID,
			lawdocumentread.MajorVersion,
			"e-gov-law-api-v2",
		},
		{
			lawsearch.CapabilityID,
			lawsearch.MajorVersion,
			"e-gov-law-api-v2",
		},
		{
			lawupdatelist.CapabilityID,
			lawupdatelist.MajorVersion,
			"e-gov-law-api-v1",
		},
	} {
		providerID, exists := routes.ProviderID(capability.id, capability.version)
		if !exists || providerID != capability.providerID {
			t.Fatalf(
				"%s@%d provider = %q, %t, want %q",
				capability.id,
				capability.version,
				providerID,
				exists,
				capability.providerID,
			)
		}
	}
	if port, exists := routes.LawSearch(); !exists || port == nil {
		t.Fatal("law.search route に到達できません")
	}
	if port, exists := routes.LawContentSearch(); !exists || port == nil {
		t.Fatal("law.content.search route に到達できません")
	}
	if port, exists := routes.LawDocumentRead(); !exists || port == nil {
		t.Fatal("law.document.read route に到達できません")
	}
	if port, exists := routes.LawArticleRead(); !exists || port == nil {
		t.Fatal("law.article.read route に到達できません")
	}
	if port, exists := routes.LawUpdateList(); !exists || port == nil {
		t.Fatal("law.update.list route に到達できません")
	}
}

func TestProviderRoutesRejectInvalidStartupConfiguration(t *testing.T) {
	t.Parallel()

	tests := map[string]config.Values{
		"未知の provider": withTestProviders(map[string]config.ProviderConfig{
			"unknown-provider": {Enabled: true},
		}),
		"無効化した provider の route": withTestProviders(map[string]config.ProviderConfig{
			"e-gov-law-api-v2": {Enabled: false},
		}),
		"provider 固有 setting": withTestProviders(map[string]config.ProviderConfig{
			"e-gov-law-api-v2": {
				Enabled:  true,
				Settings: map[string]any{"origin": "https://example.test"},
			},
		}),
		"credential slot": withTestProviders(map[string]config.ProviderConfig{
			"e-gov-law-api-v1": {
				Enabled: true,
				CredentialEnvRefs: map[string]config.CredentialEnvRef{
					"apiKey": {Type: "env", Name: "TEST_API_KEY"},
				},
			},
		}),
		"capability が一致しない provider": withTestProviderRoutes(map[string]config.ProviderRoute{
			"law.search@1": {
				Selection:         config.ProviderRouteSelectionPrimary,
				DefaultProviderID: "e-gov-law-api-v1",
			},
		}),
		"明示的に空の providers": withTestProviders(map[string]config.ProviderConfig{}),
		"明示的に空の providerRoutes": withTestProviderRoutes(
			map[string]config.ProviderRoute{},
		),
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.New(values)
			if err != nil {
				t.Fatalf("構造上有効なテスト設定を生成できません: %v", err)
			}
			_, _, routeErr := newProviderRoutes(cfg)
			if routeErr == nil {
				t.Fatalf("SOT-IF-004/026/037: %s を受理しました", name)
			}
			if !config.IsValidationError(routeErr) {
				t.Fatalf("SOT-IF-021: 設定エラーとして分類されませんでした: %v", routeErr)
			}
		})
	}
}

func TestServerRunnerRejectsProviderConfigurationBeforeTransport(t *testing.T) {
	t.Parallel()

	cfg, err := config.New(withTestProviders(map[string]config.ProviderConfig{
		"e-gov-law-api-v2": {Enabled: false},
	}))
	if err != nil {
		t.Fatalf("構造上有効なテスト設定を生成できません: %v", err)
	}

	called := false
	run := func(context.Context, stdio.Server) error {
		called = true
		return nil
	}
	err = newServerRunner("test-version", run)(context.Background(), cfg)
	if !config.IsValidationError(err) {
		t.Fatalf("SOT-IF-021: 設定エラー = %v", err)
	}
	if called {
		t.Fatal("SOT-ARCH-015: 無効な設定で transport を開始しました")
	}
}

func TestServerRunnerRejectsProviderConfigurationBeforeHTTPStart(t *testing.T) {
	t.Parallel()

	values := withTestProviders(map[string]config.ProviderConfig{
		"e-gov-law-api-v2": {Enabled: false},
	})
	values.Transport = string(config.TransportStreamableHTTP)
	values.AllowedOrigins = []string{"https://example.com"}
	cfg, err := config.New(values)
	if err != nil {
		t.Fatalf("構造上有効なテスト設定を生成できません: %v", err)
	}

	called := false
	run := func(context.Context, stdio.Server) error {
		called = true
		return nil
	}
	err = newServerRunner("test-version", run)(context.Background(), cfg)
	if !config.IsValidationError(err) {
		t.Fatalf("SOT-IF-021: HTTP 起動前の設定エラー = %v", err)
	}
	if called {
		t.Fatal("SOT-ARCH-015: 無効な設定で transport を開始しました")
	}
}

func TestInvalidProviderFileExitsAsUsageErrorBeforeTransport(t *testing.T) {
	t.Parallel()

	configDirectory := t.TempDir()
	configPath := filepath.Join(configDirectory, "config.yaml")
	if err := os.WriteFile(configPath, []byte(`
providers:
  e-gov-law-api-v2:
    enabled: false
`), 0o600); err != nil {
		t.Fatalf("設定ファイルを作成できません: %v", err)
	}

	called := false
	var stderr bytes.Buffer
	code := cli.Execute(cli.Options{
		Context: context.Background(),
		Args:    []string{"--config=" + configPath},
		Stdin:   strings.NewReader(""),
		Stdout:  &bytes.Buffer{},
		Stderr:  &stderr,
		Version: "test-version",
		Run: newServerRunner("test-version", func(context.Context, stdio.Server) error {
			called = true
			return nil
		}),
		UserConfigDir: func() (string, error) {
			return configDirectory, nil
		},
	})

	if code != cli.ExitUsage {
		t.Fatalf("SOT-IF-021: 終了コード = %d、stderr = %q", code, stderr.String())
	}
	if called {
		t.Fatal("SOT-ARCH-015: 無効な設定で transport を開始しました")
	}
	if !strings.Contains(stderr.String(), "provider route") {
		t.Fatalf("SOT-IF-021: stderr = %q", stderr.String())
	}
}

func TestExecutableServesMCPOverStdio(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPServerProcess$") //nolint:gosec // SOT-ENG-015: テスト自身を固定引数で子プロセスとして起動する。
	command.Env = isolatedServerEnvironment(t.TempDir())
	var stderr synchronizedBuffer
	command.Stderr = &stderr

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-agent", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(ctx, &sdk.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatalf("子プロセスの MCP サーバーへ接続できません: %v\n標準エラー:\n%s", err, stderr.String())
	}

	result := session.InitializeResult()
	if result == nil || result.ProtocolVersion != "2025-11-25" {
		t.Fatalf("初期化結果 = %#v", result)
	}
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("子プロセスからツール一覧を取得できません: %v", err)
	}
	if tools.Tools == nil || len(tools.Tools) != 5 ||
		tools.Tools[0].Name != "get_article" ||
		tools.Tools[1].Name != "get_law" ||
		tools.Tools[2].Name != "list_law_updates" ||
		tools.Tools[3].Name != "search_law_content" ||
		tools.Tools[4].Name != "search_laws" {
		t.Fatalf(
			"公開ツール一覧 = %#v, want get_article、get_law、list_law_updates、search_law_content、search_laws",
			tools.Tools,
		)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("子プロセスの MCP セッションを終了できません: %v\n標準エラー:\n%s", err, stderr.String())
	}
}

func TestExecutableServesMCPOverStreamableHTTP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatalf("loopback port を予約できません: %v", err)
	}
	address := reservation.Addr().String()
	if err := reservation.Close(); err != nil {
		cancel()
		t.Fatalf("loopback port の予約を解放できません: %v", err)
	}

	configRoot := t.TempDir()
	configDirectory := filepath.Join(configRoot, "japanese-law-mcp")
	if err := os.MkdirAll(configDirectory, 0o700); err != nil {
		cancel()
		t.Fatalf("設定ディレクトリを作成できません: %v", err)
	}
	configPath := filepath.Join(configDirectory, "config.yaml")
	configBody := "transport: streamable-http\nlistenAddress: \"" + address + "\"\n"
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		cancel()
		t.Fatalf("HTTP 設定を書き込めません: %v", err)
	}

	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPServerProcess$") //nolint:gosec // SOT-ENG-015: テスト自身を固定引数で子プロセスとして起動する。
	command.Env = isolatedServerEnvironment(configRoot)
	var stderr synchronizedBuffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		cancel()
		t.Fatalf("HTTP 子プロセスを起動できません: %v", err)
	}
	processResult := make(chan error, 1)
	go func() {
		processResult <- command.Wait()
		close(processResult)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-processResult:
		case <-time.After(3 * time.Second):
			_ = command.Process.Kill()
			<-processResult
		}
	})

	waitForTCP(t, ctx, address, &stderr, processResult)

	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-agent", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(
		ctx,
		&sdk.StreamableClientTransport{
			Endpoint: "http://" + address + "/mcp",
		},
		nil,
	)
	if err != nil {
		t.Fatalf("HTTP MCP サーバーへ接続できません: %v\n標準エラー:\n%s", err, stderr.String())
	}
	if session.ID() != "" {
		t.Fatalf("HTTP session ID = %q, want empty", session.ID())
	}
	result := session.InitializeResult()
	if result == nil || result.ProtocolVersion != "2025-11-25" {
		t.Fatalf("初期化結果 = %#v", result)
	}
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("HTTP 子プロセスからツール一覧を取得できません: %v", err)
	}
	if tools.Tools == nil || len(tools.Tools) != 5 ||
		tools.Tools[0].Name != "get_article" ||
		tools.Tools[1].Name != "get_law" ||
		tools.Tools[2].Name != "list_law_updates" ||
		tools.Tools[3].Name != "search_law_content" ||
		tools.Tools[4].Name != "search_laws" {
		t.Fatalf("HTTP 公開ツール一覧 = %#v", tools.Tools)
	}
	callResult, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "get_law",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("HTTP 経由で get_law を呼び出せません: %v", err)
	}
	if !callResult.IsError {
		t.Fatalf("入力不足の get_law 結果 = %#v, want tool error", callResult)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("HTTP MCP セッションを終了できません: %v", err)
	}
}

func waitForTCP(
	t *testing.T,
	ctx context.Context,
	address string,
	stderr *synchronizedBuffer,
	processResult <-chan error,
) {
	t.Helper()

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		connection, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return
		}
		select {
		case processErr := <-processResult:
			t.Fatalf(
				"HTTP 子プロセスが起動前に終了しました: %v\n標準エラー:\n%s",
				processErr,
				stderr.String(),
			)
		case <-ctx.Done():
			t.Fatalf(
				"HTTP 子プロセスの起動を待機できません: %v\n標準エラー:\n%s",
				ctx.Err(),
				stderr.String(),
			)
		case <-ticker.C:
		}
	}
}

type synchronizedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *synchronizedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(value)
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func TestMCPServerProcess(t *testing.T) {
	if os.Getenv(testServerProcessEnvironment) != "1" {
		return
	}

	code := cli.Execute(cli.Options{
		Context: context.Background(),
		Args:    nil,
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Version: "test-version",
		UserConfigDir: func() (string, error) {
			return os.Getenv(testConfigDirectoryEnvironment), nil
		},
		Run: newServerRunner("test-version", stdio.Run),
	})
	os.Exit(code)
}

func isolatedServerEnvironment(configDirectory string) []string {
	environment := make([]string, 0, len(os.Environ())+2)
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "JAPANESE_LAW_MCP_") {
			continue
		}
		environment = append(environment, value)
	}
	return append(
		environment,
		testServerProcessEnvironment+"=1",
		testConfigDirectoryEnvironment+"="+configDirectory,
	)
}

func defaultTestConfigValues() config.Values {
	defaults := config.Default()
	return config.Values{
		Transport:      string(defaults.Transport()),
		RequestTimeout: defaults.RequestTimeout(),
		ListenAddress:  defaults.ListenAddress(),
		AllowedOrigins: defaults.AllowedOrigins(),
		Diagnostics:    defaults.Diagnostics(),
	}
}

func withTestProviders(providers map[string]config.ProviderConfig) config.Values {
	values := defaultTestConfigValues()
	values.Providers = providers
	return values
}

func withTestProviderRoutes(routes map[string]config.ProviderRoute) config.Values {
	values := defaultTestConfigValues()
	values.ProviderRoutes = routes
	return values
}
