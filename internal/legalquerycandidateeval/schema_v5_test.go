package legalquerycandidateeval

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
)

// SOT-ENG-048: 新しい五成果物と exact binding を旧世代から分離する。
func TestSchemaV5は五成果物とExactBindingを分離する(t *testing.T) {
	t.Parallel()
	manifest := validCandidateManifestForSchema(t, SchemaVersionV5)
	request := validEvaluationRequestForSchema(t, manifest)
	result, err := NewEvaluationResult(request, mustCanonicalJSON(t, request), []byte("{}\n"), EvaluationOutcomeFailed)
	if err != nil {
		t.Fatal(err)
	}
	samples := []struct {
		document any
		decode   func([]byte) error
	}{
		{PointerDocument{ArtifactKindPointer, SchemaVersionV5, request.EvaluationID}, func(raw []byte) error { _, err := DecodePointer(raw); return err }},
		{manifest, func(raw []byte) error { _, err := DecodeCandidateContentManifest(raw); return err }},
		{validReviewAttestationForSchema(t, manifest, ReviewScopeArchitecture, "authority-v5"), func(raw []byte) error { _, err := DecodeReviewAttestation(raw); return err }},
		{request, func(raw []byte) error { _, err := DecodeEvaluationRequest(raw); return err }},
		{result, func(raw []byte) error { _, err := DecodeEvaluationResult(raw); return err }},
	}
	for _, sample := range samples {
		raw := mustCanonicalJSON(t, sample.document)
		if err := sample.decode(raw); err != nil {
			t.Fatal(err)
		}
		for _, replacement := range []string{`"schemaVersion":6`, `"schemaVersion":5.0`, `"schemaVersion":null`, `"schemaVersion":5,"schemaVersion":5`, `"schemaVersion":5,"unknown":1`} {
			if err := sample.decode(bytes.Replace(raw, []byte(`"schemaVersion":5`), []byte(replacement), 1)); err == nil {
				t.Fatal("不正な版又は field を受理しました")
			}
		}
	}
	v4, err := RequiredReviewSOTIDsForSchema(SchemaVersionV4)
	if err != nil {
		t.Fatal(err)
	}
	v5, err := RequiredReviewSOTIDsForSchema(SchemaVersionV5)
	if err != nil {
		t.Fatal(err)
	}
	expected := append(slices.Clone(v4), "SOT-ENG-046", "SOT-ENG-047", "SOT-ENG-048", "SOT-ENG-049")
	slices.Sort(expected)
	if !slices.Equal(v5, expected) {
		t.Fatal("v5 exact SOT 集合が一致しません")
	}
	for _, version := range []string{"legal-query-evaluator-v1", "legal-query-evaluator-v2", EvaluatorVersionV3, "legal-query-evaluator-v5", "current"} {
		invalid := request
		invalid.EvaluatorVersion = version
		invalid.EvaluationID = mustEvaluationID(t, invalid)
		if _, err := DecodeEvaluationRequest(mustCanonicalJSON(t, invalid)); err == nil {
			t.Fatal("非 exact evaluator を受理しました")
		}
	}
	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	prepared := writeCandidatePreparationForSchema(t, root, SchemaVersionV5)
	for _, version := range []int{SchemaVersionV2, SchemaVersionV3, SchemaVersionV4} {
		pointer := PointerDocument{ArtifactKindPointer, version, prepared.EvaluationID}
		writeV4Fixture(t, root, "current.json", mustCanonicalJSON(t, pointer))
		_, err := LoadPreparedCurrent(t.Context(), root, &recordingReferenceValidator{})
		requireSchemaVersionError(t, err)
	}
	invalid := request
	invalid.RequiredReviewSOTs = validSOTReferencesForSchema(t, SchemaVersionV4)
	invalid.RequiredReviewSOTSetSHA256 = SOTSetSHA256(invalid.RequiredReviewSOTs)
	invalid.EvaluationID = mustEvaluationID(t, invalid)
	if _, err := DecodeEvaluationRequest(mustCanonicalJSON(t, invalid)); err == nil {
		t.Fatal("旧世代の review 集合を受理しました")
	}
}

// SOT-ENG-048: replay は現在鮮度へ再結合せず、全旧世代の予約を保持する。
func TestSchemaV5のStrictReadyは旧世代Replayと予約を保持する(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	v2 := prepareCandidateEvaluationFixture(t, root)
	previous := []EvaluationRequest{v2, writeCandidatePreparationForSchema(t, root, SchemaVersionV3), writeCandidatePreparationForSchema(t, root, SchemaVersionV4)}
	current := writeCandidatePreparationForSchema(t, root, SchemaVersionV5)
	for _, request := range previous {
		requestRaw := mustCanonicalJSON(t, request)
		reportRaw := syntheticEvaluationReportRaw(fmt.Sprintf("v%d-history", request.SchemaVersion))
		result := mustSyntheticEvaluationResult(t, request, requestRaw, reportRaw, EvaluationOutcomeFailed)
		resultRaw := mustCanonicalJSON(t, result)
		writeV4Fixture(t, root, filepath.Join("results", request.EvaluationID+".json"), resultRaw)
		writeV4Fixture(t, root, filepath.Join("failed-reports", request.EvaluationID+".json"), reportRaw)
		writeV4Pointer(t, root, request)
		validator := &recordingReferenceValidator{reject: true}
		replay, err := LoadCurrentEvaluation(t.Context(), root, validator)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(replay.CurrentResultRaw, resultRaw) || !bytes.Equal(replay.CurrentReportRaw, reportRaw) || validator.manifestCalls != 0 || validator.requestCalls != 0 {
			t.Fatal("旧世代 replay を現在鮮度へ再結合しました")
		}
		for _, field := range []string{"baseline", "holdout", "leakage"} {
			conflict := cloneEvaluationRequest(current)
			switch field {
			case "baseline":
				conflict.BaselineVersion = request.BaselineVersion
			case "holdout":
				conflict.HoldoutDigest = request.HoldoutDigest
			case "leakage":
				conflict.HoldoutLeakageGroupDigests = slices.Clone(request.HoldoutLeakageGroupDigests)
			}
			conflict.EvaluationID = mustEvaluationID(t, conflict)
			if err := checkRequestReservationPreflight(conflict, map[string]loadedArtifact[EvaluationRequest]{request.EvaluationID: {document: request}}); err == nil {
				t.Fatal("旧世代の予約を再利用しました")
			}
		}
	}
	writeV4Pointer(t, root, current)
	inspection, err := InspectCurrentEvaluation(t.Context(), root, &recordingReferenceValidator{})
	if err != nil || inspection.ReadinessState() != CurrentReadinessReady {
		t.Fatalf("合成 ready が失敗しました: %v", err)
	}
	strict, err := LoadCurrentEvaluation(t.Context(), root, &recordingReferenceValidator{})
	if err != nil || strict.Prepared.Request.EvaluationID != current.EvaluationID || len(strict.History) != 3 {
		t.Fatalf("合成 strict が失敗しました: %v", err)
	}
}
