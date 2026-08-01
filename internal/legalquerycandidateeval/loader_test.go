package legalquerycandidateeval

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordingReferenceValidator struct {
	manifestCalls int
	requestCalls  int
	reject        bool
	currentSOTs   []SOTReference
}

func (v *recordingReferenceValidator) ValidateCandidateContent(
	_ context.Context,
	_ []byte,
	_ CandidateContentManifest,
) error {
	v.manifestCalls++
	if v.reject {
		return errRejectedReference
	}
	return nil
}

func (v *recordingReferenceValidator) ValidateEvaluationRequest(
	_ context.Context,
	_ []byte,
	document EvaluationRequest,
) (RequestReferenceValidation, error) {
	v.requestCalls++
	if v.reject {
		return RequestReferenceValidation{}, errRejectedReference
	}
	references := document.RequiredReviewSOTs
	if v.currentSOTs != nil {
		references = v.currentSOTs
	}
	return RequestReferenceValidation{
		CurrentRequiredReviewSOTs: append([]SOTReference(nil), references...),
	}, nil
}

func TestLoadPreparedCurrentはcurrentと内容固定reviewを一体で返す(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	validator := &recordingReferenceValidator{}
	prepared, err := LoadPreparedCurrent(context.Background(), root, validator)
	if err != nil {
		t.Fatalf("candidate-evaluation-current-single-target: 正しい準備成果物を拒否しました: %v", err)
	}
	if prepared.Request.EvaluationID != request.EvaluationID {
		t.Fatalf("evaluationId = %q", prepared.Request.EvaluationID)
	}
	if len(prepared.ReviewAttestations) != 2 {
		t.Fatalf("review attestation 数 = %d", len(prepared.ReviewAttestations))
	}
	if validator.manifestCalls != 1 || validator.requestCalls != 1 {
		t.Fatalf("external validator 呼出し = manifest:%d request:%d", validator.manifestCalls, validator.requestCalls)
	}
}

func TestLoadPreparedCurrentは孤立成果物とmissingTargetを拒否する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	orphan := manifestWithID(t)
	orphan.ProfileSet.ProfileSetVersion = "orphan-v1"
	orphan.Composition.ProfileSetVersion = "orphan-v1"
	orphan.Composition.DescriptorSHA256 = mustCompositionDigest(t, orphan.Composition)
	orphan.CandidateContentID = mustCandidateID(t, orphan)
	writeCandidateFixture(t, root, filepath.Join(
		"testdata/legalquery/candidate-evaluations/content-manifests",
		orphan.CandidateContentID+".json",
	), mustCanonicalJSON(t, orphan))
	if _, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("孤立 manifest を受理しました")
	}

	root = t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	pointer := PointerDocument{ArtifactKind: ArtifactKindPointer, SchemaVersion: SchemaVersionV2, EvaluationID: "evaluation-sha256-" + repeatHex('f')}
	writeCandidateFixture(t, root, "testdata/legalquery/candidate-evaluations/current.json", mustCanonicalJSON(t, pointer))
	if _, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{}); err == nil {
		t.Fatalf("存在しない current target を受理しました: original=%s", request.EvaluationID)
	}
}

func TestLoadPreparedCurrentはrootの未知Entryと評価済み履歴を拒否する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	writeCandidateFixture(t, root, "testdata/legalquery/candidate-evaluations/unknown.json", []byte("{}\n"))
	if _, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("root の未知 entry を受理しました")
	}

	root = t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	writeCandidateFixture(t, root, filepath.Join(
		"testdata/legalquery/candidate-evaluations/results",
		request.EvaluationID+".json",
	), []byte("{}\n"))
	if _, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("準備専用 loader が result を受理しました")
	}
}

