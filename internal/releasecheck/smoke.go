package releasecheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	smokeTimeout        = 60 * time.Second
	commandCloseTimeout = time.Second
	protocolVersion     = "2025-11-25"
)

var compactToolNames = []string{
	"discover_legal_tools",
	"execute_legal_tool",
	"query_legal_information",
}

var coreSpecialistToolNames = []string{
	"compare_law_versions",
	"get_article",
	"get_law",
	"list_law_revisions",
	"list_law_updates",
	"search_law_content",
	"search_laws",
}

type smokeScenario struct {
	name                string
	configYAML          string
	toolNames           []string
	specialistToolNames []string
	compact             bool
}

func smokeTarget(
	ctx context.Context,
	archivePath string,
	target releaseTarget,
	version string,
) (returnErr error) {
	smokeCtx, cancel := context.WithTimeout(ctx, smokeTimeout)
	defer cancel()

	tempDirectory, err := os.MkdirTemp("", projectName+"-release-check-")
	if err != nil {
		return fmt.Errorf("一時 directory を作成できません: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(tempDirectory); cleanupErr != nil {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("一時 directory を削除できません: %w", cleanupErr),
			)
		}
	}()

	binaryDirectory := filepath.Join(tempDirectory, "bin")
	configDirectory := filepath.Join(tempDirectory, "config")
	homeDirectory := filepath.Join(tempDirectory, "home")
	for _, directory := range []string{
		binaryDirectory,
		configDirectory,
		homeDirectory,
	} {
		if err := os.Mkdir(directory, 0o700); err != nil {
			return fmt.Errorf("隔離 directory を作成できません: %w", err)
		}
	}
	binaryPath, err := extractArchiveBinary(
		archivePath,
		target,
		binaryDirectory,
	)
	if err != nil {
		return fmt.Errorf("配布アーカイブを安全に展開できません: %w", err)
	}
	environment := isolatedEnvironment(
		os.Environ(),
		homeDirectory,
		configDirectory,
	)
	if err := verifyBinaryVersion(smokeCtx, binaryPath, version, environment); err != nil {
		return err
	}
	for _, scenario := range smokeScenarios() {
		configPath, err := writeSmokeConfig(
			configDirectory,
			scenario.name,
			scenario.configYAML,
		)
		if err != nil {
			return fmt.Errorf(
				"smoke scenario %q の設定を作成できません: %w",
				scenario.name,
				err,
			)
		}
		if err := verifyMCPServer(
			smokeCtx,
			binaryPath,
			version,
			environment,
			scenario,
			configPath,
		); err != nil {
			return fmt.Errorf("smoke scenario %q: %w", scenario.name, err)
		}
	}
	return nil
}

func verifyBinaryVersion(
	ctx context.Context,
	binaryPath, version string,
	environment []string,
) error {
	command := exec.CommandContext(ctx, binaryPath, "version")
	command.Env = append([]string(nil), environment...)
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return fmt.Errorf(
				"version コマンドが失敗しました: %w: %s",
				err,
				strings.TrimSpace(string(exitError.Stderr)),
			)
		}
		return fmt.Errorf("version コマンドを実行できません: %w", err)
	}
	expected := projectName + " " + version + "\n"
	if string(output) != expected {
		return fmt.Errorf(
			"version 出力 = %q, want %q",
			string(output),
			expected,
		)
	}
	return nil
}

func verifyMCPServer(
	ctx context.Context,
	binaryPath, version string,
	environment []string,
	scenario smokeScenario,
	configPath string,
) (returnErr error) {
	var stderr bytes.Buffer
	command := exec.CommandContext(ctx, binaryPath)
	if configPath != "" {
		command.Args = append(command.Args, "--config="+configPath)
	}
	command.Env = append([]string(nil), environment...)
	command.Stderr = &stderr
	transport := &sdk.CommandTransport{
		Command:           command,
		TerminateDuration: commandCloseTimeout,
	}
	client := sdk.NewClient(
		&sdk.Implementation{Name: "release-check", Version: version},
		nil,
	)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf(
			"MCP を初期化できません: %w: %s",
			err,
			strings.TrimSpace(stderr.String()),
		)
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			returnErr = errors.Join(
				returnErr,
				fmt.Errorf("MCP session を終了できません: %w", closeErr),
			)
		}
	}()

	if err := validateInitializeResult(session.InitializeResult(), version); err != nil {
		return err
	}
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("MCP tool 一覧を取得できません: %w", err)
	}
	names := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	if !equalStrings(names, scenario.toolNames) {
		return fmt.Errorf("MCP tool 一覧 = %v, want %v", names, scenario.toolNames)
	}
	if scenario.compact {
		if err := validateDiscoveredSpecialistTools(
			ctx,
			session,
			scenario.specialistToolNames,
		); err != nil {
			return err
		}
	}
	return validateMissingInputError(ctx, session, scenario.compact)
}

