package defaultprofile

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestEvaluatorは内蔵DefaultProfileで代表Holdoutを評価する(t *testing.T) {
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

	holdoutByID := make(map[string]legalquerycorpus.SemanticCase)
	for _, semanticCase := range corpus.Holdout() {
		holdoutByID[semanticCase.CaseID()] = semanticCase
	}
	caseIDs := []string{
		"holdout-input-01",
		"holdout-input-18",
		"holdout-intent-01",
		"holdout-pack-01",
		"holdout-pack-11",
		"holdout-pack-08",
	}
	for _, caseID := range caseIDs {
		semanticCase, exists := holdoutByID[caseID]
		if !exists {
			t.Fatalf("fixture %q が corpus-v4 にありません", caseID)
		}
		t.Run(caseID, func(t *testing.T) {
			evaluation, err := evaluator.Evaluate(
				context.Background(),
				semanticCase,
			)
			if err != nil {
				t.Fatalf("Evaluate() error = %v", err)
			}
			switch semanticCase.Expected().Kind() {
			case legalquerycorpus.SemanticExpectedKindPlan:
				if !evaluation.PlanOutcomeMatched() {
					request, requestErr := productRequest(semanticCase.Request())
					if requestErr != nil {
						t.Fatalf("診断用 request を構築できません: %v", requestErr)
					}
					plan, planErr := evaluator.selectPlan(
						context.Background(),
						semanticCase,
						request,
					)
					if planErr != nil {
						t.Fatalf("診断用 plan を構築できません: %v", planErr)
					}
					t.Fatalf(
						"decision、reason または selection が一致しません: decision=%q reasons=%#v selected=%#v ranked=%#v",
						plan.Decision(),
						plan.ReasonCodes(),
						plan.Selected(),
						plan.RankedCandidates(),
					)
				}
				for _, meaning := range evaluation.Meanings() {
					if !meaning.SignatureMatched() {
						t.Fatalf("meaning %q の意味署名が一致しません", meaning.MeaningID())
					}
					if matched, applicable := meaning.EvidenceAssertion(); applicable && !matched {
						t.Fatalf("meaning %q の根拠 assertion が一致しません", meaning.MeaningID())
					}
					if matched, applicable := meaning.ConceptAssertion(); applicable && !matched {
						t.Fatalf("meaning %q の法概念 assertion が一致しません", meaning.MeaningID())
					}
				}
			case legalquerycorpus.SemanticExpectedKindRequestError:
				if !evaluation.RequestErrorMatched() {
					t.Fatal("request error の code または field が一致しません")
				}
			default:
				t.Fatalf("expected.kind = %q は未対応です", semanticCase.Expected().Kind())
			}
		})
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test file path を取得できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}
