package lawv2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/buildinfo"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const lawDataEndpointPath = "/api/2/law_data/"

type lawDocumentRequest struct {
	identifier string
	asOf       *model.Date
}

func (r lawDocumentRequest) validate() error {
	if r.identifier == "" {
		return fmt.Errorf("law_id_or_num_or_revision_id は必須です")
	}
	if !utf8.ValidString(r.identifier) {
		return fmt.Errorf("law_id_or_num_or_revision_id は有効な UTF-8 でなければなりません")
	}
	if r.asOf != nil {
		if err := r.asOf.Validate(); err != nil {
			return fmt.Errorf("asof が有効ではありません: %w", err)
		}
	}
	return nil
}

func buildLawDocumentHTTPRequest(
	ctx context.Context,
	request lawDocumentRequest,
) (*http.Request, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if err := request.validate(); err != nil {
		return nil, err
	}

	endpoint := &url.URL{
		Scheme:  "https",
		Host:    "laws.e-gov.go.jp",
		Path:    lawDataEndpointPath + request.identifier,
		RawPath: lawDataEndpointPath + url.PathEscape(request.identifier),
	}
	query := endpoint.Query()
	query.Set("response_format", "xml")
	query.Set("law_full_text_format", "xml")
	if request.asOf != nil {
		query.Set("asof", request.asOf.String())
	}
	endpoint.RawQuery = query.Encode()

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("e-Gov 法令本文リクエストを作成できません: %w", err)
	}
	httpRequest.Header.Set("Accept", "application/xml")
	httpRequest.Header.Set("Accept-Encoding", "gzip")
	httpRequest.Header.Set(
		"User-Agent",
		"japanese-law-mcp/"+buildinfo.Version(),
	)
	return httpRequest, nil
}
