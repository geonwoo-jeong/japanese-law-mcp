package githook

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHookWrappersPreserveArgumentsStdinAndExitCode(t *testing.T) {
	repository := repositoryRootForTest(t)
	for _, test := range []struct {
		name      string
		arguments []string
		stdin     string
		want      []string
	}{
		{
			name:      "pre-commit",
			arguments: []string{"引数 空白"},
			stdin:     "commit input\n",
			want:      []string{"run", "./cmd/git-hook", "pre-commit", "引数 空白"},
		},
		{
			name:      "pre-push",
			arguments: []string{"origin", "ssh://example.invalid/空白 path"},
			stdin:     "push input\n",
			want: []string{
				"run",
				"./cmd/git-hook",
				"pre-push",
				"origin",
				"ssh://example.invalid/空白 path",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			argumentsPath := filepath.Join(directory, "arguments")
			stdinPath := filepath.Join(directory, "stdin")
			gitEnvironmentPath := filepath.Join(directory, "git-environment")
			goEnvironmentPath := filepath.Join(directory, "go-environment")
			repositoryForGit := filepath.ToSlash(repository)
			commonDirectory := filepath.Join(directory, "git common")
			if err := os.Mkdir(commonDirectory, 0o750); err != nil {
				t.Fatalf("Git common directory を作成できませんでした: %v", err)
			}
			resolvedCommon, resolveErr := filepath.EvalSymlinks(commonDirectory)
			if resolveErr != nil {
				t.Fatalf("Git common directory の実体を解決できませんでした: %v", resolveErr)
			}
			commonDirectory = resolvedCommon
			caches, err := createHookCachePaths(commonDirectory)
			if err != nil {
				t.Fatalf("共有 cache を作成できませんでした: %v", err)
			}
			writeExecutable(t, filepath.Join(directory, "git"),
				"#!/bin/sh\n"+
					"case \"$*\" in\n"+
					"  'rev-parse --show-toplevel') printf '%s\\n' "+shellQuote(repositoryForGit)+" ;;\n"+
					"  'rev-parse --path-format=absolute --git-common-dir') printf '%s\\n' "+
					shellQuote(commonDirectory)+" ;;\n"+
					"  *) exit 31 ;;\n"+
					"esac\n",
			)
			writeExecutable(t, filepath.Join(directory, "go"),
				"#!/bin/sh\n"+
					"printf '%s\\n' \"$@\" > "+shellQuote(argumentsPath)+"\n"+
					"printf '%s\\n' \"$GIT_INDEX_FILE\" \"$GIT_OBJECT_DIRECTORY\" \"$GIT_NO_REPLACE_OBJECTS\" > "+
					shellQuote(gitEnvironmentPath)+"\n"+
					"printf '%s\\n' \"$GOENV|$GOTOOLCHAIN|$GOWORK|$GOFLAGS|$GOPROXY|$GOVCS|$LC_ALL|$GOCACHE|$GOLANGCI_LINT_CACHE\" > "+
					shellQuote(goEnvironmentPath)+"\n"+
					"printf '%s\\n' \"$GOOS|$GOARCH|$GOEXPERIMENT|$GO111MODULE|$CGO_ENABLED\" >> "+
					shellQuote(goEnvironmentPath)+"\n"+
					"cat > "+shellQuote(stdinPath)+"\n"+
					"exit 29\n",
			)
			//nolint:gosec // SOT-ENG-021: リポジトリ内の固定 hook とテスト定義の argv だけを実行する。
			command := exec.CommandContext(
				t.Context(),
				filepath.Join(repository, ".githooks", test.name),
				test.arguments...,
			)
			command.Env = append(
				os.Environ(),
				"PATH="+directory,
				"GIT_INDEX_FILE=/tmp/alternate-index",
				"GIT_OBJECT_DIRECTORY=/tmp/alternate-objects",
				"GIT_NO_REPLACE_OBJECTS=0",
				"GOOS=windows",
				"GOARCH=386",
				"GOEXPERIMENT=fieldtrack",
				"GO111MODULE=off",
				"CGO_ENABLED=1",
				"GOCACHE=/tmp/unsafe-ambient-build-cache",
				"GOLANGCI_LINT_CACHE=/tmp/unsafe-ambient-lint-cache",
			)
			command.Stdin = strings.NewReader(test.stdin)
			var stderr bytes.Buffer
			command.Stderr = &stderr

			runErr := command.Run()

			var exitErr *exec.ExitError
			if !errors.As(runErr, &exitErr) || exitErr.ExitCode() != 29 {
				t.Fatalf("wrapper の終了コード = %v, stderr=%s", runErr, stderr.String())
			}
			arguments := readTestFile(t, argumentsPath)
			got := strings.Split(strings.TrimSuffix(string(arguments), "\n"), "\n")
			if strings.Join(got, "\x00") != strings.Join(test.want, "\x00") {
				t.Fatalf("引数 = %#v, want %#v", got, test.want)
			}
			stdin := readTestFile(t, stdinPath)
			if string(stdin) != test.stdin {
				t.Fatalf("stdin = %q, want %q", stdin, test.stdin)
			}
			gitEnvironment := readTestFile(t, gitEnvironmentPath)
			if got, want := string(gitEnvironment),
				"/tmp/alternate-index\n/tmp/alternate-objects\n1\n"; got != want {
				t.Fatalf("Git 環境 = %q, want %q", got, want)
			}
			goEnvironment := readTestFile(t, goEnvironmentPath)
			if got, want := string(goEnvironment),
				"off|local|off|-mod=readonly|off|public:git,private:off|C|"+
					caches.goBuild+"|"+caches.golangci+"\n||||\n"; got != want {
				t.Fatalf("Go 環境 = %q, want %q", got, want)
			}
		})
	}
}

