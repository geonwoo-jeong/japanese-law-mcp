package qualitygate

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGoEnvironmentOverridesAmbientMetadata(t *testing.T) {
	t.Parallel()

	ambient := []string{
		"PATH=/usr/bin",
		"GOWORK=/tmp/evil.work",
		"GOENV=/tmp/evil.env",
		"GOTOOLCHAIN=auto",
		"GOPROXY=https://example.invalid",
		"GOSUMDB=off",
		"GOPRIVATE=example.invalid",
		"GONOPROXY=example.invalid",
		"GONOSUMDB=example.invalid",
		"GOINSECURE=example.invalid",
		"GOVCS=*:all",
		"GOFLAGS=-mod=mod",
		"GOCACHE=/tmp/ambient-go-cache",
		"GOLANGCI_LINT_CACHE=/tmp/ambient-lint-cache",
	}
	caches := cachePaths{
		goBuild:  "/git/common/japanese-law-mcp-quality-cache/go-build",
		golangci: "/git/common/japanese-law-mcp-quality-cache/golangci",
	}

	preCommit := environmentMap(goEnvironment(ambient, false, "-mod=readonly", caches))
	wantPreCommit := map[string]string{
		"PATH":                "/usr/bin",
		"GOWORK":              "off",
		"GOENV":               "off",
		"GOTOOLCHAIN":         "local",
		"GOPROXY":             "off",
		"GOSUMDB":             "sum.golang.org",
		"GOPRIVATE":           "",
		"GONOPROXY":           "",
		"GONOSUMDB":           "",
		"GOINSECURE":          "",
		"GOVCS":               "public:git,private:off",
		"GOFLAGS":             "-mod=readonly",
		"GOCACHE":             caches.goBuild,
		"GOLANGCI_LINT_CACHE": caches.golangci,
	}
	if !reflect.DeepEqual(preCommit, wantPreCommit) {
		t.Fatalf("pre-commit 環境が一致しません:\n got: %#v\nwant: %#v", preCommit, wantPreCommit)
	}

	networked := environmentMap(goEnvironment(
		ambient,
		true,
		"-mod=readonly -modfile=tools/go.mod",
		caches,
	))
	if got, want := networked["GOPROXY"], "https://proxy.golang.org"; got != want {
		t.Fatalf("GOPROXY = %q, want %q", got, want)
	}
	if got, want := networked["GOFLAGS"], "-mod=readonly -modfile=tools/go.mod"; got != want {
		t.Fatalf("GOFLAGS = %q, want %q", got, want)
	}
}

func TestGitEnvironmentPreservesOnlyExplicitCommitIndex(t *testing.T) {
	t.Parallel()

	ambient := []string{
		"PATH=/usr/bin",
		"GIT_DIR=/tmp/wrong.git",
		"GIT_WORK_TREE=/tmp/wrong-worktree",
		"GIT_INDEX_FILE=/tmp/commit-index",
		"GIT_OBJECT_DIRECTORY=/tmp/objects",
		"GIT_ALTERNATE_OBJECT_DIRECTORIES=/tmp/alternate",
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=core.hooksPath",
		"GIT_CONFIG_VALUE_0=/tmp/hooks",
		"GIT_NO_REPLACE_OBJECTS=0",
	}
	withoutIndex := environmentMap(gitEnvironment(ambient, false, false, true))
	if _, exists := withoutIndex["GIT_INDEX_FILE"]; exists {
		t.Fatal("pre-push/CI 環境に ambient GIT_INDEX_FILE が残っています")
	}
	for _, forbidden := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_CONFIG_COUNT"} {
		if _, exists := withoutIndex[forbidden]; exists {
			t.Fatalf("%s が残っています", forbidden)
		}
	}
	if withoutIndex["GIT_TERMINAL_PROMPT"] != "0" ||
		withoutIndex["GIT_CONFIG_NOSYSTEM"] != "1" ||
		withoutIndex["GIT_NO_REPLACE_OBJECTS"] != "1" {
		t.Fatalf("決定的な Git 環境が不足しています: %#v", withoutIndex)
	}

	withIndex := environmentMap(gitEnvironment(ambient, true, true, true))
	if got, want := withIndex["GIT_INDEX_FILE"], "/tmp/commit-index"; got != want {
		t.Fatalf("GIT_INDEX_FILE = %q, want %q", got, want)
	}
	if got, want := withIndex["GIT_OBJECT_DIRECTORY"], "/tmp/objects"; got != want {
		t.Fatalf("GIT_OBJECT_DIRECTORY = %q, want %q", got, want)
	}
	if got, want := withIndex["GIT_ALTERNATE_OBJECT_DIRECTORIES"], "/tmp/alternate"; got != want {
		t.Fatalf("GIT_ALTERNATE_OBJECT_DIRECTORIES = %q, want %q", got, want)
	}
}

