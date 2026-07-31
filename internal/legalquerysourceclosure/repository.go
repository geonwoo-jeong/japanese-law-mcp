package legalquerysourceclosure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const maximumRepositoryPathBytes = 4096

var rawSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type repositoryReader struct {
	root *os.Root
}

func openRepositoryReader(path string) (*repositoryReader, error) {
	if path == "" {
		return nil, fmt.Errorf("repository root が指定されていません")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("repository root を解決できません")
	}
	expected, err := os.Lstat(absolute)
	if err != nil || expected.Mode()&os.ModeSymlink != 0 || !expected.IsDir() {
		return nil, fmt.Errorf("repository root が通常 directory ではありません")
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, fmt.Errorf("repository root を開けません")
	}
	opened, err := root.Stat(".")
	if err != nil || !opened.IsDir() || !os.SameFile(expected, opened) {
		_ = root.Close()
		return nil, fmt.Errorf("repository root の同一性を検証できません")
	}
	return &repositoryReader{root: root}, nil
}

func (r *repositoryReader) Close() error {
	if r == nil || r.root == nil {
		return nil
	}
	err := r.root.Close()
	r.root = nil
	if err != nil {
		return fmt.Errorf("repository root を閉じられません")
	}
	return nil
}

func (r *repositoryReader) validateDirectory(relative string) error {
	segments, err := validateRepositoryRelativePath(relative)
	if err != nil {
		return err
	}
	opened, closeOpened, err := openDirectorySegments(r.root, segments)
	if err != nil {
		return err
	}
	defer closeOpened()
	info, err := opened.Stat(".")
	if err != nil || !info.IsDir() {
		return fmt.Errorf("repository-relative directory を検証できません")
	}
	return nil
}

func (r *repositoryReader) readRegularContext(ctx context.Context, relative string, maximumBytes int64) ([]byte, error) {
	segments, err := validateRepositoryRelativePath(relative)
	if err != nil {
		return nil, err
	}
	if maximumBytes < 0 {
		return nil, fmt.Errorf("file size 上限が不正です")
	}
	parent := r.root
	closeParent := func() {}
	if len(segments) > 1 {
		parent, closeParent, err = openDirectorySegments(r.root, segments[:len(segments)-1])
		if err != nil {
			return nil, err
		}
	}
	defer closeParent()
	return readRegularSegment(ctx, parent, segments[len(segments)-1], maximumBytes)
}

func openDirectorySegments(base *os.Root, segments []string) (*os.Root, func(), error) {
	current := base
	opened := make([]*os.Root, 0, len(segments))
	closeOpened := func() {
		for index := len(opened) - 1; index >= 0; index-- {
			_ = opened[index].Close()
		}
	}
	for _, segment := range segments {
		expected, err := current.Lstat(segment)
		if err != nil || expected.Mode()&os.ModeSymlink != 0 || !expected.IsDir() {
			closeOpened()
			return nil, func() {}, fmt.Errorf("repository-relative directory が通常 directory ではありません")
		}
		next, err := current.OpenRoot(segment)
		if err != nil {
			closeOpened()
			return nil, func() {}, fmt.Errorf("repository-relative directory を開けません")
		}
		openedInfo, err := next.Stat(".")
		if err != nil || !openedInfo.IsDir() || !os.SameFile(expected, openedInfo) {
			_ = next.Close()
			closeOpened()
			return nil, func() {}, fmt.Errorf("repository-relative directory の同一性を検証できません")
		}
		currentInfo, err := current.Lstat(segment)
		if err != nil || currentInfo.Mode()&os.ModeSymlink != 0 || !currentInfo.IsDir() || !os.SameFile(openedInfo, currentInfo) {
			_ = next.Close()
			closeOpened()
			return nil, func() {}, fmt.Errorf("repository-relative directory が検証中に変化しました")
		}
		opened = append(opened, next)
		current = next
	}
	return current, closeOpened, nil
}

