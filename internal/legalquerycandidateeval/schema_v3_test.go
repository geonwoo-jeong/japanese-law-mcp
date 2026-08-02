package legalquerycandidateeval

import (
	"bytes"
	"context"
	"path/filepath"
	"slices"
	"testing"
)

func TestCanonicalSchemaV3は五成果物をV2から分離する(t *testing.T) {
	t.Parallel()

	schemaV3, err := ParseSchemaV3(CanonicalSchemaV3())
	if err != nil {
		t.Fatalf("candidate-evaluation-schema-v3-version-isolation: schema v3 を解決できません: %v", err)
	}
	schemaV2, err := ParseSchemaV2(CanonicalSchemaV2())
	if err != nil {
		t.Fatalf("schema v2 を解決できません: %v", err)
	}
	manifest := validCandidateManifestForSchema(t, SchemaVersionV3)
	architecture := validReviewAttestationForSchema(
		t,
		manifest,
		ReviewScopeArchitecture,
		"authority-v3-a",
	)
	request := validEvaluationRequestForSchema(t, manifest)
	samples := map[string][]byte{
		"pointer": mustCanonicalJSON(t, PointerDocument{
			ArtifactKind:  ArtifactKindPointer,
			SchemaVersion: SchemaVersionV3,
			EvaluationID:  request.EvaluationID,
		}),
		"candidate-content":  mustCanonicalJSON(t, manifest),
		"review-attestation": mustCanonicalJSON(t, architecture),
		"evaluation-request": mustCanonicalJSON(t, request),
		"evaluation-result": mustCanonicalJSON(t, EvaluationResult{
			ArtifactKind:  ArtifactKindEvaluationResult,
			SchemaVersion: SchemaVersionV3,
			EvaluationID:  request.EvaluationID,
			RequestSHA256: repeatHex('d'),
			Outcome:       EvaluationOutcomeFailed,
			ReportSHA256:  repeatHex('e'),
		}),
	}
	for name, raw := range samples {
		name, raw := name, raw
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := schemaV3.Validate(context.Background(), raw); err != nil {
				t.Fatalf("schema v3 が正しい成果物を拒否しました: %v", err)
			}
			if err := schemaV2.Validate(context.Background(), raw); err == nil {
				t.Fatal("schema v2 が schema v3 成果物を受理しました")
			}
		})
	}
}

func TestCanonicalSchemaV3は複製を返す(t *testing.T) {
	t.Parallel()

	first := CanonicalSchemaV3()
	second := CanonicalSchemaV3()
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("canonical schema v3 が空です")
	}
	first[0] ^= 0xff
	if bytes.Equal(first, second) {
		t.Fatal("canonical schema v3 が呼出し元と可変 byte を共有しています")
	}
}

func TestRequiredReviewSOTIDsForSchemaはV3追加集合をExactに返す(t *testing.T) {
	t.Parallel()

	v2, err := RequiredReviewSOTIDsForSchema(SchemaVersionV2)
	if err != nil {
		t.Fatalf("schema v2 の review SOT 集合を解決できません: %v", err)
	}
	v3, err := RequiredReviewSOTIDsForSchema(SchemaVersionV3)
	if err != nil {
		t.Fatalf("schema v3 の review SOT 集合を解決できません: %v", err)
	}
	want := append(append([]string(nil), v2...), "SOT-ENG-040", "SOT-ENG-041", "SOT-ENG-042")
	slices.Sort(want)
	if !slices.Equal(v3, want) {
		t.Fatalf("candidate-evaluation-schema-v3-review-sot-set: v3 集合 = %#v", v3)
	}
}

