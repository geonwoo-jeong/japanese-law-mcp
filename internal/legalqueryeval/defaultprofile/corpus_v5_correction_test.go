package defaultprofile

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
)

func TestEvaluatorはCorpusV5で曖昧な略称誤記の選択期待値を訂正する(
	t *testing.T,
) {
	// SOT-ENG-024: corpus-v4 の当該 fixture だけが、同じ別名衝突の
	// 二候補を selected へ保持する共通規則と一致しないことを独立 review で確認した。
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}
	previous := loadCorpusV5CorrectionCase(t, "corpus-v4")
	corrected := loadCorpusV5CorrectionCase(t, "corpus-v5")
	previousEvaluation, err := evaluator.Evaluate(context.Background(), previous)
	if err != nil {
		t.Fatalf("corpus-v4 の Evaluate() error = %v", err)
	}
	correctedEvaluation, err := evaluator.Evaluate(context.Background(), corrected)
	if err != nil {
		t.Fatalf("corpus-v5 の Evaluate() error = %v", err)
	}
	if previousEvaluation.PlanOutcomeMatched() {
		t.Fatal("SOT-ENG-024: corpus-v4 の誤った selection が一致しました")
	}
	if !correctedEvaluation.PlanOutcomeMatched() {
		t.Fatal("SOT-ENG-024: corpus-v5 の訂正済み selection が一致しません")
	}
	assertCorpusV5MeaningEvaluation(t, previousEvaluation)
	assertCorpusV5MeaningEvaluation(t, correctedEvaluation)
}

func loadCorpusV5CorrectionCase(
	t *testing.T,
	version string,
) legalquerycorpus.SemanticCase {
	t.Helper()
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/"+version,
	)
	if err != nil {
		t.Fatalf("%s を読み込めません: %v", version, err)
	}
	for _, semanticCase := range corpus.Holdout() {
		if semanticCase.CaseID() == "holdout-typo-15" {
			return semanticCase
		}
	}
	t.Fatalf("%s に holdout-typo-15 がありません", version)
	return legalquerycorpus.SemanticCase{}
}

func assertCorpusV5MeaningEvaluation(
	t *testing.T,
	evaluation legalqueryeval.SemanticCaseEvaluation,
) {
	t.Helper()
	if len(evaluation.Meanings()) != 2 {
		t.Fatalf("SOT-ENG-024: meaning 評価件数 = %d", len(evaluation.Meanings()))
	}
	for _, meaning := range evaluation.Meanings() {
		evidenceMatched, evidenceApplicable := meaning.EvidenceAssertion()
		if !meaning.SignatureMatched() ||
			!evidenceApplicable ||
			!evidenceMatched {
			t.Fatalf(
				"SOT-ENG-024: meaning %q の評価が一致しません",
				meaning.MeaningID(),
			)
		}
	}
}
