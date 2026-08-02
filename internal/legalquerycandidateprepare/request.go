package legalquerycandidateprepare

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/evaluators"
)

// BuildEvaluationRequest は、review 済み候補と corpus manifest を一 request へ固定する。
func BuildEvaluationRequest(
	ctx context.Context,
	repositoryRoot string,
	corpusVersion string,
	manifest legalquerycandidateeval.CandidateContentManifest,
	manifestRaw []byte,
	architecture legalquerycandidateeval.ReviewAttestation,
	architectureRaw []byte,
	testability legalquerycandidateeval.ReviewAttestation,
	testabilityRaw []byte,
	baselineVersion string,
) (legalquerycandidateeval.EvaluationRequest, error) {
	if err := verifyCanonicalCandidateManifest(manifest, manifestRaw); err != nil {
		return legalquerycandidateeval.EvaluationRequest{}, err
	}
	references, err := BuildRequiredSOTReferences(ctx, repositoryRoot)
	if err != nil {
		return legalquerycandidateeval.EvaluationRequest{}, err
	}
	if err := verifyRequestReviews(
		manifest,
		manifestRaw,
		references,
		architecture,
		architectureRaw,
		testability,
		testabilityRaw,
	); err != nil {
		return legalquerycandidateeval.EvaluationRequest{}, err
	}
	corpus, err := legalquerycorpus.LoadManifest(
		ctx,
		repositoryRoot,
		"testdata/legalquery/"+corpusVersion,
	)
	if err != nil {
		return legalquerycandidateeval.EvaluationRequest{}, err
	}
	if err := validateCandidateEvaluationCorpus(
		corpus.Manifest().SchemaVersion(),
		corpus.Manifest().CorpusVersion(),
	); err != nil {
		return legalquerycandidateeval.EvaluationRequest{}, err
	}
	request := newEvaluationRequest(
		corpus,
		manifest,
		manifestRaw,
		references,
		architecture,
		architectureRaw,
		testability,
		testabilityRaw,
		baselineVersion,
	)
	request.EvaluationID, err = legalquerycandidateeval.CanonicalEvaluationID(request)
	if err != nil {
		return legalquerycandidateeval.EvaluationRequest{}, err
	}
	return validateBuiltRequest(request)
}

func newEvaluationRequest(
	corpus legalquerycorpus.ManifestArtifact,
	manifest legalquerycandidateeval.CandidateContentManifest,
	manifestRaw []byte,
	references []legalquerycandidateeval.SOTReference,
	architecture legalquerycandidateeval.ReviewAttestation,
	architectureRaw []byte,
	testability legalquerycandidateeval.ReviewAttestation,
	testabilityRaw []byte,
	baselineVersion string,
) legalquerycandidateeval.EvaluationRequest {
	corpusManifest := corpus.Manifest()
	return legalquerycandidateeval.EvaluationRequest{
		ArtifactKind:                   legalquerycandidateeval.ArtifactKindEvaluationRequest,
		SchemaVersion:                  legalquerycandidateeval.SchemaVersionV2,
		EvaluatorVersion:               evaluators.Version1,
		CorpusVersion:                  corpusManifest.CorpusVersion(),
		CorpusManifestSHA256:           corpus.SHA256(),
		HoldoutDigest:                  corpusManifest.HoldoutDigest(),
		HoldoutLeakageGroupDigests:     corpusManifest.HoldoutLeakageGroupDigests(),
		CandidateContentID:             manifest.CandidateContentID,
		CandidateContentManifestSHA256: legalquerycandidateeval.RawSHA256(manifestRaw),
		ReviewRubricVersion:            legalquerycandidateeval.ReviewRubricVersion,
		ReviewRubricSHA256:             legalquerycandidateeval.ReviewRubricSHA256(),
		RequiredReviewSOTs:             cloneSOTReferences(references),
		RequiredReviewSOTSetSHA256:     legalquerycandidateeval.SOTSetSHA256(references),
		ReviewAttestations: []legalquerycandidateeval.ReviewAttestationReference{
			{
				ReviewScope:       legalquerycandidateeval.ReviewScopeArchitecture,
				AttestationID:     architecture.AttestationID,
				AttestationSHA256: legalquerycandidateeval.RawSHA256(architectureRaw),
			},
			{
				ReviewScope:       legalquerycandidateeval.ReviewScopeTestability,
				AttestationID:     testability.AttestationID,
				AttestationSHA256: legalquerycandidateeval.RawSHA256(testabilityRaw),
			},
		},
		BaselineVersion: baselineVersion,
	}
}

func verifyRequestReviews(
	manifest legalquerycandidateeval.CandidateContentManifest,
	manifestRaw []byte,
	references []legalquerycandidateeval.SOTReference,
	architecture legalquerycandidateeval.ReviewAttestation,
	architectureRaw []byte,
	testability legalquerycandidateeval.ReviewAttestation,
	testabilityRaw []byte,
) error {
	if err := verifyReviewRaw(architecture, architectureRaw); err != nil {
		return err
	}
	if err := verifyReviewRaw(testability, testabilityRaw); err != nil {
		return err
	}
	manifestDigest := legalquerycandidateeval.RawSHA256(manifestRaw)
	if architecture.ReviewScope != legalquerycandidateeval.ReviewScopeArchitecture ||
		testability.ReviewScope != legalquerycandidateeval.ReviewScopeTestability ||
		architecture.ReviewerAuthorityID == testability.ReviewerAuthorityID ||
		architecture.CandidateContentID != manifest.CandidateContentID ||
		testability.CandidateContentID != manifest.CandidateContentID ||
		architecture.CandidateContentManifestSHA256 != manifestDigest ||
		testability.CandidateContentManifestSHA256 != manifestDigest ||
		!reflect.DeepEqual(architecture.ReviewedSOTs, references) ||
		!reflect.DeepEqual(testability.ReviewedSOTs, references) {
		return fmt.Errorf("二件の review が candidate content と SOT 集合へ一致しません")
	}
	return nil
}

func verifyReviewRaw(
	review legalquerycandidateeval.ReviewAttestation,
	raw []byte,
) error {
	if _, err := legalquerycandidateeval.DecodeReviewAttestation(raw); err != nil {
		return err
	}
	canonical, err := legalquerycandidateeval.MarshalCanonicalJSON(review)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, canonical) {
		return fmt.Errorf("review typed 値と原 byte が一致しません")
	}
	return nil
}

func validateBuiltRequest(
	request legalquerycandidateeval.EvaluationRequest,
) (legalquerycandidateeval.EvaluationRequest, error) {
	raw, err := legalquerycandidateeval.MarshalCanonicalJSON(request)
	if err != nil {
		return legalquerycandidateeval.EvaluationRequest{}, err
	}
	if _, err := legalquerycandidateeval.DecodeEvaluationRequest(raw); err != nil {
		return legalquerycandidateeval.EvaluationRequest{},
			fmt.Errorf("evaluation request を自己検証できません: %w", err)
	}
	return request, nil
}
