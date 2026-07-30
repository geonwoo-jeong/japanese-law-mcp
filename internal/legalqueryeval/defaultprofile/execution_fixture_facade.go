package defaultprofile

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type executionFixtureDiagnostics struct {
	wrongResourceCallCount     int
	budgetViolationCount       int
	implicitFirstReadCount     int
	emptyReclassificationCount int
}

type executionFixtureFacade struct {
	fixture     resolvedExecutionFixture
	payloads    executionFixturePayloads
	mutex       sync.Mutex
	release     []chan struct{}
	called      map[string]struct{}
	diagnostic  executionFixtureDiagnostics
	sawEmpty    bool
	initialized bool
}

func newExecutionFixtureFacade(
	fixture resolvedExecutionFixture,
) (*executionFixtureFacade, error) {
	if len(fixture.actions) == 0 {
		return nil, fmt.Errorf("execution fixture action は一件以上必要です")
	}
	payloads, err := newExecutionFixturePayloads()
	if err != nil {
		return nil, err
	}
	release, err := newExecutionReleaseGates(fixture.actions)
	if err != nil {
		return nil, err
	}
	facade := &executionFixtureFacade{
		fixture:     fixture,
		payloads:    payloads,
		release:     release,
		called:      make(map[string]struct{}, len(fixture.actions)),
		initialized: true,
	}
	return facade, nil
}

func (f *executionFixtureFacade) Validate() error {
	if f == nil ||
		!f.initialized ||
		len(f.release) != len(f.fixture.actions)+1 {
		return fmt.Errorf("execution fixture facade が初期化されていません")
	}
	return nil
}

func (f *executionFixtureFacade) SearchLaws(
	ctx context.Context,
	input legalquery.LawSearchIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (lawsearch.Page, error) {
	action, err := f.takeAction(
		ctx,
		legalquery.InputKindLawSearch,
		input,
		budget,
	)
	if err != nil {
		return lawsearch.Page{}, err
	}
	if callErr := fixtureOutcomeError(action); callErr != nil {
		return lawsearch.Page{}, callErr
	}
	count, err := collectionSourceItemCount(action.outcome)
	if err != nil {
		return lawsearch.Page{}, err
	}
	return f.payloads.lawSearchPage(action.step.StepID(), count)
}

func (f *executionFixtureFacade) SearchLawContent(
	ctx context.Context,
	input legalquery.LawContentSearchIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (lawcontentsearch.Page, error) {
	action, err := f.takeAction(
		ctx,
		legalquery.InputKindLawContentSearch,
		input,
		budget,
	)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}
	if callErr := fixtureOutcomeError(action); callErr != nil {
		return lawcontentsearch.Page{}, callErr
	}
	count, err := collectionSourceItemCount(action.outcome)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}
	return f.payloads.lawContentPage(action.step.StepID(), count)
}

func (f *executionFixtureFacade) ReadLawDocument(
	ctx context.Context,
	input legalquery.LawReadIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	action, err := f.takeAction(
		ctx,
		legalquery.InputKindLawRead,
		input,
		budget,
	)
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	if callErr := fixtureOutcomeError(action); callErr != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, callErr
	}
	if action.outcome.Kind() != legalquerycorpus.ExecutionOutcomeKindReadSuccess {
		return model.SourcedResource[model.LawDocumentRepresentation]{},
			fmt.Errorf("law read action の outcome が read_success ではありません")
	}
	return f.payloads.lawDocument(input)
}

func (f *executionFixtureFacade) ReadLawArticle(
	ctx context.Context,
	input legalquery.LawArticleReadIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (model.SourcedResource[model.LawArticleFragment], error) {
	action, err := f.takeAction(
		ctx,
		legalquery.InputKindLawArticleRead,
		input,
		budget,
	)
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	if callErr := fixtureOutcomeError(action); callErr != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, callErr
	}
	if action.outcome.Kind() != legalquerycorpus.ExecutionOutcomeKindReadSuccess {
		return model.SourcedResource[model.LawArticleFragment]{},
			fmt.Errorf("law article read action の outcome が read_success ではありません")
	}
	return f.payloads.lawArticle(input)
}

