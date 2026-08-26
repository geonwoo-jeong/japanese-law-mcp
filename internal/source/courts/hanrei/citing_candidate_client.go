package hanrei

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"golang.org/x/net/html/charset"
)

const (
	maximumCitingCandidateResponseBytes     = 4 * 1024 * 1024
	maximumCitingCandidateDecompressedBytes = 8 * 1024 * 1024
)

func fetchCitingCandidateResponse(
	ctx context.Context,
	doer httpDoer,
	now func() time.Time,
	query string,
	remainingBytes int,
) (fetchedSearchResponse, int, error) {
	if remainingBytes <= 0 {
		return fetchedSearchResponse{}, 0, newSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	request, err := buildSearchHTTPRequest(ctx, query)
	if err != nil {
		return fetchedSearchResponse{}, 0, err
	}
	response, err := doer.Do(request)
	if err != nil {
		return fetchedSearchResponse{}, 0, classifySearchDoError(ctx, err)
	}
	if response == nil || response.Body == nil {
		return fetchedSearchResponse{}, 0, newSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	defer func() { _ = response.Body.Close() }()
	if response.ContentLength > int64(remainingBytes) {
		return fetchedSearchResponse{}, 0, newSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		consumed, readErr := consumeCitingCandidateFailureBody(
			ctx,
			response.Body,
			remainingBytes,
		)
		if readErr != nil {
			return fetchedSearchResponse{}, consumed, readErr
		}
		code := codeForSearchStatus(response.StatusCode)
		retryAfter := ""
		if code == model.SourceErrorCodeRateLimited || code == model.SourceErrorCodeSourceUnavailable {
			retryAfter = retryAfterValue(response.Header.Get("Retry-After"), now())
		}
		return fetchedSearchResponse{}, consumed, newSearchSourceError(code, retryAfter)
	}
	if !isHTMLMediaType(response.Header.Get("Content-Type")) {
		return fetchedSearchResponse{}, 0, newSearchSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	finalURL := request.URL
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL
	}
	if !isCourtsHTTPSOrigin(finalURL) {
		return fetchedSearchResponse{}, 0, newSearchSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}
	body, err := io.ReadAll(io.LimitReader(
		&searchContextReader{ctx: ctx, reader: response.Body},
		int64(remainingBytes)+1,
	))
	if err != nil {
		if ctx.Err() != nil {
			return fetchedSearchResponse{}, len(body), normalizeSearchContextError(ctx.Err())
		}
		return fetchedSearchResponse{}, len(body), newSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if len(body) > remainingBytes {
		return fetchedSearchResponse{}, len(body), newSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	return fetchedSearchResponse{
		encodedBody:     body,
		contentType:     response.Header.Get("Content-Type"),
		contentEncoding: response.Header.Get("Content-Encoding"),
		fetchedURL:      finalURL.String(),
		retrievedAt:     now().Round(0),
	}, len(body), nil
}

func consumeCitingCandidateFailureBody(
	ctx context.Context,
	reader io.Reader,
	remainingBytes int,
) (int, error) {
	body, err := io.ReadAll(io.LimitReader(
		&searchContextReader{ctx: ctx, reader: reader},
		int64(remainingBytes)+1,
	))
	if err != nil {
		if ctx.Err() != nil {
			return len(body), normalizeSearchContextError(ctx.Err())
		}
		return len(body), newSearchSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	if len(body) > remainingBytes {
		return len(body), newSearchSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	return len(body), nil
}

func decodeCitingCandidateResponse(
	parent context.Context,
	processing context.Context,
	fetched fetchedSearchResponse,
	remainingBytes int,
) ([]byte, int, error) {
	if remainingBytes < 0 {
		return nil, 0, newSearchSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	if err := searchProcessingError(parent, processing); err != nil {
		return nil, 0, err
	}
	var reader io.Reader = bytes.NewReader(fetched.encodedBody)
	var gzipReader *gzip.Reader
	switch strings.ToLower(strings.TrimSpace(fetched.contentEncoding)) {
	case "", "identity":
	case "gzip":
		var err error
		gzipReader, err = gzip.NewReader(&searchContextReader{ctx: processing, reader: reader})
		if err != nil {
			return nil, 0, classifySearchDecodeError(parent, processing)
		}
		defer func() { _ = gzipReader.Close() }()
		reader = gzipReader
	default:
		return nil, 0, newSearchSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	_, parameters, err := mime.ParseMediaType(fetched.contentType)
	if err != nil {
		return nil, 0, newSearchSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	charsetLabel := strings.TrimSpace(parameters["charset"])
	if charsetLabel != "" && !strings.EqualFold(charsetLabel, "utf-8") {
		reader, err = charset.NewReaderLabel(
			charsetLabel,
			&searchContextReader{ctx: processing, reader: reader},
		)
		if err != nil {
			return nil, 0, newSearchSourceError(model.SourceErrorCodeSourceContractChanged, "")
		}
	}
	body, err := io.ReadAll(io.LimitReader(
		&searchContextReader{ctx: processing, reader: reader},
		int64(remainingBytes)+1,
	))
	if budgetErr := searchProcessingError(parent, processing); budgetErr != nil {
		return nil, len(body), budgetErr
	}
	if err != nil {
		return nil, len(body), newSearchSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	if len(body) > remainingBytes {
		return nil, len(body), newSearchSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	return body, len(body), nil
}
