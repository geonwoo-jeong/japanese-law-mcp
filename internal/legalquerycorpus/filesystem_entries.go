package legalquerycorpus

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const (
	corpusDirectoryReadBatchSize = 128
	corpusRootEntryCount         = 4
)

func validateCorpusRootEntries(
	ctx context.Context,
	root *os.Root,
) (os.FileInfo, error) {
	entries, err := readCorpusDirectoryEntries(ctx, root, corpusRootEntryCount)
	if err != nil {
		return nil, err
	}
	expected := map[string]struct{}{
		"manifest.json": {},
		"development":   {},
		"holdout":       {},
		"execution":     {},
	}
	if len(entries) != len(expected) {
		return nil, fmt.Errorf("corpus root の entry が定義と一致しません")
	}
	for _, entry := range entries {
		if _, exists := expected[entry.Name()]; !exists {
			return nil, fmt.Errorf("corpus root に未知の entry があります")
		}
	}
	return lstatRegularCorpusFile(
		root,
		"manifest.json",
		corpusManifestMaximumBytes,
	)
}

func scanCorpusSetEntries(
	ctx context.Context,
	root *os.Root,
	fixtureCount *int,
	fixtureBytes *int64,
) ([]corpusFixtureFile, error) {
	if fixtureCount == nil || fixtureBytes == nil {
		return nil, fmt.Errorf("fixture の予算状態が初期化されていません")
	}
	remaining := corpusFixtureMaximumCount - *fixtureCount
	entries, err := readCorpusDirectoryEntries(ctx, root, remaining)
	if err != nil {
		return nil, err
	}
	*fixtureCount += len(entries)
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Name() < entries[right].Name()
	})
	files := make([]corpusFixtureFile, 0, len(entries))
	for _, entry := range entries {
		if err := checkCorpusFilesystemContext(ctx); err != nil {
			return nil, err
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			return nil, fmt.Errorf("fixture directory に未知の entry があります")
		}
		info, err := lstatRegularCorpusFile(
			root,
			entry.Name(),
			corpusFixtureMaximumBytes,
		)
		if err != nil {
			return nil, err
		}
		if info.Size() > corpusFixtureMaximumTotalBytes-*fixtureBytes {
			return nil, fmt.Errorf("fixture の原 byte 合計が上限を超えています")
		}
		*fixtureBytes += info.Size()
		files = append(files, corpusFixtureFile{
			name: entry.Name(),
			info: info,
		})
	}
	return files, nil
}

func readCorpusDirectoryEntries(
	ctx context.Context,
	root *os.Root,
	maximum int,
) (entries []os.DirEntry, err error) {
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return nil, err
	}
	if root == nil || maximum < 0 {
		return nil, fmt.Errorf("corpus filesystem の directory を列挙できません")
	}
	directory, err := root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("corpus filesystem の directory を列挙できません")
	}
	defer func() {
		if closeErr := directory.Close(); closeErr != nil && err == nil {
			entries = nil
			err = fmt.Errorf("corpus filesystem の directory を閉じられません")
		}
	}()
	for {
		if err := checkCorpusFilesystemContext(ctx); err != nil {
			return nil, err
		}
		batch, readErr := directory.ReadDir(corpusDirectoryReadBatchSize)
		if len(entries)+len(batch) > maximum {
			return nil, fmt.Errorf("fixture または root entry の件数が上限を超えています")
		}
		entries = append(entries, batch...)
		if errors.Is(readErr, io.EOF) {
			return entries, nil
		}
		if readErr != nil {
			return nil, fmt.Errorf("corpus filesystem の directory を列挙できません")
		}
	}
}

func lstatRegularCorpusFile(
	root *os.Root,
	name string,
	maximumBytes int64,
) (os.FileInfo, error) {
	if root == nil || maximumBytes < 0 {
		return nil, fmt.Errorf("corpus 成果物 file を確認できません")
	}
	info, err := root.Lstat(name)
	if err != nil ||
		info.Mode()&os.ModeSymlink != 0 ||
		!info.Mode().IsRegular() ||
		info.Size() < 0 {
		return nil, fmt.Errorf("corpus 成果物は通常 file でなければなりません")
	}
	if info.Size() > maximumBytes {
		return nil, fmt.Errorf("corpus 成果物 file の size が上限を超えています")
	}
	return info, nil
}
