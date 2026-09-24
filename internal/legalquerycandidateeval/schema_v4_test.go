package legalquerycandidateeval

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSchemaV4は五成果物を旧世代から分離する(t *testing.T) {
	t.Parallel()
	manifest := validCandidateManifestForSchema(t, SchemaVersionV4)
	request := validEvaluationRequestForSchema(t, manifest)
	result, err := NewEvaluationResult(request, mustCanonicalJSON(t, request), []byte("{}\n"), EvaluationOutcomeFailed)
	if err != nil || result.SchemaVersion != SchemaVersionV4 {
		t.Fatalf("v4 result を構築できません: %v", err)
	}
	schemas, err := loadCanonicalArtifactSchemas()
	if err != nil {
		t.Fatal(err)
	}
	samples := []struct {
		document any
		decode   func([]byte) error
	}{
		{PointerDocument{ArtifactKindPointer, SchemaVersionV4, request.EvaluationID}, func(raw []byte) error { _, err := DecodePointer(raw); return err }},
		{manifest, func(raw []byte) error { _, err := DecodeCandidateContentManifest(raw); return err }},
		{validReviewAttestationForSchema(t, manifest, ReviewScopeArchitecture, "authority-v4"), func(raw []byte) error { _, err := DecodeReviewAttestation(raw); return err }},
		{request, func(raw []byte) error { _, err := DecodeEvaluationRequest(raw); return err }},
		{result, func(raw []byte) error { _, err := DecodeEvaluationResult(raw); return err }},
	}
	for _, sample := range samples {
		raw := mustCanonicalJSON(t, sample.document)
		if err := sample.decode(raw); err != nil {
			t.Fatalf("candidate-evaluation-schema-v4-version-isolation: %v", err)
		}
		for _, old := range []int{SchemaVersionV2, SchemaVersionV3} {
			if err := schemas.validate(old, raw); err == nil {
				t.Fatalf("schema %d が v4 成果物を受理しました", old)
			}
		}
		unknown := bytes.Replace(raw, []byte(`"schemaVersion":4`), []byte(`"schemaVersion":4,"unknown":1`), 1)
		if err := sample.decode(unknown); err == nil {
			t.Fatal("未知 field を受理しました")
		}
	}
}

func TestSchemaV4はSOT集合とEvaluatorをExactに固定する(t *testing.T) {
	t.Parallel()
	v3, err := RequiredReviewSOTIDsForSchema(SchemaVersionV3)
	if err != nil {
		t.Fatal(err)
	}
	v4, err := RequiredReviewSOTIDsForSchema(SchemaVersionV4)
	if err != nil {
		t.Fatal(err)
	}
	want := slices.DeleteFunc(slices.Clone(v3), func(id string) bool { return id == "SOT-IF-040" })
	want = append(want, "SOT-IF-077", "SOT-ENG-043", "SOT-ENG-045")
	slices.Sort(want)
	if !slices.Equal(v4, want) || !slices.Contains(v3, "SOT-IF-040") {
		t.Fatal("candidate-evaluation-schema-v4-review-sot-set: 世代ごとの exact 集合が不正です")
	}
	request := validEvaluationRequestForSchema(t, validCandidateManifestForSchema(t, SchemaVersionV4))
	for _, version := range []string{"legal-query-evaluator-v2", "legal-query-evaluator-v4", "current"} {
		invalid := request
		invalid.EvaluatorVersion = version
		invalid.EvaluationID = mustEvaluationID(t, invalid)
		if err := validateEvaluationRequest(invalid); err == nil {
			t.Fatal("v4 が evaluator の非 exact 版を受理しました")
		}
	}
	invalid := request
	invalid.RequiredReviewSOTs = validSOTReferencesForSchema(t, SchemaVersionV3)
	invalid.RequiredReviewSOTSetSHA256 = SOTSetSHA256(invalid.RequiredReviewSOTs)
	invalid.EvaluationID = mustEvaluationID(t, invalid)
	if _, err := DecodeEvaluationRequest(mustCanonicalJSON(t, invalid)); err == nil {
		t.Fatal("v4 が旧世代の SOT 集合を受理しました")
	}
	first := CanonicalSchemaV4()
	first[0] ^= 0xff
	if bytes.Equal(first, CanonicalSchemaV4()) {
		t.Fatal("schema v4 が可変 byte を共有しています")
	}
}

