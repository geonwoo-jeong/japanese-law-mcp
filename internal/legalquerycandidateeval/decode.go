package legalquerycandidateeval

import (
	"bytes"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

const (
	maximumSchemaBytes        = 1 << 20
	maximumPointerBytes       = 64 << 10
	maximumManifestBytes      = 4 << 20
	maximumAttestationBytes   = 256 << 10
	maximumRequestBytes       = 256 << 10
	maximumDocumentDepth      = 16
	maximumSmallValueCount    = 8192
	maximumManifestValueCount = 65536
)

// DecodePointer は closed canonical pointer を検証して返す。
func DecodePointer(raw []byte) (PointerDocument, error) {
	var document PointerDocument
	if err := decodeCanonical(raw, maximumPointerBytes, maximumSmallValueCount, &document); err != nil {
		return PointerDocument{}, fmt.Errorf("candidate evaluation pointer が不正です: %w", err)
	}
	if err := validatePointer(document); err != nil {
		return PointerDocument{}, err
	}
	return document, nil
}

// DecodeCandidateContentManifest は closed canonical manifest を検証して返す。
func DecodeCandidateContentManifest(raw []byte) (CandidateContentManifest, error) {
	var document CandidateContentManifest
	if err := decodeCanonical(raw, maximumManifestBytes, maximumManifestValueCount, &document); err != nil {
		return CandidateContentManifest{}, fmt.Errorf("candidate content manifest が不正です: %w", err)
	}
	if err := validateCandidateContent(document); err != nil {
		return CandidateContentManifest{}, err
	}
	return document, nil
}

// DecodeReviewAttestation は closed canonical review assertion を検証して返す。
func DecodeReviewAttestation(raw []byte) (ReviewAttestation, error) {
	var document ReviewAttestation
	if err := decodeCanonical(raw, maximumAttestationBytes, maximumSmallValueCount, &document); err != nil {
		return ReviewAttestation{}, fmt.Errorf("review attestation が不正です: %w", err)
	}
	if err := validateReviewAttestation(document); err != nil {
		return ReviewAttestation{}, err
	}
	return document, nil
}

// DecodeEvaluationRequest は closed canonical request を検証して返す。
func DecodeEvaluationRequest(raw []byte) (EvaluationRequest, error) {
	var document EvaluationRequest
	if err := decodeCanonical(raw, maximumRequestBytes, maximumSmallValueCount, &document); err != nil {
		return EvaluationRequest{}, fmt.Errorf("candidate evaluation request が不正です: %w", err)
	}
	if err := validateEvaluationRequest(document); err != nil {
		return EvaluationRequest{}, err
	}
	return document, nil
}

func decodeCanonical(raw []byte, maximumBytes, maximumValues int, target any) error {
	if len(raw) == 0 || len(raw) > maximumBytes {
		return fmt.Errorf("原 byte の size が上限外です")
	}
	if err := validateCanonicalArtifactSchema(raw); err != nil {
		return err
	}
	if err := legalqueryartifact.InspectJSONObject(raw, legalqueryartifact.JSONLimits{
		Depth: maximumDocumentDepth, Values: maximumValues, RejectNull: true,
	}); err != nil {
		return err
	}
	if err := legalqueryartifact.DecodeClosed(raw, target); err != nil {
		return err
	}
	canonical, err := MarshalCanonicalJSON(target)
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, canonical) {
		return fmt.Errorf("原 byte が canonical JSON ではありません")
	}
	return nil
}
