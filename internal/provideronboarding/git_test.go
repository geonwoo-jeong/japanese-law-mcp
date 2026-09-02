package provideronboarding

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// SOT-ENG-044: 比較対象は commit、index、working tree、未追跡の和集合とする。
func TestCollectChangedPathsIncludesEveryRequiredGitLayer(t *testing.T) {
	t.Parallel()

	repository := newTestGitRepository(t, map[string]string{
		".gitignore":       "ignored.txt\n",
		"index.txt":        "base\n",
		"working-tree.txt": "base\n",
	})
	base := gitOutput(t, repository, "rev-parse", "HEAD")

	writeTestFile(t, repository, "committed.txt", "commit\n")
	gitRun(t, repository, "add", "committed.txt")
	gitCommit(t, repository, "commit change")

	writeTestFile(t, repository, "index.txt", "index\n")
	gitRun(t, repository, "add", "index.txt")
	writeTestFile(t, repository, "working-tree.txt", "working tree\n")
	writeTestFile(t, repository, "untracked.txt", "untracked\n")
	writeTestFile(t, repository, "ignored.txt", "ignored\n")

	client := newGitClient(repository)
	comparison, err := client.resolveComparison(context.Background(), base, "HEAD")
	if err != nil {
		t.Fatalf("比較開始点を解決できませんでした: %v", err)
	}
	paths, err := client.collectChangedPaths(context.Background(), comparison, changeSources{
		index:       true,
		workingTree: true,
		untracked:   true,
	})
	if err != nil {
		t.Fatalf("変更パスを収集できませんでした: %v", err)
	}

	want := []string{
		"committed.txt",
		"index.txt",
		"untracked.txt",
		"working-tree.txt",
	}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("変更パスが一致しません:\n got: %#v\nwant: %#v", paths, want)
	}
}

// SOT-ENG-044: 指定 commit そのものではなく HEAD との merge base を使用する。
func TestResolveComparisonUsesMergeBase(t *testing.T) {
	t.Parallel()

	repository := newTestGitRepository(t, map[string]string{"base.txt": "base\n"})
	common := gitOutput(t, repository, "rev-parse", "HEAD")

	gitRun(t, repository, "checkout", "-b", "side")
	writeTestFile(t, repository, "side.txt", "side\n")
	gitRun(t, repository, "add", "side.txt")
	gitCommit(t, repository, "side change")
	side := gitOutput(t, repository, "rev-parse", "HEAD")

	gitRun(t, repository, "checkout", "master")
	writeTestFile(t, repository, "head.txt", "head\n")
	gitRun(t, repository, "add", "head.txt")
	gitCommit(t, repository, "head change")

	comparison, err := newGitClient(repository).resolveComparison(
		context.Background(),
		side,
		"HEAD",
	)
	if err != nil {
		t.Fatalf("比較開始点を解決できませんでした: %v", err)
	}
	if comparison.mergeBase != common {
		t.Fatalf("merge base = %q, want %q", comparison.mergeBase, common)
	}
}

func TestGitOutputParsersRejectMalformedValues(t *testing.T) {
	t.Parallel()

	if _, err := parseOID([]byte("short\n")); err == nil {
		t.Fatal("短い object ID が許可されました")
	}
	if _, err := parseOID([]byte(strings.Repeat("g", 40) + "\n")); err == nil {
		t.Fatal("十六進数ではない object ID が許可されました")
	}
	if _, err := parseNULPaths([]byte("not-terminated")); err == nil {
		t.Fatal("NUL 終端ではない path 応答が許可されました")
	}
	for _, invalid := range []string{"", "../outside", "/absolute", `back\\slash`} {
		if err := validateGitPath(invalid); err == nil {
			t.Fatalf("不正な Git path %q が許可されました", invalid)
		}
	}
}

