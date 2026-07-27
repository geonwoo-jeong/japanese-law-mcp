package hanrei

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	maximumSearchResponseBytes     = 2 * 1024 * 1024
	maximumSearchDecompressedBytes = 4 * 1024 * 1024
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type fetchedSearchResponse struct {
	encodedBody     []byte
	contentEncoding string
	fetchedURL      string
	retrievedAt     time.Time
}

func newProductionHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableCompression = true
	return &http.Client{
		Transport:     transport,
		CheckRedirect: courtsRedirectPolicy,
	}
}

func fetchSearchResponse(
	ctx context.Context,
	doer httpDoer,
	now func() time.Time,
	query string,
) (fetchedSearchResponse, error) {
	request, err := buildSearchHTTPRequest(ctx, query)
	if err != nil {
		return fetchedSearchResponse{}, err
	}
	response, err := doer.Do(request)
	if err != nil {
		return fetchedSearchResponse{}, classifySearchDoError(ctx, err)
	}
	if response == nil || response.Body == nil {
		return fetchedSearchResponse{}, newSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	defer func() { _ = response.Body.Close() }()
	return consumeSearchResponse(ctx, response, request, now)
}

func consumeSearchResponse(
	ctx context.Context,
	response *http.Response,
	request *http.Request,
	now func() time.Time,
) (fetchedSearchResponse, error) {
	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		discardSearchBody(response.Body)
		code := codeForSearchStatus(response.StatusCode)
		retryAfter := ""
		if code == model.SourceErrorCodeRateLimited ||
			code == model.SourceErrorCodeSourceUnavailable {
			retryAfter = retryAfterValue(response.Header.Get("Retry-After"), now())
		}
		return fetchedSearchResponse{}, newSearchSourceError(code, retryAfter)
	}
	if !isHTMLMediaType(response.Header.Get("Content-Type")) {
		return fetchedSearchResponse{}, newSearchSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	return readSuccessfulSearchResponse(ctx, response, request, now)
}

func readSuccessfulSearchResponse(
	ctx context.Context,
	response *http.Response,
	request *http.Request,
	now func() time.Time,
) (fetchedSearchResponse, error) {
	finalURL := request.URL
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL
	}
	if !isCourtsHTTPSOrigin(finalURL) {
		return fetchedSearchResponse{}, newSearchSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}
	body, err := readSearchTransferBody(ctx, response)
	if err != nil {
		return fetchedSearchResponse{}, err
	}
	return fetchedSearchResponse{
		encodedBody:     body,
		contentEncoding: response.Header.Get("Content-Encoding"),
		fetchedURL:      finalURL.String(),
		retrievedAt:     now().Round(0),
	}, nil
}

func readSearchTransferBody(
	ctx context.Context,
	response *http.Response,
) ([]byte, error) {
	if response.ContentLength > maximumSearchResponseBytes {
		return nil, newSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	body, err := io.ReadAll(io.LimitReader(
		&searchContextReader{ctx: ctx, reader: response.Body},
		maximumSearchResponseBytes+1,
	))
	if err != nil {
		if ctx.Err() != nil {
			return nil, normalizeSearchContextError(ctx.Err())
		}
		return nil, newSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if len(body) > maximumSearchResponseBytes {
		return nil, newSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	return body, nil
}

func decodeSearchResponseBody(
	parent context.Context,
	processing context.Context,
	fetched fetchedSearchResponse,
) ([]byte, error) {
	if err := searchProcessingError(parent, processing); err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(fetched.contentEncoding)) {
	case "", "identity":
		return copySearchBody(parent, processing, bytes.NewReader(fetched.encodedBody))
	case "gzip":
		reader, err := gzip.NewReader(&searchContextReader{
			ctx: processing, reader: bytes.NewReader(fetched.encodedBody),
		})
		if err != nil {
			return nil, classifySearchDecodeError(parent, processing)
		}
		defer func() { _ = reader.Close() }()
		return copySearchBody(parent, processing, reader)
	default:
		return nil, newSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func copySearchBody(
	parent context.Context,
	processing context.Context,
	reader io.Reader,
) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(
		&searchContextReader{ctx: processing, reader: reader},
		maximumSearchDecompressedBytes+1,
	))
	if budgetErr := searchProcessingError(parent, processing); budgetErr != nil {
		return nil, budgetErr
	}
	if err != nil {
		return nil, newSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if len(body) > maximumSearchDecompressedBytes {
		return nil, newSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	return body, nil
}

func classifySearchDoError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return normalizeSearchContextError(ctx.Err())
	}
	if errors.Is(err, errUnsafeCourtsRedirect) {
		return newSearchSourceError(model.SourceErrorCodeUnsafeSourceContent, "")
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return newSearchSourceError(model.SourceErrorCodeSourceTimeout, "")
	}
	return newSearchSourceError(model.SourceErrorCodeSourceUnavailable, "")
}

func classifySearchDecodeError(
	parent context.Context,
	processing context.Context,
) error {
	if err := searchProcessingError(parent, processing); err != nil {
		return err
	}
	return newSearchSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}

func searchProcessingError(
	parent context.Context,
	processing context.Context,
) error {
	if parent.Err() != nil {
		return normalizeSearchContextError(parent.Err())
	}
	if errors.Is(processing.Err(), context.DeadlineExceeded) {
		return newSearchSourceError(model.SourceErrorCodeSourceProcessingLimit, "")
	}
	if errors.Is(processing.Err(), context.Canceled) {
		return context.Canceled
	}
	return nil
}

func isHTMLMediaType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && strings.EqualFold(mediaType, "text/html")
}

func discardSearchBody(reader io.Reader) {
	_, _ = io.Copy(io.Discard, io.LimitReader(reader, maximumSearchResponseBytes+1))
}

type searchContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *searchContextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}
