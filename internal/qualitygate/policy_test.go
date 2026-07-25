package qualitygate

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestVerifySnapshotCachePolicyRejectsTrackedCacheContent(t *testing.T) {
	t.Parallel()

	for _, directory := range []string{".cache", ".tmp"} {
		directory := directory
		t.Run(directory, func(t *testing.T) {
			t.Parallel()
			snapshot := t.TempDir()
			path := filepath.Join(snapshot, directory, "tracked.txt")
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				t.Fatalf("キャッシュディレクトリを作成できません: %v", err)
			}
			if err := os.WriteFile(path, []byte("追跡してはいけない\n"), 0o600); err != nil {
				t.Fatalf("キャッシュ内容を作成できません: %v", err)
			}
			if err := verifySnapshotCachePolicy(snapshot); err == nil {
				t.Fatalf("%s 内の追跡対象が受理されました", directory)
			}
		})
	}

	if err := verifySnapshotCachePolicy(t.TempDir()); err != nil {
		t.Fatalf("キャッシュを含まない snapshot が拒否されました: %v", err)
	}

	snapshot := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(snapshot, "coverage.out"),
		[]byte("mode: atomic\n"),
		0o600,
	); err != nil {
		t.Fatalf("coverage.out を作成できません: %v", err)
	}
	if err := verifySnapshotCachePolicy(snapshot); err == nil {
		t.Fatal("追跡対象の coverage.out が受理されました")
	}
}

func TestPreCommitSecretsScanWholePinnedSnapshot(t *testing.T) {
	t.Parallel()

	steps, err := buildPlan(planInput{
		profile:      ProfilePreCommit,
		repository:   "/repo",
		snapshot:     "/snapshot",
		changedPaths: []string{"README.md"},
	})
	if err != nil {
		t.Fatalf("計画の作成に失敗しました: %v", err)
	}
	for _, current := range steps {
		if current.key != "staged-secrets" {
			continue
		}
		if current.command == nil ||
			current.command.dir != "/snapshot" ||
			!slices.Contains(current.command.args, "dir") ||
			!slices.Contains(current.command.args, ".") ||
			slices.Contains(current.command.args, "--staged") ||
			current.command.preserveGitIndex ||
			current.command.preserveGitObjects {
			t.Fatalf("index 全体の snapshot 検査ではありません: %#v", current.command)
		}
		return
	}
	t.Fatal("pre-commit の秘密情報検査がありません")
}

func TestPrePushRequiresAtLeastOneGitRange(t *testing.T) {
	t.Parallel()

	_, err := buildPlan(planInput{
		profile:    ProfilePrePush,
		repository: "/repo",
		snapshot:   "/snapshot",
	})
	if err == nil {
		t.Fatal("Git 範囲のない pre-push が受理されました")
	}
}

func TestPrepareCachePathsRejectsSymlinkedCacheRoot(t *testing.T) {
	t.Parallel()

	commonDirectory := t.TempDir()
	outside := t.TempDir()
	cacheRoot := filepath.Join(commonDirectory, "japanese-law-mcp-quality-cache")
	if err := os.Symlink(outside, cacheRoot); err != nil {
		t.Fatalf("キャッシュのシンボリックリンクを作成できません: %v", err)
	}
	if _, err := prepareCachePaths(commonDirectory); err == nil {
		t.Fatal("Git 共通ディレクトリ外を指すキャッシュが受理されました")
	}
}

func TestPrepareRunCachePathsIsolatesSnapshotLintCache(t *testing.T) {
	t.Parallel()

	commonDirectory := t.TempDir()
	gitRepository := t.TempDir()
	snapshot := t.TempDir()
	caches, cleanup, err := prepareRunCachePaths(commonDirectory, snapshot, gitRepository)
	if err != nil {
		t.Fatalf("snapshot 用 cache を準備できませんでした: %v", err)
	}
	sharedRoot := filepath.Join(commonDirectory, "japanese-law-mcp-quality-cache")
	if caches.goBuild != filepath.Join(sharedRoot, "go-build") {
		t.Fatalf("Go build cache が共有領域ではありません: %s", caches.goBuild)
	}
	if caches.golangci == filepath.Join(sharedRoot, "golangci") {
		t.Fatal("path 依存の lint cache が異なる snapshot 間で共有されています")
	}
	if err := cleanup(); err != nil {
		t.Fatalf("snapshot 用 lint cache を削除できませんでした: %v", err)
	}
	if _, err := os.Stat(caches.golangci); !os.IsNotExist(err) {
		t.Fatalf("snapshot 用 lint cache が残っています: %v", err)
	}
}

func TestPrepareRunCachePathsSharesLintCacheForCheckout(t *testing.T) {
	t.Parallel()

	commonDirectory := t.TempDir()
	repository := t.TempDir()
	caches, cleanup, err := prepareRunCachePaths(commonDirectory, repository, repository)
	if err != nil {
		t.Fatalf("checkout 用 cache を準備できませんでした: %v", err)
	}
	if want := filepath.Join(
		commonDirectory,
		"japanese-law-mcp-quality-cache",
		"golangci",
	); caches.golangci != want {
		t.Fatalf("checkout の lint cache = %s, want %s", caches.golangci, want)
	}
	if err := cleanup(); err != nil {
		t.Fatalf("共有 cache の no-op cleanup が失敗しました: %v", err)
	}
}
