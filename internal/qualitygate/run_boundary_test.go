package qualitygate

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOSCommandExecutorHonorsCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	executor := osCommandExecutor{
		context:     ctx,
		environment: []string{"PATH=" + os.Getenv("PATH")},
		caches: cachePaths{
			goBuild:  filepath.Join(t.TempDir(), "go-build"),
			golangci: filepath.Join(t.TempDir(), "golangci"),
		},
	}
	_, _, err := executor.run(commandSpec{
		path:      "go",
		args:      []string{"version"},
		dir:       t.TempDir(),
		goCommand: true,
		goFlags:   readonlyGoFlags,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("キャンセルが外部コマンドへ伝播していません: %v", err)
	}
}

func TestRunRejectsNilContext(t *testing.T) {
	t.Parallel()

	err := Run(nil, Options{}, &bytes.Buffer{}, &bytes.Buffer{}) //nolint:staticcheck // SOT-ENG-020: 公開境界が nil context を拒否することを検証する。
	if err == nil {
		t.Fatal("nil context が受理されました")
	}
}

func TestCIRejectsShallowRepository(t *testing.T) {
	t.Parallel()

	if err := requireNonShallowRepository([]byte("false\n")); err != nil {
		t.Fatalf("完全な履歴が拒否されました: %v", err)
	}
	for _, output := range [][]byte{[]byte("true\n"), nil, []byte("unknown\n")} {
		if err := requireNonShallowRepository(output); err == nil {
			t.Fatalf("不完全な履歴が受理されました: %q", output)
		}
	}
}

func TestOSCommandExecutorPreservesAlternateObjectsOnlyWhenRequested(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	consumer := t.TempDir()
	base := osCommandExecutor{
		context:     t.Context(),
		environment: os.Environ(),
	}
	runGitForQualityGateTest(t, base, source, "init", "--quiet")
	runGitForQualityGateTest(t, base, source, "config", "user.name", "品質ゲートテスト")
	runGitForQualityGateTest(t, base, source, "config", "user.email", "quality@example.invalid")
	if err := os.WriteFile(filepath.Join(source, "tracked.txt"), []byte("追跡対象\n"), 0o600); err != nil {
		t.Fatalf("追跡対象を作成できません: %v", err)
	}
	runGitForQualityGateTest(t, base, source, "add", "tracked.txt")
	runGitForQualityGateTest(t, base, source, "commit", "--quiet", "-m", "test: 代替 object")
	commit := strings.TrimSpace(runGitForQualityGateTest(t, base, source, "rev-parse", "HEAD"))
	runGitForQualityGateTest(t, base, consumer, "init", "--quiet")

	objects := filepath.Join(consumer, ".git", "objects")
	alternate := filepath.Join(source, ".git", "objects")
	executor := osCommandExecutor{
		context: t.Context(),
		environment: append(
			os.Environ(),
			"GIT_OBJECT_DIRECTORY="+objects,
			"GIT_ALTERNATE_OBJECT_DIRECTORIES="+alternate,
		),
	}
	spec := commandSpec{
		path:               "git",
		args:               []string{"cat-file", "-e", commit},
		dir:                consumer,
		preserveGitObjects: true,
		isolateGitConfig:   true,
	}
	if _, stderr, err := executor.run(spec); err != nil {
		t.Fatalf("代替 object を参照できません: %v: %s", err, stderr)
	}
	spec.preserveGitObjects = false
	if _, _, err := executor.run(spec); err == nil {
		t.Fatal("保存を指定しない CI 相当のコマンドが ambient alternate object を参照しました")
	}
}

func TestCICachePolicyIgnoresUntrackedLocalCacheAndRejectsTrackedCache(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	executor := osCommandExecutor{
		context:     t.Context(),
		environment: os.Environ(),
	}
	runGitForQualityGateTest(t, executor, repository, "init", "--quiet")
	runGitForQualityGateTest(t, executor, repository, "config", "user.name", "品質ゲートテスト")
	runGitForQualityGateTest(t, executor, repository, "config", "user.email", "quality@example.invalid")
	if err := os.WriteFile(filepath.Join(repository, "tracked.txt"), []byte("追跡対象\n"), 0o600); err != nil {
		t.Fatalf("追跡対象を作成できません: %v", err)
	}
	runGitForQualityGateTest(t, executor, repository, "add", "tracked.txt")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: 基準")

	cacheFile := filepath.Join(repository, ".cache", "local.bin")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o750); err != nil {
		t.Fatalf("ローカルキャッシュを作成できません: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("未追跡キャッシュ\n"), 0o600); err != nil {
		t.Fatalf("ローカルキャッシュを書き込めません: %v", err)
	}
	coverageFile := filepath.Join(repository, "coverage.out")
	if err := os.WriteFile(coverageFile, []byte("未追跡カバレッジ\n"), 0o600); err != nil {
		t.Fatalf("ローカル coverage.out を書き込めません: %v", err)
	}
	if err := executeSteps(
		ciCachePolicySteps(repository),
		executor,
		&bytes.Buffer{},
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("未追跡キャッシュが拒否されました: %v", err)
	}

	runGitForQualityGateTest(t, executor, repository, "add", "--force", "coverage.out")
	if err := executeSteps(
		ciCachePolicySteps(repository),
		executor,
		&bytes.Buffer{},
		&bytes.Buffer{},
	); err == nil {
		t.Fatal("index にある coverage.out が受理されました")
	}
	runGitForQualityGateTest(t, executor, repository, "reset", "--quiet", "--", "coverage.out")

	runGitForQualityGateTest(t, executor, repository, "add", "--force", ".cache/local.bin")
	if err := executeSteps(
		ciCachePolicySteps(repository),
		executor,
		&bytes.Buffer{},
		&bytes.Buffer{},
	); err == nil {
		t.Fatal("index にある cache content が受理されました")
	}
}

