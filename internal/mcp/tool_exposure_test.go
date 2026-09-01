package mcp

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolExposureInventoriesAcrossExtensionPackCombinations(t *testing.T) {
	core := newCoreTestDependencies(t)
	judicialCases := mustJudicialCitationCasesDependencies(t)
	judicialCitations, err := NewJudicialCitationsDependencies(
		&stubTraceJudicialCitationsPort{result: mustJudicialCitationGraphResult(t)},
	)
	if err != nil {
		t.Fatalf("judicial-citations dependencies = %v", err)
	}
	legislativeHistory, err := NewLegislativeHistoryDependencies(
		stubSearchDietSpeechesPort{},
	)
	if err != nil {
		t.Fatalf("legislative-history dependencies = %v", err)
	}
	coreNames := []string{
		"compare_law_versions",
		"get_article",
		"get_law",
		"list_law_revisions",
		"list_law_updates",
		"query_legal_information",
		"search_law_content",
		"search_laws",
	}
	tests := []struct {
		name        string
		cases       bool
		citations   bool
		legislative bool
	}{
		{name: "core"},
		{name: "legislative", legislative: true},
		{name: "judicial cases", cases: true},
		{name: "judicial cases and legislative", cases: true, legislative: true},
		{name: "judicial citations", cases: true, citations: true},
		{name: "all", cases: true, citations: true, legislative: true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			dependencies := core
			wantFull := append([]string(nil), coreNames...)
			if testCase.cases {
				dependencies.JudicialCases = judicialCases
				wantFull = append(wantFull, "get_judicial_case", "search_judicial_cases")
			}
			if testCase.citations {
				dependencies.JudicialCitations = judicialCitations
				wantFull = append(wantFull, "trace_judicial_citations")
			}
			if testCase.legislative {
				dependencies.LegislativeHistory = legislativeHistory
				wantFull = append(wantFull, "search_diet_speeches")
			}
			sort.Strings(wantFull)

			compact, err := NewServerWithDependenciesAndExposure(
				"test-version",
				dependencies,
				ToolExposureCompact,
			)
			if err != nil {
				t.Fatalf("compact server = %v", err)
			}
			wantCompact := []string{
				"discover_legal_tools",
				"execute_legal_tool",
				"query_legal_information",
			}
			if got := listToolNames(t, compact); !reflect.DeepEqual(got, wantCompact) {
				t.Fatalf("compact tools = %#v, want %#v", got, wantCompact)
			}

			full, err := NewServerWithDependenciesAndExposure(
				"test-version",
				dependencies,
				ToolExposureFull,
			)
			if err != nil {
				t.Fatalf("full server = %v", err)
			}
			if got := listToolNames(t, full); !reflect.DeepEqual(got, wantFull) {
				t.Fatalf("full tools = %#v, want %#v", got, wantFull)
			}
		})
	}
}

func TestExplicitExposureStatefulAndSessionlessServersHaveParity(t *testing.T) {
	dependencies := newAllTestDependencies(t)
	statefulCompact := mustNewCompactServer(t, dependencies)
	sessionlessCompact := mustNewSessionlessCompactServer(t, dependencies)
	statefulNames := listToolNames(t, statefulCompact)
	sessionlessNames := listToolNames(t, sessionlessCompact)
	want := []string{
		"discover_legal_tools",
		"execute_legal_tool",
		"query_legal_information",
	}
	if !reflect.DeepEqual(statefulNames, want) {
		t.Fatalf("stateful compact tools = %#v, want %#v", statefulNames, want)
	}
	if !reflect.DeepEqual(sessionlessNames, statefulNames) {
		t.Fatalf("sessionless compact tools = %#v, want %#v", sessionlessNames, statefulNames)
	}

	statefulFull, err := NewServerWithDependenciesAndExposure(
		"test-version",
		dependencies,
		ToolExposureFull,
	)
	if err != nil {
		t.Fatalf("stateful full server = %v", err)
	}
	sessionlessFull, err := NewSessionlessServerWithDependenciesAndExposure(
		"test-version",
		dependencies,
		ToolExposureFull,
	)
	if err != nil {
		t.Fatalf("sessionless full server = %v", err)
	}
	if got, want := listToolNames(t, sessionlessFull), listToolNames(t, statefulFull); !reflect.DeepEqual(got, want) {
		t.Fatalf("sessionless full tools = %#v, want %#v", got, want)
	}
}

