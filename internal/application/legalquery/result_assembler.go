package legalquery

import (
	"fmt"
	"reflect"
)

// AssembleLegalQueryNonExecutionResult は、非実行 plan を対応する公開結果へ変換する。
func AssembleLegalQueryNonExecutionResult(
	plan LegalQueryPlan,
) (LegalQueryResult, error) {
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("plan が有効ではありません: %w", err)
	}
	switch plan.Decision() {
	case PlanDecisionNeedsClarification:
		questions, err := deriveLegalQueryClarificationQuestions(plan)
		if err != nil {
			return nil, err
		}
		result, err := NewLegalQueryNeedsClarificationResult(plan, questions)
		if err != nil {
			return nil, fmt.Errorf("明確化結果を作成できません: %w", err)
		}
		return result, nil
	case PlanDecisionCapabilityUnavailable:
		result, err := NewLegalQueryCapabilityUnavailableResult(plan)
		if err != nil {
			return nil, fmt.Errorf("能力利用不可結果を作成できません: %w", err)
		}
		return result, nil
	case PlanDecisionUnsupported:
		result, err := NewLegalQueryUnsupportedResult(plan)
		if err != nil {
			return nil, fmt.Errorf("対象外結果を作成できません: %w", err)
		}
		return result, nil
	default:
		return nil, fmt.Errorf(
			"非実行 assembler は needs_clarification、capability_unavailable または unsupported plan だけを受理します",
		)
	}
}

// AssembleLegalQueryExecutionResult は、検証済み execution を状態別の公開結果へ変換する。
func AssembleLegalQueryExecutionResult(
	plan LegalQueryPlan,
	execution LegalQueryExecution,
) (LegalQueryResult, error) {
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("plan が有効ではありません: %w", err)
	}
	if plan.Decision() != PlanDecisionSingle &&
		plan.Decision() != PlanDecisionHedged {
		return nil, fmt.Errorf(
			"実行 assembler は single または hedged plan だけを受理します",
		)
	}
	if err := execution.Validate(); err != nil {
		return nil, fmt.Errorf("execution が有効ではありません: %w", err)
	}
	attempts := execution.Attempts()
	if err := validateExecutionAttemptsAgainstPlan(plan, attempts); err != nil {
		return nil, err
	}
	return assembleLegalQueryExecutionStatus(plan, attempts)
}

