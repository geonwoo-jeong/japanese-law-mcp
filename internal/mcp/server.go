// Package mcp は、公開する MCP capability とツールを組み立てる。
package mcp

import (
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/getarticle"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/getlaw"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/listlawupdates"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/searchlawcontent"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/searchlaws"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Dependencies は、公開 MCP サーバーへ注入する能力ポートを保持する。
type Dependencies struct {
	SearchLaws       searchlaws.Port
	SearchLawContent searchlawcontent.Port
	GetLaw           getlaw.Port
	GetArticle       getarticle.Port
	ListLawUpdates   listlawupdates.Port
}

// NewServer は、依存を必要としない capability だけを持つ MCP サーバーを返す。
func NewServer(version string) *sdk.Server {
	return NewServerWithDependencies(version, Dependencies{})
}

// NewServerWithDependencies は、注入された capability を持つ MCP サーバーを返す。
func NewServerWithDependencies(
	version string,
	dependencies Dependencies,
) *sdk.Server {
	server := sdk.NewServer(
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
	if dependencies.SearchLaws != nil {
		addSearchLawsTool(server, dependencies.SearchLaws)
	}
	if dependencies.SearchLawContent != nil {
		addSearchLawContentTool(server, dependencies.SearchLawContent)
	}
	if dependencies.GetLaw != nil {
		addGetLawTool(server, dependencies.GetLaw)
	}
	if dependencies.GetArticle != nil {
		addGetArticleTool(server, dependencies.GetArticle)
	}
	if dependencies.ListLawUpdates != nil {
		addListLawUpdatesTool(server, dependencies.ListLawUpdates)
	}
	return server
}