func TestCollectChangedPathsRejectsDivergentIndexAndWorkingTreeBytes(t *testing.T) {
	t.Parallel()

	repository := newTestGitRepository(t, map[string]string{
		"conformance/providers/provider-a.yaml": "base\n",
	})
	base := gitOutput(t, repository, "rev-parse", "HEAD")
	writeTestFile(
		t,
		repository,
		"conformance/providers/provider-a.yaml",
		"staged\n",
	)
	gitRun(t, repository, "add", "conformance/providers/provider-a.yaml")
	writeTestFile(
		t,
		repository,
		"conformance/providers/provider-a.yaml",
		"working tree\n",
	)

	client := newGitClient(repository)
	comparison, err := client.resolveComparison(context.Background(), base, "HEAD")
	if err != nil {
		t.Fatalf("比較開始点を解決できませんでした: %v", err)
	}
	_, err = client.collectChangedPaths(context.Background(), comparison, changeSources{
		index:       true,
		workingTree: true,
	})
	if err == nil {
		t.Fatal("index と working tree で内容が異なる path が許可されました")
	}
	if !strings.Contains(err.Error(), "conformance/providers/provider-a.yaml") {
		t.Fatalf("不一致 path を特定できないエラーです: %v", err)
	}
}

func TestComparisonPathContentsReadsEffectiveGitLayer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prepare    func(*testing.T, string, string)
		sources    changeSources
		wantBefore string
		wantAfter  string
	}{
		{
			name: "commit",
			prepare: func(t *testing.T, repository, matrixPath string) {
				writeTestFile(t, repository, matrixPath, "commit\n")
				gitRun(t, repository, "add", matrixPath)
				gitCommit(t, repository, "matrix commit")
			},
			wantBefore: "base\n",
			wantAfter:  "commit\n",
		},
		{
			name: "index",
			prepare: func(t *testing.T, repository, matrixPath string) {
				writeTestFile(t, repository, matrixPath, "index\n")
				gitRun(t, repository, "add", matrixPath)
			},
			sources:    changeSources{index: true},
			wantBefore: "base\n",
			wantAfter:  "index\n",
		},
		{
			name: "working tree",
			prepare: func(t *testing.T, repository, matrixPath string) {
				writeTestFile(t, repository, matrixPath, "working\n")
			},
			sources:    changeSources{workingTree: true},
			wantBefore: "base\n",
			wantAfter:  "working\n",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			const matrixPath = "conformance/providers/provider-a.yaml"
			repository := newTestGitRepository(t, map[string]string{
				matrixPath: "base\n",
			})
			base := gitOutput(t, repository, "rev-parse", "HEAD")
			test.prepare(t, repository, matrixPath)

			client := newGitClient(repository)
			comparison, err := client.resolveComparison(t.Context(), base, "HEAD")
			if err != nil {
				t.Fatalf("比較対象を解決できませんでした: %v", err)
			}
			changes, err := client.collectChanges(t.Context(), comparison, test.sources)
			if err != nil {
				t.Fatalf("変更 layer を収集できませんでした: %v", err)
			}
			before, after, both, err := client.comparisonPathContents(
				t.Context(),
				repository,
				comparison,
				changes,
				matrixPath,
			)
			if err != nil {
				t.Fatalf("matrix snapshot を読み取れませんでした: %v", err)
			}
			if !both || string(before) != test.wantBefore || string(after) != test.wantAfter {
				t.Fatalf(
					"matrix snapshot = both:%t before:%q after:%q",
					both,
					before,
					after,
				)
			}
		})
	}
}