func (f *executionFixtureFacade) ListLawUpdates(
	ctx context.Context,
	input legalquery.LawUpdateListIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (lawupdatelist.Page, error) {
	action, err := f.takeAction(
		ctx,
		legalquery.InputKindLawUpdates,
		input,
		budget,
	)
	if err != nil {
		return lawupdatelist.Page{}, err
	}
	if callErr := fixtureOutcomeError(action); callErr != nil {
		return lawupdatelist.Page{}, callErr
	}
	count, err := collectionSourceItemCount(action.outcome)
	if err != nil {
		return lawupdatelist.Page{}, err
	}
	return f.payloads.lawUpdatePage(input, count)
}

func (f *executionFixtureFacade) SearchJudicialDecisions(
	ctx context.Context,
	input legalquery.JudicialDecisionSearchIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (judicialdecisionsearch.Page, error) {
	action, err := f.takeAction(
		ctx,
		legalquery.InputKindJudicialDecisionSearch,
		input,
		budget,
	)
	if err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	if callErr := fixtureOutcomeError(action); callErr != nil {
		return judicialdecisionsearch.Page{}, callErr
	}
	count, err := collectionSourceItemCount(action.outcome)
	if err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	return f.payloads.judicialSearchPage(action.step.StepID(), count)
}

func (f *executionFixtureFacade) ReadJudicialDecision(
	ctx context.Context,
	input legalquery.JudicialDecisionReadIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	action, err := f.takeAction(
		ctx,
		legalquery.InputKindJudicialDecisionRead,
		input,
		budget,
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	if callErr := fixtureOutcomeError(action); callErr != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, callErr
	}
	if action.outcome.Kind() != legalquerycorpus.ExecutionOutcomeKindReadSuccess {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf("judicial read action の outcome が read_success ではありません")
	}
	return f.payloads.judicialDecision(input)
}

func (f *executionFixtureFacade) takeAction(
	ctx context.Context,
	inputKind legalquery.LogicalInputKind,
	input legalquery.LogicalInput,
	budget legalquery.LegalQueryStepBudget,
) (resolvedExecutionAction, error) {
	if ctx == nil {
		return resolvedExecutionAction{}, fmt.Errorf("context は nil にできません")
	}
	if err := ctx.Err(); err != nil {
		return resolvedExecutionAction{}, err
	}
	f.mutex.Lock()

	action, exists := f.fixture.actionByStepID[budget.StepID()]
	if !exists {
		f.recordUnplannedCall(inputKind)
		f.mutex.Unlock()
		return resolvedExecutionAction{},
			fmt.Errorf("plan にない resource が呼び出されました")
	}
	if action.step.InputKind() != inputKind ||
		!reflect.DeepEqual(action.step.LogicalInput(), input) {
		f.recordUnplannedCall(inputKind)
		f.mutex.Unlock()
		return resolvedExecutionAction{},
			fmt.Errorf("plan と異なる resource または logical input が呼び出されました")
	}
	if _, duplicate := f.called[action.step.StepID()]; duplicate {
		f.diagnostic.wrongResourceCallCount++
		f.mutex.Unlock()
		return resolvedExecutionAction{},
			fmt.Errorf("同じ execution step が二回呼び出されました")
	}
	if !reflect.DeepEqual(action.budget, budget) {
		f.diagnostic.budgetViolationCount++
	}
	f.called[action.step.StepID()] = struct{}{}
	waitForPrevious := f.release[action.releaseOrder-1]
	releaseCurrent := f.release[action.releaseOrder]
	f.mutex.Unlock()

	select {
	case <-waitForPrevious:
	case <-ctx.Done():
		return resolvedExecutionAction{}, ctx.Err()
	}
	if isEmptyCollectionOutcome(action.outcome) {
		f.mutex.Lock()
		f.sawEmpty = true
		f.mutex.Unlock()
	}
	close(releaseCurrent)
	return action, nil
}

func (f *executionFixtureFacade) recordUnplannedCall(
	inputKind legalquery.LogicalInputKind,
) {
	f.diagnostic.wrongResourceCallCount++
	if inputKind == legalquery.InputKindLawRead ||
		inputKind == legalquery.InputKindLawArticleRead ||
		inputKind == legalquery.InputKindJudicialDecisionRead {
		f.diagnostic.implicitFirstReadCount++
	}
	if f.sawEmpty {
		f.diagnostic.emptyReclassificationCount++
	}
}

func (f *executionFixtureFacade) diagnostics() executionFixtureDiagnostics {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	value := f.diagnostic
	if missing := len(f.fixture.actions) - len(f.called); missing > 0 {
		value.wrongResourceCallCount += missing
	}
	return value
}

func collectionSourceItemCount(
	outcome legalquerycorpus.ExecutionOutcome,
) (int, error) {
	collection, ok := outcome.(legalquerycorpus.CollectionSuccessOutcome)
	if !ok {
		return 0, fmt.Errorf("collection action の outcome が collection_success ではありません")
	}
	return collection.SourceItemCount(), nil
}

func isEmptyCollectionOutcome(
	outcome legalquerycorpus.ExecutionOutcome,
) bool {
	collection, ok := outcome.(legalquerycorpus.CollectionSuccessOutcome)
	return ok && collection.SourceItemCount() == 0
}

func newExecutionReleaseGates(
	actions []resolvedExecutionAction,
) ([]chan struct{}, error) {
	release := make([]chan struct{}, len(actions)+1)
	for index := range release {
		release[index] = make(chan struct{})
	}
	close(release[0])
	seen := make(map[int]struct{}, len(actions))
	for _, action := range actions {
		if action.releaseOrder < 1 || action.releaseOrder > len(actions) {
			return nil, fmt.Errorf("releaseOrder が action 件数の範囲外です")
		}
		if _, exists := seen[action.releaseOrder]; exists {
			return nil, fmt.Errorf("releaseOrder を重複させられません")
		}
		seen[action.releaseOrder] = struct{}{}
	}
	return release, nil
}
