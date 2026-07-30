package defaultprofile

import (
	"context"
	"testing"
)

func TestEvaluatorはCorpusV7で複数Stepの正式名称根拠期待値を訂正する(
	t *testing.T,
) {
	// SOT-ARCH-029, SOT-ENG-024: corpus-v6 の holdout-budget-15 は、
	// 独立した法令名検索を成立させる official_alias を欠いていた。
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}
	previous := loadCorpusBudgetCorrectionCase(
		t,
		"corpus-v6",
		"holdout-budget-15",
	)
	corrected := loadCorpusBudgetCorrectionCase(
		t,
		"corpus-v7",
		"holdout-budget-15",
	)
	previousEvaluation, err := evaluator.Evaluate(context.Background(), previous)
	if err != nil {
		t.Fatalf("corpus-v6 Evaluate() error = %v", err)
	}
	correctedEvaluation, err := evaluator.Evaluate(context.Background(), corrected)
	if err != nil {
		t.Fatalf("corpus-v7 Evaluate() error = %v", err)
	}
	assertCorpusV6PreCorrectionMismatch(
		t,
		"holdout-budget-15",
		previousEvaluation,
	)
	assertCorpusV6MeaningEvaluation(
		t,
		"holdout-budget-15",
		correctedEvaluation,
	)
}