func TestExplicitExposureRejectsEveryMissingRequiredOperation(t *testing.T) {
	tests := []struct {
		name   string
		remove func(*Dependencies)
	}{
		{name: "compare_law_versions", remove: func(dependencies *Dependencies) {
			dependencies.CompareLawVersions = nil
		}},
		{name: "get_article", remove: func(dependencies *Dependencies) {
			dependencies.GetArticle = nil
		}},
		{name: "get_law", remove: func(dependencies *Dependencies) {
			dependencies.GetLaw = nil
		}},
		{name: "list_law_revisions", remove: func(dependencies *Dependencies) {
			dependencies.ListLawRevisions = nil
		}},
		{name: "list_law_updates", remove: func(dependencies *Dependencies) {
			dependencies.ListLawUpdates = nil
		}},
		{name: queryLegalInformationToolName, remove: func(dependencies *Dependencies) {
			dependencies.QueryLegalInformation = nil
		}},
		{name: "search_law_content", remove: func(dependencies *Dependencies) {
			dependencies.SearchLawContent = nil
		}},
		{name: "search_laws", remove: func(dependencies *Dependencies) {
			dependencies.SearchLaws = nil
		}},
	}
	constructors := []struct {
		name string
		new  func(Dependencies, ToolExposure) (*sdk.Server, error)
	}{
		{
			name: "stateful",
			new: func(dependencies Dependencies, exposure ToolExposure) (*sdk.Server, error) {
				return NewServerWithDependenciesAndExposure("test-version", dependencies, exposure)
			},
		},
		{
			name: "sessionless",
			new: func(dependencies Dependencies, exposure ToolExposure) (*sdk.Server, error) {
				return NewSessionlessServerWithDependenciesAndExposure(
					"test-version",
					dependencies,
					exposure,
				)
			},
		},
	}
	for _, testCase := range tests {
		for _, constructor := range constructors {
			for _, exposure := range []ToolExposure{ToolExposureCompact, ToolExposureFull} {
				name := testCase.name + "/" + constructor.name + "/" + string(exposure)
				t.Run(name, func(t *testing.T) {
					dependencies := newCoreTestDependencies(t)
					testCase.remove(&dependencies)
					server, err := constructor.new(dependencies, exposure)
					if err == nil || server != nil {
						t.Fatalf("server = %#v, error = %v", server, err)
					}
					if !strings.Contains(err.Error(), testCase.name) {
						t.Fatalf("error = %q, want missing %q", err, testCase.name)
					}
				})
			}
		}
	}
}

func TestInvalidToolExposureFailsBeforeServerConstruction(t *testing.T) {
	if server, err := NewServerWithDependenciesAndExposure(
		"test-version",
		Dependencies{},
		ToolExposure("unknown"),
	); err == nil || server != nil {
		t.Fatalf("invalid exposure server = %#v, error = %v", server, err)
	}
	if server, err := NewSessionlessServerWithDependenciesAndExposure(
		"test-version",
		Dependencies{},
		ToolExposure(""),
	); err == nil || server != nil {
		t.Fatalf("invalid sessionless exposure server = %#v, error = %v", server, err)
	}
}

func TestCompactRejectsHiddenDirectToolCalls(t *testing.T) {
	dependencies := newAllTestDependencies(t)
	server, err := NewServerWithDependenciesAndExposure(
		"test-version",
		dependencies,
		ToolExposureCompact,
	)
	if err != nil {
		t.Fatalf("compact server = %v", err)
	}
	withMCPClientSession(t, server, func(ctx context.Context, session *sdk.ClientSession) {
		for _, name := range []string{
			"search_laws",
			"get_judicial_case",
			"trace_judicial_citations",
			"search_diet_speeches",
		} {
			result, callErr := session.CallTool(ctx, &sdk.CallToolParams{
				Name:      name,
				Arguments: map[string]any{},
			})
			if callErr == nil || result != nil {
				t.Fatalf("hidden %s result = %#v, error = %v", name, result, callErr)
			}
			var wireErr *jsonrpc.Error
			if !errors.As(callErr, &wireErr) {
				t.Fatalf("hidden %s error type = %T, want *jsonrpc.Error", name, callErr)
			}
			if wireErr.Code != jsonrpc.CodeInvalidParams {
				t.Fatalf(
					"hidden %s error code = %d, want %d",
					name,
					wireErr.Code,
					jsonrpc.CodeInvalidParams,
				)
			}
			if want := `unknown tool "` + name + `"`; wireErr.Message != want {
				t.Fatalf("hidden %s error message = %q, want %q", name, wireErr.Message, want)
			}
		}
	})
}

