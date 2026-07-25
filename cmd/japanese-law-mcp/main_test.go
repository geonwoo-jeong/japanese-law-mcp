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

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/cli"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/config"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/transport/stdio"
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

func TestExecutableServesMCPOverStdio(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	command := exec.Command(os.Args[0], "-test.run=^TestMCPServerProcess$")
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
	if tools.Tools == nil || len(tools.Tools) != 0 {
		t.Fatalf("初期ツール一覧 = %#v, want 空配列", tools.Tools)
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
