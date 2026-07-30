package defaultprofile

import (
	"context"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestEvaluatorは安全境界の未対応表現を非実行または明確化にする(
	t *testing.T,
) {
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}

	found := 0
	for _, semanticCase := range corpus.Holdout() {
		if !strings.HasPrefix(semanticCase.CaseID(), "holdout-safety-") &&
			!strings.HasPrefix(
				semanticCase.CaseID(),
				"holdout-unsupported-",
			) {
			continue
		}
		found++
		evaluation, err := evaluator.Evaluate(
			context.Background(),
			semanticCase,
		)
		if err != nil {
			t.Fatalf("%s を評価できません: %v", semanticCase.CaseID(), err)
		}
		if !evaluation.PlanOutcomeMatched() {
			t.Errorf(
				"SOT-ENG-024: %s の安全な plan outcome が一致しません",
				semanticCase.CaseID(),
			)
		}
	}
	if found == 0 {
		t.Fatal("安全境界 fixture がありません")
	}
}
