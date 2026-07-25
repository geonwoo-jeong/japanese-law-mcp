package mcp

import (
	"context"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewServerAdvertisesInitialContract(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- NewServer("test-version").Run(ctx, serverTransport)
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
	if len(tools.Tools) != 0 {
		t.Fatalf("ツール数 = %d, want 0", len(tools.Tools))
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
