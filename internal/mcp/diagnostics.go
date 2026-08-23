package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const diagnosticToolCallMethod = "tools/call"

type diagnosticEvent struct {
	Component string          `json:"component"`
	Operation string          `json:"operation"`
	ErrorCode model.ErrorCode `json:"errorCode,omitempty"`
}

type diagnosticsWriter struct {
	mu     sync.Mutex
	output io.Writer
}

// AddDiagnostics は、SOT-ARCH-008 に従う一時診断を受信ミドルウェアへ追加する。
func AddDiagnostics(server *sdk.Server, output io.Writer) error {
	if server == nil {
		return fmt.Errorf("diagnostics を追加する server がありません")
	}
	if output == nil {
		return fmt.Errorf("diagnostics の出力先がありません")
	}
	writer := &diagnosticsWriter{output: output}
	server.AddReceivingMiddleware(writer.middleware)
	return nil
}

func (w *diagnosticsWriter) middleware(next sdk.MethodHandler) sdk.MethodHandler {
	return func(
		ctx context.Context,
		method string,
		req sdk.Request,
	) (sdk.Result, error) {
		result, err := next(ctx, method, req)
		if method != diagnosticToolCallMethod {
			return result, err
		}
		_ = w.write(diagnosticEvent{
			Component: "mcp",
			Operation: diagnosticOperation(req, result, err),
			ErrorCode: diagnosticErrorCode(result, err),
		})
		return result, err
	}
}

func diagnosticOperation(
	req sdk.Request,
	result sdk.Result,
	err error,
) string {
	call, ok := req.(*sdk.CallToolRequest)
	if !ok || call.Params == nil || call.Params.Name == "" {
		return diagnosticToolCallMethod
	}
	return knownDiagnosticOperation(call.Params.Name)
}

func knownDiagnosticOperation(operation string) string {
	switch operation {
	case "get_article",
		"get_judicial_case",
		"get_law",
		"list_law_revisions",
		"list_law_updates",
		"query_legal_information",
		"search_judicial_cases",
		"search_law_content",
		"search_laws":
		return operation
	default:
		return diagnosticToolCallMethod
	}
}

func diagnosticErrorCode(result sdk.Result, err error) model.ErrorCode {
	if err != nil {
		return model.ErrorCodeInternalError
	}
	callResult, ok := result.(*sdk.CallToolResult)
	if !ok || callResult == nil {
		return model.ErrorCodeInternalError
	}
	if !callResult.IsError {
		return ""
	}
	if len(callResult.Content) != 1 {
		return model.ErrorCodeInternalError
	}
	content, ok := callResult.Content[0].(*sdk.TextContent)
	if !ok {
		return model.ErrorCodeInternalError
	}
	var payload struct {
		Code model.ErrorCode `json:"code"`
	}
	if json.Unmarshal([]byte(content.Text), &payload) != nil {
		return model.ErrorCodeInternalError
	}
	if _, buildErr := model.NewErrorResult(model.ErrorResultValues{
		Code: payload.Code,
	}); buildErr != nil {
		return model.ErrorCodeInternalError
	}
	return payload.Code
}

func (w *diagnosticsWriter) write(event diagnosticEvent) error {
	line, err := json.Marshal(event)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.output.Write(line)
	return err
}
