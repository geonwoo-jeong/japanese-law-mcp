package legalquerycorpus

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type resolvedCorpusFilesystemPaths struct {
	repositoryRoot string
	corpusVersion  string
}

func resolveCorpusFilesystemPaths(
	repositoryRoot string,
	corpusDirectory string,
) (resolvedCorpusFilesystemPaths, error) {
	if repositoryRoot == "" || corpusDirectory == "" {
		return resolvedCorpusFilesystemPaths{}, invalidCorpusFilesystemPath()
	}
	inputRepositoryRoot, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return resolvedCorpusFilesystemPaths{}, invalidCorpusFilesystemPath()
	}
	inputRepositoryRoot = filepath.Clean(inputRepositoryRoot)
	if err := validateRepositoryRootPath(inputRepositoryRoot); err != nil {
		return resolvedCorpusFilesystemPaths{}, err
	}
	resolvedRepositoryRoot, err := filepath.EvalSymlinks(inputRepositoryRoot)
	if err != nil {
		return resolvedCorpusFilesystemPaths{}, invalidCorpusFilesystemPath()
	}
	if err := validateRepositoryRootPath(resolvedRepositoryRoot); err != nil {
		return resolvedCorpusFilesystemPaths{}, err
	}
	relativeCorpus, err := resolveCorpusRelativePath(
		inputRepositoryRoot,
		resolvedRepositoryRoot,
		corpusDirectory,
	)
	if err != nil {
		return resolvedCorpusFilesystemPaths{}, err
	}
	return resolvedCorpusFilesystemPaths{
		repositoryRoot: resolvedRepositoryRoot,
		corpusVersion:  filepath.Base(relativeCorpus),
	}, nil
}

func resolveDevelopmentFilesystemPaths(
	repositoryRoot string,
	developmentDirectory string,
) (resolvedCorpusFilesystemPaths, error) {
	if developmentDirectory == "" ||
		filepath.Clean(developmentDirectory) != developmentDirectory ||
		filepath.Base(developmentDirectory) != string(ManifestSetDevelopment) {
		return resolvedCorpusFilesystemPaths{}, invalidCorpusFilesystemPath()
	}
	return resolveCorpusFilesystemPaths(
		repositoryRoot,
		filepath.Dir(developmentDirectory),
	)
}

func resolveCorpusRelativePath(
	inputRepositoryRoot string,
	resolvedRepositoryRoot string,
	corpusDirectory string,
) (string, error) {
	if filepath.Clean(corpusDirectory) != corpusDirectory {
		return "", invalidCorpusFilesystemPath()
	}
	if !filepath.IsAbs(corpusDirectory) {
		if !isCanonicalCorpusRelativePath(corpusDirectory) {
			return "", invalidCorpusFilesystemPath()
		}
		return corpusDirectory, nil
	}
	for _, repositoryRoot := range []string{
		inputRepositoryRoot,
		resolvedRepositoryRoot,
	} {
		relative, err := filepath.Rel(repositoryRoot, corpusDirectory)
		if err == nil && isCanonicalCorpusRelativePath(relative) {
			return relative, nil
		}
	}
	return "", invalidCorpusFilesystemPath()
}

func isCanonicalCorpusRelativePath(path string) bool {
	if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != path {
		return false
	}
	parts := strings.Split(filepath.ToSlash(path), "/")
	return len(parts) == 3 &&
		parts[0] == "testdata" &&
		parts[1] == "legalquery" &&
		manifestCorpusVersionPattern.MatchString(parts[2])
}

func validateRepositoryRootPath(path string) error {
	info, err := os.Lstat(path)
	if err != nil ||
		info.Mode()&os.ModeSymlink != 0 ||
		!info.IsDir() {
		return invalidCorpusFilesystemPath()
	}
	return nil
}

func openRepositoryFilesystemRoot(path string) (*os.Root, error) {
	expected, err := os.Lstat(path)
	if err != nil ||
		expected.Mode()&os.ModeSymlink != 0 ||
		!expected.IsDir() {
		return nil, invalidCorpusFilesystemPath()
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, invalidCorpusFilesystemPath()
	}
	opened, err := root.Stat(".")
	if err != nil || !opened.IsDir() || !os.SameFile(expected, opened) {
		_ = root.Close()
		return nil, invalidCorpusFilesystemPath()
	}
	return root, nil
}

func openVerifiedChildRoot(
	ctx context.Context,
	parent *os.Root,
	name string,
) (*os.Root, error) {
	if err := checkCorpusFilesystemContext(ctx); err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, fmt.Errorf("corpus filesystem の directory を開けません")
	}
	expected, err := parent.Lstat(name)
	if err != nil || !isPlainDirectory(expected) {
		return nil, fmt.Errorf("corpus filesystem の directory が有効ではありません")
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return nil, fmt.Errorf("corpus filesystem の directory を開けません")
	}
	if err := verifyOpenedChildRoot(parent, child, name, expected); err != nil {
		_ = child.Close()
		return nil, err
	}
	return child, nil
}

func verifyOpenedChildRoot(
	parent *os.Root,
	child *os.Root,
	name string,
	expected os.FileInfo,
) error {
	opened, err := child.Stat(".")
	if err != nil ||
		!opened.IsDir() ||
		!os.SameFile(expected, opened) {
		return fmt.Errorf("corpus filesystem の directory が変化しました")
	}
	current, err := parent.Lstat(name)
	if err != nil ||
		!isPlainDirectory(current) ||
		!os.SameFile(opened, current) {
		return fmt.Errorf("corpus filesystem の directory が変化しました")
	}
	return nil
}

func isPlainDirectory(info os.FileInfo) bool {
	return info != nil &&
		info.Mode()&os.ModeSymlink == 0 &&
		info.IsDir()
}

func checkCorpusFilesystemContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("corpus filesystem の context が指定されていません")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("corpus filesystem の処理が取り消されました: %w", err)
	}
	return nil
}

func invalidCorpusFilesystemPath() error {
	return fmt.Errorf("repository または corpus directory の path が有効ではありません")
}