func TestDecodeはSchemaV3をExactに選択する(t *testing.T) {
	t.Parallel()

	manifest := validCandidateManifestForSchema(t, SchemaVersionV3)
	architecture := validReviewAttestationForSchema(
		t,
		manifest,
		ReviewScopeArchitecture,
		"authority-v3-a",
	)
	request := validEvaluationRequestForSchema(t, manifest)
	pointer := PointerDocument{
		ArtifactKind:  ArtifactKindPointer,
		SchemaVersion: SchemaVersionV3,
		EvaluationID:  request.EvaluationID,
	}
	result := EvaluationResult{
		ArtifactKind:  ArtifactKindEvaluationResult,
		SchemaVersion: SchemaVersionV3,
		EvaluationID:  request.EvaluationID,
		RequestSHA256: repeatHex('d'),
		Outcome:       EvaluationOutcomeFailed,
		ReportSHA256:  repeatHex('e'),
	}

	if _, err := DecodePointer(mustCanonicalJSON(t, pointer)); err != nil {
		t.Fatalf("schema v3 pointer を拒否しました: %v", err)
	}
	if _, err := DecodeCandidateContentManifest(mustCanonicalJSON(t, manifest)); err != nil {
		t.Fatalf("schema v3 manifest を拒否しました: %v", err)
	}
	if _, err := DecodeReviewAttestation(mustCanonicalJSON(t, architecture)); err != nil {
		t.Fatalf("schema v3 attestation を拒否しました: %v", err)
	}
	if _, err := DecodeEvaluationRequest(mustCanonicalJSON(t, request)); err != nil {
		t.Fatalf("schema v3 request を拒否しました: %v", err)
	}
	if _, err := DecodeEvaluationResult(mustCanonicalJSON(t, result)); err != nil {
		t.Fatalf("schema v3 result を拒否しました: %v", err)
	}
}

func TestDecodeのSchemaVersion判別はFailClosedである(t *testing.T) {
	t.Parallel()

	raw := mustCanonicalJSON(t, manifestWithID(t))
	tests := map[string][]byte{
		"unknown":  bytes.Replace(raw, []byte(`"schemaVersion":2`), []byte(`"schemaVersion":4`), 1),
		"missing":  bytes.Replace(raw, []byte(`"schemaVersion":2,`), nil, 1),
		"string":   bytes.Replace(raw, []byte(`"schemaVersion":2`), []byte(`"schemaVersion":"2"`), 1),
		"fraction": bytes.Replace(raw, []byte(`"schemaVersion":2`), []byte(`"schemaVersion":2.5`), 1),
		"exponent": bytes.Replace(raw, []byte(`"schemaVersion":2`), []byte(`"schemaVersion":2e0`), 1),
		"null":     bytes.Replace(raw, []byte(`"schemaVersion":2`), []byte(`"schemaVersion":null`), 1),
		"duplicate": bytes.Replace(
			raw,
			[]byte(`"schemaVersion":2`),
			[]byte(`"schemaVersion":2,"schemaVersion":2`),
			1,
		),
		"mixed-duplicate": bytes.Replace(
			raw,
			[]byte(`"schemaVersion":2`),
			[]byte(`"schemaVersion":2,"schemaVersion":3`),
			1,
		),
	}
	for name, invalid := range tests {
		name, invalid := name, invalid
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := DecodeCandidateContentManifest(invalid); err == nil {
				t.Fatal("不正な schemaVersion 判別入力を受理しました")
			}
		})
	}
}

func TestRepositoryArtifactDecodeは処理中のCancelを伝播する(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	_, err := decodeWithContext(
		ctx,
		nil,
		artifactSchemas{},
		func([]byte, artifactSchemas) (PointerDocument, error) {
			cancel()
			return PointerDocument{}, nil
		},
	)
	if err == nil {
		t.Fatal("candidate-evaluation-schema-v3-context-propagation: decode 中の cancel を受理しました")
	}
}

func TestLoadCurrentEvaluationは混在履歴をArtifact宣言版で読む(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	v2 := prepareCandidateEvaluationFixture(t, root)
	v2RequestRaw := mustCanonicalJSON(t, v2)
	v2ReportRaw := syntheticEvaluationReportRaw("schema-v2-history")
	v2Result := mustSyntheticEvaluationResult(
		t,
		v2,
		v2RequestRaw,
		v2ReportRaw,
		EvaluationOutcomeFailed,
	)
	base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
	writeCandidateFixture(t, root, filepath.Join(
		base,
		"results",
		v2.EvaluationID+".json",
	), mustCanonicalJSON(t, v2Result))
	writeCandidateFixture(t, root, filepath.Join(
		base,
		"failed-reports",
		v2.EvaluationID+".json",
	), v2ReportRaw)
	v3 := writeCandidatePreparationForSchema(t, root, SchemaVersionV3)
	pointer := PointerDocument{
		ArtifactKind:  ArtifactKindPointer,
		SchemaVersion: SchemaVersionV3,
		EvaluationID:  v3.EvaluationID,
	}
	writeCandidateFixture(t, root, filepath.Join(base, "current.json"), mustCanonicalJSON(t, pointer))

	validator := &recordingReferenceValidator{}
	current, err := LoadCurrentEvaluation(context.Background(), root, validator)
	if err != nil {
		t.Fatalf("mixed-generation history を拒否しました: %v", err)
	}
	if current.Prepared.Request.SchemaVersion != SchemaVersionV3 ||
		len(current.History) != 1 ||
		current.History[0].Request.SchemaVersion != SchemaVersionV2 ||
		current.History[0].Result.SchemaVersion != SchemaVersionV2 {
		t.Fatalf("mixed-generation history が一致しません: %+v", current)
	}
}