func TestManageBootstrapCreatesVerifiedCacheBeforeInstallGoRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell の Git hook 管理 entrypoint を検証するテストです")
	}
	sourceRepository := repositoryRootForTest(t)
	repository := newGitRepository(t)
	common := strings.TrimSpace(
		runGit(t, repository, "rev-parse", "--path-format=absolute", "--git-common-dir"),
	)
	common, err := filepath.EvalSymlinks(common)
	if err != nil {
		t.Fatalf("Git common directory を解決できませんでした: %v", err)
	}
	caches := cachePaths(filepath.Join(common, qualityCacheDirectory))
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git を解決できませんでした: %v", err)
	}
	fakeDirectory := t.TempDir()
	invokedPath := filepath.Join(fakeDirectory, "go-invoked")
	writeExecutable(
		t,
		filepath.Join(fakeDirectory, "go"),
		"#!/bin/sh\n"+
			"[ \"$GOCACHE\" = "+shellQuote(caches.goBuild)+" ] || exit 81\n"+
			"[ \"$GOLANGCI_LINT_CACHE\" = "+shellQuote(caches.golangci)+" ] || exit 82\n"+
			": > "+shellQuote(invokedPath)+"\n",
	)
	ambientCache := filepath.Join(t.TempDir(), "書込不能 cache")
	writeFile(t, filepath.Dir(ambientCache), filepath.Base(ambientCache), "directory ではない\n")

	//nolint:gosec // SOT-ENG-021: リポジトリ内の固定管理 entrypoint を一時 Git fixture から実行する。
	command := exec.CommandContext(
		t.Context(),
		filepath.Join(sourceRepository, ".githooks", "manage"),
		"install",
	)
	command.Dir = repository
	command.Env = append(
		os.Environ(),
		"PATH="+strings.Join(
			[]string{fakeDirectory, filepath.Dir(gitPath), "/usr/local/bin", "/usr/bin", "/bin"},
			string(os.PathListSeparator),
		),
		"GOCACHE="+ambientCache,
		"GOLANGCI_LINT_CACHE="+ambientCache,
	)
	if output, runErr := command.CombinedOutput(); runErr != nil {
		t.Fatalf("安全な共有 cache で管理 entrypoint を起動できませんでした: %v\n%s", runErr, output)
	}
	if _, statErr := os.Stat(invokedPath); statErr != nil {
		t.Fatalf("初回 go run が実行されませんでした: %v", statErr)
	}
	if _, cacheErr := existingHookCachePaths(t.Context(), repository); cacheErr != nil {
		t.Fatalf("entrypoint が安全な cache を作成しませんでした: %v", cacheErr)
	}
}

func TestManageBootstrapRejectsUnsafeCacheBeforeInstallGoRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell の Git hook 管理 entrypoint を検証するテストです")
	}
	sourceRepository := repositoryRootForTest(t)
	repository := newGitRepository(t)
	common := strings.TrimSpace(
		runGit(t, repository, "rev-parse", "--path-format=absolute", "--git-common-dir"),
	)
	cacheRoot := filepath.Join(common, qualityCacheDirectory)
	if err := os.Symlink(t.TempDir(), cacheRoot); err != nil {
		t.Fatalf("unsafe cache symlink を作成できませんでした: %v", err)
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git を解決できませんでした: %v", err)
	}
	fakeDirectory := t.TempDir()
	invokedPath := filepath.Join(fakeDirectory, "go-invoked")
	writeExecutable(
		t,
		filepath.Join(fakeDirectory, "go"),
		"#!/bin/sh\n: > "+shellQuote(invokedPath)+"\n",
	)

	//nolint:gosec // SOT-ENG-021: unsafe cache を持つ一時 fixture で管理 entrypoint を実行する。
	command := exec.CommandContext(
		t.Context(),
		filepath.Join(sourceRepository, ".githooks", "manage"),
		"install",
	)
	command.Dir = repository
	command.Env = append(
		os.Environ(),
		"PATH="+strings.Join(
			[]string{fakeDirectory, filepath.Dir(gitPath), "/usr/local/bin", "/usr/bin", "/bin"},
			string(os.PathListSeparator),
		),
	)
	if output, runErr := command.CombinedOutput(); runErr == nil {
		t.Fatalf("unsafe cache で管理 entrypoint が成功しました: %s", output)
	}
	if _, statErr := os.Stat(invokedPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("cache の安全性検証前に go が実行されました: %v", statErr)
	}
}

