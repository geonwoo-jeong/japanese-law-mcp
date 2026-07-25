package githook

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const qualityCacheDirectory = "japanese-law-mcp-quality-cache"

type hookCachePaths struct {
	goBuild  string
	golangci string
}

func prepareHookCachePaths(
	ctx context.Context,
	repository string,
) (hookCachePaths, error) {
	resolvedCommon, err := hookGitCommonDirectory(ctx, repository)
	if err != nil {
		return hookCachePaths{}, err
	}
	return createHookCachePaths(resolvedCommon)
}

func hookGitCommonDirectory(ctx context.Context, repository string) (string, error) {
	command := gitCommand(
		ctx,
		repository,
		nil,
		"rev-parse",
		"--path-format=absolute",
		"--git-common-dir",
	)
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("対象の Git common directory を取得できませんでした: %w", err)
	}
	commonDirectory := stringWithoutLineEnding(output)
	if commonDirectory == "" ||
		!filepath.IsAbs(commonDirectory) ||
		strings.ContainsAny(commonDirectory, "\r\n") {
		return "", errors.New("取得した Git common directory の応答が不正です")
	}
	resolvedCommon, err := resolveRealDirectory(commonDirectory)
	if err != nil {
		return "", fmt.Errorf("取得した Git common directory が安全ではありません: %w", err)
	}
	return resolvedCommon, nil
}

func createHookCachePaths(commonDirectory string) (hookCachePaths, error) {
	root := filepath.Join(commonDirectory, qualityCacheDirectory)
	if err := createRealDirectory(root); err != nil {
		return hookCachePaths{}, err
	}
	caches := cachePaths(root)
	for _, cache := range []string{caches.goBuild, caches.golangci} {
		if err := createRealDirectory(cache); err != nil {
			return hookCachePaths{}, err
		}
	}
	return caches, nil
}

func cachePaths(root string) hookCachePaths {
	return hookCachePaths{
		goBuild:  filepath.Join(root, "go-build"),
		golangci: filepath.Join(root, "golangci"),
	}
}

func createRealDirectory(directory string) error {
	err := os.Mkdir(directory, 0o750)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("品質ゲート cache directory を作成できませんでした: %w", err)
	}
	return validatePrivateCacheDirectory(directory)
}

func validatePrivateCacheDirectory(directory string) error {
	info, err := os.Lstat(directory)
	if err != nil {
		return fmt.Errorf("品質ゲート cache directory を確認できませんでした: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("品質ゲート cache path は実体のある directory ではありません: %s", directory)
	}
	if hasPOSIXPermissionBits() && info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("品質ゲート cache path に安全でない書込権限があります: %s", directory)
	}
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return fmt.Errorf("品質ゲート cache directory を解決できませんでした: %w", err)
	}
	if resolved != directory {
		return fmt.Errorf("品質ゲート cache directory が別の path を指しています: %s", directory)
	}
	return nil
}

func checkHookCachePaths(ctx context.Context, repository string) error {
	_, err := existingHookCachePaths(ctx, repository)
	return err
}

func existingHookCachePaths(
	ctx context.Context,
	repository string,
) (hookCachePaths, error) {
	commonDirectory, err := hookGitCommonDirectory(ctx, repository)
	if err != nil {
		return hookCachePaths{}, err
	}
	root := filepath.Join(commonDirectory, qualityCacheDirectory)
	caches := cachePaths(root)
	for _, directory := range []string{
		root,
		caches.goBuild,
		caches.golangci,
	} {
		if err := validatePrivateCacheDirectory(directory); err != nil {
			return hookCachePaths{}, err
		}
	}
	return caches, nil
}

func resolveRealDirectory(directory string) (string, error) {
	resolved, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("実体のある directory ではありません: %s", resolved)
	}
	return resolved, nil
}

func environmentWithHookCaches(
	environment []string,
	caches hookCachePaths,
) []string {
	result := environmentWithValue(environment, "GOCACHE", caches.goBuild)
	return environmentWithValue(result, "GOLANGCI_LINT_CACHE", caches.golangci)
}
