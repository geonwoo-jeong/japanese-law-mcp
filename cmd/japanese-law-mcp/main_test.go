package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
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

func TestServerRunnerReportsHTTPImplementationGap(t *testing.T) {
	t.Parallel()

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

	err = newServerRunner("test-version", stdio.Run)(context.Background(), cfg)
	if !errors.Is(err, errStreamableHTTPNotImplemented) {
		t.Fatalf("サーバー実行エラー = %v, want %v", err, errStreamableHTTPNotImplemented)
	}
}

func TestDefaultProviderRoutesActivateFiveEGovBindings(t *testing.T) {
	t.Parallel()

	registry, routes, err := newDefaultProviderRoutes()
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

func TestExecutableServesMCPOverStdio(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPServerProcess$") //nolint:gosec // SOT-ENG-015: テスト自身を固定引数で子プロセスとして起動する。
	command.Env = isolatedServerEnvironment(t.TempDir())
	var stderr bytes.Buffer
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
