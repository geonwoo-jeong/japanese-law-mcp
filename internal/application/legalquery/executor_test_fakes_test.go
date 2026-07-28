package legalquery

import (
	"context"
	"errors"
	"sync"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

var errExecutorTestTypedNilFacade = errors.New("試験用 facade が typed nil です")

type executorTestCall struct {
	kind   LogicalInputKind
	ctx    context.Context
	budget LegalQueryStepBudget
}

type executorTestCallHook func(
	context.Context,
	LogicalInputKind,
	LegalQueryStepBudget,
) error

type executorTestRecorder struct {
	mu    sync.Mutex
	calls []executorTestCall
	errs  map[LogicalInputKind]error
	hook  executorTestCallHook
}

func newExecutorTestRecorder() *executorTestRecorder {
	return &executorTestRecorder{errs: map[LogicalInputKind]error{}}
}

func (r *executorTestRecorder) call(
	ctx context.Context,
	kind LogicalInputKind,
	budget LegalQueryStepBudget,
) error {
	r.mu.Lock()
	r.calls = append(r.calls, executorTestCall{
		kind:   kind,
		ctx:    ctx,
		budget: budget.clone(),
	})
	hook := r.hook
	configuredErr := r.errs[kind]
	r.mu.Unlock()
	if hook != nil {
		if err := hook(ctx, kind, budget); err != nil {
			return err
		}
	}
	return configuredErr
}

func (r *executorTestRecorder) callsSnapshot() []executorTestCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := make([]executorTestCall, len(r.calls))
	copy(cloned, r.calls)
	return cloned
}

func (r *executorTestRecorder) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

type executorCoreFacadeFake struct {
	recorder    *executorTestRecorder
	payloads    executorTestPayloads
	validateErr error
}

func (f *executorCoreFacadeFake) Validate() error {
	if f == nil {
		return errExecutorTestTypedNilFacade
	}
	return f.validateErr
}

func (f *executorCoreFacadeFake) SearchLaws(
	ctx context.Context,
	_ LawSearchIntentV1,
	budget LegalQueryStepBudget,
) (lawsearch.Page, error) {
	err := f.recorder.call(ctx, InputKindLawSearch, budget)
	return f.payloads.lawSearch, err
}

func (f *executorCoreFacadeFake) SearchLawContent(
	ctx context.Context,
	_ LawContentSearchIntentV1,
	budget LegalQueryStepBudget,
) (lawcontentsearch.Page, error) {
	err := f.recorder.call(ctx, InputKindLawContentSearch, budget)
	return f.payloads.lawContent, err
}

func (f *executorCoreFacadeFake) ReadLawDocument(
	ctx context.Context,
	_ LawReadIntentV1,
	budget LegalQueryStepBudget,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	err := f.recorder.call(ctx, InputKindLawRead, budget)
	return f.payloads.lawDocument, err
}

func (f *executorCoreFacadeFake) ReadLawArticle(
	ctx context.Context,
	_ LawArticleReadIntentV1,
	budget LegalQueryStepBudget,
) (model.SourcedResource[model.LawArticleFragment], error) {
	err := f.recorder.call(ctx, InputKindLawArticleRead, budget)
	return f.payloads.lawArticle, err
}

func (f *executorCoreFacadeFake) ListLawUpdates(
	ctx context.Context,
	_ LawUpdateListIntentV1,
	budget LegalQueryStepBudget,
) (lawupdatelist.Page, error) {
	err := f.recorder.call(ctx, InputKindLawUpdates, budget)
	return f.payloads.lawUpdates, err
}

type executorJudicialFacadeFake struct {
	recorder    *executorTestRecorder
	payloads    executorTestPayloads
	validateErr error
}

func (f *executorJudicialFacadeFake) Validate() error {
	if f == nil {
		return errExecutorTestTypedNilFacade
	}
	return f.validateErr
}

func (f *executorJudicialFacadeFake) SearchJudicialDecisions(
	ctx context.Context,
	_ JudicialDecisionSearchIntentV1,
	budget LegalQueryStepBudget,
) (judicialdecisionsearch.Page, error) {
	err := f.recorder.call(ctx, InputKindJudicialDecisionSearch, budget)
	return f.payloads.judicialSearch, err
}

func (f *executorJudicialFacadeFake) ReadJudicialDecision(
	ctx context.Context,
	_ JudicialDecisionReadIntentV1,
	budget LegalQueryStepBudget,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	err := f.recorder.call(ctx, InputKindJudicialDecisionRead, budget)
	return f.payloads.judicialDecision, err
}
