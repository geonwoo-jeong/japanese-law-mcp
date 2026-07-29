package legalquerycorpus

import "fmt"

const maximumExecutionScenarioIDs = 8

// ExecutionScenarioID は、execution fixture が再現する実行条件を表す。
type ExecutionScenarioID string

const (
	// ExecutionScenarioIDAllFailed は、全 action の失敗を表す。
	ExecutionScenarioIDAllFailed ExecutionScenarioID = "execution-all-failed"
	// ExecutionScenarioIDEmpty は、全 collection が空であることを表す。
	ExecutionScenarioIDEmpty ExecutionScenarioID = "execution-empty"
	// ExecutionScenarioIDItemBudget は、collection の item 上限を表す。
	ExecutionScenarioIDItemBudget ExecutionScenarioID = "execution-item-budget"
	// ExecutionScenarioIDMixedComposition は、core と pack の混合意味の実行を表す。
	ExecutionScenarioIDMixedComposition ExecutionScenarioID = "execution-mixed-composition"
	// ExecutionScenarioIDNonempty は、一件以上を公開する成功を表す。
	ExecutionScenarioIDNonempty ExecutionScenarioID = "execution-nonempty"
	// ExecutionScenarioIDPartialFailure は、成功と失敗の混在を表す。
	ExecutionScenarioIDPartialFailure ExecutionScenarioID = "execution-partial-failure"
	// ExecutionScenarioIDReversedCompletion は、meaning 間の完了順逆転を表す。
	ExecutionScenarioIDReversedCompletion ExecutionScenarioID = "execution-reversed-completion"
	// ExecutionScenarioIDTimeout は、fake clock の timeout を表す。
	ExecutionScenarioIDTimeout ExecutionScenarioID = "execution-timeout"
)

func validateExecutionScenarioIDs(values []string) error {
	if len(values) < 1 || len(values) > maximumExecutionScenarioIDs {
		return fmt.Errorf("scenarioIds は一件以上八件以下でなければなりません")
	}
	previous := ""
	for index, value := range values {
		if !isExecutionScenarioID(ExecutionScenarioID(value)) {
			return fmt.Errorf("scenarioIds に未定義の値があります")
		}
		if index > 0 && previous >= value {
			return fmt.Errorf("scenarioIds は昇順で重複なく保持してください")
		}
		previous = value
	}
	return nil
}

func isExecutionScenarioID(value ExecutionScenarioID) bool {
	for _, scenarioID := range executionScenarioIDs() {
		if value == scenarioID {
			return true
		}
	}
	return false
}

func executionScenarioIDs() []ExecutionScenarioID {
	return []ExecutionScenarioID{
		ExecutionScenarioIDAllFailed,
		ExecutionScenarioIDEmpty,
		ExecutionScenarioIDItemBudget,
		ExecutionScenarioIDMixedComposition,
		ExecutionScenarioIDNonempty,
		ExecutionScenarioIDPartialFailure,
		ExecutionScenarioIDReversedCompletion,
		ExecutionScenarioIDTimeout,
	}
}
