package main

import (
	"bytes"
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateworker"
)

const (
	verificationCIAuthority        = "candidate-evaluation-ci-authority"
	verificationBuildIsolation     = "candidate-evaluation-build-context-isolation"
	verificationOutcomeExit        = "candidate-evaluation-outcome-exit-semantics"
	verificationOutputPrivacy      = "candidate-evaluation-output-privacy"
	verificationProductionBlocked  = "candidate-evaluation-production-unreachable"
	verificationFailureClosedSet   = "candidate-evaluation-failure-stage-closed-set"
	verificationFailurePropagation = "candidate-evaluation-failure-stage-propagation"
	verificationFailurePrivacy     = "candidate-evaluation-failure-stage-privacy"
	verificationFailureBounded     = "candidate-evaluation-failure-stage-bounded-capture"
	verificationHandoffReadStage   = "candidate-evaluation-handoff-read-stage"
	verificationUnknownFailClosed  = "candidate-evaluation-unknown-fail-closed"
	verificationIndeterminateGate  = "candidate-evaluation-indeterminate-reviewed-retry-gate"
	verificationReadinessNonBypass = "candidate-evaluation-readiness-non-bypass"
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
		"stale bypass": {
			"--repository=.",
			"--output-directory=./.artifacts/legal-query-candidate-evaluation",
			"--allow-stale",
		},
	}
	for name, args := range cases {
		name, args := name, args
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := parseFixedArguments(args); err == nil {
				t.Fatalf("%s/%s: 代替引数を受理しました: %#v", verificationCIAuthority, verificationReadinessNonBypass, args)
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

func TestChildWorkerEnvironmentRejectsDriftAndAmbientState(t *testing.T) {
	t.Parallel()

	cases := map[string]func([]string) []string{
		"必須値の欠落": func(values []string) []string {
			return withoutEnvironment(values, "GOWORK")
		},
		"PATH の欠落": func(values []string) []string {
			return withoutEnvironment(values, "PATH")
		},
		"TMPDIR の欠落": func(values []string) []string {
			return withoutEnvironment(values, "TMPDIR")
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
		"HOME": func(values []string) []string {
			return append(values, "HOME=/home/runner")
		},
		"credential": func(values []string) []string {
			return append(values, "CANDIDATE_TEST_TOKEN=sample-private-token")
		},
		"任意の環境変数": func(values []string) []string {
			return append(values, "LANG=ja_JP.UTF-8")
		},
		"stale bypass": func(values []string) []string {
			return append(values, "JAPANESE_LAW_MCP_ALLOW_STALE=1")
		},
		"不正な環境 entry": func(values []string) []string {
			return append(values, "MALFORMED")
		},
		"重複 key": func(values []string) []string {
			return append(values, "GOOS=linux")
		},
		"相対 PATH": func(values []string) []string {
			return replaceEnvironment(values, "PATH", "/opt/go/bin:bin")
		},
		"相対 GOROOT": func(values []string) []string {
			return replaceEnvironment(values, "GOROOT", "opt/go")
		},
		"相対 GOMODCACHE": func(values []string) []string {
			return replaceEnvironment(values, "GOMODCACHE", "tmp/modcache")
		},
		"相対 GOCACHE": func(values []string) []string {
			return replaceEnvironment(values, "GOCACHE", "tmp/buildcache")
		},
		"相対 TMPDIR": func(values []string) []string {
			return replaceEnvironment(values, "TMPDIR", "tmp/candidate")
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

func TestRunReturnsSuccessForPassedAndFailedHandoffs(t *testing.T) {
	t.Parallel()

	cases := []candidateHandoff{
		{
			EvaluationID: "evaluation-sha256-" + strings.Repeat("a", 64),
			Outcome:      "passed",
			ReportSHA256: strings.Repeat("b", 64),
		},
		{
			EvaluationID: "evaluation-sha256-" + strings.Repeat("c", 64),
			Outcome:      "failed",
			ReportSHA256: strings.Repeat("d", 64),
		},
	}
	for _, handoff := range cases {
		handoff := handoff
		t.Run(handoff.Outcome, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr bytes.Buffer
			code := run(
				context.Background(),
				fixedArguments(),
				fixedEnvironment(),
				&stdout,
				&stderr,
				func(context.Context, candidateOptions) (candidateHandoff, error) {
					return handoff, nil
				},
			)
			if code != exitSuccess {
				t.Fatalf("%s: outcome=%s exit code=%d, want %d", verificationOutcomeExit, handoff.Outcome, code, exitSuccess)
			}
			if stderr.Len() != 0 {
				t.Fatalf("%s: outcome=%s stderr=%q, want empty", verificationOutcomeExit, handoff.Outcome, stderr.String())
			}
		})
	}
}

func TestRunReturnsNonZeroOnlyForInfrastructureFailure(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run(
		context.Background(),
		fixedArguments(),
		fixedEnvironment(),
		&stdout,
		&stderr,
		func(context.Context, candidateOptions) (candidateHandoff, error) {
			return candidateHandoff{}, errors.New("worker infrastructure failure")
		},
	)
	if code == exitSuccess {
		t.Fatalf("%s: infrastructure failure が exit 0 になりました", verificationOutcomeExit)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("%s: failure output=(%q,%q), want empty", verificationOutputPrivacy, stdout.String(), stderr.String())
	}
}

func TestRunPropagatesAllowlistedFailureCode(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	code := run(
		context.Background(),
		fixedArguments(),
		fixedEnvironment(),
		&stdout,
		&stderr,
		func(context.Context, candidateOptions) (candidateHandoff, error) {
			return candidateHandoff{}, workerFailureError{
				code: workerFailureTrackedReplay,
				err:  errors.New("sensitive tracked replay detail"),
			}
		},
	)
	if code != workerFailureTrackedReplay {
		t.Fatalf("%s: code=%d, want %d", verificationOutcomeExit, code, workerFailureTrackedReplay)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("%s: failure output=(%q,%q), want empty", verificationOutputPrivacy, stdout.String(), stderr.String())
	}
}

func TestRunは閉じた全失敗段階を内容なしで伝達する(t *testing.T) {
	t.Parallel()

	for _, wantCode := range []int{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22} {
		wantCode := wantCode
		t.Run(strconv.Itoa(wantCode), func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := run(
				context.Background(),
				fixedArguments(),
				fixedEnvironment(),
				&stdout,
				&stderr,
				func(context.Context, candidateOptions) (candidateHandoff, error) {
					return candidateHandoff{}, workerFailureError{
						code: wantCode,
						err: errors.New(
							"永住許可 case-private /private/path sample-private-token",
						),
					}
				},
			)
			if code != wantCode || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf(
					"%s: code=%d output=(%q,%q), want %d and empty",
					verificationFailurePropagation, code, stdout.String(), stderr.String(), wantCode,
				)
			}
		})
	}
}

func TestRunDoesNotExposeWorkerInputOrInfrastructureValues(t *testing.T) {
	t.Parallel()

	sensitive := []string{
		"永住許可の条件を教えて",
		"/home/runner/private/verified-source",
		"sample-private-token-do-not-use",
	}
	var stdout, stderr bytes.Buffer
	code := run(
		context.Background(),
		fixedArguments(),
		fixedEnvironment(),
		&stdout,
		&stderr,
		func(context.Context, candidateOptions) (candidateHandoff, error) {
			return candidateHandoff{}, errors.New(strings.Join(sensitive, " "))
		},
	)
	if code == exitSuccess {
		t.Fatalf("%s: worker failure が exit 0 になりました", verificationOutputPrivacy)
	}
	output := stdout.String() + stderr.String()
	if output != "" {
		t.Fatalf("%s: failure output=%q, want empty", verificationOutputPrivacy, output)
	}
	for _, forbidden := range sensitive {
		if strings.Contains(output, forbidden) {
			t.Errorf("%s: stdout/stderr に非公開値が含まれます", verificationOutputPrivacy)
		}
	}
}

func TestRunは全失敗境界で出力を空に保つ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		ctx         context.Context
		args        []string
		environment []string
		execute     candidateExecutor
		wantCode    int
	}{
		{
			name: "bootstrap validation", ctx: nil,
			args: fixedArguments(), environment: fixedEnvironment(),
			execute: func(context.Context, candidateOptions) (candidateHandoff, error) {
				return candidateHandoff{}, nil
			},
			wantCode: exitValidation,
		},
		{
			name: "usage", ctx: context.Background(),
			args: []string{"--invalid"}, environment: fixedEnvironment(),
			execute: func(context.Context, candidateOptions) (candidateHandoff, error) {
				return candidateHandoff{}, nil
			},
			wantCode: exitUsage,
		},
		{
			name: "closed environment", ctx: context.Background(),
			args: fixedArguments(), environment: append(fixedEnvironment(), "SECRET=value"),
			execute: func(context.Context, candidateOptions) (candidateHandoff, error) {
				return candidateHandoff{}, nil
			},
			wantCode: exitValidation,
		},
		{
			name: "missing executor", ctx: context.Background(),
			args: fixedArguments(), environment: fixedEnvironment(),
			execute: nil, wantCode: exitValidation,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			code := run(test.ctx, test.args, test.environment, &stdout, &stderr, test.execute)
			if code != test.wantCode || stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf(
					"%s: code=%d output=(%q,%q), want %d and empty",
					verificationFailurePrivacy, code, stdout.String(), stderr.String(), test.wantCode,
				)
			}
		})
	}
}

