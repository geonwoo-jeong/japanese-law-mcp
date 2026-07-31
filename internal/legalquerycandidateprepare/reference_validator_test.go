package legalquerycandidateprepare

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/evaluators"
)

func TestReferenceValidatorはRequestの外部参照をManifestだけで再検証する(t *testing.T) {
	t.Parallel()

	root := candidateRepositoryRoot(t)
	validator, err := NewReferenceValidator(root)
	if err != nil {
		t.Fatalf("candidate-evaluation-referenced-file-bounds: validator を作成できません: %v", err)
	}
	corpus, err := legalquerycorpus.LoadManifest(
		context.Background(), root, "testdata/legalquery/corpus-v10",
	)
	if err != nil {
		t.Fatalf("candidate-evaluation-request-identity: corpus manifest を読めません: %v", err)
	}
	references, err := BuildRequiredSOTReferences(context.Background(), root)
	if err != nil {
		t.Fatalf("candidate-evaluation-review-content-binding: SOT 参照を作れません: %v", err)
	}
	manifest := corpus.Manifest()
	request := legalquerycandidateeval.EvaluationRequest{
		EvaluatorVersion:           evaluators.Version1,
		CorpusVersion:              manifest.CorpusVersion(),
		CorpusManifestSHA256:       corpus.SHA256(),
		HoldoutDigest:              manifest.HoldoutDigest(),
		HoldoutLeakageGroupDigests: manifest.HoldoutLeakageGroupDigests(),
		RequiredReviewSOTs:         references,
		RequiredReviewSOTSetSHA256: legalquerycandidateeval.SOTSetSHA256(references),
		BaselineVersion:            "default-2",
	}
	if _, err := validator.ValidateEvaluationRequest(
		context.Background(), []byte("canonical request placeholder\n"), request,
	); err != nil {
		t.Fatalf("candidate-evaluation-request-identity: 外部参照を検証できません: %v", err)
	}

	request.EvaluatorVersion = "legal-query-evaluator-v999"
	if _, err := validator.ValidateEvaluationRequest(
		context.Background(), []byte("canonical request placeholder\n"), request,
	); err == nil {
		t.Fatal("candidate-evaluation-evaluator-version-match: 未知 evaluator 版を受理しました")
	}
}

func TestReferenceValidatorは実際の候補評価準備Treeをloaderで再読込できる(t *testing.T) {
	t.Parallel()

	root := candidateRepositoryRoot(t)
	validator, err := NewReferenceValidator(root)
	if err != nil {
		t.Fatalf("candidate-evaluation-current-single-target: validator を作成できません: %v", err)
	}
	prepared, err := legalquerycandidateeval.LoadPreparedCurrent(
		context.Background(),
		root,
		validator,
	)
	if err != nil {
		t.Fatalf("candidate-evaluation-current-single-target: 実際の候補評価準備 tree を再読込できません: %v", err)
	}
	if prepared.Pointer.EvaluationID == "" ||
		prepared.Request.EvaluationID == "" ||
		prepared.CandidateContent.CandidateContentID == "" {
		t.Fatalf("candidate-evaluation-current-single-target: current tree の主要 ID が空です: %#v", prepared)
	}
	if prepared.Pointer.EvaluationID != prepared.Request.EvaluationID {
		t.Fatalf("candidate-evaluation-current-single-target: pointer=%q request=%q", prepared.Pointer.EvaluationID, prepared.Request.EvaluationID)
	}
	if len(prepared.ReviewAttestations) != 2 {
		t.Fatalf("candidate-evaluation-current-single-target: review 数 = %d", len(prepared.ReviewAttestations))
	}
}

func TestReferenceValidatorは不正Rootと取消を拒否する(t *testing.T) {
	t.Parallel()

	if _, err := NewReferenceValidator(""); err == nil {
		t.Fatal("candidate-evaluation-referenced-file-bounds: 空 repository root を受理しました")
	}
	validator, err := NewReferenceValidator(candidateRepositoryRoot(t))
	if err != nil {
		t.Fatalf("validator を作成できません: %v", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := validator.ValidateCandidateContent(
		cancelled, nil, legalquerycandidateeval.CandidateContentManifest{},
	); err == nil {
		t.Fatal("candidate-evaluation-referenced-file-bounds: 取消済み検証を受理しました")
	}
}

func candidateRepositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repository root を解決できません: %v", err)
	}
	return root
}
