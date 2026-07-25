package githook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type gateCall struct {
	profile       string
	repository    string
	gitRepository string
	gitIndexFile  string
	gitRanges     []string
}

func newTestApplication(repository string) *application {
	return &application{
		repository: repository,
		stdin:      strings.NewReader(""),
		stdout:     io.Discard,
		stderr:     io.Discard,
		warmUp:     func(context.Context, string) error { return nil },
		qualityGate: func(
			context.Context,
			string,
			string,
			string,
			string,
			[]string,
			io.Writer,
			io.Writer,
		) error {
			return nil
		},
	}
}

func newGitRepository(t *testing.T) string {
	t.Helper()

	repository := t.TempDir()
	runGit(t, repository, "init", "--quiet")
	runGit(t, repository, "config", "user.name", "テスト利用者")
	runGit(t, repository, "config", "user.email", "test@example.invalid")
	return repository
}

func writeFile(t *testing.T, root, name, contents string) {
	t.Helper()

	target := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("親ディレクトリを作成できませんでした: %v", err)
	}
	if err := os.WriteFile(target, []byte(contents), 0o600); err != nil {
		t.Fatalf("テストファイルを書き込めませんでした: %v", err)
	}
}

func commitAll(t *testing.T, repository, message string) string {
	t.Helper()

	runGit(t, repository, "add", "--all")
	runGit(t, repository, "commit", "--quiet", "-m", message)
	return strings.TrimSpace(runGit(t, repository, "rev-parse", "HEAD"))
}

func runGit(t *testing.T, repository string, args ...string) string {
	t.Helper()

	//nolint:gosec // SOT-ENG-021: テストが組み立てる固定 Git argv だけを一時リポジトリへ渡す。
	command := exec.CommandContext(t.Context(), "git", append([]string{"-C", repository}, args...)...)
	command.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_DATE=2020-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2020-01-01T00:00:00Z",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s が失敗しました: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func runGitWithEnvironment(
	t *testing.T,
	repository string,
	environment []string,
	args ...string,
) string {
	t.Helper()

	//nolint:gosec // SOT-ENG-021: テストが組み立てる固定 Git argv だけを一時リポジトリへ渡す。
	command := exec.CommandContext(t.Context(), "git", append([]string{"-C", repository}, args...)...)
	command.Env = append(os.Environ(), environment...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("環境付き git %s が失敗しました: %v\n%s", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("一時パスが残っています: %s (%v)", path, err)
	}
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()

	//nolint:gosec // SOT-ENG-021: テストが作成した private 一時パスだけを読み取る。
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("テストファイルを読み取れませんでした: %v", err)
	}
	return content
}

func oidLine(localRef, localOID, remoteRef, remoteOID string) string {
	return fmt.Sprintf("%s %s %s %s\n", localRef, localOID, remoteRef, remoteOID)
}

func zeroOID(length int) string {
	return strings.Repeat("0", length)
}

func executeForTest(
	t *testing.T,
	app *application,
	args []string,
) (int, string, string) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app.stdout = &stdout
	app.stderr = &stderr
	code := app.run(context.Background(), args)
	return code, stdout.String(), stderr.String()
}

func TestExecuteFindsRepositoryAndRejectsUnknownOperation(t *testing.T) {
	repository := newGitRepository(t)
	t.Chdir(repository)
	var stderr bytes.Buffer

	code := Execute(
		t.Context(),
		[]string{"unknown"},
		strings.NewReader(""),
		io.Discard,
		&stderr,
	)

	if code == 0 {
		t.Fatal("未知の操作が成功しました")
	}
	if !strings.Contains(stderr.String(), "未対応") {
		t.Fatalf("未知の操作の理由がありません: %s", stderr.String())
	}
}

func TestExecuteRejectsDirectoryOutsideGitRepository(t *testing.T) {
	t.Chdir(t.TempDir())
	var stderr bytes.Buffer

	code := Execute(
		t.Context(),
		[]string{"check"},
		strings.NewReader(""),
		io.Discard,
		&stderr,
	)

	if code == 0 {
		t.Fatal("Git repository 外で check が成功しました")
	}
	if !strings.Contains(stderr.String(), "特定できません") {
		t.Fatalf("repository 検出失敗の理由がありません: %s", stderr.String())
	}
}