func TestSchemaV4のReadyと旧世代Replayは三世代Rootで分離する(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	v2 := prepareCandidateEvaluationFixture(t, root)
	v3 := writeCandidatePreparationForSchema(t, root, SchemaVersionV3)
	v4 := writeCandidatePreparationForSchema(t, root, SchemaVersionV4)
	for _, request := range []EvaluationRequest{v2, v3} {
		requestRaw := mustCanonicalJSON(t, request)
		reportRaw := syntheticEvaluationReportRaw(fmt.Sprintf("schema-v%d-history", request.SchemaVersion))
		result := mustSyntheticEvaluationResult(t, request, requestRaw, reportRaw, EvaluationOutcomeFailed)
		resultRaw := mustCanonicalJSON(t, result)
		writeV4Fixture(t, root, filepath.Join("results", request.EvaluationID+".json"), resultRaw)
		writeV4Fixture(t, root, filepath.Join("failed-reports", request.EvaluationID+".json"), reportRaw)
		writeV4Pointer(t, root, request)
		validator := &recordingReferenceValidator{reject: true}
		replay, err := LoadCurrentEvaluation(context.Background(), root, validator)
		if err != nil {
			t.Fatalf("candidate-evaluation-schema-v4-historical-isolation: %v", err)
		}
		if !bytes.Equal(replay.RequestRaw, requestRaw) || !bytes.Equal(replay.CurrentResultRaw, resultRaw) ||
			!bytes.Equal(replay.CurrentReportRaw, reportRaw) || validator.manifestCalls != 0 || validator.requestCalls != 0 {
			t.Fatal("旧世代 replay の byte または鮮度分離が変化しました")
		}
	}
	writeV4Pointer(t, root, v4)
	validator := &recordingReferenceValidator{}
	inspection, err := InspectCurrentEvaluation(context.Background(), root, validator)
	if err != nil || inspection.ReadinessState() != CurrentReadinessReady {
		t.Fatalf("candidate-evaluation-schema-v4-ready-route: %v", err)
	}
	current := inspection.Evaluation()
	if len(current.History) != 2 || current.Prepared.Request.SchemaVersion != SchemaVersionV4 ||
		!slices.Equal(validator.requestIDs, []string{v4.EvaluationID}) || validator.manifestCalls != 1 {
		t.Fatal("v4 ready が旧世代を現在の参照へ再結合しました")
	}
	strict, err := LoadCurrentEvaluation(context.Background(), root, &recordingReferenceValidator{})
	if err != nil || strict.Prepared.Request.EvaluationID != v4.EvaluationID {
		t.Fatalf("candidate-evaluation-schema-v4-ready-route: ready の strict load が失敗しました: %v", err)
	}
}

func TestSchemaV4は旧世代の未消費予約も保持する(t *testing.T) {
	t.Parallel()
	for _, schemaVersion := range []int{SchemaVersionV2, SchemaVersionV3} {
		previous := validEvaluationRequest(t, manifestWithID(t))
		if schemaVersion == SchemaVersionV3 {
			previous = validEvaluationRequestForSchema(t, validCandidateManifestForSchema(t, schemaVersion))
		}
		for _, field := range []string{"baseline", "holdout", "leakage"} {
			t.Run(fmt.Sprintf("v%d-%s", schemaVersion, field), func(t *testing.T) {
				current := validEvaluationRequestForSchema(t, validCandidateManifestForSchema(t, SchemaVersionV4))
				switch field {
				case "baseline":
					current.BaselineVersion = previous.BaselineVersion
				case "holdout":
					current.HoldoutDigest = previous.HoldoutDigest
				case "leakage":
					current.HoldoutLeakageGroupDigests = slices.Clone(previous.HoldoutLeakageGroupDigests)
				}
				current.EvaluationID = mustEvaluationID(t, current)
				requests := map[string]loadedArtifact[EvaluationRequest]{previous.EvaluationID: {document: previous}}
				if err := checkRequestReservationPreflight(current, requests); err == nil {
					t.Fatal("candidate-evaluation-schema-v4-historical-isolation: 旧世代の予約を再利用しました")
				}
			})
		}
	}
}

