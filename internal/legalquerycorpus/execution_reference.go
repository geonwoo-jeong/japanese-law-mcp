package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

type executionReferenceStep struct {
	meaningID   string
	stepOrdinal int
	task        legalquery.Task
}

type executionReferencePlan struct {
	steps             []executionReferenceStep
	effectiveLimit    int
	hasEffectiveLimit bool
}

// validateIntegrityExecutionReferences は、失敗時に部分的な corpus を破棄する。
func validateIntegrityExecutionReferences(
	checked integrityCheckedCorpus,
) (integrityCheckedCorpus, error) {
	if err := validateExecutionReferences(
		checked.development,
		checked.execution,
	); err != nil {
		return integrityCheckedCorpus{}, err
	}
	return checked, nil
}

// validateExecutionReferences は、execution と development plan を照合する。
func validateExecutionReferences(
	development []SemanticCase,
	execution []ExecutionCase,
) error {
	developmentByID := make(map[string]SemanticCase, len(development))
	for _, semanticCase := range development {
		if err := semanticCase.Validate(); err != nil {
			return fmt.Errorf("execution 参照先の development case が有効ではありません")
		}
		developmentByID[semanticCase.CaseID()] = semanticCase
	}
	for _, executionCase := range execution {
		if err := executionCase.Validate(); err != nil {
			return fmt.Errorf("execution 参照元の execution case が有効ではありません")
		}
		semanticCase, exists := developmentByID[executionCase.SemanticCaseID()]
		if !exists {
			return fmt.Errorf(
				"execution case %q の semanticCaseId は development に存在しません",
				executionCase.CaseID(),
			)
		}
		if err := validateExecutionReference(executionCase, semanticCase); err != nil {
			return fmt.Errorf(
				"execution case %q の参照整合が有効ではありません: %w",
				executionCase.CaseID(),
				err,
			)
		}
	}
	return nil
}

func validateExecutionReference(
	executionCase ExecutionCase,
	semanticCase SemanticCase,
) error {
	plan, err := executionReferencePlanForSemanticCase(semanticCase)
	if err != nil {
		return err
	}
	return validateExecutionReferenceActions(executionCase, plan)
}

func executionReferencePlanForSemanticCase(
	semanticCase SemanticCase,
) (executionReferencePlan, error) {
	expected, ok := semanticCase.Expected().(ExpectedPlan)
	if !ok {
		return executionReferencePlan{},
			fmt.Errorf("semanticCaseId は plan を参照しなければなりません")
	}
	switch expected.Decision() {
	case legalquery.PlanDecisionSingle, legalquery.PlanDecisionHedged:
	default:
		return executionReferencePlan{},
			fmt.Errorf("参照先 plan の decision は single または hedged でなければなりません")
	}
	steps, err := flattenExecutionReferenceSteps(expected)
	if err != nil {
		return executionReferencePlan{}, err
	}
	limit, hasLimit, err := executionReferenceEffectiveLimit(
		semanticCase,
		steps,
	)
	if err != nil {
		return executionReferencePlan{}, err
	}
	return executionReferencePlan{
		steps:             steps,
		effectiveLimit:    limit,
		hasEffectiveLimit: hasLimit,
	}, nil
}

func flattenExecutionReferenceSteps(
	plan ExpectedPlan,
) ([]executionReferenceStep, error) {
	meanings := make(map[string]ExpectedMeaning, len(plan.Meanings()))
	for _, meaning := range plan.Meanings() {
		meanings[meaning.MeaningID()] = meaning
	}
	steps := make([]executionReferenceStep, 0, maximumSelectedExpectedStepCount)
	for _, meaningID := range plan.SelectedMeaningIDs() {
		meaning, exists := meanings[meaningID]
		if !exists {
			return nil, fmt.Errorf("selectedMeaningIds の参照先が meanings に存在しません")
		}
		for index, step := range meaning.Steps() {
			steps = append(steps, executionReferenceStep{
				meaningID:   meaningID,
				stepOrdinal: index + 1,
				task:        step.Task(),
			})
		}
	}
	return steps, nil
}

