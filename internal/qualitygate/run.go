package qualitygate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Run は、指定したプロファイルの品質ゲートを fail-fast で実行する。
func Run(
	ctx context.Context,
	options Options,
	stdout io.Writer,
	stderr io.Writer,
) (result error) {
	if ctx == nil {
		return errors.New("品質ゲートのコンテキストが指定されていません")
	}
	snapshot, err := existingDirectory(options.Repository, "検査対象リポジトリ")
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
	bootstrapExecutor := newOSCommandExecutor(ctx, cachePaths{})
	commonDirectory, err := gitCommonDirectory(bootstrapExecutor, gitRepository)
	if err != nil {
		return err
	}
	caches, cleanup, err := prepareRunCachePaths(commonDirectory, snapshot, gitRepository)
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := cleanup(); result == nil && cleanupErr != nil {
			result = fmt.Errorf(
				"snapshot 固有のリンターキャッシュを削除できません: %w",
				cleanupErr,
			)
		}
	}()
	options.GitRepository = gitRepository
	return runWithExecutor(options, stdout, stderr, newOSCommandExecutor(ctx, caches))
}

func runWithExecutor(
	options Options,
	stdout io.Writer,
	stderr io.Writer,
	executor commandExecutor,
) error {
	profile, err := ParseProfile(string(options.Profile))
	if err != nil {
		return err
	}
	snapshot, err := existingDirectory(options.Repository, "検査対象リポジトリ")
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

	var changedPaths []string
	var goFormatPaths []string
	if profile == ProfilePreCommit {
		changedPaths, err = stagedPaths(executor, gitRepository)
		if err != nil {
			return err
		}
		goFormatPaths, err = existingRegularGoFiles(snapshot, changedPaths)
		if err != nil {
			return err
		}
	}
	steps, err := buildPlan(planInput{
		profile:       profile,
		repository:    gitRepository,
		snapshot:      snapshot,
		changedPaths:  changedPaths,
		goFormatPaths: goFormatPaths,
		gitRanges:     options.GitRanges,
	})
	if err != nil {
		return err
	}
	return executeSteps(steps, executor, stdout, stderr)
}

func existingDirectory(path, label string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%sが指定されていません", label)
	}
	absolute, err := filepath.Abs(path)
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

func stagedPaths(executor commandExecutor, gitRepository string) ([]string, error) {
	stdout, stderr, err := executor.run(commandSpec{
		path: "git",
		args: []string{
			"diff",
			"--cached",
			"--name-only",
			"--no-renames",
			"-z",
		},
		dir:                gitRepository,
		preserveGitIndex:   true,
		preserveGitObjects: true,
		isolateGitConfig:   true,
	})
	if err != nil {
		message := strings.TrimSpace(string(stderr))
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("ステージ済みパスを取得できません: %s", message)
	}
	if len(stdout) == 0 {
		return nil, nil
	}
	if stdout[len(stdout)-1] != 0 {
		return nil, errors.New("取得した Git のステージ済みパスが NUL 終端ではありません")
	}
	rawPaths := strings.Split(string(stdout[:len(stdout)-1]), "\x00")
	return normalizeChangedPaths(rawPaths)
}

func existingRegularGoFiles(snapshot string, changedPaths []string) ([]string, error) {
	goFiles := make([]string, 0, len(changedPaths))
	for _, changedPath := range changedPaths {
		if !strings.HasSuffix(changedPath, ".go") {
			continue
		}
		path := filepath.Join(snapshot, filepath.FromSlash(changedPath))
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("対象の Go ファイルを確認できません: %s: %w", changedPath, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("対象の Go ファイルが通常ファイルではありません: %s", changedPath)
		}
		goFiles = append(goFiles, changedPath)
	}
	return goFiles, nil
}

func gitCommonDirectory(executor commandExecutor, gitRepository string) (string, error) {
	stdout, stderr, err := executor.run(commandSpec{
		path:             "git",
		args:             []string{"rev-parse", "--git-common-dir"},
		dir:              gitRepository,
		isolateGitConfig: true,
	})
	if err != nil {
		message := strings.TrimSpace(string(stderr))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("元の Git 共通ディレクトリを取得できません: %s", message)
	}
	value := strings.TrimSpace(string(stdout))
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return "", errors.New("取得した Git 共通ディレクトリの応答が不正です")
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(gitRepository, value)
	}
	commonDirectory, err := existingDirectory(value, "Git 共通ディレクトリ")
	if err != nil {
		return "", err
	}
	return commonDirectory, nil
}

func prepareCachePaths(commonDirectory string) (cachePaths, error) {
	root := filepath.Join(commonDirectory, "japanese-law-mcp-quality-cache")
	caches := cachePaths{
		goBuild:  filepath.Join(root, "go-build"),
		golangci: filepath.Join(root, "golangci"),
	}
	for _, path := range []string{root, caches.goBuild, caches.golangci} {
		if err := ensurePrivateCacheDirectory(path); err != nil {
			return cachePaths{}, err
		}
	}
	return caches, nil
}

func prepareRunCachePaths(
	commonDirectory, snapshot, gitRepository string,
) (cachePaths, func() error, error) {
	caches, err := prepareCachePaths(commonDirectory)
	if err != nil {
		return cachePaths{}, nil, err
	}
	if snapshot == gitRepository {
		return caches, func() error { return nil }, nil
	}

	lintCache, err := os.MkdirTemp("", "japanese-law-mcp-golangci-cache-")
	if err != nil {
		return cachePaths{}, nil, fmt.Errorf(
			"snapshot 固有のリンターキャッシュを作成できません: %w",
			err,
		)
	}
	caches.golangci = lintCache
	return caches, func() error { return os.RemoveAll(lintCache) }, nil
}

func ensurePrivateCacheDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if mkdirErr := os.Mkdir(path, 0o750); mkdirErr != nil {
			return fmt.Errorf("品質ゲートのキャッシュを作成できません: %w", mkdirErr)
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return fmt.Errorf("品質ゲートのキャッシュを確認できません: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("品質ゲートのキャッシュが実体のあるディレクトリではありません: %s", path)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("品質ゲートのキャッシュに安全でない書込権限があります: %s", path)
	}
	return nil
}
