package mcp

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/requestpacing"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const requestPacingToolCallMethod = "tools/call"

func requestPacingMiddleware(next sdk.MethodHandler) sdk.MethodHandler {
	return func(
		ctx context.Context,
		method string,
		request sdk.Request,
	) (sdk.Result, error) {
		if method == requestPacingToolCallMethod {
			ctx = requestpacing.WithScope(ctx)
		}
		return next(ctx, method, request)
	}
}
