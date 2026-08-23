package lawv2

import (
	"context"
	"net/http"
	"testing"
)

func TestBuildLawRevisionHTTPRequestEncodesIdentifierAsOnePathSegment(t *testing.T) {
	t.Parallel()

	request, err := buildLawRevisionHTTPRequest(context.Background(), lawRevisionRequest{
		lawIDOrNumber: "令和三年/法律第三十六号",
	})
	if err != nil {
		t.Fatalf("request を作成できません: %v", err)
	}
	if request.Method != http.MethodGet ||
		request.URL.EscapedPath() !=
			"/api/2/law_revisions/%E4%BB%A4%E5%92%8C%E4%B8%89%E5%B9%B4%2F%E6%B3%95%E5%BE%8B%E7%AC%AC%E4%B8%89%E5%8D%81%E5%85%AD%E5%8F%B7" {
		t.Fatalf("request = %s %s", request.Method, request.URL.String())
	}
	query := request.URL.Query()
	if len(query) != 1 || query.Get("response_format") != "json" {
		t.Fatalf("query = %v", query)
	}
	if request.Header.Get("Accept") != "application/json" ||
		request.Header.Get("Accept-Encoding") != "gzip" {
		t.Fatalf("header = %v", request.Header)
	}
}
