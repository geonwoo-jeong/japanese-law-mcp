package main

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/config"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/transport/stdio"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/transport/streamablehttp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerRunnerWiresDiagnosticsForEveryTransportOnlyWhenEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		transport   config.Transport
		diagnostics bool
		want        string
	}{
		{
			name:      "stdio disabled",
			transport: config.TransportStdio,
		},
		{
			name:      "streamable HTTP disabled",
			transport: config.TransportStreamableHTTP,
		},
		{
			name:        "stdio enabled",
			transport:   config.TransportStdio,
			diagnostics: true,
			want: `{"component":"mcp","operation":"search_laws","errorCode":"invalid_argument"}` +
				"\n",
		},
		{
			name:        "streamable HTTP enabled",
			transport:   config.TransportStreamableHTTP,
			diagnostics: true,
			want: `{"component":"mcp","operation":"search_laws","errorCode":"invalid_argument"}` +
				"\n",
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			values := defaultTestConfigValues()
			values.Transport = string(testCase.transport)
			values.Diagnostics = testCase.diagnostics
			cfg, err := config.New(values)
			if err != nil {
				t.Fatalf("テスト設定を生成できません: %v", err)
			}

			var output bytes.Buffer
			stdioCalled := false
			httpCalled := false
			runner := newServerRunnerWithTransportsAndDiagnostics(
				"test-version",
				func(ctx context.Context, server stdio.Server) error {
					stdioCalled = true
					return callInvalidSearchForDiagnostics(ctx, server)
				},
				func(
					ctx context.Context,
					server *sdk.Server,
					_ streamablehttp.Options,
				) error {
					httpCalled = true
					return callInvalidSearchForDiagnostics(ctx, server)
				},
				&output,
			)
			if err := runner(context.Background(), cfg); err != nil {
				t.Fatalf("server runner error = %v", err)
			}
			if stdioCalled != (testCase.transport == config.TransportStdio) {
				t.Fatalf("stdio called = %t", stdioCalled)
			}
			if httpCalled != (testCase.transport == config.TransportStreamableHTTP) {
				t.Fatalf("HTTP called = %t", httpCalled)
			}
			if got := output.String(); got != testCase.want {
				t.Fatalf(
					"SOT-ARCH-008/SOT-IF-029: diagnostics = %q, want %q",
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestServerRunnerRejectsMissingEnabledDiagnosticsWriterBeforeTransport(
	t *testing.T,
) {
	t.Parallel()

	values := defaultTestConfigValues()
	values.Diagnostics = true
	cfg, err := config.New(values)
	if err != nil {
		t.Fatalf("テスト設定を生成できません: %v", err)
	}

	transportCalled := false
	runner := newServerRunnerWithTransportsAndDiagnostics(
		"test-version",
		func(context.Context, stdio.Server) error {
			transportCalled = true
			return nil
		},
		func(context.Context, *sdk.Server, streamablehttp.Options) error {
			transportCalled = true
			return nil
		},
		nil,
	)
	if err := runner(context.Background(), cfg); err == nil {
		t.Fatal("診断出力先がない設定を受理しました")
	}
	if transportCalled {
		t.Fatal("診断の初期化失敗後に transport を起動しました")
	}
}

func callInvalidSearchForDiagnostics(
	parent context.Context,
	server stdio.Server,
) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
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
		return fmt.Errorf("MCP セッションを初期化できません: %w", err)
	}

	result, callErr := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "search_laws",
		Arguments: map[string]any{
			"query":       "診断へ出してはいけない検索語",
			"hiddenField": "診断へ出してはいけない値",
		},
	})
	closeErr := session.Close()
	runErr := <-serverResult
	if callErr != nil {
		return fmt.Errorf("CallTool() error = %w", callErr)
	}
	if result == nil || !result.IsError {
		return fmt.Errorf("入力エラーを返しませんでした: %#v", result)
	}
	if closeErr != nil {
		return fmt.Errorf("MCP セッションを終了できません: %w", closeErr)
	}
	if runErr != nil {
		return fmt.Errorf("MCP サーバーが正常終了しませんでした: %w", runErr)
	}
	return nil
}
