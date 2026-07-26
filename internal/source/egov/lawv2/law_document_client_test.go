package lawv2

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLawClientFetchesLawDocumentXML(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(request *http.Request) (*http.Response, error) {
			if request.URL.Path != "/api/2/law_data/322CO0000000016" {
				t.Fatalf("SOT-IF-011: path = %q", request.URL.Path)
			}
			return response(
				http.StatusOK,
				`<law_data_response/>`,
				map[string]string{
					"Content-Type": "application/xml; charset=UTF-8",
				},
			), nil
		}),
		now: func() time.Time { return now },
		sleep: func(context.Context, time.Duration) error {
			t.Fatal("成功時に sleep してはなりません")
			return nil
		},
	})

	result, err := client.fetchLawDocument(
		context.Background(),
		lawDocumentRequest{identifier: "322CO0000000016"},
	)
	if err != nil {
		t.Fatalf("SOT-IF-004/011: fetchLawDocument() のエラー = %v", err)
	}
	if string(result.body) != `<law_data_response/>` ||
		!result.retrievedAt.Equal(now) {
		t.Fatalf("SOT-IF-011: result = %#v", result)
	}
}

func TestLawClientMapsLawDocument404WithoutExposingBody(t *testing.T) {
	t.Parallel()

	attempts := 0
	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			attempts++
			return response(
				http.StatusNotFound,
				"利用者へ露出してはならない外部本文",
				nil,
			), nil
		}),
		now: time.Now,
		sleep: func(context.Context, time.Duration) error {
			t.Fatal("404 を再試行してはなりません")
			return nil
		},
	})

	_, err := client.fetchLawDocument(
		context.Background(),
		lawDocumentRequest{identifier: "322CO0000000016"},
	)
	if !errors.Is(err, lawdocumentread.ErrNotFound) {
		t.Fatalf("SOT-IF-011/024: error = %v", err)
	}
	if strings.Contains(err.Error(), "外部本文") {
		t.Fatal("SOT-IF-017: 外部本文がエラーへ露出した")
	}
	if attempts != 1 {
		t.Fatalf("SOT-IF-004: attempts = %d", attempts)
	}
}

func TestLawClientRejectsUnexpectedLawDocumentMediaType(t *testing.T) {
	t.Parallel()

	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			return response(
				http.StatusOK,
				`{"law_full_text":{}}`,
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
		now: time.Now,
		sleep: func(context.Context, time.Duration) error {
			return nil
		},
	})

	_, err := client.fetchLawDocument(
		context.Background(),
		lawDocumentRequest{identifier: "322CO0000000016"},
	)
	assertLawDocumentSourceError(
		t,
		err,
		model.SourceErrorCodeSourceContractChanged,
	)
}

func TestLawClientAppliesLawDocumentBodyBudgets(t *testing.T) {
	t.Parallel()

	t.Run("16 MiB response", func(t *testing.T) {
		t.Parallel()

		client := mustTestClient(t, clientDependencies{
			doer: doerFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode:    http.StatusOK,
					ContentLength: lawDocumentResponseBytes + 1,
					Header: http.Header{
						"Content-Type": []string{"application/xml"},
					},
					Body: io.NopCloser(strings.NewReader("")),
				}, nil
			}),
			now:   time.Now,
			sleep: sleepWithContext,
		})
		_, err := client.fetchLawDocument(
			context.Background(),
			lawDocumentRequest{identifier: "322CO0000000016"},
		)
		assertLawDocumentSourceError(
			t,
			err,
			model.SourceErrorCodeSourceResponseTooLarge,
		)
	})

	t.Run("32 MiB decompressed", func(t *testing.T) {
		t.Parallel()

		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err := io.Copy(
			writer,
			io.LimitReader(zeroReader{}, lawDocumentDecompressedBytes+1),
		); err != nil {
			t.Fatalf("gzip の作成に失敗した: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("gzip を閉じられない: %v", err)
		}
		client := mustTestClient(t, clientDependencies{
			doer: doerFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Content-Type":     []string{"application/xml"},
						"Content-Encoding": []string{"gzip"},
					},
					Body: io.NopCloser(bytes.NewReader(compressed.Bytes())),
				}, nil
			}),
			now:   time.Now,
			sleep: sleepWithContext,
		})
		_, err := client.fetchLawDocument(
			context.Background(),
			lawDocumentRequest{identifier: "322CO0000000016"},
		)
		assertLawDocumentSourceError(
			t,
			err,
			model.SourceErrorCodeSourceResponseTooLarge,
		)
	})
}