func TestLoadPreparedCurrentはCurrent系列のCrossVersionBindingを拒否する(t *testing.T) {
	t.Parallel()

	t.Run("pointer-request", func(t *testing.T) {
		root := t.TempDir()
		v2 := prepareCandidateEvaluationFixture(t, root)
		pointer := PointerDocument{
			ArtifactKind:  ArtifactKindPointer,
			SchemaVersion: SchemaVersionV3,
			EvaluationID:  v2.EvaluationID,
		}
		writeCandidateFixture(t, root, filepath.Join(
			"testdata",
			"legalquery",
			"candidate-evaluations",
			"current.json",
		), mustCanonicalJSON(t, pointer))
		if _, err := LoadPreparedCurrent(
			context.Background(),
			root,
			&recordingReferenceValidator{},
		); err == nil {
			t.Fatal("pointer と request の cross-version binding を受理しました")
		}
	})

	t.Run("request-manifest", func(t *testing.T) {
		root := t.TempDir()
		v2 := prepareCandidateEvaluationFixture(t, root)
		base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
		manifestRaw := mustCanonicalJSON(t, manifestWithID(t))
		architecture := crossVersionV3Attestation(
			t,
			manifestWithID(t),
			ReviewScopeArchitecture,
			"authority-cross-a",
		)
		testability := crossVersionV3Attestation(
			t,
			manifestWithID(t),
			ReviewScopeTestability,
			"authority-cross-b",
		)
		references := validSOTReferencesForSchema(t, SchemaVersionV3)
		request := validEvaluationRequestForSchema(
			t,
			validCandidateManifestForSchema(t, SchemaVersionV3),
		)
		request.CandidateContentID = v2.CandidateContentID
		request.CandidateContentManifestSHA256 = RawSHA256(manifestRaw)
		request.RequiredReviewSOTs = references
		request.RequiredReviewSOTSetSHA256 = SOTSetSHA256(references)
		request.ReviewAttestations = []ReviewAttestationReference{
			{
				ReviewScope:       ReviewScopeArchitecture,
				AttestationID:     architecture.AttestationID,
				AttestationSHA256: RawSHA256(mustCanonicalJSON(t, architecture)),
			},
			{
				ReviewScope:       ReviewScopeTestability,
				AttestationID:     testability.AttestationID,
				AttestationSHA256: RawSHA256(mustCanonicalJSON(t, testability)),
			},
		}
		request.EvaluationID = mustEvaluationID(t, request)
		for _, attestation := range []ReviewAttestation{architecture, testability} {
			writeCandidateFixture(t, root, filepath.Join(
				base,
				"review-attestations",
				attestation.AttestationID+".json",
			), mustCanonicalJSON(t, attestation))
		}
		writeCandidateFixture(t, root, filepath.Join(
			base,
			"requests",
			request.EvaluationID+".json",
		), mustCanonicalJSON(t, request))
		pointer := PointerDocument{
			ArtifactKind:  ArtifactKindPointer,
			SchemaVersion: SchemaVersionV3,
			EvaluationID:  request.EvaluationID,
		}
		writeCandidateFixture(t, root, filepath.Join(base, "current.json"), mustCanonicalJSON(t, pointer))
		if _, err := LoadPreparedCurrent(
			context.Background(),
			root,
			&recordingReferenceValidator{},
		); err == nil {
			t.Fatal("request と manifest の cross-version binding を受理しました")
		}
	})
}

