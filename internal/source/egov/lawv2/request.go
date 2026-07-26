package lawv2

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/buildinfo"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const lawsEndpoint = "https://laws.e-gov.go.jp/api/2/laws"

type lawSearchRequest struct {
	query  string
	asOf   model.Date
	limit  int
	offset int
}

func (r lawSearchRequest) validate() error {
	switch {
	case r.query == "":
		return fmt.Errorf("law_title は必須です")
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

func buildHTTPRequest(
	ctx context.Context,
	request lawSearchRequest,
) (*http.Request, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if err := request.validate(); err != nil {
		return nil, err
	}

	endpoint, err := url.Parse(lawsEndpoint)
	if err != nil {
		return nil, fmt.Errorf("e-Gov の固定 URL を解釈できません: %w", err)
	}
	query := endpoint.Query()
	query.Set("law_title", request.query)
	query.Set("asof", request.asOf.String())
	query.Set("limit", strconv.Itoa(request.limit))
	query.Set("offset", strconv.Itoa(request.offset))
	query.Set("response_format", "json")
	query.Set("order", "+law_info.law_id")
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
