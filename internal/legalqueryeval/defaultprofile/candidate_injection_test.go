package defaultprofile

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestEvaluatorは候補専用Planningを明示注入できる(t *testing.T) {
	const verificationID = "candidate-evaluation-deterministic-replay"

	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		t.Fatalf("%s: 候補 profile set を構成できません: %v", verificationID, err)
	}
	evaluator, err := NewWithPlanning(candidate)
	if err != nil {
		t.Fatalf("%s: 候補 planning を evaluator へ注入できません: %v", verificationID, err)
	}
	identity, err := evaluator.Identity()
	if err != nil {
		t.Fatalf("%s: evaluator identity を取得できません: %v", verificationID, err)
	}
	if identity.ProfileSetVersion() != candidate.Profiles().ProfileVersion() ||
		identity.RankingVersion() != candidate.Profiles().RankingVersion() {
		t.Fatalf(
			"%s: evaluator identity = %q/%q, candidate = %q/%q",
			verificationID,
			identity.ProfileSetVersion(),
			identity.RankingVersion(),
			candidate.Profiles().ProfileVersion(),
			candidate.Profiles().RankingVersion(),
		)
	}
}

func TestEvaluatorは候補Execution不一致をReport失敗へ集約する(t *testing.T) {
	const verificationID = "candidate-evaluation-outcome-exit-semantics"

	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		t.Fatalf("%s: 候補 profile set を構成できません: %v", verificationID, err)
	}
	evaluator, err := NewWithPlanning(candidate)
	if err != nil {
		t.Fatalf("%s: 候補 evaluator を構成できません: %v", verificationID, err)
	}
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v10",
	)
	if err != nil {
		t.Fatalf("%s: corpus-v10 を読み込めません: %v", verificationID, err)
	}
	report, err := evaluator.EvaluateExecution(context.Background(), corpus)
	if err != nil {
		t.Fatalf("%s: execution 不一致を report に集約できません: %v", verificationID, err)
	}
	if err := report.Validate(); err != nil {
		t.Fatalf("%s: execution report が不正です: %v", verificationID, err)
	}
	if report.CaseCount() != len(corpus.Execution()) {
		t.Fatalf("%s: execution case count = %d", verificationID, report.CaseCount())
	}
	executionCase := corpus.Execution()[0]
	unexecutable, err := unexecutableExecutionCaseEvaluation(executionCase)
	if err != nil {
		t.Fatalf("%s: 実行不能 case を評価失敗へ変換できません: %v", verificationID, err)
	}
	if unexecutable.ExpectedMatched() || unexecutable.AttemptOrderMatched() ||
		unexecutable.AttemptOrderViolationCount() !=
			len(executionCase.Expected().Attempts()) {
		t.Fatalf("%s: 実行不能 case の評価値が不正です", verificationID)
	}
}
