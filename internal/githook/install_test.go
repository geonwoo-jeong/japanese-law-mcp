package githook

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallWarmsToolsBeforeSettingLocalHooksPath(t *testing.T) {
	repository := newGitRepository(t)
	makeHookFiles(t, repository)
	app := newTestApplication(repository)
	warmed := false
	app.warmUp = func(ctx context.Context, gotRepository string) error {
		if gotRepository != repository {
			t.Fatalf("リポジトリが一致しません: %s", gotRepository)
		}
		if value := localHooksPath(t, ctx, repository); value != "" {
			t.Fatalf("ウォームアップ前に hooksPath が設定されました: %s", value)
		}
		warmed = true
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"install"})

	if code != 0 {
		t.Fatalf("install が失敗しました: %s", stderr)
	}
	if !warmed {
		t.Fatal("固定済みツールのウォームアップが実行されませんでした")
	}
	if got := localHooksPath(t, t.Context(), repository); got != ".githooks" {
		t.Fatalf("hooksPath = %q, want .githooks", got)
	}
}

func TestWarmUpToolsFailsClosedAndCleansTemporaryBuilds(t *testing.T) {
	temporaryRoot := t.TempDir()
	t.Setenv("TMPDIR", temporaryRoot)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	repository := newGitRepository(t)

	err := warmUpTools(ctx, repository)

	if err == nil {
		t.Fatal("中断済みの tool warm-up が成功しました")
	}
	entries, readErr := os.ReadDir(temporaryRoot)
	if readErr != nil {
		t.Fatalf("一時ディレクトリを確認できませんでした: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("tool warm-up の一時成果物が残っています: %v", entries)
	}
	builds := toolBuilds(temporaryRoot)
	if len(builds) != 4 || builds[0].name != "quality-gate" || builds[3].name != "gitleaks" {
		t.Fatalf("固定済み build 一覧が不正です: %#v", builds)
	}
}

func TestPrepareHookCachePathsUsesAbsoluteCommonDirectoryWithSpaces(t *testing.T) {
	repository := filepath.Join(t.TempDir(), "作業 tree")
	if err := os.Mkdir(repository, 0o750); err != nil {
		t.Fatalf("リポジトリを作成できませんでした: %v", err)
	}
	runGit(t, repository, "init", "--quiet")

	caches, err := prepareHookCachePaths(t.Context(), repository)

	if err != nil {
		t.Fatalf("hook cache を準備できませんでした: %v", err)
	}
	common := strings.TrimSpace(
		runGit(t, repository, "rev-parse", "--path-format=absolute", "--git-common-dir"),
	)
	common, err = filepath.EvalSymlinks(common)
	if err != nil {
		t.Fatalf("Git common directory を解決できませんでした: %v", err)
	}
	expectedRoot := filepath.Join(common, "japanese-law-mcp-quality-cache")
	if caches.goBuild != filepath.Join(expectedRoot, "go-build") ||
		caches.golangci != filepath.Join(expectedRoot, "golangci") {
		t.Fatalf("cache paths = %#v, want root %s", caches, expectedRoot)
	}
	for _, cache := range []string{caches.goBuild, caches.golangci} {
		info, statErr := os.Lstat(cache)
		if statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("安全な cache directory ではありません: %s (%v)", cache, statErr)
		}
	}
}

func TestPrepareHookCachePathsSharesCacheAcrossLinkedWorktrees(t *testing.T) {
	repository := filepath.Join(t.TempDir(), "main repository")
	if err := os.Mkdir(repository, 0o750); err != nil {
		t.Fatalf("main repository を作成できませんでした: %v", err)
	}
	runGit(t, repository, "init", "--quiet")
	runGit(t, repository, "config", "user.name", "テスト利用者")
	runGit(t, repository, "config", "user.email", "test@example.invalid")
	writeFile(t, repository, "README.md", "main\n")
	commitAll(t, repository, "main")
	linked := filepath.Join(t.TempDir(), "linked 作業 tree")
	runGit(t, repository, "worktree", "add", "--quiet", "-b", "linked-cache-test", linked)

	mainCaches, err := prepareHookCachePaths(t.Context(), repository)
	if err != nil {
		t.Fatalf("main cache を準備できませんでした: %v", err)
	}
	linkedCaches, err := prepareHookCachePaths(t.Context(), linked)
	if err != nil {
		t.Fatalf("linked cache を準備できませんでした: %v", err)
	}

	if mainCaches != linkedCaches {
		t.Fatalf("linked worktree が cache を共有していません: %#v != %#v", mainCaches, linkedCaches)
	}
}

