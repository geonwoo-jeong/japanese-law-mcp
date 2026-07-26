package streamablehttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandlerRejectsNonMCPPath(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPost, "/other", initializeRequestBody())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestHandlerRejectsGET(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	request.Header.Set("Origin", "https://not-allowed.example")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if allow := recorder.Header().Get("Allow"); allow != "POST" {
		t.Fatalf("Allow = %q, want %q", allow, "POST")
	}
}

func TestHandlerRejectsUnsupportedMethod(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPut, "/mcp", initializeRequestBody())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if allow := recorder.Header().Get("Allow"); allow != "POST" {
		t.Fatalf("Allow = %q, want %q", allow, "POST")
	}
}

func TestHandlerRejectsDisallowedOrigin(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody())
	request.Header.Set("Origin", "https://example.test")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestHandlerAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{
		AllowedOrigins: []string{"https://example.test"},
	})
	request := newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody())
	request.Header.Set("Origin", "https://example.test")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func TestHandlerAllowsRequestWithoutOrigin(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func TestHandlerCopiesAllowedOrigins(t *testing.T) {
	t.Parallel()

	origins := []string{"https://example.test"}
	handler := NewHandler(newTestServer(nil, nil), Options{AllowedOrigins: origins})
	origins[0] = "https://changed.test"

	request := newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody())
	request.Header.Set("Origin", "https://example.test")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
}

func TestHandlerRejectsMultipleOrigins(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{
		AllowedOrigins: []string{"https://example.test"},
	})
	request := newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody())
	request.Header.Add("Origin", "https://example.test")
	request.Header.Add("Origin", "https://example.test")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestHandlerRejectsIncomingSessionID(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody())
	request.Header.Set("Mcp-Session-Id", "unexpected")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlerRejectsEmptyIncomingSessionHeader(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody())
	request.Header["Mcp-Session-Id"] = []string{""}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlerRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPost, "/mcp", strings.Repeat("a", maxRequestBodyBytes+1))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestHandlerDoesNotRejectExactBodyLimitAsTooLarge(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	prefix := initializeRequestBody()
	body := prefix + strings.Repeat(" ", maxRequestBodyBytes-len(prefix))
	request := newJSONRequest(t, http.MethodPost, "/mcp", body)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code == http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, body size = %d", recorder.Code, len(body))
	}
}

func TestHandlerRejectsBatchAndConcatenatedJSON(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})

	tests := map[string]string{
		"batch":        `[{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}]`,
		"concatenated": `{"jsonrpc":"2.0"}{"jsonrpc":"2.0"}`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			request := newJSONRequest(t, http.MethodPost, "/mcp", body)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandlerRequiresProtocolHeaderAfterInitialize(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})

	initializeRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		initializeRecorder,
		newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody()),
	)
	if initializeRecorder.Code != http.StatusOK {
		t.Fatalf("initialize status = %d, want %d, body = %q", initializeRecorder.Code, http.StatusOK, initializeRecorder.Body.String())
	}

	request := newJSONRequest(t, http.MethodPost, "/mcp", toolCallRequestBody("wait"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlerRejectsUnsupportedProtocolHeader(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPost, "/mcp", toolCallRequestBody("wait"))
	request.Header.Set("Mcp-Protocol-Version", "2025-06-18")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHandlerReturnsAcceptedForNotification(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(
		t,
		http.MethodPost,
		"/mcp",
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
	)
	request.Header.Set("Mcp-Protocol-Version", requiredProtocolVersion)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusAccepted, recorder.Body.String())
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("notification response body = %q", recorder.Body.String())
	}
}

func TestHandlerReturnsAcceptedForResponse(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(
		t,
		http.MethodPost,
		"/mcp",
		`{"jsonrpc":"2.0","id":1,"result":{}}`,
	)
	request.Header.Set("Mcp-Protocol-Version", requiredProtocolVersion)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusAccepted, recorder.Body.String())
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("response message の HTTP body = %q", recorder.Body.String())
	}
}

func TestHandlerRejectsWrongContentType(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody())
	request.Header.Set("Content-Type", "text/plain")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnsupportedMediaType)
	}
}

func TestHandlerDoesNotEmitSessionID(t *testing.T) {
	t.Parallel()

	handler := NewHandler(newTestServer(nil, nil), Options{})
	request := newJSONRequest(t, http.MethodPost, "/mcp", initializeRequestBody())

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if values := recorder.Header().Values("Mcp-Session-Id"); len(values) != 0 {
		t.Fatalf("Mcp-Session-Id = %#v, want absent", values)
	}
}

func TestHandlerLimitsConcurrentToolCallsPerRemoteAddress(t *testing.T) {
	t.Parallel()

	started := make(chan struct{}, 4)
	release := make(chan struct{})
	handler := NewHandler(newTestServer(started, release), Options{})

	var (
		wg      sync.WaitGroup
		results [4]*httptest.ResponseRecorder
	)
	for i := range results {
		wg.Add(1)
		recorder := httptest.NewRecorder()
		results[i] = recorder
		go func(rec *httptest.ResponseRecorder) {
			defer wg.Done()
			request := newJSONRequest(t, http.MethodPost, "/mcp", toolCallRequestBody("wait"))
			request.Header.Set("Mcp-Protocol-Version", "2025-11-25")
			request.RemoteAddr = "127.0.0.1:41000"
			handler.ServeHTTP(rec, request)
		}(recorder)
	}

	for range 4 {
		<-started
	}

	rejected := httptest.NewRecorder()
	rejectedRequest := newJSONRequest(t, http.MethodPost, "/mcp", toolCallRequestBody("wait"))
	rejectedRequest.Header.Set("Mcp-Protocol-Version", "2025-11-25")
	rejectedRequest.RemoteAddr = "127.0.0.1:41000"
	handler.ServeHTTP(rejected, rejectedRequest)

	if rejected.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rejected.Code, http.StatusTooManyRequests)
	}

	close(release)
	wg.Wait()
	for _, recorder := range results {
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %q", recorder.Code, http.StatusOK, recorder.Body.String())
		}
	}
}

func newTestServer(started chan<- struct{}, release <-chan struct{}) *sdk.Server {
	server := sdk.NewServer(
		&sdk.Implementation{Name: "test", Version: "test-version"},
		&sdk.ServerOptions{
			GetSessionID: func() string { return "" },
		},
	)
	sdk.AddTool(
		server,
		&sdk.Tool{Name: "wait", Description: "wait"},
		func(ctx context.Context, req *sdk.CallToolRequest, _ any) (*sdk.CallToolResult, any, error) {
			if started != nil {
				started <- struct{}{}
			}
			if release != nil {
				select {
				case <-release:
				case <-ctx.Done():
				}
			}
			return &sdk.CallToolResult{
				Content: []sdk.Content{&sdk.TextContent{Text: "ok"}},
			}, nil, nil
		},
	)
	return server
}

func newJSONRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = "127.0.0.1:40000"
	return request
}

func initializeRequestBody() string {
	return `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}`
}

func toolCallRequestBody(name string) string {
	return `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"` + name + `","arguments":{}}}`
}
