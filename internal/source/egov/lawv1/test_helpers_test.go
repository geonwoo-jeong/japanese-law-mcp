package lawv1

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type doerFunc func(*http.Request) (*http.Response, error)

func (function doerFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

func mustTestDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("date %q を作成できません: %v", value, err)
	}
	return date
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	//nolint:gosec // SOT-IF-035: テスト専用の固定 fixtures ディレクトリだけを読み込む。
	value, err := os.ReadFile(filepath.Join("fixtures", name))
	if err != nil {
		t.Fatalf("fixture %q を読み込めません: %v", name, err)
	}
	return string(value)
}

func testResponse(
	status int,
	body string,
	headers map[string]string,
) *http.Response {
	header := make(http.Header)
	for name, value := range headers {
		header.Set(name, value)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func mustTestClient(
	t *testing.T,
	dependencies clientDependencies,
) updateListClient {
	t.Helper()
	client, err := newUpdateListClient(dependencies)
	if err != nil {
		t.Fatalf("test client を作成できません: %v", err)
	}
	return client
}

func noSleep(
	_ context.Context,
	_ time.Duration,
) error {
	return nil
}

func assertSourceErrorCode(
	t *testing.T,
	err error,
	want model.SourceErrorCode,
) {
	t.Helper()
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("error = %v、SourceError ではありません", err)
	}
	if sourceError.Code() != want {
		t.Fatalf("code = %q、期待値は %q です", sourceError.Code(), want)
	}
}
