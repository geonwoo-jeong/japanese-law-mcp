package legalqueryeval

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestEvaluateSemanticHoldoutは全件をmanifest順で集計する(t *testing.T) {
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		semanticRunnerRepositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	holdout := corpus.Holdout()

	t.Run("全件の順序と集計結果を保持する", func(t *testing.T) {
		visited := make([]string, 0, len(holdout))
		result, err := EvaluateSemanticHoldout(
			context.Background(),
			corpus,
			func(
				_ context.Context,
				semanticCase legalquerycorpus.SemanticCase,
			) (SemanticCaseEvaluation, error) {
				visited = append(visited, semanticCase.CaseID())
				return matchedSemanticEvaluation(semanticCase), nil
			},
		)
		if err != nil {
			t.Fatalf("EvaluateSemanticHoldout() error = %v", err)
		}

		wantOrder := make([]string, 0, len(holdout))
		for _, semanticCase := range holdout {
			wantOrder = append(wantOrder, semanticCase.CaseID())
		}
		if !slices.Equal(visited, wantOrder) {
			t.Fatalf("評価順 = %#v, want %#v", visited, wantOrder)
		}
		if got := result.Metrics().CaseCount(); got != len(holdout) {
			t.Fatalf("case count = %d, want %d", got, len(holdout))
		}

		evaluations := result.Evaluations()
		if len(evaluations) != len(holdout) {
			t.Fatalf("evaluation count = %d, want %d", len(evaluations), len(holdout))
		}
		for index, evaluation := range evaluations {
			if evaluation.CaseID() != wantOrder[index] {
				t.Fatalf(
					"evaluations[%d].CaseID() = %q, want %q",
					index,
					evaluation.CaseID(),
					wantOrder[index],
				)
			}
		}

		evaluations[0] = SemanticCaseEvaluation{}
		if got := result.Evaluations()[0].CaseID(); got != wantOrder[0] {
			t.Fatalf("結果の複製を書き換えた後の先頭 caseId = %q, want %q", got, wantOrder[0])
		}
	})

	t.Run("context取消後は次のcaseを評価しない", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		_, err := EvaluateSemanticHoldout(
			ctx,
			corpus,
			func(
				_ context.Context,
				semanticCase legalquerycorpus.SemanticCase,
			) (SemanticCaseEvaluation, error) {
				calls++
				cancel()
				return matchedSemanticEvaluation(semanticCase), nil
			},
		)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
		if calls != 1 {
			t.Fatalf("evaluator call count = %d, want 1", calls)
		}
	})

	t.Run("異なるcaseIdの評価結果を拒否する", func(t *testing.T) {
		_, err := EvaluateSemanticHoldout(
			context.Background(),
			corpus,
			func(
				_ context.Context,
				semanticCase legalquerycorpus.SemanticCase,
			) (SemanticCaseEvaluation, error) {
				evaluation := matchedSemanticEvaluation(semanticCase)
				evaluation.caseID = holdout[1].CaseID()
				return evaluation, nil
			},
		)
		if err == nil {
			t.Fatal("異なる caseId の評価結果を拒否しませんでした")
		}
	})
}

func matchedSemanticEvaluation(
	semanticCase legalquerycorpus.SemanticCase,
) SemanticCaseEvaluation {
	evaluation := SemanticCaseEvaluation{
		caseID:       semanticCase.CaseID(),
		categoryIDs:  semanticCase.CategoryIDs(),
		coverageIDs:  semanticCase.CoverageIDs(),
		expectedKind: semanticCase.Expected().Kind(),
		initialized:  true,
	}
	switch evaluation.expectedKind {
	case legalquerycorpus.SemanticExpectedKindPlan:
		evaluation.planOutcomeMatched = true
	case legalquerycorpus.SemanticExpectedKindRequestError:
		evaluation.requestErrorMatched = true
	}
	return evaluation
}

func semanticRunnerRepositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test file path を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
