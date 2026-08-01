package defaultprofile

import (
	"context"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type resolvedExecutionAction struct {
	meaningID    string
	stepOrdinal  int
	releaseOrder int
	candidateID  string
	step         legalquery.LegalQueryCandidateStep
	budget       legalquery.LegalQueryStepBudget
	outcome      legalquerycorpus.ExecutionOutcome
}

type resolvedExecutionFixture struct {
	actions                 []resolvedExecutionAction
	actionByStepID          map[string]resolvedExecutionAction
	meaningByInterpretation map[string]string
}

type observedExecutionAttempt struct {
	meaningID          string
	stepOrdinal        int
	stepID             string
	outcome            legalquery.LegalQueryAttemptOutcome
	publishedItemCount int
	hasMore            bool
	errorCode          model.ErrorCode
}

// EvaluateExecution は、corpus の実行 fixture を製品 executor と fake facade で評価する。
func (e *Evaluator) EvaluateExecution(
	ctx context.Context,
	corpus legalquerycorpus.Corpus,
) (legalqueryeval.ExecutionReport, error) {
	if e == nil {
		return legalqueryeval.ExecutionReport{},
			fmt.Errorf("default profile evaluator は nil にできません")
	}
	if ctx == nil {
		return legalqueryeval.ExecutionReport{},
			fmt.Errorf("context は nil にできません")
	}
	if err := corpus.Validate(); err != nil {
		return legalqueryeval.ExecutionReport{},
			fmt.Errorf("corpus が有効ではありません: %w", err)
	}
	development := make(map[string]legalquerycorpus.SemanticCase)
	for _, semanticCase := range corpus.Development() {
		development[semanticCase.CaseID()] = semanticCase
	}
	evaluations := make(
		[]legalqueryeval.ExecutionCaseEvaluation,
		0,
		len(corpus.Execution()),
	)
	for _, executionCase := range corpus.Execution() {
		if err := ctx.Err(); err != nil {
			return legalqueryeval.ExecutionReport{},
				fmt.Errorf("execution 評価を中断しました: %w", err)
		}
		semanticCase, exists := development[executionCase.SemanticCaseID()]
		if !exists {
			return legalqueryeval.ExecutionReport{},
				fmt.Errorf(
					"execution case %q の semantic case がありません",
					executionCase.CaseID(),
				)
		}
		evaluation, err := e.evaluateExecutionCase(
			ctx,
			semanticCase,
			executionCase,
		)
		if err != nil {
			return legalqueryeval.ExecutionReport{},
				fmt.Errorf(
					"execution case %q の評価に失敗しました: %w",
					executionCase.CaseID(),
					err,
				)
		}
		evaluations = append(evaluations, evaluation)
	}
	return legalqueryeval.NewExecutionReport(evaluations)
}

func (e *Evaluator) evaluateExecutionCase(
	ctx context.Context,
	semanticCase legalquerycorpus.SemanticCase,
	executionCase legalquerycorpus.ExecutionCase,
) (legalqueryeval.ExecutionCaseEvaluation, error) {
	semanticEvaluation, plan, hasPlan, err := e.EvaluateWithPlan(
		ctx,
		semanticCase,
	)
	if err != nil {
		return legalqueryeval.ExecutionCaseEvaluation{}, err
	}
	if !hasPlan ||
		(plan.Decision() != legalquery.PlanDecisionSingle &&
			plan.Decision() != legalquery.PlanDecisionHedged) {
		return unexecutableExecutionCaseEvaluation(executionCase)
	}
	fixture, err := resolveExecutionFixture(
		semanticCase,
		semanticEvaluation,
		plan,
		executionCase,
	)
	if err != nil {
		// SOT-ENG-024: 候補 plan と固定 fixture の不一致は評価失敗として集計する。
		return unexecutableExecutionCaseEvaluation(executionCase)
	}
	facade, err := newExecutionFixtureFacade(fixture)
	if err != nil {
		return legalqueryeval.ExecutionCaseEvaluation{}, err
	}
	executor, err := legalquery.NewExecutor(facade, facade)
	if err != nil {
		return legalqueryeval.ExecutionCaseEvaluation{}, err
	}
	execution, executeErr := executor.Execute(ctx, plan)
	return compareExecutionFixture(
		executionCase,
		plan,
		fixture,
		facade,
		execution,
		executeErr,
	)
}

func unexecutableExecutionCaseEvaluation(
	executionCase legalquerycorpus.ExecutionCase,
) (legalqueryeval.ExecutionCaseEvaluation, error) {
	return legalqueryeval.NewExecutionCaseEvaluation(
		legalqueryeval.ExecutionCaseEvaluationValues{
			CaseID:                     executionCase.CaseID(),
			ExpectedMatched:            false,
			AttemptOrderMatched:        false,
			AttemptOrderViolationCount: len(executionCase.Expected().Attempts()),
		},
	)
}

func resolveExecutionFixture(
	semanticCase legalquerycorpus.SemanticCase,
	evaluation legalqueryeval.SemanticCaseEvaluation,
	plan legalquery.LegalQueryPlan,
	executionCase legalquerycorpus.ExecutionCase,
) (resolvedExecutionFixture, error) {
	expectedPlan, ok := semanticCase.Expected().(legalquerycorpus.ExpectedPlan)
	if !ok {
		return resolvedExecutionFixture{},
			fmt.Errorf("execution fixture の semantic expectation は plan が必要です")
	}
	ranked := plan.RankedCandidates()
	candidateByMeaning := make(map[string]legalquery.LegalQueryCandidate)
	for _, meaning := range evaluation.Meanings() {
		rank := meaning.MatchedCandidateRank()
		if rank < 1 || rank > len(ranked) {
			return resolvedExecutionFixture{},
				fmt.Errorf("meaning %q に一致する候補がありません", meaning.MeaningID())
		}
		candidateByMeaning[meaning.MeaningID()] = ranked[rank-1]
	}
	budgetByStepID := make(map[string]legalquery.LegalQueryStepBudget)
	for _, budget := range plan.Budget().StepBudgets() {
		budgetByStepID[budget.StepID()] = budget
	}
	actions := make(
		[]resolvedExecutionAction,
		0,
		len(executionCase.Actions()),
	)
	actionByStepID := make(map[string]resolvedExecutionAction)
	for _, action := range executionCase.Actions() {
		candidate, exists := candidateByMeaning[action.MeaningID()]
		if !exists {
			return resolvedExecutionFixture{},
				fmt.Errorf("action の meaning %q が実際の候補にありません", action.MeaningID())
		}
		steps := candidate.Steps()
		if action.StepOrdinal() < 1 || action.StepOrdinal() > len(steps) {
			return resolvedExecutionFixture{},
				fmt.Errorf("action の stepOrdinal が実際の候補を超えています")
		}
		step := steps[action.StepOrdinal()-1]
		budget, exists := budgetByStepID[step.StepID()]
		if !exists {
			return resolvedExecutionFixture{},
				fmt.Errorf("action の step に固定予算がありません")
		}
		resolved := resolvedExecutionAction{
			meaningID:    action.MeaningID(),
			stepOrdinal:  action.StepOrdinal(),
			releaseOrder: action.ReleaseOrder(),
			candidateID:  candidate.CandidateID(),
			step:         step,
			budget:       budget,
			outcome:      action.Outcome(),
		}
		actions = append(actions, resolved)
		if _, duplicate := actionByStepID[step.StepID()]; duplicate {
			return resolvedExecutionFixture{},
				fmt.Errorf("action の step を重複させられません")
		}
		actionByStepID[step.StepID()] = resolved
	}
	meaningByInterpretation := make(map[string]string)
	for index, meaningID := range expectedPlan.SelectedMeaningIDs() {
		meaningByInterpretation[fmt.Sprintf("interpretation-%d", index+1)] = meaningID
	}
	return resolvedExecutionFixture{
		actions:                 actions,
		actionByStepID:          actionByStepID,
		meaningByInterpretation: meaningByInterpretation,
	}, nil
}

func compareExecutionFixture(
	executionCase legalquerycorpus.ExecutionCase,
	plan legalquery.LegalQueryPlan,
	fixture resolvedExecutionFixture,
	facade *executionFixtureFacade,
	execution legalquery.LegalQueryExecution,
	executeErr error,
) (legalqueryeval.ExecutionCaseEvaluation, error) {
	diagnostics := facade.diagnostics()
	budgetViolations := diagnostics.budgetViolationCount +
		planBudgetViolationCount(plan)
	expectedMatched := false
	attemptOrderMatched := false
	attemptOrderViolations := 0

	switch expected := executionCase.Expected().(type) {
	case legalquerycorpus.ExecutionExpectedResult:
		if executeErr == nil {
			result, err := legalquery.AssembleLegalQueryExecutionResult(
				plan,
				execution,
			)
			if err != nil {
				return legalqueryeval.ExecutionCaseEvaluation{}, err
			}
			observed, orderMatched, orderViolations, err :=
				observeResultAttempts(result, fixture, expected.Attempts())
			if err != nil {
				return legalqueryeval.ExecutionCaseEvaluation{}, err
			}
			attemptOrderMatched = orderMatched
			attemptOrderViolations = orderViolations
			expectedMatched = result.Status() == expected.Status() &&
				publishedItemCount(observed) == expected.ReturnedItemCount() &&
				compareExpectedAttempts(expected.Attempts(), observed)
			budgetViolations += resultBudgetViolationCount(plan, observed)
		}
	case legalquerycorpus.ExecutionExpectedError:
		var allFailed legalquery.LegalQueryAllFailedError
		switch {
		case errors.As(executeErr, &allFailed):
			observed := observeFailedActions(fixture.actions)
			attemptOrderMatched = compareAttemptOrder(
				expected.Attempts(),
				observed,
			)
			if !attemptOrderMatched {
				attemptOrderViolations = attemptOrderDifferenceCount(
					expected.Attempts(),
					observed,
				)
			}
			expectedMatched =
				allFailed.ErrorResult().Code() == expected.ErrorCode() &&
					compareExpectedAttempts(expected.Attempts(), observed)
		case executeErr != nil:
			return legalqueryeval.ExecutionCaseEvaluation{},
				fmt.Errorf("全失敗を公開結果へ変換できません: %w", executeErr)
		default:
			return legalqueryeval.ExecutionCaseEvaluation{},
				fmt.Errorf("全失敗 fixture が error を返しませんでした")
		}
	default:
		return legalqueryeval.ExecutionCaseEvaluation{},
			fmt.Errorf("execution expected variant が定義されていません")
	}

	return legalqueryeval.NewExecutionCaseEvaluation(
		legalqueryeval.ExecutionCaseEvaluationValues{
			CaseID:                     executionCase.CaseID(),
			ExpectedMatched:            expectedMatched,
			AttemptOrderMatched:        attemptOrderMatched,
			WrongResourceCallCount:     diagnostics.wrongResourceCallCount,
			BudgetViolationCount:       budgetViolations,
			AttemptOrderViolationCount: attemptOrderViolations,
			ImplicitFirstReadCount:     diagnostics.implicitFirstReadCount,
			EmptyReclassificationCount: diagnostics.emptyReclassificationCount,
		},
	)
}

func observeResultAttempts(
	result legalquery.LegalQueryResult,
	fixture resolvedExecutionFixture,
	expected []legalquerycorpus.ExpectedAttempt,
) ([]observedExecutionAttempt, bool, int, error) {
	attempts := result.Attempts()
	observed := make([]observedExecutionAttempt, 0, len(attempts))
	for _, attempt := range attempts {
		action, exists := fixture.actionByStepID[attempt.StepID()]
		if !exists {
			return nil, false, len(attempts),
				fmt.Errorf("公開 attempt が plan にない step を参照しています")
		}
		if fixture.meaningByInterpretation[attempt.InterpretationID()] !=
			action.meaningID {
			return nil, false, len(attempts),
				fmt.Errorf("attempt の interpretation と meaning が一致しません")
		}
		value, err := observeAttempt(attempt, action)
		if err != nil {
			return nil, false, len(attempts), err
		}
		observed = append(observed, value)
	}
	orderMatched := compareAttemptOrder(expected, observed)
	return observed,
		orderMatched,
		attemptOrderDifferenceCount(expected, observed),
		nil
}

func observeAttempt(
	attempt legalquery.LegalQueryAttempt,
	action resolvedExecutionAction,
) (observedExecutionAttempt, error) {
	value := observedExecutionAttempt{
		meaningID:   action.meaningID,
		stepOrdinal: action.stepOrdinal,
		stepID:      action.step.StepID(),
		outcome:     attempt.Outcome(),
	}
	switch typed := attempt.(type) {
	case legalquery.LegalQueryLawSearchAttempt:
		setObservedPage(&value, typed.Page())
	case legalquery.LegalQueryLawContentSearchAttempt:
		setObservedPage(&value, typed.Page())
	case legalquery.LegalQueryLawUpdatesAttempt:
		setObservedPage(&value, typed.Page())
	case legalquery.LegalQueryJudicialSearchAttempt:
		setObservedPage(&value, typed.Page())
	case legalquery.LegalQueryLawDocumentAttempt,
		legalquery.LegalQueryLawArticleAttempt,
		legalquery.LegalQueryJudicialDecisionAttempt:
		value.publishedItemCount = 1
	case legalquery.LegalQueryFailedAttempt:
		value.errorCode = typed.Error().Code()
	default:
		return observedExecutionAttempt{},
			fmt.Errorf("公開 attempt variant が定義されていません")
	}
	return value, nil
}

func setObservedPage(
	value *observedExecutionAttempt,
	page legalquery.LegalQueryPagePreview,
) {
	value.publishedItemCount = page.ReturnedCount()
	value.hasMore, _ = page.HasMore()
}

func observeFailedActions(
	actions []resolvedExecutionAction,
) []observedExecutionAttempt {
	observed := make([]observedExecutionAttempt, 0, len(actions))
	for _, action := range actions {
		value := observedExecutionAttempt{
			meaningID:   action.meaningID,
			stepOrdinal: action.stepOrdinal,
			stepID:      action.step.StepID(),
			outcome:     legalquery.LegalQueryAttemptOutcomeFailed,
		}
		switch outcome := action.outcome.(type) {
		case legalquerycorpus.TimeoutOutcome:
			value.errorCode = model.ErrorCodeSourceTimeout
		case legalquerycorpus.FailureOutcome:
			value.errorCode = outcome.ErrorCode()
		}
		observed = append(observed, value)
	}
	return observed
}

func compareExpectedAttempts(
	expected []legalquerycorpus.ExpectedAttempt,
	observed []observedExecutionAttempt,
) bool {
	if len(expected) != len(observed) {
		return false
	}
	for index, expectedAttempt := range expected {
		actual := observed[index]
		if expectedAttempt.MeaningID() != actual.meaningID ||
			expectedAttempt.StepOrdinal() != actual.stepOrdinal ||
			expectedAttempt.Outcome() != actual.outcome {
			return false
		}
		switch typed := expectedAttempt.(type) {
		case legalquerycorpus.ExpectedCompletedReadAttempt:
			if typed.PublishedItemCount() != actual.publishedItemCount {
				return false
			}
		case legalquerycorpus.ExpectedCompletedCollectionAttempt:
			if typed.PublishedItemCount() != actual.publishedItemCount ||
				typed.HasMore() != actual.hasMore {
				return false
			}
		case legalquerycorpus.ExpectedEmptyAttempt:
			if actual.publishedItemCount != 0 || actual.hasMore {
				return false
			}
		case legalquerycorpus.ExpectedFailedAttempt:
			if typed.ErrorCode() != actual.errorCode {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func compareAttemptOrder(
	expected []legalquerycorpus.ExpectedAttempt,
	observed []observedExecutionAttempt,
) bool {
	if len(expected) != len(observed) {
		return false
	}
	for index, value := range expected {
		if value.MeaningID() != observed[index].meaningID ||
			value.StepOrdinal() != observed[index].stepOrdinal {
			return false
		}
	}
	return true
}

func attemptOrderDifferenceCount(
	expected []legalquerycorpus.ExpectedAttempt,
	observed []observedExecutionAttempt,
) int {
	differences := 0
	for index := 0; index < max(len(expected), len(observed)); index++ {
		if index >= len(expected) || index >= len(observed) ||
			expected[index].MeaningID() != observed[index].meaningID ||
			expected[index].StepOrdinal() != observed[index].stepOrdinal {
			differences++
		}
	}
	return differences
}

func publishedItemCount(attempts []observedExecutionAttempt) int {
	total := 0
	for _, attempt := range attempts {
		total += attempt.publishedItemCount
	}
	return total
}

func planBudgetViolationCount(plan legalquery.LegalQueryPlan) int {
	budget := plan.Budget()
	violations := 0
	if len(plan.RankedCandidates()) > legalquery.MaxRankedCandidates {
		violations++
	}
	if len(plan.Selected()) > legalquery.MaxSelectedCandidates {
		violations++
	}
	if len(budget.StepBudgets()) > legalquery.MaxCapabilityCalls {
		violations++
	}
	if !budget.FirstPageOnly() ||
		budget.MaxReturnedItems() != legalquery.MaxReturnedItems ||
		budget.MaxItemsPerCollectionStep() !=
			legalquery.MaxItemsPerCollectionStep {
		violations++
	}
	return violations
}

func resultBudgetViolationCount(
	plan legalquery.LegalQueryPlan,
	attempts []observedExecutionAttempt,
) int {
	violations := 0
	if publishedItemCount(attempts) > legalquery.MaxReturnedItems {
		violations++
	}
	budgets := make(map[string]legalquery.LegalQueryStepBudget)
	for _, budget := range plan.Budget().StepBudgets() {
		budgets[budget.StepID()] = budget
	}
	for _, attempt := range attempts {
		budget, exists := budgets[attempt.stepID]
		if !exists {
			violations++
			continue
		}
		if limit, hasLimit := budget.EffectiveLimit(); hasLimit &&
			attempt.publishedItemCount > limit {
			violations++
		}
	}
	return violations
}
