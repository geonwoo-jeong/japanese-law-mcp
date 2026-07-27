package legalquerycorpus

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
)

const corpusSchemaV1Filename = "legal-query-corpus-v1.schema.json"

type corpusFixtureReader struct {
	filesystem *corpusFilesystem
	readFiles  map[string]struct{}
	totalBytes int64
}

func (f *corpusFilesystem) readSchemaV1(
	ctx context.Context,
) ([]byte, error) {
	if f == nil || f.schemasRoot == nil {
		return nil, fmt.Errorf("corpus schema の filesystem が初期化されていません")
	}
	return readRegularCorpusFile(
		ctx,
		f.schemasRoot,
		corpusSchemaV1Filename,
		corpusSchemaMaximumBytes,
		nil,
	)
}

func (f *corpusFilesystem) newFixtureReader() *corpusFixtureReader {
	return &corpusFixtureReader{
		filesystem: f,
		readFiles:  make(map[string]struct{}),
	}
}

func (r *corpusFixtureReader) read(
	ctx context.Context,
	kind ManifestSetKind,
	caseID string,
) ([]byte, error) {
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return nil, err
	}
	if r == nil || r.filesystem == nil {
		return nil, fmt.Errorf("fixture reader が初期化されていません")
	}
	if err := validateManifestCaseID(kind, caseID); err != nil {
		return nil, fmt.Errorf("fixture の caseId または集合が有効ではありません")
	}
	set, err := r.filesystem.fixtureSet(kind)
	if err != nil {
		return nil, err
	}
	filename := caseID + ".json"
	file, exists := findCorpusFixtureFile(set.files, filename)
	if !exists {
		return nil, fmt.Errorf("fixture file が列挙結果に存在しません")
	}
	key := string(kind) + "/" + filename
	if _, exists := r.readFiles[key]; exists {
		return nil, fmt.Errorf("同じ fixture file を二回読むことはできません")
	}
	data, err := readRegularCorpusFile(
		ctx,
		set.root,
		filename,
		corpusFixtureMaximumBytes,
		file.info,
	)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > corpusFixtureMaximumTotalBytes-r.totalBytes {
		return nil, fmt.Errorf("fixture の原 byte 合計が上限を超えています")
	}
	r.totalBytes += int64(len(data))
	r.readFiles[key] = struct{}{}
	return data, nil
}

func findCorpusFixtureFile(
	files []corpusFixtureFile,
	name string,
) (corpusFixtureFile, bool) {
	index := sort.Search(len(files), func(index int) bool {
		return files[index].name >= name
	})
	if index >= len(files) || files[index].name != name {
		return corpusFixtureFile{}, false
	}
	return files[index], true
}

func readRegularCorpusFile(
	ctx context.Context,
	root *os.Root,
	name string,
	maximumBytes int64,
	expected os.FileInfo,
) (data []byte, err error) {
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return nil, err
	}
	current, err := lstatRegularCorpusFile(root, name, maximumBytes)
	if err != nil {
		return nil, err
	}
	if expected != nil &&
		(!os.SameFile(expected, current) || expected.Size() != current.Size()) {
		return nil, fmt.Errorf("corpus 成果物 file が検証後に変化しました")
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("corpus 成果物 file を開けません")
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			data = nil
			err = fmt.Errorf("corpus 成果物 file を閉じられません")
		}
	}()
	opened, err := validateOpenedCorpusFile(file, current, maximumBytes)
	if err != nil {
		return nil, err
	}
	data, err = io.ReadAll(io.LimitReader(file, maximumBytes+1))
	if err != nil {
		return nil, fmt.Errorf("corpus 成果物 file を読めません")
	}
	if int64(len(data)) > maximumBytes ||
		int64(len(data)) != opened.Size() {
		return nil, fmt.Errorf("corpus 成果物 file の size が読取り中に変化しました")
	}
	if err := verifyReadCorpusFile(root, file, name, opened); err != nil {
		return nil, err
	}
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return nil, err
	}
	return data, nil
}

func validateOpenedCorpusFile(
	file *os.File,
	expected os.FileInfo,
	maximumBytes int64,
) (os.FileInfo, error) {
	opened, err := file.Stat()
	if err != nil ||
		!opened.Mode().IsRegular() ||
		opened.Size() < 0 ||
		opened.Size() > maximumBytes ||
		!os.SameFile(expected, opened) {
		return nil, fmt.Errorf("開いた corpus 成果物 file が有効ではありません")
	}
	return opened, nil
}

func verifyReadCorpusFile(
	root *os.Root,
	file *os.File,
	name string,
	opened os.FileInfo,
) error {
	afterRead, err := file.Stat()
	if err != nil ||
		!afterRead.Mode().IsRegular() ||
		afterRead.Size() != opened.Size() ||
		!os.SameFile(opened, afterRead) {
		return fmt.Errorf("corpus 成果物 file が読取り中に変化しました")
	}
	current, err := root.Lstat(name)
	if err != nil ||
		current.Mode()&os.ModeSymlink != 0 ||
		!current.Mode().IsRegular() ||
		current.Size() != opened.Size() ||
		!os.SameFile(opened, current) {
		return fmt.Errorf("corpus 成果物 file が読取り中に変化しました")
	}
	return nil
}
