package legalquerycorpus

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestExecutionCaseは入力とgetterを深く複製する(t *testing.T) {
	t.Parallel()

	values := validExecutionCaseValuesForTest(t)
	scenarios := values.ScenarioIDs
	actions := values.Actions
	executionCase, err := NewExecutionCase(values)
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewExecutionCase() error = %v", err)
	}
	scenarios[0] = "execution-empty"
	actions[0].meaningID = "changed"
	inputOutcome := actions[0].outcome.(CollectionSuccessOutcome)
	inputOutcome.sourceItemCount = 0
	actions[0].outcome = inputOutcome
	inputExpected := values.Expected.(ExecutionExpectedResult)
	inputAttempt := inputExpected.attempts[0].(ExpectedCompletedCollectionAttempt)
	inputAttempt.publishedItemCount = 0
	inputExpected.attempts[0] = inputAttempt
	gotActions := executionCase.Actions()
	gotActions[0].meaningID = "changed"
	gotOutcome := gotActions[0].outcome.(CollectionSuccessOutcome)
	gotOutcome.sourceItemCount = 0
	gotActions[0].outcome = gotOutcome
	gotAttempts := executionCase.Expected().Attempts()
	gotAttempt := gotAttempts[0].(ExpectedCompletedCollectionAttempt)
	gotAttempt.publishedItemCount = 0
	gotAttempts[0] = gotAttempt

	if executionCase.ScenarioIDs()[0] != "execution-nonempty" ||
		executionCase.Actions()[0].MeaningID() != "law-search" ||
		executionCase.Actions()[0].Outcome().(CollectionSuccessOutcome).
			SourceItemCount() != 1 ||
		executionCase.Expected().Attempts()[0].(ExpectedCompletedCollectionAttempt).PublishedItemCount() != 1 {
		t.Fatal("SOT-ENG-026: ExecutionCase の共有状態が外部から変更された")
	}
}

func TestExecutionCaseGetterは並行読取りで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	executionCase, err := NewExecutionCase(validExecutionCaseValuesForTest(t))
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewExecutionCase() error = %v", err)
	}
	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for repeat := 0; repeat < 100; repeat++ {
				scenarios := executionCase.ScenarioIDs()
				scenarios[0] = "changed"
				actions := executionCase.Actions()
				actions[0].meaningID = "changed"
				outcome := actions[0].outcome.(CollectionSuccessOutcome)
				outcome.sourceItemCount = 0
				actions[0].outcome = outcome
				attempts := executionCase.Expected().Attempts()
				attempt := attempts[0].(ExpectedCompletedCollectionAttempt)
				attempt.publishedItemCount = 0
				attempts[0] = attempt
			}
		}()
	}
	wait.Wait()
	if executionCase.ScenarioIDs()[0] != "execution-nonempty" ||
		executionCase.Actions()[0].MeaningID() != "law-search" {
		t.Fatal("SOT-ENG-026: 並行 getter が ExecutionCase を変更した")
	}
}

func TestExecutionCaseはJSONから直接復元できない(t *testing.T) {
	t.Parallel()

	if err := json.Unmarshal([]byte(`{}`), &ExecutionCase{}); err == nil {
		t.Fatal("SOT-ENG-026: ExecutionCase を直接 JSON 復元できた")
	}
}
