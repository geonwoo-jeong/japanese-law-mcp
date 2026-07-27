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

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"golang.org/x/net/html/charset"
)

const (
	maximumReadResponseBytes     = 1 * 1024 * 1024
	maximumReadDecompressedBytes = 2 * 1024 * 1024
)

type fetchedReadResponse struct {
	encodedBody     []byte
	contentType     string
	contentEncoding string
	fetchedURL      string
	retrievedAt     time.Time
	decisionID      string
	categoryNumber  string
}

func fetchReadResponse(
	ctx context.Context,
	doer httpDoer,
	now func() time.Time,
	request judicialdecisionread.Request,
) (fetchedReadResponse, error) {
	httpRequest, detailURL, decisionID, categoryNumber, err := buildReadHTTPRequest(ctx, request)
	if err != nil {
		return fetchedReadResponse{}, err
	}
	response, err := doer.Do(httpRequest)
	if err != nil {
		return fetchedReadResponse{}, classifyReadDoError(ctx, err)
	}
	if response == nil || response.Body == nil {
		return fetchedReadResponse{}, newReadSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	defer func() { _ = response.Body.Close() }()
	result, err := consumeReadResponse(ctx, response, httpRequest, now)
	if err != nil {
		return fetchedReadResponse{}, err
	}
	result.decisionID = decisionID
	result.categoryNumber = categoryNumber
	if result.fetchedURL == "" {
		result.fetchedURL = detailURL
	}
	return result, nil
}

func consumeReadResponse(
	ctx context.Context,
	response *http.Response,
	request *http.Request,
	now func() time.Time,
) (fetchedReadResponse, error) {
	finalURL := request.URL
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL
	}
	if !isCourtsHTTPSOrigin(finalURL) {
		return fetchedReadResponse{}, newReadSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}
	if finalURL.String() != request.URL.String() {
		return fetchedReadResponse{}, newReadSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	switch {
	case response.StatusCode == http.StatusNotFound:
		discardReadBody(response.Body)
		return fetchedReadResponse{}, judicialdecisionread.ErrNotFound
	case response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices:
		discardReadBody(response.Body)
		code := codeForReadStatus(response.StatusCode)
		retryAfter := ""
		if code == model.SourceErrorCodeRateLimited ||
			code == model.SourceErrorCodeSourceUnavailable {
			retryAfter = retryAfterValue(response.Header.Get("Retry-After"), now())
		}
		return fetchedReadResponse{}, newReadSourceError(code, retryAfter)
	}
	if !isHTMLMediaType(response.Header.Get("Content-Type")) {
		return fetchedReadResponse{}, newReadSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	body, err := readReadTransferBody(ctx, response)
	if err != nil {
		return fetchedReadResponse{}, err
	}
	return fetchedReadResponse{
		encodedBody:     body,
		contentType:     response.Header.Get("Content-Type"),
		contentEncoding: response.Header.Get("Content-Encoding"),
		fetchedURL:      finalURL.String(),
		retrievedAt:     now().Round(0),
	}, nil
}

func readReadTransferBody(ctx context.Context, response *http.Response) ([]byte, error) {
	if response.ContentLength > maximumReadResponseBytes {
		return nil, newReadSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	body, err := io.ReadAll(io.LimitReader(
		&readContextReader{ctx: ctx, reader: response.Body},
		maximumReadResponseBytes+1,
	))
	if err != nil {
		if ctx.Err() != nil {
			return nil, normalizeReadContextError(ctx.Err())
		}
		return nil, newReadSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	if len(body) > maximumReadResponseBytes {
		return nil, newReadSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	return body, nil
}

func decodeReadResponseBody(
	parent context.Context,
	processing context.Context,
	fetched fetchedReadResponse,
) ([]byte, error) {
	if err := readProcessingError(parent, processing); err != nil {
		return nil, err
	}
	var decoded []byte
	var err error
	switch strings.ToLower(strings.TrimSpace(fetched.contentEncoding)) {
	case "", "identity":
		decoded, err = copyReadBody(parent, processing, bytes.NewReader(fetched.encodedBody))
	case "gzip":
		var reader *gzip.Reader
		reader, err = gzip.NewReader(&readContextReader{
			ctx: processing, reader: bytes.NewReader(fetched.encodedBody),
		})
		if err != nil {
			return nil, classifyReadDecodeError(parent, processing)
		}
		defer func() { _ = reader.Close() }()
		decoded, err = copyReadBody(parent, processing, reader)
	default:
		return nil, newReadSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	if err != nil {
		return nil, err
	}
	return decodeReadCharset(parent, processing, fetched.contentType, decoded)
}

func decodeReadCharset(
	parent context.Context,
	processing context.Context,
	contentType string,
	body []byte,
) ([]byte, error) {
	decodedReader, converts, err := newReadCharsetReader(processing, contentType, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if !converts {
		return body, nil
	}
	remaining := maximumReadDecompressedBytes - len(body)
	return copyReadBodyWithLimit(parent, processing, decodedReader, remaining)
}

func newReadCharsetReader(
	processing context.Context,
	contentType string,
	reader io.Reader,
) (io.Reader, bool, error) {
	_, parameters, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, false, newReadSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	charsetLabel := strings.TrimSpace(parameters["charset"])
	if charsetLabel == "" || strings.EqualFold(charsetLabel, "utf-8") {
		return &readContextReader{ctx: processing, reader: reader}, false, nil
	}
	decodedReader, err := charset.NewReaderLabel(
		charsetLabel,
		&readContextReader{ctx: processing, reader: reader},
	)
	if err != nil {
		return nil, false, newReadSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	return decodedReader, true, nil
}

func copyReadBody(parent context.Context, processing context.Context, reader io.Reader) ([]byte, error) {
	return copyReadBodyWithLimit(parent, processing, reader, maximumReadDecompressedBytes)
}

func copyReadBodyWithLimit(
	parent context.Context,
	processing context.Context,
	reader io.Reader,
	limit int,
) ([]byte, error) {
	if limit < 0 {
		return nil, newReadSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	body, err := io.ReadAll(io.LimitReader(
		&readContextReader{ctx: processing, reader: reader},
		int64(limit)+1,
	))
	if budgetErr := readProcessingError(parent, processing); budgetErr != nil {
		return nil, budgetErr
	}
	if err != nil {
		return nil, newReadSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
	}
	if len(body) > limit {
		return nil, newReadSourceError(model.SourceErrorCodeSourceResponseTooLarge, "")
	}
	return body, nil
}

func classifyReadDoError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return normalizeReadContextError(ctx.Err())
	}
	if errors.Is(err, errUnsafeCourtsRedirect) {
		return newReadSourceError(model.SourceErrorCodeUnsafeSourceContent, "")
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return newReadSourceError(model.SourceErrorCodeSourceTimeout, "")
	}
	return newReadSourceError(model.SourceErrorCodeSourceUnavailable, "")
}

func classifyReadDecodeError(parent context.Context, processing context.Context) error {
	if err := readProcessingError(parent, processing); err != nil {
		return err
	}
	return newReadSourceError(model.SourceErrorCodeInvalidSourceResponse, "")
}

func readProcessingError(parent context.Context, processing context.Context) error {
	if parent.Err() != nil {
		return normalizeReadContextError(parent.Err())
	}
	if errors.Is(processing.Err(), context.DeadlineExceeded) {
		return newReadSourceError(model.SourceErrorCodeSourceProcessingLimit, "")
	}
	if errors.Is(processing.Err(), context.Canceled) {
		return context.Canceled
	}
	return nil
}

func discardReadBody(reader io.Reader) {
	_, _ = io.Copy(io.Discard, io.LimitReader(reader, maximumReadResponseBytes+1))
}

func codeForReadStatus(status int) model.SourceErrorCode {
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

type readContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *readContextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}
