package stdio

import (
	"context"
	"errors"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRunUsesSDKStdioTransport(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	server := &recordingServer{}

	if err := Run(ctx, server); err != nil {
		t.Fatalf("Run() のエラー = %v", err)
	}
	if server.context != ctx {
		t.Fatal("Run() が受け取ったコンテキストをサーバーへ渡していません")
	}
	if _, ok := server.transport.(*sdk.StdioTransport); !ok {
		t.Fatalf("トランスポート = %T, want *mcp.StdioTransport", server.transport)
	}
}

func TestRunReturnsServerError(t *testing.T) {
	t.Parallel()

	expected := errors.New("テスト用の実行エラー")
	server := &recordingServer{err: expected}

	err := Run(context.Background(), server)
	if !errors.Is(err, expected) {
		t.Fatalf("Run() のエラー = %v, want %v", err, expected)
	}
}

type recordingServer struct {
	context   context.Context
	transport sdk.Transport
	err       error
}

func (s *recordingServer) Run(ctx context.Context, transport sdk.Transport) error {
	s.context = ctx
	s.transport = transport
	return s.err
}
