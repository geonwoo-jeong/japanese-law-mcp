package qualitygate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestVerifyDevelopmentPrinciplesChecksum(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	documentPath := filepath.Join(repository, "docs", "development-principles.md")
	if err := os.MkdirAll(filepath.Dir(documentPath), 0o750); err != nil {
		t.Fatalf("docs の作成に失敗しました: %v", err)
	}
	content := []byte("変更しない原則\n")
	if err := os.WriteFile(documentPath, content, 0o600); err != nil {
		t.Fatalf("開発原則の作成に失敗しました: %v", err)
	}
	sum := sha256.Sum256(content)
	checksum := hex.EncodeToString(sum[:]) + "  docs/development-principles.md\n"
	if err := os.WriteFile(
		filepath.Join(repository, "docs", "development-principles.sha256"),
		[]byte(checksum),
		0o600,
	); err != nil {
		t.Fatalf("チェックサムの作成に失敗しました: %v", err)
	}

	if err := verifyDevelopmentPrinciples(repository); err != nil {
		t.Fatalf("正しいチェックサムが拒否されました: %v", err)
	}

	if err := os.WriteFile(documentPath, []byte("改変\n"), 0o600); err != nil {
		t.Fatalf("開発原則の改変に失敗しました: %v", err)
	}
	if err := verifyDevelopmentPrinciples(repository); err == nil {
		t.Fatal("不一致のチェックサムが受理されました")
	}
}

func TestVerifyDevelopmentPrinciplesRejectsUnexpectedManifest(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repository, "docs"), 0o750); err != nil {
		t.Fatalf("docs の作成に失敗しました: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(repository, "docs", "development-principles.sha256"),
		[]byte(strings.Repeat("0", 64)+"  ../outside\n"),
		0o600,
	); err != nil {
		t.Fatalf("チェックサムの作成に失敗しました: %v", err)
	}

	err := verifyDevelopmentPrinciples(repository)
	if err == nil || !strings.Contains(err.Error(), "docs/development-principles.md") {
		t.Fatalf("想定外の対象を拒否していません: %v", err)
	}
}

func TestGofmtOutputMustBeEmpty(t *testing.T) {
	t.Parallel()

	if err := requireEmptyOutput(nil); err != nil {
		t.Fatalf("空の出力が拒否されました: %v", err)
	}
	if err := requireEmptyOutput([]byte("internal/example/example.go\n")); err == nil {
		t.Fatal("未整形ファイルの出力が受理されました")
	}
}

func TestStagedPathsHandlesGitFailuresAndNULTermination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		result   executionResult
		want     []string
		wantFail bool
	}{
		{name: "変更なし", result: executionResult{}, want: nil},
		{
			name:   "NUL 区切り",
			result: executionResult{stdout: []byte("b.go\x00a.go\x00a.go\x00")},
			want:   []string{"a.go", "b.go"},
		},
		{
			name:     "NUL 終端がない",
			result:   executionResult{stdout: []byte("a.go")},
			wantFail: true,
		},
		{
			name:     "Git の失敗",
			result:   executionResult{stderr: []byte("Git エラー"), err: errors.New("失敗")},
			wantFail: true,
		},
		{
			name:     "危険なパス",
			result:   executionResult{stdout: []byte("../outside.go\x00")},
			wantFail: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			executor := &recordingExecutor{results: []executionResult{tt.result}}
			got, err := stagedPaths(executor, "/repo")
			if tt.wantFail {
				if err == nil {
					t.Fatal("失敗が返されませんでした")
				}
				return
			}
			if err != nil {
				t.Fatalf("ステージ済みパスを取得できません: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("変更パス = %q, want %q", got, tt.want)
			}
			assertCommandBoundary(t, executor.specs[0], "/repo", "git")
			if slices.Contains(executor.specs[0].args, "--diff-filter=ACMR") {
				t.Fatalf("削除を除外する diff-filter が残っています: %q", executor.specs[0].args)
			}
			if !slices.Contains(executor.specs[0].args, "--no-renames") {
				t.Fatalf("rename の旧新パスを検査する指定がありません: %q", executor.specs[0].args)
			}
		})
	}
}

func TestExistingRegularGoFilesSkipsDeletionAndRejectsSymlink(t *testing.T) {
	t.Parallel()

	snapshot := t.TempDir()
	regular := filepath.Join(snapshot, "internal", "regular.go")
	if err := os.MkdirAll(filepath.Dir(regular), 0o750); err != nil {
		t.Fatalf("ディレクトリを作成できません: %v", err)
	}
	if err := os.WriteFile(regular, []byte("package internal\n"), 0o600); err != nil {
		t.Fatalf("Go ファイルを作成できません: %v", err)
	}
	got, err := existingRegularGoFiles(
		snapshot,
		[]string{"internal/regular.go", "internal/deleted.go", "README.md"},
	)
	if err != nil {
		t.Fatalf("Go ファイルを列挙できません: %v", err)
	}
	if want := []string{"internal/regular.go"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Go ファイル = %q, want %q", got, want)
	}

	symlink := filepath.Join(snapshot, "internal", "linked.go")
	if err := os.Symlink(regular, symlink); err != nil {
		t.Fatalf("シンボリックリンクを作成できません: %v", err)
	}
	if _, err := existingRegularGoFiles(snapshot, []string{"internal/linked.go"}); err == nil {
		t.Fatal("Go のシンボリックリンクが受理されました")
	}
}