func readRegularSegment(ctx context.Context, parent *os.Root, name string, maximumBytes int64) ([]byte, error) {
	expected, err := parent.Lstat(name)
	if err != nil || expected.Mode()&os.ModeSymlink != 0 || !expected.Mode().IsRegular() || expected.Size() < 0 || expected.Size() > maximumBytes {
		return nil, fmt.Errorf("repository-relative file が上限内の通常 file ではありません")
	}
	file, err := parent.Open(name)
	if err != nil {
		return nil, fmt.Errorf("repository-relative file を開けません")
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || opened.Size() != expected.Size() || !os.SameFile(expected, opened) {
		return nil, fmt.Errorf("repository-relative file の同一性を検証できません")
	}
	raw, err := io.ReadAll(io.LimitReader(contextReader{ctx: ctx, reader: file}, maximumBytes+1))
	if err != nil || int64(len(raw)) != opened.Size() || int64(len(raw)) > maximumBytes {
		return nil, fmt.Errorf("repository-relative file を上限内で完全に読めません")
	}
	afterRead, err := file.Stat()
	if err != nil || !afterRead.Mode().IsRegular() || afterRead.Size() != opened.Size() || !os.SameFile(opened, afterRead) {
		return nil, fmt.Errorf("repository-relative file が読取り中に変化しました")
	}
	current, err := parent.Lstat(name)
	if err != nil || current.Mode()&os.ModeSymlink != 0 || !current.Mode().IsRegular() || current.Size() != opened.Size() || !os.SameFile(opened, current) {
		return nil, fmt.Errorf("repository-relative file が読取り中に置換されました")
	}
	return raw, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(target []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(target)
}

func validateRepositoryRelativePath(value string) ([]string, error) {
	if value == "" || len(value) > maximumRepositoryPathBytes || !utf8.ValidString(value) ||
		!fs.ValidPath(value) || value == "." || strings.Contains(value, `\`) {
		return nil, fmt.Errorf("path は正規化済み repository-relative POSIX path でなければなりません")
	}
	segments := strings.Split(value, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return nil, fmt.Errorf("path に不正な segment があります")
		}
	}
	return segments, nil
}

func rawSHA256(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// VerifyFiles は、整列済み manifest file 集合を root-scoped の同じ byte で再検証する。
func VerifyFiles(ctx context.Context, repositoryPath string, files []SourceFile) error {
	if len(files) > MaximumSourceFiles {
		return fmt.Errorf("semantic source file 数が上限を超えています")
	}
	if err := validateSourceFileOrder(files); err != nil {
		return err
	}
	repository, err := openRepositoryReader(repositoryPath)
	if err != nil {
		return err
	}
	defer func() { _ = repository.Close() }()
	var totalBytes int64
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("semantic source file 検証が中止されました: %w", err)
		}
		raw, readErr := repository.readRegularContext(ctx, file.Path, MaximumSourceFileBytes)
		if readErr != nil {
			return fmt.Errorf("semantic source file %q を検証できません: %w", file.Path, readErr)
		}
		if int64(len(raw)) > MaximumSourceTotalBytes-totalBytes {
			return fmt.Errorf("semantic source file の合計 size が上限を超えています")
		}
		totalBytes += int64(len(raw))
		if rawSHA256(raw) != file.RawSHA256 {
			return fmt.Errorf("semantic source file %q の raw SHA-256 が一致しません", file.Path)
		}
	}
	return nil
}

func validateSourceFileOrder(files []SourceFile) error {
	if !slices.IsSortedFunc(files, func(left, right SourceFile) int {
		return strings.Compare(left.Path, right.Path)
	}) {
		return fmt.Errorf("semantic source file は path 順でなければなりません")
	}
	for index, file := range files {
		if _, err := validateRepositoryRelativePath(file.Path); err != nil {
			return err
		}
		if !rawSHA256Pattern.MatchString(file.RawSHA256) {
			return fmt.Errorf("semantic source file の raw SHA-256 が不正です")
		}
		if index > 0 && files[index-1].Path == file.Path {
			return fmt.Errorf("semantic source file path が重複しています")
		}
	}
	return nil
}
