package main

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/config"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestLegalQueryPlanningDependenciesLoadCoreAndJudicialCasesInFixedOrder(
	t *testing.T,
) {
	t.Parallel()

	planning, err := loadLegalQueryPlanningDependencies()
	if err != nil {
		t.Fatalf("統合照会の planning dependencies を読み込めません: %v", err)
	}

	coreResult := collectPlanningCandidates(
		t,
		planning,
		"独禁法の正式な法令を検索してください。",
	)
	coreCandidates := coreResult.RankedCandidates()
	if len(coreCandidates) != 1 ||
		coreCandidates[0].CandidateID() != "candidate-1-1" ||
		len(coreCandidates[0].RequiredPacks()) != 0 {
		t.Fatalf("core profile の候補 = %#v", coreCandidates)
	}

	const judicialQuery = "医療過誤の裁判例を検索してください。"
	request := newLegalQueryRequest(t, judicialQuery)
	preprocessed, err := planning.preprocessor.Preprocess(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("裁判例照会を前処理できません: %v", err)
	}
	for _, want := range [][2]string{
		{"core", "reserved-pack-judicial-cases"},
		{"judicial-cases", "resource-judicial-decision"},
		{"judicial-cases", "task-search"},
	} {
		if !hasCueMention(
			preprocessed.CueMentions(),
			want[0],
			want[1],
		) {
			t.Fatalf(
				"profileId=%q cueId=%q がありません: %#v",
				want[0],
				want[1],
				preprocessed.CueMentions(),
			)
		}
	}
	judicialResult, err := planning.profiles.Collect(preprocessed)
	if err != nil {
		t.Fatalf("裁判例 profile contribution を集約できません: %v", err)
	}
	judicialCandidates := judicialResult.RankedCandidates()
	if len(judicialCandidates) != 1 ||
		judicialCandidates[0].CandidateID() != "candidate-2-1" ||
		!slices.Equal(
			judicialCandidates[0].RequiredPacks(),
			[]string{legalqueryplanning.JudicialCasesPackID},
		) {
		t.Fatalf("judicial-cases profile の候補 = %#v", judicialCandidates)
	}
}

func TestLegalQueryServiceReturnsCapabilityUnavailableForDisabledJudicialSearch(
	t *testing.T,
) {
	t.Parallel()

	cfg := config.Default()
	calls := &legalQueryTestPortCalls{}
	_, routes := newLegalQueryTestRoutes(t, cfg, calls)
	service, err := newLegalQueryService(cfg, routes)
	if err != nil {
		t.Fatalf("統合照会 service を初期化できません: %v", err)
	}
	result, err := service.Query(
		context.Background(),
		newLegalQueryRequest(t, "医療過誤の裁判例を検索してください。"),
	)
	if err != nil {
		t.Fatalf("pack 無効時の裁判例照会エラー = %v", err)
	}
	if result.Status() != legalquery.LegalQueryResultStatusCapabilityUnavailable ||
		result.Decision() != legalquery.LegalQueryResultDecisionNoExecution ||
		len(result.Attempts()) != 0 {
		t.Fatalf(
			"pack 無効時の裁判例照会 = status:%q decision:%q attempts:%#v",
			result.Status(),
			result.Decision(),
			result.Attempts(),
		)
	}
	if got := calls.snapshot(); len(got) != 0 {
		t.Fatalf("pack 無効時に provider を呼び出しました: %#v", got)
	}
}

