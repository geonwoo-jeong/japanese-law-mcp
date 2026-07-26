package lawv1

import (
	"context"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const firstSupportedDate = "2020-11-24"

var sharedHTTPGate = make(chan struct{}, 1)

type adapterDependencies struct {
	client updateListClient
	now    func() time.Time
	gate   chan struct{}
}

// LawUpdateListAdapter は、e-Gov API Version 1 の更新一覧を共通契約へ対応させる。
type LawUpdateListAdapter struct {
	dependencies adapterDependencies
}

var _ lawupdatelist.Port = (*LawUpdateListAdapter)(nil)

// NewLawUpdateListAdapter は、固定接続先を使う更新一覧 adapter を返す。
func NewLawUpdateListAdapter() (*LawUpdateListAdapter, error) {
	return newLawUpdateListAdapter(adapterDependencies{
		client: newProductionClient(),
		now:    time.Now,
		gate:   sharedHTTPGate,
	})
}

func newLawUpdateListAdapter(
	dependencies adapterDependencies,
) (*LawUpdateListAdapter, error) {
	if dependencies.now == nil ||
		dependencies.gate == nil ||
		dependencies.client.dependencies.doer == nil ||
		dependencies.client.dependencies.now == nil ||
		dependencies.client.dependencies.sleep == nil {
		return nil, fmt.Errorf("e-Gov Version 1 adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("e-Gov Version 1 の同時実行枠は一件でなければなりません")
	}
	return &LawUpdateListAdapter{dependencies: dependencies}, nil
}

// List は、対象日の完全な更新一覧を取得する。
func (a *LawUpdateListAdapter) List(
	ctx context.Context,
	request lawupdatelist.Request,
) (lawupdatelist.Page, error) {
	if ctx == nil {
		return lawupdatelist.Page{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return lawupdatelist.Page{}, err
	}
	if err := a.validateSupportedDate(request.Date()); err != nil {
		return lawupdatelist.Page{}, err
	}
	if err := a.acquire(ctx); err != nil {
		return lawupdatelist.Page{}, err
	}
	defer a.release()

	fetched, err := a.dependencies.client.fetch(
		ctx,
		updateListRequest{date: request.Date()},
	)
	if err != nil {
		return lawupdatelist.Page{}, err
	}
	defer fetched.cancelProcessing()
	response, err := parseResponseWithBudget(
		ctx,
		fetched.processingContext,
		fetched.body,
	)
	if err != nil {
		return lawupdatelist.Page{}, err
	}
	return mapPage(
		response,
		fetched.statusCode,
		request.Date(),
		fetched.retrievedAt,
	)
}

func (a *LawUpdateListAdapter) validateSupportedDate(
	date model.Date,
) error {
	tokyo := time.FixedZone("Asia/Tokyo", 9*60*60)
	currentDate := a.dependencies.now().In(tokyo).Format(time.DateOnly)
	if date.String() < firstSupportedDate || date.String() > currentDate {
		return newSourceError(model.SourceErrorCodeUnsupportedQuery, "")
	}
	return nil
}

func (a *LawUpdateListAdapter) acquire(ctx context.Context) error {
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

func (a *LawUpdateListAdapter) release() {
	<-a.dependencies.gate
}