func smokeScenarios() []smokeScenario {
	return []smokeScenario{
		newCompactSmokeScenario("default", ""),
		newCompactSmokeScenario(
			"judicial-cases",
			"extensionPacks:\n"+
				"  judicial-cases:\n"+
				"    enabled: true\n",
			"get_judicial_case",
			"search_judicial_cases",
		),
		newCompactSmokeScenario(
			"legislative-history",
			"extensionPacks:\n"+
				"  legislative-history:\n"+
				"    enabled: true\n",
			"search_diet_speeches",
		),
		newCompactSmokeScenario(
			"judicial-cases-and-legislative-history",
			"extensionPacks:\n"+
				"  judicial-cases:\n"+
				"    enabled: true\n"+
				"  legislative-history:\n"+
				"    enabled: true\n",
			"get_judicial_case",
			"search_diet_speeches",
			"search_judicial_cases",
		),
		newCompactSmokeScenario(
			"judicial-citations",
			"extensionPacks:\n"+
				"  judicial-cases:\n"+
				"    enabled: true\n"+
				"  judicial-citations:\n"+
				"    enabled: true\n",
			"get_judicial_case",
			"search_judicial_cases",
			"trace_judicial_citations",
		),
		newCompactSmokeScenario(
			"all-extension-packs",
			"extensionPacks:\n"+
				"  judicial-cases:\n"+
				"    enabled: true\n"+
				"  judicial-citations:\n"+
				"    enabled: true\n"+
				"  legislative-history:\n"+
				"    enabled: true\n",
			"get_judicial_case",
			"search_diet_speeches",
			"search_judicial_cases",
			"trace_judicial_citations",
		),
		newFullSmokeScenario("full-default", ""),
		newFullSmokeScenario(
			"full-all-extension-packs",
			"extensionPacks:\n"+
				"  judicial-cases:\n"+
				"    enabled: true\n"+
				"  judicial-citations:\n"+
				"    enabled: true\n"+
				"  legislative-history:\n"+
				"    enabled: true\n",
			"get_judicial_case",
			"search_diet_speeches",
			"search_judicial_cases",
			"trace_judicial_citations",
		),
	}
}

func newCompactSmokeScenario(
	name, configYAML string,
	extraSpecialistToolNames ...string,
) smokeScenario {
	specialistToolNames := sortedToolNames(
		coreSpecialistToolNames,
		extraSpecialistToolNames,
	)
	return smokeScenario{
		name:                name,
		configYAML:          configYAML,
		toolNames:           append([]string(nil), compactToolNames...),
		specialistToolNames: specialistToolNames,
		compact:             true,
	}
}

func newFullSmokeScenario(
	name, configYAML string,
	extraSpecialistToolNames ...string,
) smokeScenario {
	specialistToolNames := sortedToolNames(
		coreSpecialistToolNames,
		extraSpecialistToolNames,
	)
	return smokeScenario{
		name:       name,
		configYAML: "toolExposure: full\n" + configYAML,
		toolNames: sortedToolNames(
			specialistToolNames,
			[]string{"query_legal_information"},
		),
	}
}

func sortedToolNames(base, extra []string) []string {
	names := make([]string, 0, len(base)+len(extra))
	names = append(names, base...)
	names = append(names, extra...)
	sort.Strings(names)
	return names
}

func writeSmokeConfig(
	configDirectory, scenarioName, configYAML string,
) (string, error) {
	if strings.TrimSpace(configYAML) == "" {
		return "", nil
	}
	if err := os.MkdirAll(configDirectory, 0o700); err != nil {
		return "", err
	}
	configPath := filepath.Join(
		configDirectory,
		"config-"+scenarioName+".yaml",
	)
	if err := os.WriteFile(configPath, []byte(configYAML), 0o600); err != nil {
		return "", err
	}
	return configPath, nil
}

func validateInitializeResult(result *sdk.InitializeResult, version string) error {
	if result == nil {
		return fmt.Errorf("MCP 初期化結果がありません")
	}
	if result.ProtocolVersion != protocolVersion {
		return fmt.Errorf(
			"MCP protocol version = %q, want %q",
			result.ProtocolVersion,
			protocolVersion,
		)
	}
	if result.ServerInfo == nil ||
		result.ServerInfo.Name != projectName ||
		result.ServerInfo.Version != version {
		return fmt.Errorf("MCP server 情報が配布物と一致しません")
	}
	if result.Capabilities == nil || result.Capabilities.Tools == nil {
		return fmt.Errorf("MCP tools capability がありません")
	}
	if result.Capabilities.Resources != nil {
		return fmt.Errorf("未提供の MCP resources capability があります")
	}
	if result.Capabilities.Prompts != nil {
		return fmt.Errorf("未提供の MCP prompts capability があります")
	}
	return nil
}

