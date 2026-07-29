package legalqueryeval

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

// SemanticMetricID は、semantic holdout の機械可読な指標を表す。
type SemanticMetricID string

const (
	// SemanticMetricPlanOutcome は、decision、reason および selection の一致を表す。
	SemanticMetricPlanOutcome SemanticMetricID = "plan-outcome"
	// SemanticMetricRequestError は、入力エラーの code と field の一致を表す。
	SemanticMetricRequestError SemanticMetricID = "request-error"
	// SemanticMetricMeaningSignature は、期待 meaning の意味署名一致を表す。
	SemanticMetricMeaningSignature SemanticMetricID = "meaning-signature"
	// SemanticMetricTop1 は、主正解が順位一位にある割合を表す。
	SemanticMetricTop1 SemanticMetricID = "top-1"
	// SemanticMetricTop2 は、主正解が上位二候補にある割合を表す。
	SemanticMetricTop2 SemanticMetricID = "top-2"
	// SemanticMetricHighConfidence は、高確信度一位候補の precision を表す。
	SemanticMetricHighConfidence SemanticMetricID = "high-confidence-precision"
	// SemanticMetricEvidenceAssertion は、根拠 assertion の一致を表す。
	SemanticMetricEvidenceAssertion SemanticMetricID = "evidence-assertion"
	// SemanticMetricConceptAssertion は、法概念 assertion の一致を表す。
	SemanticMetricConceptAssertion SemanticMetricID = "concept-assertion"
)

func semanticMetricIDs() []SemanticMetricID {
	return []SemanticMetricID{
		SemanticMetricPlanOutcome,
		SemanticMetricRequestError,
		SemanticMetricMeaningSignature,
		SemanticMetricTop1,
		SemanticMetricTop2,
		SemanticMetricHighConfidence,
		SemanticMetricEvidenceAssertion,
		SemanticMetricConceptAssertion,
	}
}

// SemanticMetricReport は、一指標の分子、分母、割合および不一致 case を保持する。
type SemanticMetricReport struct {
	metricID      SemanticMetricID
	matched       int
	total         int
	failedCaseIDs []string
}

// MetricID は、機械可読な指標 ID を返す。
func (r SemanticMetricReport) MetricID() SemanticMetricID { return r.metricID }

// Matched は、指標に一致した case 数を返す。
func (r SemanticMetricReport) Matched() int { return r.matched }

// Total は、指標の適用対象 case 数を返す。
func (r SemanticMetricReport) Total() int { return r.total }

// Ratio は、分母が零件の場合に零を返す。
func (r SemanticMetricReport) Ratio() float64 {
	if r.total == 0 {
		return 0
	}
	return float64(r.matched) / float64(r.total)
}

// FailedCaseIDs は、当該指標に一致しない case ID を評価順で返す。
func (r SemanticMetricReport) FailedCaseIDs() []string {
	return append([]string{}, r.failedCaseIDs...)
}

// MarshalJSON は、照会本文を含まない固定の指標 object を返す。
func (r SemanticMetricReport) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		MetricID      SemanticMetricID `json:"metricId"`
		Matched       int              `json:"matched"`
		Total         int              `json:"total"`
		Ratio         float64          `json:"ratio"`
		FailedCaseIDs []string         `json:"failedCaseIds"`
	}{
		MetricID:      r.MetricID(),
		Matched:       r.Matched(),
		Total:         r.Total(),
		Ratio:         r.Ratio(),
		FailedCaseIDs: r.FailedCaseIDs(),
	})
}

// SemanticCategoryReport は、一 category の件数と同じ semantic 指標を保持する。
type SemanticCategoryReport struct {
	categoryID string
	caseCount  int
	metrics    []SemanticMetricReport
}

// CategoryID は、corpus の category ID を返す。
func (r SemanticCategoryReport) CategoryID() string { return r.categoryID }

// CaseCount は、category に属する case 数を返す。
func (r SemanticCategoryReport) CaseCount() int { return r.caseCount }

// Metrics は、固定順の指標 report を返す。
func (r SemanticCategoryReport) Metrics() []SemanticMetricReport {
	return append([]SemanticMetricReport{}, r.metrics...)
}

// MarshalJSON は、category の件数と指標を固定 object で返す。
func (r SemanticCategoryReport) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CategoryID string                 `json:"categoryId"`
		CaseCount  int                    `json:"caseCount"`
		Metrics    []SemanticMetricReport `json:"metrics"`
	}{
		CategoryID: r.CategoryID(),
		CaseCount:  r.CaseCount(),
		Metrics:    r.Metrics(),
	})
}

// SemanticReport は、holdout 全体と category 別の semantic 評価結果を保持する。
type SemanticReport struct {
	caseCount     int
	metrics       []SemanticMetricReport
	categories    []SemanticCategoryReport
	failedCaseIDs []string
	initialized   bool
}

// CaseCount は、評価した holdout case 数を返す。
func (r SemanticReport) CaseCount() int { return r.caseCount }

// Metrics は、固定順の全体指標 report を返す。
func (r SemanticReport) Metrics() []SemanticMetricReport {
	return append([]SemanticMetricReport{}, r.metrics...)
}

// Categories は、category ID 順の report を返す。
func (r SemanticReport) Categories() []SemanticCategoryReport {
	return append([]SemanticCategoryReport{}, r.categories...)
}

