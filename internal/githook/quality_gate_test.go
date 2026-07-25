package githook

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunQualityGateUsesSnapshotRunnerAndPreservesGitRepositoryAndExit(t *testing.T) {
	directory := t.TempDir()
	argumentsPath := filepath.Join(directory, "arguments")
	workingDirectoryPath := filepath.Join(directory, "working-directory")
	cacheEnvironmentPath := filepath.Join(directory, "cache-environment")
	fakeGo := filepath.Join(directory, "go")
	script := "#!/bin/sh\n" +
		"pwd > " + shellQuote(workingDirectoryPath) + "\n" +
		"printf '%s\\n' \"$@\" > " + shellQuote(argumentsPath) + "\n" +
		"printf '%s\\n' \"$GOCACHE\" \"$GOLANGCI_LINT_CACHE\" > " +
		shellQuote(cacheEnvironmentPath) + "\n" +
		"exit 17\n"
	if err := os.WriteFile(fakeGo, []byte(script), 0o600); err != nil {
		t.Fatalf("偽の go を作成できませんでした: %v", err)
	}
	//nolint:gosec // SOT-ENG-021: quality gate 起動テストの private fixture にだけ実行権限を付ける。
	if err := os.Chmod(fakeGo, 0o700); err != nil {
		t.Fatalf("偽の go を実行可能にできませんでした: %v", err)
	}
	runner := newGitRepository(t)
	caches, err := prepareHookCachePaths(t.Context(), runner)
	if err != nil {
		t.Fatalf("共有 cache を準備できませんでした: %v", err)
	}
	target := t.TempDir()
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))

	err = runQualityGate(
		context.Background(),
		"pre-push",
		target,
		runner,
		"",
		[]string{"old..new", "new"},
		os.Stdout,
		os.Stderr,
	)

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 17 {
		t.Fatalf("終了コードが保持されませんでした: %v", err)
	}
	workingDirectory := readTestFile(t, workingDirectoryPath)
	resolvedRunner, resolveErr := filepath.EvalSymlinks(runner)
	if resolveErr != nil {
		t.Fatalf("runner を解決できませんでした: %v", resolveErr)
	}
	resolvedTarget, resolveErr := filepath.EvalSymlinks(target)
	if resolveErr != nil {
		t.Fatalf("target を解決できませんでした: %v", resolveErr)
	}
	if strings.TrimSpace(string(workingDirectory)) != resolvedTarget {
		t.Fatalf("runner cwd = %q, want snapshot %q (git repository %q)", workingDirectory, target, resolvedRunner)
	}
	arguments := readTestFile(t, argumentsPath)
	got := strings.Split(strings.TrimSpace(string(arguments)), "\n")
	want := []string{
		"run",
		"./cmd/quality-gate",
		"--profile=pre-push",
		"--repository=" + target,
		"--git-repository=" + runner,
		"--git-range=old..new",
		"--git-range=new",
	}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("quality gate 引数 = %#v, want %#v", got, want)
	}
	cacheEnvironment := readTestFile(t, cacheEnvironmentPath)
	if got, want := string(cacheEnvironment),
		caches.goBuild+"\n"+caches.golangci+"\n"; got != want {
		t.Fatalf("quality gate の cache 環境 = %q, want %q", got, want)
	}
}

func TestRunQualityGateRejectsUnsafeCacheBeforeStartingGo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows では POSIX の書込権限を検証できません")
	}
	repository := newGitRepository(t)
	caches, err := prepareHookCachePaths(t.Context(), repository)
	if err != nil {
		t.Fatalf("共有 cache を準備できませんでした: %v", err)
	}
	cacheRoot := filepath.Dir(caches.goBuild)
	//nolint:gosec // SOT-ENG-021: runtime が安全でない cache を拒否する test fixture に限定する。
	if err := os.Chmod(cacheRoot, 0o770); err != nil {
		t.Fatalf("cache root の権限を変更できませんでした: %v", err)
	}
	directory := t.TempDir()
	marker := filepath.Join(directory, "go-called")
	writeExecutable(t, filepath.Join(directory, "go"),
		"#!/bin/sh\nprintf called > "+shellQuote(marker)+"\n",
	)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))

	err = runQualityGate(
		t.Context(),
		"pre-commit",
		t.TempDir(),
		repository,
		"",
		nil,
		os.Stdout,
		os.Stderr,
	)

	if err == nil {
		t.Fatal("安全でない cache で quality gate が起動しました")
	}
	assertNotExists(t, marker)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
