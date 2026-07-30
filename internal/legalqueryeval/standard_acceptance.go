package legalqueryeval

import "fmt"

// VerifyStandardAcceptance は、SOT-ENG-024 の全受入基準を標準 report に適用する。
func VerifyStandardAcceptance(report StandardReport) error {
	if err := report.Validate(); err != nil {
		return err
	}
	holdout := report.Sets().Holdout()
	if err := verifyMetricThresholds("holdout", holdout.Metrics(), true); err != nil {
		return err
	}
	for _, category := range holdout.Categories() {
		if err := verifyMetricThresholds(
			"category "+category.CategoryID(),
			category.Metrics(),
			false,
		); err != nil {
			return err
		}
	}
	for _, observation := range holdout.DerivedObservations() {
		if observation.Numerator() != observation.Denominator() {
			return fmt.Errorf(
				"派生観測 %s が受入基準を満たしません: failedCaseIds=%v",
				observation.MetricID(),
				observation.FailedCaseIDs(),
			)
		}
	}
	for _, metric := range report.Sets().Execution().Metrics() {
		if metric.Numerator() != metric.Denominator() {
			return fmt.Errorf(
				"execution 指標 %s が受入基準を満たしません: failedCaseIds=%v",
				metric.MetricID(),
				metric.FailedCaseIDs(),
			)
		}
	}
	return nil
}

func verifyMetricThresholds(
	scope string,
	metrics []EvaluationMetricReport,
	requireHighConfidencePopulation bool,
) error {
	thresholds := map[string]float64{
		"plan-reproducibility":                  1,
		string(SemanticMetricPlanOutcome):       1,
		string(SemanticMetricRequestError):      1,
		string(SemanticMetricMeaningSignature):  1,
		string(SemanticMetricTop1):              0.90,
		string(SemanticMetricTop2):              0.98,
		string(SemanticMetricHighConfidence):    0.95,
		string(SemanticMetricEvidenceAssertion): 1,
		string(SemanticMetricConceptAssertion):  1,
	}
	seen := make(map[string]struct{}, len(metrics))
	for _, metric := range metrics {
		if err := metric.Validate(); err != nil {
			return fmt.Errorf("%s 指標が有効ではありません: %w", scope, err)
		}
		threshold, exists := thresholds[metric.MetricID()]
		if !exists {
			return fmt.Errorf("%s に未知の指標 %s があります", scope, metric.MetricID())
		}
		seen[metric.MetricID()] = struct{}{}
		if metric.MetricID() == string(SemanticMetricHighConfidence) &&
			metric.Denominator() == 0 {
			if requireHighConfidencePopulation {
				return fmt.Errorf(
					"%s の high-confidence 指標の分母が零件です",
					scope,
				)
			}
			continue
		}
		if metric.Denominator() == 0 {
			continue
		}
		if metric.Ratio() < threshold {
			return fmt.Errorf(
				"%s の指標 %s が受入基準 %.2f を満たしません: failedCaseIds=%v",
				scope,
				metric.MetricID(),
				threshold,
				metric.FailedCaseIDs(),
			)
		}
	}
	for metricID := range thresholds {
		if metricID == "plan-reproducibility" && scope != "holdout" {
			continue
		}
		if _, exists := seen[metricID]; !exists {
			return fmt.Errorf("%s に指標 %s がありません", scope, metricID)
		}
	}
	return nil
}
