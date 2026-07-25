// Package stdio は、MCP サーバーを標準入出力へ接続する。
package stdio

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server は、一つの MCP トランスポートで実行できるサーバーを表す。
type Server interface {
	Run(context.Context, sdk.Transport) error
}

// Run は、SOT-DEL-001 に従い、サーバーを現在のプロセスの標準入出力へ接続して実行する。
func Run(ctx context.Context, server Server) error {
	return server.Run(ctx, &sdk.StdioTransport{})
}
