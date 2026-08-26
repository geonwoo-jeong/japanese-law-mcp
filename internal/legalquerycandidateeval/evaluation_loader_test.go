package legalquerycandidateeval

import (
	"context"
	"path/filepath"
	"testing"
)

type staleReferenceValidator struct{}

func (staleReferenceValidator) ValidateEvaluatorVersion(string) error {
	return nil
}

type staleThenFatalReferenceValidator struct {
	unknownReason bool
}

func (staleThenFatalReferenceValidator) ValidateEvaluatorVersion(string) error {
	return nil
}

func (staleThenFatalReferenceValidator) ValidateCandidateContent(
	context.Context,
	[]byte,
	CandidateContentManifest,
) error {
	return NewCurrentStaleError(StaleReasonCandidateContentDrift)
}

func (v staleThenFatalReferenceValidator) ValidateEvaluationRequest(
	_ context.Context,
	_ []byte,
	document EvaluationRequest,
) (RequestReferenceValidation, error) {
	if v.unknownReason {
		return RequestReferenceValidation{
			CurrentRequiredReviewSOTs: append([]SOTReference(nil), document.RequiredReviewSOTs...),
			StaleReasons:              []StaleReason{"unknown_reason"},
		}, nil
	}
	return RequestReferenceValidation{}, errRejectedReference
}

func (staleReferenceValidator) ValidateCandidateContent(
	context.Context,
	[]byte,
	CandidateContentManifest,
) error {
	return NewCurrentStaleError(StaleReasonCandidateContentDrift)
}

func (staleReferenceValidator) ValidateEvaluationRequest(
	_ context.Context,
	_ []byte,
	document EvaluationRequest,
) (RequestReferenceValidation, error) {
	return RequestReferenceValidation{
		CurrentRequiredReviewSOTs: append([]SOTReference(nil), document.RequiredReviewSOTs...),
	}, NewCurrentStaleError(StaleReasonReviewSOTLifecycleDrift)
}

func TestInspectCurrentEvaluationは完全性を保ったStaleを隔離する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	inspection, err := InspectCurrentEvaluation(
		context.Background(),
		root,
		staleReferenceValidator{},
	)
	if err != nil {
		t.Fatalf("candidate-evaluation-stale-product-quality-pass: stale current の完全性検査に失敗しました: %v", err)
	}
	if inspection.ReadinessState() != CurrentReadinessStale ||
		inspection.Evaluation().Prepared.Request.EvaluationID != request.EvaluationID ||
		!EqualStaleReasons(inspection.StaleReasons(), []StaleReason{
			StaleReasonCandidateContentDrift,
			StaleReasonReviewSOTLifecycleDrift,
		}) {
		t.Fatalf("candidate-evaluation-stale-current-repository-integrity: inspection=%+v reasons=%v", inspection.Evaluation(), inspection.StaleReasons())
	}
	first := inspection.Evaluation()
	originalRequestByte := first.RequestRaw[0]
	originalHoldoutDigest := first.Prepared.Request.HoldoutLeakageGroupDigests[0]
	originalProfileID := first.Prepared.CandidateContent.ProfileArtifacts[0].ProfileID
	first.RequestRaw[0] ^= 0xff
	first.Prepared.Request.HoldoutLeakageGroupDigests[0] = "changed"
	first.Prepared.CandidateContent.ProfileArtifacts[0].ProfileID = "changed"
	reasons := inspection.StaleReasons()
	reasons[0] = StaleReasonCurrentEvaluatorDrift
	second := inspection.Evaluation()
	if second.RequestRaw[0] != originalRequestByte ||
		second.Prepared.Request.HoldoutLeakageGroupDigests[0] != originalHoldoutDigest ||
		second.Prepared.CandidateContent.ProfileArtifacts[0].ProfileID != originalProfileID ||
		inspection.StaleReasons()[0] != StaleReasonCandidateContentDrift {
		t.Fatal("candidate-evaluation-stale-current-repository-integrity: inspection の不変値を共有しました")
	}

	if _, err := LoadCurrentEvaluation(
		context.Background(),
		root,
		staleReferenceValidator{},
	); !IsCurrentStale(err) {
		t.Fatalf("candidate-evaluation-stale-strict-loader-rejection: stale current を strict loader が受理しました: %v", err)
	}
}

func TestInspectCurrentEvaluationはStaleでArtifact改変を隠さない(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	writeCandidateFixture(
		t,
		root,
		"testdata/legalquery/candidate-evaluations/schema-v2.json",
		[]byte("{}\n"),
	)
	if _, err := InspectCurrentEvaluation(
		context.Background(),
		root,
		staleReferenceValidator{},
	); err == nil || IsCurrentStale(err) {
		t.Fatalf("candidate-evaluation-stale-does-not-mask-artifact-corruption: schema 改変を stale に変換しました: %v", err)
	}
}