func TestPublicLegalQueryReturnsCapabilityUnavailableWithoutProviderCalls(
	t *testing.T,
) {
	t.Parallel()

	cfg := config.Default()
	calls := &legalQueryTestPortCalls{}
	registry, routes := newLegalQueryTestRoutes(t, cfg, calls)
	server, err := newPublicServer(
		"test-version",
		cfg,
		registry,
		routes,
		false,
	)
	if err != nil {
		t.Fatalf("公開 MCP server を初期化できません: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("公開 tool 一覧を取得できません: %v", err)
	}
	if len(tools.Tools) != 6 || tools.Tools[3].Name != "query_legal_information" {
		t.Fatalf("法令コア公開 tool 一覧 = %#v", tools.Tools)
	}

	callResult, err := session.CallTool(ctx, &sdk.CallToolParams{
		Name: "query_legal_information",
		Arguments: map[string]any{
			"query": "医療過誤の裁判例を検索してください。",
		},
	})
	if err != nil {
		t.Fatalf("公開統合照会を呼び出せません: %v", err)
	}
	if callResult == nil || callResult.IsError || len(callResult.Content) != 1 {
		t.Fatalf("公開統合照会の結果 = %#v", callResult)
	}
	content, ok := callResult.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("公開統合照会 content の型 = %T", callResult.Content[0])
	}
	var payload struct {
		Status legalquery.LegalQueryResultStatus `json:"status"`
	}
	if err := json.Unmarshal([]byte(content.Text), &payload); err != nil {
		t.Fatalf("公開統合照会の JSON を解析できません: %v", err)
	}
	if payload.Status != legalquery.LegalQueryResultStatusCapabilityUnavailable {
		t.Fatalf("公開統合照会 status = %q", payload.Status)
	}
	if got := calls.snapshot(); len(got) != 0 {
		t.Fatalf("公開 pack 無効照会で provider を呼び出しました: %#v", got)
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

func TestPublicLegalQueryNonExecutionGuidanceScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		query                 string
		wantStatus            legalquery.LegalQueryResultStatus
		wantNotices           []string
		wantQuestions         []string
		wantRequiredPacks     []string
		wantProviderCallCount int
	}{
		{
			name:        "曖昧な一般語は明確化へ導く",
			query:       "制度について法情報を探したいです。",
			wantStatus:  legalquery.LegalQueryResultStatusNeedsClarification,
			wantNotices: []string{},
			wantQuestions: []string{
				string(legalquery.LegalQueryQuestionResource),
				string(legalquery.LegalQueryQuestionLaw),
			},
		},
		{
			name:        "五件超の明示主題は分割質問だけを返す",
			query:       "量子相続、月面抵当、火星登記、宇宙帰化、海底供託について教えてください。",
			wantStatus:  legalquery.LegalQueryResultStatusNeedsClarification,
			wantNotices: []string{},
			wantQuestions: []string{
				string(legalquery.LegalQueryQuestionStepLimitExceeded),
			},
		},
		{
			name:       "無効な裁判例要求は必要packを返す",
			query:      "司法パック無効時に医療過誤の裁判例を検索してください。",
			wantStatus: legalquery.LegalQueryResultStatusCapabilityUnavailable,
			wantNotices: []string{
				legalquery.LegalQueryPackDisabledNotice,
			},
			wantRequiredPacks: []string{"judicial-cases"},
		},
		{
			name:       "法的助言だけの要求は対象外にする",
			query:      "賃金が支払われません。どうすればよいですか。",
			wantStatus: legalquery.LegalQueryResultStatusUnsupported,
			wantNotices: []string{
				legalquery.LegalQueryUnsupportedScopeNotice,
			},
		},
		{
			name:       "翻訳だけの要求は対象外にする",
			query:      "民法を英語に翻訳してください。",
			wantStatus: legalquery.LegalQueryResultStatusUnsupported,
			wantNotices: []string{
				legalquery.LegalQueryUnsupportedScopeNotice,
			},
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Default()
			calls := &legalQueryTestPortCalls{}
			registry, routes := newLegalQueryTestRoutes(t, cfg, calls)
			server, err := newPublicServer(
				"test-version",
				cfg,
				registry,
				routes,
				false,
			)
			if err != nil {
				t.Fatalf("公開 MCP server を初期化できません: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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

			callResult, err := session.CallTool(ctx, &sdk.CallToolParams{
				Name: "query_legal_information",
				Arguments: map[string]any{
					"query": testCase.query,
				},
			})
			if err != nil {
				t.Fatalf("公開統合照会を呼び出せません: %v", err)
			}
			if callResult == nil || callResult.IsError || len(callResult.Content) != 1 {
				t.Fatalf("公開統合照会の結果 = %#v", callResult)
			}

			payload := decodePublicLegalQueryPayload(t, callResult)
			if payload.Status != testCase.wantStatus {
				t.Fatalf("公開統合照会 status = %q, want %q", payload.Status, testCase.wantStatus)
			}
			if !slices.Equal(payload.Notices, testCase.wantNotices) {
				t.Fatalf("公開統合照会 notices = %#v, want %#v", payload.Notices, testCase.wantNotices)
			}
			if !slices.Equal(payload.Clarification.Questions, testCase.wantQuestions) {
				t.Fatalf(
					"公開統合照会 clarification.questions = %#v, want %#v",
					payload.Clarification.Questions,
					testCase.wantQuestions,
				)
			}
			if len(testCase.wantRequiredPacks) == 0 {
				if len(payload.Interpretations) != 0 {
					t.Fatalf("公開統合照会 interpretations = %#v, want empty", payload.Interpretations)
				}
			} else {
				if len(payload.Interpretations) != 1 {
					t.Fatalf("公開統合照会 interpretations = %#v, want one", payload.Interpretations)
				}
				if !slices.Equal(
					payload.Interpretations[0].RequiredPacks,
					testCase.wantRequiredPacks,
				) {
					t.Fatalf(
						"公開統合照会 requiredPacks = %#v, want %#v",
						payload.Interpretations[0].RequiredPacks,
						testCase.wantRequiredPacks,
					)
				}
			}
			if got := calls.snapshot(); len(got) != testCase.wantProviderCallCount {
				t.Fatalf("公開統合照会の provider 呼出し = %#v", got)
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
		})
	}
}

func TestLegalQueryServiceExecutesEnabledJudicialSearchThroughFacadeRoute(
	t *testing.T,
) {
	t.Parallel()

	cfg, err := config.New(withJudicialCasesEnabled())
	if err != nil {
		t.Fatalf("judicial-cases 有効設定を構築できません: %v", err)
	}
	calls := &legalQueryTestPortCalls{}
	_, routes := newLegalQueryTestRoutes(t, cfg, calls)
	service, err := newLegalQueryService(cfg, routes)
	if err != nil {
		t.Fatalf("統合照会 service を初期化できません: %v", err)
	}
	result, err := service.Query(
		context.Background(),
		newLegalQueryRequest(t, "医療過誤の裁判例を検索してください。"),
	)
	if err != nil {
		t.Fatalf("pack 有効時の裁判例照会エラー = %v", err)
	}
	attempts := result.Attempts()
	if result.Status() != legalquery.LegalQueryResultStatusEmpty ||
		result.Decision() != legalquery.LegalQueryResultDecisionSingle ||
		len(attempts) != 1 {
		t.Fatalf(
			"pack 有効時の裁判例照会 = status:%q decision:%q attempts:%#v",
			result.Status(),
			result.Decision(),
			attempts,
		)
	}
	if _, ok := attempts[0].(legalquery.LegalQueryJudicialSearchAttempt); !ok {
		t.Fatalf("裁判例検索 attempt の型 = %T", attempts[0])
	}
	if got := calls.snapshot(); !slices.Equal(
		got,
		[]string{"judicial-decision.search:医療過誤"},
	) {
		t.Fatalf("pack 有効時の provider 呼出し = %#v", got)
	}
}

func TestLegalQueryServiceReturnsCapabilityUnavailableForDisabledJudicialRead(
	t *testing.T,
) {
	t.Parallel()

	cfg := config.Default()
	calls := &legalQueryTestPortCalls{}
	_, routes := newLegalQueryTestRoutes(t, cfg, calls)
	service, err := newLegalQueryService(cfg, routes)
	if err != nil {
		t.Fatalf("統合照会 service を初期化できません: %v", err)
	}
	ref := newLegalQueryTestJudicialRef(t)
	result, err := service.Query(
		context.Background(),
		newLegalQueryRequestWithRef(
			t,
			"指定参照の最高裁判例を取得してください。",
			ref,
		),
	)
	if err != nil {
		t.Fatalf("pack 無効時の裁判例読取りエラー = %v", err)
	}
	if result.Status() != legalquery.LegalQueryResultStatusCapabilityUnavailable ||
		result.Decision() != legalquery.LegalQueryResultDecisionNoExecution ||
		len(result.Attempts()) != 0 {
		t.Fatalf(
			"pack 無効時の裁判例読取り = status:%q decision:%q attempts:%#v",
			result.Status(),
			result.Decision(),
			result.Attempts(),
		)
	}
	if got := calls.snapshot(); len(got) != 0 {
		t.Fatalf("pack 無効時の裁判例読取りで provider を呼び出しました: %#v", got)
	}
}

func TestLegalQueryServiceExecutesEnabledJudicialReadThroughFacadeRoute(
	t *testing.T,
) {
	t.Parallel()

	cfg, err := config.New(withJudicialCasesEnabled())
	if err != nil {
		t.Fatalf("judicial-cases 有効設定を構築できません: %v", err)
	}
	calls := &legalQueryTestPortCalls{}
	_, routes := newLegalQueryTestRoutes(t, cfg, calls)
	service, err := newLegalQueryService(cfg, routes)
	if err != nil {
		t.Fatalf("統合照会 service を初期化できません: %v", err)
	}
	ref := newLegalQueryTestJudicialRef(t)
	result, err := service.Query(
		context.Background(),
		newLegalQueryRequestWithRef(
			t,
			"指定参照の最高裁判例を取得してください。",
			ref,
		),
	)
	if err != nil {
		t.Fatalf("pack 有効時の裁判例読取りエラー = %v", err)
	}
	attempts := result.Attempts()
	if result.Status() != legalquery.LegalQueryResultStatusCompleted ||
		result.Decision() != legalquery.LegalQueryResultDecisionSingle ||
		len(attempts) != 1 {
		t.Fatalf(
			"pack 有効時の裁判例読取り = status:%q decision:%q attempts:%#v",
			result.Status(),
			result.Decision(),
			attempts,
		)
	}
	attempt, ok := attempts[0].(legalquery.LegalQueryJudicialDecisionAttempt)
	if !ok {
		t.Fatalf("裁判例読取り attempt の型 = %T", attempts[0])
	}
	if err := attempt.Validate(); err != nil {
		t.Fatalf("裁判例読取り attempt が有効ではありません: %v", err)
	}
	if attempt.Item().Ref() != ref ||
		attempt.Item().Data().Summary().DecisionID() != ref.Key().ResourceID() {
		t.Fatalf("裁判例読取り item = %#v", attempt.Item())
	}
	if got := calls.snapshot(); !slices.Equal(
		got,
		[]string{"judicial-decision.read:" + ref.Key().ResourceID()},
	) {
		t.Fatalf("裁判例読取りの provider 呼出し = %#v", got)
	}
}

func TestLegalQueryServiceKeepsCoreQueryOnCoreFacadeRoute(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	calls := &legalQueryTestPortCalls{}
	_, routes := newLegalQueryTestRoutes(t, cfg, calls)
	service, err := newLegalQueryService(cfg, routes)
	if err != nil {
		t.Fatalf("統合照会 service を初期化できません: %v", err)
	}
	result, err := service.Query(
		context.Background(),
		newLegalQueryRequest(t, "独禁法の正式な法令を検索してください。"),
	)
	if err != nil {
		t.Fatalf("法令コア照会エラー = %v", err)
	}
	attempts := result.Attempts()
	if result.Status() != legalquery.LegalQueryResultStatusEmpty ||
		result.Decision() != legalquery.LegalQueryResultDecisionSingle ||
		len(attempts) != 1 {
		t.Fatalf(
			"法令コア照会 = status:%q decision:%q attempts:%#v",
			result.Status(),
			result.Decision(),
			attempts,
		)
	}
	if _, ok := attempts[0].(legalquery.LegalQueryLawSearchAttempt); !ok {
		t.Fatalf("法令検索 attempt の型 = %T", attempts[0])
	}
	got := calls.snapshot()
	if len(got) != 1 || !strings.HasPrefix(got[0], "law.search:") {
		t.Fatalf("法令コアの provider 呼出し = %#v", got)
	}
}

func TestLegalQueryServiceFailsClosedForIncompleteEnabledJudicialRoutes(
	t *testing.T,
) {
	t.Parallel()

	cfg, err := config.New(withJudicialCasesEnabled())
	if err != nil {
		t.Fatalf("judicial-cases 有効設定を構築できません: %v", err)
	}
	for _, capabilityID := range []string{
		"judicial-decision.search",
		"judicial-decision.read",
	} {
		capabilityID := capabilityID
		t.Run(capabilityID, func(t *testing.T) {
			t.Parallel()

			routes := providerRoutesWithoutCapability(t, cfg, capabilityID)
			_, serviceErr := newLegalQueryService(cfg, routes)
			if !config.IsValidationError(serviceErr) {
				t.Fatalf(
					"不完全な judicial facade/route の起動エラー = %v",
					serviceErr,
				)
			}
		})
	}
}

func collectPlanningCandidates(
	t *testing.T,
	planning legalQueryPlanningDependencies,
	query string,
) legalquery.QueryProfileSetResult {
	t.Helper()

	request := newLegalQueryRequest(t, query)
	preprocessed, err := planning.preprocessor.Preprocess(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("統合照会を前処理できません: %v", err)
	}
	result, err := planning.profiles.Collect(preprocessed)
	if err != nil {
		t.Fatalf("profile contribution を集約できません: %v", err)
	}
	return result
}

func newLegalQueryRequest(
	t *testing.T,
	query string,
) legalquery.Request {
	t.Helper()

	request, err := legalquery.NewRequest(
		legalquery.RequestValues{Query: query},
	)
	if err != nil {
		t.Fatalf("統合照会 request を構築できません: %v", err)
	}
	return request
}

func newLegalQueryRequestWithRef(
	t *testing.T,
	query string,
	ref model.SourceResourceRef,
) legalquery.Request {
	t.Helper()

	request, err := legalquery.NewRequest(
		legalquery.RequestValues{Query: query, Ref: &ref},
	)
	if err != nil {
		t.Fatalf("参照付き統合照会 request を構築できません: %v", err)
	}
	return request
}

func newLegalQueryTestJudicialRef(t *testing.T) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "95878/detail3",
	})
	if err != nil {
		t.Fatalf("試験用裁判例 key を構築できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用裁判例 ref を構築できません: %v", err)
	}
	return ref
}

