package legalqueryartifact

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Repository は、検証済み directory descriptor に限定した読取り境界である。
type Repository struct {
	root *os.Root
}

// DirectoryEntry は、列挙時に検証した一 entry の名前と metadata を保持する。
type DirectoryEntry struct {
	name string
	info fs.FileInfo
}

// OpenRepository は、symlink ではない repository root を descriptor として開く。
func OpenRepository(path string) (*Repository, error) {
	if path == "" {
		return nil, fmt.Errorf("repository root が指定されていません")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("repository root を確認できません")
	}
	expected, err := os.Lstat(absolute)
	if err != nil || expected.Mode()&os.ModeSymlink != 0 || !expected.IsDir() {
		return nil, fmt.Errorf("repository root が有効ではありません")
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, fmt.Errorf("repository root を開けません")
	}
	opened, err := root.Stat(".")
	if err != nil || !opened.IsDir() || !os.SameFile(expected, opened) {
		_ = root.Close()
		return nil, fmt.Errorf("repository root の検証に失敗しました")
	}
	return &Repository{root: root}, nil
}

// Close は保持する directory descriptor を閉じる。
func (r *Repository) Close() error {
	if r == nil || r.root == nil {
		return nil
	}
	if err := r.root.Close(); err != nil {
		return fmt.Errorf("repository root を閉じられません")
	}
	r.root = nil
	return nil
}

// OpenChild は、一 segment の通常 directory を検証して開く。
func (r *Repository) OpenChild(name string) (*Repository, error) {
	if r == nil || r.root == nil {
		return nil, fmt.Errorf("repository handle が初期化されていません")
	}
	if err := validateSingleSegment(name); err != nil {
		return nil, err
	}
	expected, err := r.root.Lstat(name)
	if err != nil || expected.Mode()&os.ModeSymlink != 0 || !expected.IsDir() {
		return nil, fmt.Errorf("子 directory が有効ではありません")
	}
	child, err := r.root.OpenRoot(name)
	if err != nil {
		return nil, fmt.Errorf("子 directory を開けません")
	}
	opened, err := child.Stat(".")
	if err != nil || !opened.IsDir() || !os.SameFile(expected, opened) {
		_ = child.Close()
		return nil, fmt.Errorf("子 directory の検証に失敗しました")
	}
	current, err := r.root.Lstat(name)
	if err != nil || current.Mode()&os.ModeSymlink != 0 || !current.IsDir() || !os.SameFile(opened, current) {
		_ = child.Close()
		return nil, fmt.Errorf("子 directory が読取り中に変化しました")
	}
	return &Repository{root: child}, nil
}

// ReadRegular は、一 segment の通常 file を上限内で完全に読む。
func (r *Repository) ReadRegular(name string, maximumBytes int64) ([]byte, error) {
	if r == nil || r.root == nil {
		return nil, fmt.Errorf("repository handle が初期化されていません")
	}
	if err := validateSingleSegment(name); err != nil {
		return nil, err
	}
	if maximumBytes < 0 {
		return nil, fmt.Errorf("file 上限が不正です")
	}
	expected, err := r.root.Lstat(name)
	if err != nil || expected.Mode()&os.ModeSymlink != 0 || !expected.Mode().IsRegular() || expected.Size() < 0 || expected.Size() > maximumBytes {
		return nil, fmt.Errorf("regular file が有効ではありません")
	}
	file, err := r.root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("regular file を開けません")
	}
	defer func() {
		_ = file.Close()
	}()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || opened.Size() != expected.Size() || !os.SameFile(expected, opened) {
		return nil, fmt.Errorf("regular file の検証に失敗しました")
	}
	data, err := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	if err != nil {
		return nil, fmt.Errorf("regular file を読めません")
	}
	if int64(len(data)) > maximumBytes || int64(len(data)) != opened.Size() {
		return nil, fmt.Errorf("regular file が読取り中に変化しました")
	}
	afterRead, err := file.Stat()
	if err != nil || !afterRead.Mode().IsRegular() || afterRead.Size() != opened.Size() || !os.SameFile(opened, afterRead) {
		return nil, fmt.Errorf("regular file が読取り中に変化しました")
	}
	current, err := r.root.Lstat(name)
	if err != nil || current.Mode()&os.ModeSymlink != 0 || !current.Mode().IsRegular() || current.Size() != opened.Size() || !os.SameFile(opened, current) {
		return nil, fmt.Errorf("regular file が読取り中に変化しました")
	}
	return data, nil
}

// ReadDirectory は、entry 数と通常 file の宣言 size 合計を上限付きで列挙する。
func (r *Repository) ReadDirectory(maximumEntries int, maximumTotalBytes int64) ([]DirectoryEntry, error) {
	if r == nil || r.root == nil {
		return nil, fmt.Errorf("repository handle が初期化されていません")
	}
	if maximumEntries < 0 || maximumTotalBytes < 0 {
		return nil, fmt.Errorf("directory 上限が不正です")
	}
	directory, err := r.root.Open(".")
	if err != nil {
		return nil, fmt.Errorf("directory を開けません")
	}
	defer func() {
		_ = directory.Close()
	}()
	info, err := directory.Stat()
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("directory の検証に失敗しました")
	}
	entries := make([]DirectoryEntry, 0)
	var totalBytes int64
	for {
		batch, readErr := directory.ReadDir(32)
		if readErr != nil && readErr != io.EOF {
			return nil, fmt.Errorf("directory を列挙できません")
		}
		for _, entry := range batch {
			if len(entries) == maximumEntries {
				return nil, fmt.Errorf("directory entry 数が上限を超えています")
			}
			name := entry.Name()
			if !fs.ValidPath(name) || strings.Contains(name, "/") {
				return nil, fmt.Errorf("directory entry 名が不正です")
			}
			entryInfo, infoErr := entry.Info()
			if infoErr != nil {
				return nil, fmt.Errorf("directory entry を確認できません")
			}
			if entryInfo.Mode().IsRegular() {
				if entryInfo.Size() < 0 {
					return nil, fmt.Errorf("directory entry の size が不正です")
				}
				if entryInfo.Size() > maximumTotalBytes-totalBytes {
					return nil, fmt.Errorf("directory entry の合計 size が上限を超えています")
				}
				totalBytes += entryInfo.Size()
			}
			entries = append(entries, DirectoryEntry{name: name, info: entryInfo})
		}
		if readErr == io.EOF {
			break
		}
	}
	slices.SortFunc(entries, func(left, right DirectoryEntry) int {
		return strings.Compare(left.name, right.name)
	})
	return entries, nil
}

// Name は entry の一 segment 名を返す。
func (e DirectoryEntry) Name() string {
	return e.name
}

// Info は列挙時の file metadata を返す。
func (e DirectoryEntry) Info() fs.FileInfo {
	return e.info
}

func validateSingleSegment(name string) error {
	if !fs.ValidPath(name) || name == "." || strings.Contains(name, "/") {
		return fmt.Errorf("単一 segment 名だけを受け付けます")
	}
	return nil
}
