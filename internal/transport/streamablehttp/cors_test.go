package streamablehttp

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// TestHandlerPreflight は SOT-DEL-015 の事前確認と拒否境界を検証する。
func TestHandlerPreflight(t *testing.T) {
	t.Parallel()

	const origin = "https://example.test"
	tests := []struct {
		name    string
		origins []string
		methods []string
		headers []string
		session bool
		empty   bool
		status  int
	}{
		{name: "初期化", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"content-type"}, status: 204},
		{name: "初期化後", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"accept, content-type, mcp-protocol-version"}, status: 204},
		{name: "名前の大小文字と空白", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"Content-Type,\t MCP-Protocol-Version "}, status: 204},
		{name: "複数行と重複", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"Content-Type", "Content-Type,Accept"}, status: 204},
		{name: "要求headerなし", origins: []string{origin}, methods: []string{"POST"}, status: 204},
		{name: "空の許可リスト", origins: []string{origin}, methods: []string{"POST"}, empty: true, status: 403},
		{name: "Originなし", methods: []string{"POST"}, status: 400},
		{name: "空Origin", origins: []string{""}, methods: []string{"POST"}, status: 403},
		{name: "複数Origin", origins: []string{origin, origin}, methods: []string{"POST"}, status: 403},
		{name: "結合Origin", origins: []string{origin + ", " + origin}, methods: []string{"POST"}, status: 403},
		{name: "null", origins: []string{"null"}, methods: []string{"POST"}, status: 403},
		{name: "wildcard Origin", origins: []string{"*"}, methods: []string{"POST"}, status: 403},
		{name: "HTTP Origin", origins: []string{"http://example.test"}, methods: []string{"POST"}, status: 403},
		{name: "部分一致", origins: []string{origin + ".evil.test"}, methods: []string{"POST"}, status: 403},
		{name: "末尾slash", origins: []string{origin + "/"}, methods: []string{"POST"}, status: 403},
		{name: "既定port", origins: []string{origin + ":443"}, methods: []string{"POST"}, status: 403},
		{name: "大小文字差", origins: []string{"https://EXAMPLE.test"}, methods: []string{"POST"}, status: 403},
		{name: "methodなし", origins: []string{origin}, status: 400},
		{name: "複数method", origins: []string{origin}, methods: []string{"POST", "POST"}, status: 400},
		{name: "GET", origins: []string{origin}, methods: []string{"GET"}, status: 400},
		{name: "小文字method", origins: []string{origin}, methods: []string{"post"}, status: 400},
		{name: "空header", origins: []string{origin}, methods: []string{"POST"}, headers: []string{""}, status: 400},
		{name: "空要素", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"content-type,"}, status: 400},
		{name: "不正な名前", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"content type"}, status: 400},
		{name: "非ASCIIの大小文字変換", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"MCP-Protocol-Versİon"}, status: 400},
		{name: "未知header", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"content-type,x-unknown"}, status: 400},
		{name: "後続行の未知header", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"content-type", "x-unknown"}, status: 400},
		{name: "認証", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"Authorization"}, status: 400},
		{name: "session", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"Mcp-Session-Id"}, status: 400},
		{name: "再開", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"Last-Event-ID"}, status: 400},
		{name: "wildcard header", origins: []string{origin}, methods: []string{"POST"}, headers: []string{"*"}, status: 400},
		{name: "OPTIONS自体のsession", origins: []string{origin}, methods: []string{"POST"}, session: true, status: 400},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			started := make(chan struct{}, 1)
			options := Options{AllowedOrigins: []string{origin}}
			if test.empty {
				options = Options{}
			}
			handler := NewHandler(newTestServer(started, nil), options)
			request := newJSONRequest(t, http.MethodOptions, "/mcp", toolCallRequestBody("wait"))
			request.Header["Origin"] = test.origins
			request.Header["Access-Control-Request-Method"] = test.methods
			request.Header["Access-Control-Request-Headers"] = test.headers
			if test.session {
				request.Header.Set(sessionIDHeader, "")
			}
			recorder := httptest.NewRecorder()
			recorder.Header().Set("Vary", "Accept-Encoding")
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("SOT-DEL-015: status = %d、期待値 = %d、本文 = %q", recorder.Code, test.status, recorder.Body.String())
			}
			wantOrigin := ""
			if test.status == http.StatusNoContent {
				wantOrigin = origin
				if recorder.Body.Len() != 0 {
					t.Fatalf("SOT-DEL-015: preflight の本文 = %q", recorder.Body.String())
				}
			}
			assertCORSResponse(t, recorder, wantOrigin, true)
			if !strings.Contains(strings.Join(recorder.Header().Values("Vary"), ","), "Accept-Encoding") || len(started) != 0 {
				t.Fatal("SOT-DEL-015: 既存 Vary の消失または preflight での tool 実行")
			}
		})
	}
}

