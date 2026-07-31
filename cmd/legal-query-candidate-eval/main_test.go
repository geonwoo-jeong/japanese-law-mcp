package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const (
	verificationCIAuthority       = "candidate-evaluation-ci-authority"
	verificationBuildIsolation    = "candidate-evaluation-build-context-isolation"
	verificationProductionBlocked = "candidate-evaluation-production-unreachable"
)

func TestFixedArguments(t *testing.T) {
	t.Parallel()

	want := candidateOptions{
		Repository:      ".",
		OutputDirectory: "./.artifacts/legal-query-candidate-evaluation",
	}
	got, err := parseFixedArguments([]string{
		"--repository=.",
		"--output-directory=./.artifacts/legal-query-candidate-evaluation",
	})
	if err != nil {
		t.Fatalf("%s: 固定引数を受理できません: %v", verificationCIAuthority, err)
	}
	if got != want {
		t.Fatalf("%s: options = %#v, want %#v", verificationCIAuthority, got, want)
	}
}

func TestFixedArgumentsRejectEveryAlternative(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"引数なし": nil,
		"repository の別値": {
			"--repository=/tmp/repository",
			"--output-directory=./.artifacts/legal-query-candidate-evaluation",
		},
		"output の別値": {
			"--repository=.",
			"--output-directory=./result",
		},
		"順序違い": {
			"--output-directory=./.artifacts/legal-query-candidate-evaluation",
			"--repository=.",
		},
		"分離表記": {
			"--repository", ".",
			"--output-directory", "./.artifacts/legal-query-candidate-evaluation",
		},
		"追加引数": {
			"--repository=.",
			"--output-directory=./.artifacts/legal-query-candidate-evaluation",
			"--profile=next",
		},
	}
	for name, args := range cases {
		name, args := name, args
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseFixedArguments(args); err == nil {
				t.Fatalf("%s: 代替引数を受理しました: %#v", verificationCIAuthority, args)
			}
		})
	}
}

func TestFixedBuildEnvironment(t *testing.T) {
	t.Parallel()

	if err := validateBuildEnvironment(fixedEnvironment()); err != nil {
		t.Fatalf("%s: 固定 build 環境を受理できません: %v", verificationBuildIsolation, err)
	}
}

func TestFixedBuildEnvironmentRejectsDriftAndHiddenGoState(t *testing.T) {
	t.Parallel()

	cases := map[string]func([]string) []string{
		"必須値の欠落": func(values []string) []string {
			return withoutEnvironment(values, "GOWORK")
		},
		"必須値の差": func(values []string) []string {
			return replaceEnvironment(values, "GOARCH", "arm64")
		},
		"空値の未設定": func(values []string) []string {
			return withoutEnvironment(values, "GOEXPERIMENT")
		},
		"build tag": func(values []string) []string {
			return replaceEnvironment(values, "GOFLAGS", "-mod=readonly -buildvcs=false -tags=holdout")
		},
		"hidden mode": func(values []string) []string {
			return append(values, "GODEBUG=gocacheverify=1")
		},
		"private proxy": func(values []string) []string {
			return append(values, "GOPRIVATE=example.invalid")
		},
		"重複 key": func(values []string) []string {
			return append(values, "GOOS=linux")
		},
	}
	for name, mutate := range cases {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := validateBuildEnvironment(mutate(fixedEnvironment())); err == nil {
				t.Fatalf("%s: build 環境の差分を受理しました", verificationBuildIsolation)
			}
		})
	}
}

func TestPreparedBoundaryNeverProducesEvaluation(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run(
		[]string{
			"--repository=.",
			"--output-directory=./.artifacts/legal-query-candidate-evaluation",
		},
		fixedEnvironment(),
		&stdout,
		&stderr,
		preparedHandoff,
	)
	if code != exitValidation {
		t.Fatalf("%s: exit code = %d, want %d", verificationProductionBlocked, code, exitValidation)
	}
	if stdout.Len() != 0 {
		t.Fatalf("%s: stdout = %q, want empty", verificationProductionBlocked, stdout.String())
	}
	if !errors.Is(preparedHandoff(candidateOptions{}), errWorkerNotConnected) {
		t.Fatalf("%s: prepared handoff が固定 sentinel を返しません", verificationProductionBlocked)
	}
	for _, forbidden := range []string{"report.json", "result.json", "holdout"} {
		if strings.Contains(stderr.String(), forbidden) {
			t.Fatalf("%s: stderr に非公開情報 %q が含まれます", verificationProductionBlocked, forbidden)
		}
	}
}

func TestRunRejectsBeforeExecutor(t *testing.T) {
	t.Parallel()

	called := false
	executor := func(candidateOptions) error {
		called = true
		return nil
	}
	var stdout, stderr bytes.Buffer
	code := run(
		[]string{"--repository=.", "--output-directory=./invalid"},
		fixedEnvironment(),
		&stdout,
		&stderr,
		executor,
	)
	if code != exitUsage || called {
		t.Fatalf("%s: code=%d called=%t, want %d,false", verificationCIAuthority, code, called, exitUsage)
	}
}

