package legalquerycorpus

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LoadDevelopment は、holdout、execution、manifest および評価結果を読まずに
// development fixture だけを検証して返す。
func LoadDevelopment(
	ctx context.Context,
	repositoryRoot string,
	developmentDirectory string,
) (development DevelopmentCorpus, err error) {
	filesystem, err := openDevelopmentFilesystem(
		ctx,
		repositoryRoot,
		developmentDirectory,
	)
	if err != nil {
		return DevelopmentCorpus{}, err
	}
	defer func() {
		if closeErr := filesystem.close(); closeErr != nil {
			development = DevelopmentCorpus{}
			err = errors.Join(err, closeErr)
		}
	}()
	return loadOpenDevelopment(ctx, filesystem)
}

func loadOpenDevelopment(
	ctx context.Context,
	filesystem *corpusFilesystem,
) (DevelopmentCorpus, error) {
	if filesystem == nil || filesystem.development.root == nil {
		return DevelopmentCorpus{}, fmt.Errorf(
			"development loader の filesystem が初期化されていません",
		)
	}
	if len(filesystem.development.files) == 0 {
		return DevelopmentCorpus{}, fmt.Errorf("development fixture がありません")
	}
	reader := filesystem.newFixtureReader()
	var schema corpusSchema
	schemaVersion := 0
	cases := make([]SemanticCase, 0, len(filesystem.development.files))
	digests := make(
		[]fixtureDigestEntry,
		0,
		len(filesystem.development.files),
	)
	for _, file := range filesystem.development.files {
		if err := checkCorpusFilesystemContext(ctx); err != nil {
			return DevelopmentCorpus{}, err
		}
		caseID := strings.TrimSuffix(file.name, ".json")
		data, err := reader.read(ctx, ManifestSetDevelopment, caseID)
		if err != nil {
			return DevelopmentCorpus{}, err
		}
		header, err := inspectJSONDocument(data)
		if err != nil {
			return DevelopmentCorpus{}, err
		}
		if schemaVersion == 0 {
			schemaVersion = header.schemaVersion
			schema, err = loadDevelopmentSchema(
				ctx,
				filesystem,
				schemaVersion,
			)
			if err != nil {
				return DevelopmentCorpus{}, err
			}
		}
		if header.schemaVersion != schemaVersion {
			return DevelopmentCorpus{}, fmt.Errorf(
				"development fixture の schemaVersion が一致しません",
			)
		}
		artifact, err := schema.validateAndDecode(data, header)
		if err != nil {
			return DevelopmentCorpus{}, err
		}
		semanticCase := artifact.semanticCase
		if artifact.kind != ArtifactKindSemanticCase ||
			semanticCase.CaseID() != caseID ||
			validateManifestCaseID(
				ManifestSetDevelopment,
				semanticCase.CaseID(),
			) != nil {
			return DevelopmentCorpus{}, fmt.Errorf(
				"development fixture の種別または caseId が不正です",
			)
		}
		cases = append(cases, semanticCase)
		digests = append(digests, fixtureDigestEntry{
			caseID: semanticCase.CaseID(),
			sha256: sha256Hex(data),
		})
	}
	return newDevelopmentCorpus(
		filesystem.corpusVersion,
		schemaVersion,
		computeFixtureDigest(digests),
		cases,
	)
}

func loadDevelopmentSchema(
	ctx context.Context,
	filesystem *corpusFilesystem,
	schemaVersion int,
) (corpusSchema, error) {
	if !isSupportedCorpusSchemaVersion(schemaVersion) {
		return corpusSchema{}, fmt.Errorf(
			"development schemaVersion は実装済みではありません",
		)
	}
	data, err := filesystem.readSchema(ctx, schemaVersion)
	if err != nil {
		return corpusSchema{}, err
	}
	return newCorpusSchema(schemaVersion, data)
}
