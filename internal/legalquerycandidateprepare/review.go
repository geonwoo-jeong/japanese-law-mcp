package legalquerycandidateprepare

import (
	"bytes"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

// BuildReviewAttestation は、独立 reviewer の点数を exact candidate byte へ固定する。
func BuildReviewAttestation(
	manifest legalquerycandidateeval.CandidateContentManifest,
	manifestRaw []byte,
	scope string,
	authorityID string,
	references []legalquerycandidateeval.SOTReference,
	scores []legalquerycandidateeval.CriterionScore,
	majorCount int,
	minorCount int,
) (legalquerycandidateeval.ReviewAttestation, error) {
	if err := verifyCanonicalCandidateManifest(manifest, manifestRaw); err != nil {
		return legalquerycandidateeval.ReviewAttestation{}, err
	}
	score100 := 0
	for _, score := range scores {
		score100 += score.Score20
	}
	review := legalquerycandidateeval.ReviewAttestation{
		ArtifactKind:                   legalquerycandidateeval.ArtifactKindReviewAttestation,
		SchemaVersion:                  legalquerycandidateeval.SchemaVersionV2,
		CandidateContentID:             manifest.CandidateContentID,
		CandidateContentManifestSHA256: legalquerycandidateeval.RawSHA256(manifestRaw),
		ReviewScope:                    scope,
		RubricVersion:                  legalquerycandidateeval.ReviewRubricVersion,
		RubricSHA256:                   legalquerycandidateeval.ReviewRubricSHA256(),
		ReviewerAuthorityID:            authorityID,
		ReviewedSOTs:                   cloneSOTReferences(references),
		ReviewedSOTSetSHA256:           legalquerycandidateeval.SOTSetSHA256(references),
		CriterionScores:                append([]legalquerycandidateeval.CriterionScore(nil), scores...),
		Score100:                       score100,
		BlockerCount:                   0,
		MajorCount:                     majorCount,
		MinorCount:                     minorCount,
		Decision:                       legalquerycandidateeval.ReviewDecisionApproved,
	}
	var err error
	review.AttestationID, err = legalquerycandidateeval.CanonicalReviewAttestationID(review)
	if err != nil {
		return legalquerycandidateeval.ReviewAttestation{}, err
	}
	if err := validateBuiltReview(review); err != nil {
		return legalquerycandidateeval.ReviewAttestation{}, err
	}
	return review, nil
}

func verifyCanonicalCandidateManifest(
	manifest legalquerycandidateeval.CandidateContentManifest,
	raw []byte,
) error {
	decoded, err := legalquerycandidateeval.DecodeCandidateContentManifest(raw)
	if err != nil {
		return err
	}
	canonical, err := legalquerycandidateeval.MarshalCanonicalJSON(manifest)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, canonical) || decoded.CandidateContentID != manifest.CandidateContentID {
		return fmt.Errorf("candidate manifest の typed 値と原 byte が一致しません")
	}
	return nil
}

func validateBuiltReview(review legalquerycandidateeval.ReviewAttestation) error {
	raw, err := legalquerycandidateeval.MarshalCanonicalJSON(review)
	if err != nil {
		return err
	}
	if _, err := legalquerycandidateeval.DecodeReviewAttestation(raw); err != nil {
		return fmt.Errorf("review attestation を自己検証できません: %w", err)
	}
	return nil
}

func cloneSOTReferences(
	values []legalquerycandidateeval.SOTReference,
) []legalquerycandidateeval.SOTReference {
	return append([]legalquerycandidateeval.SOTReference(nil), values...)
}
