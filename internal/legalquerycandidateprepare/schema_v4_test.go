package legalquerycandidateprepare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

func TestBuildContentManifestForSchemaは世代を明示して内容を固定する(t *testing.T) {
	t.Parallel()

	root := candidateRepositoryRoot(t)
	source := validSourceSetForTest(t)
	v3, err := BuildContentManifest(t.Context(), root, source)
	if err != nil {
		t.Fatal(err)
	}
	v4, err := BuildContentManifestForSchema(t.Context(), root, source, legalquerycandidateeval.SchemaVersionV4)
	if err != nil {
		t.Fatal(err)
	}
	if v3.SchemaVersion != legalquerycandidateeval.SchemaVersionV3 ||
		v4.SchemaVersion != legalquerycandidateeval.SchemaVersionV4 ||
		v3.CandidateContentID == v4.CandidateContentID ||
		!reflect.DeepEqual(v3.ProfileArtifacts, v4.ProfileArtifacts) ||
		!reflect.DeepEqual(v3.Composition, v4.Composition) ||
		!reflect.DeepEqual(v3.SemanticSourceSet, v4.SemanticSourceSet) {
		t.Fatal("candidate-evaluation-schema-v4-ready-route: 世代分離又は候補内容を変更しました")
	}
	if _, err := legalquerycandidateeval.DecodeCandidateContentManifest(prepareCanonicalJSON(t, v4)); err != nil {
		t.Fatal(err)
	}
	for _, version := range []int{0, 2, 5} {
		if _, err := BuildContentManifestForSchema(t.Context(), root, source, version); err == nil {
			t.Fatalf("準備できない schema %d を受理しました", version)
		}
	}
}

func TestSchemaV4準備はFixtureなしでRequestの参照を結合する(t *testing.T) {
	t.Parallel()

	root := prepareV4ReferenceRoot(t)
	manifest, err := BuildContentManifestForSchema(
		t.Context(), candidateRepositoryRoot(t), validSourceSetForTest(t), legalquerycandidateeval.SchemaVersionV4,
	)
	if err != nil {
		t.Fatal(err)
	}
	manifestRaw := prepareCanonicalJSON(t, manifest)
	references, err := BuildRequiredSOTReferences(t.Context(), root, manifest.SchemaVersion)
	if err != nil {
		t.Fatal(err)
	}
	architecture := mustReviewForTest(t, manifest, manifestRaw, references,
		legalquerycandidateeval.ReviewScopeArchitecture, "v4-architecture")
	testability := mustReviewForTest(t, manifest, manifestRaw, references,
		legalquerycandidateeval.ReviewScopeTestability, "v4-testability")
	request, err := BuildEvaluationRequest(t.Context(), root, "corpus-v99", manifest, manifestRaw,
		architecture, prepareCanonicalJSON(t, architecture), testability, prepareCanonicalJSON(t, testability), "default-99")
	if err != nil {
		t.Fatalf("candidate-evaluation-schema-v4-ready-route: request を構成できません: %v", err)
	}
	validator, err := NewReferenceValidator(root)
	if err != nil {
		t.Fatal(err)
	}
	validation, err := validator.ValidateEvaluationRequest(t.Context(), prepareCanonicalJSON(t, request), request)
	if err != nil || len(validation.StaleReasons) != 0 ||
		!reflect.DeepEqual(validation.CurrentRequiredReviewSOTs, references) ||
		request.SchemaVersion != manifest.SchemaVersion ||
		request.EvaluatorVersion != legalquerycandidateeval.EvaluatorVersionV3 {
		t.Fatalf("candidate-evaluation-schema-v4-ready-route: 外部参照=(%#v,%v)", validation, err)
	}
	for _, version := range []int{0, 2, 5} {
		invalid := request
		invalid.SchemaVersion = version
		if _, err := validator.ValidateEvaluationRequest(t.Context(), nil, invalid); err == nil {
			t.Fatalf("readiness が schema %d を受理しました", version)
		}
	}
	// SOT-ENG-045: digest は同じでも、参照する review の世代は一致しなければならない。
	wrongReview := architecture
	wrongReview.SchemaVersion = legalquerycandidateeval.SchemaVersionV3
	if err := verifyRequestReviews(manifest, manifestRaw, references,
		wrongReview, prepareCanonicalJSON(t, wrongReview), testability, prepareCanonicalJSON(t, testability)); err == nil {
		t.Fatal("cross-generation review を受理しました")
	}
}

func TestReferenceValidatorはSchemaV4内容を同じ世代で再構築する(t *testing.T) {
	if !useExactCandidateToolchain(t) {
		t.Skip("候補再現用 Go 環境がないため local では実行しません")
	}
	root := candidateRepositoryRoot(t)
	source, err := BuildSemanticSourceSet(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := BuildContentManifestForSchema(t.Context(), root, source, legalquerycandidateeval.SchemaVersionV4)
	if err != nil {
		t.Fatal(err)
	}
	validator, err := NewReferenceValidator(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := validator.ValidateCandidateContent(t.Context(), prepareCanonicalJSON(t, manifest), manifest); err != nil {
		t.Fatalf("candidate-evaluation-schema-v4-ready-route: v4 内容を再構築できません: %v", err)
	}
}

func prepareCanonicalJSON(t *testing.T, document any) []byte {
	t.Helper()
	raw, err := legalquerycandidateeval.MarshalCanonicalJSON(document)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func prepareV4ReferenceRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	repository := candidateRepositoryRoot(t)
	// SOT-ENG-045: SOT、adoption identity と corpus schema だけを複製し、fixture は開かない。
	for _, relative := range []string{"sot", "testdata/legalquery/adoptions", "testdata/legalquery/schemas"} {
		if err := os.CopyFS(filepath.Join(root, relative), os.DirFS(filepath.Join(repository, relative))); err != nil {
			t.Fatal(err)
		}
	}
	// 構造メタデータだけを基に、一時 directory 内に別 identity の合成 manifest を作る。
	corpus, err := legalquerycorpus.LoadManifest(t.Context(), repository, "testdata/legalquery/corpus-v16")
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]json.RawMessage
	if err := json.Unmarshal(corpus.RawBytes(), &manifest); err != nil {
		t.Fatal(err)
	}
	manifest["corpusVersion"] = json.RawMessage(`"corpus-v99"`)
	manifest["holdoutDigest"] = json.RawMessage(`"` + strings.Repeat("a", 64) + `"`)
	manifest["holdoutLeakageGroupDigests"] = json.RawMessage(`["` + strings.Repeat("b", 64) + `"]`)
	corpusRoot := filepath.Join(root, "testdata/legalquery/corpus-v99")
	sets := make(map[string]any)
	for _, set := range []string{"development", "holdout", "execution"} {
		if err := os.MkdirAll(filepath.Join(corpusRoot, set), 0o750); err != nil {
			t.Fatal(err)
		}
		sets[set] = map[string]any{
			"caseCount": 1,
			"cases":     []map[string]string{{"caseId": set + "-synthetic", "sha256": strings.Repeat("c", 64)}},
		}
	}
	manifest["sets"] = prepareCanonicalJSON(t, sets)
	if err := os.WriteFile(filepath.Join(corpusRoot, "manifest.json"), prepareCanonicalJSON(t, manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}
