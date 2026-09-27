package qualitygate

import (
	"path/filepath"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/sdkpatchcheck"
)

func sdkPatchCheckStep(snapshot string) step {
	return commandStep(
		"sdk-patch-check",
		"MCP SDK の保存内容と修正差分",
		"SOT-ENG-052",
		goCommand(snapshot, true, "run", "./cmd/sdk-patch-check", "--repository=."),
	)
}

func sdkJSONTestStep(snapshot string) step {
	return commandStep(
		"sdk-json-test",
		"MCP SDK の JSON 互換性",
		"SOT-ENG-052",
		goCommand(
			filepath.Join(snapshot, "third_party", "modelcontextprotocol-go-sdk"),
			true,
			"test", "-p=1", "-count=1", "./internal/json",
		),
	)
}

func sdkUpstreamVulnerabilityStep(snapshot string) step {
	command := goToolCommand(snapshot, true, []string{
		"tool", "-modfile=tools/go.mod", "govulncheck", "-mode=query", "-format=json",
		sdkpatchcheck.ModulePath + "@" + sdkpatchcheck.UpstreamVersion,
	})
	command.validateOutput = requireNoSDKUpstreamVulnerabilities
	return commandStep(
		"sdk-upstream-vulnerabilities",
		"MCP SDK 元版の既知の脆弱性照会",
		"SOT-ENG-052",
		command,
	)
}
