package legalquery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

type servicePreprocessCall struct {
	ctx     context.Context
	request Request
}

type servicePreprocessorFake struct {
	mu    sync.Mutex
	calls []servicePreprocessCall
	hook  func(context.Context, Request) (PreprocessResult, error)
}

func (p *servicePreprocessorFake) Preprocess(
	ctx context.Context,
	request Request,
) (PreprocessResult, error) {
	p.mu.Lock()
	p.calls = append(p.calls, servicePreprocessCall{
		ctx:     ctx,
		request: request,
	})
	hook := p.hook
	p.mu.Unlock()
	if hook != nil {
		return hook(ctx, request)
	}
	return servicePreprocessResultForRequest(request)
}

func (p *servicePreprocessorFake) callsSnapshot() []servicePreprocessCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]servicePreprocessCall(nil), p.calls...)
}

func (p *servicePreprocessorFake) callCount() int {
	return len(p.callsSnapshot())
}

type servicePackStateFake struct {
	state func(string) (bool, bool)
}

func (s *servicePackStateFake) State(packID string) (bool, bool) {
	return s.state(packID)
}

func servicePreprocessResultForRequest(
	request Request,
) (PreprocessResult, error) {
	values := PreprocessResultValues{
		Query:         request.Query(),
		ComparisonKey: querynormalization.ComparisonKey(request.Query()),
	}
	if ref, exists := request.Ref(); exists {
		values.Ref = &ref
	}
	return NewPreprocessResult(values)
}

func mustServiceRequest(t *testing.T, limit *int) Request {
	t.Helper()
	request, err := NewRequest(RequestValues{
		Query:           "法情報を検索してください",
		LimitPerAttempt: limit,
	})
	if err != nil {
		t.Fatalf("試験用 request を作成できません: %v", err)
	}
	return request
}

func mustServiceProfileSet(
	t *testing.T,
	profile selectorTestProfile,
) QueryProfileSet {
	t.Helper()
	if profile.metadata.ProfileID() == "" {
		profile.metadata = mustSelectorTestMetadata(
			t,
			"core",
			selectorTestProfileVersion,
			selectorTestRankingVersion,
		)
	}
	profiles, err := NewQueryProfileSet([]QueryProfile{profile})
	if err != nil {
		t.Fatalf("試験用 profile set を作成できません: %v", err)
	}
	return profiles
}

func mustServiceExecutor(
	t *testing.T,
) (Executor, *executorTestRecorder, *executorCoreFacadeFake) {
	t.Helper()
	recorder, core, judicial := newExecutorTestFacades(t)
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("試験用 Executor を作成できません: %v", err)
	}
	return executor, recorder, core
}

func mustService(
	t *testing.T,
	preprocessor QueryPreprocessor,
	profiles QueryProfileSet,
	packState PackState,
	executor Executor,
	timeout time.Duration,
) *Service {
	t.Helper()
	service, err := NewService(
		preprocessor,
		profiles,
		packState,
		executor,
		timeout,
	)
	if err != nil {
		t.Fatalf("試験用 Service を作成できません: %v", err)
	}
	return service
}

func mustServiceSingleCandidate(t *testing.T) LegalQueryCandidate {
	t.Helper()
	return mustSelectorTestCandidate(
		t,
		"candidate-service",
		200,
		nil,
		1,
	)
}

func mustServiceSingleProfileSet(t *testing.T) QueryProfileSet {
	t.Helper()
	return mustServiceProfileSet(t, selectorTestProfile{
		candidates:     []LegalQueryCandidate{mustServiceSingleCandidate(t)},
		selectionMode:  QuerySelectionModeAutomatic,
		profileVersion: selectorTestProfileVersion,
		rankingVersion: selectorTestRankingVersion,
	})
}

func serviceResultStatus(
	t *testing.T,
	result LegalQueryResult,
) LegalQueryResultStatus {
	t.Helper()
	if result == nil {
		t.Fatal("統合照会 result が nil です")
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("統合照会 result が有効ではありません: %v", err)
	}
	return result.Status()
}

func serviceStageRecorder() (
	func(string),
	func() []string,
) {
	var mu sync.Mutex
	stages := make([]string, 0, 3)
	record := func(stage string) {
		mu.Lock()
		defer mu.Unlock()
		stages = append(stages, stage)
	}
	snapshot := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), stages...)
	}
	return record, snapshot
}
