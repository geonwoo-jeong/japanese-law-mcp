package hanrei

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func TestFetchSearchResponseRejectsUnexpectedSuccessfulResponse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		change func(*http.Response)
		code   model.SourceErrorCode
	}{
		{
			"content type",
			func(response *http.Response) {
				response.Header.Set("Content-Type", "application/json")
			},
			model.SourceErrorCodeSourceContractChanged,
		},
		{
			"redirected origin",
			func(response *http.Response) {
				target, _ := url.Parse("https://example.com/hanrei/search1/index.html")
				response.Request = &http.Request{URL: target}
			},
			model.SourceErrorCodeUnsafeSourceContent,
		},
		{
			"content length",
			func(response *http.Response) {
				response.ContentLength = maximumSearchResponseBytes + 1
			},
			model.SourceErrorCodeSourceResponseTooLarge,
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			doer := clientTestRoundTripFunc(func(request *http.Request) (*http.Response, error) {
				response := clientTestHTMLResponse(request, []byte("<html></html>"))
				testCase.change(response)
				return response, nil
			})
			_, err := fetchSearchResponse(context.Background(), doer, fixedNow, "契約")
			assertSourceError(t, err, testCase.code)
		})
	}
}

func TestDecodeSearchResponseBodyAcceptsShiftJISHTML(t *testing.T) {
	t.Parallel()

	encoded := encodeShiftJIS(t, []byte("<html><body><p>裁判例検索</p></body></html>"))
	body, err := decodeSearchResponseBody(
		context.Background(),
		context.Background(),
		fetchedSearchResponse{
			encodedBody: encoded,
			contentType: "text/html; charset=Shift_JIS",
		},
	)
	if err != nil {
		t.Fatalf("Shift_JIS HTML の復号エラー = %v", err)
	}
	if !bytes.Contains(body, []byte("裁判例検索")) {
		t.Fatalf("Shift_JIS HTML を UTF-8 へ変換できていない: %q", body)
	}
}

func TestDecodeSearchResponseBodyLimitsBytesBeforeCharsetConversion(t *testing.T) {
	t.Parallel()
	source := bytes.Repeat([]byte("x"), maximumSearchDecompressedBytes/2+1)
	encoded, _, err := transform.Bytes(
		unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder(),
		source,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = decodeSearchResponseBody(
		context.Background(),
		context.Background(),
		fetchedSearchResponse{
			encodedBody:     gzipBytes(t, encoded),
			contentType:     "text/html; charset=utf-16le",
			contentEncoding: "gzip",
		},
	)
	assertSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
}

func TestDecodeSearchResponseBodyUsesCumulativeDecodeBudget(t *testing.T) {
	t.Parallel()
	source := bytes.Repeat([]byte("x"), maximumSearchDecompressedBytes*3/8)
	encoded, _, err := transform.Bytes(
		unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder(),
		source,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = decodeSearchResponseBody(
		context.Background(),
		context.Background(),
		fetchedSearchResponse{
			encodedBody:     gzipBytes(t, encoded),
			contentType:     "text/html; charset=utf-16le",
			contentEncoding: "gzip",
		},
	)
	assertSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
}

func TestFetchSearchResponseDoesNotRetry(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	doer := clientTestRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		response := clientTestHTMLResponse(request, []byte("secret body"))
		response.StatusCode = http.StatusServiceUnavailable
		response.Header.Set("Retry-After", "3")
		return response, nil
	})
	_, err := fetchSearchResponse(context.Background(), doer, fixedNow, "secret query")
	assertSourceError(t, err, model.SourceErrorCodeSourceUnavailable)
	if calls.Load() != 1 {
		t.Fatalf("SOT-IF-043: 自動再試行回数を含む呼出し回数 = %d", calls.Load())
	}
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatal(err)
	}
	if retryAfter, exists := sourceError.RetryAfter(); !exists || retryAfter != "3" {
		t.Errorf("retryAfter = %q, %t", retryAfter, exists)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("SOT-IF-017: エラーが入力または本文を含む: %v", err)
	}
}

