package legalqueryeval

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

const (
	derivedObservationCorePack      = "composition-core-pack"
	derivedObservationPackDisabled  = "composition-pack-disabled"
	derivedObservationRefReadSearch = "composition-ref-read-search"
	derivedObservationFourStep      = "composition-four-step-budget"
)

// EvaluateDerivedObservations は、期待 meaning から導出した合成条件を評価する。
func EvaluateDerivedObservations(
	semanticCases []legalquerycorpus.SemanticCase,
	semantic SemanticReport,
) ([]EvaluationMetricReport, error) {
	if !semantic.initialized || semantic.CaseCount() != len(semanticCases) {
		return nil, fmt.Errorf(
			"semantic report と holdout case 件数が一致しません",
		)
	}
	failures, err := derivedObservationFailureCases(semantic)
	if err != nil {
		return nil, err
	}
	type observationCounter struct {
		total   int
		matched int
		failed  []string
	}
	observationIDs := derivedObservationIDs()
	counters := make(map[string]*observationCounter, len(observationIDs))
	for _, observationID := range observationIDs {
		counters[observationID] = &observationCounter{}
	}
	for _, semanticCase := range semanticCases {
		if err := semanticCase.Validate(); err != nil {
			return nil, fmt.Errorf(
				"semantic case %q が有効ではありません: %w",
				semanticCase.CaseID(),
				err,
			)
		}
		expected, ok := semanticCase.Expected().(legalquerycorpus.ExpectedPlan)
		if !ok {
			continue
		}
		applicable, err := applicableDerivedObservations(expected)
		if err != nil {
			return nil, fmt.Errorf(
				"semantic case %q の派生観測を計算できません: %w",
				semanticCase.CaseID(),
				err,
			)
		}
		for _, observationID := range applicable {
			counter := counters[observationID]
			counter.total++
			if _, failed := failures[semanticCase.CaseID()]; !failed {
				counter.matched++
				continue
			}
			counter.failed = append(counter.failed, semanticCase.CaseID())
		}
	}
	reports := make([]EvaluationMetricReport, 0, len(observationIDs))
	for _, observationID := range observationIDs {
		counter := counters[observationID]
		if counter.total == 0 {
			return nil, fmt.Errorf(
				"派生観測 %s の対象 case がありません",
				observationID,
			)
		}
		report, err := NewEvaluationMetricReport(
			EvaluationMetricReportValues{
				MetricID:      observationID,
				Numerator:     counter.matched,
				Denominator:   counter.total,
				FailedCaseIDs: counter.failed,
			},
		)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func derivedObservationIDs() []string {
	return []string{
		derivedObservationCorePack,
		derivedObservationPackDisabled,
		derivedObservationRefReadSearch,
		derivedObservationFourStep,
	}
}

func derivedObservationFailureCases(
	semantic SemanticReport,
) (map[string]struct{}, error) {
	failures := make(map[string]struct{})
	required := map[SemanticMetricID]bool{
		SemanticMetricPlanOutcome:      false,
		SemanticMetricMeaningSignature: false,
	}
	for _, metric := range semantic.Metrics() {
		if _, exists := required[metric.MetricID()]; !exists {
			continue
		}
		required[metric.MetricID()] = true
		for _, caseID := range metric.FailedCaseIDs() {
			failures[caseID] = struct{}{}
		}
	}
	for metricID, found := range required {
		if !found {
			return nil, fmt.Errorf(
				"semantic report に指標 %s がありません",
				metricID,
			)
		}
	}
	return failures, nil
}

func applicableDerivedObservations(
	expected legalquerycorpus.ExpectedPlan,
) ([]string, error) {
	byID := make(map[string]legalquerycorpus.ExpectedMeaning)
	for _, meaning := range expected.Meanings() {
		byID[meaning.MeaningID()] = meaning
	}
	applicable := make(map[string]struct{})
	for _, meaningID := range expected.SelectedMeaningIDs() {
		meaning, exists := byID[meaningID]
		if !exists {
			return nil, fmt.Errorf(
				"selected meaning %q が meanings にありません",
				meaningID,
			)
		}
		steps := meaning.Steps()
		if isCorePackComposition(meaning, steps) {
			applicable[derivedObservationCorePack] = struct{}{}
		}
		if len(meaning.RequiredPacks()) > 0 &&
			expected.Decision() == legalquery.PlanDecisionCapabilityUnavailable {
			applicable[derivedObservationPackDisabled] = struct{}{}
		}
		if isRefReadSearchComposition(steps) {
			applicable[derivedObservationRefReadSearch] = struct{}{}
		}
		if len(steps) == 4 &&
			(expected.Decision() == legalquery.PlanDecisionSingle ||
				expected.Decision() == legalquery.PlanDecisionHedged) {
			applicable[derivedObservationFourStep] = struct{}{}
		}
	}
	result := make([]string, 0, len(applicable))
	for _, observationID := range derivedObservationIDs() {
		if _, exists := applicable[observationID]; exists {
			result = append(result, observationID)
		}
	}
	return result, nil
}

func isCorePackComposition(
	meaning legalquerycorpus.ExpectedMeaning,
	steps []legalquerycorpus.ExpectedStep,
) bool {
	if len(meaning.RequiredPacks()) == 0 {
		return false
	}
	hasCore := false
	hasPack := false
	for _, step := range steps {
		if step.Resource() == legalquery.ResourceJudicialDecision {
			hasPack = true
			continue
		}
		hasCore = true
	}
	return hasCore && hasPack
}

func isRefReadSearchComposition(
	steps []legalquerycorpus.ExpectedStep,
) bool {
	hasRefRead := false
	hasSearch := false
	for _, step := range steps {
		if step.InputKind() == legalquery.InputKindJudicialDecisionRead {
			hasRefRead = true
		}
		if step.Task() == legalquery.TaskSearch {
			hasSearch = true
		}
	}
	return hasRefRead && hasSearch
}
