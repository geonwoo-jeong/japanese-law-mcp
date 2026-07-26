package lawv1

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/buildinfo"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const updateListEndpointPath = "/api/1/updatelawlists/"

type updateListRequest struct {
	date model.Date
}

func (r updateListRequest) validate() error {
	if err := r.date.Validate(); err != nil {
		return fmt.Errorf("date が有効ではありません: %w", err)
	}
	return nil
}

func (r updateListRequest) upstreamDate() string {
	return strings.ReplaceAll(r.date.String(), "-", "")
}

func buildHTTPRequest(
	ctx context.Context,
	request updateListRequest,
) (*http.Request, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if err := request.validate(); err != nil {
		return nil, err
	}
	date := request.upstreamDate()
	endpoint := &url.URL{
		Scheme:  "https",
		Host:    "laws.e-gov.go.jp",
		Path:    updateListEndpointPath + date,
		RawPath: updateListEndpointPath + url.PathEscape(date),
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("e-Gov 更新一覧リクエストを作成できません: %w", err)
	}
	httpRequest.Header.Set("Accept", "text/xml")
	httpRequest.Header.Set("Accept-Encoding", "gzip")
	httpRequest.Header.Set(
		"User-Agent",
		"japanese-law-mcp/"+buildinfo.Version(),
	)
	return httpRequest, nil
}
