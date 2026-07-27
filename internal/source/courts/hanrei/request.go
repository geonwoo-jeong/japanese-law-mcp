package hanrei

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/buildinfo"
)

var errUnsafeCourtsRedirect = errors.New(
	"裁判所の redirect 先が固定 HTTPS origin と一致しません",
)

func buildSearchHTTPRequest(
	ctx context.Context,
	query string,
) (*http.Request, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if query == "" {
		return nil, fmt.Errorf("query は必須です")
	}
	endpoint, err := url.Parse(searchEndpoint)
	if err != nil {
		return nil, fmt.Errorf("裁判所の固定 URL を解釈できません")
	}
	parameters := endpoint.Query()
	parameters.Set("query1", query)
	endpoint.RawQuery = parameters.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("裁判所の検索要求を作成できません")
	}
	setSearchHeaders(request)
	return request, nil
}

func setSearchHeaders(request *http.Request) {
	request.Header.Set("Accept", "text/html")
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set("User-Agent", "japanese-law-mcp/"+buildinfo.Version())
}

func courtsRedirectPolicy(
	request *http.Request,
	via []*http.Request,
) error {
	if request == nil || request.URL == nil ||
		!isCourtsHTTPSOrigin(request.URL) {
		return errUnsafeCourtsRedirect
	}
	if len(via) >= 10 {
		return fmt.Errorf("裁判所の redirect 回数が上限を超えました")
	}
	return nil
}

func isCourtsHTTPSOrigin(value *url.URL) bool {
	return value != nil &&
		strings.EqualFold(value.Scheme, "https") &&
		strings.EqualFold(value.Hostname(), "www.courts.go.jp") &&
		value.Port() == "" &&
		value.User == nil &&
		value.Opaque == ""
}