func TestSchemaV4Rootは未知Entryと各Schemaの欠損改変を拒否する(t *testing.T) {
	t.Parallel()
	for name, canonical := range map[string][]byte{
		"schema-v2.json": CanonicalSchemaV2(), "schema-v3.json": CanonicalSchemaV3(),
		"schema-v4.json": CanonicalSchemaV4(), "schema-v5.json": CanonicalSchemaV5(),
		"schema-v6.json": CanonicalSchemaV5(),
	} {
		for _, action := range []string{"missing", "changed"} {
			t.Run(name+"-"+action, func(t *testing.T) {
				root := t.TempDir()
				prepareCandidateEvaluationFixture(t, root)
				if action == "missing" && name != "schema-v6.json" {
					removeCandidateFixture(t, root, filepath.Join("testdata/legalquery/candidate-evaluations", name))
				} else {
					writeV4Fixture(t, root, name, append(canonical, '\n'))
				}
				if _, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{}); err == nil {
					t.Fatal("candidate-evaluation-schema-v4-mixed-root: 不正な root を受理しました")
				}
			})
		}
	}
}

func TestSchemaV4は系列内の世代混在を拒否する(t *testing.T) {
	t.Parallel()
	manifest := validCandidateManifestForSchema(t, SchemaVersionV4)
	request := validEvaluationRequestForSchema(t, manifest)
	older := validCandidateManifestForSchema(t, SchemaVersionV3)
	t.Run("pointer", func(t *testing.T) {
		root := t.TempDir()
		v2 := prepareCandidateEvaluationFixture(t, root)
		pointer := PointerDocument{ArtifactKindPointer, SchemaVersionV4, v2.EvaluationID}
		writeV4Fixture(t, root, "current.json", mustCanonicalJSON(t, pointer))
		_, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{})
		requireSchemaVersionError(t, err)
	})
	t.Run("manifest", func(t *testing.T) {
		cross := request
		cross.CandidateContentID = older.CandidateContentID
		cross.CandidateContentManifestSHA256 = RawSHA256(mustCanonicalJSON(t, older))
		artifacts := preparationArtifacts{manifests: map[string]loadedArtifact[CandidateContentManifest]{
			older.CandidateContentID: {document: older, digest: cross.CandidateContentManifestSHA256},
		}}
		err := bindRequest(cross, artifacts, map[string]struct{}{}, map[string]struct{}{})
		requireSchemaVersionError(t, err)
	})
	t.Run("attestation", func(t *testing.T) {
		attestation := validReviewAttestationForSchema(t, older, ReviewScopeArchitecture, "authority-old")
		cross := cloneEvaluationRequest(request)
		digest := RawSHA256(mustCanonicalJSON(t, attestation))
		cross.ReviewAttestations[0] = ReviewAttestationReference{ReviewScopeArchitecture, attestation.AttestationID, digest}
		artifacts := preparationArtifacts{
			manifests: map[string]loadedArtifact[CandidateContentManifest]{manifest.CandidateContentID: {
				document: manifest, digest: request.CandidateContentManifestSHA256,
			}},
			attestations: map[string]loadedArtifact[ReviewAttestation]{attestation.AttestationID: {document: attestation, digest: digest}},
		}
		err := bindRequest(cross, artifacts, map[string]struct{}{}, map[string]struct{}{})
		requireSchemaVersionError(t, err)
	})
	t.Run("result", func(t *testing.T) {
		result := EvaluationResult{ArtifactKindEvaluationResult, SchemaVersionV3, request.EvaluationID,
			RawSHA256(mustCanonicalJSON(t, request)), EvaluationOutcomeFailed, repeatHex('e')}
		requireSchemaVersionError(t, validateConsumedEvaluation(ConsumedEvaluation{Request: request, Result: result}))
	})
}

func requireSchemaVersionError(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("candidate-evaluation-schema-v4-mixed-root: 世代差で拒否しませんでした: %v", err)
	}
}

func writeV4Pointer(t *testing.T, root string, request EvaluationRequest) {
	t.Helper()
	pointer := PointerDocument{ArtifactKindPointer, request.SchemaVersion, request.EvaluationID}
	writeV4Fixture(t, root, "current.json", mustCanonicalJSON(t, pointer))
}

func writeV4Fixture(t *testing.T, root, name string, raw []byte) {
	t.Helper()
	writeCandidateFixture(t, root, filepath.Join("testdata/legalquery/candidate-evaluations", name), raw)
}
