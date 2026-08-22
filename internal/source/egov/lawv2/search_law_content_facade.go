package lawv2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlawcontent"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type searchLawContentFacadeDependencies struct {
	client lawClient
	gate   chan struct{}
}

// SearchLawContentFacade は、公開 search_law_content の raw 検索式と数値 offset を e-Gov へ対応させる。
type SearchLawContentFacade struct {
	dependencies searchLawContentFacadeDependencies
}

var _ searchlawcontent.Port = (*SearchLawContentFacade)(nil)

// NewSearchLawContentFacade は、固定接続先と共有同時実行枠を使う facade を返す。
func NewSearchLawContentFacade() (*SearchLawContentFacade, error) {
	return newSearchLawContentFacade(searchLawContentFacadeDependencies{
		client: newProductionClient(),
		gate:   sharedEGovHTTPGate,
	})
}

func newSearchLawContentFacade(
	dependencies searchLawContentFacadeDependencies,
) (*SearchLawContentFacade, error) {
	if err := verifyEmbeddedLawContentContract(); err != nil {
		return nil, err
	}
	if dependencies.gate == nil ||
		dependencies.client.dependencies.doer == nil ||
		dependencies.client.dependencies.now == nil ||
		dependencies.client.dependencies.sleep == nil {
		return nil, fmt.Errorf("公開 search_law_content facade の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("e-Gov の同時実行枠は一件でなければなりません")
	}
	return &SearchLawContentFacade{dependencies: dependencies}, nil
}

// Search は、検証済み raw 検索式と offset をそのまま GET /keyword へ対応させる。
func (f *SearchLawContentFacade) Search(
	ctx context.Context,
	request searchlawcontent.Request,
) (model.LawContentSearchResult, error) {
	if ctx == nil {
		return model.LawContentSearchResult{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return model.LawContentSearchResult{}, err
	}
	if err := f.acquire(ctx); err != nil {
		return model.LawContentSearchResult{}, err
	}
	defer f.release()

	asOf, hasAsOf := request.AsOf()
	providerRequest := publicLawContentSearchRequest{
		keyword: request.Query(),
		limit:   request.Limit(),
		offset:  request.Offset(),
	}
	if hasAsOf {
		providerRequest.asOf = &asOf
	}
	fetched, err := f.dependencies.client.fetchWith(ctx, fetchSpec{
		build: func(requestContext context.Context) (*http.Request, error) {
			return buildPublicLawContentHTTPRequest(requestContext, providerRequest)
		},
		responseBytes:     lawContentResponseBytes,
		decompressedBytes: lawContentDecompressedBytes,
		mediaType:         "application/json",
		mediaTypeError:    model.SourceErrorCodeInvalidSourceResponse,
		sourceError:       newLawContentSourceError,
	})
	if err != nil {
		return model.LawContentSearchResult{}, err
	}
	response, nextOffset, err := parseLawContentSearchResponse(
		ctx,
		fetched.body,
		request.Limit(),
		request.Offset(),
	)
	if err != nil {
		return model.LawContentSearchResult{}, err
	}
	resources, err := mapLawContentItems(response, fetched.retrievedAt)
	if err != nil {
		return model.LawContentSearchResult{}, err
	}
	items := make([]model.LawContentMatch, len(resources))
	for index, resource := range resources {
		items[index] = resource.Data()
	}
	result, err := model.NewLawContentSearchResult(model.LawContentSearchResultValues{
		TotalCount: response.totalCount,
		Items:      items,
		NextOffset: nextOffset,
	})
	if err != nil {
		return model.LawContentSearchResult{}, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return result, nil
}

func (f *SearchLawContentFacade) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeLawContentContextError(err)
	}
	select {
	case f.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newLawContentSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (f *SearchLawContentFacade) release() {
	<-f.dependencies.gate
}
