package legalqueryeval

import (
	"encoding/json"
	"fmt"
)

const (
	executionMetricExpectedResult        = "expected-execution"
	executionMetricWrongResourceCall     = "no-wrong-resource-call"
	executionMetricBudgetAdherence       = "budget-adherence"
	executionMetricAttemptOrder          = "attempt-order-determinism"
	executionMetricImplicitFirstRead     = "no-implicit-first-read"
	executionMetricEmptyReclassification = "no-empty-reclassification"
)

// ExecutionReport は、execution 集合の固定指標と違反件数を保持する。
type ExecutionReport struct {
	caseCount                  int
	metrics                    []EvaluationMetricReport
	wrongResourceCallCount     int
	budgetViolationCount       int
	attemptOrderViolationCount int
	implicitFirstReadCount     int
	emptyReclassificationCount int
	failedCaseIDs              []string
	initialized                bool
}

// NewExecutionReport は、manifest 順の case 評価を標準 report へ集計する。
func NewExecutionReport(
	evaluations []ExecutionCaseEvaluation,
) (ExecutionReport, error) {
	if len(evaluations) == 0 {
		return ExecutionReport{}, fmt.Errorf("execution evaluation は一件以上必要です")
	}
	if err := validateExecutionCaseEvaluations(evaluations); err != nil {
		return ExecutionReport{}, err
	}
	report := ExecutionReport{
		caseCount:     len(evaluations),
		metrics:       buildExecutionMetrics(evaluations),
		failedCaseIDs: []string{},
		initialized:   true,
	}
	for _, evaluation := range evaluations {
		report.wrongResourceCallCount += evaluation.WrongResourceCallCount()
		report.budgetViolationCount += evaluation.BudgetViolationCount()
		report.attemptOrderViolationCount += evaluation.AttemptOrderViolationCount()
		report.implicitFirstReadCount += evaluation.ImplicitFirstReadCount()
		report.emptyReclassificationCount += evaluation.EmptyReclassificationCount()
		if len(evaluation.FailedChecks()) > 0 {
			report.failedCaseIDs = append(
				report.failedCaseIDs,
				evaluation.CaseID(),
			)
		}
	}
	return report, nil
}

// CaseCount は、評価した execution fixture 件数を返す。
func (r ExecutionReport) CaseCount() int { return r.caseCount }

// Metrics は、固定順の実行指標を返す。
func (r ExecutionReport) Metrics() []EvaluationMetricReport {
	return append([]EvaluationMetricReport{}, r.metrics...)
}

// WrongResourceCallCount は、誤 resource 呼出し総数を返す。
func (r ExecutionReport) WrongResourceCallCount() int {
	return r.wrongResourceCallCount
}

// BudgetViolationCount は、固定予算違反総数を返す。
func (r ExecutionReport) BudgetViolationCount() int {
	return r.budgetViolationCount
}

// AttemptOrderViolationCount は、attempt 順序違反総数を返す。
func (r ExecutionReport) AttemptOrderViolationCount() int {
	return r.attemptOrderViolationCount
}

// ImplicitFirstReadCount は、暗黙 read 総数を返す。
func (r ExecutionReport) ImplicitFirstReadCount() int {
	return r.implicitFirstReadCount
}

// EmptyReclassificationCount は、空結果後の再分類総数を返す。
func (r ExecutionReport) EmptyReclassificationCount() int {
	return r.emptyReclassificationCount
}

// FailedCaseIDs は、一つ以上の実行検査に失敗した case ID を評価順で返す。
func (r ExecutionReport) FailedCaseIDs() []string {
	return append([]string{}, r.failedCaseIDs...)
}

// Validate は、集計済み report の件数と指標順を確認する。
func (r ExecutionReport) Validate() error {
	if !r.initialized {
		return fmt.Errorf("ExecutionReport は NewExecutionReport で作成しなければなりません")
	}
	if r.caseCount < 1 || len(r.metrics) != len(executionMetricIDs()) {
		return fmt.Errorf("execution report の件数または指標数が不正です")
	}
	for name, count := range map[string]int{
		"wrongResourceCallCount":     r.wrongResourceCallCount,
		"budgetViolationCount":       r.budgetViolationCount,
		"attemptOrderViolationCount": r.attemptOrderViolationCount,
		"implicitFirstReadCount":     r.implicitFirstReadCount,
		"emptyReclassificationCount": r.emptyReclassificationCount,
	} {
		if count < 0 {
			return fmt.Errorf("%s は零以上でなければなりません", name)
		}
	}
	for index, metricID := range executionMetricIDs() {
		if err := r.metrics[index].Validate(); err != nil {
			return err
		}
		if r.metrics[index].MetricID() != metricID ||
			r.metrics[index].Denominator() != r.caseCount {
			return fmt.Errorf("execution metrics が固定順または母集団と一致しません")
		}
	}
	return validateFailedCaseIDs(r.failedCaseIDs, r.caseCount)
}

