package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const executeLegalToolToolName = "execute_legal_tool"

type executeLegalToolInput struct {
	toolName  string
	arguments json.RawMessage
}

func addExecuteLegalToolTool(
	server toolRegistrar,
	registry operationRegistry,
) {
	openWorld := true
	server.AddTool(&sdk.Tool{
		Name: executeLegalToolToolName,
		Description: "discover_legal_tools で確認した専門操作を、" +
			"その操作の引数と結果契約を保ったまま実行します。",
		Annotations: &sdk.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: &openWorld,
		},
		InputSchema:  newExecuteLegalToolInputSchema(),
		OutputSchema: &jsonschema.Schema{Type: "object"},
	}, func(
		ctx context.Context,
		request *sdk.CallToolRequest,
	) (*sdk.CallToolResult, error) {
		return callExecuteLegalTool(ctx, registry, request)
	})
}

func callExecuteLegalTool(
	ctx context.Context,
	registry operationRegistry,
	request *sdk.CallToolRequest,
) (*sdk.CallToolResult, error) {
	if request == nil || request.Params == nil {
		return metaToolInvalidArgumentResult(newMetaToolInputError(
			"arguments",
			"は JSON object でなければなりません",
		))
	}
	input, err := decodeExecuteLegalToolInput(request.Params.Arguments)
	if err != nil {
		return metaToolInvalidArgumentResult(err)
	}
	operation, exists := registry.lookup(input.toolName)
	if !exists {
		return metaToolInvalidArgumentResult(newMetaToolInputError(
			"toolName",
			"は利用可能な専門操作ではありません",
		))
	}

	forwarded := *request
	forwarded.Params = &sdk.CallToolParamsRaw{
		Meta:      cloneToolRequestMeta(request.Params.Meta),
		Name:      operation.tool.Name,
		Arguments: append(json.RawMessage(nil), input.arguments...),
	}
	return operation.handler(ctx, &forwarded)
}

func cloneToolRequestMeta(meta sdk.Meta) sdk.Meta {
	if meta == nil {
		return nil
	}
	cloned := make(sdk.Meta, len(meta))
	for name, value := range meta {
		cloned[name] = cloneToolRequestMetaValue(value)
	}
	return cloned
}

func cloneToolRequestMetaValue(value any) any {
	switch typed := value.(type) {
	case sdk.Meta:
		return cloneToolRequestMeta(typed)
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for name, nested := range typed {
			cloned[name] = cloneToolRequestMetaValue(nested)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for index, nested := range typed {
			cloned[index] = cloneToolRequestMetaValue(nested)
		}
		return cloned
	case json.RawMessage:
		return append(json.RawMessage(nil), typed...)
	case []byte:
		return append([]byte(nil), typed...)
	default:
		return value
	}
}

func decodeExecuteLegalToolInput(
	arguments json.RawMessage,
) (executeLegalToolInput, error) {
	fields, err := decodeStrictMetaToolObject(
		arguments,
		executeLegalToolMaxArgumentsBytes,
		map[string]struct{}{"toolName": {}, "arguments": {}},
	)
	if err != nil {
		return executeLegalToolInput{}, err
	}
	toolName, _, err := decodeMetaToolString(fields, "toolName", true)
	if err != nil {
		return executeLegalToolInput{}, err
	}
	toolName, err = validateExecuteLegalToolName(toolName)
	if err != nil {
		return executeLegalToolInput{}, err
	}
	nested, exists := fields["arguments"]
	if !exists {
		return executeLegalToolInput{}, newMetaToolInputError(
			"arguments",
			"は必須です",
		)
	}
	if queryLegalInformationJSONNull(nested) {
		return executeLegalToolInput{}, newMetaToolInputError(
			"arguments",
			"に null は使用できません",
		)
	}
	if err := validateJSONObjectWithoutDuplicateKeys(nested); err != nil {
		return executeLegalToolInput{}, err
	}
	return executeLegalToolInput{
		toolName:  toolName,
		arguments: append(json.RawMessage(nil), nested...),
	}, nil
}

func validateExecuteLegalToolName(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 || len(trimmed) > 128 {
		return "", newMetaToolInputError(
			"toolName",
			"は trim 後に UTF-8 で 1 byte 以上 128 byte 以下でなければなりません",
		)
	}
	for _, character := range trimmed {
		if character <= '\u001f' || character == '\u007f' {
			return "", newMetaToolInputError(
				"toolName",
				"に ASCII 制御文字を含めることはできません",
			)
		}
	}
	return trimmed, nil
}

func newExecuteLegalToolInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:                 "object",
		AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		Required:             []string{"toolName", "arguments"},
		Properties: map[string]*jsonschema.Schema{
			"toolName": {
				Type:      "string",
				MinLength: jsonschema.Ptr(1),
				MaxLength: jsonschema.Ptr(128),
				Extra: map[string]any{
					"x-maxUTF8Bytes": 128,
				},
			},
			"arguments": {
				Type: "object",
			},
		},
		Extra: map[string]any{
			"x-maxJsonBytes": executeLegalToolMaxArgumentsBytes,
		},
	}
}
