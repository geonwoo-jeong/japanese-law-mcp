// Package mcp は、公開する MCP capability とツールを組み立てる。
package mcp

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/comparelawversions"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/getarticle"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/getlaw"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/listlawrevisions"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/listlawupdates"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlawcontent"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlaws"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ToolExposure は、MCP tools/list へ提示する公開面を表す。
type ToolExposure string

const (
	// ToolExposureCompact は、統合照会と専門操作の発見・実行入口だけを公開する。
	ToolExposureCompact ToolExposure = "compact"
	// ToolExposureFull は、従来の専門ツールをすべて直接公開する。
	ToolExposureFull ToolExposure = "full"
)

// Dependencies は、公開 MCP サーバーへ注入する能力ポートを保持する。
type Dependencies struct {
	SearchLaws            searchlaws.Port
	SearchLawContent      searchlawcontent.Port
	GetLaw                getlaw.Port
	CompareLawVersions    comparelawversions.Port
	GetArticle            getarticle.Port
	ListLawRevisions      listlawrevisions.Port
	ListLawUpdates        listlawupdates.Port
	QueryLegalInformation legalquery.Port
	JudicialCases         JudicialCasesDependencies
	JudicialCitations     JudicialCitationsDependencies
	LegislativeHistory    LegislativeHistoryDependencies
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
	return newLegacyServer(version, dependencies, nil)
}

// NewServerWithDependenciesAndExposure は、指定した公開面の MCP サーバーを返す。
func NewServerWithDependenciesAndExposure(
	version string,
	dependencies Dependencies,
	exposure ToolExposure,
) (*sdk.Server, error) {
	return newServer(version, dependencies, exposure, nil)
}

// NewSessionlessServerWithDependencies は、HTTP session ID を発行しない MCP サーバーを返す。
func NewSessionlessServerWithDependencies(
	version string,
	dependencies Dependencies,
) *sdk.Server {
	return newLegacyServer(version, dependencies, func() string { return "" })
}

// NewSessionlessServerWithDependenciesAndExposure は、HTTP session ID を発行せず、
// 指定した公開面を持つ MCP サーバーを返す。
func NewSessionlessServerWithDependenciesAndExposure(
	version string,
	dependencies Dependencies,
	exposure ToolExposure,
) (*sdk.Server, error) {
	return newServer(
		version,
		dependencies,
		exposure,
		func() string { return "" },
	)
}

func newServer(
	version string,
	dependencies Dependencies,
	exposure ToolExposure,
	getSessionID func() string,
) (*sdk.Server, error) {
	if exposure != ToolExposureCompact && exposure != ToolExposureFull {
		return nil, fmt.Errorf("tool exposure %q は compact または full でなければなりません", exposure)
	}
	registry, err := newOperationRegistry(dependencies)
	if err != nil {
		return nil, err
	}
	if err := validateRequiredPublicOperations(registry); err != nil {
		return nil, err
	}
	server := newSDKServer(version, getSessionID)
	if exposure == ToolExposureFull {
		if err := registry.addAllTo(server); err != nil {
			return nil, err
		}
		return server, nil
	}

	specialists := registry.specialists()
	addDiscoverLegalToolsTool(server, specialists)
	addExecuteLegalToolTool(server, specialists)
	query, _ := registry.lookup(queryLegalInformationToolName)
	tool, cloneErr := cloneToolDefinition(query.tool)
	if cloneErr != nil {
		return nil, fmt.Errorf("%s を登録できません: %w", queryLegalInformationToolName, cloneErr)
	}
	server.AddTool(tool, query.handler)
	return server, nil
}

func newSDKServer(
	version string,
	getSessionID func() string,
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
			GetSessionID: getSessionID,
		},
	)
	server.AddReceivingMiddleware(requestPacingMiddleware)
	return server
}

func newLegacyServer(
	version string,
	dependencies Dependencies,
	getSessionID func() string,
) *sdk.Server {
	server := newSDKServer(version, getSessionID)
	if dependencies.SearchLaws != nil {
		addSearchLawsTool(server, dependencies.SearchLaws)
	}
	if dependencies.SearchLawContent != nil {
		addSearchLawContentTool(server, dependencies.SearchLawContent)
	}
	if dependencies.GetLaw != nil {
		addGetLawTool(server, dependencies.GetLaw)
	}
	if !isNilCompareLawVersionsPort(dependencies.CompareLawVersions) {
		addCompareLawVersionsTool(server, dependencies.CompareLawVersions)
	}
	if dependencies.GetArticle != nil {
		addGetArticleTool(server, dependencies.GetArticle)
	}
	if !isNilListLawRevisionsPort(dependencies.ListLawRevisions) {
		addListLawRevisionsTool(server, dependencies.ListLawRevisions)
	}
	if dependencies.ListLawUpdates != nil {
		addListLawUpdatesTool(server, dependencies.ListLawUpdates)
	}
	if !isNilQueryLegalInformationPort(dependencies.QueryLegalInformation) {
		addQueryLegalInformationTool(server, dependencies.QueryLegalInformation)
	}
	dependencies.JudicialCases.addTools(server)
	if dependencies.JudicialCases.ready() {
		dependencies.JudicialCitations.addTools(server)
	}
	dependencies.LegislativeHistory.addTools(server)
	return server
}
