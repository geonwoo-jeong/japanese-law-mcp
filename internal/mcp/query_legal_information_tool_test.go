package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestQueryLegalInformationToolCallsApplicationOnce(t *testing.T) {
	t.Parallel()

	fixture := newQuerySchemaModelFixture(t)
	expected := queryLegalInformationResultFixture(t, fixture)["unsupported"]
	port := &recordingQueryLegalInformationPort{result: expected}
	contextKey := queryLegalInformationContextKey{}
	ctx := context.WithValue(context.Background(), contextKey, "request-context")

	result, err := callQueryLegalInformation(
		ctx,
		port,
		json.RawMessage(`{
			"query":"　永住許可について教えてください　",
			"ref":{
				"providerId":"unknown-provider",
				"key":{
					"sourceId":"opaque-source",
					"resourceType":"law",
					"resourceId":"law-1",
					"versionId":"revision-2"
				}
			}
		}`),
	)
	if err != nil {
		t.Fatalf("tool 呼出しが失敗しました: %v", err)
	}
	assertQueryLegalInformationSuccess(t, result, expected)

	calls, calledContext, request := port.snapshot()
	if calls != 1 {
		t.Fatalf("application の呼出し回数 = %d", calls)
	}
	if calledContext.Value(contextKey) != "request-context" {
		t.Fatal("呼出し元の context を application へ渡しませんでした")
	}
	if request.Query() != "永住許可について教えてください" {
		t.Fatalf("request.query = %q", request.Query())
	}
	if request.LimitPerAttempt() != 10 {
		t.Fatalf("request.limitPerAttempt = %d", request.LimitPerAttempt())
	}
	ref, exists := request.Ref()
	if !exists {
		t.Fatal("request.ref がありません")
	}
	versionID, hasVersion := ref.Key().VersionID()
	if ref.ProviderID() != "unknown-provider" ||
		ref.Key().SourceID() != "opaque-source" ||
		ref.Key().ResourceType() != "law" ||
		ref.Key().ResourceID() != "law-1" ||
		!hasVersion ||
		versionID != "revision-2" {
		t.Fatalf("request.ref = %#v", ref)
	}
}

func TestQueryLegalInformationToolReturnsEverySuccessVariant(t *testing.T) {
	t.Parallel()

	fixture := newQuerySchemaModelFixture(t)
	for name, expected := range queryLegalInformationResultFixture(t, fixture) {
		name, expected := name, expected
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callQueryLegalInformation(
				context.Background(),
				&recordingQueryLegalInformationPort{result: expected},
				json.RawMessage(`{"query":"民法について教えてください"}`),
			)
			if err != nil {
				t.Fatalf("tool 呼出しが失敗しました: %v", err)
			}
			assertQueryLegalInformationSuccess(t, result, expected)
		})
	}
}

func TestQueryLegalInformationToolAcceptsExactIntegerForms(t *testing.T) {
	t.Parallel()

	for input, expected := range map[string]int{
		"1":      1,
		"1.0":    1,
		"1e1":    10,
		"2.0e1":  20,
		"10.000": 10,
	} {
		input, expected := input, expected
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			fixture := newQuerySchemaModelFixture(t)
			port := &recordingQueryLegalInformationPort{
				result: queryLegalInformationResultFixture(t, fixture)["unsupported"],
			}
			arguments := json.RawMessage(
				`{"query":"民法","limitPerAttempt":` + input + `}`,
			)
			result, err := callQueryLegalInformation(
				context.Background(),
				port,
				arguments,
			)
			if err != nil {
				t.Fatalf("tool 呼出しが失敗しました: %v", err)
			}
			if result.IsError {
				t.Fatalf("正確な整数を拒否しました: %s", toolResultText(t, result))
			}
			_, _, request := port.snapshot()
			if request.LimitPerAttempt() != expected {
				t.Fatalf(
					"limitPerAttempt = %d, want %d",
					request.LimitPerAttempt(),
					expected,
				)
			}
		})
	}
}

