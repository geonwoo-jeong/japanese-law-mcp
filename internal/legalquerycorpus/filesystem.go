package legalquerycorpus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
)

const (
	corpusSchemaMaximumBytes       int64 = 1 << 20
	corpusManifestMaximumBytes     int64 = 2 << 20
	corpusFixtureMaximumBytes      int64 = 256 << 10
	corpusFixtureMaximumCount            = 4096
	corpusFixtureMaximumTotalBytes int64 = 64 << 20
)

type corpusFixtureFile struct {
	name string
	info os.FileInfo
}

type corpusSetFilesystem struct {
	root  *os.Root
	files []corpusFixtureFile
}

type corpusFilesystem struct {
	repositoryRoot *os.Root
	testdataRoot   *os.Root
	legalQueryRoot *os.Root
	schemasRoot    *os.Root
	corpusRoot     *os.Root
	development    corpusSetFilesystem
	holdout        corpusSetFilesystem
	execution      corpusSetFilesystem
	corpusVersion  string
	manifest       []byte
}

// openCorpusFilesystem は、閉域化した root と検証済み file 一覧を開く。
func openCorpusFilesystem(
	ctx context.Context,
	repositoryRoot string,
	corpusDirectory string,
) (filesystem *corpusFilesystem, err error) {
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return nil, err
	}
	paths, err := resolveCorpusFilesystemPaths(repositoryRoot, corpusDirectory)
	if err != nil {
		return nil, err
	}
	candidate := &corpusFilesystem{corpusVersion: paths.corpusVersion}
	defer func() {
		if err != nil {
			_ = candidate.close()
		}
	}()
	if err = candidate.openRoots(ctx, paths.repositoryRoot); err != nil {
		return nil, err
	}
	manifestInfo, err := validateCorpusRootEntries(ctx, candidate.corpusRoot)
	if err != nil {
		return nil, err
	}
	if err = candidate.openAndScanSets(ctx); err != nil {
		return nil, err
	}
	candidate.manifest, err = readRegularCorpusFile(
		ctx,
		candidate.corpusRoot,
		"manifest.json",
		corpusManifestMaximumBytes,
		manifestInfo,
	)
	if err != nil {
		return nil, err
	}
	return candidate, nil
}

func (f *corpusFilesystem) openRoots(
	ctx context.Context,
	repositoryRoot string,
) error {
	var err error
	f.repositoryRoot, err = openRepositoryFilesystemRoot(repositoryRoot)
	if err != nil {
		return err
	}
	if f.testdataRoot, err = openVerifiedChildRoot(
		ctx,
		f.repositoryRoot,
		"testdata",
	); err != nil {
		return err
	}
	if f.legalQueryRoot, err = openVerifiedChildRoot(
		ctx,
		f.testdataRoot,
		"legalquery",
	); err != nil {
		return err
	}
	if f.schemasRoot, err = openVerifiedChildRoot(
		ctx,
		f.legalQueryRoot,
		"schemas",
	); err != nil {
		return err
	}
	f.corpusRoot, err = openVerifiedChildRoot(
		ctx,
		f.legalQueryRoot,
		f.corpusVersion,
	)
	return err
}

func (f *corpusFilesystem) openAndScanSets(ctx context.Context) error {
	var err error
	if f.development.root, err = openVerifiedChildRoot(
		ctx,
		f.corpusRoot,
		string(ManifestSetDevelopment),
	); err != nil {
		return err
	}
	if f.holdout.root, err = openVerifiedChildRoot(
		ctx,
		f.corpusRoot,
		string(ManifestSetHoldout),
	); err != nil {
		return err
	}
	if f.execution.root, err = openVerifiedChildRoot(
		ctx,
		f.corpusRoot,
		string(ManifestSetExecution),
	); err != nil {
		return err
	}
	fixtureCount := 0
	var fixtureBytes int64
	for _, set := range []*corpusSetFilesystem{
		&f.development,
		&f.holdout,
		&f.execution,
	} {
		set.files, err = scanCorpusSetEntries(
			ctx,
			set.root,
			&fixtureCount,
			&fixtureBytes,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *corpusFilesystem) manifestBytes() []byte {
	if f == nil {
		return nil
	}
	return bytes.Clone(f.manifest)
}

func (f *corpusFilesystem) fixtureFileNames(
	kind ManifestSetKind,
) ([]string, error) {
	set, err := f.fixtureSet(kind)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(set.files))
	for index, file := range set.files {
		names[index] = file.name
	}
	return names, nil
}

func (f *corpusFilesystem) fixtureSet(
	kind ManifestSetKind,
) (*corpusSetFilesystem, error) {
	if f == nil {
		return nil, fmt.Errorf("corpus filesystem が初期化されていません")
	}
	switch kind {
	case ManifestSetDevelopment:
		return &f.development, nil
	case ManifestSetHoldout:
		return &f.holdout, nil
	case ManifestSetExecution:
		return &f.execution, nil
	default:
		return nil, fmt.Errorf("fixture の集合が定義されていません")
	}
}

func (f *corpusFilesystem) close() error {
	if f == nil {
		return nil
	}
	return errors.Join(
		closeCorpusFilesystemRoot(&f.execution.root),
		closeCorpusFilesystemRoot(&f.holdout.root),
		closeCorpusFilesystemRoot(&f.development.root),
		closeCorpusFilesystemRoot(&f.corpusRoot),
		closeCorpusFilesystemRoot(&f.schemasRoot),
		closeCorpusFilesystemRoot(&f.legalQueryRoot),
		closeCorpusFilesystemRoot(&f.testdataRoot),
		closeCorpusFilesystemRoot(&f.repositoryRoot),
	)
}

func closeCorpusFilesystemRoot(root **os.Root) error {
	if root == nil || *root == nil {
		return nil
	}
	err := (*root).Close()
	*root = nil
	if err != nil {
		return fmt.Errorf("corpus filesystem の root を閉じられません")
	}
	return nil
}