func TestHookWrappersBootstrapWithVerifiedCommonDirectoryCache(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell の Git hook wrapper を検証するテストです")
	}
	sourceRepository := repositoryRootForTest(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git を解決できませんでした: %v", err)
	}

	for _, test := range []struct {
		name         string
		objectFormat string
		linked       bool
	}{
		{name: "pre-commit", objectFormat: "sha1"},
		{name: "pre-push", objectFormat: "sha256", linked: true},
	} {
		t.Run(test.objectFormat+"/"+test.name, func(t *testing.T) {
			mainRepository := filepath.Join(t.TempDir(), "main repository")
			if err := os.Mkdir(mainRepository, 0o750); err != nil {
				t.Fatalf("main repository を作成できませんでした: %v", err)
			}
			runGit(t, mainRepository, "init", "--quiet", "--object-format="+test.objectFormat)
			runGit(t, mainRepository, "config", "user.name", "テスト利用者")
			runGit(t, mainRepository, "config", "user.email", "test@example.invalid")
			writeFile(t, mainRepository, "README.md", "cache bootstrap\n")
			commitAll(t, mainRepository, "cache bootstrap")

			repository := mainRepository
			if test.linked {
				repository = filepath.Join(t.TempDir(), "linked worktree")
				runGit(
					t,
					mainRepository,
					"worktree",
					"add",
					"--quiet",
					"-b",
					"cache-bootstrap-test",
					repository,
				)
			}
			caches, cacheErr := prepareHookCachePaths(t.Context(), repository)
			if cacheErr != nil {
				t.Fatalf("共有 cache を準備できませんでした: %v", cacheErr)
			}

			fakeDirectory := t.TempDir()
			invokedPath := filepath.Join(fakeDirectory, "go-invoked")
			writeExecutable(
				t,
				filepath.Join(fakeDirectory, "go"),
				"#!/bin/sh\n"+
					"[ \"$GOCACHE\" = "+shellQuote(caches.goBuild)+" ] || exit 81\n"+
					"[ \"$GOLANGCI_LINT_CACHE\" = "+shellQuote(caches.golangci)+" ] || exit 82\n"+
					": > "+shellQuote(invokedPath)+"\n",
			)
			ambientCache := filepath.Join(t.TempDir(), "書込不能 cache")
			writeFile(t, filepath.Dir(ambientCache), filepath.Base(ambientCache), "directory ではない\n")

			//nolint:gosec // SOT-ENG-021: リポジトリ内の固定 hook を一時 Git fixture から実行する。
			command := exec.CommandContext(
				t.Context(),
				filepath.Join(sourceRepository, ".githooks", test.name),
			)
			command.Dir = repository
			command.Env = append(
				os.Environ(),
				"PATH="+strings.Join(
					[]string{fakeDirectory, filepath.Dir(gitPath), "/usr/local/bin", "/usr/bin", "/bin"},
					string(os.PathListSeparator),
				),
				"GOCACHE="+ambientCache,
				"GOLANGCI_LINT_CACHE="+ambientCache,
			)
			if output, runErr := command.CombinedOutput(); runErr != nil {
				t.Fatalf("安全な共有 cache で wrapper を起動できませんでした: %v\n%s", runErr, output)
			}
			if _, statErr := os.Stat(invokedPath); statErr != nil {
				t.Fatalf("初回 go run が実行されませんでした: %v", statErr)
			}
		})
	}
}

