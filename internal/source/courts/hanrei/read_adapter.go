package hanrei

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const readParseTimeout = 2 * time.Second

type readAdapterDependencies struct {
	doer         httpDoer
	now          func() time.Time
	gate         chan struct{}
	parseTimeout time.Duration
}

// JudicialDecisionReadAdapter は、裁判所 HTML の judicial-decision.read@1 adapter である。
type JudicialDecisionReadAdapter struct {
	dependencies readAdapterDependencies
}

var _ judicialdecisionread.Port = (*JudicialDecisionReadAdapter)(nil)

// NewJudicialDecisionReadAdapter は、固定された裁判所 HTTPS origin を使う adapter を返す。
func NewJudicialDecisionReadAdapter() (*JudicialDecisionReadAdapter, error) {
	return newJudicialDecisionReadAdapter(readAdapterDependencies{
		doer:         newProductionHTTPClient(),
		now:          time.Now,
		gate:         sharedCourtsHanreiGate,
		parseTimeout: readParseTimeout,
	})
}

func newJudicialDecisionReadAdapter(
	dependencies readAdapterDependencies,
) (*JudicialDecisionReadAdapter, error) {
	if dependencies.doer == nil ||
		dependencies.now == nil ||
		dependencies.gate == nil ||
		dependencies.parseTimeout < 0 {
		return nil, fmt.Errorf("裁判所 read adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("裁判所の共有同時実行枠は一件でなければなりません")
	}
	return &JudicialDecisionReadAdapter{dependencies: dependencies}, nil
}

func (a *JudicialDecisionReadAdapter) Read(
	ctx context.Context,
	request judicialdecisionread.Request,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	if ctx == nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	if _, _, err := validateReadRef(request); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	if err := a.acquire(ctx); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	defer a.release()
	fetched, err := fetchReadResponse(ctx, a.dependencies.doer, a.dependencies.now, request)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	processing, cancel := context.WithTimeout(ctx, a.dependencies.parseTimeout)
	defer cancel()
	body, err := decodeReadResponseBody(ctx, processing, fetched)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	response, err := parseReadResponse(processing, body)
	if err != nil {
		if ctx.Err() != nil {
			return model.SourcedResource[model.JudicialDecisionDetails]{}, normalizeReadContextError(ctx.Err())
		}
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	return mapReadDetails(response, fetched, request.Ref())
}

func normalizeReadContextError(err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return newReadSourceError(model.SourceErrorCodeSourceTimeout, "")
	case errors.Is(err, context.Canceled):
		return context.Canceled
	default:
		return newReadSourceError(model.SourceErrorCodeSourceUnavailable, "")
	}
}

func (a *JudicialDecisionReadAdapter) acquire(ctx context.Context) error {
	if ctx.Err() != nil {
		return normalizeReadContextError(ctx.Err())
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newReadSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (a *JudicialDecisionReadAdapter) release() {
	<-a.dependencies.gate
}