func TestLoadPreparedCurrentはGitで追跡不能な空履歴Directoryの不在を空として扱う(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	base := filepath.Join(root, "testdata", "legalquery", "candidate-evaluations")
	for _, name := range []string{"results", "failed-reports"} {
		if err := os.Remove(filepath.Join(base, name)); err != nil {
			t.Fatalf("空の %s directory を削除できません: %v", name, err)
		}
	}
	prepared, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{})
	if err != nil {
		t.Fatalf("空履歴 directory が Git checkout にない準備成果物を拒否しました: %v", err)
	}
	if prepared.Request.EvaluationID != request.EvaluationID {
		t.Fatalf("evaluationId = %q", prepared.Request.EvaluationID)
	}
}

func TestLoadPreparedCurrentは固定Schema以外を拒否する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	writeCandidateFixture(
		t,
		root,
		"testdata/legalquery/candidate-evaluations/schema-v2.json",
		testSchemaV2(),
	)
	if _, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("固定済み schema v2 と異なる byte を受理しました")
	}
}

func TestLoadPreparedCurrentは複数の未評価Requestを拒否する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	request.BaselineVersion = "default-3"
	request.EvaluationID = mustEvaluationID(t, request)
	writeCandidateFixture(t, root, filepath.Join(
		"testdata/legalquery/candidate-evaluations/requests",
		request.EvaluationID+".json",
	), mustCanonicalJSON(t, request))
	if _, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("複数の未評価 request を受理しました")
	}
}

func TestLoadPreparedCurrentは外部参照検証失敗を伝播する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	_, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{reject: true})
	if err == nil || !strings.Contains(err.Error(), errRejectedReference.Error()) {
		t.Fatalf("外部参照検証失敗を伝播しませんでした: %v", err)
	}
}

func TestLoadPreparedCurrentはcurrentSOT原Byteの変化を拒否する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	current := validSOTReferences()
	current[0].SOTDocumentSHA256 = repeatHex('f')
	_, err := LoadPreparedCurrent(
		context.Background(),
		root,
		&recordingReferenceValidator{currentSOTs: current},
	)
	if err == nil {
		t.Fatal("review 後に変化した current SOT 原 byte digest を受理しました")
	}
}

func TestLoadPreparedCurrentはnil境界とcancelを拒否する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	prepareCandidateEvaluationFixture(t, root)
	if _, err := LoadPreparedCurrent(context.Background(), root, nil); err == nil {
		t.Fatal("nil external validator を受理しました")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LoadPreparedCurrent(ctx, root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("cancel 済み context を受理しました")
	}
	//nolint:staticcheck // SOT-ENG-038: nil context を fail-closed に拒否する境界を直接検証する。
	if _, err := LoadPreparedCurrent(nil, root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("nil context を受理しました")
	}
}

func TestLoadPreparedCurrentはrawDigestとreviewer独立性を検証する(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	request := prepareCandidateEvaluationFixture(t, root)
	base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
	removeCandidateFixture(t, root, filepath.Join(base, "requests", request.EvaluationID+".json"))
	request.CandidateContentManifestSHA256 = repeatHex('0')
	request.EvaluationID = mustEvaluationID(t, request)
	writeCandidateFixture(t, root, filepath.Join(base, "requests", request.EvaluationID+".json"), mustCanonicalJSON(t, request))
	pointer := PointerDocument{ArtifactKind: ArtifactKindPointer, SchemaVersion: SchemaVersionV2, EvaluationID: request.EvaluationID}
	writeCandidateFixture(t, root, filepath.Join(base, "current.json"), mustCanonicalJSON(t, pointer))
	if _, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("manifest raw digest の不一致を受理しました")
	}

	root = t.TempDir()
	request = prepareCandidateEvaluationFixture(t, root)
	manifest := manifestWithID(t)
	originalTestability := validReviewAttestation(t, manifest, ReviewScopeTestability, "authority-b")
	sameAuthority := validReviewAttestation(t, manifest, ReviewScopeTestability, "authority-a")
	removeCandidateFixture(t, root, filepath.Join(base, "review-attestations", originalTestability.AttestationID+".json"))
	writeCandidateFixture(t, root, filepath.Join(base, "review-attestations", sameAuthority.AttestationID+".json"), mustCanonicalJSON(t, sameAuthority))
	removeCandidateFixture(t, root, filepath.Join(base, "requests", request.EvaluationID+".json"))
	request.ReviewAttestations[1].AttestationID = sameAuthority.AttestationID
	request.ReviewAttestations[1].AttestationSHA256 = RawSHA256(mustCanonicalJSON(t, sameAuthority))
	request.EvaluationID = mustEvaluationID(t, request)
	writeCandidateFixture(t, root, filepath.Join(base, "requests", request.EvaluationID+".json"), mustCanonicalJSON(t, request))
	pointer = PointerDocument{ArtifactKind: ArtifactKindPointer, SchemaVersion: SchemaVersionV2, EvaluationID: request.EvaluationID}
	writeCandidateFixture(t, root, filepath.Join(base, "current.json"), mustCanonicalJSON(t, pointer))
	if _, err := LoadPreparedCurrent(context.Background(), root, &recordingReferenceValidator{}); err == nil {
		t.Fatal("同一 reviewerAuthorityId の二 scope を受理しました")
	}
}

