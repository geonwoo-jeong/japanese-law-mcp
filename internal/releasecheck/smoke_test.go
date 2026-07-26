package releasecheck

import (
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
