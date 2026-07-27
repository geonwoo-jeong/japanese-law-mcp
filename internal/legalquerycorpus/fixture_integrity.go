package legalquerycorpus

import (
	"context"
	"fmt"
)

func loadSemanticIntegritySet(
	ctx context.Context,
	reader *corpusFixtureReader,
	schema corpusSchemaV1,
	schemaVersion int,
	set ManifestSet,
) ([]SemanticCase, []fixtureDigestEntry, error) {
	cases := make([]SemanticCase, 0, set.CaseCount())
	digests := make([]fixtureDigestEntry, 0, set.CaseCount())
	for _, entry := range set.Cases() {
		artifact, digest, err := readIntegrityFixture(
			ctx,
			reader,
			schema,
			set.Kind(),
			entry,
		)
		if err != nil {
			return nil, nil, err
		}
		if artifact.kind != ArtifactKindSemanticCase ||
			artifact.semanticCase.ArtifactKind() != ArtifactKindSemanticCase ||
			artifact.semanticCase.SchemaVersion() != schemaVersion ||
			artifact.semanticCase.CaseID() != entry.CaseID() ||
			validateManifestCaseID(
				set.Kind(),
				artifact.semanticCase.CaseID(),
			) != nil {
			return nil, nil, fmt.Errorf(
				"semantic fixture が manifest の集合、版または caseId と一致しません",
			)
		}
		cases = append(cases, artifact.semanticCase)
		digests = append(digests, fixtureDigestEntry{
			caseID: entry.CaseID(),
			sha256: digest,
		})
	}
	return cases, digests, nil
}

func loadExecutionIntegritySet(
	ctx context.Context,
	reader *corpusFixtureReader,
	schema corpusSchemaV1,
	schemaVersion int,
	set ManifestSet,
) ([]ExecutionCase, error) {
	cases := make([]ExecutionCase, 0, set.CaseCount())
	for _, entry := range set.Cases() {
		artifact, _, err := readIntegrityFixture(
			ctx,
			reader,
			schema,
			set.Kind(),
			entry,
		)
		if err != nil {
			return nil, err
		}
		if set.Kind() != ManifestSetExecution ||
			artifact.kind != ArtifactKindExecutionCase ||
			artifact.executionCase.ArtifactKind() !=
				ArtifactKindExecutionCase ||
			artifact.executionCase.SchemaVersion() != schemaVersion ||
			artifact.executionCase.CaseID() != entry.CaseID() ||
			validateManifestCaseID(
				ManifestSetExecution,
				artifact.executionCase.CaseID(),
			) != nil {
			return nil, fmt.Errorf(
				"execution fixture が manifest の集合、版または caseId と一致しません",
			)
		}
		cases = append(cases, artifact.executionCase)
	}
	return cases, nil
}

func readIntegrityFixture(
	ctx context.Context,
	reader *corpusFixtureReader,
	schema corpusSchemaV1,
	kind ManifestSetKind,
	entry ManifestEntry,
) (decodedCorpusArtifact, string, error) {
	data, err := reader.read(ctx, kind, entry.CaseID())
	if err != nil {
		return decodedCorpusArtifact{}, "", err
	}
	digest := sha256Hex(data)
	if digest != entry.SHA256() {
		return decodedCorpusArtifact{}, "", fmt.Errorf(
			"fixture の SHA-256 checksum が manifest と一致しません",
		)
	}
	header, err := inspectJSONDocument(data)
	if err != nil {
		return decodedCorpusArtifact{}, "", err
	}
	artifact, err := schema.validateAndDecode(data, header)
	if err != nil {
		return decodedCorpusArtifact{}, "", err
	}
	return artifact, digest, nil
}
