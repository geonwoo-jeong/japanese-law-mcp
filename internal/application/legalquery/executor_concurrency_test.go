package legalquery

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutorRunsStepsSeriallyWithinCandidate(t *testing.T) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	recorder.hook = func(
		_ context.Context,
		kind LogicalInputKind,
		_ LegalQueryStepBudget,
	) error {
		switch kind {
		case InputKindLawSearch:
			close(firstStarted)
			<-releaseFirst
		case InputKindLawContentSearch:
			close(secondStarted)
		}
		return nil
	}
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	plan := mustExecutorSinglePlan(
		t,
		mustExecutorStep(t, "法令検索", "step-first"),
		mustExecutorStep(t, "法令本文検索", "step-second"),
	)
	done := make(chan error, 1)
	go func() {
		_, executeErr := executor.Execute(context.Background(), plan)
		done <- executeErr
	}()

	awaitExecutorSignal(t, firstStarted, "一番目の step が開始されませんでした")
	select {
	case <-secondStarted:
		t.Fatal("SOT-ARCH-023: 同じ候補の二番目の step を並列に開始しました")
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseFirst)
	if executeErr := awaitExecutorError(t, done); executeErr != nil {
		t.Fatalf("SOT-ARCH-023: Execute error = %v", executeErr)
	}
	calls := recorder.callsSnapshot()
	if len(calls) != 2 ||
		calls[0].kind != InputKindLawSearch ||
		calls[1].kind != InputKindLawContentSearch {
		t.Fatalf("SOT-ARCH-023: candidate 内の呼出し順 = %#v", calls)
	}
}

func TestExecutorRunsTwoCandidatesInParallelAndKeepsPlanOrder(
	t *testing.T,
) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	secondCompleted := make(chan struct{})
	releaseFirst := make(chan struct{})
	recorder.hook = func(
		_ context.Context,
		kind LogicalInputKind,
		_ LegalQueryStepBudget,
	) error {
		switch kind {
		case InputKindLawSearch:
			close(firstStarted)
			<-releaseFirst
		case InputKindLawContentSearch:
			close(secondStarted)
			close(secondCompleted)
		}
		return nil
	}
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	plan := mustExecutorHedgedPlan(
		t,
		[]LegalQueryCandidateStep{
			mustExecutorStep(t, "法令検索", "step-first"),
		},
		[]LegalQueryCandidateStep{
			mustExecutorStep(t, "法令本文検索", "step-second"),
		},
	)
	type executionResult struct {
		execution LegalQueryExecution
		err       error
	}
	done := make(chan executionResult, 1)
	go func() {
		execution, executeErr := executor.Execute(context.Background(), plan)
		done <- executionResult{execution: execution, err: executeErr}
	}()

	awaitExecutorSignal(t, firstStarted, "第一候補が開始されませんでした")
	awaitExecutorSignal(t, secondStarted, "第二候補を第一候補と並列に開始しませんでした")
	awaitExecutorSignal(t, secondCompleted, "第二候補が先に完了しませんでした")
	close(releaseFirst)
	result := awaitExecutorExecution(t, done)
	if result.err != nil {
		t.Fatalf("SOT-ARCH-023: Execute error = %v", result.err)
	}
	attempts := result.execution.Attempts()
	if len(attempts) != 2 ||
		attempts[0].StepID() != "step-first" ||
		attempts[1].StepID() != "step-second" {
		t.Fatalf(
			"SOT-ARCH-023/025: reverse completion 後の attempt 順 = %#v",
			attempts,
		)
	}
}