func TestHandlerCORSResponses(t *testing.T) {
	t.Parallel()

	const origin = "https://example.test"
	tests := []struct {
		name       string
		method     string
		body       string
		header     string
		value      string
		status     int
		wantOrigin string
	}{
		{name: "初期化", body: initializeRequestBody(), status: 200, wantOrigin: origin},
		{name: "ツール一覧", body: `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, status: 200, wantOrigin: origin},
		{name: "通知", body: `{"jsonrpc":"2.0","method":"notifications/initialized"}`, status: 202, wantOrigin: origin},
		{name: "不正JSON", body: `{`, status: 400, wantOrigin: origin},
		{name: "不正protocol", body: toolCallRequestBody("wait"), header: protocolVersionHeader, value: "invalid", status: 400, wantOrigin: origin},
		{name: "session拒否", body: initializeRequestBody(), header: sessionIDHeader, status: 400, wantOrigin: origin},
		{name: "本文上限", body: strings.Repeat("a", maxRequestBodyBytes+1), status: 413, wantOrigin: origin},
		{name: "SDKのmedia type拒否", body: initializeRequestBody(), header: "Content-Type", value: "text/plain", status: 415, wantOrigin: origin},
		{name: "SDKのAccept拒否", body: initializeRequestBody(), header: "Accept", value: "text/plain", status: 400, wantOrigin: origin},
		{name: "GET", method: "GET", status: 405, wantOrigin: origin},
		{name: "Origin拒否", body: initializeRequestBody(), header: "Origin", value: "https://other.test", status: 403},
		{name: "GETのOrigin拒否優先", method: "GET", header: "Origin", value: "https://other.test", status: 403},
	}
	handler := NewHandler(newTestServer(nil, nil), Options{AllowedOrigins: []string{origin}})
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			method := test.method
			if method == "" {
				method = http.MethodPost
			}
			request := newJSONRequest(t, method, "/mcp", test.body)
			request.Header.Set("Origin", origin)
			request.Header.Set(protocolVersionHeader, requiredProtocolVersion)
			if test.header != "" {
				request.Header.Set(test.header, test.value)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("SOT-DEL-015: status = %d、期待値 = %d、本文 = %q", recorder.Code, test.status, recorder.Body.String())
			}
			assertCORSResponse(t, recorder, test.wantOrigin, false)
			if test.status == http.StatusOK && recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatal("SOT-DEL-013: JSON 応答が必要です")
			}
		})
	}
}

func assertCORSResponse(t *testing.T, recorder *httptest.ResponseRecorder, origin string, preflight bool) {
	t.Helper()

	header := recorder.Header()
	if got := header.Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("SOT-DEL-015: 許可 Origin = %q、期待値 = %q", got, origin)
	}
	methods, names := "", ""
	if preflight && origin != "" {
		methods, names = "POST", "Accept, Content-Type, MCP-Protocol-Version"
	}
	if header.Get("Access-Control-Allow-Methods") != methods || header.Get("Access-Control-Allow-Headers") != names {
		t.Fatalf("SOT-DEL-015: 許可 method または header が不正です: %v", header)
	}
	for _, name := range []string{"Access-Control-Allow-Credentials", "Access-Control-Expose-Headers", "Access-Control-Max-Age", "Mcp-Session-Id", "Set-Cookie"} {
		if len(header.Values(name)) != 0 {
			t.Fatalf("SOT-DEL-015: 不要な応答 header: %s", name)
		}
	}
	vary := strings.Split(strings.ReplaceAll(strings.Join(header.Values("Vary"), ","), " ", ""), ",")
	if !slices.Contains(vary, "Origin") || (preflight && (!slices.Contains(vary, "Access-Control-Request-Method") || !slices.Contains(vary, "Access-Control-Request-Headers"))) {
		t.Fatalf("SOT-DEL-015: Vary が不足しています: %v", vary)
	}
}
