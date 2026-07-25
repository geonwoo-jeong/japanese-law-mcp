package githook

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreCommitUsesIndexNotWorktreeForPartialStaging(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "対象.txt", "基準\n")
	commitAll(t, repository, "基準")
	writeFile(t, repository, "対象.txt", "ステージ済み\n")
	runGit(t, repository, "add", "対象.txt")
	writeFile(t, repository, "対象.txt", "作業ツリーのみ\n")
	app := newTestApplication(repository)
	var call gateCall
	app.qualityGate = func(
		_ context.Context,
		profile, snapshot, gotGitRepository string,
		gitIndexFile string,
		gitRanges []string,
		_, _ io.Writer,
	) error {
		call = gateCall{
			profile:       profile,
			repository:    snapshot,
			gitRepository: gotGitRepository,
			gitIndexFile:  gitIndexFile,
			gitRanges:     gitRanges,
		}
		content := readTestFile(t, filepath.Join(snapshot, "対象.txt"))
		if string(content) != "ステージ済み\n" {
			return errors.New("インデックスと異なる内容です")
		}
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-commit"})

	if code != 0 {
		t.Fatalf("pre-commit が失敗しました: %s", stderr)
	}
	if call.profile != "pre-commit" || call.gitRepository != repository {
		t.Fatalf("quality gate の引数が不正です: %#v", call)
	}
	if len(call.gitRanges) != 0 {
		t.Fatalf("pre-commit に git range が渡されました: %v", call.gitRanges)
	}
	if call.gitIndexFile == "" {
		t.Fatal("固定済み index file が quality gate に渡されませんでした")
	}
	assertNotExists(t, call.repository)
	assertNotExists(t, call.gitIndexFile)
	content := readTestFile(t, filepath.Join(repository, "対象.txt"))
	if string(content) != "作業ツリーのみ\n" {
		t.Fatalf("作業ツリーが変更されました: %q", content)
	}
}

func TestPreCommitPreservesGitProvidedAlternateIndex(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "対象.txt", "既定 index\n")
	commitAll(t, repository, "基準")
	alternateIndex := filepath.Join(t.TempDir(), "alternate-index")
	runGitWithEnvironment(
		t,
		repository,
		[]string{"GIT_INDEX_FILE=" + alternateIndex},
		"read-tree",
		"HEAD",
	)
	writeFile(t, repository, "対象.txt", "alternate index\n")
	runGitWithEnvironment(
		t,
		repository,
		[]string{"GIT_INDEX_FILE=" + alternateIndex},
		"add",
		"対象.txt",
	)
	writeFile(t, repository, "対象.txt", "作業ツリー\n")
	t.Setenv("GIT_INDEX_FILE", alternateIndex)
	app := newTestApplication(repository)
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		pinnedIndex string,
		_ []string,
		_, _ io.Writer,
	) error {
		if pinnedIndex == "" || pinnedIndex == alternateIndex {
			return errors.New("alternate index が独立した一時 index に固定されていません")
		}
		content := readTestFile(t, filepath.Join(snapshot, "対象.txt"))
		if string(content) != "alternate index\n" {
			return errors.New("Git が渡した alternate index と snapshot が一致しません")
		}
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-commit"})

	if code != 0 {
		t.Fatalf("alternate index の検査が失敗しました: %s", stderr)
	}
}

func TestPreCommitPinsIndexBeforeConcurrentChange(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "対象.txt", "基準\n")
	commitAll(t, repository, "基準")
	writeFile(t, repository, "対象.txt", "固定対象\n")
	runGit(t, repository, "add", "対象.txt")
	app := newTestApplication(repository)
	app.indexPinned = func(string) {
		writeFile(t, repository, "対象.txt", "後発変更\n")
		runGit(t, repository, "add", "対象.txt")
	}
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		_ string,
		_ []string,
		_, _ io.Writer,
	) error {
		content := readTestFile(t, filepath.Join(snapshot, "対象.txt"))
		if string(content) != "固定対象\n" {
			return errors.New("固定した tree と異なる snapshot です")
		}
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-commit"})

	if code != 0 {
		t.Fatalf("固定済み index の検査が失敗しました: %s", stderr)
	}
	if got := runGit(t, repository, "show", ":対象.txt"); got != "後発変更\n" {
		t.Fatalf("後発の index 更新が保持されませんでした: %q", got)
	}
}

func TestPreCommitMaterializesRawIndexBlobWithoutCheckoutConversion(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, ".gitattributes", "対象.txt filter=adversarial text eol=crlf\n")
	runGit(t, repository, "config", "filter.adversarial.clean", "cat")
	runGit(t, repository, "config", "filter.adversarial.smudge", "false")
	runGit(t, repository, "config", "filter.adversarial.required", "true")
	writeFile(t, repository, "対象.txt", "index blob\n")
	runGit(t, repository, "add", ".gitattributes", "対象.txt")
	app := newTestApplication(repository)
	called := false
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		_ string,
		_ []string,
		_, _ io.Writer,
	) error {
		called = true
		content := readTestFile(t, filepath.Join(snapshot, "対象.txt"))
		if got, want := string(content), "index blob\n"; got != want {
			return errors.New("index blob が checkout 変換されました")
		}
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-commit"})

	if code != 0 {
		t.Fatalf("raw index blob の検査が失敗しました: %s", stderr)
	}
	if !called {
		t.Fatal("raw index snapshot に quality gate が実行されませんでした")
	}
}