func prepareCandidateEvaluationFixture(t *testing.T, root string) EvaluationRequest {
	t.Helper()
	base := filepath.Join("testdata", "legalquery", "candidate-evaluations")
	for _, directory := range []string{"content-manifests", "review-attestations", "requests", "results", "failed-reports"} {
		if err := os.MkdirAll(filepath.Join(root, base, directory), 0o750); err != nil {
			t.Fatalf("candidate evaluation test directory を作れません: %v", err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "testdata/legalquery/baselines/versions"), 0o750); err != nil {
		t.Fatalf("candidate evaluation baseline test directory を作れません: %v", err)
	}
	writeCandidateFixture(t, root, filepath.Join(base, "schema-v2.json"), CanonicalSchemaV2())
	manifest := manifestWithID(t)
	architecture := validReviewAttestation(t, manifest, ReviewScopeArchitecture, "authority-a")
	testability := validReviewAttestation(t, manifest, ReviewScopeTestability, "authority-b")
	request := validEvaluationRequest(t, manifest)
	writeCandidateFixture(t, root, filepath.Join(base, "content-manifests", manifest.CandidateContentID+".json"), mustCanonicalJSON(t, manifest))
	writeCandidateFixture(t, root, filepath.Join(base, "review-attestations", architecture.AttestationID+".json"), mustCanonicalJSON(t, architecture))
	writeCandidateFixture(t, root, filepath.Join(base, "review-attestations", testability.AttestationID+".json"), mustCanonicalJSON(t, testability))
	writeCandidateFixture(t, root, filepath.Join(base, "requests", request.EvaluationID+".json"), mustCanonicalJSON(t, request))
	pointer := PointerDocument{ArtifactKind: ArtifactKindPointer, SchemaVersion: SchemaVersionV2, EvaluationID: request.EvaluationID}
	writeCandidateFixture(t, root, filepath.Join(base, "current.json"), mustCanonicalJSON(t, pointer))
	return request
}

func writeCandidateFixture(t *testing.T, root, relative string, raw []byte) {
	t.Helper()
	target := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("candidate evaluation test directory を作れません: %v", err)
	}
	if err := os.WriteFile(target, raw, 0o600); err != nil {
		t.Fatalf("candidate evaluation test file を書けません: %v", err)
	}
}

func removeCandidateFixture(t *testing.T, root, relative string) {
	t.Helper()
	if err := os.Remove(filepath.Join(root, relative)); err != nil {
		t.Fatalf("candidate evaluation test file を削除できません: %v", err)
	}
}

var errRejectedReference = &referenceValidationError{}

type referenceValidationError struct{}

func (*referenceValidationError) Error() string { return "外部参照を検証できません" }
