package provideronboarding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Run は、SOT-ENG-017/018 に基づく provider 追加 fitness gate を実行する。
func Run(ctx context.Context, options Options) error {
	return runWithDependencies(ctx, options, dependencies{
		load: loadCanonicalRows,
		test: runProviderConformanceTests,
	})
}

func runWithDependencies(
	ctx context.Context,
	options Options,
	deps dependencies,
) error {
	if ctx == nil {
		return errors.New("provider 追加 fitness gate のコンテキストが指定されていません")
	}
	if deps.load == nil || deps.test == nil {
		return errors.New("provider 追加 fitness gate の依存処理が指定されていません")
	}
	if options.BaseRef == "" {
		return ErrInvalidBaseRef
	}

	repository, err := existingDirectory(options.Repository, "検査対象リポジトリ")
	if err != nil {
		return err
	}
	gitRepositoryValue := options.GitRepository
	if gitRepositoryValue == "" {
		gitRepositoryValue = options.Repository
	}
	gitRepository, err := existingDirectory(gitRepositoryValue, "Git リポジトリ")
	if err != nil {
		return err
	}
	headRef := options.HeadRef
	if headRef == "" {
		headRef = "HEAD"
	}

	client := newGitClient(gitRepository)
	resolved, err := client.resolveComparison(ctx, options.BaseRef, headRef)
	if err != nil {
		return err
	}
	paths, err := client.collectChangedPaths(ctx, resolved, changeSources{
		index:       options.IncludeIndex,
		workingTree: options.IncludeWorkingTree,
		untracked:   options.IncludeUntracked,
	})
	if err != nil {
		return err
	}
	bootstrap, err := detectBootstrap(ctx, client, repository, resolved.mergeBase)
	if err != nil {
		return err
	}

	rows, err := deps.load(repository)
	if err != nil {
		return fmt.Errorf("canonical conformance matrix を読み込めませんでした: %w", err)
	}
	applicable := bootstrap
	if bootstrap {
		err = validateBootstrapChanges(paths, rows)
	} else {
		applicable, err = evaluateNormalChanges(paths, rows)
	}
	if err != nil {
		return err
	}
	if !applicable {
		return nil
	}
	if err := validateProviderImports(repository, rows); err != nil {
		return err
	}

	stdout := options.Stdout
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := options.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	if err := deps.test(ctx, repository, stdout, stderr); err != nil {
		return fmt.Errorf("provider conformance の通常テストが失敗しました: %w", err)
	}
	return nil
}

func existingDirectory(value, label string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%sが指定されていません", label)
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("%sの絶対パスを解決できません: %w", label, err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("%sを解決できません: %w", label, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("%sを確認できません: %w", label, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%sはディレクトリではありません: %s", label, resolved)
	}
	return resolved, nil
}

func detectBootstrap(
	ctx context.Context,
	client gitClient,
	repository, baseCommit string,
) (bool, error) {
	basePaths, err := client.treePaths(ctx, baseCommit)
	if err != nil {
		return false, err
	}
	canonicalPaths := []string{
		canonicalSchemaPath,
		canonicalLoaderPath,
		canonicalCommandPath,
	}
	baseCount := 0
	currentCount := 0
	for _, canonicalPath := range canonicalPaths {
		if _, ok := basePaths[canonicalPath]; ok {
			baseCount++
		}
		exists, currentErr := regularFileExists(repository, canonicalPath)
		if currentErr != nil {
			return false, currentErr
		}
		if exists {
			currentCount++
		}
	}

	switch {
	case baseCount == 0 && currentCount == len(canonicalPaths):
		return true, nil
	case baseCount == len(canonicalPaths) && currentCount == len(canonicalPaths):
		return false, nil
	case currentCount != len(canonicalPaths):
		return false, errors.New(
			"canonical schema、loader、command が現在の検査対象にすべて存在しません",
		)
	default:
		return false, errors.New(
			"比較元の canonical schema、loader、command の導入状態が一致していません",
		)
	}
}

func regularFileExists(repository, name string) (bool, error) {
	target := filepath.Join(repository, filepath.FromSlash(name))
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("canonical artifact を確認できません: %s: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("canonical artifact が通常ファイルではありません: %s", name)
	}
	return true, nil
}