func TestComparisonPathContentsDoesNotClassifyFileAdditionOrDeletion(t *testing.T) {
	t.Parallel()

	t.Run("addition", func(t *testing.T) {
		t.Parallel()
		repository := newTestGitRepository(t, map[string]string{"README.md": "base\n"})
		base := gitOutput(t, repository, "rev-parse", "HEAD")
		const matrixPath = "conformance/providers/provider-a.yaml"
		writeTestFile(t, repository, matrixPath, "added\n")
		client := newGitClient(repository)
		comparison, err := client.resolveComparison(t.Context(), base, "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		changes, err := client.collectChanges(
			t.Context(),
			comparison,
			changeSources{untracked: true},
		)
		if err != nil {
			t.Fatal(err)
		}
		_, _, both, err := client.comparisonPathContents(
			t.Context(),
			repository,
			comparison,
			changes,
			matrixPath,
		)
		if err != nil || both {
			t.Fatalf("追加 file が両 snapshot に存在しました: both=%t err=%v", both, err)
		}
	})

	t.Run("deletion", func(t *testing.T) {
		t.Parallel()
		const matrixPath = "conformance/providers/provider-a.yaml"
		repository := newTestGitRepository(t, map[string]string{matrixPath: "base\n"})
		base := gitOutput(t, repository, "rev-parse", "HEAD")
		if err := os.Remove(filepath.Join(repository, filepath.FromSlash(matrixPath))); err != nil {
			t.Fatal(err)
		}
		client := newGitClient(repository)
		comparison, err := client.resolveComparison(t.Context(), base, "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		changes, err := client.collectChanges(
			t.Context(),
			comparison,
			changeSources{workingTree: true},
		)
		if err != nil {
			t.Fatal(err)
		}
		_, _, both, err := client.comparisonPathContents(
			t.Context(),
			repository,
			comparison,
			changes,
			matrixPath,
		)
		if err != nil || both {
			t.Fatalf("削除 file が両 snapshot に存在しました: both=%t err=%v", both, err)
		}
	})
}

func TestComparisonPathContentsRejectsWorkingTreeSymlink(t *testing.T) {
	t.Parallel()

	const matrixPath = "conformance/providers/provider-a.yaml"
	repository := newTestGitRepository(t, map[string]string{
		matrixPath:    "base\n",
		"target.yaml": "target\n",
	})
	base := gitOutput(t, repository, "rev-parse", "HEAD")
	target := filepath.Join(repository, filepath.FromSlash(matrixPath))
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(repository, "target.yaml"), target); err != nil {
		t.Fatal(err)
	}
	client := newGitClient(repository)
	comparison, err := client.resolveComparison(t.Context(), base, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	changes, err := client.collectChanges(
		t.Context(),
		comparison,
		changeSources{workingTree: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = client.comparisonPathContents(
		t.Context(),
		repository,
		comparison,
		changes,
		matrixPath,
	)
	if err == nil {
		t.Fatal("working tree の symlink matrix が許可されました")
	}
}

func newTestGitRepository(t *testing.T, files map[string]string) string {
	t.Helper()

	repository := t.TempDir()
	gitRun(t, repository, "init", "--initial-branch=master")
	for name, content := range files {
		writeTestFile(t, repository, name, content)
	}
	gitRun(t, repository, "add", ".")
	gitCommit(t, repository, "base")
	return repository
}

func writeTestFile(t *testing.T, repository, name, content string) {
	t.Helper()

	target := filepath.Join(repository, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("親ディレクトリを作成できませんでした: %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatalf("テストファイルを書き込めませんでした: %v", err)
	}
}

func gitCommit(t *testing.T, repository, message string) {
	t.Helper()
	gitRun(
		t,
		repository,
		"-c",
		"user.name=Provider Onboarding Test",
		"-c",
		"user.email=provider-onboarding@example.invalid",
		"commit",
		"-m",
		message,
	)
}

func gitRun(t *testing.T, repository string, args ...string) {
	t.Helper()
	_ = gitOutput(t, repository, args...)
}

func gitOutput(t *testing.T, repository string, args ...string) string {
	t.Helper()

	output, err := testCommand(t.Context(), repository, "git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v が失敗しました: %v\n%s", args, err, output)
	}
	return trimCommandOutput(output)
}