func TestExecutorPassesExactBudgetsAndSameDerivedContext(t *testing.T) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	steps := []LegalQueryCandidateStep{
		mustExecutorStep(t, "法令検索", "step-search"),
		mustExecutorStep(t, "法令読取り", "step-read"),
		mustExecutorStep(t, "法令本文検索", "step-content"),
		mustExecutorStep(t, "更新一覧", "step-updates"),
	}
	candidate := mustExecutorCandidate(t, "candidate-1", 600, steps...)
	plan := mustExecutorPlan(
		t,
		PlanDecisionSingle,
		[]LegalQueryCandidate{candidate},
		20,
	)
	type executorContextKey struct{}
	root := context.WithValue(
		context.Background(),
		executorContextKey{},
		"root の値",
	)

	if _, err := executor.Execute(root, plan); err != nil {
		t.Fatalf("SOT-MODEL-023: Execute error = %v", err)
	}
	calls := recorder.callsSnapshot()
	if len(calls) != len(steps) {
		t.Fatalf("SOT-MODEL-023: facade call count = %d", len(calls))
	}
	expectedStepIDs := []string{
		"step-search",
		"step-read",
		"step-content",
		"step-updates",
	}
	for index, call := range calls {
		if call.budget.CandidateID() != "candidate-1" ||
			call.budget.StepID() != expectedStepIDs[index] {
			t.Fatalf(
				"SOT-MODEL-023: calls[%d] budget = (%q, %q)",
				index,
				call.budget.CandidateID(),
				call.budget.StepID(),
			)
		}
		if call.ctx == root ||
			call.ctx.Value(executorContextKey{}) != "root の値" {
			t.Fatalf("SOT-ARCH-023: calls[%d] context が root から派生していません", index)
		}
		if index > 0 && call.ctx != calls[0].ctx {
			t.Fatalf("SOT-ARCH-023: calls[%d] に異なる実行 context を渡しました", index)
		}
		limit, hasLimit := call.budget.EffectiveLimit()
		if call.kind == InputKindLawRead {
			if call.budget.ReservedItems() != 1 || hasLimit {
				t.Fatalf("SOT-MODEL-023: read budget = %#v", call.budget)
			}
			continue
		}
		if call.budget.ReservedItems() != 0 ||
			!hasLimit ||
			limit != 13 {
			t.Fatalf(
				"SOT-MODEL-023: collection budget = (%d, %d, %t)",
				call.budget.ReservedItems(),
				limit,
				hasLimit,
			)
		}
	}
}

func TestExecutorDoesNotStartNextSiblingStepAfterFatalCancellation(
	t *testing.T,
) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	firstStarted := make(chan struct{})
	secondStepStarted := make(chan struct{})
	fatalCause := errors.New("第二候補の materialization に失敗しました")
	recorder.errs[InputKindLawRead] = fatalCause
	recorder.hook = func(
		ctx context.Context,
		kind LogicalInputKind,
		_ LegalQueryStepBudget,
	) error {
		switch kind {
		case InputKindLawSearch:
			close(firstStarted)
			<-ctx.Done()
		case InputKindLawRead:
			<-firstStarted
		case InputKindLawContentSearch:
			close(secondStepStarted)
		}
		return nil
	}
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	plan := mustExecutorHedgedPlan(
		t,
		[]LegalQueryCandidateStep{
			mustExecutorStep(t, "法令検索", "step-first"),
			mustExecutorStep(t, "法令本文検索", "step-must-not-start"),
		},
		[]LegalQueryCandidateStep{
			mustExecutorStep(t, "法令読取り", "step-fatal"),
		},
	)

	if _, err := executor.Execute(context.Background(), plan); !errors.Is(
		err,
		fatalCause,
	) {
		t.Fatalf("fatal cause を保持しませんでした: %v", err)
	}
	select {
	case <-secondStepStarted:
		t.Fatal("SOT-ARCH-023: fatal cancellation 後に次の sibling step を開始しました")
	default:
	}
	if recorder.callCount() != 2 {
		t.Fatalf(
			"SOT-ARCH-023: fatal cancellation 後の call count = %d",
			recorder.callCount(),
		)
	}
}

func awaitExecutorSignal(
	t *testing.T,
	signal <-chan struct{},
	message string,
) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal(message)
	}
}

func awaitExecutorError(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("Executor が完了しませんでした")
		return nil
	}
}

func awaitExecutorExecution[T any](
	t *testing.T,
	result <-chan T,
) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("Executor が完了しませんでした")
		var zero T
		return zero
	}
}
