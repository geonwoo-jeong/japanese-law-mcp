package hanrei

import (
	"context"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const searchParseTimeout = 2 * time.Second

var sharedCourtsHanreiGate = make(chan struct{}, 1)

type searchAdapterDependencies struct {
	doer         httpDoer
	now          func() time.Time
	gate         chan struct{}
	parseTimeout time.Duration
}

// JudicialDecisionSearchAdapter は、裁判所 HTML の judicial-decision.search@1 adapter である。
type JudicialDecisionSearchAdapter struct {
	dependencies searchAdapterDependencies
}

var _ judicialdecisionsearch.Port = (*JudicialDecisionSearchAdapter)(nil)

// NewJudicialDecisionSearchAdapter は、固定された裁判所 HTTPS origin を使う adapter を返す。
func NewJudicialDecisionSearchAdapter() (*JudicialDecisionSearchAdapter, error) {
	return newJudicialDecisionSearchAdapter(searchAdapterDependencies{
		doer:         newProductionHTTPClient(),
		now:          time.Now,
		gate:         sharedCourtsHanreiGate,
		parseTimeout: searchParseTimeout,
	})
}

func newJudicialDecisionSearchAdapter(
	dependencies searchAdapterDependencies,
) (*JudicialDecisionSearchAdapter, error) {
	if dependencies.doer == nil ||
		dependencies.now == nil ||
		dependencies.gate == nil ||
		dependencies.parseTimeout < 0 {
		return nil, fmt.Errorf("裁判所 search adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("裁判所の共有同時実行枠は一件でなければなりません")
	}
	return &JudicialDecisionSearchAdapter{dependencies: dependencies}, nil
}

// Search は、検索語を固定 GET query1 へ一度だけ対応させる。
func (a *JudicialDecisionSearchAdapter) Search(
	ctx context.Context,
	request judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	if ctx == nil {
		return judicialdecisionsearch.Page{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	if _, exists := request.ContinuationToken(); exists {
		argumentError, err := judicialdecisionsearch.NewArgumentError(
			"continuationToken",
			"courts-hanrei-html では使用できません",
		)
		if err != nil {
			return judicialdecisionsearch.Page{}, fmt.Errorf(
				"裁判所 search adapter の入力エラーを分類できません: %w",
				err,
			)
		}
		return judicialdecisionsearch.Page{}, argumentError
	}
	if err := a.acquire(ctx); err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	defer a.release()
	return a.searchAcquired(ctx, request)
}

func (a *JudicialDecisionSearchAdapter) searchAcquired(
	ctx context.Context,
	request judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	fetched, err := fetchSearchResponse(
		ctx,
		a.dependencies.doer,
		a.dependencies.now,
		request.Query(),
	)
	if err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	processing, cancel := context.WithTimeout(ctx, a.dependencies.parseTimeout)
	defer cancel()
	body, err := decodeSearchResponseBody(ctx, processing, fetched)
	if err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	response, err := parseSearchResponse(processing, body)
	if err != nil {
		if ctx.Err() != nil {
			return judicialdecisionsearch.Page{}, normalizeSearchContextError(ctx.Err())
		}
		return judicialdecisionsearch.Page{}, err
	}
	return mapSearchPage(
		response,
		fetched.fetchedURL,
		fetched.retrievedAt,
		request.Limit(),
	)
}

func (a *JudicialDecisionSearchAdapter) acquire(ctx context.Context) error {
	if ctx.Err() != nil {
		return normalizeSearchContextError(ctx.Err())
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newSearchSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (a *JudicialDecisionSearchAdapter) release() {
	<-a.dependencies.gate
}
