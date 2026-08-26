package legalquerycandidateprepare

import (
	"context"
	"os"
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
		context.Background(), root, "testdata/legalquery/corpus-v16",
	)
	if err != nil {
		t.Fatalf("candidate-evaluation-request-identity: corpus manifest を読めません: %v", err)
	}
	references := fixedHistoricalSOTReferencesForTest(
		t,
		legalquerycandidateeval.SchemaVersionV3,
	)
	manifest := corpus.Manifest()
	request := legalquerycandidateeval.EvaluationRequest{
		SchemaVersion:              legalquerycandidateeval.SchemaVersionV3,
		EvaluatorVersion:           evaluators.CurrentVersion,
		CorpusVersion:              manifest.CorpusVersion(),
		CorpusManifestSHA256:       corpus.SHA256(),
		HoldoutDigest:              manifest.HoldoutDigest(),
		HoldoutLeakageGroupDigests: manifest.HoldoutLeakageGroupDigests(),
		RequiredReviewSOTs:         references,
		RequiredReviewSOTSetSHA256: legalquerycandidateeval.SOTSetSHA256(references),
		BaselineVersion:            "default-8",
	}
	validation, err := validator.ValidateEvaluationRequest(
		context.Background(), []byte("canonical request placeholder\n"), request,
	)
	if reasons, stale := legalquerycandidateeval.StaleReasonsFromError(err); !stale || len(validation.CurrentRequiredReviewSOTs) != 0 ||
		!legalquerycandidateeval.EqualStaleReasons(
			reasons,
			[]legalquerycandidateeval.StaleReason{
				legalquerycandidateeval.StaleReasonReviewSOTLifecycleDrift,
			},
		) {
		t.Fatalf("candidate-evaluation-stale-candidate-readiness-fail: 外部参照結果=(%#v,%v)", validation, err)
	}

	request.EvaluatorVersion = evaluators.Version2
	if _, err := validator.ValidateEvaluationRequest(
		context.Background(), []byte("canonical request placeholder\n"), request,
	); err == nil {
		t.Fatal("candidate-evaluation-evaluator-version-match: current ではない v2 を新規準備として受理しました")
	}

	request.EvaluatorVersion = evaluators.CurrentVersion
	request.BaselineVersion = "default-1"
	if _, err := validator.ValidateEvaluationRequest(
		context.Background(), []byte("canonical request placeholder\n"), request,
	); err == nil {
		t.Fatal("candidate-evaluation-request-identity: active baseline 版を候補予約として受理しました")
	}

	request.BaselineVersion = "default-8"
	request.EvaluatorVersion = "legal-query-evaluator-v999"
	if _, err := validator.ValidateEvaluationRequest(
		context.Background(), []byte("canonical request placeholder\n"), request,
	); err == nil {
		t.Fatal("candidate-evaluation-evaluator-version-match: 未知 evaluator 版を受理しました")
	}
}

func TestReferenceValidatorはSchemaV2を新しいCurrentとして拒否する(t *testing.T) {
	t.Parallel()

	validator := ReferenceValidator{repositoryRoot: candidateRepositoryRoot(t)}
	request := legalquerycandidateeval.EvaluationRequest{
		SchemaVersion:    legalquerycandidateeval.SchemaVersionV2,
		EvaluatorVersion: evaluators.Version3,
	}
	if _, err := validator.ValidateEvaluationRequest(
		context.Background(),
		nil,
		request,
	); err == nil {
		t.Fatal("candidate-evaluation-schema-v3-exact-evaluator-binding: schema v2 を新しい current request として受理しました")
	}
}