func executionReferenceEffectiveLimit(
	semanticCase SemanticCase,
	steps []executionReferenceStep,
) (int, bool, error) {
	request, err := productRequestFromRaw(semanticCase.Request())
	if err != nil {
		return 0, false,
			fmt.Errorf("参照先 semantic case の request を予算へ変換できません")
	}
	readSteps, collectionSteps, err := countExecutionReferenceSteps(steps)
	if err != nil {
		return 0, false, err
	}
	limit, exists, err := legalquery.EffectiveLimitForSteps(
		request.LimitPerAttempt(),
		readSteps,
		collectionSteps,
	)
	if err != nil {
		return 0, false, fmt.Errorf("参照先 plan の固定予算を導出できません")
	}
	return limit, exists, nil
}

func countExecutionReferenceSteps(
	steps []executionReferenceStep,
) (int, int, error) {
	readSteps := 0
	collectionSteps := 0
	for _, step := range steps {
		switch step.task {
		case legalquery.TaskRead:
			readSteps++
		case legalquery.TaskSearch, legalquery.TaskListUpdates:
			collectionSteps++
		default:
			return 0, 0, fmt.Errorf("参照先 plan に予算を割り当てられない task があります")
		}
	}
	return readSteps, collectionSteps, nil
}

func validateExecutionReferenceActions(
	executionCase ExecutionCase,
	plan executionReferencePlan,
) error {
	actions := executionCase.Actions()
	attempts := executionCase.Expected().Attempts()
	if len(actions) != len(plan.steps) || len(attempts) != len(plan.steps) {
		return fmt.Errorf("actions は選択した全 step を正確に一回ずつ参照してください")
	}
	for index, step := range plan.steps {
		action := actions[index]
		attempt := attempts[index]
		if action.MeaningID() != step.meaningID ||
			action.StepOrdinal() != step.stepOrdinal ||
			attempt.MeaningID() != step.meaningID ||
			attempt.StepOrdinal() != step.stepOrdinal {
			return fmt.Errorf("actions と expected attempts は参照先 plan 順でなければなりません")
		}
		if err := validateExecutionReferenceOutcome(
			step,
			action.Outcome(),
			attempt,
			plan,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateExecutionReferenceOutcome(
	step executionReferenceStep,
	outcome ExecutionOutcome,
	attempt ExpectedAttempt,
	plan executionReferencePlan,
) error {
	switch outcome.(type) {
	case CollectionSuccessOutcome:
		if step.task != legalquery.TaskSearch &&
			step.task != legalquery.TaskListUpdates {
			return fmt.Errorf("collection_success は collection step だけに使用できます")
		}
		return validateExecutionCollectionLimit(attempt, plan)
	case ReadSuccessOutcome:
		if step.task != legalquery.TaskRead {
			return fmt.Errorf("read_success は read step だけに使用できます")
		}
		if _, ok := attempt.(ExpectedCompletedReadAttempt); !ok {
			return fmt.Errorf("read_success の expected attempt が read と一致しません")
		}
	case FailureOutcome, TimeoutOutcome:
		if _, ok := attempt.(ExpectedFailedAttempt); !ok {
			return fmt.Errorf("失敗 action の expected attempt が failed と一致しません")
		}
	default:
		return fmt.Errorf("execution outcome の参照整合種別が定義されていません")
	}
	return nil
}

func validateExecutionCollectionLimit(
	attempt ExpectedAttempt,
	plan executionReferencePlan,
) error {
	if !plan.hasEffectiveLimit {
		return fmt.Errorf("collection step の effectiveLimit を導出できません")
	}
	switch typed := attempt.(type) {
	case ExpectedCompletedCollectionAttempt:
		if typed.PublishedItemCount() > plan.effectiveLimit {
			return fmt.Errorf("collection の公開件数が effectiveLimit を超えています")
		}
	case ExpectedEmptyAttempt:
		return nil
	default:
		return fmt.Errorf("collection_success の expected attempt が collection と一致しません")
	}
	return nil
}
