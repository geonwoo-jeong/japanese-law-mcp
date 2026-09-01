package mcp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const queryLegalInformationToolName = "query_legal_information"

var requiredPublicOperationNames = [...]string{
	"compare_law_versions",
	"get_article",
	"get_law",
	"list_law_revisions",
	"list_law_updates",
	queryLegalInformationToolName,
	"search_law_content",
	"search_laws",
}

// toolRegistrar は、公開サーバーと起動時レジストリに共通するツール登録境界である。
type toolRegistrar interface {
	AddTool(*sdk.Tool, sdk.ToolHandler)
}

type registeredOperation struct {
	tool    *sdk.Tool
	handler sdk.ToolHandler
}

// operationRegistry は、起動時に確定した専門操作を名前順で保持する。
// 作成後は内部の slice、map および Tool を変更しない。
type operationRegistry struct {
	operations []registeredOperation
	byName     map[string]registeredOperation
}

type operationRegistryBuilder struct {
	operations []registeredOperation
	names      map[string]struct{}
	err        error
}

func newOperationRegistryBuilder() *operationRegistryBuilder {
	return &operationRegistryBuilder{names: make(map[string]struct{})}
}

func (builder *operationRegistryBuilder) AddTool(
	tool *sdk.Tool,
	handler sdk.ToolHandler,
) {
	if builder.err != nil {
		return
	}
	if tool == nil {
		builder.err = fmt.Errorf("専門操作の tool がありません")
		return
	}
	if strings.TrimSpace(tool.Name) == "" {
		builder.err = fmt.Errorf("専門操作の tool 名がありません")
		return
	}
	if !validRegisteredToolName(tool.Name) {
		builder.err = fmt.Errorf("専門操作の tool 名 %q が有効ではありません", tool.Name)
		return
	}
	if tool.Name == discoverLegalToolsToolName || tool.Name == executeLegalToolToolName {
		builder.err = fmt.Errorf("専門操作の tool 名 %q は予約されています", tool.Name)
		return
	}
	if strings.TrimSpace(tool.Description) == "" {
		builder.err = fmt.Errorf("専門操作 %q の説明がありません", tool.Name)
		return
	}
	if handler == nil {
		builder.err = fmt.Errorf("専門操作 %q の handler がありません", tool.Name)
		return
	}
	if tool.InputSchema == nil {
		builder.err = fmt.Errorf("専門操作 %q の inputSchema がありません", tool.Name)
		return
	}
	if tool.OutputSchema == nil {
		builder.err = fmt.Errorf("専門操作 %q の outputSchema がありません", tool.Name)
		return
	}
	if _, duplicate := builder.names[tool.Name]; duplicate {
		builder.err = fmt.Errorf("専門操作 %q が重複しています", tool.Name)
		return
	}
	cloned, err := cloneToolDefinition(tool)
	if err != nil {
		builder.err = fmt.Errorf("専門操作 %q を固定できません: %w", tool.Name, err)
		return
	}
	if cloned.InputSchema == nil || cloned.OutputSchema == nil {
		builder.err = fmt.Errorf("専門操作 %q の schema が null です", tool.Name)
		return
	}
	if !isObjectSchema(cloned.InputSchema) || !isObjectSchema(cloned.OutputSchema) {
		builder.err = fmt.Errorf("専門操作 %q の schema は object でなければなりません", tool.Name)
		return
	}
	builder.names[tool.Name] = struct{}{}
	builder.operations = append(builder.operations, registeredOperation{
		tool:    cloned,
		handler: handler,
	})
}

func (builder *operationRegistryBuilder) build() (operationRegistry, error) {
	if builder.err != nil {
		return operationRegistry{}, builder.err
	}
	operations := make([]registeredOperation, 0, len(builder.operations))
	for _, operation := range builder.operations {
		tool, err := cloneToolDefinition(operation.tool)
		if err != nil {
			return operationRegistry{}, fmt.Errorf(
				"専門操作 %q の snapshot を固定できません: %w",
				operation.tool.Name,
				err,
			)
		}
		operations = append(operations, registeredOperation{
			tool:    tool,
			handler: operation.handler,
		})
	}
	sort.Slice(operations, func(left, right int) bool {
		return operations[left].tool.Name < operations[right].tool.Name
	})
	byName := make(map[string]registeredOperation, len(operations))
	for _, operation := range operations {
		byName[operation.tool.Name] = operation
	}
	return operationRegistry{
		operations: operations,
		byName:     byName,
	}, nil
}