func TestQueryLegalInformationToolAcceptsArgumentsByteBoundary(t *testing.T) {
	t.Parallel()

	prefix := `{"query":"民法","ref":{"providerId":"unknown-provider","key":{` +
		`"sourceId":"opaque-source","resourceType":"law","resourceId":"`
	suffix := `"}}}`
	resourceIDLength := queryLegalInformationMaxArgumentsBytes -
		len(prefix) -
		len(suffix)
	arguments := json.RawMessage(
		prefix + strings.Repeat("x", resourceIDLength) + suffix,
	)
	if len(arguments) != queryLegalInformationMaxArgumentsBytes {
		t.Fatalf("試験入力の byte 数 = %d", len(arguments))
	}
	fixture := newQuerySchemaModelFixture(t)
	port := &recordingQueryLegalInformationPort{
		result: queryLegalInformationResultFixture(t, fixture)["unsupported"],
	}

	result, err := callQueryLegalInformation(
		context.Background(),
		port,
		arguments,
	)
	if err != nil {
		t.Fatalf("tool 呼出しが失敗しました: %v", err)
	}
	if result.IsError {
		t.Fatalf("16384 byte の arguments を拒否しました: %s", toolResultText(t, result))
	}
	calls, _, request := port.snapshot()
	if calls != 1 {
		t.Fatalf("application の呼出し回数 = %d", calls)
	}
	ref, exists := request.Ref()
	if !exists || len(ref.Key().ResourceID()) != resourceIDLength {
		t.Fatal("境界上の opaque resourceId を変更しました")
	}
}

func TestQueryLegalInformationToolRejectsInvalidInputBeforeApplication(
	t *testing.T,
) {
	t.Parallel()

	invalidUTF8 := append(
		[]byte(`{"query":"`),
		append([]byte{0xff}, []byte(`"}`)...)...,
	)
	tooLong := strings.Repeat("あ", 683)
	oversizedArguments := json.RawMessage(
		`{"query":"民法","ref":{"providerId":"unknown-provider","key":{` +
			`"sourceId":"opaque-source","resourceType":"law","resourceId":"` +
			strings.Repeat("x", queryLegalInformationMaxArgumentsBytes) +
			`"}}}`,
	)
	tests := map[string]json.RawMessage{
		"空入力":                     nil,
		"null":                    json.RawMessage(`null`),
		"配列":                      json.RawMessage(`[]`),
		"壊れた object":              json.RawMessage(`{"query":"民法"`),
		"末尾 data":                 json.RawMessage(`{"query":"民法"}true`),
		"不正 UTF-8":                json.RawMessage(invalidUTF8),
		"arguments 上限超過":          oversizedArguments,
		"未知の項目":                   json.RawMessage(`{"query":"民法","task":"search"}`),
		"query の重複":               json.RawMessage(`{"query":"民法","query":"刑法"}`),
		"query 欠落":                json.RawMessage(`{}`),
		"query null":              json.RawMessage(`{"query":null}`),
		"query 型不一致":              json.RawMessage(`{"query":7}`),
		"query 空白だけ":              json.RawMessage(`{"query":"　 "}`),
		"query 制御文字":              json.RawMessage("{\"query\":\"民法\\n\"}"),
		"query 上限超過":              json.RawMessage(`{"query":"` + tooLong + `"}`),
		"query 単独 high surrogate": json.RawMessage(`{"query":"\ud800"}`),
		"query 単独 low surrogate":  json.RawMessage(`{"query":"\udc00"}`),
		"limit null":              json.RawMessage(`{"query":"民法","limitPerAttempt":null}`),
		"limit 文字列":               json.RawMessage(`{"query":"民法","limitPerAttempt":"10"}`),
		"limit 小数":                json.RawMessage(`{"query":"民法","limitPerAttempt":1.5}`),
		"limit 負数":                json.RawMessage(`{"query":"民法","limitPerAttempt":-1}`),
		"limit 真偽値":               json.RawMessage(`{"query":"民法","limitPerAttempt":true}`),
		"limit 負の指数":              json.RawMessage(`{"query":"民法","limitPerAttempt":1e-1}`),
		"limit 下限未満":              json.RawMessage(`{"query":"民法","limitPerAttempt":0}`),
		"limit 上限超過":              json.RawMessage(`{"query":"民法","limitPerAttempt":21}`),
		"limit 巨大指数":              json.RawMessage(`{"query":"民法","limitPerAttempt":1e999}`),
		"ref null":                json.RawMessage(`{"query":"民法","ref":null}`),
		"ref 型不一致":                json.RawMessage(`{"query":"民法","ref":"law-1"}`),
		"ref 未知の項目":               json.RawMessage(`{"query":"民法","ref":{"providerId":"egov-law-api-v2","key":{},"route":"fallback"}}`),
		"ref providerId 欠落":       json.RawMessage(`{"query":"民法","ref":{"key":{}}}`),
		"ref key 欠落":              json.RawMessage(`{"query":"民法","ref":{"providerId":"egov-law-api-v2"}}`),
		"ref key 未知の項目":           json.RawMessage(`{"query":"民法","ref":{"providerId":"egov-law-api-v2","key":{"sourceId":"egov-laws","resourceType":"law","resourceId":"law-1","raw":"secret"}}}`),
		"ref resourceId の重複":      json.RawMessage(`{"query":"民法","ref":{"providerId":"egov-law-api-v2","key":{"sourceId":"egov-laws","resourceType":"law","resourceId":"law-1","resourceId":"law-2"}}}`),
		"ref sourceId 欠落":         json.RawMessage(`{"query":"民法","ref":{"providerId":"egov-law-api-v2","key":{"resourceType":"law","resourceId":"law-1"}}}`),
		"ref resourceType 対象外":    json.RawMessage(`{"query":"民法","ref":{"providerId":"egov-law-api-v2","key":{"sourceId":"egov-laws","resourceType":"tax","resourceId":"law-1"}}}`),
		"ref resourceId 空":        json.RawMessage(`{"query":"民法","ref":{"providerId":"egov-law-api-v2","key":{"sourceId":"egov-laws","resourceType":"law","resourceId":""}}}`),
		"ref versionId null":      json.RawMessage(`{"query":"民法","ref":{"providerId":"egov-law-api-v2","key":{"sourceId":"egov-laws","resourceType":"law","resourceId":"law-1","versionId":null}}}`),
		"ref versionId 空":         json.RawMessage(`{"query":"民法","ref":{"providerId":"egov-law-api-v2","key":{"sourceId":"egov-laws","resourceType":"law","resourceId":"law-1","versionId":""}}}`),
		"ref providerId 不正":       json.RawMessage(`{"query":"民法","ref":{"providerId":"EGOV","key":{"sourceId":"egov-laws","resourceType":"law","resourceId":"law-1"}}}`),
	}

	for name, arguments := range tests {
		name, arguments := name, arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			port := &recordingQueryLegalInformationPort{}
			result, err := callQueryLegalInformation(
				context.Background(),
				port,
				arguments,
			)
			if err != nil {
				t.Fatalf("入力エラーを protocol error にしました: %v", err)
			}
			payload := assertQueryLegalInformationError(
				t,
				result,
				model.ErrorCodeInvalidArgument,
			)
			if payload.Retryable {
				t.Fatal("invalid_argument を retryable にしました")
			}
			if payload.Details["field"] == "" ||
				payload.Details["reason"] == "" {
				t.Fatalf("修正可能な details がありません: %#v", payload)
			}
			calls, _, _ := port.snapshot()
			if calls != 0 {
				t.Fatalf("不正入力で application を %d 回呼びました", calls)
			}
		})
	}
}