func TestPreCommitMaterializesOnlyTheStagedHunk(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "部分.txt", "一行目\n二行目\n")
	commitAll(t, repository, "基準")
	writeFile(t, repository, "部分.txt", "ステージ済み一行目\n未ステージ二行目\n")
	patch := strings.Join([]string{
		"diff --git a/部分.txt b/部分.txt",
		"--- a/部分.txt",
		"+++ b/部分.txt",
		"@@ -1,2 +1,2 @@",
		"-一行目",
		"+ステージ済み一行目",
		" 二行目",
		"",
	}, "\n")
	command := gitCommand(
		t.Context(),
		repository,
		strings.NewReader(patch),
		"apply",
		"--cached",
		"--whitespace=nowarn",
		"-",
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("部分 patch を stage できませんでした: %v\n%s", err, output)
	}
	app := newTestApplication(repository)
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		_ string,
		_ []string,
		_, _ io.Writer,
	) error {
		content := readTestFile(t, filepath.Join(snapshot, "部分.txt"))
		if got, want := string(content), "ステージ済み一行目\n二行目\n"; got != want {
			return errors.New("部分 stage の snapshot が一致しません")
		}
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-commit"})

	if code != 0 {
		t.Fatalf("部分 stage の検査が失敗しました: %s", stderr)
	}
}

func TestPreCommitSupportsUnbornHeadAndStagedDeletion(t *testing.T) {
	t.Run("unborn HEAD", func(t *testing.T) {
		repository := newGitRepository(t)
		writeFile(t, repository, "空白 と 日本語.txt", "追加\n")
		runGit(t, repository, "add", "空白 と 日本語.txt")
		app := newTestApplication(repository)
		app.qualityGate = func(
			_ context.Context,
			_, snapshot, _ string,
			_ string,
			_ []string,
			_, _ io.Writer,
		) error {
			content := readTestFile(t, filepath.Join(snapshot, "空白 と 日本語.txt"))
			if string(content) != "追加\n" {
				return errors.New("追加内容が一致しません")
			}
			return nil
		}

		code, _, stderr := executeForTest(t, app, []string{"pre-commit"})
		if code != 0 {
			t.Fatalf("unborn HEAD の検査が失敗しました: %s", stderr)
		}
	})

	t.Run("staged deletion", func(t *testing.T) {
		repository := newGitRepository(t)
		writeFile(t, repository, "削除.txt", "削除対象\n")
		commitAll(t, repository, "追加")
		if err := os.Remove(filepath.Join(repository, "削除.txt")); err != nil {
			t.Fatalf("削除できませんでした: %v", err)
		}
		runGit(t, repository, "add", "--all")
		app := newTestApplication(repository)
		app.qualityGate = func(
			_ context.Context,
			_, snapshot, _ string,
			_ string,
			_ []string,
			_, _ io.Writer,
		) error {
			if _, err := os.Lstat(filepath.Join(snapshot, "削除.txt")); !errors.Is(err, os.ErrNotExist) {
				return errors.New("削除済みファイルが snapshot に存在します")
			}
			return nil
		}

		code, _, stderr := executeForTest(t, app, []string{"pre-commit"})
		if code != 0 {
			t.Fatalf("削除の検査が失敗しました: %s", stderr)
		}
	})
}

func TestPreCommitRejectsSymlinkAndCleansSnapshot(t *testing.T) {
	repository := newGitRepository(t)
	outside := filepath.Join(t.TempDir(), "外部.txt")
	writeFile(t, filepath.Dir(outside), filepath.Base(outside), "外部\n")
	if err := os.Symlink(outside, filepath.Join(repository, "リンク")); err != nil {
		t.Fatalf("symlink を作成できませんでした: %v", err)
	}
	runGit(t, repository, "add", "リンク")
	app := newTestApplication(repository)
	called := false
	app.qualityGate = func(
		context.Context,
		string,
		string,
		string,
		string,
		[]string,
		io.Writer,
		io.Writer,
	) error {
		called = true
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-commit"})

	if code == 0 {
		t.Fatal("symlink を含む index が受理されました")
	}
	if called {
		t.Fatal("危険な snapshot に quality gate が実行されました")
	}
	if !strings.Contains(stderr, "シンボリックリンク") {
		t.Fatalf("原因が表示されませんでした: %s", stderr)
	}
}

func TestPreCommitPreservesQualityGateExitCode(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "go.mod", "module example.invalid/test\n")
	runGit(t, repository, "add", "go.mod")
	app := newTestApplication(repository)
	app.qualityGate = func(
		context.Context,
		string,
		string,
		string,
		string,
		[]string,
		io.Writer,
		io.Writer,
	) error {
		return testExitError{code: 23}
	}

	code, _, _ := executeForTest(t, app, []string{"pre-commit"})

	if code != 23 {
		t.Fatalf("終了コード = %d, want 23", code)
	}
}

type testExitError struct {
	code int
}

func (err testExitError) Error() string {
	return "テスト用終了"
}

func (err testExitError) ExitCode() int {
	return err.code
}
