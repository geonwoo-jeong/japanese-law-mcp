package lawv2

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawrevisionlist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type lawRevisionAdapterDependencies struct {
	client lawClient
	gate   chan struct{}
}

// LawRevisionListAdapter は、e-Gov API Version 2 の law.revision.list@1 binding である。
type LawRevisionListAdapter struct {
	dependencies lawRevisionAdapterDependencies
}

var _ lawrevisionlist.Port = (*LawRevisionListAdapter)(nil)

// NewLawRevisionListAdapter は、固定接続先を使う law.revision.list@1 adapter を組み立てる。
func NewLawRevisionListAdapter() (*LawRevisionListAdapter, error) {
	return newLawRevisionListAdapter(lawRevisionAdapterDependencies{
		client: newProductionClient(),
		gate:   sharedEGovHTTPGate,
	})
}

func newLawRevisionListAdapter(
	dependencies lawRevisionAdapterDependencies,
) (*LawRevisionListAdapter, error) {
	if err := verifyEmbeddedLawRevisionContract(); err != nil {
		return nil, err
	}
	if dependencies.gate == nil ||
		dependencies.client.dependencies.doer == nil ||
		dependencies.client.dependencies.now == nil ||
		dependencies.client.dependencies.sleep == nil {
		return nil, fmt.Errorf("e-Gov law.revision.list adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("e-Gov の同時実行枠は一件でなければなりません")
	}
	return &LawRevisionListAdapter{dependencies: dependencies}, nil
}

// List は、一つの法令の完全な改正履歴を固定された e-Gov endpoint から取得する。
func (a *LawRevisionListAdapter) List(
	ctx context.Context,
	request lawrevisionlist.Request,
) (lawrevisionlist.Page, error) {
	if ctx == nil {
		return lawrevisionlist.Page{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return lawrevisionlist.Page{}, err
	}
	providerRequest := lawRevisionRequest{lawIDOrNumber: request.LawIDOrNumber()}
	endpointURL, err := lawRevisionURL(providerRequest)
	if err != nil {
		return lawrevisionlist.Page{}, err
	}
	if err := a.acquire(ctx); err != nil {
		return lawrevisionlist.Page{}, err
	}
	defer a.release()

	fetched, err := a.dependencies.client.fetchLawRevisions(ctx, providerRequest)
	if err != nil {
		return lawrevisionlist.Page{}, err
	}
	response, err := parseLawRevisionResponse(ctx, fetched.body)
	if err != nil {
		return lawrevisionlist.Page{}, err
	}
	items, err := mapLawRevisions(response, fetched.retrievedAt, endpointURL)
	if err != nil {
		return lawrevisionlist.Page{}, err
	}
	count := len(items)
	sourcePage, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: count,
		TotalCount:    &count,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		return lawrevisionlist.Page{}, newLawRevisionSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	page, err := lawrevisionlist.NewPage(lawrevisionlist.PageValues{
		LawID: response.lawID,
		Items: items,
		Page:  sourcePage,
	})
	if err != nil {
		return lawrevisionlist.Page{}, newLawRevisionSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return page, nil
}

func (a *LawRevisionListAdapter) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeContextErrorWithFactory(err, newLawRevisionSourceError)
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newLawRevisionSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (a *LawRevisionListAdapter) release() {
	<-a.dependencies.gate
}
