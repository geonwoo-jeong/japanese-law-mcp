package legalqueryeval

import (
	"fmt"
)

// MetricCounter は、一つの評価指標の分子と分母を保持する。
type MetricCounter struct {
	matched int
	total   int
}

// Matched は、分子を返す。
func (c MetricCounter) Matched() int {
	return c.matched
}

// Total は、分母を返す。
func (c MetricCounter) Total() int {
	return c.total
}

// Ratio は、分母が 0 の場合 0 を返す。
func (c MetricCounter) Ratio() float64 {
	if c.total == 0 {
		return 0
	}
	return float64(c.matched) / float64(c.total)
}

// CategoryMetrics は、一 category に属する case の指標を保持する。
type CategoryMetrics struct {
	categoryID        string
	caseCount         int
	planOutcome       MetricCounter
	requestError      MetricCounter
	top1              MetricCounter
	top2              MetricCounter
	highConfidence    MetricCounter
	evidenceAssertion MetricCounter
	conceptAssertion  MetricCounter
}

// CategoryID は、集計対象 category を返す。
func (m CategoryMetrics) CategoryID() string { return m.categoryID }

// CaseCount は、当該 category に属する case 数を返す。
func (m CategoryMetrics) CaseCount() int { return m.caseCount }

// PlanOutcomeMetric は、decision/reason/selection 完全一致指標を返す。
func (m CategoryMetrics) PlanOutcomeMetric() MetricCounter { return m.planOutcome }

// RequestErrorMetric は、request_error 完全一致指標を返す。
func (m CategoryMetrics) RequestErrorMetric() MetricCounter { return m.requestError }

// Top1Metric は、主正解 top-1 指標を返す。
func (m CategoryMetrics) Top1Metric() MetricCounter { return m.top1 }

// Top2Metric は、主正解 top-2 指標を返す。
func (m CategoryMetrics) Top2Metric() MetricCounter { return m.top2 }

// HighConfidenceMetric は、高確信度 precision 指標を返す。
func (m CategoryMetrics) HighConfidenceMetric() MetricCounter { return m.highConfidence }

// EvidenceAssertionMetric は、根拠 assertion 指標を返す。
func (m CategoryMetrics) EvidenceAssertionMetric() MetricCounter { return m.evidenceAssertion }

// ConceptAssertionMetric は、法概念 assertion 指標を返す。
func (m CategoryMetrics) ConceptAssertionMetric() MetricCounter { return m.conceptAssertion }

// SemanticCaseMetrics は、semantic case 集計の主要指標を保持する。
type SemanticCaseMetrics struct {
	caseCount         int
	planOutcome       MetricCounter
	requestError      MetricCounter
	top1              MetricCounter
	top2              MetricCounter
	highConfidence    MetricCounter
	evidenceAssertion MetricCounter
	conceptAssertion  MetricCounter
	categories        []CategoryMetrics
}

// CaseCount は、評価した case 数を返す。
func (m SemanticCaseMetrics) CaseCount() int { return m.caseCount }

// PlanOutcomeMetric は、decision/reason/selection 完全一致指標を返す。
func (m SemanticCaseMetrics) PlanOutcomeMetric() MetricCounter { return m.planOutcome }

// RequestErrorMetric は、request_error 完全一致指標を返す。
func (m SemanticCaseMetrics) RequestErrorMetric() MetricCounter { return m.requestError }

// Top1Metric は、主正解 top-1 指標を返す。
func (m SemanticCaseMetrics) Top1Metric() MetricCounter { return m.top1 }

// Top2Metric は、主正解 top-2 指標を返す。
func (m SemanticCaseMetrics) Top2Metric() MetricCounter { return m.top2 }

// HighConfidenceMetric は、高確信度 precision 指標を返す。
func (m SemanticCaseMetrics) HighConfidenceMetric() MetricCounter { return m.highConfidence }

// EvidenceAssertionMetric は、根拠 assertion 指標を返す。
func (m SemanticCaseMetrics) EvidenceAssertionMetric() MetricCounter { return m.evidenceAssertion }

