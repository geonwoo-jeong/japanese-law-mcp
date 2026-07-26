package lawv2

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/buildinfo"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const lawContentEndpoint = "https://laws.e-gov.go.jp/api/2/keyword"

type lawContentSearchRequest struct {
	keyword string
	asOf    model.Date
	limit   int
	offset  int
}

type publicLawContentSearchRequest struct {
	keyword string
	asOf    *model.Date
	limit   int
	offset  int
}

func buildLawContentKeyword(request lawcontentsearch.Request) (string, error) {
	allTerms := request.AllTerms()
	anyTerms := request.AnyTerms()
	excludeTerms := request.ExcludeTerms()
	parts := make([]string, 0, 3)
	if len(allTerms) > 0 {
		parts = append(parts, strings.Join(allTerms, " "))
	}
	switch len(anyTerms) {
	case 0:
	case 1:
		parts = append(parts, anyTerms[0])
	default:
		parts = append(parts, "("+strings.Join(anyTerms, "|")+")")
	}
	if len(excludeTerms) > 0 {
		values := make([]string, len(excludeTerms))
		for index, term := range excludeTerms {
			values[index] = "!" + term
		}
		parts = append(parts, strings.Join(values, " "))
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("keyword を構成できません")
	}
	return strings.Join(parts, " "), nil
}

func (r lawContentSearchRequest) validate() error {
	switch {
	case r.keyword == "":
		return fmt.Errorf("keyword は必須です")
	case r.asOf.IsZero():
		return fmt.Errorf("asof は必須です")
	case r.asOf.Validate() != nil:
		return fmt.Errorf("asof が有効ではありません")
	case r.limit < 1 || r.limit > 100:
		return fmt.Errorf("limit は 1 以上 100 以下でなければなりません")
	case r.offset < 0 || int64(r.offset) > math.MaxInt32:
		return fmt.Errorf("offset は 0 以上 2147483647 以下でなければなりません")
	default:
		return nil
	}
}

func buildLawContentHTTPRequest(
	ctx context.Context,
	request lawContentSearchRequest,
) (*http.Request, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if err := request.validate(); err != nil {
		return nil, err
	}

	endpoint, err := url.Parse(lawContentEndpoint)
	if err != nil {
		return nil, fmt.Errorf("e-Gov の固定 URL を解釈できません: %w", err)
	}
	query := endpoint.Query()
	query.Set("keyword", request.keyword)
	query.Set("asof", request.asOf.String())
	query.Set("limit", strconv.Itoa(request.limit))
	query.Set("offset", strconv.Itoa(request.offset))
	query.Set("response_format", "json")
	query.Set("order", "+law_info.law_id")
	query.Set("highlight_tag", "mark")
	endpoint.RawQuery = query.Encode()

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("e-Gov リクエストを作成できません: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Accept-Encoding", "gzip")
	httpRequest.Header.Set(
		"User-Agent",
		"japanese-law-mcp/"+buildinfo.Version(),
	)
	return httpRequest, nil
}

func (r publicLawContentSearchRequest) validate() error {
	switch {
	case r.keyword == "":
		return fmt.Errorf("keyword は必須です")
	case r.limit < 1 || r.limit > 100:
		return fmt.Errorf("limit は 1 以上 100 以下でなければなりません")
	case r.offset < 0 || int64(r.offset) > math.MaxInt32:
		return fmt.Errorf("offset は 0 以上 2147483647 以下でなければなりません")
	}
	if r.asOf != nil {
		if err := r.asOf.Validate(); err != nil {
			return fmt.Errorf("asof が有効ではありません")
		}
		if r.asOf.String() < "2017-04-01" {
			return fmt.Errorf("asof は 2017-04-01 以降でなければなりません")
		}
	}
	return nil
}

func buildPublicLawContentHTTPRequest(
	ctx context.Context,
	request publicLawContentSearchRequest,
) (*http.Request, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if err := request.validate(); err != nil {
		return nil, err
	}

	endpoint, err := url.Parse(lawContentEndpoint)
	if err != nil {
		return nil, fmt.Errorf("e-Gov の固定 URL を解釈できません: %w", err)
	}
	query := endpoint.Query()
	query.Set("keyword", request.keyword)
	if request.asOf != nil {
		query.Set("asof", request.asOf.String())
	}
	query.Set("limit", strconv.Itoa(request.limit))
	query.Set("offset", strconv.Itoa(request.offset))
	query.Set("response_format", "json")
	query.Set("order", "+law_info.law_id")
	query.Set("highlight_tag", "mark")
	endpoint.RawQuery = query.Encode()

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("e-Gov リクエストを作成できません: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Accept-Encoding", "gzip")
	httpRequest.Header.Set(
		"User-Agent",
		"japanese-law-mcp/"+buildinfo.Version(),
	)
	return httpRequest, nil
}