func TestInspectCurrentEvaluationはStaleよりFatalと未知理由を優先する(t *testing.T) {
	t.Parallel()

	for name, validator := range map[string]ReferenceValidator{
		"readiness error":      staleThenFatalReferenceValidator{},
		"unknown stale reason": staleThenFatalReferenceValidator{unknownReason: true},
	} {
		name, validator := name, validator
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			prepareCandidateEvaluationFixture(t, root)
			if _, err := InspectCurrentEvaluation(
				context.Background(),
				root,
				validator,
			); err == nil || IsCurrentStale(err) {
				t.Fatalf("candidate-evaluation-stale-does-not-mask-artifact-corruption: error=%v", err)
			}
		})
	}
}

func TestLoadCurrentEvaluationは未評価とReplay履歴を同じ入口で返す(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	pendingValidator := &recordingReferenceValidator{}
	current, err := LoadCurrentEvaluation(context.Background(), root, pendingValidator)
	if err != nil {
		t.Fatalf("未評価 current を読めません: %v", err)
	}
	if current.Prepared.Request.EvaluationID != request.EvaluationID || current.CurrentResult != nil || len(current.History) != 0 {
		t.Fatalf("未評価 current の状態が不正です: %+v", current)
	}
	if pendingValidator.manifestCalls != 1 || pendingValidator.requestCalls != 1 {
		t.Fatalf(
			"未評価 current の外部参照検証 = manifest:%d request:%d",
			pendingValidator.manifestCalls,
			pendingValidator.requestCalls,
		)
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
	replayValidator := &recordingReferenceValidator{reject: true}
	current, err = LoadCurrentEvaluation(context.Background(), root, replayValidator)
	if err != nil {
		t.Fatalf("tracked replay current を読めません: %v", err)
	}
	if current.CurrentResult == nil || *current.CurrentResult != result || len(current.History) != 1 {
		t.Fatalf("tracked replay 履歴が一致しません: %+v", current)
	}
	if replayValidator.manifestCalls != 0 || replayValidator.requestCalls != 0 {
		t.Fatalf(
			"tracked replay が現在の外部参照を再検証しました: manifest:%d request:%d",
			replayValidator.manifestCalls,
			replayValidator.requestCalls,
		)
	}
	if replayValidator.evaluatorVersionCalls != 1 ||
		len(replayValidator.evaluatorVersions) != 1 ||
		replayValidator.evaluatorVersions[0] != request.EvaluatorVersion {
		t.Fatalf(
			"tracked replay の evaluator version 検証 = %v",
			replayValidator.evaluatorVersions,
		)
	}

	rejectedVersion := &recordingReferenceValidator{
		rejectEvaluatorVersion: true,
	}
	if _, err := LoadCurrentEvaluation(
		context.Background(),
		root,
		rejectedVersion,
	); err == nil {
		t.Fatal("tracked replay が未対応 evaluator version を受理しました")
	}
	if rejectedVersion.manifestCalls != 0 ||
		rejectedVersion.requestCalls != 0 ||
		rejectedVersion.evaluatorVersionCalls != 1 {
		t.Fatalf(
			"tracked replay の閉じた version 検証回数が不正です: %+v",
			rejectedVersion,
		)
	}
}

func TestLoadCurrentEvaluationはReport前に継承されたRequestを保持して新Currentを返す(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	fixture := prepareSupersededCandidateEvaluationFixture(t, root, "default-3")
	validator := &recordingReferenceValidator{}
	current, err := LoadCurrentEvaluation(context.Background(), root, validator)
	if err != nil {
		t.Fatalf("report 前に継承された current evaluation を拒否しました: %v", err)
	}
	if current.Prepared.Request.EvaluationID != fixture.currentRequest.EvaluationID ||
		current.Prepared.CandidateContent.CandidateContentID != fixture.currentManifest.CandidateContentID ||
		current.CurrentResult != nil || len(current.History) != 0 {
		t.Fatalf("継承後の current evaluation が不正です: %+v", current)
	}
	if fixture.previousRequest.BaselineVersion != "default-2" ||
		fixture.currentRequest.BaselineVersion != "default-3" {
		t.Fatalf(
			"継承前後の baseline 予約が不正です: previous=%q current=%q",
			fixture.previousRequest.BaselineVersion,
			fixture.currentRequest.BaselineVersion,
		)
	}
	if validator.manifestCalls != 1 || validator.requestCalls != 1 ||
		len(validator.manifestIDs) != 1 || validator.manifestIDs[0] != fixture.currentManifest.CandidateContentID ||
		len(validator.requestIDs) != 1 || validator.requestIDs[0] != fixture.currentRequest.EvaluationID {
		t.Fatalf(
			"external validator が current 以外を検証しました: manifests=%v requests=%v",
			validator.manifestIDs,
			validator.requestIDs,
		)
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
