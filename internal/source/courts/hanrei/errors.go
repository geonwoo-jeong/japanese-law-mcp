package hanrei

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type operation string

const operationSearch operation = "GET /hanrei/search1/index.html"

func (operation) SourceOperationProviderID() string {
	return providerID
}

func (o operation) SourceOperationName() string {
	return string(o)
}

func (o operation) ValidateSourceOperation() error {
	if o != operationSearch && o != operationRead {
		return fmt.Errorf("裁判所 operation が定義されていません")
	}
	return nil
}

func newSearchSourceError(
	code model.SourceErrorCode,
	retryAfter string,
) error {
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   Descriptor(),
		Capability: judicialDecisionSearchCapability(),
		Operation:  operationSearch,
		RetryAfter: retryAfter,
	})
	if err != nil {
		return fmt.Errorf("裁判所の情報源エラーを正規化できません: %w", err)
	}
	return sourceError
}

func normalizeSearchContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return newSearchSourceError(model.SourceErrorCodeSourceTimeout, "")
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return newSearchSourceError(model.SourceErrorCodeSourceUnavailable, "")
}

func codeForSearchStatus(status int) model.SourceErrorCode {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return model.SourceErrorCodeSourceAuthFailed
	case status == http.StatusTooManyRequests:
		return model.SourceErrorCodeRateLimited
	case status >= http.StatusInternalServerError && status <= 599:
		return model.SourceErrorCodeSourceUnavailable
	default:
		return model.SourceErrorCodeInvalidSourceResponse
	}
}

func retryAfterValue(value string, now time.Time) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if _, err := strconv.ParseUint(value, 10, 31); err == nil {
		return value
	}
	date, err := http.ParseTime(value)
	if err != nil || !date.After(now) {
		return ""
	}
	return value
}