func TestLoadCurrentEvaluationはRequestResultのCrossVersionBindingを拒否する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
	v3 := writeCandidatePreparationForSchema(t, root, SchemaVersionV3)
	pointer := PointerDocument{
		ArtifactKind:  ArtifactKindPointer,
		SchemaVersion: SchemaVersionV3,
		EvaluationID:  v3.EvaluationID,
	}
	writeCandidateFixture(t, root, filepath.Join(base, "current.json"), mustCanonicalJSON(t, pointer))
	reportRaw := syntheticEvaluationReportRaw("cross-version-result")
	result := EvaluationResult{
		ArtifactKind:  ArtifactKindEvaluationResult,
		SchemaVersion: SchemaVersionV2,
		EvaluationID:  v3.EvaluationID,
		RequestSHA256: RawSHA256(mustCanonicalJSON(t, v3)),
		Outcome:       EvaluationOutcomeFailed,
		ReportSHA256:  RawSHA256(reportRaw),
	}
	writeCandidateFixture(t, root, filepath.Join(
		base,
		"results",
		v3.EvaluationID+".json",
	), mustCanonicalJSON(t, result))
	writeCandidateFixture(t, root, filepath.Join(
		base,
		"failed-reports",
		v3.EvaluationID+".json",
	), reportRaw)

	if _, err := LoadCurrentEvaluation(
		context.Background(),
		root,
		&recordingReferenceValidator{},
	); err == nil {
		t.Fatal("request と result の cross-version binding を受理しました")
	}
}

func TestLoadCurrentEvaluationはSchemaV3併存後もV2ReplayByteを変えない(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	requestRaw := mustCanonicalJSON(t, request)
	reportRaw := syntheticEvaluationReportRaw("schema-v2-byte-replay")
	result := mustSyntheticEvaluationResult(
		t,
		request,
		requestRaw,
		reportRaw,
		EvaluationOutcomeFailed,
	)
	resultRaw := mustCanonicalJSON(t, result)
	base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
	writeCandidateFixture(t, root, filepath.Join(
		base,
		"results",
		request.EvaluationID+".json",
	), resultRaw)
	writeCandidateFixture(t, root, filepath.Join(
		base,
		"failed-reports",
		request.EvaluationID+".json",
	), reportRaw)

	validator := &recordingReferenceValidator{reject: true}
	current, err := LoadCurrentEvaluation(context.Background(), root, validator)
	if err != nil {
		t.Fatalf("schema v2 replay を拒否しました: %v", err)
	}
	if !bytes.Equal(current.RequestRaw, requestRaw) ||
		!bytes.Equal(current.CurrentResultRaw, resultRaw) ||
		!bytes.Equal(current.CurrentReportRaw, reportRaw) {
		t.Fatal("candidate-evaluation-schema-v2-historical-replay: replay byte が変化しました")
	}
	replayed, err := NewEvaluationResult(
		request,
		current.RequestRaw,
		current.CurrentReportRaw,
		EvaluationOutcomeFailed,
	)
	if err != nil {
		t.Fatalf("schema v2 result を再構成できません: %v", err)
	}
	if !bytes.Equal(mustCanonicalJSON(t, replayed), resultRaw) {
		t.Fatal("schema v2 result の再構成 byte が変化しました")
	}
	if validator.manifestCalls != 0 || validator.requestCalls != 0 {
		t.Fatal("historical replay が current 外部参照を再検証しました")
	}
}

func TestLoadPreparedCurrentはV2CurrentとSchemaV3を共存させる(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
	writeCandidateFixture(t, root, filepath.Join(base, "schema-v3.json"), CanonicalSchemaV3())

	prepared, err := LoadPreparedCurrent(
		context.Background(),
		root,
		&recordingReferenceValidator{},
	)
	if err != nil {
		t.Fatalf("candidate-evaluation-schema-v2-replay-with-schema-v3-present: %v", err)
	}
	if prepared.Request.EvaluationID != request.EvaluationID ||
		prepared.Request.SchemaVersion != SchemaVersionV2 {
		t.Fatalf("schema v2 current が変化しました: %+v", prepared.Request)
	}
}

