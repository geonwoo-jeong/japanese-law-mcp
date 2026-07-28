package legalquery

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Executor は、確定済み plan を capability facade へ接続する。
type Executor struct {
	core        CoreCapabilityFacade
	judicial    JudicialCasesCapabilityFacade
	initialized bool
}

// NewExecutor は、必須の core facade と任意の judicial facade を検証する。
func NewExecutor(
	core CoreCapabilityFacade,
	judicial JudicialCasesCapabilityFacade,
) (Executor, error) {
	executor := Executor{
		core:        core,
		judicial:    judicial,
		initialized: true,
	}
	if err := executor.Validate(); err != nil {
		return Executor{}, err
	}
	return executor, nil
}

// Validate は、facade の必須性、typed nil および起動時契約を確認する。
func (e Executor) Validate() error {
	if !e.initialized {
		return fmt.Errorf("Executor は NewExecutor で作成しなければなりません")
	}
	if isNilInterfaceValue(e.core) {
		return fmt.Errorf("core capability facade は必須です")
	}
	if err := e.core.Validate(); err != nil {
		return fmt.Errorf("core capability facade が有効ではありません: %w", err)
	}
	if e.judicial == nil {
		return nil
	}
	if isNilInterfaceValue(e.judicial) {
		return fmt.Errorf("judicial capability facade に typed nil を指定できません")
	}
	if err := e.judicial.Validate(); err != nil {
		return fmt.Errorf(
			"judicial capability facade が有効ではありません: %w",
			err,
		)
	}
	return nil
}

// Execute は、single または hedged plan を固定予算と計画順のまま実行する。
func (e Executor) Execute(
	ctx context.Context,
	plan LegalQueryPlan,
) (LegalQueryExecution, error) {
	if err := e.validateExecutionRequest(ctx, plan); err != nil {
		return LegalQueryExecution{}, err
	}
	candidates, err := e.prepareCandidates(plan)
	if err != nil {
		return LegalQueryExecution{}, err
	}
	results, err := e.executeCandidates(ctx, candidates)
	if err != nil {
		return LegalQueryExecution{}, err
	}
	return newExecutorExecution(plan, results)
}

func (e Executor) validateExecutionRequest(
	ctx context.Context,
	plan LegalQueryPlan,
) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if isNilInterfaceValue(ctx) {
		return fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("plan が有効ではありません: %w", err)
	}
	switch plan.Decision() {
	case PlanDecisionSingle, PlanDecisionHedged:
		return nil
	default:
		return fmt.Errorf(
			"decision=%s は Executor で実行できません",
			plan.Decision(),
		)
	}
}

func (e Executor) executeCandidates(
	ctx context.Context,
	candidates []executorCandidate,
) ([]executorCandidateResult, error) {
	executionContext, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]executorCandidateResult, len(candidates))
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(candidates))
	for index := range candidates {
		index := index
		go func() {
			defer waitGroup.Done()
			results[index] = e.executeCandidate(
				executionContext,
				ctx,
				cancel,
				candidates[index],
			)
		}()
	}
	waitGroup.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, result := range results {
		if result.err != nil {
			return nil, result.err
		}
	}
	return results, nil
}

func newExecutorExecution(
	plan LegalQueryPlan,
	results []executorCandidateResult,
) (LegalQueryExecution, error) {
	attempts := make([]LegalQueryAttempt, 0, MaxCapabilityCalls)
	for _, result := range results {
		attempts = append(attempts, result.attempts...)
	}
	if executorAllAttemptsFailed(attempts) {
		first, err := executorFailedAttemptError(attempts[0])
		if err != nil {
			return LegalQueryExecution{}, err
		}
		allFailed, err := newLegalQueryAllFailedError(first)
		if err != nil {
			return LegalQueryExecution{}, err
		}
		return LegalQueryExecution{}, allFailed
	}
	execution, err := NewLegalQueryExecution(
		LegalQueryExecutionValues{
			Plan:     plan,
			Attempts: attempts,
		},
	)
	if err != nil {
		return LegalQueryExecution{}, fmt.Errorf(
			"execution artifact を作成できません: %w",
			err,
		)
	}
	return execution, nil
}

type executorCandidate struct {
	interpretationID string
	steps            []executorStep
}

type executorStep struct {
	step   LegalQueryCandidateStep
	budget LegalQueryStepBudget
}

type executorCandidateResult struct {
	attempts []LegalQueryAttempt
	err      error
}

func (e Executor) prepareCandidates(
	plan LegalQueryPlan,
) ([]executorCandidate, error) {
	selected := plan.Selected()
	if err := validateExecutorSelectionShape(plan.Decision(), selected); err != nil {
		return nil, err
	}
	references := executorCandidateReferences(plan.RankedCandidates())
	stepBudgets := plan.Budget().StepBudgets()
	if len(stepBudgets) > MaxCapabilityCalls {
		return nil, fmt.Errorf("step budget は四件以下でなければなりません")
	}
	prepared := make([]executorCandidate, 0, len(selected))
	budgetIndex := 0
	for interpretationIndex, selection := range selected {
		candidate, exists := references[selection.CandidateID()]
		if !exists {
			return nil, fmt.Errorf(
				"selected candidate=%s が rankedCandidates にありません",
				selection.CandidateID(),
			)
		}
		preparedCandidate, nextBudgetIndex, err := e.prepareCandidate(
			interpretationIndex,
			selection,
			candidate,
			stepBudgets,
			budgetIndex,
		)
		if err != nil {
			return nil, err
		}
		budgetIndex = nextBudgetIndex
		prepared = append(prepared, preparedCandidate)
	}
	if err := validateExecutorBudgetCount(budgetIndex, len(stepBudgets)); err != nil {
		return nil, err
	}
	return prepared, nil
}

