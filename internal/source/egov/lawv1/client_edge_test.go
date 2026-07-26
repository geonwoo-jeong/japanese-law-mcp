package lawv1

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestClientの依存関係と異常応答を拒否する(t *testing.T) {
	t.Parallel()

	if _, err := newUpdateListClient(clientDependencies{}); err == nil {
		t.Fatal("依存関係のない client を受理しました")
	}
	validRequest := updateListRequest{date: mustTestDate(t, "2023-02-01")}
	tests := []struct {
		name string
		doer doerFunc
		code model.SourceErrorCode
	}{
		{
			name: "nil response",
			doer: func(*http.Request) (*http.Response, error) {
				return nil, nil
			},
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "network timeout",
			doer: func(*http.Request) (*http.Response, error) {
				return nil, timeoutError{}
			},
			code: model.SourceErrorCodeSourceTimeout,
		},
		{
			name: "network unavailable",
			doer: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("接続失敗")
			},
			code: model.SourceErrorCodeSourceUnavailable,
		},
		{
			name: "unknown content encoding",
			doer: func(*http.Request) (*http.Response, error) {
				return testResponse(
					http.StatusOK,
					"<DataRoot/>",
					map[string]string{
						"Content-Type":     "text/xml",
						"Content-Encoding": "br",
					},
				), nil
			},
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "invalid gzip",
			doer: func(*http.Request) (*http.Response, error) {
				return testResponse(
					http.StatusOK,
					"gzipではない",
					map[string]string{
						"Content-Type":     "text/xml",
						"Content-Encoding": "gzip",
					},
				), nil
			},
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
		{
			name: "other 4xx",
			doer: func(*http.Request) (*http.Response, error) {
				return testResponse(http.StatusBadRequest, "外部本文", nil), nil
			},
			code: model.SourceErrorCodeInvalidSourceResponse,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := mustTestClient(t, clientDependencies{
				doer:  test.doer,
				now:   time.Now,
				sleep: noSleep,
			})
			_, err := client.fetch(context.Background(), validRequest)
			assertSourceErrorCode(t, err, test.code)
		})
	}
}

func TestClientの時間helperは取消とRetryAfterを扱う(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	if raw, delay, valid := parseRetryAfter("7", now); !valid ||
		raw != "7" ||
		delay != 7*time.Second {
		t.Fatalf("seconds Retry-After = %q, %v, %t", raw, delay, valid)
	}
	httpDate := now.Add(time.Minute).Format(http.TimeFormat)
	if _, delay, valid := parseRetryAfter(httpDate, now); !valid ||
		delay != time.Minute {
		t.Fatalf("date Retry-After = %v, %t", delay, valid)
	}
	if _, _, valid := parseRetryAfter("不正", now); valid {
		t.Fatal("不正な Retry-After を受理しました")
	}
	deadlineContext, cancelDeadline := context.WithDeadline(
		context.Background(),
		now.Add(time.Second),
	)
	defer cancelDeadline()
	if canWait(deadlineContext, now, 2*time.Second) {
		t.Fatal("期限を超える待機を受理しました")
	}
	if err := sleepWithContext(context.Background(), 0); err != nil {
		t.Fatalf("sleepWithContext(0) error = %v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepWithContext(canceled, time.Hour); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("sleep error = %v", err)
	}
	assertSourceErrorCode(
		t,
		normalizeContextError(context.DeadlineExceeded),
		model.SourceErrorCodeSourceTimeout,
	)
}

func TestReadAtMostはreaderエラーを正規化する(t *testing.T) {
	t.Parallel()

	_, err := readAtMost(errorReader{}, 10)
	assertSourceErrorCode(t, err, model.SourceErrorCodeInvalidSourceResponse)
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
