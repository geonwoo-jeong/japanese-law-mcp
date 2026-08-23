package kokkai

import (
	"context"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const speechSearchGateHold = 3 * time.Second

var sharedDietSpeechGate = make(chan struct{}, 1)

type speechSearchSleeper func(context.Context, time.Duration) error

type speechSearchClient interface {
	fetchSpeechSearch(context.Context, parliamentspeechsearch.Request) (fetchedSpeechSearchResponse, error)
}

type speechSearchAdapterDependencies struct {
	client speechSearchClient
	now    func() time.Time
	sleep  speechSearchSleeper
	gate   chan struct{}
}

// SpeechSearchAdapter は、国会発言検索の provider-local adapter である。
type SpeechSearchAdapter struct {
	dependencies speechSearchAdapterDependencies
}

var _ parliamentspeechsearch.Port = (*SpeechSearchAdapter)(nil)

// NewSpeechSearchAdapter は、固定接続先を使う国会発言検索 adapter を返す。
func NewSpeechSearchAdapter() (*SpeechSearchAdapter, error) {
	client := newProductionSpeechSearchClient()
	return newSpeechSearchAdapter(speechSearchAdapterDependencies{
		client: client,
		now:    time.Now,
		sleep:  sleepSpeechSearchWithContext,
		gate:   sharedDietSpeechGate,
	})
}

func newSpeechSearchAdapter(
	dependencies speechSearchAdapterDependencies,
) (*SpeechSearchAdapter, error) {
	if dependencies.client == nil ||
		dependencies.now == nil ||
		dependencies.sleep == nil ||
		dependencies.gate == nil {
		return nil, fmt.Errorf("国会発言検索 adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("国会発言検索の共有同時実行枠は一件でなければなりません")
	}
	return &SpeechSearchAdapter{dependencies: dependencies}, nil
}

// Search は、固定された国会会議録検索 API を一度だけ呼び出して共通 page を返す。
func (a *SpeechSearchAdapter) Search(
	ctx context.Context,
	request parliamentspeechsearch.Request,
) (page parliamentspeechsearch.Page, err error) {
	if ctx == nil {
		return parliamentspeechsearch.Page{}, fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return parliamentspeechsearch.Page{}, err
	}
	if err := a.acquire(ctx); err != nil {
		return parliamentspeechsearch.Page{}, err
	}
	defer a.release()
	waitUntil := time.Time{}
	defer func() {
		if !waitUntil.IsZero() {
			remaining := waitUntil.Sub(a.dependencies.now())
			if remaining > 0 {
				if sleepErr := a.dependencies.sleep(ctx, remaining); sleepErr != nil && err == nil {
					err = normalizeSpeechSearchContextError(sleepErr)
					page = parliamentspeechsearch.Page{}
				}
			}
		}
	}()

	fetched, err := a.dependencies.client.fetchSpeechSearch(ctx, request)
	if err != nil {
		return parliamentspeechsearch.Page{}, err
	}
	waitUntil = fetched.retrievedAt.Add(speechSearchGateHold)

	result, err := parseAndMapSpeechSearchPage(ctx, fetched, request.Limit())
	if err != nil {
		return parliamentspeechsearch.Page{}, err
	}
	return result, nil
}

func (a *SpeechSearchAdapter) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeSpeechSearchContextError(err)
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newSpeechSearchSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (a *SpeechSearchAdapter) release() {
	<-a.dependencies.gate
}

func sleepSpeechSearchWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
