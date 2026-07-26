package lawv2

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLawClientFetchesLawArticleWithArticleErrorContext(t *testing.T) {
	t.Parallel()

	client := mustTestClient(t, clientDependencies{
		doer: doerFunc(func(*http.Request) (*http.Response, error) {
			return response(
				http.StatusOK,
				`{"unexpected":true}`,
				map[string]string{"Content-Type": "application/json"},
			), nil
		}),
		now:   time.Now,
		sleep: sleepWithContext,
	})

	_, err := client.fetchLawArticle(
		context.Background(),
		lawDocumentRequest{identifier: "322CO0000000016"},
	)
	assertLawArticleSourceError(
		t,
		err,
		model.SourceErrorCodeSourceContractChanged,
	)
}

func TestLawClientMapsLawArticle404WithoutExposingBody(t *testing.T) {
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

	_, err := client.fetchLawArticle(
		context.Background(),
		lawDocumentRequest{identifier: "322CO0000000016"},
	)
	if !errors.Is(err, lawarticleread.ErrNotFound) {
		t.Fatalf("SOT-IF-012/025: error = %v", err)
	}
	if strings.Contains(err.Error(), "外部本文") {
		t.Fatal("SOT-IF-017: 外部本文がエラーへ露出した")
	}
	if attempts != 1 {
		t.Fatalf("SOT-IF-004: attempts = %d", attempts)
	}
}
