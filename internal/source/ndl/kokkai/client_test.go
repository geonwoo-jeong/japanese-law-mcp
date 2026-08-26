package kokkai

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
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
			response := &http.Response{
				StatusCode:    http.StatusTooManyRequests,
				Header:        make(http.Header),
				Body:          io.NopCloser(strings.NewReader("busy")),
				ContentLength: speechSearchResponseBytes + 1,
				Request:       request,
			}
			response.Header.Set("Retry-After", "7")
			return response, nil
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
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatal("rate limit error を取得できません")
	}
	retryAfter, exists := sourceError.RetryAfter()
	if !exists || retryAfter != "7" {
		t.Fatal("明示された Retry-After を保持していません")
	}
}

func TestSpeechSearchHTTPClientMapsEveryServerErrorToUnavailable(t *testing.T) {
	t.Parallel()

	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(request *http.Request) (*http.Response, error) {
			response := &http.Response{
				StatusCode: http.StatusNotImplemented,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader("")),
				Request:    request,
			}
			response.Header.Set("Retry-After", "9")
			return response, nil
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
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatal("source unavailable error を取得できません")
	}
	retryAfter, exists := sourceError.RetryAfter()
	if !exists || retryAfter != "9" {
		t.Fatal("5xx が明示した Retry-After を保持していません")
	}
}

func TestSpeechSearchHTTPClientRejectsNonStandardStatusAbove599(t *testing.T) {
	t.Parallel()

	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 600,
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
	assertSpeechSearchSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
}

func TestSpeechSearchRetryAfterValueAcceptsOnlyExplicitValidValues(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	tests := map[string]struct {
		value string
		want  string
	}{
		"秒数":     {value: "7", want: "7"},
		"大きい秒数":  {value: "4294967296", want: "4294967296"},
		"未来の日時":  {value: now.Add(time.Minute).Format(http.TimeFormat), want: now.Add(time.Minute).Format(http.TimeFormat)},
		"過去の日時":  {value: now.Add(-time.Minute).Format(http.TimeFormat), want: now.Add(-time.Minute).Format(http.TimeFormat)},
		"不正な値":   {value: "later"},
		"空白だけの値": {value: "  "},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := speechSearchRetryAfterValue(test.value); got != test.want {
				t.Fatal("Retry-After の検証結果が一致しません")
			}
		})
	}
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
		t.Fatal("取得済み response の metadata が一致しません")
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
		t.Fatal("gzip 復号後の response が一致しません")
	}
}

func TestNewProductionSpeechSearchClientUsesFixedNetworkPolicy(t *testing.T) {
	t.Parallel()

	client := newProductionSpeechSearchClient()
	httpClient, ok := client.doer.(*http.Client)
	if !ok || httpClient.CheckRedirect == nil {
		t.Fatal("production client の redirect policy がありません")
	}
	transport, ok := httpClient.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil || !transport.DisableCompression {
		t.Fatal("production client の proxy または圧縮 policy が一致しません")
	}
	if err := httpClient.CheckRedirect(&http.Request{}, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatal("redirect を拒否しません")
	}
}

