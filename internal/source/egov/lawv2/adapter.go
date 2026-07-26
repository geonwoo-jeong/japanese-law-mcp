package lawv2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/continuation"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

var (
	// sharedEGovHTTPGate は、将来追加する e-Gov operation と同じ process 枠を共有する。
	sharedEGovHTTPGate = make(chan struct{}, 1)
)

type adapterDependencies struct {
	client lawClient
	now    func() time.Time
	gate   chan struct{}
}

// LawSearchAdapter は、e-Gov API Version 2 の law.search@1 planned binding である。
//
// runtime registry と MCP route への登録は、四能力を揃える後続変更まで行わない。
type LawSearchAdapter struct {
	manager      *continuation.Manager
	dependencies adapterDependencies
}

var _ lawsearch.Port = (*LawSearchAdapter)(nil)

// NewLawSearchAdapter は、固定接続先を使う law.search@1 adapter を組み立てる。
func NewLawSearchAdapter(
	manager *continuation.Manager,
) (*LawSearchAdapter, error) {
	return newLawSearchAdapter(manager, adapterDependencies{
		client: newProductionClient(),
		now:    time.Now,
		gate:   sharedEGovHTTPGate,
	})
}

func newLawSearchAdapter(
	manager *continuation.Manager,
	dependencies adapterDependencies,
) (*LawSearchAdapter, error) {
	if manager == nil ||
		dependencies.now == nil ||
		dependencies.gate == nil ||
		dependencies.client.dependencies.doer == nil ||
		dependencies.client.dependencies.now == nil ||
		dependencies.client.dependencies.sleep == nil {
		return nil, fmt.Errorf("e-Gov law.search adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("e-Gov の同時実行枠は一件でなければなりません")
	}
	return &LawSearchAdapter{
		manager:      manager,
		dependencies: dependencies,
	}, nil
}

// Search は、法令名検索を固定された e-Gov GET /laws へ対応させる。
func (a *LawSearchAdapter) Search(
	ctx context.Context,
	request lawsearch.Request,
) (lawsearch.Page, error) {
	if ctx == nil {
		return lawsearch.Page{}, fmt.Errorf("context は必須です")
	}
	startedAt := a.dependencies.now()
	if err := request.Validate(); err != nil {
		return lawsearch.Page{}, err
	}
	if err := validateProviderQuery(request); err != nil {
		return lawsearch.Page{}, err
	}

	condition, err := conditionFingerprint(a.manager, request)
	if err != nil {
		return lawsearch.Page{}, err
	}
	config, err := configFingerprint(a.manager)
	if err != nil {
		return lawsearch.Page{}, err
	}

	effectiveAsOf, offset, err := a.resumeValues(
		request,
		startedAt,
		condition,
		config,
	)
	if err != nil {
		return lawsearch.Page{}, err
	}
	if effectiveAsOf.String() < "2017-04-01" {
		return lawsearch.Page{}, newSourceError(
			model.SourceErrorCodeUnsupportedQuery,
			"",
		)
	}

	if err := a.acquire(ctx); err != nil {
		return lawsearch.Page{}, err
	}
	defer a.release()

	fetched, err := a.dependencies.client.fetch(ctx, lawSearchRequest{
		query:  request.Query(),
		asOf:   effectiveAsOf,
		limit:  request.Limit(),
		offset: offset,
	})
	if err != nil {
		return lawsearch.Page{}, err
	}
	response, nextOffset, err := parseLawSearchResponse(
		ctx,
		fetched.body,
		request.Limit(),
		offset,
	)
	if err != nil {
		return lawsearch.Page{}, err
	}
	items, err := mapLawSearchItems(response, fetched.retrievedAt)
	if err != nil {
		return lawsearch.Page{}, err
	}

	nextToken := ""
	if nextOffset != nil {
		nextToken, err = issueContinuation(
			a.manager,
			request,
			condition,
			config,
			*nextOffset,
			effectiveAsOf,
		)
		if err != nil {
			return lawsearch.Page{}, err
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
		return lawsearch.Page{}, err
	}
	return lawsearch.NewPage(lawsearch.PageValues{
		Items: items,
		Page:  sourcePage,
	})
}

func validateProviderQuery(request lawsearch.Request) error {
	query := request.Query()
	if strings.HasPrefix(query, "/") && strings.HasSuffix(query, "/") {
		return newSourceError(model.SourceErrorCodeUnsupportedQuery, "")
	}
	if asOf, exists := request.AsOf(); exists &&
		asOf.String() < "2017-04-01" {
		return newSourceError(model.SourceErrorCodeUnsupportedQuery, "")
	}
	return nil
}

func (a *LawSearchAdapter) resumeValues(
	request lawsearch.Request,
	startedAt time.Time,
	condition continuation.ConditionFingerprint,
	config continuation.ConfigFingerprint,
) (model.Date, int, error) {
	if _, exists := request.ContinuationToken(); exists {
		position, err := verifyContinuation(
			a.manager,
			request,
			condition,
			config,
		)
		if err != nil {
			return model.Date{}, 0, err
		}
		return position.asOf, position.offset, nil
	}
	if asOf, exists := request.AsOf(); exists {
		return asOf, 0, nil
	}

	tokyo := time.FixedZone("Asia/Tokyo", 9*60*60)
	asOf, err := model.NewDate(startedAt.In(tokyo).Format(time.DateOnly))
	if err != nil {
		return model.Date{}, 0, err
	}
	return asOf, 0, nil
}

func (a *LawSearchAdapter) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeContextError(err)
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (a *LawSearchAdapter) release() {
	<-a.dependencies.gate
}
