package kokkai

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestSpeechSearchHTTPClientReturnsRateLimitedBeforeBodyBudget(t *testing.T) {
	t.Parallel()

	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusTooManyRequests,
				Header:        make(http.Header),
				Body:          io.NopCloser(strings.NewReader("busy")),
				ContentLength: speechSearchResponseBytes + 1,
				Request:       request,
			}, nil
		}),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.fetchSpeechSearch(
		context.Background(),
		mustSpeechSearchRequest(t, parliamentspeechsearch.RequestValues{Query: "永住許可"}),
	)
	assertSpeechSearchSourceError(t, err, model.SourceErrorCodeRateLimited)
}

func TestSpeechSearchHTTPClientMapsEveryServerErrorToUnavailable(t *testing.T) {
	t.Parallel()

	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotImplemented,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    request,
			}, nil
		}),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.fetchSpeechSearch(
		context.Background(),
		mustSpeechSearchRequest(t, parliamentspeechsearch.RequestValues{Query: "民法"}),
	)
	assertSpeechSearchSourceError(t, err, model.SourceErrorCodeSourceUnavailable)
}

func TestSpeechSearchHTTPClientRejectsUnsafeFinalURL(t *testing.T) {
	t.Parallel()

	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(request *http.Request) (*http.Response, error) {
			unsafeURL, parseErr := url.Parse("https://example.com/api/speech?any=%E6%B0%91%E6%B3%95")
			if parseErr != nil {
				t.Fatalf("unsafe URL を作成できません: %v", parseErr)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"numberOfRecords":0,"numberOfReturn":0,"startRecord":1}`)),
				Request:    &http.Request{URL: unsafeURL},
			}, nil
		}),
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.fetchSpeechSearch(
		context.Background(),
		mustSpeechSearchRequest(t, parliamentspeechsearch.RequestValues{Query: "民法"}),
	)
	assertSpeechSearchSourceError(t, err, model.SourceErrorCodeUnsafeSourceContent)
}

func TestSpeechSearchHTTPClientPreservesEncodedBodyForGzip(t *testing.T) {
	t.Parallel()

	var encoded bytes.Buffer
	writer := gzip.NewWriter(&encoded)
	if _, err := writer.Write([]byte(`{"numberOfRecords":0,"numberOfReturn":0,"startRecord":1}`)); err != nil {
		t.Fatalf("gzip body を作成できません: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip writer を閉じられません: %v", err)
	}

	fixedNow := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				Header:        http.Header{"Content-Type": []string{"application/json"}, "Content-Encoding": []string{"gzip"}},
				Body:          io.NopCloser(bytes.NewReader(encoded.Bytes())),
				ContentLength: int64(encoded.Len()),
				Request:       request,
			}, nil
		}),
		func() time.Time { return fixedNow },
	)
	if err != nil {
		t.Fatal(err)
	}

	fetched, err := client.fetchSpeechSearch(
		context.Background(),
		mustSpeechSearchRequest(t, parliamentspeechsearch.RequestValues{Query: "民法"}),
	)
	if err != nil {
		t.Fatalf("fetchSpeechSearch() のエラー = %v", err)
	}
	if fetched.contentEncoding != "gzip" ||
		!bytes.Equal(fetched.encodedBody, encoded.Bytes()) ||
		fetched.fetchedURL == "" ||
		!fetched.retrievedAt.Equal(fixedNow) {
		t.Fatalf("fetched = %#v", fetched)
	}
}

func TestDecodeSpeechSearchResponseBodyInflatesGzip(t *testing.T) {
	t.Parallel()

	var encoded bytes.Buffer
	writer := gzip.NewWriter(&encoded)
	if _, err := writer.Write([]byte(`{"numberOfRecords":0,"numberOfReturn":0,"startRecord":1}`)); err != nil {
		t.Fatalf("gzip body を作成できません: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip writer を閉じられません: %v", err)
	}

	body, err := decodeSpeechSearchResponseBody(
		context.Background(),
		context.Background(),
		fetchedSpeechSearchResponse{
			encodedBody:     encoded.Bytes(),
			contentEncoding: "gzip",
		},
	)
	if err != nil {
		t.Fatalf("decodeSpeechSearchResponseBody() のエラー = %v", err)
	}
	if string(body) != `{"numberOfRecords":0,"numberOfReturn":0,"startRecord":1}` {
		t.Fatalf("decoded body = %s", body)
	}
}
