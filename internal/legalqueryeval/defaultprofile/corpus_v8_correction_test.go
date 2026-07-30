package defaultprofile

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
)

func TestEvaluatorはCorpusV8で独立した形態素検索根拠の期待値を訂正する(
	t *testing.T,
) {
	// SOT-ARCH-029, SOT-ENG-024: corpus-v7 の五 fixture は、
	// 独立した検索 step を成立させる morphological_context を欠いていた。
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}
	previous := loadCorpusV8CorrectionCases(t, "corpus-v7")
	corrected := loadCorpusV8CorrectionCases(t, "corpus-v8")
	for _, caseID := range corpusV8CorrectionCaseIDs() {
		previousEvaluation, err := evaluator.Evaluate(
			context.Background(),
			previous[caseID],
		)
		if err != nil {
			t.Fatalf("%s の corpus-v7 Evaluate() error = %v", caseID, err)
		}
		correctedEvaluation, err := evaluator.Evaluate(
			context.Background(),
			corrected[caseID],
		)
		if err != nil {
			t.Fatalf("%s の corpus-v8 Evaluate() error = %v", caseID, err)
		}
		assertCorpusV8PreCorrectionMismatch(t, caseID, previousEvaluation)
		assertCorpusV8CorrectedMeaning(t, caseID, correctedEvaluation)
	}
}

func loadCorpusV8CorrectionCases(
	t *testing.T,
	version string,
) map[string]legalquerycorpus.SemanticCase {
	t.Helper()
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/"+version,
	)
	if err != nil {
		t.Fatalf("%s を読み込めません: %v", version, err)
	}
	result := make(map[string]legalquerycorpus.SemanticCase)
	for _, semanticCase := range corpus.Holdout() {
		result[semanticCase.CaseID()] = semanticCase
	}
	for _, caseID := range corpusV8CorrectionCaseIDs() {
		if _, exists := result[caseID]; !exists {
			t.Fatalf("%s に %s がありません", version, caseID)
		}
	}
	return result
}

func assertCorpusV8PreCorrectionMismatch(
	t *testing.T,
	caseID string,
	evaluation legalqueryeval.SemanticCaseEvaluation,
) {
	t.Helper()
	if len(evaluation.Meanings()) != 1 {
		t.Fatalf(
			"SOT-ENG-024: %s の訂正前 meaning 評価件数 = %d",
			caseID,
			len(evaluation.Meanings()),
		)
	}
	meaning := evaluation.Meanings()[0]
	evidenceMatched, evidenceApplicable := meaning.EvidenceAssertion()
	if !meaning.SignatureMatched() {
		t.Fatalf(
			"SOT-ENG-024: %s の訂正前 meaning %q 署名が崩れました",
			caseID,
			meaning.MeaningID(),
		)
	}
	if !evidenceApplicable || evidenceMatched {
		t.Fatalf(
			"SOT-ENG-024: %s の訂正前 evidence assertion が不正です",
			caseID,
		)
	}
}

func assertCorpusV8CorrectedMeaning(
	t *testing.T,
	caseID string,
	evaluation legalqueryeval.SemanticCaseEvaluation,
) {
	t.Helper()
	if len(evaluation.Meanings()) != 1 {
		t.Fatalf(
			"SOT-ENG-024: %s の訂正後 meaning 評価件数 = %d",
			caseID,
			len(evaluation.Meanings()),
		)
	}
	meaning := evaluation.Meanings()[0]
	evidenceMatched, evidenceApplicable := meaning.EvidenceAssertion()
	conceptMatched, conceptApplicable := meaning.ConceptAssertion()
	if !meaning.SignatureMatched() ||
		!evidenceApplicable ||
		!evidenceMatched ||
		(conceptApplicable && !conceptMatched) {
		t.Fatalf(
			"SOT-ENG-024: %s の訂正後 meaning %q 評価が一致しません",
			caseID,
			meaning.MeaningID(),
		)
	}
}

func corpusV8CorrectionCaseIDs() []string {
	return []string{
		"holdout-budget-06-explicit-date",
		"holdout-intent-16",
		"holdout-intent-17",
		"holdout-intent-18",
		"holdout-structure-03",
	}
}