func TestPrepareHookCachePathsRejectsSymlinkedCacheRoot(t *testing.T) {
	repository := newGitRepository(t)
	common := strings.TrimSpace(
		runGit(t, repository, "rev-parse", "--path-format=absolute", "--git-common-dir"),
	)
	outside := t.TempDir()
	cacheRoot := filepath.Join(common, "japanese-law-mcp-quality-cache")
	if err := os.Symlink(outside, cacheRoot); err != nil {
		t.Fatalf("cache root symlink を作成できませんでした: %v", err)
	}

	if _, err := prepareHookCachePaths(t.Context(), repository); err == nil {
		t.Fatal("リポジトリ外を指す cache root symlink が受理されました")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatalf("外部 directory を確認できませんでした: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("symlink 先に cache が作成されました: %v", entries)
	}
}

func TestPrepareHookCachePathsRejectsGroupWritableCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows では POSIX の書込権限を検証できません")
	}
	repository := newGitRepository(t)
	common := strings.TrimSpace(
		runGit(t, repository, "rev-parse", "--path-format=absolute", "--git-common-dir"),
	)
	cacheRoot := filepath.Join(common, "japanese-law-mcp-quality-cache")
	if err := os.Mkdir(cacheRoot, 0o750); err != nil {
		t.Fatalf("cache root を作成できませんでした: %v", err)
	}
	//nolint:gosec // SOT-ENG-021: 安全でない cache 権限を拒否するテスト fixture に限定する。
	if err := os.Chmod(cacheRoot, 0o770); err != nil {
		t.Fatalf("cache root の権限を変更できませんでした: %v", err)
	}

	if _, err := prepareHookCachePaths(t.Context(), repository); err == nil {
		t.Fatal("group writable な cache root が受理されました")
	}
}

func TestResolveRealDirectoryRejectsMissingPathAndFile(t *testing.T) {
	parent := t.TempDir()
	missing := filepath.Join(parent, "missing")
	if _, err := resolveRealDirectory(missing); err == nil {
		t.Fatal("存在しない directory が受理されました")
	}
	file := filepath.Join(parent, "file")
	writeFile(t, parent, "file", "not directory\n")
	if _, err := resolveRealDirectory(file); err == nil {
		t.Fatal("通常ファイルが directory として受理されました")
	}
}

func TestInstallDoesNotSetHooksPathWhenWarmUpFails(t *testing.T) {
	repository := newGitRepository(t)
	makeHookFiles(t, repository)
	app := newTestApplication(repository)
	app.warmUp = func(context.Context, string) error {
		return errors.New("意図したウォームアップ失敗")
	}

	code, _, _ := executeForTest(t, app, []string{"install"})

	if code == 0 {
		t.Fatal("ウォームアップ失敗時に install が成功しました")
	}
	if got := localHooksPath(t, t.Context(), repository); got != "" {
		t.Fatalf("失敗後に hooksPath が残りました: %s", got)
	}
}

func TestInstallOverridesButDoesNotModifyInheritedGlobalHooksPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, home, ".gitconfig", "[core]\n\thooksPath = 継承フック\n")
	repository := newGitRepository(t)
	makeHookFiles(t, repository)
	app := newTestApplication(repository)

	code, stdout, stderr := executeForTest(t, app, []string{"install"})

	if code != 0 {
		t.Fatalf("install が失敗しました: %s", stderr)
	}
	if got := localHooksPath(t, t.Context(), repository); got != ".githooks" {
		t.Fatalf("local hooksPath = %q, want .githooks", got)
	}
	global := strings.TrimSpace(runGit(t, repository, "config", "--global", "--get", "core.hooksPath"))
	if global != "継承フック" {
		t.Fatalf("global hooksPath が変更されました: %q", global)
	}
	if !strings.Contains(stdout, "上書き") {
		t.Fatalf("継承フックを隠す通知がありません: %s", stdout)
	}
}

