package mcp

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDiscoverLegalToolsReturnsEnabledSortedInventoryAndExactCounts(t *testing.T) {
	registry := mustSpecialistRegistry(t, newAllTestDependencies(t))

	all := callDiscoverLegalToolsForTest(t, registry, `{"limit":16}`)
	wantNames := []string{
		"compare_law_versions",
		"get_article",
		"get_judicial_case",
		"get_law",
		"list_law_revisions",
		"list_law_updates",
		"search_diet_speeches",
		"search_judicial_cases",
		"search_law_content",
		"search_laws",
		"trace_judicial_citations",
	}
	if got := discoveredToolNames(all.Tools); !reflect.DeepEqual(got, wantNames) {
		t.Fatalf("discovered tools = %#v, want %#v", got, wantNames)
	}
	if all.TotalCount != 11 || all.ReturnedCount != 11 ||
		all.OmittedCount != 0 || all.Truncated {
		t.Fatalf("all counts = %#v", all)
	}
	for _, tool := range all.Tools {
		if tool.Description == "" || tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatalf("discovered tool = %#v", tool)
		}
	}

	limited := callDiscoverLegalToolsForTest(t, registry, `{}`)
	if limited.TotalCount != 11 || limited.ReturnedCount != 5 ||
		limited.OmittedCount != 6 || !limited.Truncated || len(limited.Tools) != 5 {
		t.Fatalf("default limit result = %#v", limited)
	}

	filtered := callDiscoverLegalToolsForTest(
		t,
		registry,
		`{"query":"  JUDICIAL  ","limit":16}`,
	)
	wantFiltered := []string{
		"get_judicial_case",
		"search_judicial_cases",
		"trace_judicial_citations",
	}
	if got := discoveredToolNames(filtered.Tools); !reflect.DeepEqual(got, wantFiltered) {
		t.Fatalf("filtered tools = %#v, want %#v", got, wantFiltered)
	}
	if filtered.TotalCount != 3 || filtered.ReturnedCount != 3 ||
		filtered.OmittedCount != 0 || filtered.Truncated {
		t.Fatalf("filtered counts = %#v", filtered)
	}

	noMatches := callDiscoverLegalToolsForTest(
		t,
		registry,
		`{"query":"該当しない操作名","limit":16}`,
	)
	if noMatches.TotalCount != 0 || noMatches.ReturnedCount != 0 ||
		noMatches.OmittedCount != 0 || noMatches.Truncated || len(noMatches.Tools) != 0 {
		t.Fatalf("no-match result = %#v", noMatches)
	}
}

func TestDiscoverLegalToolsExcludesQueryAndDisabledPackOperations(t *testing.T) {
	registry := mustSpecialistRegistry(t, newCoreTestDependencies(t))
	output := callDiscoverLegalToolsForTest(t, registry, `{"limit":16}`)
	want := []string{
		"compare_law_versions",
		"get_article",
		"get_law",
		"list_law_revisions",
		"list_law_updates",
		"search_law_content",
		"search_laws",
	}
	if got := discoveredToolNames(output.Tools); !reflect.DeepEqual(got, want) {
		t.Fatalf("core operations = %#v, want %#v", got, want)
	}
	if output.TotalCount != 7 || output.ReturnedCount != 7 {
		t.Fatalf("core counts = %#v", output)
	}
}

func TestDiscoverLegalToolsReturnsIndependentSchemaClones(t *testing.T) {
	registry := mustSpecialistRegistry(t, newCoreTestDependencies(t))
	first := callDiscoverLegalToolsForTest(t, registry, `{"query":"search_laws"}`)
	if len(first.Tools) != 1 {
		t.Fatalf("first tools = %#v", first.Tools)
	}
	input, ok := first.Tools[0].InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("inputSchema type = %T", first.Tools[0].InputSchema)
	}
	input["mutated"] = true
	output, ok := first.Tools[0].OutputSchema.(map[string]any)
	if !ok {
		t.Fatalf("outputSchema type = %T", first.Tools[0].OutputSchema)
	}
	output["mutated"] = true

	second := callDiscoverLegalToolsForTest(t, registry, `{"query":"search_laws"}`)
	secondInput := second.Tools[0].InputSchema.(map[string]any)
	secondOutput := second.Tools[0].OutputSchema.(map[string]any)
	if _, exists := secondInput["mutated"]; exists {
		t.Fatal("inputSchema の変更が registry に反映されました")
	}
	if _, exists := secondOutput["mutated"]; exists {
		t.Fatal("outputSchema の変更が registry に反映されました")
	}

	registered, exists := registry.lookup("search_laws")
	if !exists {
		t.Fatal("search_laws が registry にありません")
	}
	wantInput, _ := json.Marshal(registered.tool.InputSchema)
	wantOutput, _ := json.Marshal(registered.tool.OutputSchema)
	gotInput, _ := json.Marshal(second.Tools[0].InputSchema)
	gotOutput, _ := json.Marshal(second.Tools[0].OutputSchema)
	if string(gotInput) != string(wantInput) || string(gotOutput) != string(wantOutput) {
		t.Fatal("discover の schema が登録済み schema と一致しません")
	}
}