// MarshalJSON は、実行指標、違反件数および失敗 case だけを返す。
func (r ExecutionReport) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		CaseCount                  int                      `json:"caseCount"`
		Metrics                    []EvaluationMetricReport `json:"metrics"`
		WrongResourceCallCount     int                      `json:"wrongResourceCallCount"`
		BudgetViolationCount       int                      `json:"budgetViolationCount"`
		AttemptOrderViolationCount int                      `json:"attemptOrderViolationCount"`
		ImplicitFirstReadCount     int                      `json:"implicitFirstReadCount"`
		EmptyReclassificationCount int                      `json:"emptyReclassificationCount"`
		FailedCaseIDs              []string                 `json:"failedCaseIds"`
	}{
		CaseCount:                  r.CaseCount(),
		Metrics:                    r.Metrics(),
		WrongResourceCallCount:     r.WrongResourceCallCount(),
		BudgetViolationCount:       r.BudgetViolationCount(),
		AttemptOrderViolationCount: r.AttemptOrderViolationCount(),
		ImplicitFirstReadCount:     r.ImplicitFirstReadCount(),
		EmptyReclassificationCount: r.EmptyReclassificationCount(),
		FailedCaseIDs:              r.FailedCaseIDs(),
	})
}

// UnmarshalJSON は、baseline loader を介さない直接復元を拒否する。
func (*ExecutionReport) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ExecutionReport は JSON から直接復元できません。baseline loader を使用してください",
	)
}

func executionMetricIDs() []string {
	return []string{
		executionMetricExpectedResult,
		executionMetricWrongResourceCall,
		executionMetricBudgetAdherence,
		executionMetricAttemptOrder,
		executionMetricImplicitFirstRead,
		executionMetricEmptyReclassification,
	}
}

func validateExecutionCaseEvaluations(
	evaluations []ExecutionCaseEvaluation,
) error {
	seen := make(map[string]struct{}, len(evaluations))
	for _, evaluation := range evaluations {
		if err := evaluation.Validate(); err != nil {
			return fmt.Errorf("execution evaluation が有効ではありません: %w", err)
		}
		if _, exists := seen[evaluation.CaseID()]; exists {
			return fmt.Errorf("execution evaluation の caseId を重複させられません")
		}
		seen[evaluation.CaseID()] = struct{}{}
	}
	return nil
}

func buildExecutionMetrics(
	evaluations []ExecutionCaseEvaluation,
) []EvaluationMetricReport {
	reports := make([]EvaluationMetricReport, 0, len(executionMetricIDs()))
	for _, metricID := range executionMetricIDs() {
		numerator := 0
		failed := make([]string, 0)
		for _, evaluation := range evaluations {
			if executionMetricMatched(metricID, evaluation) {
				numerator++
				continue
			}
			failed = append(failed, evaluation.CaseID())
		}
		report, err := NewEvaluationMetricReport(
			EvaluationMetricReportValues{
				MetricID:      metricID,
				Numerator:     numerator,
				Denominator:   len(evaluations),
				FailedCaseIDs: failed,
			},
		)
		if err != nil {
			panic(fmt.Sprintf("検証済み execution metric の構築に失敗しました: %v", err))
		}
		reports = append(reports, report)
	}
	return reports
}

func executionMetricMatched(
	metricID string,
	evaluation ExecutionCaseEvaluation,
) bool {
	switch metricID {
	case executionMetricExpectedResult:
		return evaluation.ExpectedMatched()
	case executionMetricWrongResourceCall:
		return evaluation.WrongResourceCallCount() == 0
	case executionMetricBudgetAdherence:
		return evaluation.BudgetViolationCount() == 0
	case executionMetricAttemptOrder:
		return evaluation.AttemptOrderMatched() &&
			evaluation.AttemptOrderViolationCount() == 0
	case executionMetricImplicitFirstRead:
		return evaluation.ImplicitFirstReadCount() == 0
	case executionMetricEmptyReclassification:
		return evaluation.EmptyReclassificationCount() == 0
	default:
		return false
	}
}
