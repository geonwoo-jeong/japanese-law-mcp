package lawv1

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	maximumResponseBytes     = 2 * 1024 * 1024
	maximumDecompressedBytes = 4 * 1024 * 1024
	maximumRetries           = 3
)

var retryBackoffs = [...]time.Duration{
	time.Second,
	2 * time.Second,
	4 * time.Second,
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type sleeper func(context.Context, time.Duration) error

type clientDependencies struct {
	doer  httpDoer
	now   func() time.Time
	sleep sleeper
}

type updateListClient struct {
	dependencies clientDependencies
}

type fetchedResponse struct {
	statusCode        int
	body              []byte
	retrievedAt       time.Time
	processingContext context.Context
	cancelProcessing  context.CancelFunc
}

func newProductionClient() updateListClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableCompression = true
	return updateListClient{dependencies: clientDependencies{
		doer:  &http.Client{Transport: transport},
		now:   time.Now,
		sleep: sleepWithContext,
	}}
}

func newUpdateListClient(
	dependencies clientDependencies,
) (updateListClient, error) {
	if dependencies.doer == nil ||
		dependencies.now == nil ||
		dependencies.sleep == nil {
		return updateListClient{}, fmt.Errorf("e-Gov Version 1 client の依存関係が不足しています")
	}
	return updateListClient{dependencies: dependencies}, nil
}

func (c updateListClient) fetch(
	ctx context.Context,
	request updateListRequest,
) (fetchedResponse, error) {
	if ctx == nil {
		return fetchedResponse{}, fmt.Errorf("context は必須です")
	}
	for retry := 0; ; retry++ {
		response, err := c.do(ctx, request)
		if err != nil {
			return fetchedResponse{}, err
		}
		if response.StatusCode == http.StatusOK ||
			response.StatusCode == http.StatusNotFound {
			compressed, readErr := readTransferBody(ctx, response)
			if readErr != nil {
				return fetchedResponse{}, readErr
			}
			if !isXMLMediaType(response.Header.Get("Content-Type")) {
				return fetchedResponse{}, newSourceError(
					model.SourceErrorCodeSourceContractChanged,
					"",
				)
			}
			processingContext, cancel := context.WithTimeout(
				ctx,
				parseTimeout,
			)
			body, decodeErr := decodeResponseBody(
				ctx,
				processingContext,
				compressed,
				response.Header.Get("Content-Encoding"),
			)
			if decodeErr != nil {
				cancel()
				return fetchedResponse{}, decodeErr
			}
			return fetchedResponse{
				statusCode:        response.StatusCode,
				body:              body,
				retrievedAt:       c.dependencies.now().Round(0),
				processingContext: processingContext,
				cancelProcessing:  cancel,
			}, nil
		}

		discardResponseBody(response)
		code := codeForHTTPStatus(response.StatusCode)
		retryAfter, delay, hasRetryAfter := parseRetryAfter(
			response.Header.Get("Retry-After"),
			c.dependencies.now(),
		)
		if !isRetryStatus(response.StatusCode) || retry >= maximumRetries {
			return fetchedResponse{}, newSourceError(code, retryAfter)
		}
		if !hasRetryAfter {
			delay = retryBackoffs[retry]
		}
		if !canWait(ctx, c.dependencies.now(), delay) {
			return fetchedResponse{}, newSourceError(code, retryAfter)
		}
		if err := c.dependencies.sleep(ctx, delay); err != nil {
			return fetchedResponse{}, normalizeContextError(err)
		}
	}
}

func (c updateListClient) do(
	ctx context.Context,
	request updateListRequest,
) (*http.Response, error) {
	httpRequest, err := buildHTTPRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	response, err := c.dependencies.doer.Do(httpRequest)
	if err == nil {
		if response == nil || response.Body == nil {
			return nil, newSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		return response, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, normalizeContextError(ctxErr)
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return nil, newSourceError(model.SourceErrorCodeSourceTimeout, "")
	}
	return nil, newSourceError(model.SourceErrorCodeSourceUnavailable, "")
}

func readTransferBody(
	ctx context.Context,
	response *http.Response,
) ([]byte, error) {
	defer func() {
		_ = response.Body.Close()
	}()
	if response.ContentLength > maximumResponseBytes {
		return nil, newSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	compressed, err := readAtMost(
		response.Body,
		maximumResponseBytes,
	)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, normalizeContextError(contextErr)
		}
		return nil, err
	}
	return compressed, nil
}

