package legalquerycandidateprepare

import (
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

func TestBuildReviewAndRequestは同じContentとSOT集合へ固定する(t *testing.T) {
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
	references, err := BuildRequiredSOTReferences(t.Context(), repository)
	if err != nil {
		t.Fatalf("review SOT 集合を解決できません: %v", err)
	}
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
		t.Context(), repository, "corpus-v10", manifest, manifestRaw,
		architecture, architectureRaw, testability, testabilityRaw,
		"default-2",
	)
	if err != nil {
		t.Fatalf("candidate-evaluation-request-identity: request を構成できません: %v", err)
	}
	requestRaw, err := legalquerycandidateeval.MarshalCanonicalJSON(request)
	if err != nil {
		t.Fatalf("request を直列化できません: %v", err)
	}
	if _, err := legalquerycandidateeval.DecodeEvaluationRequest(requestRaw); err != nil {
		t.Fatalf("candidate-evaluation-request-identity: request を自己検証できません: %v", err)
	}
	if request.CandidateContentManifestSHA256 !=
		legalquerycandidateeval.RawSHA256(manifestRaw) ||
		request.RequiredReviewSOTSetSHA256 !=
			legalquerycandidateeval.SOTSetSHA256(references) ||
		request.EvaluatorVersion != "legal-query-evaluator-v1" ||
		request.BaselineVersion != "default-2" {
		t.Fatalf("candidate-evaluation-review-content-binding: request binding = %#v", request)
	}
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