func TestHookWrappersRejectUnsafeCommonDirectoryCacheBeforeGoRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell の Git hook wrapper を検証するテストです")
	}
	sourceRepository := repositoryRootForTest(t)
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("git を解決できませんでした: %v", err)
	}

	for _, test := range []struct {
		name   string
		damage func(*testing.T, hookCachePaths)
	}{
		{
			name: "symlink",
			damage: func(t *testing.T, caches hookCachePaths) {
				t.Helper()
				if err := os.Remove(caches.goBuild); err != nil {
					t.Fatalf("cache directory を削除できませんでした: %v", err)
				}
				if err := os.Symlink(t.TempDir(), caches.goBuild); err != nil {
					t.Fatalf("cache symlink を作成できませんでした: %v", err)
				}
			},
		},
		{
			name: "group-writable",
			damage: func(t *testing.T, caches hookCachePaths) {
				t.Helper()
				//nolint:gosec // SOT-ENG-021: unsafe cache を再現する private fixture に限定する。
				if err := os.Chmod(filepath.Dir(caches.goBuild), 0o770); err != nil {
					t.Fatalf("cache root の権限を変更できませんでした: %v", err)
				}
			},
		},
		{
			name: "group-writable-common-directory",
			damage: func(t *testing.T, caches hookCachePaths) {
				t.Helper()
				commonDirectory := filepath.Dir(filepath.Dir(caches.goBuild))
				//nolint:gosec // SOT-ENG-021: unsafe Git common directory を再現する private fixture に限定する。
				if err := os.Chmod(commonDirectory, 0o770); err != nil {
					t.Fatalf("Git common directory の権限を変更できませんでした: %v", err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := newGitRepository(t)
			caches, cacheErr := prepareHookCachePaths(t.Context(), repository)
			if cacheErr != nil {
				t.Fatalf("共有 cache を準備できませんでした: %v", cacheErr)
			}
			test.damage(t, caches)

			fakeDirectory := t.TempDir()
			invokedPath := filepath.Join(fakeDirectory, "go-invoked")
			writeExecutable(
				t,
				filepath.Join(fakeDirectory, "go"),
				"#!/bin/sh\n: > "+shellQuote(invokedPath)+"\n",
			)
			//nolint:gosec // SOT-ENG-021: リポジトリ内の固定 hook を一時 Git fixture から実行する。
			command := exec.CommandContext(
				t.Context(),
				filepath.Join(sourceRepository, ".githooks", "pre-commit"),
			)
			command.Dir = repository
			command.Env = append(
				os.Environ(),
				"PATH="+strings.Join(
					[]string{fakeDirectory, filepath.Dir(gitPath), "/usr/local/bin", "/usr/bin", "/bin"},
					string(os.PathListSeparator),
				),
			)
			if output, runErr := command.CombinedOutput(); runErr == nil {
				t.Fatalf("安全でない cache で wrapper が成功しました: %s", output)
			}
			if _, statErr := os.Stat(invokedPath); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("cache の安全性検証前に go が実行されました: %v", statErr)
			}
		})
	}
}

func TestControlledGoEnvironmentRemovesHostBuildOverrides(t *testing.T) {
	environment := []string{
		"PATH=/usr/bin",
		"GOOS=windows",
		"GOARCH=386",
		"GOEXPERIMENT=fieldtrack",
		"GO111MODULE=off",
		"CGO_ENABLED=1",
		"GOCACHE=/ambient/go",
		"GOLANGCI_LINT_CACHE=/ambient/lint",
	}
	caches := hookCachePaths{goBuild: "/shared/go", golangci: "/shared/lint"}

	got := environmentWithHookCaches(controlledGoEnvironment(environment, false), caches)

	joined := strings.Join(got, "\n")
	for _, forbidden := range []string{
		"GOOS=",
		"GOARCH=",
		"GOEXPERIMENT=",
		"GO111MODULE=",
		"CGO_ENABLED=",
		"GOCACHE=/ambient",
		"GOLANGCI_LINT_CACHE=/ambient",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("ambient build override が残っています: %s", forbidden)
		}
	}
	for _, expected := range []string{
		"GOCACHE=/shared/go",
		"GOLANGCI_LINT_CACHE=/shared/lint",
		"GOPROXY=off",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("固定環境がありません: %s", expected)
		}
	}
}

func repositoryRootForTest(t *testing.T) string {
	t.Helper()

	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("テストの作業ディレクトリを取得できませんでした: %v", err)
	}
	for {
		module, moduleErr := os.Stat(filepath.Join(directory, "go.mod"))
		hooks, hooksErr := os.Stat(filepath.Join(directory, ".githooks"))
		if moduleErr == nil && module.Mode().IsRegular() &&
			hooksErr == nil && hooks.IsDir() {
			return directory
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("go.mod と .githooks を持つリポジトリルートを取得できませんでした")
		}
		directory = parent
	}
}

func writeExecutable(t *testing.T, target, contents string) {
	t.Helper()
	if err := os.WriteFile(target, []byte(contents), 0o600); err != nil {
		t.Fatalf("実行ファイルを作成できませんでした: %v", err)
	}
	//nolint:gosec // SOT-ENG-021: wrapper テストの private fixture にだけ実行権限を付ける。
	if err := os.Chmod(target, 0o700); err != nil {
		t.Fatalf("テスト fixture を実行可能にできませんでした: %v", err)
	}
}
