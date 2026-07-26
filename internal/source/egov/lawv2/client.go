package lawv2

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
	maximumResponseBytes     = 8 * 1024 * 1024
	maximumDecompressedBytes = 16 * 1024 * 1024
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

type lawClient struct {
	dependencies clientDependencies
}

type fetchedResponse struct {
	body        []byte
	retrievedAt time.Time
}

func newProductionClient() lawClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableCompression = true
	return lawClient{dependencies: clientDependencies{
		doer:  &http.Client{Transport: transport},
		now:   time.Now,
		sleep: sleepWithContext,
	}}
}

func newLawClient(dependencies clientDependencies) (lawClient, error) {
	if dependencies.doer == nil ||
		dependencies.now == nil ||
		dependencies.sleep == nil {
		return lawClient{}, fmt.Errorf("e-Gov client の依存関係が不足しています")
	}
	return lawClient{dependencies: dependencies}, nil
}

func (c lawClient) fetch(
	ctx context.Context,
	request lawSearchRequest,
) (fetchedResponse, error) {
	if ctx == nil {
		return fetchedResponse{}, fmt.Errorf("context は必須です")
	}
	for retry := 0; ; retry++ {
		response, err := c.do(ctx, request)
		if err != nil {
			return fetchedResponse{}, err
		}
		body, readErr := readResponseBody(response)
		if readErr != nil {
			return fetchedResponse{}, readErr
		}
		if response.StatusCode >= http.StatusOK &&
			response.StatusCode < http.StatusMultipleChoices {
			if !isJSONMediaType(response.Header.Get("Content-Type")) {
				return fetchedResponse{}, newSourceError(
					model.SourceErrorCodeSourceContractChanged,
					"",
				)
			}
			return fetchedResponse{
				body:        body,
				retrievedAt: c.dependencies.now().Round(0),
			}, nil
		}

		code := codeForStatus(response.StatusCode)
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

func (c lawClient) do(
	ctx context.Context,
	request lawSearchRequest,
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

func readResponseBody(response *http.Response) ([]byte, error) {
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(
			io.Discard,
			io.LimitReader(response.Body, maximumResponseBytes+1),
		)
		return nil, nil
	}
	if response.ContentLength > maximumResponseBytes {
		return nil, newSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	compressed, err := readAtMost(response.Body, maximumResponseBytes)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Encoding"))) {
	case "", "identity":
		if len(compressed) > maximumDecompressedBytes {
			return nil, newSourceError(
				model.SourceErrorCodeSourceResponseTooLarge,
				"",
			)
		}
		return compressed, nil
	case "gzip":
		reader, gzipErr := gzip.NewReader(bytes.NewReader(compressed))
		if gzipErr != nil {
			return nil, newSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		defer func() {
			_ = reader.Close()
		}()
		return readAtMost(reader, maximumDecompressedBytes)
	default:
		return nil, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func readAtMost(reader io.Reader, maximum int64) ([]byte, error) {
	limited := io.LimitReader(reader, maximum+1)
	value, err := io.ReadAll(limited)
	if err != nil {
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return nil, normalizeContextError(err)
		}
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

func isJSONMediaType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == "application/json"
}

func isRetryStatus(status int) bool {
	switch status {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func codeForStatus(status int) model.SourceErrorCode {
	switch status {
	case http.StatusTooManyRequests:
		return model.SourceErrorCodeRateLimited
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return model.SourceErrorCodeSourceUnavailable
	default:
		return model.SourceErrorCodeInvalidSourceResponse
	}
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

func canWait(ctx context.Context, now time.Time, delay time.Duration) bool {
	if delay < 0 {
		return false
	}
	deadline, exists := ctx.Deadline()
	return !exists || !now.Add(delay).After(deadline)
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
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
	if errors.Is(err, context.DeadlineExceeded) {
		return newSourceError(model.SourceErrorCodeSourceTimeout, "")
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return newSourceError(model.SourceErrorCodeSourceUnavailable, "")
}