func TestInstallDoesNotOverwriteAnotherLocalHooksPath(t *testing.T) {
	repository := newGitRepository(t)
	makeHookFiles(t, repository)
	runGit(t, repository, "config", "--local", "core.hooksPath", "既存のフック")
	app := newTestApplication(repository)
	warmed := false
	app.warmUp = func(context.Context, string) error {
		warmed = true
		return nil
	}

	code, _, _ := executeForTest(t, app, []string{"install"})

	if code == 0 {
		t.Fatal("異なる hooksPath があるのに install が成功しました")
	}
	if warmed {
		t.Fatal("設定衝突時に不要なウォームアップが実行されました")
	}
	if got := localHooksPath(t, t.Context(), repository); got != "既存のフック" {
		t.Fatalf("既存の hooksPath が変更されました: %s", got)
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	repository := newGitRepository(t)
	makeHookFiles(t, repository)
	runGit(t, repository, "config", "--local", "core.hooksPath", ".githooks")
	app := newTestApplication(repository)
	warmed := false
	app.warmUp = func(context.Context, string) error {
		warmed = true
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"install"})

	if code != 0 {
		t.Fatalf("再 install が失敗しました: %s", stderr)
	}
	if !warmed {
		t.Fatal("設定済みの再 install でウォームアップが実行されませんでした")
	}
}

func TestUninstallOnlyRemovesTheRepositoryLocalHooksPath(t *testing.T) {
	t.Run("configured by this repository", func(t *testing.T) {
		repository := newGitRepository(t)
		runGit(t, repository, "config", "--local", "core.hooksPath", ".githooks")
		app := newTestApplication(repository)

		code, _, stderr := executeForTest(t, app, []string{"uninstall"})

		if code != 0 {
			t.Fatalf("uninstall が失敗しました: %s", stderr)
		}
		if got := localHooksPath(t, t.Context(), repository); got != "" {
			t.Fatalf("local hooksPath が残っています: %s", got)
		}
	})

	t.Run("different local value", func(t *testing.T) {
		repository := newGitRepository(t)
		runGit(t, repository, "config", "--local", "core.hooksPath", "別のフック")
		app := newTestApplication(repository)

		code, _, _ := executeForTest(t, app, []string{"uninstall"})

		if code == 0 {
			t.Fatal("別の local hooksPath を uninstall が削除しました")
		}
		if got := localHooksPath(t, t.Context(), repository); got != "別のフック" {
			t.Fatalf("別の local hooksPath が変更されました: %s", got)
		}
	})
}

func TestCheckRequiresLocalSettingAndExecutableRegularHooks(t *testing.T) {
	repository := newGitRepository(t)
	makeHookFiles(t, repository)
	runGit(t, repository, "config", "--local", "core.hooksPath", ".githooks")
	if _, err := prepareHookCachePaths(t.Context(), repository); err != nil {
		t.Fatalf("共有 cache を準備できませんでした: %v", err)
	}
	app := newTestApplication(repository)

	code, _, stderr := executeForTest(t, app, []string{"check"})
	if code != 0 {
		t.Fatalf("正常な設定の check が失敗しました: %s", stderr)
	}

	if err := os.Chmod(filepath.Join(repository, ".githooks", "pre-push"), 0o600); err != nil {
		t.Fatalf("フックの権限を変更できませんでした: %v", err)
	}
	code, _, _ = executeForTest(t, app, []string{"check"})
	if code == 0 {
		t.Fatal("実行不能なフックを check が受理しました")
	}
}

func TestCheckDoesNotCreateMissingSharedCache(t *testing.T) {
	repository := newGitRepository(t)
	makeHookFiles(t, repository)
	runGit(t, repository, "config", "--local", "core.hooksPath", ".githooks")
	common := strings.TrimSpace(
		runGit(t, repository, "rev-parse", "--path-format=absolute", "--git-common-dir"),
	)
	cacheRoot := filepath.Join(common, qualityCacheDirectory)
	app := newTestApplication(repository)

	code, _, _ := executeForTest(t, app, []string{"check"})

	if code == 0 {
		t.Fatal("共有 cache がないのに check が成功しました")
	}
	assertNotExists(t, cacheRoot)
}

