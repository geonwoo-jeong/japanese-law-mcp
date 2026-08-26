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

func TestSmokeScenariosCoverOptionalExtensionPacks(t *testing.T) {
	t.Parallel()

	scenarios := smokeScenarios()
	if len(scenarios) != 6 {
		t.Fatalf("smoke scenario 数 = %d, want 6", len(scenarios))
	}

	expected := map[string][]string{
		"default": {
			"compare_law_versions",
			"get_article",
			"get_law",
			"list_law_revisions",
			"list_law_updates",
			"query_legal_information",
			"search_law_content",
			"search_laws",
		},
		"judicial-cases": {
			"compare_law_versions",
			"get_article",
			"get_judicial_case",
			"get_law",
			"list_law_revisions",
			"list_law_updates",
			"query_legal_information",
			"search_judicial_cases",
			"search_law_content",
			"search_laws",
		},
		"legislative-history": {
			"compare_law_versions",
			"get_article",
			"get_law",
			"list_law_revisions",
			"list_law_updates",
			"query_legal_information",
			"search_diet_speeches",
			"search_law_content",
			"search_laws",
		},
		"judicial-cases-and-legislative-history": {
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
		},
		"judicial-citations": {
			"compare_law_versions",
			"get_article",
			"get_judicial_case",
			"get_law",
			"list_law_revisions",
			"list_law_updates",
			"query_legal_information",
			"search_judicial_cases",
			"search_law_content",
			"search_laws",
			"trace_judicial_citations",
		},
		"all-extension-packs": {
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
	}

	for _, scenario := range scenarios {
		want, exists := expected[scenario.name]
		if !exists {
			t.Fatalf("未知の scenario が含まれています: %q", scenario.name)
		}
		if !slices.Equal(scenario.toolNames, want) {
			t.Fatalf(
				"scenario %q の tool 一覧 = %v, want %v",
				scenario.name,
				scenario.toolNames,
				want,
			)
		}
		delete(expected, scenario.name)
	}
	if len(expected) != 0 {
		t.Fatalf("不足 scenario = %v", expected)
	}
}
