package legalquery

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecutorValidatesRequiredAndOptionalFacades(t *testing.T) {
	t.Parallel()

	_, core, judicial := newExecutorTestFacades(t)
	if _, err := NewExecutor(nil, nil); err == nil {
		t.Fatal("SOT-ARCH-022: nil core facade を受理しました")
	}
	var typedNilCore *executorCoreFacadeFake
	if _, err := NewExecutor(typedNilCore, nil); err == nil {
		t.Fatal("SOT-ENG-025: typed nil core facade を受理しました")
	}
	var typedNilJudicial *executorJudicialFacadeFake
	if _, err := NewExecutor(core, typedNilJudicial); err == nil {
		t.Fatal("SOT-ENG-025: typed nil judicial facade を受理しました")
	}
	coreCause := errors.New("core facade の起動時検証に失敗しました")
	core.validateErr = coreCause
	if _, err := NewExecutor(core, nil); err == nil || !errors.Is(err, coreCause) {
		t.Fatalf("core facade の検証原因を保持しませんでした: %v", err)
	}
	core.validateErr = nil
	judicialCause := errors.New("judicial facade の起動時検証に失敗しました")
	judicial.validateErr = judicialCause
	if _, err := NewExecutor(core, judicial); err == nil ||
		!errors.Is(err, judicialCause) {
		t.Fatalf("judicial facade の検証原因を保持しませんでした: %v", err)
	}

	executor, err := NewExecutor(core, nil)
	if err != nil {
		t.Fatalf("optional judicial facade なしで Executor を作成できません: %v", err)
	}
	if err := executor.Validate(); err != nil {
		t.Fatalf("core-only Executor が有効ではありません: %v", err)
	}
}

func TestExecutorFailsClosedWhenJudicialStepHasNoFacade(t *testing.T) {
	t.Parallel()

	recorder, core, _ := newExecutorTestFacades(t)
	executor, err := NewExecutor(core, nil)
	if err != nil {
		t.Fatalf("core-only Executor を作成できません: %v", err)
	}
	plan := mustExecutorSinglePlan(
		t,
		mustExecutorStep(t, "裁判例検索", "step-judicial"),
	)

	execution, err := executor.Execute(context.Background(), plan)
	if err == nil {
		t.Fatal("SOT-ARCH-023: judicial facade なしで司法 step を実行しました")
	}
	if execution.Validate() == nil {
		t.Fatal("SOT-ARCH-023: configuration error と execution を同時に返しました")
	}
	if recorder.callCount() != 0 {
		t.Fatalf("SOT-ARCH-023: configuration error 後の call count = %d", recorder.callCount())
	}
}

func TestExecutorRejectsNonExecutingDecisionsWithoutCalls(t *testing.T) {
	t.Parallel()

	tests := []PlanDecision{
		PlanDecisionNeedsClarification,
		PlanDecisionCapabilityUnavailable,
		PlanDecisionUnsupported,
	}
	for _, decision := range tests {
		decision := decision
		t.Run(string(decision), func(t *testing.T) {
			t.Parallel()
			recorder, core, judicial := newExecutorTestFacades(t)
			executor, err := NewExecutor(core, judicial)
			if err != nil {
				t.Fatalf("Executor を作成できません: %v", err)
			}
			execution, err := executor.Execute(
				context.Background(),
				mustExecutorNonExecutingPlan(t, decision),
			)
			if err == nil {
				t.Fatalf("SOT-ARCH-023: decision=%s を Execute が受理しました", decision)
			}
			if execution.Validate() == nil {
				t.Fatal("非実行 decision で execution を返しました")
			}
			if recorder.callCount() != 0 {
				t.Fatalf("非実行 decision の call count = %d", recorder.callCount())
			}
		})
	}
}

func TestExecutorRejectsInvalidPlanAndBudgetBeforeCalls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*testing.T, *LegalQueryPlan)
	}{
		{
			name: "selected が二候補を超える",
			mutate: func(t *testing.T, plan *LegalQueryPlan) {
				t.Helper()
				third := mustExecutorCandidate(
					t,
					"candidate-3",
					580,
					mustExecutorStep(t, "更新一覧", "step-third"),
				)
				selection, err := NewLegalQueryPlanSelection(
					LegalQueryPlanSelectionValues{
						CandidateID:   third.CandidateID(),
						Availability:  SelectionAvailabilityAvailable,
						RequiredPacks: third.RequiredPacks(),
					},
				)
				if err != nil {
					t.Fatalf("試験用 selection を作成できません: %v", err)
				}
				plan.rankedCandidates = append(plan.rankedCandidates, third)
				plan.selected = append(plan.selected, selection)
			},
		},
		{
			name: "capability call が四件を超える",
			mutate: func(t *testing.T, plan *LegalQueryPlan) {
				t.Helper()
				plan.rankedCandidates[0].steps = append(
					plan.rankedCandidates[0].steps,
					mustExecutorStep(t, "法令検索", "step-fifth"),
				)
			},
		},
		{
			name: "step budget の対応が異なる",
			mutate: func(t *testing.T, plan *LegalQueryPlan) {
				t.Helper()
				plan.budget.stepBudgets[0].stepID = "different-step"
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			recorder, core, judicial := newExecutorTestFacades(t)
			executor, err := NewExecutor(core, judicial)
			if err != nil {
				t.Fatalf("Executor を作成できません: %v", err)
			}
			plan := mustExecutorHedgedPlan(
				t,
				[]LegalQueryCandidateStep{
					mustExecutorStep(t, "法令検索", "step-first"),
					mustExecutorStep(t, "法令本文検索", "step-second"),
				},
				[]LegalQueryCandidateStep{
					mustExecutorStep(t, "法令読取り", "step-third"),
					mustExecutorStep(t, "条文読取り", "step-fourth"),
				},
			)
			test.mutate(t, &plan)

			if _, err := executor.Execute(context.Background(), plan); err == nil {
				t.Fatal("SOT-MODEL-023: 不正な plan を受理しました")
			}
			if recorder.callCount() != 0 {
				t.Fatalf("不正な plan の call count = %d", recorder.callCount())
			}
		})
	}
}

