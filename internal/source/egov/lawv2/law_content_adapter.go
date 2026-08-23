package lawv2

import (
	"context"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/continuation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type lawContentAdapterDependencies struct {
	client lawClient
	now    func() time.Time
	gate   chan struct{}
}

// LawContentSearchAdapter は、e-Gov API Version 2 の law.content.search@1 planned binding である。
//
// runtime registry と MCP route への登録は、provider binding factory で一括して行う。
type LawContentSearchAdapter struct {
	manager      *continuation.Manager
	dependencies lawContentAdapterDependencies
}

var _ lawcontentsearch.Port = (*LawContentSearchAdapter)(nil)

// NewLawContentSearchAdapter は、固定接続先を使う law.content.search@1 adapter を組み立てる。
func NewLawContentSearchAdapter(
	manager *continuation.Manager,
) (*LawContentSearchAdapter, error) {
	return newLawContentSearchAdapter(manager, lawContentAdapterDependencies{
		client: newProductionClient(),
		now:    time.Now,
		gate:   sharedEGovHTTPGate,
	})
}

func newLawContentSearchAdapter(
	manager *continuation.Manager,
	dependencies lawContentAdapterDependencies,
) (*LawContentSearchAdapter, error) {
	if err := verifyEmbeddedLawContentContract(); err != nil {
		return nil, err
	}
	if manager == nil ||
		dependencies.now == nil ||
		dependencies.gate == nil ||
		dependencies.client.dependencies.doer == nil ||
		dependencies.client.dependencies.now == nil ||
		dependencies.client.dependencies.sleep == nil {
		return nil, fmt.Errorf("e-Gov law.content.search adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("e-Gov の同時実行枠は一件でなければなりません")
	}
	return &LawContentSearchAdapter{
		manager:      manager,
		dependencies: dependencies,
	}, nil
}

// Search は、構造化本文検索を固定された e-Gov GET /keyword へ対応させる。
func (a *LawContentSearchAdapter) Search(
	ctx context.Context,
	request lawcontentsearch.Request,
) (lawcontentsearch.Page, error) {
	if ctx == nil {
		return lawcontentsearch.Page{}, fmt.Errorf("context は必須です")
	}
	startedAt := a.dependencies.now()
	if err := request.Validate(); err != nil {
		return lawcontentsearch.Page{}, err
	}

	condition, err := lawContentConditionFingerprint(a.manager, request)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}
	config, err := lawContentConfigFingerprint(a.manager)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}
	effectiveAsOf, offset, err := effectiveLawContentAsOf(
		request,
		startedAt,
		a.manager,
		condition,
		config,
	)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}
	if effectiveAsOf.String() < "2017-04-01" {
		return lawcontentsearch.Page{}, newLawContentSourceError(
			model.SourceErrorCodeUnsupportedQuery,
			"",
		)
	}
	keyword, err := buildLawContentKeyword(request)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}

	if err := a.acquire(ctx); err != nil {
		return lawcontentsearch.Page{}, err
	}
	defer a.release()

	fetched, err := a.dependencies.client.fetchLawContent(ctx, lawContentSearchRequest{
		keyword: keyword,
		asOf:    effectiveAsOf,
		limit:   request.Limit(),
		offset:  offset,
	})
	if err != nil {
		return lawcontentsearch.Page{}, err
	}
	response, nextOffset, err := parseLawContentSearchResponse(
		ctx,
		fetched.body,
		request.Limit(),
		offset,
	)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}
	items, err := mapLawContentItems(response, fetched.retrievedAt)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}

	nextToken := ""
	if nextOffset != nil {
		nextToken, err = issueLawContentContinuation(
			a.manager,
			request,
			condition,
			config,
			*nextOffset,
			effectiveAsOf,
		)
		if err != nil {
			return lawcontentsearch.Page{}, err
		}
	}
	totalCount := response.totalCount
	sourcePage, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: len(items),
		NextToken:     nextToken,
		TotalCount:    &totalCount,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		return lawcontentsearch.Page{}, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	page, err := lawcontentsearch.NewPage(lawcontentsearch.PageValues{
		Items: items,
		Page:  sourcePage,
	})
	if err != nil {
		return lawcontentsearch.Page{}, newLawContentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return page, nil
}

func (a *LawContentSearchAdapter) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeLawContentContextError(err)
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newLawContentSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (a *LawContentSearchAdapter) release() {
	<-a.dependencies.gate
}