func TestFetchSearchResponseNormalizesUnsafeRedirect(t *testing.T) {
	t.Parallel()
	doer := clientTestRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errUnsafeCourtsRedirect
	})
	_, err := fetchSearchResponse(context.Background(), doer, fixedNow, "redirect")
	assertSourceError(t, err, model.SourceErrorCodeUnsafeSourceContent)
}

func TestFetchSearchResponseClosesBody(t *testing.T) {
	t.Parallel()
	body := &clientTestTrackedBody{Reader: strings.NewReader("<html></html>")}
	doer := clientTestRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := clientTestHTMLResponse(request, nil)
		response.Body = body
		response.ContentLength = -1
		return response, nil
	})
	if _, err := fetchSearchResponse(context.Background(), doer, fixedNow, "close"); err != nil {
		t.Fatal(err)
	}
	if !body.closed {
		t.Fatal("SOT-ENG-016: response body が閉じられていない")
	}
}

func TestDecodeSearchResponseBodyRejectsEncodingFailures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		encoding string
		body     []byte
	}{
		{"unsupported", "br", []byte("body")},
		{"invalid gzip", "gzip", []byte("not gzip")},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeSearchResponseBody(
				context.Background(),
				context.Background(),
				fetchedSearchResponse{
					encodedBody:     testCase.body,
					contentEncoding: testCase.encoding,
				},
			)
			assertSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestDecodeSearchResponseBodyUsesProcessingBudget(t *testing.T) {
	t.Parallel()
	processing, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := decodeSearchResponseBody(
		context.Background(),
		processing,
		fetchedSearchResponse{encodedBody: []byte("<html></html>")},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ENG-016: canceled processing のエラー = %v", err)
	}
}

func TestNewJudicialDecisionSearchAdapterValidatesDependencies(t *testing.T) {
	t.Parallel()
	valid := searchAdapterDependencies{
		doer:         clientTestRoundTripFunc(nil),
		now:          fixedNow,
		gate:         make(chan struct{}, 1),
		parseTimeout: searchParseTimeout,
	}
	cases := []struct {
		name   string
		change func(*searchAdapterDependencies)
	}{
		{"doer", func(values *searchAdapterDependencies) { values.doer = nil }},
		{"now", func(values *searchAdapterDependencies) { values.now = nil }},
		{"gate", func(values *searchAdapterDependencies) { values.gate = nil }},
		{"gate capacity", func(values *searchAdapterDependencies) {
			values.gate = make(chan struct{}, 2)
		}},
		{"negative timeout", func(values *searchAdapterDependencies) {
			values.parseTimeout = -time.Nanosecond
		}},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			values := valid
			testCase.change(&values)
			if _, err := newJudicialDecisionSearchAdapter(values); err == nil {
				t.Fatal("不正な依存関係を受理した")
			}
		})
	}
}

func TestSearchContextCancellationDoesNotFetch(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	adapter := newTestAdapter(t, clientTestRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("呼び出されてはなりません")
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := adapter.Search(ctx, mustSearchRequest(t, "cancel", 20, ""))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Search() のエラー = %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("cancel 後の外部呼出し回数 = %d", calls.Load())
	}
}

type clientTestRoundTripFunc func(*http.Request) (*http.Response, error)

func (function clientTestRoundTripFunc) Do(
	request *http.Request,
) (*http.Response, error) {
	return function(request)
}

func clientTestHTMLResponse(
	request *http.Request,
	body []byte,
) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Type": []string{"text/html;charset=UTF-8"}},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       request,
	}
}

func encodeShiftJIS(t *testing.T, value []byte) []byte {
	t.Helper()

	reader := transform.NewReader(bytes.NewReader(value), japanese.ShiftJIS.NewEncoder())
	encoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("Shift_JIS への変換に失敗: %v", err)
	}
	return encoded
}

type clientTestTrackedBody struct {
	io.Reader
	closed bool
}

func (body *clientTestTrackedBody) Close() error {
	body.closed = true
	return nil
}