func decodeResponseBody(
	parent context.Context,
	processingContext context.Context,
	compressed []byte,
	contentEncoding string,
) ([]byte, error) {
	if err := processingBudgetError(parent, processingContext); err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(contentEncoding)) {
	case "", "identity":
		return readWithProcessingBudget(
			parent,
			processingContext,
			bytes.NewReader(compressed),
			maximumDecompressedBytes,
		)
	case "gzip":
		reader, gzipErr := gzip.NewReader(&xmlContextReader{
			ctx:    processingContext,
			reader: bytes.NewReader(compressed),
		})
		if gzipErr != nil {
			if err := processingBudgetError(
				parent,
				processingContext,
			); err != nil {
				return nil, err
			}
			return nil, newSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		defer func() {
			_ = reader.Close()
		}()
		return readWithProcessingBudget(
			parent,
			processingContext,
			reader,
			maximumDecompressedBytes,
		)
	default:
		return nil, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func readWithProcessingBudget(
	parent context.Context,
	processingContext context.Context,
	reader io.Reader,
	maximum int64,
) ([]byte, error) {
	value, err := readAtMost(
		&xmlContextReader{
			ctx:    processingContext,
			reader: reader,
		},
		maximum,
	)
	if budgetErr := processingBudgetError(
		parent,
		processingContext,
	); budgetErr != nil {
		return nil, budgetErr
	}
	return value, err
}

func processingBudgetError(
	parent context.Context,
	processingContext context.Context,
) error {
	if parentErr := parent.Err(); parentErr != nil {
		return normalizeContextError(parentErr)
	}
	if errors.Is(processingContext.Err(), context.DeadlineExceeded) {
		return newSourceError(model.SourceErrorCodeSourceProcessingLimit, "")
	}
	if errors.Is(processingContext.Err(), context.Canceled) {
		return context.Canceled
	}
	return nil
}

func readAtMost(reader io.Reader, maximum int64) ([]byte, error) {
	value, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		return nil, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if int64(len(value)) > maximum {
		return nil, newSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	return value, nil
}

func discardResponseBody(response *http.Response) {
	defer func() {
		_ = response.Body.Close()
	}()
	_, _ = io.Copy(
		io.Discard,
		io.LimitReader(response.Body, maximumResponseBytes+1),
	)
}

func isXMLMediaType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == "text/xml"
}

func isRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError &&
			status <= 599
}

func codeForHTTPStatus(status int) model.SourceErrorCode {
	if status == http.StatusTooManyRequests {
		return model.SourceErrorCodeRateLimited
	}
	if status >= http.StatusInternalServerError && status <= 599 {
		return model.SourceErrorCodeSourceUnavailable
	}
	return model.SourceErrorCodeInvalidSourceResponse
}

func parseRetryAfter(
	value string,
	now time.Time,
) (string, time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, false
	}
	if seconds, err := strconv.ParseUint(value, 10, 31); err == nil {
		return value, time.Duration(seconds) * time.Second, true
	}
	date, err := http.ParseTime(value)
	if err != nil || !date.After(now) {
		return "", 0, false
	}
	return value, date.Sub(now), true
}

func canWait(
	ctx context.Context,
	now time.Time,
	delay time.Duration,
) bool {
	if delay < 0 {
		return false
	}
	deadline, exists := ctx.Deadline()
	return !exists || !now.Add(delay).After(deadline)
}

func sleepWithContext(
	ctx context.Context,
	delay time.Duration,
) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func normalizeContextError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return newSourceError(model.SourceErrorCodeSourceTimeout, "")
	default:
		return newSourceError(model.SourceErrorCodeSourceUnavailable, "")
	}
}
