package legalquerycorpus

import (
	"context"
	"errors"
	"fmt"
)

// Load は、閉域化した成果物を検証して不変な評価コーパスを返す。
func Load(
	ctx context.Context,
	repositoryRoot string,
	corpusDirectory string,
) (corpus Corpus, err error) {
	filesystem, err := openCorpusFilesystem(
		ctx,
		repositoryRoot,
		corpusDirectory,
	)
	if err != nil {
		return Corpus{}, err
	}
	return loadAndCloseCorpus(
		ctx,
		filesystem,
		loadOpenCorpus,
		filesystem.close,
	)
}

func loadAndCloseCorpus(
	ctx context.Context,
	filesystem *corpusFilesystem,
	load func(context.Context, *corpusFilesystem) (Corpus, error),
	closeFilesystem func() error,
) (corpus Corpus, err error) {
	defer func() {
		closeErr := closeFilesystem()
		if closeErr == nil {
			return
		}
		corpus = Corpus{}
		err = errors.Join(err, closeErr)
	}()

	corpus, err = load(ctx, filesystem)
	if err != nil {
		return Corpus{}, err
	}
	return corpus, nil
}

func loadOpenCorpus(
	ctx context.Context,
	filesystem *corpusFilesystem,
) (Corpus, error) {
	if filesystem == nil {
		return Corpus{}, fmt.Errorf(
			"corpus loader の filesystem が初期化されていません",
		)
	}
	manifestData := filesystem.manifestBytes()
	header, err := inspectJSONDocument(manifestData)
	if err != nil {
		return Corpus{}, err
	}
	if header.artifactKind != ArtifactKindCorpusManifest {
		return Corpus{}, fmt.Errorf(
			"corpus root の manifest は corpus_manifest でなければなりません",
		)
	}
	if header.schemaVersion != corpusSchemaVersion {
		return Corpus{}, fmt.Errorf(
			"manifest の schemaVersion は実装済みではありません",
		)
	}
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return Corpus{}, err
	}

	schemaData, err := filesystem.readSchemaV1(ctx)
	if err != nil {
		return Corpus{}, err
	}
	schema, err := newCorpusSchemaV1(schemaData)
	if err != nil {
		return Corpus{}, err
	}
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return Corpus{}, err
	}

	artifact, err := schema.validateAndDecode(manifestData, header)
	if err != nil {
		return Corpus{}, err
	}
	if artifact.kind != ArtifactKindCorpusManifest {
		return Corpus{}, fmt.Errorf(
			"corpus root の manifest を復元できません",
		)
	}
	checked, err := validateManifestIntegrity(
		ctx,
		filesystem,
		schema,
		artifact.manifest,
	)
	if err != nil {
		return Corpus{}, err
	}
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return Corpus{}, err
	}
	return newCorpus(
		checked.manifest,
		checked.development,
		checked.holdout,
		checked.execution,
	)
}
