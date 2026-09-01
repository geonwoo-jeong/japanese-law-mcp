package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const discoverLegalToolsToolName = "discover_legal_tools"

type discoverLegalToolsInput struct {
	query string
	limit int
}

type discoverLegalToolsOutput struct {
	TotalCount    int                   `json:"totalCount"`
	ReturnedCount int                   `json:"returnedCount"`
	OmittedCount  int                   `json:"omittedCount"`
	Truncated     bool                  `json:"truncated"`
	Tools         []discoveredLegalTool `json:"tools"`
}

type discoveredLegalTool struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	InputSchema  any    `json:"inputSchema"`
	OutputSchema any    `json:"outputSchema"`
}

func addDiscoverLegalToolsTool(
	server toolRegistrar,
	registry operationRegistry,
) {
	openWorld := false
	server.AddTool(&sdk.Tool{
		Name: discoverLegalToolsToolName,
		Description: "利用可能な専門操作を名前または説明から検索し、" +
			"各操作の正確な入出力スキーマを返します。",
		Annotations: &sdk.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &openWorld,
		},
		InputSchema:  newDiscoverLegalToolsInputSchema(),
		OutputSchema: newDiscoverLegalToolsOutputSchema(),
	}, func(
		ctx context.Context,
		request *sdk.CallToolRequest,
	) (*sdk.CallToolResult, error) {
		return callDiscoverLegalTools(ctx, registry, request)
	})
}

func callDiscoverLegalTools(
	_ context.Context,
	registry operationRegistry,
	request *sdk.CallToolRequest,
) (*sdk.CallToolResult, error) {
	if request == nil || request.Params == nil {
		return metaToolInvalidArgumentResult(newMetaToolInputError(
			"arguments",
			"は JSON object でなければなりません",
		))
	}
	input, err := decodeDiscoverLegalToolsInput(request.Params.Arguments)
	if err != nil {
		return metaToolInvalidArgumentResult(err)
	}

	matches := make([]registeredOperation, 0, len(registry.operations))
	for _, operation := range registry.operations {
		if input.query != "" && !operationMatchesQuery(operation, input.query) {
			continue
		}
		matches = append(matches, operation)
	}
	totalCount := len(matches)
	if len(matches) > input.limit {
		matches = matches[:input.limit]
	}
	tools := make([]discoveredLegalTool, 0, len(matches))
	for _, operation := range matches {
		inputSchema, cloneErr := cloneJSONValue(operation.tool.InputSchema)
		if cloneErr != nil {
			return nil, fmt.Errorf(
				"専門操作 %q の inputSchema を複製できません: %w",
				operation.tool.Name,
				cloneErr,
			)
		}
		outputSchema, cloneErr := cloneJSONValue(operation.tool.OutputSchema)
		if cloneErr != nil {
			return nil, fmt.Errorf(
				"専門操作 %q の outputSchema を複製できません: %w",
				operation.tool.Name,
				cloneErr,
			)
		}
		tools = append(tools, discoveredLegalTool{
			Name:         operation.tool.Name,
			Description:  operation.tool.Description,
			InputSchema:  inputSchema,
			OutputSchema: outputSchema,
		})
	}
	output := discoverLegalToolsOutput{
		TotalCount:    totalCount,
		ReturnedCount: len(tools),
		OmittedCount:  totalCount - len(tools),
		Truncated:     totalCount > len(tools),
		Tools:         tools,
	}
	return successMetaToolResult(output)
}

func decodeDiscoverLegalToolsInput(
	arguments json.RawMessage,
) (discoverLegalToolsInput, error) {
	fields, err := decodeStrictMetaToolObject(
		arguments,
		discoverLegalToolsMaxArgumentsBytes,
		map[string]struct{}{"query": {}, "limit": {}},
	)
	if err != nil {
		return discoverLegalToolsInput{}, err
	}
	query, exists, err := decodeMetaToolString(fields, "query", false)
	if err != nil {
		return discoverLegalToolsInput{}, err
	}
	if exists {
		query, err = validateDiscoverLegalToolsQuery(query)
		if err != nil {
			return discoverLegalToolsInput{}, err
		}
	}
	limit, err := decodeDiscoverLegalToolsLimit(fields)
	if err != nil {
		return discoverLegalToolsInput{}, err
	}
	return discoverLegalToolsInput{
		query: strings.ToLower(query),
		limit: limit,
	}, nil
}

func operationMatchesQuery(operation registeredOperation, query string) bool {
	return strings.Contains(strings.ToLower(operation.tool.Name), query) ||
		strings.Contains(strings.ToLower(operation.tool.Description), query)
}

func cloneJSONValue(value any) (any, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var cloned any
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return nil, err
	}
	return cloned, nil
}

func successMetaToolResult(output any) (*sdk.CallToolResult, error) {
	payload, err := json.Marshal(output)
	if err != nil {
		return nil, err
	}
	return &sdk.CallToolResult{
		Content: []sdk.Content{
			&sdk.TextContent{Text: string(payload)},
		},
		StructuredContent: json.RawMessage(payload),
		IsError:           false,
	}, nil
}

func newDiscoverLegalToolsInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:                 "object",
		AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		Properties: map[string]*jsonschema.Schema{
			"query": {
				Type:        "string",
				Description: "tool 名または日本語説明に対する部分一致検索。",
				MinLength:   jsonschema.Ptr(1),
				Extra: map[string]any{
					"x-maxUTF8Bytes": discoverLegalToolsMaxQueryBytes,
				},
			},
			"limit": {
				Type:    "integer",
				Minimum: jsonschema.Ptr(1.0),
				Maximum: jsonschema.Ptr(float64(discoverLegalToolsMaxLimit)),
				Default: json.RawMessage("5"),
			},
		},
		Extra: map[string]any{
			"x-maxJsonBytes": discoverLegalToolsMaxArgumentsBytes,
		},
	}
}

func newDiscoverLegalToolsOutputSchema() *jsonschema.Schema {
	count := func() *jsonschema.Schema {
		return &jsonschema.Schema{
			Type:    "integer",
			Minimum: jsonschema.Ptr(0.0),
		}
	}
	toolSchema := &jsonschema.Schema{
		Type:                 "object",
		AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		Required: []string{
			"name",
			"description",
			"inputSchema",
			"outputSchema",
		},
		Properties: map[string]*jsonschema.Schema{
			"name":         {Type: "string"},
			"description":  {Type: "string"},
			"inputSchema":  {Type: "object"},
			"outputSchema": {Type: "object"},
		},
	}
	return &jsonschema.Schema{
		Type:                 "object",
		AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		Required: []string{
			"totalCount",
			"returnedCount",
			"omittedCount",
			"truncated",
			"tools",
		},
		Properties: map[string]*jsonschema.Schema{
			"totalCount":    count(),
			"returnedCount": count(),
			"omittedCount":  count(),
			"truncated":     {Type: "boolean"},
			"tools": {
				Type:  "array",
				Items: toolSchema,
			},
		},
	}
}
