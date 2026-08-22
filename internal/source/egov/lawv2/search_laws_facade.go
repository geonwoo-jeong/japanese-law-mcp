package lawv2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlaws"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type searchLawsFacadeDependencies struct {
	client lawClient
	gate   chan struct{}
}

// SearchLawsFacade は、公開 search_laws の数値 offset 契約を e-Gov へ対応させる。
type SearchLawsFacade struct {
	dependencies searchLawsFacadeDependencies
}

var _ searchlaws.Port = (*SearchLawsFacade)(nil)

// NewSearchLawsFacade は、固定接続先と共有同時実行枠を使う facade を返す。
func NewSearchLawsFacade() (*SearchLawsFacade, error) {
	return newSearchLawsFacade(searchLawsFacadeDependencies{
		client: newProductionClient(),
		gate:   sharedEGovHTTPGate,
	})
}

func newSearchLawsFacade(
	dependencies searchLawsFacadeDependencies,
) (*SearchLawsFacade, error) {
	if err := verifyEmbeddedLawSearchContract(); err != nil {
		return nil, err
	}
	if dependencies.gate == nil ||
		dependencies.client.dependencies.doer == nil ||
		dependencies.client.dependencies.now == nil ||
		dependencies.client.dependencies.sleep == nil {
		return nil, fmt.Errorf("公開 search_laws facade の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("e-Gov の同時実行枠は一件でなければなりません")
	}
	return &SearchLawsFacade{dependencies: dependencies}, nil
}

// Search は、公開 offset をそのまま GET /laws の取得位置へ対応させる。
func (f *SearchLawsFacade) Search(
	ctx context.Context,
	request searchlaws.Request,
) (model.LawSearchResult, error) {
	if ctx == nil {
		return model.LawSearchResult{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return model.LawSearchResult{}, err
	}
	if err := f.acquire(ctx); err != nil {
		return model.LawSearchResult{}, err
	}
	defer f.release()

	asOf, hasAsOf := request.AsOf()
	providerRequest := publicLawSearchRequest{
		query:  request.Query(),
		limit:  request.Limit(),
		offset: request.Offset(),
	}
	if hasAsOf {
		providerRequest.asOf = &asOf
	}
	fetched, err := f.dependencies.client.fetchWith(ctx, fetchSpec{
		build: func(requestContext context.Context) (*http.Request, error) {
			return buildPublicLawSearchHTTPRequest(
				requestContext,
				providerRequest,
			)
		},
		responseBytes:     maximumResponseBytes,
		decompressedBytes: maximumDecompressedBytes,
		mediaType:         "application/json",
		mediaTypeError:    model.SourceErrorCodeInvalidSourceResponse,
		sourceError:       newSourceError,
	})
	if err != nil {
		return model.LawSearchResult{}, err
	}
	response, nextOffset, err := parseLawSearchResponse(
		ctx,
		fetched.body,
		request.Limit(),
		request.Offset(),
	)
	if err != nil {
		return model.LawSearchResult{}, err
	}
	resources, err := mapLawSearchItems(response, fetched.retrievedAt)
	if err != nil {
		return model.LawSearchResult{}, err
	}
	items := make([]model.LawSummary, len(resources))
	for index, resource := range resources {
		items[index] = resource.Data()
	}
	result, err := model.NewLawSearchResult(model.LawSearchResultValues{
		TotalCount: response.totalCount,
		Items:      items,
		NextOffset: nextOffset,
	})
	if err != nil {
		return model.LawSearchResult{}, newSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return result, nil
}

func (f *SearchLawsFacade) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeContextError(err)
	}
	select {
	case f.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (f *SearchLawsFacade) release() {
	<-f.dependencies.gate
}