func TestResourcesAndPromptsRemainUnpublishedAcrossExposureModes(t *testing.T) {
	dependencies := newAllTestDependencies(t)
	servers := []struct {
		name   string
		server *sdk.Server
	}{
		{
			name:   "compact stateful",
			server: mustNewCompactServer(t, dependencies),
		},
		{
			name:   "compact sessionless",
			server: mustNewSessionlessCompactServer(t, dependencies),
		},
		{
			name:   "full stateful",
			server: mustNewFullServer(t, dependencies),
		},
		{
			name:   "full sessionless",
			server: mustNewSessionlessFullServer(t, dependencies),
		},
	}
	for _, testCase := range servers {
		t.Run(testCase.name, func(t *testing.T) {
			withMCPClientSession(t, testCase.server, func(ctx context.Context, session *sdk.ClientSession) {
				capabilities := session.InitializeResult().Capabilities
				if capabilities.Resources != nil || capabilities.Prompts != nil {
					t.Fatalf("capabilities = %#v", capabilities)
				}
				resources, err := session.ListResources(ctx, nil)
				if err != nil {
					t.Fatalf("resources/list error = %v", err)
				}
				if resources.Resources == nil || len(resources.Resources) != 0 {
					t.Fatalf("resources/list = %#v", resources)
				}
				prompts, err := session.ListPrompts(ctx, nil)
				if err != nil {
					t.Fatalf("prompts/list error = %v", err)
				}
				if prompts.Prompts == nil || len(prompts.Prompts) != 0 {
					t.Fatalf("prompts/list = %#v", prompts)
				}
			})
		})
	}
}

func newCoreTestDependencies(t *testing.T) Dependencies {
	t.Helper()
	return Dependencies{
		SearchLaws:            stubSearchLawsPort{},
		SearchLawContent:      stubSearchLawContentPort{},
		GetLaw:                stubGetLawPort{},
		CompareLawVersions:    &stubCompareLawVersionsPort{},
		GetArticle:            &recordingGetArticlePort{},
		ListLawRevisions:      &recordingListLawRevisionsPort{},
		ListLawUpdates:        &recordingListLawUpdatesPort{result: mustListLawUpdatesResult(t)},
		QueryLegalInformation: &recordingQueryLegalInformationPort{},
	}
}

func newAllTestDependencies(t *testing.T) Dependencies {
	t.Helper()
	dependencies := newCoreTestDependencies(t)
	dependencies.JudicialCases = mustJudicialCitationCasesDependencies(t)
	judicialCitations, err := NewJudicialCitationsDependencies(
		&stubTraceJudicialCitationsPort{result: mustJudicialCitationGraphResult(t)},
	)
	if err != nil {
		t.Fatalf("judicial-citations dependencies = %v", err)
	}
	dependencies.JudicialCitations = judicialCitations
	legislativeHistory, err := NewLegislativeHistoryDependencies(
		stubSearchDietSpeechesPort{},
	)
	if err != nil {
		t.Fatalf("legislative-history dependencies = %v", err)
	}
	dependencies.LegislativeHistory = legislativeHistory
	return dependencies
}

func listToolNames(t *testing.T, server *sdk.Server) []string {
	t.Helper()
	var names []string
	withMCPClientSession(t, server, func(ctx context.Context, session *sdk.ClientSession) {
		capabilities := session.InitializeResult().Capabilities
		assertInitialCapabilities(t, capabilities)
		tools, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("tools/list error = %v", err)
		}
		names = make([]string, len(tools.Tools))
		for index, tool := range tools.Tools {
			names[index] = tool.Name
			if tool.InputSchema == nil || tool.OutputSchema == nil {
				t.Fatalf("%s の schema がありません", tool.Name)
			}
		}
	})
	return names
}

func withMCPClientSession(
	t *testing.T,
	server *sdk.Server,
	operation func(context.Context, *sdk.ClientSession),
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
		t.Fatalf("MCP connect = %v", err)
	}
	operation(ctx, session)
	if err := session.Close(); err != nil {
		t.Fatalf("MCP close = %v", err)
	}
	select {
	case err := <-serverResult:
		if err != nil {
			t.Fatalf("MCP server = %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("MCP server close timeout = %v", ctx.Err())
	}
}
