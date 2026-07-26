package lawv2

import (
	"context"
	"net/url"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestBuildLawDocumentHTTPRequestUsesFixedXMLContract(t *testing.T) {
	t.Parallel()

	request, err := buildLawDocumentHTTPRequest(
		context.Background(),
		lawDocumentRequest{
			identifier: "322CO0000000016_20240401_506CO0000000161",
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-011: リクエスト作成のエラー = %v", err)
	}
	if request.Method != "GET" ||
		request.URL.Scheme != "https" ||
		request.URL.Host != "laws.e-gov.go.jp" ||
		request.URL.Path !=
			"/api/2/law_data/322CO0000000016_20240401_506CO0000000161" {
		t.Fatalf("SOT-IF-004/011: request URL = %q", request.URL.String())
	}
	query := request.URL.Query()
	if len(query) != 2 ||
		query.Get("response_format") != "xml" ||
		query.Get("law_full_text_format") != "xml" {
		t.Fatalf("SOT-IF-011: query = %#v", query)
	}
	if request.Header.Get("Accept") != "application/xml" ||
		request.Header.Get("Accept-Encoding") != "gzip" ||
		request.Header.Get("User-Agent") != "japanese-law-mcp/dev" {
		t.Fatalf("SOT-IF-004/011: headers = %#v", request.Header)
	}
}

func TestBuildLawDocumentHTTPRequestAddsAsOfAndEscapesIdentifier(t *testing.T) {
	t.Parallel()

	asOf := mustDate("2024-04-01")
	identifier := "法令/番号 ?#"
	request, err := buildLawDocumentHTTPRequest(
		context.Background(),
		lawDocumentRequest{
			identifier: identifier,
			asOf:       &asOf,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-011: リクエスト作成のエラー = %v", err)
	}
	wantEscapedPath := "/api/2/law_data/" + url.PathEscape(identifier)
	if request.URL.EscapedPath() != wantEscapedPath {
		t.Fatalf(
			"SOT-IF-011: escaped path = %q、期待値 = %q",
			request.URL.EscapedPath(),
			wantEscapedPath,
		)
	}
	if request.URL.Query().Get("asof") != "2024-04-01" {
		t.Fatalf("SOT-IF-011: asof = %q", request.URL.Query().Get("asof"))
	}
}

func TestLawDocumentRequestRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	zeroDate := model.Date{}
	for name, request := range map[string]lawDocumentRequest{
		"identifier 欠落": {},
		"asof が無効": {
			identifier: "322CO0000000016",
			asOf:       &zeroDate,
		},
	} {
		request := request
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := request.validate(); err == nil {
				t.Fatal("SOT-IF-011: 不正な provider request を受理した")
			}
		})
	}
}
