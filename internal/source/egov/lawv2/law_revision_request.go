package lawv2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/buildinfo"
)

const lawRevisionEndpointPath = "/api/2/law_revisions/"

type lawRevisionRequest struct {
	lawIDOrNumber string
}

func (r lawRevisionRequest) validate() error {
	if r.lawIDOrNumber == "" {
		return fmt.Errorf("law_id_or_num は必須です")
	}
	if !utf8.ValidString(r.lawIDOrNumber) {
		return fmt.Errorf("law_id_or_num は有効な UTF-8 でなければなりません")
	}
	return nil
}

func buildLawRevisionHTTPRequest(
	ctx context.Context,
	request lawRevisionRequest,
) (*http.Request, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	endpoint, err := lawRevisionURL(request)
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("e-Gov 法令改正履歴リクエストを作成できません: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Accept-Encoding", "gzip")
	httpRequest.Header.Set(
		"User-Agent",
		"japanese-law-mcp/"+buildinfo.Version(),
	)
	return httpRequest, nil
}

func lawRevisionURL(request lawRevisionRequest) (string, error) {
	if err := request.validate(); err != nil {
		return "", err
	}
	endpoint := &url.URL{
		Scheme:  "https",
		Host:    "laws.e-gov.go.jp",
		Path:    lawRevisionEndpointPath + request.lawIDOrNumber,
		RawPath: lawRevisionEndpointPath + url.PathEscape(request.lawIDOrNumber),
	}
	query := endpoint.Query()
	query.Set("response_format", "json")
	endpoint.RawQuery = query.Encode()
	return endpoint.String(), nil
}
