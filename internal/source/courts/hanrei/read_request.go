package hanrei

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/buildinfo"
)

var detailResourceIDPattern = regexp.MustCompile(`^([0-9]+)/detail([2-8])$`)

func buildReadHTTPRequest(
	ctx context.Context,
	request judicialdecisionread.Request,
) (*http.Request, string, string, string, error) {
	if ctx == nil {
		return nil, "", "", "", fmt.Errorf("context は必須です")
	}
	decisionID, categoryNumber, err := validateReadRef(request)
	if err != nil {
		return nil, "", "", "", err
	}
	endpoint, err := url.Parse(readDetailURL(decisionID, categoryNumber))
	if err != nil {
		return nil, "", "", "", fmt.Errorf("裁判所の固定 URL を解釈できません")
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint.String(),
		nil,
	)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("裁判所の詳細要求を作成できません")
	}
	httpRequest.Header.Set("Accept", "text/html")
	httpRequest.Header.Set("Accept-Encoding", "gzip")
	httpRequest.Header.Set("User-Agent", "japanese-law-mcp/"+buildinfo.Version())
	return httpRequest, endpoint.String(), decisionID, categoryNumber, nil
}

func validateReadRef(request judicialdecisionread.Request) (string, string, error) {
	if err := request.Validate(); err != nil {
		return "", "", err
	}
	ref := request.Ref()
	key := ref.Key()
	switch {
	case ref.ProviderID() != providerID:
		return "", "", newReadArgumentError("providerId が courts-hanrei-html ではありません")
	case key.SourceID() != sourceID:
		return "", "", newReadArgumentError("sourceId が courts-hanrei ではありません")
	case key.ResourceType() != judicialDecisionResourceType:
		return "", "", newReadArgumentError("resourceType が judicial-decision ではありません")
	}
	if _, exists := key.VersionID(); exists {
		return "", "", newReadArgumentError("versionId は指定できません")
	}
	match := detailResourceIDPattern.FindStringSubmatch(key.ResourceID())
	if match == nil {
		return "", "", newReadArgumentError("resourceId は {decisionId}/detail{2..8} でなければなりません")
	}
	return match[1], match[2], nil
}

func newReadArgumentError(reason string) error {
	err, createErr := judicialdecisionread.NewArgumentError("ref", reason)
	if createErr != nil {
		return fmt.Errorf("裁判所 read adapter の入力エラーを分類できません: %w", createErr)
	}
	return err
}
