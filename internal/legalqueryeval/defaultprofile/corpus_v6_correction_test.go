package defaultprofile

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
)

func TestEvaluatorはCorpusV6で予算統合caseの法概念根拠期待値を訂正する(
	t *testing.T,
) {
	// SOT-ENG-024: corpus-v5 の当該 fixture だけが、profile 横断合成後も
	// 保持される legal_concept 根拠と concept source を欠いていたことを
	// 独立 review で確認した。
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}
	for _, caseID := range corpusV6BudgetCorrectionCaseIDs() {
		previous := loadCorpusBudgetCorrectionCase(t, "corpus-v5", caseID)
		corrected := loadCorpusBudgetCorrectionCase(t, "corpus-v6", caseID)
		previousEvaluation, err := evaluator.Evaluate(context.Background(), previous)
		if err != nil {
			t.Fatalf("%s の corpus-v5 Evaluate() error = %v", caseID, err)
		}
		correctedEvaluation, err := evaluator.Evaluate(context.Background(), corrected)
		if err != nil {
			t.Fatalf("%s の corpus-v6 Evaluate() error = %v", caseID, err)
		}
		assertCorpusV6PreCorrectionMismatch(t, caseID, previousEvaluation)
		assertCorpusV6MeaningEvaluation(t, caseID, correctedEvaluation)
	}
}

func loadCorpusBudgetCorrectionCase(
	t *testing.T,
	version string,
	caseID string,
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
		if semanticCase.CaseID() == caseID {
			return semanticCase
		}
	}
	t.Fatalf("%s に %s がありません", version, caseID)
	return legalquerycorpus.SemanticCase{}
}

func corpusV6BudgetCorrectionCaseIDs() []string {
	return []string{
		"holdout-budget-01-explicit-date",
		"holdout-budget-02",
		"holdout-budget-11",
		"holdout-budget-12",
		"holdout-budget-13",
		"holdout-budget-14",
		"holdout-budget-16",
	}
}

func assertCorpusV6MeaningEvaluation(
	t *testing.T,
	caseID string,
	evaluation legalqueryeval.SemanticCaseEvaluation,
) {
	t.Helper()
	if len(evaluation.Meanings()) != 1 {
		t.Fatalf("SOT-ENG-024: %s の meaning 評価件数 = %d", caseID, len(evaluation.Meanings()))
	}
	meaning := evaluation.Meanings()[0]
	evidenceMatched, evidenceApplicable := meaning.EvidenceAssertion()
	conceptMatched, conceptApplicable := meaning.ConceptAssertion()
	if !meaning.SignatureMatched() ||
		!evidenceApplicable ||
		!evidenceMatched ||
		!conceptApplicable ||
		!conceptMatched {
		t.Fatalf(
			"SOT-ENG-024: %s の meaning %q 評価が一致しません",
			caseID,
			meaning.MeaningID(),
		)
	}
}

func assertCorpusV6PreCorrectionMismatch(
	t *testing.T,
	caseID string,
	evaluation legalqueryeval.SemanticCaseEvaluation,
) {
	t.Helper()
	if len(evaluation.Meanings()) != 1 {
		t.Fatalf("SOT-ENG-024: %s の訂正前 meaning 評価件数 = %d", caseID, len(evaluation.Meanings()))
	}
	meaning := evaluation.Meanings()[0]
	evidenceMatched, evidenceApplicable := meaning.EvidenceAssertion()
	conceptMatched, conceptApplicable := meaning.ConceptAssertion()
	if !meaning.SignatureMatched() {
		t.Fatalf(
			"SOT-ENG-024: %s の訂正前 meaning %q 署名が崩れました",
			caseID,
			meaning.MeaningID(),
		)
	}
	if evidenceApplicable && evidenceMatched &&
		conceptApplicable && conceptMatched {
		t.Fatalf(
			"SOT-ENG-024: %s の訂正前 meaning %q assertion が誤って一致しました",
			caseID,
			meaning.MeaningID(),
		)
	}
}
