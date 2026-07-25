// Package mcp は、公開する MCP capability とツールを組み立てる。
package mcp

import sdk "github.com/modelcontextprotocol/go-sdk/mcp"

// NewServer は、SOT-IF-013 に従い、現在実装済みの capability だけを持つ MCP サーバーを返す。
func NewServer(version string) *sdk.Server {
	return sdk.NewServer(
		&sdk.Implementation{
			Name:    "japanese-law-mcp",
			Title:   "Japanese Law MCP",
			Version: version,
		},
		&sdk.ServerOptions{
			Capabilities: &sdk.ServerCapabilities{
				Tools: &sdk.ToolCapabilities{},
			},
		},
	)
}
