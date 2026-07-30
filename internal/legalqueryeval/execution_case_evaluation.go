package legalqueryeval

import "fmt"

// ExecutionCaseEvaluationValues は、一件の実行 fixture に対する観測値を保持する。
type ExecutionCaseEvaluationValues struct {
	CaseID                     string
	ExpectedMatched            bool
	AttemptOrderMatched        bool
	WrongResourceCallCount     int
	BudgetViolationCount       int
	AttemptOrderViolationCount int
	ImplicitFirstReadCount     int
	EmptyReclassificationCount int
}

// ExecutionCaseEvaluation は、一件の実行 fixture の安全な判定だけを保持する。
type ExecutionCaseEvaluation struct {
	caseID                     string
	expectedMatched            bool
	attemptOrderMatched        bool
	wrongResourceCallCount     int
	budgetViolationCount       int
	attemptOrderViolationCount int
	implicitFirstReadCount     int
	emptyReclassificationCount int
}

// NewExecutionCaseEvaluation は、実行 fixture の違反数と一致判定を検証する。
func NewExecutionCaseEvaluation(
	values ExecutionCaseEvaluationValues,
) (ExecutionCaseEvaluation, error) {
	evaluation := ExecutionCaseEvaluation{
		caseID:                     values.CaseID,
		expectedMatched:            values.ExpectedMatched,
		attemptOrderMatched:        values.AttemptOrderMatched,
		wrongResourceCallCount:     values.WrongResourceCallCount,
		budgetViolationCount:       values.BudgetViolationCount,
		attemptOrderViolationCount: values.AttemptOrderViolationCount,
		implicitFirstReadCount:     values.ImplicitFirstReadCount,
		emptyReclassificationCount: values.EmptyReclassificationCount,
	}
	if err := evaluation.Validate(); err != nil {
		return ExecutionCaseEvaluation{}, err
	}
	return evaluation, nil
}

// CaseID は、execution fixture の case ID を返す。
func (e ExecutionCaseEvaluation) CaseID() string { return e.caseID }

// ExpectedMatched は、終端と attempt 投影が期待値へ一致したか返す。
func (e ExecutionCaseEvaluation) ExpectedMatched() bool {
	return e.expectedMatched
}

// AttemptOrderMatched は、公開 attempt が plan 順を保ったか返す。
func (e ExecutionCaseEvaluation) AttemptOrderMatched() bool {
	return e.attemptOrderMatched
}

// WrongResourceCallCount は、計画と異なる resource 呼出し件数を返す。
func (e ExecutionCaseEvaluation) WrongResourceCallCount() int {
	return e.wrongResourceCallCount
}

// BudgetViolationCount は、候補、呼出し、item または page 予算の違反件数を返す。
func (e ExecutionCaseEvaluation) BudgetViolationCount() int {
	return e.budgetViolationCount
}

// AttemptOrderViolationCount は、plan 順と異なる attempt 件数を返す。
func (e ExecutionCaseEvaluation) AttemptOrderViolationCount() int {
	return e.attemptOrderViolationCount
}

// ImplicitFirstReadCount は、検索第一件を暗黙に読んだ件数を返す。
func (e ExecutionCaseEvaluation) ImplicitFirstReadCount() int {
	return e.implicitFirstReadCount
}

// EmptyReclassificationCount は、空結果後に別 resource へ再分類した件数を返す。
func (e ExecutionCaseEvaluation) EmptyReclassificationCount() int {
	return e.emptyReclassificationCount
}

// FailedChecks は、この case で満たさなかった固定検査 ID を返す。
func (e ExecutionCaseEvaluation) FailedChecks() []string {
	failed := make([]string, 0, 6)
	if !e.expectedMatched {
		failed = append(failed, executionMetricExpectedResult)
	}
	if e.wrongResourceCallCount > 0 {
		failed = append(failed, executionMetricWrongResourceCall)
	}
	if e.budgetViolationCount > 0 {
		failed = append(failed, executionMetricBudgetAdherence)
	}
	if !e.attemptOrderMatched || e.attemptOrderViolationCount > 0 {
		failed = append(failed, executionMetricAttemptOrder)
	}
	if e.implicitFirstReadCount > 0 {
		failed = append(failed, executionMetricImplicitFirstRead)
	}
	if e.emptyReclassificationCount > 0 {
		failed = append(failed, executionMetricEmptyReclassification)
	}
	return failed
}

// Validate は、case ID と非負の違反件数を確認する。
func (e ExecutionCaseEvaluation) Validate() error {
	if e.caseID == "" {
		return fmt.Errorf("execution evaluation の caseId は必須です")
	}
	for name, count := range map[string]int{
		"wrongResourceCallCount":     e.wrongResourceCallCount,
		"budgetViolationCount":       e.budgetViolationCount,
		"attemptOrderViolationCount": e.attemptOrderViolationCount,
		"implicitFirstReadCount":     e.implicitFirstReadCount,
		"emptyReclassificationCount": e.emptyReclassificationCount,
	} {
		if count < 0 {
			return fmt.Errorf("%s は零以上でなければなりません", name)
		}
	}
	if e.attemptOrderMatched && e.attemptOrderViolationCount > 0 {
		return fmt.Errorf("attempt 順一致と順序違反件数が矛盾しています")
	}
	return nil
}