func assembleLegalQueryExecutionStatus(
	plan LegalQueryPlan,
	attempts []LegalQueryAttempt,
) (LegalQueryResult, error) {
	completed, empty, failed, err := countLegalQueryAttemptOutcomes(attempts)
	if err != nil {
		return nil, err
	}
	switch {
	case failed > 0:
		result, resultErr := NewLegalQueryPartialResult(plan, attempts)
		if resultErr != nil {
			return nil, fmt.Errorf("部分結果を作成できません: %w", resultErr)
		}
		return result, nil
	case completed > 0:
		result, resultErr := NewLegalQueryCompletedResult(plan, attempts)
		if resultErr != nil {
			return nil, fmt.Errorf("完了結果を作成できません: %w", resultErr)
		}
		return result, nil
	case empty == len(attempts):
		result, resultErr := NewLegalQueryEmptyResult(plan, attempts)
		if resultErr != nil {
			return nil, fmt.Errorf("空結果を作成できません: %w", resultErr)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("attempts から公開結果の状態を決定できません")
	}
}

func countLegalQueryAttemptOutcomes(
	attempts []LegalQueryAttempt,
) (completed int, empty int, failed int, err error) {
	for index, attempt := range attempts {
		switch attempt.Outcome() {
		case LegalQueryAttemptOutcomeCompleted:
			completed++
		case LegalQueryAttemptOutcomeEmpty:
			empty++
		case LegalQueryAttemptOutcomeFailed:
			failed++
		default:
			return 0, 0, 0, fmt.Errorf(
				"attempts[%d].outcome が定義されていません",
				index,
			)
		}
	}
	return completed, empty, failed, nil
}

func validateExecutionAttemptsAgainstPlan(
	plan LegalQueryPlan,
	attempts []LegalQueryAttempt,
) error {
	interpretations, err := NewLegalQueryInterpretations(plan)
	if err != nil {
		return fmt.Errorf("plan の interpretations を作成できません: %w", err)
	}
	if err := validateAttemptPlanOrder(interpretations, attempts); err != nil {
		return fmt.Errorf("execution が指定された plan と一致しません: %w", err)
	}
	plannedSteps, err := plannedExecutionSteps(plan)
	if err != nil {
		return err
	}
	for index, attempt := range attempts {
		key := executionStepKey{
			interpretationID: attempt.InterpretationID(),
			stepID:           attempt.StepID(),
		}
		step, exists := plannedSteps[key]
		if !exists {
			return fmt.Errorf("attempts[%d] が指定された plan にありません", index)
		}
		header, err := legalQueryAttemptHeaderFrom(attempt)
		if err != nil {
			return fmt.Errorf("attempts[%d] が有効ではありません: %w", index, err)
		}
		if header.inputKind != step.InputKind() ||
			!reflect.DeepEqual(header.logicalInput, step.LogicalInput()) {
			return fmt.Errorf(
				"attempts[%d] の logical input が指定された plan と一致しません",
				index,
			)
		}
	}
	return nil
}

type executionStepKey struct {
	interpretationID string
	stepID           string
}

func plannedExecutionSteps(
	plan LegalQueryPlan,
) (map[executionStepKey]LegalQueryCandidateStep, error) {
	candidates := make(map[string]LegalQueryCandidate, len(plan.rankedCandidates))
	for _, candidate := range plan.rankedCandidates {
		candidates[candidate.CandidateID()] = candidate
	}
	steps := make(map[executionStepKey]LegalQueryCandidateStep)
	for index, selection := range plan.selected {
		candidate, exists := candidates[selection.CandidateID()]
		if !exists {
			return nil, fmt.Errorf("selection が参照する candidate がありません")
		}
		for _, step := range candidate.Steps() {
			steps[executionStepKey{
				interpretationID: interpretationIDForIndex(index),
				stepID:           step.StepID(),
			}] = step
		}
	}
	return steps, nil
}

func legalQueryAttemptHeaderFrom(
	attempt LegalQueryAttempt,
) (legalQueryAttemptHeader, error) {
	switch typed := attempt.(type) {
	case LegalQueryLawSearchAttempt:
		return typed.data.header, nil
	case LegalQueryLawContentSearchAttempt:
		return typed.data.header, nil
	case LegalQueryLawDocumentAttempt:
		return typed.data.header, nil
	case LegalQueryLawArticleAttempt:
		return typed.data.header, nil
	case LegalQueryLawUpdatesAttempt:
		return typed.data.header, nil
	case LegalQueryJudicialSearchAttempt:
		return typed.data.header, nil
	case LegalQueryJudicialDecisionAttempt:
		return typed.data.header, nil
	case LegalQueryFailedAttempt:
		return typed.header, nil
	default:
		return legalQueryAttemptHeader{}, fmt.Errorf("attempt の concrete variant が定義されていません")
	}
}

func deriveLegalQueryClarificationQuestions(
	plan LegalQueryPlan,
) ([]LegalQueryQuestion, error) {
	if plan.Decision() != PlanDecisionNeedsClarification {
		return nil, fmt.Errorf("明確化質問には needs_clarification plan が必要です")
	}
	pool, err := legalQueryClarificationCandidatePool(plan)
	if err != nil {
		return nil, err
	}
	if len(pool) == 0 {
		return []LegalQueryQuestion{
			LegalQueryQuestionTask,
			LegalQueryQuestionResource,
		}, nil
	}
	tasks, resources := collectClarificationTargets(pool)
	questions := make([]LegalQueryQuestion, 0, 2)
	if len(tasks) >= 2 {
		questions = append(questions, LegalQueryQuestionTask)
	}
	if len(resources) >= 2 {
		questions = append(questions, LegalQueryQuestionResource)
	}
	if len(questions) == 2 {
		return questions, nil
	}
	if onlyLawResources(resources) {
		return append(questions, LegalQueryQuestionLaw), nil
	}
	if onlyJudicialDecisionResource(resources) {
		return append(questions, LegalQueryQuestionJudicialDecision), nil
	}
	return questions, nil
}

func legalQueryClarificationCandidatePool(
	plan LegalQueryPlan,
) ([]LegalQueryCandidate, error) {
	ranked := plan.RankedCandidates()
	selected := plan.Selected()
	if len(selected) == 0 {
		if len(ranked) > MaxSelectedCandidates {
			ranked = ranked[:MaxSelectedCandidates]
		}
		return ranked, nil
	}
	byID := make(map[string]LegalQueryCandidate, len(ranked))
	for _, candidate := range ranked {
		byID[candidate.CandidateID()] = candidate
	}
	pool := make([]LegalQueryCandidate, 0, len(selected))
	for _, selection := range selected {
		candidate, exists := byID[selection.CandidateID()]
		if !exists {
			return nil, fmt.Errorf("selection が参照する candidate がありません")
		}
		pool = append(pool, candidate)
	}
	return pool, nil
}

func collectClarificationTargets(
	pool []LegalQueryCandidate,
) (map[Task]struct{}, map[Resource]struct{}) {
	tasks := make(map[Task]struct{})
	resources := make(map[Resource]struct{})
	for _, candidate := range pool {
		for _, step := range candidate.Steps() {
			tasks[step.Task()] = struct{}{}
			resources[step.Resource()] = struct{}{}
		}
	}
	return tasks, resources
}

func onlyLawResources(resources map[Resource]struct{}) bool {
	if len(resources) == 0 {
		return false
	}
	for resource := range resources {
		if resource != ResourceLaw && resource != ResourceLawProvision {
			return false
		}
	}
	return true
}

func onlyJudicialDecisionResource(resources map[Resource]struct{}) bool {
	if len(resources) != 1 {
		return false
	}
	_, exists := resources[ResourceJudicialDecision]
	return exists
}