func TestGitCommonDirectorySupportsLinkedWorktreeAndCreatesPrivateCaches(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	worktree := filepath.Join(root, "worktrees", "feature")
	common := filepath.Join(root, ".git")
	for _, path := range []string{worktree, common} {
		if err := os.MkdirAll(path, 0o750); err != nil {
			t.Fatalf("ディレクトリを作成できません: %v", err)
		}
	}
	executor := &recordingExecutor{
		results: []executionResult{{stdout: []byte("../../.git\n")}},
	}

	got, err := gitCommonDirectory(executor, worktree)
	if err != nil {
		t.Fatalf("Git 共通ディレクトリを解決できません: %v", err)
	}
	resolvedCommon := resolvedPath(t, common)
	if got != resolvedCommon {
		t.Fatalf("Git 共通ディレクトリ = %q, want %q", got, resolvedCommon)
	}
	common = resolvedCommon
	caches, err := prepareCachePaths(got)
	if err != nil {
		t.Fatalf("キャッシュを作成できません: %v", err)
	}
	for _, path := range []string{caches.goBuild, caches.golangci} {
		info, statErr := os.Stat(path)
		if statErr != nil {
			t.Fatalf("キャッシュを確認できません: %v", statErr)
		}
		if !info.IsDir() {
			t.Fatalf("キャッシュがディレクトリではありません: %s", path)
		}
		if !strings.HasPrefix(path, common+string(filepath.Separator)) {
			t.Fatalf("キャッシュが Git 共通ディレクトリ外です: %s", path)
		}
	}
}

func TestGitCommonDirectoryRejectsInvalidResponses(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	tests := []executionResult{
		{stderr: []byte("Git 失敗"), err: errors.New("失敗")},
		{stdout: nil},
		{stdout: []byte("one\ntwo\n")},
		{stdout: []byte("missing\n")},
	}
	for _, result := range tests {
		executor := &recordingExecutor{results: []executionResult{result}}
		if _, err := gitCommonDirectory(executor, repository); err == nil {
			t.Fatalf("不正な応答が受理されました: %#v", result)
		}
	}
}

func TestVerifyTotalCoverage(t *testing.T) {
	t.Parallel()

	profilePath := filepath.Join(t.TempDir(), "coverage.out")
	profile := strings.Join([]string{
		"mode: atomic",
		"github.com/example/project/a.go:1.1,2.1 3 1",
		"github.com/example/project/a.go:1.1,2.1 3 0",
		"github.com/example/project/b.go:1.1,2.1 1 0",
	}, "\n") + "\n"
	if err := os.WriteFile(profilePath, []byte(profile), 0o600); err != nil {
		t.Fatalf("coverage profile を作成できませんでした: %v", err)
	}

	if err := verifyTotalCoverage(profilePath, 75); err != nil {
		t.Fatalf("重複 block を統合すると 75%% を満たす coverage が拒否されました: %v", err)
	}
	if err := verifyTotalCoverage(profilePath, 80); err == nil {
		t.Fatal("下限未満の coverage が受理されました")
	}
}

func TestVerifyTotalCoverageRejectsInvalidProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{name: "空", content: ""},
		{name: "先頭行不正", content: "invalid\n"},
		{name: "項目数不足", content: "mode: atomic\nfile.go:1.1,2.1 3\n"},
		{name: "statement 不正", content: "mode: atomic\nfile.go:1.1,2.1 x 1\n"},
		{name: "count 不正", content: "mode: atomic\nfile.go:1.1,2.1 1 x\n"},
		{name: "statement なし", content: "mode: atomic\n"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			profilePath := filepath.Join(t.TempDir(), "coverage.out")
			if err := os.WriteFile(profilePath, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("coverage profile を作成できませんでした: %v", err)
			}
			if err := verifyTotalCoverage(profilePath, 80); err == nil {
				t.Fatalf("不正な profile が受理されました: %s", fmt.Sprintf("%q", tt.content))
			}
		})
	}
}

func TestVerifyTotalCoverageAcceptsLongProfileLine(t *testing.T) {
	t.Parallel()

	profilePath := filepath.Join(t.TempDir(), "coverage.out")
	longFileName := strings.Repeat("very-long-directory-name/", 4000) + "target.go"
	profile := strings.Join([]string{
		"mode: atomic",
		longFileName + ":1.1,2.1 1 1",
	}, "\n") + "\n"
	if err := os.WriteFile(profilePath, []byte(profile), 0o600); err != nil {
		t.Fatalf("coverage profile を作成できませんでした: %v", err)
	}

	if err := verifyTotalCoverage(profilePath, 100); err != nil {
		t.Fatalf("長い coverage 行が拒否されました: %v", err)
	}
}

func TestVerifyTotalCoverageIgnoresBlankLines(t *testing.T) {
	t.Parallel()

	profilePath := filepath.Join(t.TempDir(), "coverage.out")
	profile := strings.Join([]string{
		"mode: atomic",
		"",
		"github.com/example/project/a.go:1.1,2.1 1 1",
		"",
	}, "\n")
	if err := os.WriteFile(profilePath, []byte(profile), 0o600); err != nil {
		t.Fatalf("coverage profile を作成できませんでした: %v", err)
	}

	if err := verifyTotalCoverage(profilePath, 100); err != nil {
		t.Fatalf("末尾や途中の空行を含む coverage が拒否されました: %v", err)
	}
}
