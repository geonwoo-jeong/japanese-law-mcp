package lawv2

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (function doerFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestBuildHTTPRequestUsesFixedEGovContract(t *testing.T) {
	t.Parallel()

	request, err := buildHTTPRequest(context.Background(), lawSearchRequest{
		query:  "行政",
		asOf:   mustDate("2026-07-26"),
		limit:  20,
		offset: 40,
	})
	if err != nil {
		t.Fatalf("buildHTTPRequest() error = %v", err)
	}

	if request.Method != http.MethodGet {
		t.Fatalf("Method = %q", request.Method)
	}
	if request.URL.Scheme != "https" ||
		request.URL.Host != "laws.e-gov.go.jp" ||
		request.URL.Path != "/api/2/laws" {
		t.Fatalf("URL = %q", request.URL.String())
	}
	query := request.URL.Query()
	expected := map[string]string{
		"law_title":       "行政",
		"asof":            "2026-07-26",
		"limit":           "20",
		"offset":          "40",
		"response_format": "json",
		"order":           "+law_info.law_id",
	}
	for key, value := range expected {
		if query.Get(key) != value {
			t.Errorf("%s = %q, want %q", key, query.Get(key), value)
		}
	}
	if request.Header.Get("User-Agent") != "japanese-law-mcp/dev" {
		t.Errorf("User-Agent = %q", request.Header.Get("User-Agent"))
	}
}

func TestLawSearchRequestRejectsInvalidProviderValues(t *testing.T) {
	t.Parallel()

	validDate := mustDate("2026-07-26")
	cases := []lawSearchRequest{
		{asOf: validDate, limit: 20},
		{query: "民法", limit: 20},
		{query: "民法", asOf: validDate, limit: 0},
		{query: "民法", asOf: validDate, limit: 101},
		{query: "民法", asOf: validDate, limit: 20, offset: -1},
	}
	for _, request := range cases {
		if err := request.validate(); err == nil {
			t.Errorf("validate(%+v) error = nil", request)
		}
	}
}

func TestLawClientReturnsSuccessfulJSON(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.String() == "" {
				t.Fatal("request URL is empty")
			}
			return response(
				http.StatusOK,
				`{"total_count":0,"count":0,"laws":[]}`,
				map[string]string{"Content-Type": "application/json; charset=utf-8"},
			), nil
		}),
		now: func() time.Time { return now },
		sleep: func(context.Context, time.Duration) error {
			t.Fatal("success must not sleep")
			return nil
		},
	})

	result, err := client.fetch(context.Background(), lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	})
	if err != nil {
		t.Fatalf("fetch() error = %v", err)
	}
	if string(result.body) != `{"total_count":0,"count":0,"laws":[]}` {
		t.Fatalf("body = %q", result.body)
	}
	if !result.retrievedAt.Equal(now) {
		t.Fatalf("retrievedAt = %v", result.retrievedAt)
	}
}

func TestProductionClientDoesNotFollowRedirects(t *testing.T) {
	t.Parallel()

	production := newProductionClient()
	httpClient, ok := production.dependencies.doer.(*http.Client)
	if !ok {
		t.Fatalf("production HTTP client の型 = %T", production.dependencies.doer)
	}
	if httpClient.CheckRedirect == nil {
		t.Fatal("e-Gov client の redirect 拒否 policy がありません")
	}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://redirect.example.test/resource",
		nil,
	)
	if err != nil {
		t.Fatalf("redirect request を作成できません: %v", err)
	}
	if err := httpClient.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect policy error = %v", err)
	}
}

