package defaultprofile

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
)

func TestEvaluatorはCorpusV9で法令名と法概念の期待値を訂正する(
	t *testing.T,
) {
	// SOT-ENG-022, SOT-ENG-023, SOT-ENG-024: corpus-v8 の五 fixture は、
	// 明示 task、同一意味の誤記根拠または法概念出典の期待値を誤っていた。
	evaluator, err := New()
	if err != nil {
		t.Fatalf("default profile evaluator を構築できません: %v", err)
	}
	previous := loadCorpusV9CorrectionCases(t, "corpus-v8")
	corrected := loadCorpusV9CorrectionCases(t, "corpus-v9")
	for _, caseID := range corpusV9CorrectionCaseIDs() {
		previousEvaluation, err := evaluator.Evaluate(
			context.Background(),
			previous[caseID],
		)
		if err != nil {
			t.Fatalf("%s の corpus-v8 Evaluate() error = %v", caseID, err)
		}
		correctedEvaluation, err := evaluator.Evaluate(
			context.Background(),
			corrected[caseID],
		)
		if err != nil {
			t.Fatalf("%s の corpus-v9 Evaluate() error = %v", caseID, err)
		}
		assertCorpusV9PreCorrectionMismatch(t, caseID, previousEvaluation)
		assertCorpusV9CorrectedMeaning(t, caseID, correctedEvaluation)
	}
}

func loadCorpusV9CorrectionCases(
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
	for _, caseID := range corpusV9CorrectionCaseIDs() {
		if _, exists := result[caseID]; !exists {
			t.Fatalf("%s に %s がありません", version, caseID)
		}
	}
	return result
}

func assertCorpusV9PreCorrectionMismatch(
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
	conceptMatched, conceptApplicable := meaning.ConceptAssertion()
	if !meaning.SignatureMatched() {
		t.Fatalf(
			"SOT-ENG-024: %s の訂正前 meaning %q 署名が崩れました",
			caseID,
			meaning.MeaningID(),
		)
	}
	if (!evidenceApplicable || evidenceMatched) &&
		(!conceptApplicable || conceptMatched) {
		t.Fatalf(
			"SOT-ENG-024: %s の訂正前 assertion が誤って一致しました",
			caseID,
		)
	}
}

func assertCorpusV9CorrectedMeaning(
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

func corpusV9CorrectionCaseIDs() []string {
	return []string{
		"holdout-name-10",
		"holdout-name-11",
		"holdout-name-15",
		"holdout-typo-08",
		"holdout-typo-10",
	}
}
