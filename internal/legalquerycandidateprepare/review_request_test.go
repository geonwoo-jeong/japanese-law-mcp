package legalquerycandidateprepare

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

func TestBuildReviewは同じContentとHistoricalSOT集合へ固定する(t *testing.T) {
	t.Parallel()

	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repository を解決できません: %v", err)
	}
	manifest, err := BuildContentManifest(t.Context(), repository, validSourceSetForTest(t))
	if err != nil {
		t.Fatalf("candidate content を構成できません: %v", err)
	}
	manifestRaw, err := legalquerycandidateeval.MarshalCanonicalJSON(manifest)
	if err != nil {
		t.Fatalf("candidate content を直列化できません: %v", err)
	}
	references := fixedHistoricalSOTReferencesForTest(t, manifest.SchemaVersion)
	architecture := mustReviewForTest(
		t, manifest, manifestRaw, references,
		legalquerycandidateeval.ReviewScopeArchitecture,
		"review-authority-architecture-test",
	)
	testability := mustReviewForTest(
		t, manifest, manifestRaw, references,
		legalquerycandidateeval.ReviewScopeTestability,
		"review-authority-testability-test",
	)
	architectureRaw, err := legalquerycandidateeval.MarshalCanonicalJSON(architecture)
	if err != nil {
		t.Fatalf("architecture review を直列化できません: %v", err)
	}
	testabilityRaw, err := legalquerycandidateeval.MarshalCanonicalJSON(testability)
	if err != nil {
		t.Fatalf("testability review を直列化できません: %v", err)
	}
	request, err := BuildEvaluationRequest(
		t.Context(), repository, "corpus-v16", manifest, manifestRaw,
		architecture, architectureRaw, testability, testabilityRaw,
		"default-8",
	)
	if request.EvaluationID != "" || !legalquerycandidateeval.IsCurrentStale(err) {
		t.Fatalf("candidate-evaluation-stale-candidate-readiness-fail: stale schema から request を構成しました: request=%#v error=%v", request, err)
	}
	if architecture.SchemaVersion != legalquerycandidateeval.SchemaVersionV3 ||
		testability.SchemaVersion != legalquerycandidateeval.SchemaVersionV3 ||
		architecture.ReviewedSOTSetSHA256 != legalquerycandidateeval.SOTSetSHA256(references) ||
		testability.ReviewedSOTSetSHA256 != legalquerycandidateeval.SOTSetSHA256(references) {
		t.Fatalf("candidate-evaluation-review-content-binding: review binding=(%#v,%#v)", architecture, testability)
	}
}

func fixedHistoricalSOTReferencesForTest(
	t *testing.T,
	schemaVersion int,
) []legalquerycandidateeval.SOTReference {
	t.Helper()
	ids, err := legalquerycandidateeval.RequiredReviewSOTIDsForSchema(schemaVersion)
	if err != nil {
		t.Fatalf("schema version %d の SOT ID を解決できません: %v", schemaVersion, err)
	}
	references := make([]legalquerycandidateeval.SOTReference, 0, len(ids))
	for _, id := range ids {
		references = append(references, legalquerycandidateeval.SOTReference{
			SOTID:             id,
			SOTDocumentSHA256: strings.Repeat("a", 64),
		})
	}
	return references
}

func mustReviewForTest(
	t *testing.T,
	manifest legalquerycandidateeval.CandidateContentManifest,
	manifestRaw []byte,
	references []legalquerycandidateeval.SOTReference,
	scope string,
	authority string,
) legalquerycandidateeval.ReviewAttestation {
	t.Helper()
	criteria := legalquerycandidateeval.ArchitectureCriterionIDs()
	if scope == legalquerycandidateeval.ReviewScopeTestability {
		criteria = legalquerycandidateeval.TestabilityCriterionIDs()
	}
	scores := make([]legalquerycandidateeval.CriterionScore, 0, len(criteria))
	for _, criterion := range criteria {
		scores = append(scores, legalquerycandidateeval.CriterionScore{
			CriterionID: criterion, Score20: 16,
		})
	}
	review, err := BuildReviewAttestation(
		manifest, manifestRaw, scope, authority, references, scores, 0, 0,
	)
	if err != nil {
		t.Fatalf("review attestation を構成できません: %v", err)
	}
	return review
}