func TestExecutorRejectsNilAndCancelledContext(t *testing.T) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	plan := mustExecutorSinglePlan(
		t,
		mustExecutorStep(t, "法令検索", "step-search"),
	)

	//nolint:staticcheck // SOT-ARCH-023: executor 境界の nil context 拒否を確認する。
	if _, err := executor.Execute(nil, plan); err == nil {
		t.Fatal("SOT-ARCH-023: nil context を受理しました")
	}
	var typedNil *executorTestContext
	if _, err := executor.Execute(typedNil, plan); err == nil {
		t.Fatal("SOT-ENG-025: typed nil context を受理しました")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := executor.Execute(cancelled, plan); !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ARCH-023: cancelled context error = %v", err)
	}
	if recorder.callCount() != 0 {
		t.Fatalf("無効 context の call count = %d", recorder.callCount())
	}
}

func TestExecutorPropagatesCancellationToRunningStep(t *testing.T) {
	t.Parallel()

	recorder, core, judicial := newExecutorTestFacades(t)
	started := make(chan struct{})
	recorder.hook = func(
		ctx context.Context,
		_ LogicalInputKind,
		_ LegalQueryStepBudget,
	) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}
	executor, err := NewExecutor(core, judicial)
	if err != nil {
		t.Fatalf("Executor を作成できません: %v", err)
	}
	plan := mustExecutorSinglePlan(
		t,
		mustExecutorStep(t, "法令検索", "step-search"),
	)
	root, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, executeErr := executor.Execute(root, plan)
		done <- executeErr
	}()
	awaitExecutorSignal(t, started, "facade call が開始されませんでした")
	cancel()
	if executeErr := awaitExecutorError(t, done); !errors.Is(
		executeErr,
		context.Canceled,
	) {
		t.Fatalf("SOT-ARCH-023: cancellation error = %v", executeErr)
	}
}

func TestExecutorZeroValueFailsClosed(t *testing.T) {
	t.Parallel()

	var executor Executor
	if err := executor.Validate(); err == nil {
		t.Fatal("SOT-ENG-025: zero-value Executor を有効と判定しました")
	}
	plan := mustExecutorSinglePlan(
		t,
		mustExecutorStep(t, "法令検索", "step-search"),
	)
	if execution, err := executor.Execute(
		context.Background(),
		plan,
	); err == nil || execution.Validate() == nil {
		t.Fatal("SOT-ENG-025: zero-value Executor が execution を返しました")
	}
}

type executorTestContext struct{}

func (*executorTestContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (*executorTestContext) Done() <-chan struct{} { return nil }
func (*executorTestContext) Err() error            { return nil }
func (*executorTestContext) Value(any) any         { return nil }

func mustExecutorNonExecutingPlan(
	t *testing.T,
	decision PlanDecision,
) LegalQueryPlan {
	t.Helper()
	values := LegalQueryPlanValues{
		ProfileVersion:  "executor-non-executing-v1",
		Decision:        decision,
		LimitPerAttempt: 10,
	}
	switch decision {
	case PlanDecisionNeedsClarification:
		values.RankedCandidates = []LegalQueryCandidate{
			mustExecutorCandidate(
				t,
				"candidate-1",
				100,
				mustExecutorStep(t, "法令検索", "step-search"),
			),
		}
		values.Selected = []LegalQueryPlanSelection{}
		values.ReasonCodes = []ReasonCode{ReasonCodeBelowExecutionThreshold}
	case PlanDecisionCapabilityUnavailable:
		candidate := mustExecutorCandidate(
			t,
			"candidate-1",
			600,
			mustExecutorStep(t, "裁判例検索", "step-judicial"),
		)
		selection, err := NewLegalQueryPlanSelection(
			LegalQueryPlanSelectionValues{
				CandidateID:   candidate.CandidateID(),
				Availability:  SelectionAvailabilityPackDisabled,
				RequiredPacks: candidate.RequiredPacks(),
			},
		)
		if err != nil {
			t.Fatalf("試験用 disabled selection を作成できません: %v", err)
		}
		values.RankedCandidates = []LegalQueryCandidate{candidate}
		values.Selected = []LegalQueryPlanSelection{selection}
		values.ReasonCodes = []ReasonCode{ReasonCodeRequiredPackDisabled}
	case PlanDecisionUnsupported:
		values.RankedCandidates = []LegalQueryCandidate{}
		values.Selected = []LegalQueryPlanSelection{}
		values.ReasonCodes = []ReasonCode{ReasonCodeUnsupportedTaskOrResource}
	default:
		t.Fatalf("非実行 decision が定義されていません: %q", decision)
	}
	plan, err := NewLegalQueryPlan(values)
	if err != nil {
		t.Fatalf("試験用 non-executing plan を作成できません: %v", err)
	}
	return plan
}