// FailedCaseIDs は、一つ以上の適用指標に不一致がある case ID を評価順で返す。
func (r SemanticReport) FailedCaseIDs() []string {
	return append([]string{}, r.failedCaseIDs...)
}

// MarshalJSON は、照会本文を含まない semantic 評価 object を返す。
func (r SemanticReport) MarshalJSON() ([]byte, error) {
	if !r.initialized {
		return nil, fmt.Errorf("SemanticReport は NewSemanticReport で作成しなければなりません")
	}
	return json.Marshal(struct {
		CaseCount     int                      `json:"caseCount"`
		Metrics       []SemanticMetricReport   `json:"metrics"`
		Categories    []SemanticCategoryReport `json:"categories"`
		FailedCaseIDs []string                 `json:"failedCaseIds"`
	}{
		CaseCount:     r.CaseCount(),
		Metrics:       r.Metrics(),
		Categories:    r.Categories(),
		FailedCaseIDs: r.FailedCaseIDs(),
	})
}

// NewSemanticReport は、runner の評価結果を決定的な JSON report へ投影する。
func NewSemanticReport(result SemanticHoldoutResult) (SemanticReport, error) {
	evaluations := result.Evaluations()
	if len(evaluations) == 0 {
		return SemanticReport{}, fmt.Errorf("semantic evaluation は一件以上必要です")
	}
	if _, err := AggregateSemanticCaseEvaluations(evaluations); err != nil {
		return SemanticReport{}, fmt.Errorf(
			"semantic evaluation が有効ではありません: %w",
			err,
		)
	}
	return SemanticReport{
		caseCount:     len(evaluations),
		metrics:       buildSemanticMetricReports(evaluations),
		categories:    buildSemanticCategoryReports(evaluations),
		failedCaseIDs: semanticFailedCaseIDs(evaluations),
		initialized:   true,
	}, nil
}

func buildSemanticMetricReports(
	evaluations []SemanticCaseEvaluation,
) []SemanticMetricReport {
	metricIDs := semanticMetricIDs()
	reports := make([]SemanticMetricReport, 0, len(metricIDs))
	for _, metricID := range metricIDs {
		report := SemanticMetricReport{metricID: metricID}
		for _, evaluation := range evaluations {
			matched, applicable := semanticMetricResult(metricID, evaluation)
			if !applicable {
				continue
			}
			report.total++
			if matched {
				report.matched++
				continue
			}
			report.failedCaseIDs = append(
				report.failedCaseIDs,
				evaluation.CaseID(),
			)
		}
		reports = append(reports, report)
	}
	return reports
}

func buildSemanticCategoryReports(
	evaluations []SemanticCaseEvaluation,
) []SemanticCategoryReport {
	byCategory := make(map[string][]SemanticCaseEvaluation)
	for _, evaluation := range evaluations {
		for _, categoryID := range evaluation.CategoryIDs() {
			byCategory[categoryID] = append(byCategory[categoryID], evaluation)
		}
	}
	categoryIDs := make([]string, 0, len(byCategory))
	for categoryID := range byCategory {
		categoryIDs = append(categoryIDs, categoryID)
	}
	sort.Strings(categoryIDs)

	reports := make([]SemanticCategoryReport, 0, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		values := byCategory[categoryID]
		reports = append(reports, SemanticCategoryReport{
			categoryID: categoryID,
			caseCount:  len(values),
			metrics:    buildSemanticMetricReports(values),
		})
	}
	return reports
}

func semanticFailedCaseIDs(
	evaluations []SemanticCaseEvaluation,
) []string {
	caseIDs := make([]string, 0)
	for _, evaluation := range evaluations {
		for _, metricID := range semanticMetricIDs() {
			matched, applicable := semanticMetricResult(metricID, evaluation)
			if applicable && !matched {
				caseIDs = append(caseIDs, evaluation.CaseID())
				break
			}
		}
	}
	return caseIDs
}

func semanticMetricResult(
	metricID SemanticMetricID,
	evaluation SemanticCaseEvaluation,
) (matched bool, applicable bool) {
	switch metricID {
	case SemanticMetricPlanOutcome:
		return evaluation.PlanOutcomeMatched(),
			evaluation.ExpectedKind() == legalquerycorpus.SemanticExpectedKindPlan
	case SemanticMetricRequestError:
		return evaluation.RequestErrorMatched(),
			evaluation.ExpectedKind() == legalquerycorpus.SemanticExpectedKindRequestError
	case SemanticMetricMeaningSignature:
		meanings := evaluation.Meanings()
		return allMeaningSignaturesMatched(meanings), len(meanings) > 0
	case SemanticMetricTop1:
		return evaluation.PrimaryTop1Matched(), hasPlanRanking(evaluation)
	case SemanticMetricTop2:
		return evaluation.PrimaryTop2Matched(), hasPlanRanking(evaluation)
	case SemanticMetricHighConfidence:
		return evaluation.HighConfidencePrecision()
	case SemanticMetricEvidenceAssertion:
		return caseEvidenceAssertion(evaluation.Meanings())
	case SemanticMetricConceptAssertion:
		return caseConceptAssertion(evaluation.Meanings())
	default:
		return false, false
	}
}
