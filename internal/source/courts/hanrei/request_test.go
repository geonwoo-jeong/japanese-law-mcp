package hanrei

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestBuildSearchHTTPRequestUsesOnlySingleEncodedQuery1(t *testing.T) {
	t.Parallel()
	request, err := buildSearchHTTPRequest(context.Background(), "判例 %2F")
	if err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodGet {
		t.Errorf("method = %q", request.Method)
	}
	if got := request.URL.String(); got !=
		"https://www.courts.go.jp/hanrei/search1/index.html?query1=%E5%88%A4%E4%BE%8B+%252F" {
		t.Errorf("SOT-IF-044: URL = %q", got)
	}
	if got := request.URL.Query(); len(got) != 1 || got.Get("query1") != "判例 %2F" {
		t.Errorf("SOT-IF-044: query = %#v", got)
	}
	if request.Header.Get("Cookie") != "" {
		t.Fatal("SOT-IF-043: Cookie header が設定された")
	}
}

func TestCourtsRedirectPolicyAllowsOnlyFixedHTTPSOrigin(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{"same origin", "https://www.courts.go.jp/hanrei/search1/index.html", false},
		{"http", "http://www.courts.go.jp/hanrei/search1/index.html", true},
		{"different host", "https://example.com/hanrei/search1/index.html", true},
		{"port", "https://www.courts.go.jp:443/hanrei/search1/index.html", true},
		{"userinfo", "https://user@www.courts.go.jp/hanrei/search1/index.html", true},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			target, err := url.Parse(testCase.rawURL)
			if err != nil {
				t.Fatal(err)
			}
			request := &http.Request{URL: target}
			gotErr := courtsRedirectPolicy(request, nil)
			if (gotErr != nil) != testCase.wantErr {
				t.Errorf("courtsRedirectPolicy() のエラー = %v", gotErr)
			}
		})
	}
}
