package hanrei

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func TestFetchReadResponseRejectsUnexpectedSuccessfulResponse(t *testing.T) {
	t.Parallel()
	request := mustReadRequest(t, providerID, sourceID, "95878/detail3")
	cases := []struct {
		name   string
		change func(*http.Response)
		code   model.SourceErrorCode
	}{
		{
			"content type",
			func(response *http.Response) { response.Header.Set("Content-Type", "application/json") },
			model.SourceErrorCodeSourceContractChanged,
		},
		{
			"redirected origin",
			func(response *http.Response) {
				target, _ := url.Parse("https://example.com/hanrei/95878/detail3/index.html")
				response.Request = &http.Request{URL: target}
			},
			model.SourceErrorCodeUnsafeSourceContent,
		},
		{
			"redirected detail",
			func(response *http.Response) {
				target, _ := url.Parse(
					"https://www.courts.go.jp/hanrei/95877/detail4/index.html",
				)
				response.Request = &http.Request{URL: target}
			},
			model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			"content length",
			func(response *http.Response) { response.ContentLength = maximumReadResponseBytes + 1 },
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
			_, err := fetchReadResponse(context.Background(), doer, fixedNow, request)
			assertReadSourceError(t, err, testCase.code)
		})
	}
}

func TestFetchReadResponseMapsNotFound(t *testing.T) {
	t.Parallel()
	request := mustReadRequest(t, providerID, sourceID, "95878/detail3")
	doer := clientTestRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := clientTestHTMLResponse(request, []byte("<html></html>"))
		response.StatusCode = http.StatusNotFound
		return response, nil
	})
	_, err := fetchReadResponse(context.Background(), doer, fixedNow, request)
	if !errors.Is(err, judicialdecisionread.ErrNotFound) {
		t.Fatalf("ErrNotFound ではない: %v", err)
	}
}

func TestDecodeReadResponseBodyAcceptsShiftJISHTML(t *testing.T) {
	t.Parallel()
	encoded := encodeShiftJIS(t, []byte("<html><body><p>裁判例結果詳細</p></body></html>"))
	body, err := decodeReadResponseBody(
		context.Background(),
		context.Background(),
		fetchedReadResponse{
			encodedBody: encoded,
			contentType: "text/html; charset=Shift_JIS",
		},
	)
	if err != nil {
		t.Fatalf("Shift_JIS HTML の復号エラー = %v", err)
	}
	if !bytes.Contains(body, []byte("裁判例結果詳細")) {
		t.Fatalf("Shift_JIS HTML を UTF-8 へ変換できていない: %q", body)
	}
}

func TestDecodeReadResponseBodyUsesCumulativeDecodeBudget(t *testing.T) {
	t.Parallel()
	source := bytes.Repeat([]byte("x"), maximumReadDecompressedBytes*3/8)
	encoded, _, err := transform.Bytes(
		unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewEncoder(),
		source,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = decodeReadResponseBody(
		context.Background(),
		context.Background(),
		fetchedReadResponse{
			encodedBody:     gzipBytes(t, encoded),
			contentType:     "text/html; charset=utf-16le",
			contentEncoding: "gzip",
		},
	)
	assertReadSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
}

func TestDecodeReadResponseBodyRejectsEncodingFailures(t *testing.T) {
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
			_, err := decodeReadResponseBody(
				context.Background(),
				context.Background(),
				fetchedReadResponse{
					encodedBody:     testCase.body,
					contentEncoding: testCase.encoding,
				},
			)
			assertReadSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

func TestFetchReadResponseDoesNotRetry(t *testing.T) {
	t.Parallel()
	request := mustReadRequest(t, providerID, sourceID, "95878/detail3")
	var calls atomic.Int32
	doer := clientTestRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		response := clientTestHTMLResponse(request, []byte("secret body"))
		response.StatusCode = http.StatusServiceUnavailable
		response.Header.Set("Retry-After", "3")
		return response, nil
	})
	_, err := fetchReadResponse(context.Background(), doer, fixedNow, request)
	assertReadSourceError(t, err, model.SourceErrorCodeSourceUnavailable)
	if calls.Load() != 1 {
		t.Fatalf("自動再試行回数を含む呼出し回数 = %d", calls.Load())
	}
	if strings.Contains(err.Error(), "secret") {
		t.Errorf("SOT-IF-017: エラーが本文を含む: %v", err)
	}
}

func TestFetchReadResponseNormalizesUnsafeRedirect(t *testing.T) {
	t.Parallel()
	request := mustReadRequest(t, providerID, sourceID, "95878/detail3")
	doer := clientTestRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errUnsafeCourtsRedirect
	})
	_, err := fetchReadResponse(context.Background(), doer, fixedNow, request)
	assertReadSourceError(t, err, model.SourceErrorCodeUnsafeSourceContent)
}

func TestNewJudicialDecisionReadAdapterValidatesDependencies(t *testing.T) {
	t.Parallel()
	valid := readAdapterDependencies{
		doer:         clientTestRoundTripFunc(nil),
		now:          fixedNow,
		gate:         make(chan struct{}, 1),
		parseTimeout: readParseTimeout,
	}
	cases := []struct {
		name   string
		change func(*readAdapterDependencies)
	}{
		{"doer", func(values *readAdapterDependencies) { values.doer = nil }},
		{"now", func(values *readAdapterDependencies) { values.now = nil }},
		{"gate", func(values *readAdapterDependencies) { values.gate = nil }},
		{"gate capacity", func(values *readAdapterDependencies) { values.gate = make(chan struct{}, 2) }},
		{"negative timeout", func(values *readAdapterDependencies) { values.parseTimeout = -time.Nanosecond }},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			values := valid
			testCase.change(&values)
			if _, err := newJudicialDecisionReadAdapter(values); err == nil {
				t.Fatal("不正な依存関係を受理した")
			}
		})
	}
}

func TestReadContextCancellationDoesNotFetch(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	adapter := newTestReadAdapter(t, clientTestRoundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return nil, errors.New("呼び出されてはなりません")
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := adapter.Read(ctx, mustReadRequest(t, providerID, sourceID, "95878/detail3"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Read() のエラー = %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("cancel 後の外部呼出し回数 = %d", calls.Load())
	}
}
