// Package streamablehttp は、loopback 限定の Streamable HTTP 実行方式を提供する。
package streamablehttp

import (
	"bytes"
	"io"
	"net/http"
	"slices"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const endpointPath = "/mcp"

// NewHandler は、SOT-DEL-013 と SOT-DEL-008 に従う HTTP handler を返す。
func NewHandler(server *sdk.Server, options Options) http.Handler {
	allowedOrigins := slices.Clone(options.AllowedOrigins)
	limiter := newConcurrencyLimiter(maxConcurrentToolCalls)
	sdkHandler := sdk.NewStreamableHTTPHandler(
		func(*http.Request) *sdk.Server {
			return server
		},
		&sdk.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
			EventStore:   nil,
		},
	)

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != endpointPath {
			http.NotFound(writer, request)
			return
		}
		if request.Method != http.MethodPost {
			writer.Header().Set("Allow", http.MethodPost)
			http.Error(writer, "POST 以外の HTTP method は使用できません", http.StatusMethodNotAllowed)
			return
		}
		if !originAllowed(request.Header, allowedOrigins) {
			http.Error(writer, "この Origin からの接続は許可されていません", http.StatusForbidden)
			return
		}
		if hasSessionHeader(request.Header) {
			http.Error(
				writer,
				"Mcp-Session-Id は使用できません",
				http.StatusBadRequest,
			)
			return
		}

		body, metadata, validationErr := readRequest(request)
		if validationErr != nil {
			http.Error(writer, validationErr.Error(), validationErr.status)
			return
		}
		if !metadata.isInitialize && !hasRequiredProtocolVersion(request.Header) {
			http.Error(
				writer,
				"MCP-Protocol-Version は 2025-11-25 でなければなりません",
				http.StatusBadRequest,
			)
			return
		}

		release, err := limiter.acquire(
			remoteIdentity(request.RemoteAddr),
			metadata.isToolCall,
		)
		if err != nil {
			http.Error(writer, "同時に実行できるツール呼び出しの上限を超えました", http.StatusTooManyRequests)
			return
		}
		if release != nil {
			defer release()
		}

		delegated := request.Clone(request.Context())
		delegated.Body = io.NopCloser(bytes.NewReader(body))
		delegated.ContentLength = int64(len(body))
		sdkHandler.ServeHTTP(writer, delegated)
	})
}