func validateDiscoveredSpecialistTools(
	ctx context.Context,
	session *sdk.ClientSession,
	wantNames []string,
) error {
	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "discover_legal_tools",
		Arguments: map[string]any{
			"limit": 16,
		},
	})
	if err != nil {
		return fmt.Errorf("専門 tool の探索を実行できません: %w", err)
	}
	if result == nil || result.IsError {
		return fmt.Errorf("専門 tool の探索が成功結果を返しません")
	}
	structuredJSON, err := json.Marshal(result.StructuredContent)
	if err != nil {
		return fmt.Errorf("専門 tool の探索結果を読み取れません: %w", err)
	}
	var payload struct {
		TotalCount    int  `json:"totalCount"`
		ReturnedCount int  `json:"returnedCount"`
		OmittedCount  int  `json:"omittedCount"`
		Truncated     bool `json:"truncated"`
		Tools         []struct {
			Name         string          `json:"name"`
			Description  string          `json:"description"`
			InputSchema  json.RawMessage `json:"inputSchema"`
			OutputSchema json.RawMessage `json:"outputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(structuredJSON, &payload); err != nil {
		return fmt.Errorf("専門 tool の探索結果を解析できません: %w", err)
	}
	if payload.TotalCount != len(wantNames) ||
		payload.ReturnedCount != len(wantNames) ||
		payload.OmittedCount != 0 ||
		payload.Truncated {
		return fmt.Errorf(
			"専門 tool の探索件数 = total:%d returned:%d omitted:%d truncated:%t, want total:%d returned:%d omitted:0 truncated:false",
			payload.TotalCount,
			payload.ReturnedCount,
			payload.OmittedCount,
			payload.Truncated,
			len(wantNames),
			len(wantNames),
		)
	}
	gotNames := make([]string, 0, len(payload.Tools))
	for _, tool := range payload.Tools {
		gotNames = append(gotNames, tool.Name)
		if strings.TrimSpace(tool.Description) == "" {
			return fmt.Errorf("探索した専門 tool %q に説明がありません", tool.Name)
		}
		if !isJSONObject(tool.InputSchema) || !isJSONObject(tool.OutputSchema) {
			return fmt.Errorf("探索した専門 tool %q の schema が不正です", tool.Name)
		}
	}
	if !equalStrings(gotNames, wantNames) {
		return fmt.Errorf(
			"探索した専門 tool 一覧 = %v, want %v",
			gotNames,
			wantNames,
		)
	}
	return nil
}

func isJSONObject(value json.RawMessage) bool {
	var object map[string]json.RawMessage
	return len(value) != 0 && json.Unmarshal(value, &object) == nil && object != nil
}

func validateMissingInputError(
	ctx context.Context,
	session *sdk.ClientSession,
	compact bool,
) error {
	toolName := "search_laws"
	arguments := map[string]any{}
	if compact {
		toolName = "execute_legal_tool"
		arguments = map[string]any{
			"toolName":  "search_laws",
			"arguments": map[string]any{},
		}
	}
	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      toolName,
		Arguments: arguments,
	})
	if err != nil {
		return fmt.Errorf("入力欠落の tools/call を実行できません: %w", err)
	}
	if result == nil || !result.IsError || len(result.Content) != 1 {
		return fmt.Errorf("入力欠落の tools/call がエラー結果を返しません")
	}
	if result.StructuredContent != nil {
		return fmt.Errorf("入力欠落の tools/call に structuredContent があります")
	}
	text, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		return fmt.Errorf("入力欠落の tools/call が text error を返しません")
	}
	var payload struct {
		Code      string `json:"code"`
		Retryable bool   `json:"retryable"`
		Details   struct {
			Field  string `json:"field"`
			Reason string `json:"reason"`
		} `json:"details"`
	}
	if err := json.Unmarshal([]byte(text.Text), &payload); err != nil {
		return fmt.Errorf("入力欠落の error 結果を解析できません: %w", err)
	}
	if payload.Code != "invalid_argument" {
		return fmt.Errorf(
			"入力欠落の error code = %q, want %q",
			payload.Code,
			"invalid_argument",
		)
	}
	if payload.Retryable ||
		payload.Details.Field != "arguments" ||
		!strings.Contains(payload.Details.Reason, "query") {
		return fmt.Errorf(
			"入力欠落の error details = retryable:%t field:%q reason:%q",
			payload.Retryable,
			payload.Details.Field,
			payload.Details.Reason,
		)
	}
	return nil
}

func isolatedEnvironment(base []string, home, config string) []string {
	result := make([]string, 0, len(base)+5)
	for _, entry := range append([]string(nil), base...) {
		key, _, _ := strings.Cut(entry, "=")
		upperKey := strings.ToUpper(key)
		if strings.HasPrefix(upperKey, "JAPANESE_LAW_MCP_") ||
			isUserDirectoryVariable(upperKey) {
			continue
		}
		result = append(result, entry)
	}
	return append(result,
		"HOME="+home,
		"USERPROFILE="+home,
		"XDG_CONFIG_HOME="+config,
		"APPDATA="+config,
		"LOCALAPPDATA="+config,
	)
}

func isUserDirectoryVariable(key string) bool {
	switch key {
	case "HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA":
		return true
	default:
		return false
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