func newOperationRegistry(
	dependencies Dependencies,
) (operationRegistry, error) {
	if dependencies.JudicialCases.configured() && !dependencies.JudicialCases.ready() {
		return operationRegistry{}, fmt.Errorf(
			"judicial-cases の専門操作が完全ではありません",
		)
	}
	if dependencies.JudicialCitations.configured() && !dependencies.JudicialCitations.ready() {
		return operationRegistry{}, fmt.Errorf(
			"judicial-citations の専門操作が完全ではありません",
		)
	}
	if dependencies.LegislativeHistory.configured() && !dependencies.LegislativeHistory.ready() {
		return operationRegistry{}, fmt.Errorf(
			"legislative-history の専門操作が完全ではありません",
		)
	}
	if dependencies.JudicialCitations.ready() && !dependencies.JudicialCases.ready() {
		return operationRegistry{}, fmt.Errorf(
			"judicial-citations には有効な judicial-cases が必要です",
		)
	}
	builder := newOperationRegistryBuilder()
	if dependencies.SearchLaws != nil {
		addSearchLawsTool(builder, dependencies.SearchLaws)
	}
	if dependencies.SearchLawContent != nil {
		addSearchLawContentTool(builder, dependencies.SearchLawContent)
	}
	if dependencies.GetLaw != nil {
		addGetLawTool(builder, dependencies.GetLaw)
	}
	if !isNilCompareLawVersionsPort(dependencies.CompareLawVersions) {
		addCompareLawVersionsTool(builder, dependencies.CompareLawVersions)
	}
	if dependencies.GetArticle != nil {
		addGetArticleTool(builder, dependencies.GetArticle)
	}
	if !isNilListLawRevisionsPort(dependencies.ListLawRevisions) {
		addListLawRevisionsTool(builder, dependencies.ListLawRevisions)
	}
	if dependencies.ListLawUpdates != nil {
		addListLawUpdatesTool(builder, dependencies.ListLawUpdates)
	}
	if !isNilQueryLegalInformationPort(dependencies.QueryLegalInformation) {
		addQueryLegalInformationTool(builder, dependencies.QueryLegalInformation)
	}
	dependencies.JudicialCases.addTools(builder)
	if dependencies.JudicialCases.ready() {
		dependencies.JudicialCitations.addTools(builder)
	}
	dependencies.LegislativeHistory.addTools(builder)
	return builder.build()
}

func validateRequiredPublicOperations(registry operationRegistry) error {
	missing := make([]string, 0, len(requiredPublicOperationNames))
	for _, name := range requiredPublicOperationNames {
		if _, exists := registry.lookup(name); !exists {
			missing = append(missing, name)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf(
			"公開 MCP サーバーに必要な専門操作がありません: %s",
			strings.Join(missing, ", "),
		)
	}
	return nil
}

func validRegisteredToolName(name string) bool {
	if len(name) == 0 || len(name) > 128 {
		return false
	}
	for _, character := range name {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '_' || character == '-' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func isObjectSchema(schema any) bool {
	object, ok := schema.(map[string]any)
	if !ok {
		return false
	}
	typeName, ok := object["type"].(string)
	return ok && typeName == "object"
}

func (registry operationRegistry) specialists() operationRegistry {
	operations := make([]registeredOperation, 0, len(registry.operations))
	byName := make(map[string]registeredOperation, len(registry.byName))
	for _, operation := range registry.operations {
		if operation.tool.Name == queryLegalInformationToolName {
			continue
		}
		operations = append(operations, operation)
		byName[operation.tool.Name] = operation
	}
	return operationRegistry{operations: operations, byName: byName}
}

func (registry operationRegistry) lookup(name string) (registeredOperation, bool) {
	operation, exists := registry.byName[name]
	return operation, exists
}

func (registry operationRegistry) addAllTo(registrar toolRegistrar) error {
	for _, operation := range registry.operations {
		tool, err := cloneToolDefinition(operation.tool)
		if err != nil {
			return fmt.Errorf("専門操作 %q を登録できません: %w", operation.tool.Name, err)
		}
		registrar.AddTool(tool, operation.handler)
	}
	return nil
}

func cloneToolDefinition(tool *sdk.Tool) (*sdk.Tool, error) {
	payload, err := json.Marshal(tool)
	if err != nil {
		return nil, err
	}
	var cloned sdk.Tool
	if err := json.Unmarshal(payload, &cloned); err != nil {
		return nil, err
	}
	return &cloned, nil
}
