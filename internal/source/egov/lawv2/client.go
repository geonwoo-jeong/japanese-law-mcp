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

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	maximumResponseBytes         = 8 * 1024 * 1024
	maximumDecompressedBytes     = 16 * 1024 * 1024
	lawDocumentResponseBytes     = 16 * 1024 * 1024
	lawDocumentDecompressedBytes = 32 * 1024 * 1024
	maximumRetries               = 3
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

type sourceErrorFactory func(model.SourceErrorCode, string) error

type fetchSpec struct {
	build             func(context.Context) (*http.Request, error)
	responseBytes     int64
	decompressedBytes int64
	mediaType         string
	sourceError       sourceErrorFactory
	notFound          error
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
	return c.fetchWith(ctx, fetchSpec{
		build: func(requestContext context.Context) (*http.Request, error) {
			return buildHTTPRequest(requestContext, request)
		},
		responseBytes:     maximumResponseBytes,
		decompressedBytes: maximumDecompressedBytes,
		mediaType:         "application/json",
		sourceError:       newSourceError,
	})
}

func (c lawClient) fetchLawDocument(
	ctx context.Context,
	request lawDocumentRequest,
) (fetchedResponse, error) {
	return c.fetchWith(ctx, fetchSpec{
		build: func(requestContext context.Context) (*http.Request, error) {
			return buildLawDocumentHTTPRequest(requestContext, request)
		},
		responseBytes:     lawDocumentResponseBytes,
		decompressedBytes: lawDocumentDecompressedBytes,
		mediaType:         "application/xml",
		sourceError:       newLawDocumentSourceError,
		notFound:          lawdocumentread.ErrNotFound,
	})
}

func (c lawClient) fetchWith(
	ctx context.Context,
	spec fetchSpec,
) (fetchedResponse, error) {
	if ctx == nil {
		return fetchedResponse{}, fmt.Errorf("context は必須です")
	}
	for retry := 0; ; retry++ {
		response, err := c.do(ctx, spec)
		if err != nil {
			return fetchedResponse{}, err
		}
		body, readErr := readResponseBodyWithLimits(
			response,
			spec.responseBytes,
			spec.decompressedBytes,
			spec.sourceError,
		)
		if readErr != nil {
			return fetchedResponse{}, readErr
		}
		if response.StatusCode >= http.StatusOK &&
			response.StatusCode < http.StatusMultipleChoices {
			if !isMediaType(
				response.Header.Get("Content-Type"),
				spec.mediaType,
			) {
				return fetchedResponse{}, spec.sourceError(
					model.SourceErrorCodeSourceContractChanged,
					"",
				)
			}
			return fetchedResponse{
				body:        body,
				retrievedAt: c.dependencies.now().Round(0),
			}, nil
		}
		if response.StatusCode == http.StatusNotFound &&
			spec.notFound != nil {
			return fetchedResponse{}, spec.notFound
		}

		code := codeForStatus(response.StatusCode)
		retryAfter, delay, hasRetryAfter := parseRetryAfter(
			response.Header.Get("Retry-After"),
			c.dependencies.now(),
		)
		if !isRetryStatus(response.StatusCode) || retry >= maximumRetries {
			return fetchedResponse{}, spec.sourceError(code, retryAfter)
		}
		if !hasRetryAfter {
			delay = retryBackoffs[retry]
		}
		if !canWait(ctx, c.dependencies.now(), delay) {
			return fetchedResponse{}, spec.sourceError(code, retryAfter)
		}
		if err := c.dependencies.sleep(ctx, delay); err != nil {
			return fetchedResponse{}, normalizeContextErrorWithFactory(
				err,
				spec.sourceError,
			)
		}
	}
}

func (c lawClient) do(
	ctx context.Context,
	spec fetchSpec,
) (*http.Response, error) {
	httpRequest, err := spec.build(ctx)
	if err != nil {
		return nil, err
	}
	response, err := c.dependencies.doer.Do(httpRequest)
	if err == nil {
		if response == nil || response.Body == nil {
			return nil, spec.sourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		return response, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, normalizeContextErrorWithFactory(
			ctxErr,
			spec.sourceError,
		)
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return nil, spec.sourceError(
			model.SourceErrorCodeSourceTimeout,
			"",
		)
	}
	return nil, spec.sourceError(
		model.SourceErrorCodeSourceUnavailable,
		"",
	)
}

func readResponseBody(response *http.Response) ([]byte, error) {
	return readResponseBodyWithLimits(
		response,
		maximumResponseBytes,
		maximumDecompressedBytes,
		newSourceError,
	)
}

func readResponseBodyWithLimits(
	response *http.Response,
	responseBytes int64,
	decompressedBytes int64,
	sourceError sourceErrorFactory,
) ([]byte, error) {
	defer func() {
		_ = response.Body.Close()
	}()
	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(
			io.Discard,
			io.LimitReader(response.Body, responseBytes+1),
		)
		return nil, nil
	}
	if response.ContentLength > responseBytes {
		return nil, sourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	compressed, err := readAtMostWithFactory(
		response.Body,
		responseBytes,
		sourceError,
	)
	if err != nil {
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Encoding"))) {
	case "", "identity":
		if int64(len(compressed)) > decompressedBytes {
			return nil, sourceError(
				model.SourceErrorCodeSourceResponseTooLarge,
				"",
			)
		}
		return compressed, nil
	case "gzip":
		reader, gzipErr := gzip.NewReader(bytes.NewReader(compressed))
		if gzipErr != nil {
			return nil, sourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		defer func() {
			_ = reader.Close()
		}()
		return readAtMostWithFactory(
			reader,
			decompressedBytes,
			sourceError,
		)
	default:
		return nil, sourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func readAtMostWithFactory(
	reader io.Reader,
	maximum int64,
	sourceError sourceErrorFactory,
) ([]byte, error) {
	limited := io.LimitReader(reader, maximum+1)
	value, err := io.ReadAll(limited)
	if err != nil {
		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return nil, normalizeContextErrorWithFactory(err, sourceError)
		}
		return nil, sourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if int64(len(value)) > maximum {
		return nil, sourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	return value, nil
}

func isMediaType(value string, expected string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == expected
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
	return normalizeContextErrorWithFactory(err, newSourceError)
}

func normalizeContextErrorWithFactory(
	err error,
	sourceError sourceErrorFactory,
) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return sourceError(model.SourceErrorCodeSourceTimeout, "")
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	return sourceError(model.SourceErrorCodeSourceUnavailable, "")
}
