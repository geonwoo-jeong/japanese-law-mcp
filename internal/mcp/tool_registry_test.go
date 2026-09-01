package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestOperationRegistryRejectsInvalidDefinitionsFailClosed(t *testing.T) {
	validHandler := func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return &sdk.CallToolResult{}, nil
	}
	newTool := func() *sdk.Tool {
		return &sdk.Tool{
			Name:         "valid_operation",
			Description:  "有効なテスト用専門操作です。",
			InputSchema:  &jsonschema.Schema{Type: "object"},
			OutputSchema: &jsonschema.Schema{Type: "object"},
		}
	}
	var typedNilSchema *jsonschema.Schema
	tests := []struct {
		name    string
		tool    func() *sdk.Tool
		handler sdk.ToolHandler
	}{
		{name: "nil tool", tool: func() *sdk.Tool { return nil }, handler: validHandler},
		{name: "empty name", tool: func() *sdk.Tool { tool := newTool(); tool.Name = ""; return tool }, handler: validHandler},
		{name: "blank name", tool: func() *sdk.Tool { tool := newTool(); tool.Name = "  "; return tool }, handler: validHandler},
		{name: "invalid name", tool: func() *sdk.Tool { tool := newTool(); tool.Name = "invalid/name"; return tool }, handler: validHandler},
		{name: "too long name", tool: func() *sdk.Tool { tool := newTool(); tool.Name = strings.Repeat("a", 129); return tool }, handler: validHandler},
		{name: "reserved discover name", tool: func() *sdk.Tool { tool := newTool(); tool.Name = discoverLegalToolsToolName; return tool }, handler: validHandler},
		{name: "reserved execute name", tool: func() *sdk.Tool { tool := newTool(); tool.Name = executeLegalToolToolName; return tool }, handler: validHandler},
		{name: "empty description", tool: func() *sdk.Tool { tool := newTool(); tool.Description = " "; return tool }, handler: validHandler},
		{name: "nil handler", tool: newTool, handler: nil},
		{name: "nil input schema", tool: func() *sdk.Tool { tool := newTool(); tool.InputSchema = nil; return tool }, handler: validHandler},
		{name: "typed nil input schema", tool: func() *sdk.Tool { tool := newTool(); tool.InputSchema = typedNilSchema; return tool }, handler: validHandler},
		{name: "nil output schema", tool: func() *sdk.Tool { tool := newTool(); tool.OutputSchema = nil; return tool }, handler: validHandler},
		{name: "non-object input schema", tool: func() *sdk.Tool { tool := newTool(); tool.InputSchema = &jsonschema.Schema{Type: "array"}; return tool }, handler: validHandler},
		{name: "non-object output schema", tool: func() *sdk.Tool {
			tool := newTool()
			tool.OutputSchema = &jsonschema.Schema{Type: "array"}
			return tool
		}, handler: validHandler},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			builder := newOperationRegistryBuilder()
			builder.AddTool(testCase.tool(), testCase.handler)
			if registry, err := builder.build(); err == nil {
				t.Fatalf("registry = %#v, error = nil", registry)
			}
		})
	}

	builder := newOperationRegistryBuilder()
	builder.AddTool(newTool(), validHandler)
	builder.AddTool(newTool(), validHandler)
	if registry, err := builder.build(); err == nil {
		t.Fatalf("duplicate registry = %#v, error = nil", registry)
	}
}

func TestOperationRegistryClonesDefinitionsAtRegistrationAndPublication(t *testing.T) {
	input := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
	}
	output := map[string]any{"type": "object"}
	tool := &sdk.Tool{
		Name:         "immutable_operation",
		Description:  "変更されないテスト用専門操作です。",
		InputSchema:  input,
		OutputSchema: output,
	}
	builder := newOperationRegistryBuilder()
	builder.AddTool(tool, func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return &sdk.CallToolResult{}, nil
	})
	registry, err := builder.build()
	if err != nil {
		t.Fatalf("registry = %v", err)
	}
	builder.operations[0].tool.Description = "builder の変更"
	tool.Description = "変更後"
	input["type"] = "array"
	output["type"] = "array"
	registered, exists := registry.lookup("immutable_operation")
	if !exists {
		t.Fatal("immutable_operation がありません")
	}
	if registered.tool.Description != "変更されないテスト用専門操作です。" ||
		registered.tool.InputSchema.(map[string]any)["type"] != "object" ||
		registered.tool.OutputSchema.(map[string]any)["type"] != "object" {
		t.Fatalf("registered tool = %#v", registered.tool)
	}

	capture := &capturingToolRegistrar{}
	if err := registry.addAllTo(capture); err != nil {
		t.Fatalf("addAllTo() error = %v", err)
	}
	if len(capture.tools) != 1 || capture.tools[0] == registered.tool {
		t.Fatalf("published tools = %#v", capture.tools)
	}
	capture.tools[0].Description = "公開後の変更"
	if registered.tool.Description == capture.tools[0].Description {
		t.Fatal("公開先の変更が registry に反映されました")
	}
}

func TestOperationRegistryRejectsReadyCitationsWithoutJudicialCases(t *testing.T) {
	citations, err := NewJudicialCitationsDependencies(
		&stubTraceJudicialCitationsPort{result: mustJudicialCitationGraphResult(t)},
	)
	if err != nil {
		t.Fatalf("judicial-citations dependencies = %v", err)
	}
	if registry, err := newOperationRegistry(Dependencies{
		JudicialCitations: citations,
	}); err == nil {
		t.Fatalf("registry = %#v, error = nil", registry)
	}
}

func TestOperationRegistryRejectsPartialExtensionPackUnits(t *testing.T) {
	tests := []struct {
		name         string
		dependencies Dependencies
	}{
		{
			name: "judicial cases",
			dependencies: Dependencies{JudicialCases: JudicialCasesDependencies{
				search:      stubSearchJudicialCasesPort{},
				initialized: true,
			}},
		},
		{
			name: "judicial citations",
			dependencies: Dependencies{JudicialCitations: JudicialCitationsDependencies{
				initialized: true,
			}},
		},
		{
			name: "legislative history",
			dependencies: Dependencies{LegislativeHistory: LegislativeHistoryDependencies{
				initialized: true,
			}},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if registry, err := newOperationRegistry(testCase.dependencies); err == nil {
				t.Fatalf("registry = %#v, error = nil", registry)
			}
		})
	}
}

type capturingToolRegistrar struct {
	tools []*sdk.Tool
}

func (registrar *capturingToolRegistrar) AddTool(
	tool *sdk.Tool,
	_ sdk.ToolHandler,
) {
	registrar.tools = append(registrar.tools, tool)
}