func executorCandidateReferences(
	ranked []LegalQueryCandidate,
) map[string]LegalQueryCandidate {
	references := make(map[string]LegalQueryCandidate, len(ranked))
	for _, candidate := range ranked {
		references[candidate.CandidateID()] = candidate
	}
	return references
}

func validateExecutorSelectionShape(
	decision PlanDecision,
	selected []LegalQueryPlanSelection,
) error {
	if len(selected) < 1 || len(selected) > MaxSelectedCandidates {
		return fmt.Errorf("selected は一件以上二件以下でなければなりません")
	}
	if decision == PlanDecisionSingle && len(selected) != 1 {
		return fmt.Errorf("single plan の selected は一件でなければなりません")
	}
	if decision == PlanDecisionHedged && len(selected) != 2 {
		return fmt.Errorf("hedged plan の selected は二件でなければなりません")
	}
	return nil
}

func (e Executor) prepareCandidate(
	interpretationIndex int,
	selection LegalQueryPlanSelection,
	candidate LegalQueryCandidate,
	stepBudgets []LegalQueryStepBudget,
	budgetIndex int,
) (executorCandidate, int, error) {
	if executorRequiresJudicialFacade(selection, candidate) &&
		e.judicial == nil {
		return executorCandidate{}, budgetIndex, fmt.Errorf(
			"judicial-cases を実行するには judicial capability facade が必要です",
		)
	}
	steps := candidate.Steps()
	executionSteps := make([]executorStep, 0, len(steps))
	for _, step := range steps {
		if budgetIndex >= len(stepBudgets) {
			return executorCandidate{}, budgetIndex, fmt.Errorf(
				"step に対応する固定予算がありません",
			)
		}
		budget := stepBudgets[budgetIndex]
		budgetIndex++
		if err := validateExecutorStepBudget(candidate, step, budget); err != nil {
			return executorCandidate{}, budgetIndex, err
		}
		executionSteps = append(executionSteps, executorStep{
			step: step, budget: budget,
		})
	}
	return executorCandidate{
		interpretationID: fmt.Sprintf(
			"interpretation-%d",
			interpretationIndex+1,
		),
		steps: executionSteps,
	}, budgetIndex, nil
}

func validateExecutorStepBudget(
	candidate LegalQueryCandidate,
	step LegalQueryCandidateStep,
	budget LegalQueryStepBudget,
) error {
	if err := budget.Validate(); err != nil {
		return fmt.Errorf("step budget が有効ではありません: %w", err)
	}
	if budget.CandidateID() != candidate.CandidateID() ||
		budget.StepID() != step.StepID() {
		return fmt.Errorf(
			"step budget が candidate と step の計画順に一致しません",
		)
	}
	return nil
}

func validateExecutorBudgetCount(used int, available int) error {
	if used != available {
		return fmt.Errorf("selected step に対応しない固定予算があります")
	}
	if used < 1 || used > MaxCapabilityCalls {
		return fmt.Errorf("実行 step は一件以上四件以下でなければなりません")
	}
	return nil
}

func executorRequiresJudicialFacade(
	selection LegalQueryPlanSelection,
	candidate LegalQueryCandidate,
) bool {
	for _, packID := range selection.RequiredPacks() {
		if packID == "judicial-cases" {
			return true
		}
	}
	for _, step := range candidate.Steps() {
		if step.InputKind() == InputKindJudicialDecisionSearch ||
			step.InputKind() == InputKindJudicialDecisionRead {
			return true
		}
	}
	return false
}

func (e Executor) executeCandidate(
	executionContext context.Context,
	rootContext context.Context,
	cancel context.CancelFunc,
	candidate executorCandidate,
) executorCandidateResult {
	result := executorCandidateResult{
		attempts: make([]LegalQueryAttempt, 0, len(candidate.steps)),
	}
	for _, preparedStep := range candidate.steps {
		if executionContext.Err() != nil {
			return result
		}
		attempt, err := e.executeStep(
			executionContext,
			candidate.interpretationID,
			preparedStep.step,
			preparedStep.budget,
		)
		if err == nil {
			result.attempts = append(result.attempts, attempt)
			continue
		}
		if rootContext.Err() != nil {
			return result
		}
		if executionContext.Err() != nil &&
			errors.Is(err, context.Canceled) {
			return result
		}
		result.err = err
		cancel()
		return result
	}
	return result
}

func executorAllAttemptsFailed(attempts []LegalQueryAttempt) bool {
	if len(attempts) == 0 {
		return false
	}
	for _, attempt := range attempts {
		if attempt.Outcome() != LegalQueryAttemptOutcomeFailed {
			return false
		}
	}
	return true
}

func executorFailedAttemptError(
	attempt LegalQueryAttempt,
) (model.ErrorResult, error) {
	switch failed := attempt.(type) {
	case LegalQueryFailedAttempt:
		return failed.Error(), nil
	case *LegalQueryFailedAttempt:
		if failed != nil {
			return failed.Error(), nil
		}
	}
	return model.ErrorResult{}, fmt.Errorf(
		"failed outcome に LegalQueryFailedAttempt がありません",
	)
}