func TestPreparedHandoffParsesWorkerExitStatusWithoutLeakingRawStderr(t *testing.T) {
	t.Parallel()

	forbiddenSample := "永住許可 /private/path redacted-marker"
	_, err := preparedHandoffWithRunner(
		context.Background(),
		fixedOutputDirectory,
		fixedEnvironment(),
		func(context.Context, []string, []string, io.Writer, io.Writer) (bool, error) {
			_, _ = io.WriteString(io.Discard, "")
			return true, nil
		},
		nil,
	)
	if err == nil {
		t.Fatal("nil reader を受理しました")
	}

	_, err = preparedHandoffWithRunner(
		context.Background(),
		fixedOutputDirectory,
		fixedEnvironment(),
		func(_ context.Context, _ []string, _ []string, _ io.Writer, stderr io.Writer) (bool, error) {
			_, _ = io.WriteString(stderr, forbiddenSample+"\nexit status 18\n")
			return true, errors.New("go run failed")
		},
		func(string) (candidateHandoff, error) {
			t.Fatal("worker failure で handoff reader が呼ばれました")
			return candidateHandoff{}, nil
		},
	)
	var failed workerFailure
	if !errors.As(err, &failed) || failed.FailureExitCode() != workerFailureHandoffWrite {
		t.Fatalf("%s: err=%v, want code %d", verificationOutcomeExit, err, workerFailureHandoffWrite)
	}
	if strings.Contains(err.Error(), forbiddenSample) {
		t.Fatalf("%s: raw stderr secret が error に残りました", verificationOutputPrivacy)
	}
}