func TestQueryLegalInformationToolは複数の入力違反を固定順で返す(
	t *testing.T,
) {
	t.Parallel()

	const invalidRef = `"ref":{` +
		`"providerId":"E-Gov-Law-Api-V2",` +
		`"key":{` +
		`"sourceId":"e-gov-law-api-v2",` +
		`"resourceType":"law",` +
		`"resourceId":"405AC0000000088"` +
		`}}`
	tests := []struct {
		name          string
		arguments     json.RawMessage
		expectedField string
	}{
		{
			name: "query は limit と ref より先",
			arguments: json.RawMessage(
				`{"query":"   ","limitPerAttempt":0,` +
					invalidRef + `}`,
			),
			expectedField: "query",
		},
		{
			name: "limit は ref より先",
			arguments: json.RawMessage(
				`{"query":"行政手続法を検索","limitPerAttempt":0,` +
					invalidRef + `}`,
			),
			expectedField: "limitPerAttempt",
		},
		{
			name: "query と limit が有効なら ref",
			arguments: json.RawMessage(
				`{"query":"行政手続法を検索","limitPerAttempt":1,` +
					invalidRef + `}`,
			),
			expectedField: "ref",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			port := &recordingQueryLegalInformationPort{}
			result, err := callQueryLegalInformation(
				context.Background(),
				port,
				test.arguments,
			)
			if err != nil {
				t.Fatalf("入力エラーを protocol error にしました: %v", err)
			}
			payload := assertQueryLegalInformationError(
				t,
				result,
				model.ErrorCodeInvalidArgument,
			)
			if payload.Details["field"] != test.expectedField {
				t.Fatalf(
					"details.field = %#v, want %q",
					payload.Details["field"],
					test.expectedField,
				)
			}
			calls, _, _ := port.snapshot()
			if calls != 0 {
				t.Fatalf("不正入力で application を %d 回呼びました", calls)
			}
		})
	}
}