func TestLoadPreparedCurrentはArtifactごとにV2V3をDispatchする(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
	writeCandidateFixture(t, root, filepath.Join(base, "schema-v3.json"), CanonicalSchemaV3())
	v3 := writeCandidatePreparationForSchema(t, root, SchemaVersionV3)

	pointer := PointerDocument{
		ArtifactKind:  ArtifactKindPointer,
		SchemaVersion: SchemaVersionV3,
		EvaluationID:  v3.EvaluationID,
	}
	writeCandidateFixture(t, root, filepath.Join(base, "current.json"), mustCanonicalJSON(t, pointer))
	prepared, err := LoadPreparedCurrent(
		context.Background(),
		root,
		&recordingReferenceValidator{},
	)
	if err != nil {
		t.Fatalf("candidate-evaluation-schema-v3-load-with-schema-v2-present: %v", err)
	}
	if prepared.Request.EvaluationID != v3.EvaluationID ||
		prepared.Request.SchemaVersion != SchemaVersionV3 {
		t.Fatalf("schema v3 current を返しませんでした: %+v", prepared.Request)
	}
}

func TestLoadPreparedCurrentは置換済みRequestとの予約衝突を拒否する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	previous := prepareCandidateEvaluationFixture(t, root)
	base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
	writeCandidateFixture(t, root, filepath.Join(base, "schema-v3.json"), CanonicalSchemaV3())
	v3 := writeCandidatePreparationForSchema(t, root, SchemaVersionV3)
	removeCandidateFixture(t, root, filepath.Join("testdata", "legalquery", "candidate-evaluations", "requests", v3.EvaluationID+".json"))
	v3.HoldoutDigest = previous.HoldoutDigest
	v3.EvaluationID = mustEvaluationID(t, v3)
	writeCandidateFixture(t, root, filepath.Join(base, "requests", v3.EvaluationID+".json"), mustCanonicalJSON(t, v3))
	pointer := PointerDocument{
		ArtifactKind:  ArtifactKindPointer,
		SchemaVersion: SchemaVersionV3,
		EvaluationID:  v3.EvaluationID,
	}
	writeCandidateFixture(t, root, filepath.Join(base, "current.json"), mustCanonicalJSON(t, pointer))

	if _, err := LoadPreparedCurrent(
		context.Background(),
		root,
		&recordingReferenceValidator{},
	); err == nil {
		t.Fatal("candidate-evaluation-schema-v3-superseded-request-reservation: 置換済み request の holdout 再利用を受理しました")
	}
}

func TestNewEvaluationResultはRequestのSchemaVersionを保持する(t *testing.T) {
	t.Parallel()

	manifest := validCandidateManifestForSchema(t, SchemaVersionV3)
	request := validEvaluationRequestForSchema(t, manifest)
	requestRaw := mustCanonicalJSON(t, request)
	result, err := NewEvaluationResult(
		request,
		requestRaw,
		[]byte("{}\n"),
		EvaluationOutcomeFailed,
	)
	if err != nil {
		t.Fatalf("schema v3 result を作れません: %v", err)
	}
	if result.SchemaVersion != SchemaVersionV3 {
		t.Fatalf("result schemaVersion = %d", result.SchemaVersion)
	}
}

func writeCandidatePreparationForSchema(
	t *testing.T,
	root string,
	schemaVersion int,
) EvaluationRequest {
	t.Helper()
	base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
	manifest := validCandidateManifestForSchema(t, schemaVersion)
	architecture := validReviewAttestationForSchema(
		t,
		manifest,
		ReviewScopeArchitecture,
		"authority-v3-a",
	)
	testability := validReviewAttestationForSchema(
		t,
		manifest,
		ReviewScopeTestability,
		"authority-v3-b",
	)
	request := validEvaluationRequestForSchema(t, manifest)
	writeCandidateFixture(t, root, filepath.Join(
		base,
		"content-manifests",
		manifest.CandidateContentID+".json",
	), mustCanonicalJSON(t, manifest))
	for _, attestation := range []ReviewAttestation{architecture, testability} {
		writeCandidateFixture(t, root, filepath.Join(
			base,
			"review-attestations",
			attestation.AttestationID+".json",
		), mustCanonicalJSON(t, attestation))
	}
	writeCandidateFixture(t, root, filepath.Join(
		base,
		"requests",
		request.EvaluationID+".json",
	), mustCanonicalJSON(t, request))
	return request
}

