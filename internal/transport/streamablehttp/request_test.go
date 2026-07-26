package streamablehttp

import (
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadRequestClassifiesMessages(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body           string
		wantInitialize bool
		wantToolCall   bool
	}{
		"initialize": {
			body:           initializeRequestBody(),
			wantInitialize: true,
		},
		"tools call": {
			body:         toolCallRequestBody("wait"),
			wantToolCall: true,
		},
		"notification": {
			body: `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		},
		"response": {
			body: `{"jsonrpc":"2.0","id":1,"result":{}}`,
		},
		"invalid method type": {
			body: `{"jsonrpc":"2.0","id":1,"method":1}`,
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest("POST", "/mcp", strings.NewReader(test.body))
			body, metadata, validationErr := readRequest(request)
			if validationErr != nil {
				t.Fatalf("readRequest() のエラー = %v", validationErr)
			}
			if string(body) != test.body {
				t.Fatalf("body = %q, want %q", body, test.body)
			}
			if metadata.isInitialize != test.wantInitialize {
				t.Fatalf(
					"isInitialize = %t, want %t",
					metadata.isInitialize,
					test.wantInitialize,
				)
			}
			if metadata.isToolCall != test.wantToolCall {
				t.Fatalf("isToolCall = %t, want %t", metadata.isToolCall, test.wantToolCall)
			}
		})
	}
}

func TestReadRequestAcceptsExactBodyLimit(t *testing.T) {
	t.Parallel()

	prefix := initializeRequestBody()
	body := prefix + strings.Repeat(" ", maxRequestBodyBytes-len(prefix))
	request := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))

	got, metadata, validationErr := readRequest(request)
	if validationErr != nil {
		t.Fatalf("readRequest() のエラー = %v", validationErr)
	}
	if len(got) != maxRequestBodyBytes {
		t.Fatalf("body bytes = %d, want %d", len(got), maxRequestBodyBytes)
	}
	if !metadata.isInitialize {
		t.Fatal("initialize として分類されませんでした")
	}
}

func TestReadRequestRejectsInvalidEnvelope(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		body       string
		wantStatus int
	}{
		"empty": {
			body:       "",
			wantStatus: 400,
		},
		"array": {
			body:       `[{"jsonrpc":"2.0"}]`,
			wantStatus: 400,
		},
		"concatenated": {
			body:       `{"jsonrpc":"2.0"}{"jsonrpc":"2.0"}`,
			wantStatus: 400,
		},
		"malformed": {
			body:       `{"jsonrpc":`,
			wantStatus: 400,
		},
		"too large": {
			body:       strings.Repeat(" ", maxRequestBodyBytes+1),
			wantStatus: 413,
		},
	}

	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest("POST", "/mcp", strings.NewReader(test.body))
			_, _, validationErr := readRequest(request)
			if validationErr == nil {
				t.Fatal("readRequest() が入力を受理しました")
			}
			if validationErr.status != test.wantStatus {
				t.Fatalf("status = %d, want %d", validationErr.status, test.wantStatus)
			}
		})
	}
}

func TestReadRequestRejectsBodyReadFailure(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest("POST", "/mcp", nil)
	request.Body = io.NopCloser(errorReader{})

	_, _, validationErr := readRequest(request)
	if validationErr == nil {
		t.Fatal("readRequest() が読取り失敗を受理しました")
	}
	if validationErr.status != 400 {
		t.Fatalf("status = %d, want 400", validationErr.status)
	}
}

func TestRemoteIdentityNormalizesIPRepresentations(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"127.0.0.1:40000":      "127.0.0.1",
		"[0:0:0:0:0:0:0:1]:42": "::1",
		"malformed":            "malformed",
	}
	for remoteAddress, want := range tests {
		if got := remoteIdentity(remoteAddress); got != want {
			t.Fatalf("remoteIdentity(%q) = %q, want %q", remoteAddress, got, want)
		}
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("読取り失敗")
}
