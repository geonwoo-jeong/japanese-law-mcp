package lawv1

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestBuildHTTPRequestは固定GETと日付pathだけを使う(t *testing.T) {
	t.Parallel()

	request, err := buildHTTPRequest(
		context.Background(),
		updateListRequest{date: mustTestDate(t, "2023-02-01")},
	)
	if err != nil {
		t.Fatalf("buildHTTPRequest() error = %v", err)
	}
	if request.Method != http.MethodGet {
		t.Fatalf("method = %q", request.Method)
	}
	if request.URL.Scheme != "https" ||
		request.URL.Host != "laws.e-gov.go.jp" ||
		request.URL.Path != "/api/1/updatelawlists/20230201" ||
		request.URL.RawQuery != "" {
		t.Fatalf("URL = %q", request.URL.String())
	}
	if !strings.Contains(request.Header.Get("Accept"), "text/xml") {
		t.Fatalf("Accept = %q", request.Header.Get("Accept"))
	}
	if request.Header.Get("Authorization") != "" {
		t.Fatal("認証 header が付与されました")
	}
}

func TestBuildHTTPRequestはnilContextを拒否する(t *testing.T) {
	t.Parallel()

	//nolint:staticcheck // SOT-IF-015: nil context を拒否する境界契約を直接確認する。
	if _, err := buildHTTPRequest(
		nil,
		updateListRequest{date: mustTestDate(t, "2023-02-01")},
	); err == nil {
		t.Fatal("nil context を受理しました")
	}
}