func TestCICachePolicyIncludesDetachedUnreferencedHEADHistory(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	executor := osCommandExecutor{
		context:     t.Context(),
		environment: os.Environ(),
	}
	runGitForQualityGateTest(t, executor, repository, "init", "--quiet")
	runGitForQualityGateTest(t, executor, repository, "config", "user.name", "品質ゲートテスト")
	runGitForQualityGateTest(t, executor, repository, "config", "user.email", "quality@example.invalid")
	if err := os.WriteFile(filepath.Join(repository, "baseline.txt"), []byte("基準\n"), 0o600); err != nil {
		t.Fatalf("基準ファイルを作成できません: %v", err)
	}
	runGitForQualityGateTest(t, executor, repository, "add", "baseline.txt")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: 基準")
	runGitForQualityGateTest(t, executor, repository, "checkout", "--quiet", "--detach")

	cacheFile := filepath.Join(repository, ".cache", "detached.txt")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o750); err != nil {
		t.Fatalf("detached cache を作成できません: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("一時的な追跡内容\n"), 0o600); err != nil {
		t.Fatalf("detached cache を書き込めません: %v", err)
	}
	runGitForQualityGateTest(t, executor, repository, "add", "--force", ".cache/detached.txt")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: cache を追加")
	runGitForQualityGateTest(t, executor, repository, "rm", "--quiet", ".cache/detached.txt")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: cache を削除")

	if err := executeSteps(
		ciCachePolicySteps(repository),
		executor,
		&bytes.Buffer{},
		&bytes.Buffer{},
	); err == nil {
		t.Fatal("detached HEAD の親にある cache path が見逃されました")
	}
}

func TestCICachePolicyIgnoresGitReplaceObjects(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	executor := osCommandExecutor{
		context:     t.Context(),
		environment: os.Environ(),
	}
	runGitForQualityGateTest(t, executor, repository, "init", "--quiet")
	runGitForQualityGateTest(t, executor, repository, "config", "user.name", "品質ゲートテスト")
	runGitForQualityGateTest(t, executor, repository, "config", "user.email", "quality@example.invalid")
	if err := os.WriteFile(filepath.Join(repository, "baseline.txt"), []byte("基準\n"), 0o600); err != nil {
		t.Fatalf("基準ファイルを作成できません: %v", err)
	}
	runGitForQualityGateTest(t, executor, repository, "add", "baseline.txt")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: 基準")
	baseline := strings.TrimSpace(runGitForQualityGateTest(t, executor, repository, "rev-parse", "HEAD"))

	cacheFile := filepath.Join(repository, ".cache", "hidden.txt")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0o750); err != nil {
		t.Fatalf("cache ディレクトリを作成できません: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("追跡禁止\n"), 0o600); err != nil {
		t.Fatalf("cache file を作成できません: %v", err)
	}
	runGitForQualityGateTest(t, executor, repository, "add", "--force", ".cache/hidden.txt")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: cache を追加")
	cacheCommit := strings.TrimSpace(runGitForQualityGateTest(t, executor, repository, "rev-parse", "HEAD"))
	runGitForQualityGateTest(t, executor, repository, "rm", "--quiet", ".cache/hidden.txt")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: cache を削除")
	runGitForQualityGateTest(t, executor, repository, "replace", cacheCommit, baseline)

	if err := executeSteps(
		ciCachePolicySteps(repository),
		executor,
		&bytes.Buffer{},
		&bytes.Buffer{},
	); err == nil {
		t.Fatal("replace object で隠された cache path が見逃されました")
	}
}

