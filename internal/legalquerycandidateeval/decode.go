package legalquerycandidateeval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

const (
	maximumSchemaBytes        = 1 << 20
	maximumPointerBytes       = 64 << 10
	maximumManifestBytes      = 4 << 20
	maximumAttestationBytes   = 256 << 10
	maximumRequestBytes       = 256 << 10
	maximumResultBytes        = 256 << 10
	maximumDocumentDepth      = 16
	maximumSmallValueCount    = 8192
	maximumManifestValueCount = 65536
)

// DecodePointer は closed canonical pointer を検証して返す。
func DecodePointer(raw []byte) (PointerDocument, error) {
	schemas, err := loadCanonicalArtifactSchemas()
	if err != nil {
		return PointerDocument{}, err
	}
	return decodePointer(context.Background(), raw, schemas)
}

func decodePointer(
	ctx context.Context,
	raw []byte,
	schemas artifactSchemas,
) (PointerDocument, error) {
	var document PointerDocument
	if err := decodeCanonical(
		ctx,
		raw,
		maximumPointerBytes,
		maximumSmallValueCount,
		ArtifactKindPointer,
		schemas,
		&document,
	); err != nil {
		return PointerDocument{}, fmt.Errorf("candidate evaluation pointer が不正です: %w", err)
	}
	if err := validatePointer(document); err != nil {
		return PointerDocument{}, err
	}
	return document, nil
}

// DecodeCandidateContentManifest は closed canonical manifest を検証して返す。
func DecodeCandidateContentManifest(raw []byte) (CandidateContentManifest, error) {
	schemas, err := loadCanonicalArtifactSchemas()
	if err != nil {
		return CandidateContentManifest{}, err
	}
	return decodeCandidateContentManifest(context.Background(), raw, schemas)
}

func decodeCandidateContentManifest(
	ctx context.Context,
	raw []byte,
	schemas artifactSchemas,
) (CandidateContentManifest, error) {
	var document CandidateContentManifest
	if err := decodeCanonical(
		ctx,
		raw,
		maximumManifestBytes,
		maximumManifestValueCount,
		ArtifactKindCandidateContent,
		schemas,
		&document,
	); err != nil {
		return CandidateContentManifest{}, fmt.Errorf("candidate content manifest が不正です: %w", err)
	}
	if err := validateCandidateContent(document); err != nil {
		return CandidateContentManifest{}, err
	}
	return document, nil
}

// DecodeReviewAttestation は closed canonical review assertion を検証して返す。
func DecodeReviewAttestation(raw []byte) (ReviewAttestation, error) {
	schemas, err := loadCanonicalArtifactSchemas()
	if err != nil {
		return ReviewAttestation{}, err
	}
	return decodeReviewAttestation(context.Background(), raw, schemas)
}

func decodeReviewAttestation(
	ctx context.Context,
	raw []byte,
	schemas artifactSchemas,
) (ReviewAttestation, error) {
	var document ReviewAttestation
	if err := decodeCanonical(
		ctx,
		raw,
		maximumAttestationBytes,
		maximumSmallValueCount,
		ArtifactKindReviewAttestation,
		schemas,
		&document,
	); err != nil {
		return ReviewAttestation{}, fmt.Errorf("review attestation が不正です: %w", err)
	}
	if err := validateReviewAttestation(document); err != nil {
		return ReviewAttestation{}, err
	}
	return document, nil
}

// DecodeEvaluationRequest は closed canonical request を検証して返す。
func DecodeEvaluationRequest(raw []byte) (EvaluationRequest, error) {
	schemas, err := loadCanonicalArtifactSchemas()
	if err != nil {
		return EvaluationRequest{}, err
	}
	return decodeEvaluationRequest(context.Background(), raw, schemas)
}

func decodeEvaluationRequest(
	ctx context.Context,
	raw []byte,
	schemas artifactSchemas,
) (EvaluationRequest, error) {
	var document EvaluationRequest
	if err := decodeCanonical(
		ctx,
		raw,
		maximumRequestBytes,
		maximumSmallValueCount,
		ArtifactKindEvaluationRequest,
		schemas,
		&document,
	); err != nil {
		return EvaluationRequest{}, fmt.Errorf("candidate evaluation request が不正です: %w", err)
	}
	if err := validateEvaluationRequest(document); err != nil {
		return EvaluationRequest{}, err
	}
	return document, nil
}

// DecodeEvaluationResult は closed canonical result を検証して返す。
func DecodeEvaluationResult(raw []byte) (EvaluationResult, error) {
	schemas, err := loadCanonicalArtifactSchemas()
	if err != nil {
		return EvaluationResult{}, err
	}
	return decodeEvaluationResult(context.Background(), raw, schemas)
}

func decodeEvaluationResult(
	ctx context.Context,
	raw []byte,
	schemas artifactSchemas,
) (EvaluationResult, error) {
	var document EvaluationResult
	if err := decodeCanonical(
		ctx,
		raw,
		maximumResultBytes,
		maximumSmallValueCount,
		ArtifactKindEvaluationResult,
		schemas,
		&document,
	); err != nil {
		return EvaluationResult{}, fmt.Errorf("candidate evaluation result が不正です: %w", err)
	}
	if err := validateEvaluationResult(document); err != nil {
		return EvaluationResult{}, err
	}
	return document, nil
}

type artifactHeader struct {
	ArtifactKind  string `json:"artifactKind"`
	SchemaVersion int    `json:"schemaVersion"`
}

func decodeCanonical(
	ctx context.Context,
	raw []byte,
	maximumBytes int,
	maximumValues int,
	expectedArtifactKind string,
	schemas artifactSchemas,
	target any,
) error {
	if err := checkContext(ctx); err != nil {
		return err
	}
	if len(raw) == 0 || len(raw) > maximumBytes {
		return fmt.Errorf("原 byte の size が上限外です")
	}
	if err := legalqueryartifact.InspectJSONObject(raw, legalqueryartifact.JSONLimits{
		Depth: maximumDocumentDepth, Values: maximumValues, RejectNull: true,
	}); err != nil {
		return err
	}
	header, err := decodeArtifactHeader(raw, expectedArtifactKind)
	if err != nil {
		return err
	}
	if err := schemas.validate(ctx, header.SchemaVersion, raw); err != nil {
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

func decodeArtifactHeader(raw []byte, expectedArtifactKind string) (artifactHeader, error) {
	var header artifactHeader
	if err := json.Unmarshal(raw, &header); err != nil {
		return artifactHeader{}, fmt.Errorf("candidate evaluation 成果物 header を解釈できません")
	}
	if header.ArtifactKind != expectedArtifactKind {
		return artifactHeader{}, fmt.Errorf("artifactKind が期待する成果物と一致しません")
	}
	if !isSupportedSchemaVersion(header.SchemaVersion) {
		return artifactHeader{}, fmt.Errorf("schemaVersion が未対応です")
	}
	return header, nil
}
