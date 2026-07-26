package lawv2

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawcontentsearch"
)

func TestBuildLawContentKeywordGolden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values lawcontentsearch.RequestValues
		want   string
	}{
		{
			name:   "allTerms だけ",
			values: lawcontentsearch.RequestValues{AllTerms: []string{"行政", "手続"}},
			want:   "行政 手続",
		},
		{
			name:   "anyTerms 一件",
			values: lawcontentsearch.RequestValues{AnyTerms: []string{"申請"}},
			want:   "申請",
		},
		{
			name:   "anyTerms 複数",
			values: lawcontentsearch.RequestValues{AnyTerms: []string{"申請", "届出"}},
			want:   "(申請|届出)",
		},
		{
			name: "正の条件と除外条件",
			values: lawcontentsearch.RequestValues{
				AllTerms:     []string{"行政"},
				AnyTerms:     []string{"申請", "届出"},
				ExcludeTerms: []string{"廃止", "旧法"},
			},
			want: "行政 (申請|届出) !廃止 !旧法",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			request := mustLawContentSearchRequest(t, test.values)
			got, err := buildLawContentKeyword(request)
			if err != nil {
				t.Fatalf("SOT-IF-028: keyword 生成のエラー = %v", err)
			}
			if got != test.want {
				t.Fatalf("SOT-IF-028: keyword = %q、期待値 = %q", got, test.want)
			}
		})
	}
}

func TestBuildLawContentHTTPRequestUsesFixedQueryAndPercentEncoding(t *testing.T) {
	t.Parallel()

	request, err := buildLawContentHTTPRequest(
		context.Background(),
		lawContentSearchRequest{
			keyword: "(行政|地方) !廃止",
			asOf:    mustDate("2026-07-26"),
			limit:   20,
			offset:  40,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-028: HTTP request 生成のエラー = %v", err)
	}
	if request.Method != http.MethodGet ||
		request.URL.Scheme != "https" ||
		request.URL.Host != "laws.e-gov.go.jp" ||
		request.URL.Path != "/api/2/keyword" {
		t.Fatalf("SOT-IF-028: request = %#v", request)
	}
	query := request.URL.Query()
	want := map[string]string{
		"keyword":         "(行政|地方) !廃止",
		"asof":            "2026-07-26",
		"limit":           "20",
		"offset":          "40",
		"response_format": "json",
		"order":           "+law_info.law_id",
		"highlight_tag":   "mark",
	}
	for key, value := range want {
		if got := query.Get(key); got != value {
			t.Errorf("SOT-IF-028: %s = %q、期待値 = %q", key, got, value)
		}
	}
	if query.Has("sentences_limit") {
		t.Fatal("SOT-IF-028: sentences_limit を送信した")
	}
	if len(query) != len(want) {
		t.Fatalf("SOT-IF-028: query = %#v", query)
	}
	if strings.Contains(request.URL.RawQuery, "(行政|地方)") ||
		!strings.Contains(request.URL.RawQuery, "keyword=%28") ||
		!strings.Contains(request.URL.RawQuery, "%7C") {
		t.Fatalf("SOT-IF-028: percent-encoding = %q", request.URL.RawQuery)
	}
}

func TestBuildPublicLawContentHTTPRequestPreservesRawDSLAndOmittedAsOf(t *testing.T) {
	t.Parallel()

	request, err := buildPublicLawContentHTTPRequest(
		context.Background(),
		publicLawContentSearchRequest{
			keyword: "情報公開|個人情報",
			limit:   20,
			offset:  40,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-010: HTTP request 生成のエラー = %v", err)
	}
	query := request.URL.Query()
	if query.Get("keyword") != "情報公開|個人情報" {
		t.Fatalf("SOT-IF-010: keyword = %q", query.Get("keyword"))
	}
	if query.Has("asof") {
		t.Fatalf("SOT-IF-010: 省略した asof を送信した: %q", query.Get("asof"))
	}
	if query.Get("limit") != "20" ||
		query.Get("offset") != "40" ||
		query.Get("response_format") != "json" ||
		query.Get("order") != "+law_info.law_id" ||
		query.Get("highlight_tag") != "mark" {
		t.Fatalf("SOT-IF-010: query = %#v", query)
	}
	if query.Has("sentences_limit") {
		t.Fatal("SOT-IF-010: sentences_limit を送信した")
	}
	if strings.Contains(request.URL.RawQuery, "情報公開|個人情報") {
		t.Fatalf("SOT-IF-010: keyword が percent-encoding されていない: %q", request.URL.RawQuery)
	}
}

func TestBuildPublicLawContentHTTPRequestSendsExplicitAsOf(t *testing.T) {
	t.Parallel()

	asOf := mustDate("2026-07-26")
	request, err := buildPublicLawContentHTTPRequest(
		context.Background(),
		publicLawContentSearchRequest{
			keyword: "第?条",
			asOf:    &asOf,
			limit:   1,
			offset:  2147483647,
		},
	)
	if err != nil {
		t.Fatalf("SOT-IF-010: HTTP request 生成のエラー = %v", err)
	}
	query := request.URL.Query()
	if query.Get("keyword") != "第?条" ||
		query.Get("asof") != "2026-07-26" ||
		query.Get("offset") != "2147483647" {
		t.Fatalf("SOT-IF-010: query = %#v", query)
	}
}