// ConceptAssertionMetric は、法概念 assertion 指標を返す。
func (m SemanticCaseMetrics) ConceptAssertionMetric() MetricCounter { return m.conceptAssertion }

// CategoryMetrics は、category ごとの micro 集計を返す。
func (m SemanticCaseMetrics) CategoryMetrics() []CategoryMetrics {
	return append([]CategoryMetrics{}, m.categories...)
}

// AggregateSemanticCaseEvaluations は、semantic case 評価結果を micro 集計する。
func AggregateSemanticCaseEvaluations(
	evaluations []SemanticCaseEvaluation,
) (SemanticCaseMetrics, error) {
	categoryMap := make(map[string]*CategoryMetrics)
	categoryOrder := make([]string, 0)
	metrics := SemanticCaseMetrics{}

	for _, evaluation := range evaluations {
		metrics.caseCount++
		applyEvaluation(&metrics.planOutcome, evaluation.ExpectedKind() == "plan", evaluation.PlanOutcomeMatched())
		applyEvaluation(&metrics.requestError, evaluation.ExpectedKind() == "request_error", evaluation.RequestErrorMatched())
		applyEvaluation(&metrics.top1, hasPlanRanking(evaluation), evaluation.PrimaryTop1Matched())
		applyEvaluation(&metrics.top2, hasPlanRanking(evaluation), evaluation.PrimaryTop2Matched())
		highMatched, highApplicable := evaluation.HighConfidencePrecision()
		applyEvaluation(&metrics.highConfidence, highApplicable, highMatched)
		for _, meaning := range evaluation.Meanings() {
			evidenceMatched, evidenceApplicable := meaning.EvidenceAssertion()
			conceptMatched, conceptApplicable := meaning.ConceptAssertion()
			applyEvaluation(&metrics.evidenceAssertion, evidenceApplicable, evidenceMatched)
			applyEvaluation(&metrics.conceptAssertion, conceptApplicable, conceptMatched)
		}

		for _, categoryID := range evaluation.CategoryIDs() {
			categoryMetrics, exists := categoryMap[categoryID]
			if !exists {
				categoryMetrics = &CategoryMetrics{categoryID: categoryID}
				categoryMap[categoryID] = categoryMetrics
				categoryOrder = append(categoryOrder, categoryID)
			}
			categoryMetrics.caseCount++
			applyEvaluation(&categoryMetrics.planOutcome, evaluation.ExpectedKind() == "plan", evaluation.PlanOutcomeMatched())
			applyEvaluation(&categoryMetrics.requestError, evaluation.ExpectedKind() == "request_error", evaluation.RequestErrorMatched())
			applyEvaluation(&categoryMetrics.top1, hasPlanRanking(evaluation), evaluation.PrimaryTop1Matched())
			applyEvaluation(&categoryMetrics.top2, hasPlanRanking(evaluation), evaluation.PrimaryTop2Matched())
			applyEvaluation(&categoryMetrics.highConfidence, highApplicable, highMatched)
			for _, meaning := range evaluation.Meanings() {
				evidenceMatched, evidenceApplicable := meaning.EvidenceAssertion()
				conceptMatched, conceptApplicable := meaning.ConceptAssertion()
				applyEvaluation(&categoryMetrics.evidenceAssertion, evidenceApplicable, evidenceMatched)
				applyEvaluation(&categoryMetrics.conceptAssertion, conceptApplicable, conceptMatched)
			}
		}
	}

	metrics.categories = make([]CategoryMetrics, 0, len(categoryOrder))
	for _, categoryID := range categoryOrder {
		value, exists := categoryMap[categoryID]
		if !exists {
			return SemanticCaseMetrics{}, fmt.Errorf("category metrics の内部状態が壊れています")
		}
		metrics.categories = append(metrics.categories, *value)
	}
	return metrics, nil
}

func applyEvaluation(counter *MetricCounter, applicable bool, matched bool) {
	if !applicable {
		return
	}
	counter.total++
	if matched {
		counter.matched++
	}
}

func hasPlanRanking(evaluation SemanticCaseEvaluation) bool {
	return evaluation.ExpectedKind() == "plan" && len(evaluation.Meanings()) > 0
}