func TestReferenceValidatorは実際の候補評価Treeを状態対応Loaderで再読込できる(t *testing.T) {
	if !useExactCandidateToolchain(t) {
		t.Skip("候補再現用 Go 環境がないため local では実行しません")
	}

	const historyEvaluationID = "evaluation-sha256-398e801b2d7edd6068f36fa34fe94827d7d44891d59976fdc8630e4d5be7e89c"

	root := candidateRepositoryRoot(t)
	validator, err := NewReferenceValidator(root)
	if err != nil {
		t.Fatalf("candidate-evaluation-current-single-target: validator を作成できません: %v", err)
	}
	inspection, err := legalquerycandidateeval.InspectCurrentEvaluation(
		context.Background(),
		root,
		validator,
	)
	if err != nil {
		t.Fatalf("candidate-evaluation-stale-current-repository-integrity: 実際の候補評価 tree の完全性を検証できません: %v", err)
	}
	if inspection.ReadinessState() != legalquerycandidateeval.CurrentReadinessStale ||
		!legalquerycandidateeval.EqualStaleReasons(
			inspection.StaleReasons(),
			[]legalquerycandidateeval.StaleReason{
				legalquerycandidateeval.StaleReasonCandidateContentDrift,
				legalquerycandidateeval.StaleReasonReviewSOTLifecycleDrift,
			},
		) {
		t.Fatalf("candidate-evaluation-stale-product-quality-pass: readiness=%q reasons=%v", inspection.ReadinessState(), inspection.StaleReasons())
	}
	if _, err := legalquerycandidateeval.LoadCurrentEvaluation(
		context.Background(),
		root,
		validator,
	); !legalquerycandidateeval.IsCurrentStale(err) {
		t.Fatalf("candidate-evaluation-stale-strict-loader-rejection: strict loader error=%v", err)
	}
	current := inspection.Evaluation()
	prepared := current.Prepared
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
	if prepared.Pointer.SchemaVersion != legalquerycandidateeval.SchemaVersionV3 ||
		prepared.Request.SchemaVersion != legalquerycandidateeval.SchemaVersionV3 ||
		prepared.CandidateContent.SchemaVersion != legalquerycandidateeval.SchemaVersionV3 ||
		prepared.Request.EvaluatorVersion != evaluators.Version3 {
		t.Fatalf("candidate-evaluation-current-single-target: current schema binding = %#v", prepared)
	}
	if len(current.History) != 1 {
		t.Fatalf("candidate-evaluation-single-holdout-use: 消費済み履歴数 = %d", len(current.History))
	}
	if current.CurrentResult != nil {
		t.Fatalf("candidate-evaluation-current-single-target: 未評価 current に result があります: %#v", current.CurrentResult)
	}
	history := current.History[0]
	if history.Request.EvaluationID != historyEvaluationID ||
		history.Request.BaselineVersion != "default-3" ||
		history.Result.EvaluationID != historyEvaluationID ||
		history.Result.Outcome != legalquerycandidateeval.EvaluationOutcomeFailed {
		t.Fatalf("candidate-evaluation-failure-history: 消費済み履歴 = %#v", history)
	}
	if prepared.Request.CorpusVersion != "corpus-v16" ||
		prepared.Request.BaselineVersion != "default-8" {
		t.Fatalf("candidate-evaluation-request-identity: current request = %#v", prepared.Request)
	}
	if len(current.RequestRaw) == 0 ||
		len(current.CurrentResultRaw) != 0 ||
		len(current.CurrentReportRaw) != 0 {
		t.Fatal("candidate-evaluation-current-single-target: 未評価 current の原 byte 状態が不正です")
	}
}

func Test不確定終了Requestを再利用せず新しいCurrentへ進める(t *testing.T) {
	if !useExactCandidateToolchain(t) {
		t.Skip("候補再現用 Go 環境がないため local では実行しません")
	}

	const indeterminateEvaluationID = "evaluation-sha256-2f8790cd9a969372660571031ed00069565443521ca840cdce9ef86fb1290c42"
	root := candidateRepositoryRoot(t)
	validator, err := NewReferenceValidator(root)
	if err != nil {
		t.Fatalf("candidate-evaluation-indeterminate-reviewed-retry-gate: validator を作成できません: %v", err)
	}
	inspection, err := legalquerycandidateeval.InspectCurrentEvaluation(context.Background(), root, validator)
	if err != nil {
		t.Fatalf("candidate-evaluation-indeterminate-reviewed-retry-gate: current の完全性を検証できません: %v", err)
	}
	current := inspection.Evaluation()
	oldPath := filepath.Join(
		root, "testdata", "legalquery", "candidate-evaluations", "requests",
		indeterminateEvaluationID+".json",
	)
	//nolint:gosec // SOT-ENG-040: 固定した置換済み request の通常 file だけを読む。
	oldRaw, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatalf("candidate-evaluation-indeterminate-reviewed-retry-gate: 旧 request を読めません: %v", err)
	}
	oldRequest, err := legalquerycandidateeval.DecodeEvaluationRequest(oldRaw)
	if err != nil {
		t.Fatalf("candidate-evaluation-indeterminate-reviewed-retry-gate: 旧 request が不正です: %v", err)
	}
	request := current.Prepared.Request
	if current.Prepared.Pointer.EvaluationID == indeterminateEvaluationID ||
		oldRequest.BaselineVersion != "default-4" ||
		request.BaselineVersion != "default-8" ||
		oldRequest.HoldoutDigest == request.HoldoutDigest {
		t.Fatal("candidate-evaluation-indeterminate-reviewed-retry-gate: 不確定終了の identity を再利用しました")
	}
	oldGroups := make(map[string]struct{}, len(oldRequest.HoldoutLeakageGroupDigests))
	for _, digest := range oldRequest.HoldoutLeakageGroupDigests {
		oldGroups[digest] = struct{}{}
	}
	for _, digest := range request.HoldoutLeakageGroupDigests {
		if _, reused := oldGroups[digest]; reused {
			t.Fatal("candidate-evaluation-indeterminate-reviewed-retry-gate: 不確定終了の leakage group digest を再利用しました")
		}
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
