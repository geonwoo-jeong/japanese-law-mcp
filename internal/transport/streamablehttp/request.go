package streamablehttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
)

const (
	maxRequestBodyBytes     = 1 << 20
	maxConcurrentToolCalls  = 4
	initializeMethod        = "initialize"
	toolCallMethod          = "tools/call"
	protocolVersionHeader   = "MCP-Protocol-Version"
	sessionIDHeader         = "Mcp-Session-Id"
	requiredProtocolVersion = "2025-11-25"
)

type requestMetadata struct {
	isInitialize bool
	isToolCall   bool
}

type requestValidationError struct {
	status  int
	message string
}

func (e *requestValidationError) Error() string {
	return e.message
}

func readRequest(
	request *http.Request,
) ([]byte, requestMetadata, *requestValidationError) {
	if request.Body == nil {
		return nil, requestMetadata{}, invalidRequest(
			http.StatusBadRequest,
			"request body が必要です",
		)
	}
	defer request.Body.Close()

	body, err := io.ReadAll(io.LimitReader(request.Body, maxRequestBodyBytes+1))
	if err != nil {
		return nil, requestMetadata{}, invalidRequest(
			http.StatusBadRequest,
			"request body を読み取れません",
		)
	}
	if len(body) > maxRequestBodyBytes {
		return nil, requestMetadata{}, invalidRequest(
			http.StatusRequestEntityTooLarge,
			"request body は 1 MiB 以下でなければなりません",
		)
	}

	metadata, validationErr := inspectMessage(body)
	if validationErr != nil {
		return nil, requestMetadata{}, validationErr
	}
	return body, metadata, nil
}

func inspectMessage(body []byte) (requestMetadata, *requestValidationError) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return requestMetadata{}, invalidRequest(
			http.StatusBadRequest,
			"top-level JSON object が必要です",
		)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	var object map[string]json.RawMessage
	if err := decoder.Decode(&object); err != nil {
		return requestMetadata{}, invalidRequest(
			http.StatusBadRequest,
			"JSON message を解釈できません",
		)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return requestMetadata{}, invalidRequest(
			http.StatusBadRequest,
			"一つの JSON message だけを受け付けます",
		)
	}

	var method string
	if rawMethod, exists := object["method"]; exists {
		_ = json.Unmarshal(rawMethod, &method)
	}
	return requestMetadata{
		isInitialize: method == initializeMethod,
		isToolCall:   method == toolCallMethod,
	}, nil
}

func invalidRequest(status int, message string) *requestValidationError {
	return &requestValidationError{status: status, message: message}
}

func originAllowed(header http.Header, allowedOrigins []string) bool {
	origins := header.Values("Origin")
	if len(origins) == 0 {
		return true
	}
	if len(origins) != 1 {
		return false
	}
	for _, allowed := range allowedOrigins {
		if origins[0] == allowed {
			return true
		}
	}
	return false
}

func hasSessionHeader(header http.Header) bool {
	return len(header.Values(sessionIDHeader)) != 0
}

func hasRequiredProtocolVersion(header http.Header) bool {
	versions := header.Values(protocolVersionHeader)
	return len(versions) == 1 && versions[0] == requiredProtocolVersion
}

func remoteIdentity(remoteAddress string) string {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil || host == "" {
		return remoteAddress
	}
	if ip := net.ParseIP(strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")); ip != nil {
		return ip.String()
	}
	return host
}