func TestExecuteStepsStopsAtFirstFailure(t *testing.T) {
	t.Parallel()

	executor := &recordingExecutor{
		results: []executionResult{
			{stdout: []byte("一件目\n")},
			{stderr: []byte("二件目の失敗\n"), err: errors.New("終了コード 1")},
			{stdout: []byte("実行してはいけません\n")},
		},
	}
	steps := []step{
		commandStep("first", "一件目", "SOT-ENG-020", commandSpec{path: "first"}),
		commandStep("second", "二件目", "SOT-ENG-020", commandSpec{path: "second"}),
		commandStep("third", "三件目", "SOT-ENG-020", commandSpec{path: "third"}),
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := executeSteps(steps, executor, &stdout, &stderr)
	if err == nil {
		t.Fatal("失敗が返されませんでした")
	}
	if got, want := executor.paths, []string{"first", "second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fail-fast ではありません: got %q, want %q", got, want)
	}
	if !strings.Contains(stdout.String(), "一件目") || !strings.Contains(stdout.String(), "成功") {
		t.Fatalf("成功時の日本語出力が不足しています: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "失敗") || !strings.Contains(stderr.String(), "SOT-ENG-020") {
		t.Fatalf("失敗時の日本語出力が不足しています: %q", stderr.String())
	}
}

func TestOSCommandExecutorAppliesGoEnvironment(t *testing.T) {
	t.Parallel()

	caches := cachePaths{
		goBuild:  filepath.Join(t.TempDir(), "go-build"),
		golangci: filepath.Join(t.TempDir(), "golangci"),
	}
	executor := osCommandExecutor{
		context: context.Background(),
		environment: []string{
			"PATH=" + os.Getenv("PATH"),
			"GOWORK=/tmp/ambient.work",
		},
		caches: caches,
	}
	stdout, stderr, err := executor.run(commandSpec{
		path:      "go",
		args:      []string{"env", "GOWORK"},
		dir:       t.TempDir(),
		goCommand: true,
		network:   false,
		goFlags:   readonlyGoFlags,
	})
	if err != nil {
		t.Fatalf("Go コマンドを実行できません: %v, stderr=%q", err, stderr)
	}
	if got := strings.TrimSpace(string(stdout)); got != "off" {
		t.Fatalf("GOWORK = %q, want off", got)
	}
}

func TestExecuteStepsHandlesInternalAndInvalidSteps(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	if err := executeSteps(
		[]step{internalStep("internal", "内部検査", "SOT-ENG-020", func() error { return nil })},
		&recordingExecutor{},
		&stdout,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("内部検査が失敗しました: %v", err)
	}
	if !strings.Contains(stdout.String(), "成功") {
		t.Fatalf("成功メッセージがありません: %q", stdout.String())
	}

	var stderr bytes.Buffer
	err := executeSteps(
		[]step{{key: "invalid", name: "不正な検査", sotID: "SOT-ENG-020"}},
		&recordingExecutor{},
		nil,
		&stderr,
	)
	if err == nil || !strings.Contains(stderr.String(), "実行内容がありません") {
		t.Fatalf("不正な検査を拒否していません: %v, stderr=%q", err, stderr.String())
	}
}

func TestExecuteStepsAddsOfflineCacheGuidance(t *testing.T) {
	t.Parallel()

	executor := &recordingExecutor{
		results: []executionResult{{
			stderr: []byte("module lookup disabled by GOPROXY=off"),
			err:    errors.New("exit status 1"),
		}},
	}
	var stderr bytes.Buffer
	err := executeSteps(
		[]step{commandStep(
			"offline",
			"オフライン検査",
			"SOT-ENG-020",
			goCommand("/snapshot", false, "test", "./..."),
		)},
		executor,
		&bytes.Buffer{},
		&stderr,
	)
	if err == nil || !strings.Contains(err.Error(), ".githooks/manage install") {
		t.Fatalf("オフラインキャッシュの案内がありません: %v", err)
	}
	if strings.Contains(err.Error(), "pre-push を一度実行") {
		t.Fatalf("オフラインの pre-push を準備手順として案内しています: %v", err)
	}

	genericExecutor := &recordingExecutor{
		results: []executionResult{{
			stderr: []byte("テストが失敗しました"),
			err:    errors.New("exit status 1"),
		}},
	}
	err = executeSteps(
		[]step{commandStep(
			"offline",
			"オフライン検査",
			"SOT-ENG-020",
			goCommand("/snapshot", false, "test", "./..."),
		)},
		genericExecutor,
		&bytes.Buffer{},
		&bytes.Buffer{},
	)
	if err == nil || strings.Contains(err.Error(), ".githooks/manage install") {
		t.Fatalf("一般的な Go エラーに依存解決の案内が付きました: %v", err)
	}
}