func TestSpeechSearchHTTPClientNormalizesFailures(t *testing.T) {
	t.Parallel()

	request := mustSpeechSearchRequest(
		t,
		parliamentspeechsearch.RequestValues{Query: "民法"},
	)
	tests := []struct {
		name string
		doer doerFunc
		want model.SourceErrorCode
	}{
		{
			name: "外部 timeout",
			doer: func(*http.Request) (*http.Response, error) {
				return nil, context.DeadlineExceeded
			},
			want: model.SourceErrorCodeSourceTimeout,
		},
		{
			name: "接続失敗",
			doer: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("接続失敗")
			},
			want: model.SourceErrorCodeSourceUnavailable,
		},
		{
			name: "redirect error",
			doer: func(*http.Request) (*http.Response, error) {
				return nil, http.ErrUseLastResponse
			},
			want: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "nil response",
			doer: func(*http.Request) (*http.Response, error) {
				return nil, nil
			},
			want: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "nil body",
			doer: func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK}, nil
			},
			want: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "redirect response",
			doer: func(outbound *http.Request) (*http.Response, error) {
				return speechSearchTestResponse(outbound, http.StatusFound, "application/json", ""), nil
			},
			want: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "JSON 以外",
			doer: func(outbound *http.Request) (*http.Response, error) {
				return speechSearchTestResponse(outbound, http.StatusOK, "text/plain", "{}"), nil
			},
			want: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "Content-Length 超過",
			doer: func(outbound *http.Request) (*http.Response, error) {
				response := speechSearchTestResponse(
					outbound,
					http.StatusOK,
					"application/json",
					"",
				)
				response.ContentLength = speechSearchResponseBytes + 1
				return response, nil
			},
			want: model.SourceErrorCodeSourceResponseTooLarge,
		},
		{
			name: "body 読取り失敗",
			doer: func(outbound *http.Request) (*http.Response, error) {
				response := speechSearchTestResponse(
					outbound,
					http.StatusOK,
					"application/json",
					"",
				)
				response.Body = io.NopCloser(speechSearchErrorReader{})
				response.ContentLength = -1
				return response, nil
			},
			want: model.SourceErrorCodeInvalidSourceResponse,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client, err := newSpeechSearchHTTPClient(test.doer, time.Now)
			if err != nil {
				t.Fatalf("検証用 client を作成できません: %v", err)
			}
			_, err = client.fetchSpeechSearch(context.Background(), request)
			assertSpeechSearchSourceError(t, err, test.want)
		})
	}
}

func TestSpeechSearchHTTPClientRejectsNilContextAndDependencies(t *testing.T) {
	t.Parallel()

	if _, err := newSpeechSearchHTTPClient(nil, time.Now); err == nil {
		t.Fatal("nil doer を受理しました")
	}
	client, err := newSpeechSearchHTTPClient(
		doerFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("nil context で外部呼出ししました")
			return nil, nil
		}),
		time.Now,
	)
	if err != nil {
		t.Fatalf("検証用 client を作成できません: %v", err)
	}
	if _, err := client.fetchSpeechSearch(
		nilSpeechSearchContext(),
		mustSpeechSearchRequest(t, parliamentspeechsearch.RequestValues{Query: "民法"}),
	); err == nil {
		t.Fatal("nil context を受理しました")
	}
}

func TestReadSpeechSearchAtMostUsesMeasuredByteLimit(t *testing.T) {
	t.Parallel()

	accepted, err := readSpeechSearchAtMost(
		context.Background(),
		context.Background(),
		strings.NewReader("1234"),
		4,
	)
	if err != nil || string(accepted) != "1234" {
		t.Fatalf("上限と同じ byte 数を受理できません: %v", err)
	}
	_, err = readSpeechSearchAtMost(
		context.Background(),
		context.Background(),
		strings.NewReader("12345"),
		4,
	)
	assertSpeechSearchSourceError(t, err, model.SourceErrorCodeSourceResponseTooLarge)
}

func TestDecodeSpeechSearchResponseBodyRejectsUnsupportedContentEncoding(t *testing.T) {
	t.Parallel()

	for name, fetched := range map[string]fetchedSpeechSearchResponse{
		"未知 encoding": {encodedBody: []byte("{}"), contentEncoding: "br"},
		"壊れた gzip":    {encodedBody: []byte("not-gzip"), contentEncoding: "gzip"},
	} {
		name, fetched := name, fetched
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeSpeechSearchResponseBody(
				context.Background(),
				context.Background(),
				fetched,
			)
			assertSpeechSearchSourceError(t, err, model.SourceErrorCodeInvalidSourceResponse)
		})
	}
}

type speechSearchErrorReader struct{}

func (speechSearchErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("読取り失敗")
}

func speechSearchTestResponse(
	request *http.Request,
	status int,
	mediaType string,
	body string,
) *http.Response {
	response := &http.Response{
		StatusCode:    status,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       request,
	}
	if mediaType != "" {
		response.Header.Set("Content-Type", mediaType)
	}
	return response
}