func TestQueryLegalInformationToolMapsApplicationErrors(t *testing.T) {
	t.Parallel()

	argumentError, err := legalquery.NewArgumentError(
		"ref",
		"の sourceId が採用済み provider と一致しません",
	)
	if err != nil {
		t.Fatalf("ArgumentError を作成できません: %v", err)
	}
	fixture := newQuerySchemaModelFixture(t)
	allFailed := queryLegalInformationAllFailedError(t, fixture)

	tests := []struct {
		name         string
		err          error
		expectedCode model.ErrorCode
		expected     map[string]any
	}{
		{
			name:         "入力エラー",
			err:          fmt.Errorf("service stage: %w", argumentError),
			expectedCode: model.ErrorCodeInvalidArgument,
			expected: map[string]any{
				"field":  "ref",
				"reason": "の sourceId が採用済み provider と一致しません",
			},
		},
		{
			name:         "全 step 失敗",
			err:          fmt.Errorf("service stage: %w", allFailed),
			expectedCode: model.ErrorCodeRateLimited,
			expected: map[string]any{
				"retryAfter": "120",
			},
		},
		{
			name:         "未分類エラー",
			err:          errors.New("raw query=秘密 /private/token"),
			expectedCode: model.ErrorCodeInternalError,
		},
		{
			name:         "未初期化の入力エラー",
			err:          legalquery.ArgumentError{},
			expectedCode: model.ErrorCodeInternalError,
		},
		{
			name:         "呼出し元の中断",
			err:          context.Canceled,
			expectedCode: model.ErrorCodeInternalError,
		},
		{
			name:         "呼出し元の期限超過",
			err:          context.DeadlineExceeded,
			expectedCode: model.ErrorCodeInternalError,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			port := &recordingQueryLegalInformationPort{err: test.err}
			result, callErr := callQueryLegalInformation(
				context.Background(),
				port,
				json.RawMessage(`{"query":"秘密の照会"}`),
			)
			if callErr != nil {
				t.Fatalf("application error を protocol error にしました: %v", callErr)
			}
			payload := assertQueryLegalInformationError(
				t,
				result,
				test.expectedCode,
			)
			if test.expected != nil {
				for key, expected := range test.expected {
					if payload.Details[key] != expected {
						t.Fatalf(
							"details.%s = %#v, want %#v",
							key,
							payload.Details[key],
							expected,
						)
					}
				}
			}
			text := toolResultText(t, result)
			for _, secret := range []string{
				"秘密の照会",
				"raw query",
				"/private/token",
			} {
				if strings.Contains(text, secret) {
					t.Fatalf("公開エラーに入力または内部値があります: %q", text)
				}
			}
			calls, _, _ := port.snapshot()
			if calls != 1 {
				t.Fatalf("application の呼出し回数 = %d", calls)
			}
		})
	}
}

func TestQueryLegalInformationToolPreservesAllFailedPublicError(t *testing.T) {
	t.Parallel()

	fixture := newQuerySchemaModelFixture(t)
	allFailedError := queryLegalInformationAllFailedError(t, fixture)
	var allFailed legalquery.LegalQueryAllFailedError
	if !errors.As(allFailedError, &allFailed) {
		t.Fatalf("LegalQueryAllFailedError ではありません: %v", allFailedError)
	}
	expected, err := json.Marshal(allFailed.ErrorResult())
	if err != nil {
		t.Fatalf("公開 ErrorResult を JSON に変換できません: %v", err)
	}

	result, err := callQueryLegalInformation(
		context.Background(),
		&recordingQueryLegalInformationPort{
			err: fmt.Errorf("service stage: %w", allFailedError),
		},
		json.RawMessage(`{"query":"民法"}`),
	)
	if err != nil {
		t.Fatalf("全失敗を protocol error にしました: %v", err)
	}
	assertQueryLegalInformationError(t, result, model.ErrorCodeRateLimited)
	assertSameJSONValue(
		t,
		json.RawMessage(toolResultText(t, result)),
		json.RawMessage(expected),
	)
}

