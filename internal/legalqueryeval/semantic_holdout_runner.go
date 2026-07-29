package legalqueryeval

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

// SemanticCaseEvaluationFunc は、一件の semantic case を製品 profile で評価する。
type SemanticCaseEvaluationFunc func(
	context.Context,
	legalquerycorpus.SemanticCase,
) (SemanticCaseEvaluation, error)

// SemanticHoldoutResult は、manifest 順の評価結果と micro 集計を不変に保持する。
type SemanticHoldoutResult struct {
	evaluations []SemanticCaseEvaluation
	metrics     SemanticCaseMetrics
}

// Evaluations は、manifest 順の評価結果の複製を返す。
func (r SemanticHoldoutResult) Evaluations() []SemanticCaseEvaluation {
	return append([]SemanticCaseEvaluation{}, r.evaluations...)
}

// Metrics は、holdout 全体と category 別の micro 集計を返す。
func (r SemanticHoldoutResult) Metrics() SemanticCaseMetrics {
	return r.metrics
}

// EvaluateSemanticHoldout は、SOT-ENG-024 の holdout 全件を manifest 順に評価する。
func EvaluateSemanticHoldout(
	ctx context.Context,
	corpus legalquerycorpus.Corpus,
	evaluate SemanticCaseEvaluationFunc,
) (SemanticHoldoutResult, error) {
	if ctx == nil {
		return SemanticHoldoutResult{}, fmt.Errorf("context は nil にできません")
	}
	if err := corpus.Validate(); err != nil {
		return SemanticHoldoutResult{}, fmt.Errorf("corpus が有効ではありません: %w", err)
	}
	if evaluate == nil {
		return SemanticHoldoutResult{}, fmt.Errorf("semantic case evaluator は nil にできません")
	}

	holdout := corpus.Holdout()
	evaluations := make([]SemanticCaseEvaluation, 0, len(holdout))
	for _, semanticCase := range holdout {
		if err := ctx.Err(); err != nil {
			return SemanticHoldoutResult{}, fmt.Errorf(
				"semantic holdout の評価を中断しました: %w",
				err,
			)
		}
		evaluation, err := evaluate(ctx, semanticCase)
		if err != nil {
			return SemanticHoldoutResult{}, fmt.Errorf(
				"caseId %q の評価に失敗しました: %w",
				semanticCase.CaseID(),
				err,
			)
		}
		if evaluation.CaseID() != semanticCase.CaseID() {
			return SemanticHoldoutResult{}, fmt.Errorf(
				"caseId %q の evaluator が異なる caseId %q を返しました",
				semanticCase.CaseID(),
				evaluation.CaseID(),
			)
		}
		evaluations = append(evaluations, evaluation)
	}

	metrics, err := AggregateSemanticCaseEvaluations(evaluations)
	if err != nil {
		return SemanticHoldoutResult{}, fmt.Errorf(
			"semantic holdout の集計に失敗しました: %w",
			err,
		)
	}
	return SemanticHoldoutResult{
		evaluations: append([]SemanticCaseEvaluation{}, evaluations...),
		metrics:     metrics,
	}, nil
}