func TestDiscoverLegalToolsStrictInputBoundaries(t *testing.T) {
	registry := mustSpecialistRegistry(t, newCoreTestDependencies(t))
	invalidUTF8 := append(
		[]byte(`{"query":"`),
		append([]byte{0xff}, []byte(`"}`)...)...,
	)
	tests := []struct {
		name      string
		arguments []byte
		field     string
	}{
		{name: "non-object", arguments: []byte(`[]`), field: "arguments"},
		{name: "trailing JSON", arguments: []byte(`{} {}`), field: "arguments"},
		{name: "unknown field", arguments: []byte(`{"unknown":true}`), field: "arguments"},
		{name: "duplicate query", arguments: []byte(`{"query":"law","query":"law"}`), field: "arguments"},
		{name: "query null", arguments: []byte(`{"query":null}`), field: "query"},
		{name: "query type", arguments: []byte(`{"query":1}`), field: "query"},
		{name: "query empty", arguments: []byte(`{"query":"  "}`), field: "query"},
		{name: "query too large", arguments: []byte(`{"query":"` + strings.Repeat("a", 257) + `"}`), field: "query"},
		{name: "query control", arguments: []byte(`{"query":"law\u0000"}`), field: "query"},
		{name: "limit null", arguments: []byte(`{"limit":null}`), field: "limit"},
		{name: "limit zero", arguments: []byte(`{"limit":0}`), field: "limit"},
		{name: "limit too large", arguments: []byte(`{"limit":17}`), field: "limit"},
		{name: "limit fraction", arguments: []byte(`{"limit":1.5}`), field: "limit"},
		{name: "invalid UTF-8", arguments: invalidUTF8, field: "arguments"},
		{name: "high surrogate", arguments: []byte(`{"query":"\uD800"}`), field: "arguments"},
		{name: "low surrogate", arguments: []byte(`{"query":"\uDC00"}`), field: "arguments"},
		{
			name:      "arguments too large",
			arguments: []byte(`{"query":"` + strings.Repeat("a", discoverLegalToolsMaxArgumentsBytes) + `"}`),
			field:     "arguments",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := callDiscoverLegalTools(
				context.Background(),
				registry,
				&sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{
					Name:      discoverLegalToolsToolName,
					Arguments: testCase.arguments,
				}},
			)
			if err != nil {
				t.Fatalf("callDiscoverLegalTools() error = %v", err)
			}
			assertMetaInvalidArgumentField(t, result, testCase.field)
		})
	}

	validEscapedLiteral := callDiscoverLegalToolsForTest(
		t,
		registry,
		`{"query":"\\uD800","limit":16}`,
	)
	if validEscapedLiteral.TotalCount != 0 {
		t.Fatalf("escaped literal result = %#v", validEscapedLiteral)
	}
}

func TestCompactMetaToolAnnotationsAreAccurate(t *testing.T) {
	server := mustNewCompactServer(t, newCoreTestDependencies(t))
	withMCPClientSession(t, server, func(ctx context.Context, session *sdk.ClientSession) {
		tools, err := session.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("tools/list error = %v", err)
		}
		annotations := make(map[string]*sdk.ToolAnnotations)
		for _, tool := range tools.Tools {
			annotations[tool.Name] = tool.Annotations
		}
		discover := annotations[discoverLegalToolsToolName]
		if discover == nil || !discover.ReadOnlyHint ||
			discover.OpenWorldHint == nil || *discover.OpenWorldHint {
			t.Fatalf("discover annotations = %#v", discover)
		}
		execute := annotations[executeLegalToolToolName]
		if execute == nil || !execute.ReadOnlyHint ||
			execute.OpenWorldHint == nil || !*execute.OpenWorldHint {
			t.Fatalf("execute annotations = %#v", execute)
		}
	})
}

func callDiscoverLegalToolsForTest(
	t *testing.T,
	registry operationRegistry,
	arguments string,
) discoverLegalToolsOutput {
	t.Helper()
	result, err := callDiscoverLegalTools(
		context.Background(),
		registry,
		&sdk.CallToolRequest{Params: &sdk.CallToolParamsRaw{
			Name:      discoverLegalToolsToolName,
			Arguments: json.RawMessage(arguments),
		}},
	)
	if err != nil {
		t.Fatalf("callDiscoverLegalTools() error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("discover result = %#v", result)
	}
	payload, ok := result.StructuredContent.(json.RawMessage)
	if !ok {
		t.Fatalf("structuredContent type = %T", result.StructuredContent)
	}
	var output discoverLegalToolsOutput
	if err := json.Unmarshal(payload, &output); err != nil {
		t.Fatalf("discover output decode = %v", err)
	}
	return output
}

func discoveredToolNames(tools []discoveredLegalTool) []string {
	names := make([]string, len(tools))
	for index, tool := range tools {
		names[index] = tool.Name
	}
	return names
}

func mustSpecialistRegistry(
	t *testing.T,
	dependencies Dependencies,
) operationRegistry {
	t.Helper()
	registry, err := newOperationRegistry(dependencies)
	if err != nil {
		t.Fatalf("operation registry = %v", err)
	}
	return registry.specialists()
}