func TestQueryLegalInformationToolFailsClosedForInvalidDependenciesAndResults(
	t *testing.T,
) {
	t.Parallel()

	fixture := newQuerySchemaModelFixture(t)
	validResult := queryLegalInformationResultFixture(t, fixture)["unsupported"]
	var typedNilPort *recordingQueryLegalInformationPort
	var typedNilResult *legalquery.LegalQueryUnsupportedResult
	tests := []struct {
		name string
		port legalquery.Port
	}{
		{name: "nil port", port: nil},
		{name: "typed nil port", port: typedNilPort},
		{
			name: "nil result",
			port: &recordingQueryLegalInformationPort{},
		},
		{
			name: "typed nil result",
			port: &recordingQueryLegalInformationPort{
				result: typedNilResult,
			},
		},
		{
			name: "invalid result",
			port: &recordingQueryLegalInformationPort{
				result: legalquery.LegalQueryUnsupportedResult{},
			},
		},
		{
			name: "result and error",
			port: &recordingQueryLegalInformationPort{
				result: validResult,
				err:    errors.New("結果より優先する固定エラー"),
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := callQueryLegalInformation(
				context.Background(),
				test.port,
				json.RawMessage(`{"query":"民法"}`),
			)
			if err != nil {
				t.Fatalf("fail closed を protocol error にしました: %v", err)
			}
			assertQueryLegalInformationError(
				t,
				result,
				model.ErrorCodeInternalError,
			)
		})
	}
}

type queryLegalInformationContextKey struct{}

type recordingQueryLegalInformationPort struct {
	mu      sync.Mutex
	result  legalquery.LegalQueryResult
	err     error
	calls   int
	ctx     context.Context
	request legalquery.Request
}

func (port *recordingQueryLegalInformationPort) Query(
	ctx context.Context,
	request legalquery.Request,
) (legalquery.LegalQueryResult, error) {
	port.mu.Lock()
	defer port.mu.Unlock()
	port.calls++
	port.ctx = ctx
	port.request = request
	return port.result, port.err
}

func (port *recordingQueryLegalInformationPort) snapshot() (
	int,
	context.Context,
	legalquery.Request,
) {
	port.mu.Lock()
	defer port.mu.Unlock()
	return port.calls, port.ctx, port.request
}

type queryLegalInformationErrorPayload struct {
	Code      model.ErrorCode `json:"code"`
	Message   string          `json:"message"`
	Retryable bool            `json:"retryable"`
	Details   map[string]any  `json:"details"`
}

func assertQueryLegalInformationError(
	t *testing.T,
	result *sdk.CallToolResult,
	expected model.ErrorCode,
) queryLegalInformationErrorPayload {
	t.Helper()

	if result == nil || !result.IsError {
		t.Fatalf("tool error ではありません: %#v", result)
	}
	if result.StructuredContent != nil {
		t.Fatalf("tool error に structuredContent があります: %#v", result)
	}
	var payload queryLegalInformationErrorPayload
	if err := json.Unmarshal([]byte(toolResultText(t, result)), &payload); err != nil {
		t.Fatalf("ErrorResult を解析できません: %v", err)
	}
	if payload.Code != expected {
		t.Fatalf("error code = %q, want %q", payload.Code, expected)
	}
	return payload
}

func assertQueryLegalInformationSuccess(
	t *testing.T,
	result *sdk.CallToolResult,
	expected legalquery.LegalQueryResult,
) {
	t.Helper()

	if result == nil || result.IsError {
		t.Fatalf("成功結果ではありません: %#v", result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content の件数 = %d", len(result.Content))
	}
	text := toolResultText(t, result)
	for _, internalField := range []string{
		`"semanticScore"`,
		`"candidateId"`,
		`"rankedCandidates"`,
		`"trace"`,
		`"continuationToken"`,
		`"offset"`,
	} {
		if strings.Contains(text, internalField) {
			t.Fatalf("公開結果に内部項目 %s があります", internalField)
		}
	}
	structured, ok := result.StructuredContent.(json.RawMessage)
	if !ok {
		t.Fatalf("structuredContent の型 = %T", result.StructuredContent)
	}
	if string(structured) != text {
		t.Fatalf("content と structuredContent の JSON byte が一致しません")
	}
	assertSameJSONValue(t, structured, json.RawMessage(text))
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("期待結果を JSON に変換できません: %v", err)
	}
	assertSameJSONValue(t, json.RawMessage(text), json.RawMessage(expectedJSON))

	var instance any
	if err := json.Unmarshal([]byte(text), &instance); err != nil {
		t.Fatalf("公開結果を解析できません: %v", err)
	}
	assertQuerySchemaAccepts(t, newQueryLegalInformationOutputSchema(), instance)
}

func toolResultText(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()

	if result == nil || len(result.Content) != 1 {
		t.Fatalf("content が一件ではありません: %#v", result)
	}
	content, ok := result.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("content の型 = %T", result.Content[0])
	}
	return content.Text
}
