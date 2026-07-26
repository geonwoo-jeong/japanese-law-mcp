package lawv1

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestClientは429と5xxだけを上限付き再試行する(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var sleeps atomic.Int32
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return testResponse(
				http.StatusServiceUnavailable,
				"秘密の外部本文",
				nil,
			), nil
		}),
		now: time.Now,
		sleep: func(context.Context, time.Duration) error {
			sleeps.Add(1)
			return nil
		},
	})
	_, err := client.fetch(
		context.Background(),
		updateListRequest{date: mustTestDate(t, "2023-02-01")},
	)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceUnavailable)
	if calls.Load() != 4 || sleeps.Load() != 3 {
		t.Fatalf("calls = %d, sleeps = %d", calls.Load(), sleeps.Load())
	}
	if strings.Contains(err.Error(), "秘密") {
		t.Fatal("外部本文がエラーへ露出しました")
	}
}

func TestClientはContentTypeと応答上限を検証する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		doer doerFunc
		code model.SourceErrorCode
	}{
		{
			name: "Content-Type不一致",
			doer: func(*http.Request) (*http.Response, error) {
				return testResponse(
					http.StatusOK,
					"<DataRoot/>",
					map[string]string{"Content-Type": "application/json"},
				), nil
			},
			code: model.SourceErrorCodeSourceContractChanged,
		},
		{
			name: "response bytes超過",
			doer: func(*http.Request) (*http.Response, error) {
				return testResponse(
					http.StatusOK,
					strings.Repeat("x", maximumResponseBytes+1),
					map[string]string{"Content-Type": "text/xml"},
				), nil
			},
			code: model.SourceErrorCodeSourceResponseTooLarge,
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
			_, err := client.fetch(
				context.Background(),
				updateListRequest{date: mustTestDate(t, "2023-02-01")},
			)
			assertSourceErrorCode(t, err, test.code)
		})
	}
}

func TestClientはgzip展開上限を検証する(t *testing.T) {
	t.Parallel()

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(
		[]byte(strings.Repeat("x", maximumDecompressedBytes+1)),
	); err != nil {
		t.Fatalf("gzip write error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip close error = %v", err)
	}
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			response := testResponse(
				http.StatusOK,
				compressed.String(),
				map[string]string{
					"Content-Type":     "text/xml",
					"Content-Encoding": "gzip",
				},
			)
			return response, nil
		}),
		now:   time.Now,
		sleep: noSleep,
	})
	_, err := client.fetch(
		context.Background(),
		updateListRequest{date: mustTestDate(t, "2023-02-01")},
	)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceResponseTooLarge)
}

func Test展開と解析は同じ処理期限を消費する(t *testing.T) {
	t.Parallel()

	xmlBody := []byte(
		`<DataRoot><Result><Code>0</Code><Message/></Result>` +
			`<ApplData><Date>20230201</Date></ApplData></DataRoot>`,
	)
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(xmlBody); err != nil {
		t.Fatalf("gzip write error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("gzip close error = %v", err)
	}

	parent := context.Background()
	processingContext, cancel := context.WithTimeout(
		parent,
		100*time.Millisecond,
	)
	defer cancel()
	decoded, err := decodeResponseBody(
		parent,
		processingContext,
		compressed.Bytes(),
		"gzip",
	)
	if err != nil {
		t.Fatalf("decodeResponseBody() error = %v", err)
	}
	<-processingContext.Done()
	_, err = parseResponseWithBudget(
		parent,
		processingContext,
		decoded,
	)
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceProcessingLimit)
}