func TestGitleaksLogOptionsIncludeTransientTypeChanges(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	executor := osCommandExecutor{
		context:     t.Context(),
		environment: os.Environ(),
	}
	runGitForQualityGateTest(t, executor, repository, "init", "--quiet")
	runGitForQualityGateTest(t, executor, repository, "config", "user.name", "品質ゲートテスト")
	runGitForQualityGateTest(t, executor, repository, "config", "user.email", "quality@example.invalid")
	payload := filepath.Join(repository, "payload.txt")
	if err := os.Symlink("safe-target", payload); err != nil {
		t.Skipf("シンボリックリンクを作成できない環境です: %v", err)
	}
	runGitForQualityGateTest(t, executor, repository, "add", "payload.txt")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: 基準")
	baseline := strings.TrimSpace(runGitForQualityGateTest(t, executor, repository, "rev-parse", "HEAD"))

	if err := os.Remove(payload); err != nil {
		t.Fatalf("基準のシンボリックリンクを削除できません: %v", err)
	}
	const marker = "QUALITY_GATE_TRANSIENT_MARKER"
	if err := os.WriteFile(payload, []byte(marker+"\n"), 0o600); err != nil {
		t.Fatalf("type-change の内容を作成できません: %v", err)
	}
	runGitForQualityGateTest(t, executor, repository, "add", "--all")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: type-change")
	runGitForQualityGateTest(t, executor, repository, "rm", "--quiet", "payload.txt")
	runGitForQualityGateTest(t, executor, repository, "commit", "--quiet", "-m", "test: 削除")

	args := append([]string{"log", "-p", "-U0"}, strings.Fields(gitleaksLogOptions)...)
	args = append(args, baseline+"..HEAD")
	if output := runGitForQualityGateTest(t, executor, repository, args...); !strings.Contains(output, marker) {
		t.Fatal("一時的な type-change の秘密情報が Git 差分から除外されました")
	}
}

func TestRunWithExecutorKeepsSnapshotAndGitRepositorySeparate(t *testing.T) {
	t.Parallel()

	snapshot := writeValidPrinciples(t)
	writeWorkflowFile(t, snapshot, "quality.yml")
	gitRepository := t.TempDir()
	goFile := filepath.Join(snapshot, "internal", "example", "example.go")
	if err := os.MkdirAll(filepath.Dir(goFile), 0o750); err != nil {
		t.Fatalf("Go ディレクトリを作成できません: %v", err)
	}
	if err := os.WriteFile(goFile, []byte("package example\n"), 0o600); err != nil {
		t.Fatalf("Go ファイルを作成できません: %v", err)
	}
	executor := &recordingExecutor{
		results: []executionResult{
			{
				stdout: []byte(
					"internal/example/example.go\x00" +
						"wiki/review.md\x00" +
						".github/workflows/quality.yml\x00" +
						".golangci.yml\x00",
				),
			},
		},
	}

	err := runWithExecutor(
		Options{
			Profile:       ProfilePreCommit,
			Repository:    snapshot,
			GitRepository: gitRepository,
		},
		&bytes.Buffer{},
		&bytes.Buffer{},
		executor,
	)
	if err != nil {
		t.Fatalf("pre-commit の実行に失敗しました: %v", err)
	}
	snapshot = resolvedPath(t, snapshot)
	gitRepository = resolvedPath(t, gitRepository)

	if len(executor.specs) != 7 {
		t.Fatalf("外部コマンド数 = %d, want 7", len(executor.specs))
	}
	assertCommandBoundary(t, executor.specs[0], gitRepository, "git")
	assertCommandBoundary(t, executor.specs[1], snapshot, "gofmt")
	assertCommandBoundary(t, executor.specs[2], gitRepository, "git")
	assertCommandBoundary(t, executor.specs[3], snapshot, "go")
	for _, spec := range executor.specs[4:] {
		assertCommandBoundary(t, spec, snapshot, "go")
	}

	gitleaks := executor.specs[3]
	joinedArgs := strings.Join(gitleaks.args, "\n")
	for _, expected := range []string{
		"tools/gitleaks/go.mod",
		".gitleaks.toml",
		"dir",
		".",
	} {
		if !strings.Contains(joinedArgs, expected) {
			t.Fatalf("staged gitleaks が snapshot の設定を参照していません: %q", gitleaks.args)
		}
	}
}

func runGitForQualityGateTest(
	t *testing.T,
	executor osCommandExecutor,
	directory string,
	args ...string,
) string {
	t.Helper()
	stdout, stderr, err := executor.run(commandSpec{
		path:             "git",
		args:             args,
		dir:              directory,
		isolateGitConfig: true,
	})
	if err != nil {
		t.Fatalf("git %s が失敗しました: %v: %s", strings.Join(args, " "), err, stderr)
	}
	return string(stdout)
}

func TestRunWithExecutorRejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	valid := writeValidPrinciples(t)
	tests := []struct {
		name    string
		options Options
	}{
		{
			name: "不正なプロファイル",
			options: Options{
				Profile:       Profile("unknown"),
				Repository:    valid,
				GitRepository: valid,
			},
		},
		{
			name: "検査対象がない",
			options: Options{
				Profile:       ProfileCI,
				Repository:    filepath.Join(valid, "missing"),
				GitRepository: valid,
			},
		},
		{
			name: "Git リポジトリがファイル",
			options: Options{
				Profile:       ProfileCI,
				Repository:    valid,
				GitRepository: filepath.Join(valid, "docs", "development-principles.md"),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := runWithExecutor(
				tt.options,
				&bytes.Buffer{},
				&bytes.Buffer{},
				&recordingExecutor{},
			)
			if err == nil {
				t.Fatal("不正な入力が受理されました")
			}
		})
	}
}
