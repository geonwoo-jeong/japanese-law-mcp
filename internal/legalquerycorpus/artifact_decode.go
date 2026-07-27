package legalquerycorpus

import "fmt"

// decodedCorpusArtifact は、v1 の閉じた三 variant のうち一つだけを保持する。
type decodedCorpusArtifact struct {
	kind          ArtifactKind
	manifest      Manifest
	semanticCase  SemanticCase
	executionCase ExecutionCase
}

// validateAndDecode は、同じ原 byte を schema 検証してから v1 DTO へ復元する。
func (s corpusSchemaV1) validateAndDecode(
	data []byte,
	header jsonDocumentHeader,
) (decodedCorpusArtifact, error) {
	if s.resolved == nil {
		return decodedCorpusArtifact{}, fmt.Errorf(
			"corpus schema v1 が初期化されていません",
		)
	}
	inspected, err := inspectJSONDocument(data)
	if err != nil {
		return decodedCorpusArtifact{}, err
	}
	if inspected != header {
		return decodedCorpusArtifact{}, fmt.Errorf(
			"JSON 成果物の検査済み header が一致しません",
		)
	}
	if header.schemaVersion != corpusSchemaVersion {
		return decodedCorpusArtifact{}, fmt.Errorf(
			"JSON 成果物の schemaVersion は実装済みではありません",
		)
	}
	if !isSupportedCorpusArtifactKind(header.artifactKind) {
		return decodedCorpusArtifact{}, fmt.Errorf(
			"JSON 成果物の artifactKind は実装済みではありません",
		)
	}
	if err := s.validate(data); err != nil {
		return decodedCorpusArtifact{}, err
	}
	return decodeCorpusArtifactV1(data, header.artifactKind)
}

func isSupportedCorpusArtifactKind(kind ArtifactKind) bool {
	switch kind {
	case ArtifactKindCorpusManifest,
		ArtifactKindSemanticCase,
		ArtifactKindExecutionCase:
		return true
	default:
		return false
	}
}

func decodeCorpusArtifactV1(
	data []byte,
	kind ArtifactKind,
) (decodedCorpusArtifact, error) {
	switch kind {
	case ArtifactKindCorpusManifest:
		manifest, err := decodeManifestV1(data)
		if err != nil {
			return decodedCorpusArtifact{}, fmt.Errorf(
				"manifest を v1 成果物へ復元できません: %w",
				err,
			)
		}
		return decodedCorpusArtifact{
			kind:     ArtifactKindCorpusManifest,
			manifest: manifest,
		}, nil
	case ArtifactKindSemanticCase:
		semanticCase, err := decodeSemanticCaseV1(data)
		if err != nil {
			return decodedCorpusArtifact{}, fmt.Errorf(
				"semantic case を v1 成果物へ復元できません: %w",
				err,
			)
		}
		return decodedCorpusArtifact{
			kind:         ArtifactKindSemanticCase,
			semanticCase: semanticCase,
		}, nil
	case ArtifactKindExecutionCase:
		executionCase, err := decodeExecutionCaseV1(data)
		if err != nil {
			return decodedCorpusArtifact{}, fmt.Errorf(
				"execution case を v1 成果物へ復元できません: %w",
				err,
			)
		}
		return decodedCorpusArtifact{
			kind:          ArtifactKindExecutionCase,
			executionCase: executionCase,
		}, nil
	default:
		return decodedCorpusArtifact{}, fmt.Errorf(
			"JSON 成果物の artifactKind は実装済みではありません",
		)
	}
}
