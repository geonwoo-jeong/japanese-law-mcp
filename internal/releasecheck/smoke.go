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
	smokeTimeout        = 30 * time.Second
	commandCloseTimeout = time.Second
	protocolVersion     = "2025-11-25"
)

var officialToolNames = []string{
	"get_article",
	"get_law",
	"list_law_revisions",
	"list_law_updates",
	"query_legal_information",
	"search_law_content",
	"search_laws",
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
	return verifyMCPServer(smokeCtx, binaryPath, version, environment)
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
) (returnErr error) {
	var stderr bytes.Buffer
	command := exec.CommandContext(ctx, binaryPath)
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
	if !equalStrings(names, officialToolNames) {
		return fmt.Errorf("MCP tool 一覧 = %v, want %v", names, officialToolNames)
	}
	return validateMissingInputError(ctx, session)
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
	return nil
}

func validateMissingInputError(
	ctx context.Context,
	session *sdk.ClientSession,
) error {
	result, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name:      "search_laws",
		Arguments: map[string]any{},
	})
	if err != nil {
		return fmt.Errorf("入力欠落の tools/call を実行できません: %w", err)
	}
	if !result.IsError || len(result.Content) != 1 {
		return fmt.Errorf("入力欠落の tools/call がエラー結果を返しません")
	}
	text, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		return fmt.Errorf("入力欠落の tools/call が text error を返しません")
	}
	var payload struct {
		Code string `json:"code"`
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