func validCandidateManifestForSchema(
	t *testing.T,
	schemaVersion int,
) CandidateContentManifest {
	t.Helper()
	manifest := validCandidateManifest(t)
	manifest.SchemaVersion = schemaVersion
	manifest.ProfileSet.ProfileSetVersion = "profile-set-schema-v3"
	manifest.Composition.ProfileSetVersion = manifest.ProfileSet.ProfileSetVersion
	manifest.Composition.CompositionVersion = "composition-schema-v3"
	manifest.Composition.DescriptorSHA256 = mustCompositionDigest(t, manifest.Composition)
	manifest.CandidateContentID = mustCandidateID(t, manifest)
	return manifest
}

func validReviewAttestationForSchema(
	t *testing.T,
	manifest CandidateContentManifest,
	scope string,
	authority string,
) ReviewAttestation {
	t.Helper()
	attestation := validReviewAttestation(t, manifest, scope, authority)
	references := validSOTReferencesForSchema(t, manifest.SchemaVersion)
	attestation.SchemaVersion = manifest.SchemaVersion
	attestation.ReviewedSOTs = references
	attestation.ReviewedSOTSetSHA256 = SOTSetSHA256(references)
	attestation.CandidateContentManifestSHA256 = RawSHA256(mustCanonicalJSON(t, manifest))
	attestation.AttestationID = mustReviewID(t, attestation)
	return attestation
}

func crossVersionV3Attestation(
	t *testing.T,
	manifest CandidateContentManifest,
	scope string,
	authority string,
) ReviewAttestation {
	t.Helper()
	attestation := validReviewAttestation(t, manifest, scope, authority)
	references := validSOTReferencesForSchema(t, SchemaVersionV3)
	attestation.SchemaVersion = SchemaVersionV3
	attestation.ReviewedSOTs = references
	attestation.ReviewedSOTSetSHA256 = SOTSetSHA256(references)
	attestation.AttestationID = mustReviewID(t, attestation)
	return attestation
}

func validEvaluationRequestForSchema(
	t *testing.T,
	manifest CandidateContentManifest,
) EvaluationRequest {
	t.Helper()
	architecture := validReviewAttestationForSchema(
		t,
		manifest,
		ReviewScopeArchitecture,
		"authority-v3-a",
	)
	testability := validReviewAttestationForSchema(
		t,
		manifest,
		ReviewScopeTestability,
		"authority-v3-b",
	)
	references := validSOTReferencesForSchema(t, manifest.SchemaVersion)
	request := EvaluationRequest{
		ArtifactKind:                   ArtifactKindEvaluationRequest,
		SchemaVersion:                  manifest.SchemaVersion,
		EvaluatorVersion:               "legal-query-evaluator-v3",
		CorpusVersion:                  "corpus-v99",
		CorpusManifestSHA256:           repeatHex('1'),
		HoldoutDigest:                  repeatHex('2'),
		HoldoutLeakageGroupDigests:     []string{repeatHex('3'), repeatHex('4')},
		CandidateContentID:             manifest.CandidateContentID,
		CandidateContentManifestSHA256: RawSHA256(mustCanonicalJSON(t, manifest)),
		ReviewRubricVersion:            ReviewRubricVersion,
		ReviewRubricSHA256:             ReviewRubricSHA256(),
		RequiredReviewSOTs:             references,
		RequiredReviewSOTSetSHA256:     SOTSetSHA256(references),
		ReviewAttestations: []ReviewAttestationReference{
			{
				ReviewScope:       ReviewScopeArchitecture,
				AttestationID:     architecture.AttestationID,
				AttestationSHA256: RawSHA256(mustCanonicalJSON(t, architecture)),
			},
			{
				ReviewScope:       ReviewScopeTestability,
				AttestationID:     testability.AttestationID,
				AttestationSHA256: RawSHA256(mustCanonicalJSON(t, testability)),
			},
		},
		BaselineVersion: "default-99",
	}
	request.EvaluationID = mustEvaluationID(t, request)
	return request
}

func validSOTReferencesForSchema(t *testing.T, schemaVersion int) []SOTReference {
	t.Helper()
	ids, err := RequiredReviewSOTIDsForSchema(schemaVersion)
	if err != nil {
		t.Fatalf("schema version %d の review SOT 集合を解決できません: %v", schemaVersion, err)
	}
	references := make([]SOTReference, 0, len(ids))
	for index, id := range ids {
		digit := "0123456789abcdef"[index%16]
		references = append(references, SOTReference{
			SOTID:             id,
			SOTDocumentSHA256: repeatHex(digit),
		})
	}
	return references
}