func hasCueMention(
	mentions []legalquery.CueMention,
	profileID string,
	cueID string,
) bool {
	for _, mention := range mentions {
		if mention.ProfileID() == profileID && mention.CueID() == cueID {
			return true
		}
	}
	return false
}

type publicLegalQueryPayload struct {
	Status        legalquery.LegalQueryResultStatus `json:"status"`
	Notices       []string                          `json:"notices"`
	Clarification struct {
		Questions []string `json:"questions"`
	} `json:"clarification"`
	Interpretations []struct {
		RequiredPacks []string `json:"requiredPacks"`
	} `json:"interpretations"`
}

func decodePublicLegalQueryPayload(
	t *testing.T,
	result *sdk.CallToolResult,
) publicLegalQueryPayload {
	t.Helper()

	content, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("公開統合照会 content の型 = %T", result.Content[0])
	}
	structuredRaw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("公開統合照会 structuredContent を JSON に変換できません: %v", err)
	}
	var structuredValue any
	if err := json.Unmarshal(structuredRaw, &structuredValue); err != nil {
		t.Fatalf("公開統合照会 structuredContent を比較用に解析できません: %v", err)
	}
	var contentValue any
	if err := json.Unmarshal([]byte(content.Text), &contentValue); err != nil {
		t.Fatalf("公開統合照会 content を比較用に解析できません: %v", err)
	}
	if !slices.EqualFunc([]any{structuredValue}, []any{contentValue}, func(left, right any) bool {
		return jsonEqual(left, right)
	}) {
		t.Fatalf("公開統合照会 content と structuredContent が一致しません")
	}

	var payload publicLegalQueryPayload
	if err := json.Unmarshal([]byte(content.Text), &payload); err != nil {
		t.Fatalf("公開統合照会の JSON を解析できません: %v", err)
	}
	return payload
}

func jsonEqual(left any, right any) bool {
	leftRaw, leftErr := json.Marshal(left)
	rightRaw, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftRaw) == string(rightRaw)
}

func newLegalQueryTestRoutes(
	t *testing.T,
	cfg config.Config,
	calls *legalQueryTestPortCalls,
) (application.ProviderBindingRegistry, application.ProviderRoutes) {
	t.Helper()

	bindings, err := newEnabledProviderBindings(cfg)
	if err != nil {
		t.Fatalf("試験用 provider bindings を構築できません: %v", err)
	}
	replaceLegalQueryTestPorts(bindings, calls)
	registry, err := application.NewProviderBindingRegistry(bindings)
	if err != nil {
		t.Fatalf("試験用 provider registry を構築できません: %v", err)
	}
	routes, err := application.NewProviderRoutes(
		registry,
		configuredProviderRouteValues(cfg.ProviderRoutes()),
	)
	if err != nil {
		t.Fatalf("試験用 provider routes を構築できません: %v", err)
	}
	return registry, routes
}
