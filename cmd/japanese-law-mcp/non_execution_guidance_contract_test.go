package main

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/config"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const expectedUnsupportedScopeNotice = "指定した法的助言、翻訳、task または resource は統合照会の採用範囲外です。採用済みの法情報取得要求だけを入力してください。"

func TestSOTSCN010PublicNonExecutionGuidance(t *testing.T) {
	cfg := config.Default()
	calls := &legalQueryTestPortCalls{}
	registry, routes := newLegalQueryTestRoutes(t, cfg, calls)
	server, err := newPublicServer("test-version", cfg, registry, routes, false)
	if err != nil {
		t.Fatalf("公開 MCP server を初期化できません: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	serverResult := make(chan error, 1)
	go func() {
		serverResult <- server.Run(ctx, serverTransport)
	}()
	client := sdk.NewClient(
		&sdk.Implementation{Name: "test-client", Version: "test-version"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("公開 MCP server へ接続できません: %v", err)
	}

	tests := []struct {
		name             string
		query            string
		status           legalquery.LegalQueryResultStatus
		notices          []string
		questions        []legalquery.LegalQueryQuestion
		requireQuestions bool
		requiredPack     string
	}{
		{
			name:             "曖昧な法情報要求",
			query:            "制度について法情報を探したいです。",
			status:           legalquery.LegalQueryResultStatusNeedsClarification,
			notices:          []string{},
			requireQuestions: true,
		},
		{
			name:    "五つの明示主題",
			query:   "民法、刑法、会社法、行政手続法、独禁法をそれぞれ検索してください。",
			status:  legalquery.LegalQueryResultStatusNeedsClarification,
			notices: []string{},
			questions: []legalquery.LegalQueryQuestion{
				legalquery.LegalQueryQuestionStepLimitExceeded,
			},
		},
		{
			name:   "無効な裁判例拡張",
			query:  "医療過誤の裁判例を検索してください。",
			status: legalquery.LegalQueryResultStatusCapabilityUnavailable,
			notices: []string{
				legalquery.LegalQueryPackDisabledNotice,
			},
			requiredPack: legalqueryplanning.JudicialCasesPackID,
		},
		{
			name:   "取得と対象外要求の混在",
			query:  "民法を検索して影響グラフを作成してください。",
			status: legalquery.LegalQueryResultStatusUnsupported,
			notices: []string{
				legalquery.LegalQueryMixedUnsupportedNotice,
				expectedUnsupportedScopeNotice,
			},
		},
		{
			name:   "法的助言だけの要求",
			query:  "賃金が支払われません。どうすればよいですか。",
			status: legalquery.LegalQueryResultStatusUnsupported,
			notices: []string{
				expectedUnsupportedScopeNotice,
			},
		},
		{
			name:   "翻訳だけの要求",
			query:  "民法を英語に翻訳してください。",
			status: legalquery.LegalQueryResultStatusUnsupported,
			notices: []string{
				expectedUnsupportedScopeNotice,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, callErr := session.CallTool(ctx, &sdk.CallToolParams{
				Name: "query_legal_information",
				Arguments: map[string]any{
					"query": test.query,
				},
			})
			if callErr != nil {
				t.Fatalf("公開統合照会を呼び出せません: %v", callErr)
			}
			payload := decodeNonExecutionGuidance(t, result)
			if payload.Status != test.status ||
				payload.Decision != legalquery.LegalQueryResultDecisionNoExecution ||
				len(payload.Attempts) != 0 ||
				!slices.Equal(payload.Notices, test.notices) {
				t.Fatalf("非実行案内 payload = %#v", payload)
			}
			if test.requireQuestions &&
				(len(payload.Clarification.Questions) < 1 ||
					len(payload.Clarification.Questions) > 2) {
				t.Fatalf("clarification.questions = %#v", payload.Clarification.Questions)
			}
			if test.questions != nil &&
				!slices.Equal(payload.Clarification.Questions, test.questions) {
				t.Fatalf(
					"clarification.questions = %#v, want %#v",
					payload.Clarification.Questions,
					test.questions,
				)
			}
			if test.requiredPack != "" {
				if len(payload.Interpretations) != 1 ||
					payload.Interpretations[0].Availability !=
						legalquery.SelectionAvailabilityPackDisabled ||
					!slices.Equal(
						payload.Interpretations[0].RequiredPacks,
						[]string{test.requiredPack},
					) {
					t.Fatalf("pack 無効時の interpretations = %#v", payload.Interpretations)
				}
			}
		})
	}

	if got := calls.snapshot(); len(got) != 0 {
		t.Fatalf("非実行案内で provider を呼び出しました: %#v", got)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("公開 MCP session を終了できません: %v", err)
	}
	select {
	case runErr := <-serverResult:
		if runErr != nil {
			t.Fatalf("公開 MCP server が正常終了しませんでした: %v", runErr)
		}
	case <-ctx.Done():
		t.Fatalf("公開 MCP server の終了を待機できません: %v", ctx.Err())
	}
}

type nonExecutionGuidancePayload struct {
	Status          legalquery.LegalQueryResultStatus   `json:"status"`
	Decision        legalquery.LegalQueryResultDecision `json:"decision"`
	Notices         []string                            `json:"notices"`
	Attempts        []json.RawMessage                   `json:"attempts"`
	Interpretations []struct {
		Availability  legalquery.SelectionAvailability `json:"availability"`
		RequiredPacks []string                         `json:"requiredPacks"`
	} `json:"interpretations"`
	Clarification struct {
		Questions []legalquery.LegalQueryQuestion `json:"questions"`
	} `json:"clarification"`
}

func decodeNonExecutionGuidance(
	t *testing.T,
	result *sdk.CallToolResult,
) nonExecutionGuidancePayload {
	t.Helper()
	if result == nil || result.IsError || len(result.Content) != 1 {
		t.Fatalf("公開統合照会の結果 = %#v", result)
	}
	content, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("公開統合照会 content の型 = %T", result.Content[0])
	}
	var payload nonExecutionGuidancePayload
	if err := json.Unmarshal([]byte(content.Text), &payload); err != nil {
		t.Fatalf("公開統合照会の JSON を解析できません: %v", err)
	}
	var contentValue any
	if err := json.Unmarshal([]byte(content.Text), &contentValue); err != nil {
		t.Fatalf("content を JSON 値へ変換できません: %v", err)
	}
	structuredJSON, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("structuredContent を JSON に変換できません: %v", err)
	}
	var structuredValue any
	if err := json.Unmarshal(structuredJSON, &structuredValue); err != nil {
		t.Fatalf("structuredContent を JSON 値へ変換できません: %v", err)
	}
	if !reflect.DeepEqual(contentValue, structuredValue) {
		t.Fatalf("content と structuredContent が一致しません")
	}
	return payload
}
