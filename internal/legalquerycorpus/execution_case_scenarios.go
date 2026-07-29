package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

type executionScenarioSummary struct {
	terminal            ExecutionExpectedTerminal
	status              legalquery.LegalQueryResultStatus
	successCount        int
	readSuccessCount    int
	collectionCount     int
	failureCount        int
	timeoutCount        int
	publishedItemCount  int
	allZeroCollections  bool
	reversedMeanings    bool
	truncatedCollection bool
}

func validateExecutionScenarios(
	scenarioIDs []string,
	actions []ExecutionAction,
	expected ExecutionExpected,
) error {
	summary := summarizeExecutionScenarios(actions, expected)
	for _, scenarioID := range scenarioIDs {
		var valid bool
		switch ExecutionScenarioID(scenarioID) {
		case ExecutionScenarioIDAllFailed:
			valid = summary.failureCount == len(actions) &&
				summary.terminal == ExecutionExpectedTerminalError
		case ExecutionScenarioIDEmpty:
			valid = summary.allZeroCollections &&
				summary.terminal == ExecutionExpectedTerminalResult &&
				summary.status == legalquery.LegalQueryResultStatusEmpty
		case ExecutionScenarioIDItemBudget:
			valid = summary.truncatedCollection
		case ExecutionScenarioIDMixedComposition:
			valid = summary.readSuccessCount > 0 &&
				summary.collectionCount > 0 &&
				summary.failureCount == 0 &&
				summary.terminal == ExecutionExpectedTerminalResult
		case ExecutionScenarioIDNonempty:
			valid = summary.publishedItemCount > 0 &&
				summary.terminal == ExecutionExpectedTerminalResult
		case ExecutionScenarioIDPartialFailure:
			valid = summary.successCount > 0 &&
				summary.failureCount > 0 &&
				summary.terminal == ExecutionExpectedTerminalResult &&
				summary.status == legalquery.LegalQueryResultStatusPartial
		case ExecutionScenarioIDReversedCompletion:
			valid = summary.reversedMeanings
		case ExecutionScenarioIDTimeout:
			valid = summary.timeoutCount > 0
		}
		if !valid {
			return fmt.Errorf("execution scenario の局所必要条件を満たしていません")
		}
	}
	return nil
}

func summarizeExecutionScenarios(
	actions []ExecutionAction,
	expected ExecutionExpected,
) executionScenarioSummary {
	summary := executionScenarioSummary{
		terminal:           expected.Terminal(),
		allZeroCollections: true,
		reversedMeanings:   hasReversedMeaningCompletion(actions),
	}
	if result, ok := expected.(ExecutionExpectedResult); ok {
		summary.status = result.Status()
	}
	attempts := expected.Attempts()
	for index, action := range actions {
		summarizeExecutionScenarioAction(
			&summary,
			action.Outcome(),
			attempts[index],
		)
	}
	return summary
}

func summarizeExecutionScenarioAction(
	summary *executionScenarioSummary,
	outcome ExecutionOutcome,
	attempt ExpectedAttempt,
) {
	switch typed := outcome.(type) {
	case CollectionSuccessOutcome:
		summary.successCount++
		summary.collectionCount++
		if typed.SourceItemCount() != 0 {
			summary.allZeroCollections = false
		}
		summarizeCollectionScenario(summary, typed, attempt)
	case ReadSuccessOutcome:
		summary.successCount++
		summary.readSuccessCount++
		summary.publishedItemCount++
		summary.allZeroCollections = false
	case FailureOutcome:
		summary.failureCount++
		summary.allZeroCollections = false
	case TimeoutOutcome:
		summary.failureCount++
		summary.timeoutCount++
		summary.allZeroCollections = false
	}
}

func summarizeCollectionScenario(
	summary *executionScenarioSummary,
	outcome CollectionSuccessOutcome,
	attempt ExpectedAttempt,
) {
	completed, ok := attempt.(ExpectedCompletedCollectionAttempt)
	if !ok {
		return
	}
	summary.publishedItemCount += completed.PublishedItemCount()
	if outcome.SourceItemCount() > completed.PublishedItemCount() &&
		completed.HasMore() {
		summary.truncatedCollection = true
	}
}

func hasReversedMeaningCompletion(actions []ExecutionAction) bool {
	meanings := make(map[string]struct{}, 2)
	for _, action := range actions {
		meanings[action.MeaningID()] = struct{}{}
	}
	if len(meanings) != 2 {
		return false
	}
	for left := 0; left < len(actions); left++ {
		for right := left + 1; right < len(actions); right++ {
			if actions[left].MeaningID() != actions[right].MeaningID() &&
				actions[left].ReleaseOrder() > actions[right].ReleaseOrder() {
				return true
			}
		}
	}
	return false
}
