package legalqueryeval

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var evaluationMetricIDPattern = regexp.MustCompile(
	`^[a-z0-9]+(?:-[a-z0-9]+)*$`,
)

// EvaluationMetricReportValues は、標準評価指標の入力値を保持する。
type EvaluationMetricReportValues struct {
	MetricID      string
	Numerator     int
	Denominator   int
	FailedCaseIDs []string
}

// EvaluationMetricReport は、標準 command が公開する分子、分母および失敗 case を表す。
type EvaluationMetricReport struct {
	metricID      string
	numerator     int
	denominator   int
	failedCaseIDs []string
}

// NewEvaluationMetricReport は、一つの決定的な評価指標を作成する。
func NewEvaluationMetricReport(
	values EvaluationMetricReportValues,
) (EvaluationMetricReport, error) {
	report := EvaluationMetricReport{
		metricID:      values.MetricID,
		numerator:     values.Numerator,
		denominator:   values.Denominator,
		failedCaseIDs: append([]string{}, values.FailedCaseIDs...),
	}
	if err := report.Validate(); err != nil {
		return EvaluationMetricReport{}, err
	}
	return report, nil
}

// MetricID は、固定の機械可読な指標 ID を返す。
func (r EvaluationMetricReport) MetricID() string { return r.metricID }

// Numerator は、指標の分子を返す。
func (r EvaluationMetricReport) Numerator() int { return r.numerator }

// Denominator は、指標の分母を返す。
func (r EvaluationMetricReport) Denominator() int { return r.denominator }

// Ratio は、分母が零の場合に零を返す。
func (r EvaluationMetricReport) Ratio() float64 {
	if r.denominator == 0 {
		return 0
	}
	return float64(r.numerator) / float64(r.denominator)
}

// FailedCaseIDs は、評価順の不一致 case ID の複製を返す。
func (r EvaluationMetricReport) FailedCaseIDs() []string {
	return append([]string{}, r.failedCaseIDs...)
}

// Validate は、指標 ID、件数および失敗 case の整合を確認する。
func (r EvaluationMetricReport) Validate() error {
	if !evaluationMetricIDPattern.MatchString(r.metricID) {
		return fmt.Errorf("metricId が定義済みの形式ではありません")
	}
	if r.denominator < 0 ||
		r.numerator < 0 ||
		r.numerator > r.denominator {
		return fmt.Errorf("指標の分子は零以上かつ分母以下でなければなりません")
	}
	if len(r.failedCaseIDs) != r.denominator-r.numerator {
		return fmt.Errorf("failedCaseIds は不一致件数と一致しなければなりません")
	}
	seen := make(map[string]struct{}, len(r.failedCaseIDs))
	for _, caseID := range r.failedCaseIDs {
		if caseID == "" {
			return fmt.Errorf("failedCaseIds に空の case ID を含められません")
		}
		if _, exists := seen[caseID]; exists {
			return fmt.Errorf("failedCaseIds に同じ case ID を重複させられません")
		}
		seen[caseID] = struct{}{}
	}
	return nil
}

// MarshalJSON は、分子と分母を明示した固定 object を返す。
func (r EvaluationMetricReport) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		MetricID      string   `json:"metricId"`
		Numerator     int      `json:"numerator"`
		Denominator   int      `json:"denominator"`
		Ratio         float64  `json:"ratio"`
		FailedCaseIDs []string `json:"failedCaseIds"`
	}{
		MetricID:      r.MetricID(),
		Numerator:     r.Numerator(),
		Denominator:   r.Denominator(),
		Ratio:         r.Ratio(),
		FailedCaseIDs: r.FailedCaseIDs(),
	})
}

// UnmarshalJSON は、baseline loader を介さない直接復元を拒否する。
func (*EvaluationMetricReport) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"EvaluationMetricReport は JSON から直接復元できません。baseline loader を使用してください",
	)
}