func TestPreparedHandoffReaderFailureUsesDedicatedCode(t *testing.T) {
	t.Parallel()

	forbiddenSample := "永住許可 /private/path handoff-read-marker"
	_, err := preparedHandoffWithRunner(
		context.Background(),
		fixedOutputDirectory,
		fixedEnvironment(),
		func(_ context.Context, args []string, _ []string, _ io.Writer, _ io.Writer) (bool, error) {
			if len(args) != 3 || args[2] != "--output-directory="+fixedOutputDirectory {
				t.Fatalf("%s: args=%#v", verificationCIAuthority, args)
			}
			return true, nil
		},
		func(string) (candidateHandoff, error) {
			return candidateHandoff{}, errors.New(forbiddenSample)
		},
	)
	var failed workerFailure
	if !errors.As(err, &failed) || failed.FailureExitCode() != workerFailureHandoffRead {
		t.Fatalf("%s: err=%v, want code %d", verificationHandoffReadStage, err, workerFailureHandoffRead)
	}
	if strings.Contains(err.Error(), forbiddenSample) {
		t.Fatalf("%s: reader detail が error に残りました", verificationOutputPrivacy)
	}
}

func TestPreparedHandoffClassifiesUnknownAndSpawnFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stderr    string
		wantCode  int
		started   bool
		runnerErr error
	}{
		{
			name:      "非許可コードはunknown",
			stderr:    "exit status 99\n",
			wantCode:  workerFailureUnknown,
			started:   true,
			runnerErr: errors.New("failed"),
		},
		{
			name:      "開始後の終了コードなしはunknown",
			stderr:    "fork/exec go: no such file or directory\n",
			wantCode:  workerFailureUnknown,
			started:   true,
			runnerErr: errors.New("failed"),
		},
		{
			name:      "go run の exit status 1 はunknown",
			stderr:    "exit status 1\n",
			wantCode:  workerFailureUnknown,
			started:   true,
			runnerErr: errors.New("failed"),
		},
		{
			name:      "最後の非空行だけを使う",
			stderr:    "exit status 18\nexit status 99\n",
			wantCode:  workerFailureUnknown,
			started:   true,
			runnerErr: errors.New("failed"),
		},
		{
			name:      "末尾が非終了行ならunknown",
			stderr:    "exit status 18\ngeneric worker line\n",
			wantCode:  workerFailureUnknown,
			started:   true,
			runnerErr: errors.New("failed"),
		},
		{
			name:      "前後空白を正規化しない",
			stderr:    " exit status 18 \n",
			wantCode:  workerFailureUnknown,
			started:   true,
			runnerErr: errors.New("failed"),
		},
		{
			name:      "長い数値を解釈しない",
			stderr:    "exit status 999999999999999999999999999999\n",
			wantCode:  workerFailureUnknown,
			started:   true,
			runnerErr: errors.New("failed"),
		},
		{
			name:      "開始前の失敗だけはworker_start",
			wantCode:  workerFailureSpawn,
			started:   false,
			runnerErr: errors.New("failed to start"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := preparedHandoffWithRunner(
				context.Background(),
				fixedOutputDirectory,
				fixedEnvironment(),
				func(_ context.Context, _ []string, _ []string, _ io.Writer, stderr io.Writer) (bool, error) {
					_, _ = io.WriteString(stderr, test.stderr)
					return test.started, test.runnerErr
				},
				func(string) (candidateHandoff, error) {
					t.Fatal("failure で handoff reader が呼ばれました")
					return candidateHandoff{}, nil
				},
			)
			var failed workerFailure
			if !errors.As(err, &failed) || failed.FailureExitCode() != test.wantCode {
				t.Fatalf("%s: err=%v, want code %d", verificationUnknownFailClosed, err, test.wantCode)
			}
		})
	}
}

