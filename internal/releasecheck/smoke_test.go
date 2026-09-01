package releasecheck

import (
	"slices"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestValidateInitializeResult(t *testing.T) {
	t.Parallel()

	valid := &sdk.InitializeResult{
		ProtocolVersion: protocolVersion,
		ServerInfo: &sdk.Implementation{
			Name:    projectName,
			Version: "1.2.3",
		},
		Capabilities: &sdk.ServerCapabilities{
			Tools: &sdk.ToolCapabilities{},
		},
	}
	tests := map[string]struct {
		result  *sdk.InitializeResult
		wantErr string
	}{
		"有効": {
			result: valid,
		},
		"結果なし": {
			wantErr: "初期化結果",
		},
		"protocol 不一致": {
			result: &sdk.InitializeResult{
				ProtocolVersion: "old",
			},
			wantErr: "protocol version",
		},
		"server 情報不一致": {
			result: &sdk.InitializeResult{
				ProtocolVersion: protocolVersion,
				ServerInfo: &sdk.Implementation{
					Name: "other", Version: "1.2.3",
				},
			},
			wantErr: "server 情報",
		},
		"tools capability なし": {
			result: &sdk.InitializeResult{
				ProtocolVersion: protocolVersion,
				ServerInfo: &sdk.Implementation{
					Name: projectName, Version: "1.2.3",
				},
				Capabilities: &sdk.ServerCapabilities{},
			},
			wantErr: "tools capability",
		},
		"resources capability あり": {
			result: &sdk.InitializeResult{
				ProtocolVersion: protocolVersion,
				ServerInfo: &sdk.Implementation{
					Name: projectName, Version: "1.2.3",
				},
				Capabilities: &sdk.ServerCapabilities{
					Tools:     &sdk.ToolCapabilities{},
					Resources: &sdk.ResourceCapabilities{},
				},
			},
			wantErr: "resources capability",
		},
		"prompts capability あり": {
			result: &sdk.InitializeResult{
				ProtocolVersion: protocolVersion,
				ServerInfo: &sdk.Implementation{
					Name: projectName, Version: "1.2.3",
				},
				Capabilities: &sdk.ServerCapabilities{
					Tools:   &sdk.ToolCapabilities{},
					Prompts: &sdk.PromptCapabilities{},
				},
			},
			wantErr: "prompts capability",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := validateInitializeResult(test.result, "1.2.3")
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("validateInitializeResult() のエラー = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("validateInitializeResult() のエラー = %v", err)
			}
		})
	}
}

func TestIsolatedEnvironment(t *testing.T) {
	t.Parallel()

	got := isolatedEnvironment([]string{
		"PATH=/bin",
		"HOME=/real",
		"JAPANESE_LAW_MCP_TRANSPORT=streamable-http",
		"OTHER=value",
	}, "/temporary/home", "/temporary/config")
	joined := strings.Join(got, "\n")
	for _, unwanted := range []string{"/real", "streamable-http"} {
		if strings.Contains(joined, unwanted) {
			t.Fatalf("隔離後の環境に %q が残っています: %s", unwanted, joined)
		}
	}
	for _, required := range []string{
		"PATH=/bin",
		"OTHER=value",
		"HOME=/temporary/home",
		"XDG_CONFIG_HOME=/temporary/config",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("隔離後の環境に %q がありません: %s", required, joined)
		}
	}
}

func TestSmokeScenariosCoverCompactAndFullExposure(t *testing.T) {
	t.Parallel()

	scenarios := smokeScenarios()
	if len(scenarios) != 8 {
		t.Fatalf("smoke scenario 数 = %d, want 8", len(scenarios))
	}

	type expectedScenario struct {
		toolNames           []string
		specialistToolNames []string
		compact             bool
		fullConfig          bool
	}
	compactTools := []string{
		"discover_legal_tools",
		"execute_legal_tool",
		"query_legal_information",
	}
	coreSpecialists := []string{
		"compare_law_versions",
		"get_article",
		"get_law",
		"list_law_revisions",
		"list_law_updates",
		"search_law_content",
		"search_laws",
	}
	expected := map[string]expectedScenario{
		"default": {
			toolNames:           compactTools,
			specialistToolNames: coreSpecialists,
			compact:             true,
		},
		"judicial-cases": {
			toolNames: compactTools,
			specialistToolNames: []string{
				"compare_law_versions",
				"get_article",
				"get_judicial_case",
				"get_law",
				"list_law_revisions",
				"list_law_updates",
				"search_judicial_cases",
				"search_law_content",
				"search_laws",
			},
			compact: true,
		},
		"legislative-history": {
			toolNames: compactTools,
			specialistToolNames: []string{
				"compare_law_versions",
				"get_article",
				"get_law",
				"list_law_revisions",
				"list_law_updates",
				"search_diet_speeches",
				"search_law_content",
				"search_laws",
			},
			compact: true,
		},
		"judicial-cases-and-legislative-history": {
			toolNames: compactTools,
			specialistToolNames: []string{
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
			},
			compact: true,
		},
		"judicial-citations": {
			toolNames: compactTools,
			specialistToolNames: []string{
				"compare_law_versions",
				"get_article",
				"get_judicial_case",
				"get_law",
				"list_law_revisions",
				"list_law_updates",
				"search_judicial_cases",
				"search_law_content",
				"search_laws",
				"trace_judicial_citations",
			},
			compact: true,
		},
		"all-extension-packs": {
			toolNames: compactTools,
			specialistToolNames: []string{
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
			},
			compact: true,
		},
		"full-default": {
			toolNames: []string{
				"compare_law_versions",
				"get_article",
				"get_law",
				"list_law_revisions",
				"list_law_updates",
				"query_legal_information",
				"search_law_content",
				"search_laws",
			},
			fullConfig: true,
		},
		"full-all-extension-packs": {
			toolNames: []string{
				"compare_law_versions",
				"get_article",
				"get_judicial_case",
				"get_law",
				"list_law_revisions",
				"list_law_updates",
				"query_legal_information",
				"search_diet_speeches",
				"search_judicial_cases",
				"search_law_content",
				"search_laws",
				"trace_judicial_citations",
			},
			fullConfig: true,
		},
	}

	for _, scenario := range scenarios {
		want, exists := expected[scenario.name]
		if !exists {
			t.Fatalf("未知の scenario が含まれています: %q", scenario.name)
		}
		if !slices.Equal(scenario.toolNames, want.toolNames) {
			t.Fatalf(
				"scenario %q の tool 一覧 = %v, want %v",
				scenario.name,
				scenario.toolNames,
				want.toolNames,
			)
		}
		if !slices.Equal(
			scenario.specialistToolNames,
			want.specialistToolNames,
		) {
			t.Fatalf(
				"scenario %q の専門 tool 一覧 = %v, want %v",
				scenario.name,
				scenario.specialistToolNames,
				want.specialistToolNames,
			)
		}
		if scenario.compact != want.compact {
			t.Fatalf(
				"scenario %q の compact = %t, want %t",
				scenario.name,
				scenario.compact,
				want.compact,
			)
		}
		if got := strings.Contains(
			scenario.configYAML,
			"toolExposure: full",
		); got != want.fullConfig {
			t.Fatalf(
				"scenario %q の full 設定有無 = %t, want %t",
				scenario.name,
				got,
				want.fullConfig,
			)
		}
		delete(expected, scenario.name)
	}
	if len(expected) != 0 {
		t.Fatalf("不足 scenario = %v", expected)
	}
}
