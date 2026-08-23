package kokkai

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
	"net/url"
	"strings"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	speechSearchResponseBytes     = 8 * 1024 * 1024
	speechSearchDecompressedBytes = 16 * 1024 * 1024
	speechSearchHTTPTimeout       = 20 * time.Second
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type speechSearchHTTPClient struct {
	doer httpDoer
	now  func() time.Time
}

type fetchedSpeechSearchResponse struct {
	encodedBody     []byte
	contentEncoding string
	fetchedURL      string
	retrievedAt     time.Time
}

func newProductionSpeechSearchClient() speechSearchHTTPClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DisableCompression = true
	return speechSearchHTTPClient{
		doer: &http.Client{
			Transport:     transport,
			CheckRedirect: rejectSpeechSearchRedirect,
		},
		now: time.Now,
	}
}

func rejectSpeechSearchRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

func newSpeechSearchHTTPClient(
	doer httpDoer,
	now func() time.Time,
) (speechSearchHTTPClient, error) {
	if doer == nil || now == nil {
		return speechSearchHTTPClient{}, fmt.Errorf("国会発言検索 client の依存関係が不足しています")
	}
	return speechSearchHTTPClient{doer: doer, now: now}, nil
}

func (c speechSearchHTTPClient) fetchSpeechSearch(
	ctx context.Context,
	request parliamentspeechsearch.Request,
) (fetchedSpeechSearchResponse, error) {
	if ctx == nil {
		return fetchedSpeechSearchResponse{}, fmt.Errorf("context は必須です")
	}
	requestContext, cancel := context.WithTimeout(ctx, speechSearchHTTPTimeout)
	defer cancel()

	httpRequest, err := buildSpeechSearchHTTPRequest(requestContext, request)
	if err != nil {
		return fetchedSpeechSearchResponse{}, err
	}
	response, err := c.doer.Do(httpRequest)
	if err != nil {
		if ctxErr := requestContext.Err(); ctxErr != nil {
			return fetchedSpeechSearchResponse{}, normalizeSpeechSearchContextError(ctxErr)
		}
		if errors.Is(err, http.ErrUseLastResponse) {
			return fetchedSpeechSearchResponse{}, newSpeechSearchSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			return fetchedSpeechSearchResponse{}, newSpeechSearchSourceError(
				model.SourceErrorCodeSourceTimeout,
				"",
			)
		}
		return fetchedSpeechSearchResponse{}, newSpeechSearchSourceError(
			model.SourceErrorCodeSourceUnavailable,
			"",
		)
	}
	if response == nil || response.Body == nil {
		return fetchedSpeechSearchResponse{}, newSpeechSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	defer closeSpeechSearchResponseBody(response)

	finalURL, err := resolveSpeechSearchFinalURL(response, httpRequest)
	if err != nil {
		return fetchedSpeechSearchResponse{}, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		switch {
		case response.StatusCode == http.StatusTooManyRequests:
			return fetchedSpeechSearchResponse{}, newSpeechSearchSourceError(
				model.SourceErrorCodeRateLimited,
				"",
			)
		case response.StatusCode >= http.StatusInternalServerError:
			return fetchedSpeechSearchResponse{}, newSpeechSearchSourceError(
				model.SourceErrorCodeSourceUnavailable,
				"",
			)
		default:
			return fetchedSpeechSearchResponse{}, newSpeechSearchSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
	}
	if !isSpeechSearchMediaType(response.Header.Get("Content-Type"), "application/json") {
		return fetchedSpeechSearchResponse{}, newSpeechSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	body, err := readSpeechSearchTransferBody(requestContext, response)
	if err != nil {
		if ctxErr := requestContext.Err(); ctxErr != nil {
			return fetchedSpeechSearchResponse{}, normalizeSpeechSearchContextError(ctxErr)
		}
		return fetchedSpeechSearchResponse{}, err
	}
	return fetchedSpeechSearchResponse{
		encodedBody:     body,
		contentEncoding: response.Header.Get("Content-Encoding"),
		fetchedURL:      finalURL.String(),
		retrievedAt:     c.now().Round(0),
	}, nil
}

func resolveSpeechSearchFinalURL(
	response *http.Response,
	request *http.Request,
) (*url.URL, error) {
	finalURL := request.URL
	if response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL
	}
	if !isSpeechSearchSafeOriginURL(finalURL) {
		return nil, newSpeechSearchSourceError(
			model.SourceErrorCodeUnsafeSourceContent,
			"",
		)
	}
	if finalURL.String() != request.URL.String() {
		return nil, newSpeechSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return finalURL, nil
}

func isSpeechSearchSafeOriginURL(value *url.URL) bool {
	return value != nil &&
		value.Scheme == "https" &&
		value.User == nil &&
		value.Opaque == "" &&
		value.Fragment == "" &&
		value.Port() == "" &&
		strings.EqualFold(value.Hostname(), "kokkai.ndl.go.jp")
}

func readSpeechSearchTransferBody(
	ctx context.Context,
	response *http.Response,
) ([]byte, error) {
	if response.ContentLength > speechSearchResponseBytes {
		return nil, newSpeechSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	compressed, err := readSpeechSearchAtMost(
		ctx,
		ctx,
		&speechSearchContextReader{ctx: ctx, reader: response.Body},
		speechSearchResponseBytes,
	)
	if err != nil {
		return nil, err
	}
	return compressed, nil
}

func decodeSpeechSearchResponseBody(
	parent context.Context,
	processing context.Context,
	fetched fetchedSpeechSearchResponse,
) ([]byte, error) {
	if err := speechSearchProcessingError(parent, processing); err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(fetched.contentEncoding)) {
	case "", "identity":
		return copySpeechSearchBody(
			parent,
			processing,
			bytes.NewReader(fetched.encodedBody),
			speechSearchDecompressedBytes,
		)
	case "gzip":
		reader, err := gzip.NewReader(&speechSearchContextReader{
			ctx:    processing,
			reader: bytes.NewReader(fetched.encodedBody),
		})
		if err != nil {
			return nil, newSpeechSearchSourceError(
				model.SourceErrorCodeInvalidSourceResponse,
				"",
			)
		}
		defer func() {
			_ = reader.Close()
		}()
		return copySpeechSearchBody(parent, processing, reader, speechSearchDecompressedBytes)
	default:
		return nil, newSpeechSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func copySpeechSearchBody(
	parent context.Context,
	processing context.Context,
	reader io.Reader,
	limit int64,
) ([]byte, error) {
	if err := speechSearchProcessingError(parent, processing); err != nil {
		return nil, err
	}
	return readSpeechSearchAtMost(
		parent,
		processing,
		&speechSearchContextReader{ctx: processing, reader: reader},
		limit,
	)
}

func readSpeechSearchAtMost(
	parent context.Context,
	processing context.Context,
	reader io.Reader,
	maximum int64,
) ([]byte, error) {
	value, err := io.ReadAll(io.LimitReader(reader, maximum+1))
	if err != nil {
		if processingErr := speechSearchProcessingError(parent, processing); processingErr != nil {
			return nil, processingErr
		}
		return nil, newSpeechSearchSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if err := speechSearchProcessingError(parent, processing); err != nil {
		return nil, err
	}
	if int64(len(value)) > maximum {
		return nil, newSpeechSearchSourceError(
			model.SourceErrorCodeSourceResponseTooLarge,
			"",
		)
	}
	return value, nil
}

func isSpeechSearchMediaType(value string, expected string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == expected
}

func closeSpeechSearchResponseBody(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}
	_ = response.Body.Close()
}

func speechSearchProcessingError(
	parent context.Context,
	processing context.Context,
) error {
	if err := parent.Err(); err != nil {
		return normalizeSpeechSearchContextError(err)
	}
	if err := processing.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return newSpeechSearchSourceError(
				model.SourceErrorCodeSourceProcessingLimit,
				"",
			)
		}
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
	}
	return nil
}