func TestTailBufferKeepsOnlyBoundedSuffix(t *testing.T) {
	t.Parallel()

	var buffer tailBuffer
	buffer.max = 8
	if _, err := buffer.Write([]byte("12345")); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if _, err := buffer.Write([]byte("67890")); err != nil {
		t.Fatalf("second write failed: %v", err)
	}
	if got := buffer.String(); got != "34567890" {
		t.Fatalf("%s: tail=%q, want %q", verificationFailureBounded, got, "34567890")
	}
	var production tailBuffer
	production.max = maximumWorkerStderr
	payload := strings.Repeat("x", maximumWorkerStderr+1)
	if _, err := production.Write([]byte(payload)); err != nil {
		t.Fatalf("%s: production write failed: %v", verificationFailureBounded, err)
	}
	if len(production.String()) != 4096 || production.String() != payload[1:] {
		t.Fatalf("%s: production tail size=%d", verificationFailureBounded, len(production.String()))
	}
}

func TestWorkerFailureCodeTableMatchesInternalWorker(t *testing.T) {
	t.Parallel()

	want := []int{
		legalquerycandidateworker.FailureCodePreparedLoad,
		legalquerycandidateworker.FailureCodeRequestBinding,
		legalquerycandidateworker.FailureCodeEvaluateBuild,
		legalquerycandidateworker.FailureCodeReportBinding,
		legalquerycandidateworker.FailureCodeAccept,
		legalquerycandidateworker.FailureCodeResultBind,
		legalquerycandidateworker.FailureCodeResultEncode,
		legalquerycandidateworker.FailureCodeResultDecode,
		legalquerycandidateworker.FailureCodeHandoffWrite,
		legalquerycandidateworker.FailureCodeTrackedReplay,
		legalquerycandidateworker.FailureCodeUnknown,
	}
	got := []int{
		workerFailurePreparedLoad,
		workerFailureRequestBinding,
		workerFailureEvaluateBuild,
		workerFailureReportBinding,
		workerFailureAccept,
		workerFailureResultBind,
		workerFailureResultEncode,
		workerFailureResultDecode,
		workerFailureHandoffWrite,
		workerFailureTrackedReplay,
		workerFailureUnknown,
	}
	if len(got) != len(want) {
		t.Fatalf("%s: failure code count=%d, want %d", verificationFailureClosedSet, len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("%s: failure code[%d]=%d, want %d", verificationFailurePropagation, index, got[index], want[index])
		}
	}
	if workerFailureHandoffRead != 20 || workerFailureSpawn != 21 {
		t.Fatalf("%s: bootstrap failure codes=(%d,%d)", verificationFailureClosedSet, workerFailureHandoffRead, workerFailureSpawn)
	}
}

