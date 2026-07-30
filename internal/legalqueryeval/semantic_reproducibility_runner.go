package legalqueryeval

import (
	"context"
	"fmt"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

// SemanticCasePlanEvaluationFunc は、一件の評価値と plan の有無を返す。
type SemanticCasePlanEvaluationFunc func(
	context.Context,
	legalquerycorpus.SemanticCase,
) (
	SemanticCaseEvaluation,
	legalquery.LegalQueryPlan,
	bool,
	error,
)

// EvaluateSemanticHoldoutReproducibility は、holdout 全件と plan case の二回目を評価する。
func EvaluateSemanticHoldoutReproducibility(
	ctx context.Context,
	corpus legalquerycorpus.Corpus,
	evaluate SemanticCasePlanEvaluationFunc,
) (SemanticReport, EvaluationMetricReport, error) {
	if ctx == nil {
		return SemanticReport{}, EvaluationMetricReport{},
			fmt.Errorf("context は nil にできません")
	}
	if err := corpus.Validate(); err != nil {
		return SemanticReport{}, EvaluationMetricReport{},
			fmt.Errorf("corpus が有効ではありません: %w", err)
	}
	if evaluate == nil {
		return SemanticReport{}, EvaluationMetricReport{},
			fmt.Errorf("semantic plan evaluator は nil にできません")
	}

	evaluations := make(
		[]SemanticCaseEvaluation,
		0,
		len(corpus.Holdout()),
	)
	reproducible := 0
	planCases := 0
	failedCaseIDs := make([]string, 0)
	for _, semanticCase := range corpus.Holdout() {
		if err := ctx.Err(); err != nil {
			return SemanticReport{}, EvaluationMetricReport{},
				fmt.Errorf("semantic holdout の評価を中断しました: %w", err)
		}
		evaluation, firstPlan, hasPlan, err := evaluate(
			ctx,
			semanticCase,
		)
		if err != nil {
			return SemanticReport{}, EvaluationMetricReport{},
				fmt.Errorf(
					"caseId %q の評価に失敗しました: %w",
					semanticCase.CaseID(),
					err,
				)
		}
		if evaluation.CaseID() != semanticCase.CaseID() {
			return SemanticReport{}, EvaluationMetricReport{},
				fmt.Errorf(
					"caseId %q の evaluator が異なる caseId %q を返しました",
					semanticCase.CaseID(),
					evaluation.CaseID(),
				)
		}
		expectPlan := semanticCase.Expected().Kind() ==
			legalquerycorpus.SemanticExpectedKindPlan
		if hasPlan != expectPlan {
			return SemanticReport{}, EvaluationMetricReport{},
				fmt.Errorf(
					"caseId %q の plan 有無が期待値と一致しません",
					semanticCase.CaseID(),
				)
		}
		evaluations = append(evaluations, evaluation)
		if !hasPlan {
			continue
		}
		if err := firstPlan.Validate(); err != nil {
			return SemanticReport{}, EvaluationMetricReport{},
				fmt.Errorf(
					"caseId %q の plan が有効ではありません: %w",
					semanticCase.CaseID(),
					err,
				)
		}
		planCases++
		secondEvaluation, secondPlan, secondHasPlan, err := evaluate(
			ctx,
			semanticCase,
		)
		if err != nil {
			return SemanticReport{}, EvaluationMetricReport{},
				fmt.Errorf(
					"caseId %q の再評価に失敗しました: %w",
					semanticCase.CaseID(),
					err,
				)
		}
		if !secondHasPlan ||
			secondEvaluation.CaseID() != semanticCase.CaseID() {
			return SemanticReport{}, EvaluationMetricReport{},
				fmt.Errorf(
					"caseId %q の再評価が同じ plan case を返しません",
					semanticCase.CaseID(),
				)
		}
		if err := secondPlan.Validate(); err != nil {
			return SemanticReport{}, EvaluationMetricReport{},
				fmt.Errorf(
					"caseId %q の再評価 plan が有効ではありません: %w",
					semanticCase.CaseID(),
					err,
				)
		}
		if reflect.DeepEqual(firstPlan, secondPlan) {
			reproducible++
			continue
		}
		failedCaseIDs = append(failedCaseIDs, semanticCase.CaseID())
	}

	metrics, err := AggregateSemanticCaseEvaluations(evaluations)
	if err != nil {
		return SemanticReport{}, EvaluationMetricReport{},
			fmt.Errorf("semantic holdout の集計に失敗しました: %w", err)
	}
	semantic, err := NewSemanticReport(SemanticHoldoutResult{
		evaluations: append([]SemanticCaseEvaluation{}, evaluations...),
		metrics:     metrics,
	})
	if err != nil {
		return SemanticReport{}, EvaluationMetricReport{}, err
	}
	reproducibility, err := NewEvaluationMetricReport(
		EvaluationMetricReportValues{
			MetricID:      "plan-reproducibility",
			Numerator:     reproducible,
			Denominator:   planCases,
			FailedCaseIDs: failedCaseIDs,
		},
	)
	if err != nil {
		return SemanticReport{}, EvaluationMetricReport{}, err
	}
	return semantic, reproducibility, nil
}