func TestBootstrapImportsOnlyStandardLibrary(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("%s: command directory を列挙できません: %v", verificationProductionBlocked, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), entry.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("%s: %s を parse できません: %v", verificationProductionBlocked, entry.Name(), err)
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("%s: import path を decode できません: %v", verificationProductionBlocked, err)
			}
			if strings.Contains(strings.Split(path, "/")[0], ".") {
				t.Fatalf("%s: bootstrap が標準 library 外を import しています: %s", verificationProductionBlocked, path)
			}
		}
		if len(file.Decls) == 0 && !ast.IsExported(file.Name.Name) {
			t.Fatalf("%s: 空の bootstrap source です", verificationProductionBlocked)
		}
	}
}

func TestCandidateCommandIsNotRegisteredInProduction(t *testing.T) {
	t.Parallel()

	repositoryRoot, err := os.OpenRoot(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("%s: repository root を開けません: %v", verificationProductionBlocked, err)
	}
	t.Cleanup(func() {
		if err := repositoryRoot.Close(); err != nil {
			t.Errorf("%s: repository root を閉じられません: %v", verificationProductionBlocked, err)
		}
	})
	targets := []string{
		"cmd/japanese-law-mcp",
		"internal/cli",
		"internal/mcp",
		"internal/qualitygate",
		".github/workflows/quality.yml",
	}
	for _, target := range targets {
		target := target
		err := fs.WalkDir(repositoryRoot.FS(), target, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			content, err := fs.ReadFile(repositoryRoot.FS(), path)
			if err != nil {
				return err
			}
			if strings.Contains(string(content), "legal-query-candidate-eval") {
				t.Errorf("%s: 製品または通常品質ゲートから候補 command へ到達できます: %s", verificationProductionBlocked, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s: 到達性を検査できません: %v", verificationProductionBlocked, err)
		}
	}
}

func TestManualWorkflowContract(t *testing.T) {
	t.Parallel()

	repositoryRoot, err := os.OpenRoot(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("%s: repository root を開けません: %v", verificationCIAuthority, err)
	}
	t.Cleanup(func() {
		if err := repositoryRoot.Close(); err != nil {
			t.Errorf("%s: repository root を閉じられません: %v", verificationCIAuthority, err)
		}
	})
	content, err := fs.ReadFile(repositoryRoot.FS(), ".github/workflows/candidate-evaluation.yml")
	if err != nil {
		t.Fatalf("%s: workflow を読めません: %v", verificationCIAuthority, err)
	}
	text := string(content)
	required := []string{
		"workflow_dispatch:",
		"preparation_commit:",
		"required: true",
		"permissions:\n  contents: read",
		"actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1",
		"ref: ${{ inputs.preparation_commit }}",
		"persist-credentials: false",
		"${{ github.sha }}",
		"git rev-parse HEAD",
		"env -i",
		"GOWORK=off GOENV=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off",
		"GOFLAGS='-mod=readonly -buildvcs=false'",
		"GOOS=linux GOARCH=amd64 GOAMD64=v1 GOEXPERIMENT= CGO_ENABLED=0 GOMAXPROCS=1",
		"go run ./cmd/legal-query-candidate-eval --repository=. --output-directory=./.artifacts/legal-query-candidate-evaluation",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Errorf("%s: workflow に %q がありません", verificationCIAuthority, fragment)
		}
	}
	for _, forbidden := range []string{
		"\n  push:",
		"\n  pull_request:",
		"\n  schedule:",
		"\n  workflow_call:",
		"secrets.",
		"contents: write",
		"continue-on-error:",
		"GITHUB_TOKEN:",
		"ACTIONS_RUNTIME_TOKEN:",
	} {
		if strings.Contains(text, forbidden) {
			t.Errorf("%s: workflow に禁止設定 %q があります", verificationCIAuthority, forbidden)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "uses:") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			t.Errorf("%s: action 指定を解釈できません: %s", verificationCIAuthority, trimmed)
			continue
		}
		action := fields[1]
		at := strings.LastIndex(action, "@")
		if at < 0 || len(action)-at-1 != 40 {
			t.Errorf("%s: action が完全な commit SHA で固定されていません: %s", verificationCIAuthority, trimmed)
		}
	}
}

func fixedEnvironment() []string {
	return []string{
		"GOWORK=off",
		"GOENV=off",
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOFLAGS=-mod=readonly -buildvcs=false",
		"GOOS=linux",
		"GOARCH=amd64",
		"GOAMD64=v1",
		"GOEXPERIMENT=",
		"CGO_ENABLED=0",
		"GOMAXPROCS=1",
		"GOROOT=/opt/go",
		"GOMODCACHE=/tmp/modcache",
		"GOCACHE=/tmp/buildcache",
	}
}

func withoutEnvironment(values []string, key string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !strings.HasPrefix(value, key+"=") {
			result = append(result, value)
		}
	}
	return result
}

func replaceEnvironment(values []string, key, replacement string) []string {
	result := append([]string(nil), values...)
	for index, value := range result {
		if strings.HasPrefix(value, key+"=") {
			result[index] = key + "=" + replacement
		}
	}
	return result
}