func TestRunRejectsBeforeExecutor(t *testing.T) {
	t.Parallel()

	called := false
	executor := func(context.Context, candidateOptions) (candidateHandoff, error) {
		called = true
		return candidateHandoff{}, nil
	}
	var stdout, stderr bytes.Buffer
	code := run(
		context.Background(),
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
		"if [[ \"$PREPARATION_COMMIT\" != \"$WORKFLOW_COMMIT\" ]]",
		"git rev-parse HEAD",
		"env -i",
		"GOWORK=off GOENV=off GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off",
		"GOFLAGS='-mod=readonly -buildvcs=false'",
		"GOOS=linux GOARCH=amd64 GOAMD64=v1 GOEXPERIMENT= CGO_ENABLED=0 GOMAXPROCS=1",
		"go run ./cmd/legal-query-candidate-eval --repository=. --output-directory=./.artifacts/legal-query-candidate-evaluation",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Errorf("%s: workflow に %q がありません", verificationIndeterminateGate, fragment)
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

	moduleStepName := "- name: 固定 module archive を準備する"
	workerStepName := "- name: 閉じた候補評価 command を実行する"
	moduleStep := workflowStep(text, moduleStepName)
	workerStep := workflowStep(text, workerStepName)
	if moduleStep == "" || workerStep == "" ||
		strings.Index(text, moduleStepName) >= strings.Index(text, workerStepName) {
		t.Errorf("%s: module 準備が閉じた worker 実行より前にありません", verificationBuildIsolation)
	}
	if !strings.Contains(moduleStep, "chmod -R a-w") {
		t.Errorf("%s: module 準備物を worker 実行前に read-only にしていません", verificationBuildIsolation)
	}
	for _, fragment := range []string{
		"env -i",
		"PATH=\"$PATH\"",
		"GOROOT=\"$CANDIDATE_GOROOT\"",
		"GOMODCACHE=\"$CANDIDATE_GOMODCACHE\"",
		"GOCACHE=\"$CANDIDATE_GOCACHE\"",
		"TMPDIR=\"$CANDIDATE_TMPDIR\"",
		"GOPROXY=off",
	} {
		if !strings.Contains(workerStep, fragment) {
			t.Errorf("%s: 閉じた worker step に %q がありません", verificationBuildIsolation, fragment)
		}
	}

	uploadStep := workflowStep(text, "- name: 候補評価 handoff を保存する")
	exactUploadPaths := "path: |\n" +
		"            ./.artifacts/legal-query-candidate-evaluation/*/report.json\n" +
		"            ./.artifacts/legal-query-candidate-evaluation/*/result.json"
	if !strings.Contains(uploadStep, exactUploadPaths) {
		t.Errorf("%s: workflow artifact は同じ評価 directory の report/result 二 file だけを upload しなければなりません", verificationCIAuthority)
	}
}

func fixedArguments() []string {
	return []string{
		"--repository=.",
		"--output-directory=./.artifacts/legal-query-candidate-evaluation",
	}
}

func fixedEnvironment() []string {
	return []string{
		"PATH=/opt/go/bin:/usr/bin",
		"GOROOT=/opt/go",
		"GOMODCACHE=/tmp/modcache",
		"GOCACHE=/tmp/buildcache",
		"TMPDIR=/tmp/candidate",
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
	}
}

func workflowStep(workflow, name string) string {
	start := strings.Index(workflow, name)
	if start < 0 {
		return ""
	}
	rest := workflow[start:]
	next := strings.Index(rest[len(name):], "\n      - name:")
	if next < 0 {
		return rest
	}
	return rest[:len(name)+next]
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
