package hanreipdf

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type operation string

const (
	operationFetch operation = "GET /assets/hanrei/{id}.pdf"
	operationParse operation = "private worker parse"
)

func (operation) SourceOperationProviderID() string { return providerID }

func (o operation) SourceOperationName() string { return string(o) }

func (o operation) ValidateSourceOperation() error {
	switch o {
	case operationFetch, operationParse:
		return nil
	default:
		return fmt.Errorf("裁判所 PDF operation が定義されていません")
	}
}

func newSourceError(
	code model.SourceErrorCode,
	op operation,
	retryAfter string,
) model.SourceError {
	if code == model.SourceErrorCodeSourceAuthFailed {
		code = model.SourceErrorCodeInvalidSourceResponse
		retryAfter = ""
	}
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: mustCapability(judicialcasecitationextract.CapabilityID),
		Operation:  op,
		RetryAfter: retryAfter,
	})
	if err != nil {
		panic(fmt.Sprintf("裁判所 PDF 情報源エラーを正規化できません: %v", err))
	}
	return sourceError
}

func normalizeContextError(
	ctx context.Context,
	op operation,
	err error,
) error {
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return newSourceError(model.SourceErrorCodeSourceTimeout, op, "")
	}
	return newSourceError(model.SourceErrorCodeSourceUnavailable, op, "")
}

func codeForStatus(status int) model.SourceErrorCode {
	switch {
	case status == http.StatusNotFound:
		return model.SourceErrorCodeInvalidSourceResponse
	case status == http.StatusTooManyRequests:
		return model.SourceErrorCodeRateLimited
	case status >= http.StatusInternalServerError && status <= 599:
		return model.SourceErrorCodeSourceUnavailable
	default:
		return model.SourceErrorCodeInvalidSourceResponse
	}
}

func errorForHTTPStatus(status int) error {
	if status == http.StatusNotFound {
		return judicialcasecitationextract.ErrNotFound
	}
	return newSourceError(codeForStatus(status), operationFetch, "")
}
