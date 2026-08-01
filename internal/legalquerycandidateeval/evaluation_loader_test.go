package legalquerycandidateeval

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLoadCurrentEvaluationは未評価とReplay履歴を同じ入口で返す(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	current, err := LoadCurrentEvaluation(context.Background(), root, &recordingReferenceValidator{})
	if err != nil {
		t.Fatalf("未評価 current を読めません: %v", err)
	}
	if current.Prepared.Request.EvaluationID != request.EvaluationID || current.CurrentResult != nil || len(current.History) != 0 {
		t.Fatalf("未評価 current の状態が不正です: %+v", current)
	}

	requestRaw := mustCanonicalJSON(t, request)
	reportRaw := syntheticEvaluationReportRaw("tracked-replay")
	result := mustSyntheticEvaluationResult(
		t,
		request,
		requestRaw,
		reportRaw,
		EvaluationOutcomeFailed,
	)
	writeCandidateFixture(t, root, filepath.Join(
		"testdata/legalquery/candidate-evaluations/results",
		request.EvaluationID+".json",
	), mustCanonicalJSON(t, result))
	writeCandidateFixture(t, root, filepath.Join(
		"testdata/legalquery/candidate-evaluations/failed-reports",
		request.EvaluationID+".json",
	), reportRaw)
	current, err = LoadCurrentEvaluation(context.Background(), root, &recordingReferenceValidator{})
	if err != nil {
		t.Fatalf("tracked replay current を読めません: %v", err)
	}
	if current.CurrentResult == nil || *current.CurrentResult != result || len(current.History) != 1 {
		t.Fatalf("tracked replay 履歴が一致しません: %+v", current)
	}
}

func TestLoadCurrentEvaluationはResultのRequestBinding変更を拒否する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	result := mustSyntheticEvaluationResult(
		t,
		request,
		mustCanonicalJSON(t, request),
		syntheticEvaluationReportRaw("changed-binding"),
		EvaluationOutcomePassed,
	)
	result.RequestSHA256 = repeatHex('f')
	writeCandidateFixture(t, root, filepath.Join(
		"testdata/legalquery/baselines/versions",
		request.BaselineVersion+".json",
	), syntheticEvaluationReportRaw("changed-binding"))
	writeCandidateFixture(t, root, filepath.Join(
		"testdata/legalquery/candidate-evaluations/results",
		request.EvaluationID+".json",
	), mustCanonicalJSON(t, result))
	if _, err := LoadCurrentEvaluation(context.Background(), root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("requestSha256 が一致しない tracked result を受理しました")
	}
}

func TestLoadCurrentEvaluationはOutcome別ReportBindingを事前検証する(t *testing.T) {
	t.Parallel()

	t.Run("failed report missing", func(t *testing.T) {
		root := t.TempDir()
		request := prepareCandidateEvaluationFixture(t, root)
		result := mustSyntheticEvaluationResult(
			t,
			request,
			mustCanonicalJSON(t, request),
			syntheticEvaluationReportRaw("missing-failed-report"),
			EvaluationOutcomeFailed,
		)
		writeCandidateFixture(t, root, filepath.Join(
			"testdata/legalquery/candidate-evaluations/results",
			request.EvaluationID+".json",
		), mustCanonicalJSON(t, result))
		if _, err := LoadCurrentEvaluation(context.Background(), root, &recordingReferenceValidator{}); err == nil {
			t.Fatal("failed result に対応する report の欠落を受理しました")
		}
	})

	t.Run("passed baseline", func(t *testing.T) {
		root := t.TempDir()
		request := prepareCandidateEvaluationFixture(t, root)
		reportRaw := syntheticEvaluationReportRaw("passed-baseline")
		result := mustSyntheticEvaluationResult(
			t,
			request,
			mustCanonicalJSON(t, request),
			reportRaw,
			EvaluationOutcomePassed,
		)
		writeCandidateFixture(t, root, filepath.Join(
			"testdata/legalquery/candidate-evaluations/results",
			request.EvaluationID+".json",
		), mustCanonicalJSON(t, result))
		writeCandidateFixture(t, root, filepath.Join(
			"testdata/legalquery/baselines/versions",
			request.BaselineVersion+".json",
		), reportRaw)
		if _, err := LoadCurrentEvaluation(context.Background(), root, &recordingReferenceValidator{}); err != nil {
			t.Fatalf("一致する passed baseline binding を拒否しました: %v", err)
		}
		writeCandidateFixture(t, root, filepath.Join(
			"testdata/legalquery/candidate-evaluations/failed-reports",
			request.EvaluationID+".json",
		), reportRaw)
		if _, err := LoadCurrentEvaluation(context.Background(), root, &recordingReferenceValidator{}); err == nil {
			t.Fatal("passed result と同名 failed report の併存を受理しました")
		}
	})
}