func TestLawClientRetriesCandidateStatusAtMostThreeTimes(t *testing.T) {
	t.Parallel()

	var mutex sync.Mutex
	attempts := 0
	var delays []time.Duration
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			mutex.Lock()
			defer mutex.Unlock()
			attempts++
			return response(http.StatusServiceUnavailable, "秘密の外部本文", nil), nil
		}),
		now: time.Now,
		sleep: func(_ context.Context, delay time.Duration) error {
			mutex.Lock()
			defer mutex.Unlock()
			delays = append(delays, delay)
			return nil
		},
	})

	_, err := client.fetch(context.Background(), lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	})
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceUnavailable)
	if strings.Contains(err.Error(), "秘密") {
		t.Fatal("外部本文がエラーへ露出しました")
	}
	mutex.Lock()
	defer mutex.Unlock()
	if attempts != 4 {
		t.Fatalf("attempts = %d, want 4", attempts)
	}
	expectedDelays := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	if len(delays) != len(expectedDelays) {
		t.Fatalf("delays = %v", delays)
	}
	for index := range expectedDelays {
		if delays[index] != expectedDelays[index] {
			t.Errorf("delays[%d] = %v, want %v", index, delays[index], expectedDelays[index])
		}
	}
}

func TestLawClientUsesValidRetryAfter(t *testing.T) {
	t.Parallel()

	attempts := 0
	var delay time.Duration
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return response(
					http.StatusTooManyRequests,
					"rate limited",
					map[string]string{"Retry-After": "7"},
				), nil
			}
			return response(
				http.StatusOK,
				`{"total_count":0,"count":0,"laws":[]}`,
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
		now: time.Now,
		sleep: func(_ context.Context, value time.Duration) error {
			delay = value
			return nil
		},
	})

	_, err := client.fetch(context.Background(), lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	})
	if err != nil {
		t.Fatalf("fetch() error = %v", err)
	}
	if delay != 7*time.Second {
		t.Fatalf("delay = %v", delay)
	}
}

func TestLawClientRaisesRetryAfterToMinimumInterval(t *testing.T) {
	t.Parallel()

	attempts := 0
	var delay time.Duration
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return response(
					http.StatusTooManyRequests,
					"rate limited",
					map[string]string{"Retry-After": "0"},
				), nil
			}
			return response(
				http.StatusOK,
				`{"total_count":0,"count":0,"laws":[]}`,
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
		now: time.Now,
		sleep: func(_ context.Context, value time.Duration) error {
			delay = value
			return nil
		},
	})

	_, err := client.fetch(context.Background(), lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	})
	if err != nil {
		t.Fatalf("fetch() error = %v", err)
	}
	if delay != time.Second {
		t.Fatalf("Retry-After の実効 delay = %v、期待値 = %v", delay, time.Second)
	}
}

func TestLawClientDoesNotWaitPastDeadlineAfterMinimumInterval(t *testing.T) {
	t.Parallel()

	now := time.Now().Round(0)
	ctx, cancel := context.WithDeadline(
		context.Background(),
		now.Add(500*time.Millisecond),
	)
	defer cancel()
	attempts := 0
	sleeps := 0
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return response(
				http.StatusTooManyRequests,
				"rate limited",
				map[string]string{"Retry-After": "0"},
			), nil
		}),
		now: func() time.Time { return now },
		sleep: func(context.Context, time.Duration) error {
			sleeps++
			return nil
		},
	})

	_, err := client.fetch(ctx, lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	})
	assertSourceErrorCode(t, err, model.SourceErrorCodeRateLimited)
	if attempts != 1 || sleeps != 0 {
		t.Fatalf("attempts = %d、sleeps = %d", attempts, sleeps)
	}
}

