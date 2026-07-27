package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

// validateIntegrityExecutionScenarioRequirements は、失敗時に部分結果を破棄する。
func validateIntegrityExecutionScenarioRequirements(
	checked integrityCheckedCorpus,
) (integrityCheckedCorpus, error) {
	if err := validateExecutionScenarioRequirements(
		checked.manifest,
		checked.development,
		checked.execution,
	); err != nil {
		return integrityCheckedCorpus{}, err
	}
	return checked, nil
}

// validateExecutionScenarioRequirements は、横断条件と全集合の網羅を確認する。
func validateExecutionScenarioRequirements(
	manifest Manifest,
	development []SemanticCase,
	execution []ExecutionCase,
) error {
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("execution scenario の manifest が有効ではありません")
	}
	developmentByID, err := indexExecutionScenarioDevelopment(development)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(manifest.RequiredExecutionScenarioIDs()))
	for _, executionCase := range execution {
		if err := executionCase.Validate(); err != nil {
			return fmt.Errorf("execution scenario の execution case が有効ではありません")
		}
		semanticCase, exists := developmentByID[executionCase.SemanticCaseID()]
		if !exists {
			return fmt.Errorf(
				"execution case %q の semanticCaseId は development に存在しません",
				executionCase.CaseID(),
			)
		}
		plan, err := executionReferencePlanForSemanticCase(semanticCase)
		if err != nil {
			return fmt.Errorf(
				"execution case %q の scenario 参照先 plan が有効ではありません",
				executionCase.CaseID(),
			)
		}
		if err := validateExecutionScenarioCrossConditions(
			executionCase,
			plan,
		); err != nil {
			return err
		}
		for _, scenarioID := range executionCase.ScenarioIDs() {
			seen[scenarioID] = struct{}{}
		}
	}
	return validateExecutionScenarioCoverage(
		manifest.RequiredExecutionScenarioIDs(),
		seen,
	)
}

func indexExecutionScenarioDevelopment(
	development []SemanticCase,
) (map[string]SemanticCase, error) {
	values := make(map[string]SemanticCase, len(development))
	for _, semanticCase := range development {
		if err := semanticCase.Validate(); err != nil {
			return nil, fmt.Errorf(
				"execution scenario の development case が有効ではありません",
			)
		}
		values[semanticCase.CaseID()] = semanticCase
	}
	return values, nil
}

func validateExecutionScenarioCrossConditions(
	executionCase ExecutionCase,
	plan executionReferencePlan,
) error {
	for _, scenarioID := range executionCase.ScenarioIDs() {
		switch ExecutionScenarioID(scenarioID) {
		case ExecutionScenarioIDReversedCompletion:
			if plan.decision != legalquery.PlanDecisionHedged {
				return executionScenarioCrossConditionError(scenarioID)
			}
		case ExecutionScenarioIDItemBudget:
			if !hasExecutionSourceBeyondEffectiveLimit(executionCase, plan) {
				return executionScenarioCrossConditionError(scenarioID)
			}
		}
	}
	return nil
}

func hasExecutionSourceBeyondEffectiveLimit(
	executionCase ExecutionCase,
	plan executionReferencePlan,
) bool {
	if !plan.hasEffectiveLimit {
		return false
	}
	for _, action := range executionCase.Actions() {
		outcome, ok := action.Outcome().(CollectionSuccessOutcome)
		if ok && outcome.SourceItemCount() > plan.effectiveLimit {
			return true
		}
	}
	return false
}

func validateExecutionScenarioCoverage(
	required []string,
	seen map[string]struct{},
) error {
	for _, scenarioID := range required {
		if _, exists := seen[scenarioID]; !exists {
			return fmt.Errorf(
				"execution scenario %q を一件以上含めなければなりません",
				scenarioID,
			)
		}
	}
	return nil
}

func executionScenarioCrossConditionError(scenarioID string) error {
	return fmt.Errorf(
		"execution scenario %q の集合横断条件を満たしていません",
		scenarioID,
	)
}