func TestCheckRejectsSymlinkedHookDirectory(t *testing.T) {
	repository := newGitRepository(t)
	outside := t.TempDir()
	makeHookFiles(t, outside)
	if err := os.Symlink(
		filepath.Join(outside, ".githooks"),
		filepath.Join(repository, ".githooks"),
	); err != nil {
		t.Fatalf("hook directory symlink を作成できませんでした: %v", err)
	}
	runGit(t, repository, "config", "--local", "core.hooksPath", ".githooks")
	app := newTestApplication(repository)

	code, _, _ := executeForTest(t, app, []string{"check"})

	if code == 0 {
		t.Fatal("リポジトリ外を指す hook directory が受理されました")
	}
}

func TestCheckRejectsTamperedHookContents(t *testing.T) {
	repository := newGitRepository(t)
	makeHookFiles(t, repository)
	runGit(t, repository, "config", "--local", "core.hooksPath", ".githooks")
	if _, err := prepareHookCachePaths(t.Context(), repository); err != nil {
		t.Fatalf("共有 cache を準備できませんでした: %v", err)
	}
	writeFile(t, repository, ".githooks/manage", "#!/bin/sh\nexit 0\n")
	//nolint:gosec // SOT-ENG-021: 変造済み hook の検出用 fixture にだけ実行権限を付ける。
	if err := os.Chmod(filepath.Join(repository, ".githooks", "manage"), 0o700); err != nil {
		t.Fatalf("変造済み hook を実行可能にできませんでした: %v", err)
	}
	app := newTestApplication(repository)

	code, _, _ := executeForTest(t, app, []string{"check"})

	if code == 0 {
		t.Fatal("本文が変造された hook を check が受理しました")
	}
}

func TestCheckRejectsOneByteHookChange(t *testing.T) {
	repository := newGitRepository(t)
	makeHookFiles(t, repository)
	runGit(t, repository, "config", "--local", "core.hooksPath", ".githooks")
	if _, err := prepareHookCachePaths(t.Context(), repository); err != nil {
		t.Fatalf("共有 cache を準備できませんでした: %v", err)
	}
	target := filepath.Join(repository, ".githooks", "pre-push")
	content := readTestFile(t, target)
	content[len(content)-1] ^= 1
	//nolint:gosec // SOT-ENG-021: 一 byte の hook 変造を検出する private test fixture に限定する。
	if err := os.WriteFile(target, content, 0o700); err != nil {
		t.Fatalf("一 byte を変更できませんでした: %v", err)
	}
	app := newTestApplication(repository)

	code, _, _ := executeForTest(t, app, []string{"check"})

	if code == 0 {
		t.Fatal("一 byte が変更された hook を check が受理しました")
	}
}

func makeHookFiles(t *testing.T, repository string) {
	t.Helper()

	sourceRoot := repositoryRootForTest(t)
	for _, name := range []string{"manage", "pre-commit", "pre-push"} {
		content := readTestFile(t, filepath.Join(sourceRoot, ".githooks", name))
		writeFile(t, repository, filepath.Join(".githooks", name), string(content))
		//nolint:gosec // SOT-ENG-021: Git hook の実行可能性を検証するテスト fixture だけに実行権限を付ける。
		if err := os.Chmod(filepath.Join(repository, ".githooks", name), 0o700); err != nil {
			t.Fatalf("フックへ実行権限を設定できませんでした: %v", err)
		}
	}
}

func localHooksPath(t *testing.T, ctx context.Context, repository string) string {
	t.Helper()

	command := gitCommand(ctx, repository, nil,
		"config", "--local", "--get", "core.hooksPath",
	)
	output, err := command.Output()
	if err != nil {
		if exitCode(err) == 1 {
			return ""
		}
		t.Fatalf("hooksPath を取得できませんでした: %v", err)
	}
	return stringWithoutLineEnding(output)
}