func TestLawClientDoesNotRetryOther4xx(t *testing.T) {
	t.Parallel()

	attempts := 0
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return response(http.StatusBadRequest, "利用者へ出さない本文", nil), nil
		}),
		now: time.Now,
		sleep: func(context.Context, time.Duration) error {
			t.Fatal("400 must not sleep")
			return nil
		},
	})

	_, err := client.fetch(context.Background(), lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	})
	assertSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
	if attempts != 1 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestLawClientKeepsStatusRetryForOversizedErrorBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status int
		code   model.SourceErrorCode
	}{
		{
			name:   "429",
			status: http.StatusTooManyRequests,
			code:   model.SourceErrorCodeRateLimited,
		},
		{
			name:   "503",
			status: http.StatusServiceUnavailable,
			code:   model.SourceErrorCodeSourceUnavailable,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			attempts := 0
			client := mustTestClient(t, clientDependencies{
				doer: doerFunc(func(*http.Request) (*http.Response, error) {
					attempts++
					return &http.Response{
						StatusCode: test.status,
						Header:     make(http.Header),
						Body: io.NopCloser(io.LimitReader(
							zeroReader{},
							maximumResponseBytes+1,
						)),
					}, nil
				}),
				now: time.Now,
				sleep: func(context.Context, time.Duration) error {
					return nil
				},
			})

			_, err := client.fetch(context.Background(), lawSearchRequest{
				query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
			})
			assertSourceErrorCode(t, err, test.code)
			if attempts != 4 {
				t.Fatalf("attempts = %d, want 4", attempts)
			}
		})
	}
}

func TestLawClientHonorsCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(request *http.Request) (*http.Response, error) {
			return nil, request.Context().Err()
		}),
		now:   time.Now,
		sleep: sleepWithContext,
	})

	_, err := client.fetch(ctx, lawSearchRequest{
		query: "民法", asOf: mustDate("2026-07-26"), limit: 20,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestRetryAfterAndContextSleepEdges(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	httpDate := now.Add(time.Minute).Format(http.TimeFormat)
	raw, delay, valid := parseRetryAfter(httpDate, now)
	if !valid || raw != httpDate || delay != time.Minute {
		t.Fatalf("parseRetryAfter() = %q, %v, %t", raw, delay, valid)
	}
	if _, _, valid := parseRetryAfter("invalid", now); valid {
		t.Fatal("invalid Retry-After was accepted")
	}
	subsecondNow := now.Add(500 * time.Millisecond)
	subsecondDate := subsecondNow.Add(500 * time.Millisecond).Format(http.TimeFormat)
	raw, delay, valid = parseRetryAfter(subsecondDate, subsecondNow)
	if !valid || raw != subsecondDate || delay != time.Second {
		t.Fatalf("一秒未満の HTTP-date = %q, %v, %t", raw, delay, valid)
	}
	if err := sleepWithContext(context.Background(), 0); err != nil {
		t.Fatalf("sleepWithContext(0) error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepWithContext(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("sleepWithContext(canceled) error = %v", err)
	}
}

func TestReadResponseBodyEnforcesCompressedAndDecompressedLimits(t *testing.T) {
	t.Parallel()

	tooLarge := bytes.Repeat([]byte("x"), maximumResponseBytes+1)
	_, err := readResponseBody(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(tooLarge)),
	})
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceResponseTooLarge)

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, writeErr := io.Copy(
		writer,
		io.LimitReader(zeroReader{}, maximumDecompressedBytes+1),
	); writeErr != nil {
		t.Fatalf("gzip write: %v", writeErr)
	}
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatalf("gzip close: %v", closeErr)
	}
	_, err = readResponseBody(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(compressed.Bytes())),
		Header: http.Header{
			"Content-Encoding": []string{"gzip"},
		},
	})
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceResponseTooLarge)
}

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = 0
	}
	return len(buffer), nil
}

func mustTestClient(t *testing.T, dependencies clientDependencies) lawClient {
	t.Helper()
	client, err := newLawClient(dependencies)
	if err != nil {
		t.Fatalf("newLawClient() error = %v", err)
	}
	return client
}

func response(
	status int,
	body string,
	headers map[string]string,
) *http.Response {
	header := make(http.Header, len(headers))
	for key, value := range headers {
		header.Set(key, value)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func assertSourceErrorCode(
	t *testing.T,
	err error,
	code model.SourceErrorCode,
) {
	t.Helper()
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("error = %T %v, want model.SourceError", err, err)
	}
	if sourceError.Code() != code {
		t.Fatalf("code = %q, want %q", sourceError.Code(), code)
	}
}
