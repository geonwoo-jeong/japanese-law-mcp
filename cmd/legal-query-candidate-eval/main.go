// legal-query-candidate-eval は、統合照会候補を CI へ handoff する専用 bootstrap である。
//
// 第 4 段階 5 では引数と環境を固定し、候補評価 worker の公開可能な handoff だけを返す。
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	fixedRepository      = "."
	fixedOutputDirectory = "./.artifacts/legal-query-candidate-evaluation"
	maximumWorkerStderr  = 4096

	exitSuccess    = 0
	exitValidation = 1
	exitUsage      = 2
)

const (
	workerFailurePreparedLoad   = 10
	workerFailureRequestBinding = 11
	workerFailureEvaluateBuild  = 12
	workerFailureReportBinding  = 13
	workerFailureAccept         = 14
	workerFailureResultBind     = 15
	workerFailureResultEncode   = 16
	workerFailureResultDecode   = 17
	workerFailureHandoffWrite   = 18
	workerFailureTrackedReplay  = 19
	workerFailureSpawn          = 20
	workerFailureUnknown        = 21
)

type candidateOptions struct {
	Repository      string
	OutputDirectory string
}

type candidateHandoff struct {
	EvaluationID string
	Outcome      string
	ReportSHA256 string
}

type candidateExecutor func(context.Context, candidateOptions) (candidateHandoff, error)

type workerFailure interface {
	FailureExitCode() int
}

type workerFailureError struct {
	code int
	err  error
}

func (e workerFailureError) Error() string {
	return "候補評価 worker の失敗段階を特定しました"
}

func (e workerFailureError) Unwrap() error {
	return e.err
}

func (e workerFailureError) FailureExitCode() int {
	return e.code
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(
		ctx,
		os.Args[1:],
		os.Environ(),
		os.Stdout,
		os.Stderr,
		preparedHandoff,
	)
	stop()
	os.Exit(code)
}

func run(
	ctx context.Context,
	args []string,
	environment []string,
	stdout, stderr io.Writer,
	execute candidateExecutor,
) int {
	if ctx == nil || ctx.Err() != nil {
		_, _ = fmt.Fprintln(stderr, "候補評価 bootstrap の context が不正です")
		return exitValidation
	}
	options, err := parseFixedArguments(args)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "候補評価 bootstrap の引数が不正です: %v\n", err)
		return exitUsage
	}
	if err := validateBuildEnvironment(environment); err != nil {
		_, _ = fmt.Fprintf(stderr, "候補評価 bootstrap の build 環境が不正です: %v\n", err)
		return exitValidation
	}
	if execute == nil {
		_, _ = fmt.Fprintln(stderr, "候補評価 bootstrap の実行境界がありません")
		return exitValidation
	}
	handoff, err := execute(ctx, options)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "候補評価 worker を正常に完了できませんでした")
		return failureExitCode(err)
	}
	if err := validateCandidateHandoff(handoff); err != nil {
		_, _ = fmt.Fprintln(stderr, "候補評価 worker の handoff が不正です")
		return exitValidation
	}
	_, _ = fmt.Fprintf(
		stdout,
		"evaluationId=%s outcome=%s reportSha256=%s\n",
		handoff.EvaluationID,
		handoff.Outcome,
		handoff.ReportSHA256,
	)
	return exitSuccess
}

func failureExitCode(err error) int {
	var failed workerFailure
	if errors.As(err, &failed) {
		code := failed.FailureExitCode()
		if isWorkerFailureCode(code) {
			return code
		}
	}
	return exitValidation
}

func validateCandidateHandoff(handoff candidateHandoff) error {
	if !strings.HasPrefix(handoff.EvaluationID, "evaluation-sha256-") ||
		len(handoff.EvaluationID) != len("evaluation-sha256-")+64 ||
		!lowerHex(handoff.EvaluationID[len("evaluation-sha256-"):]) {
		return fmt.Errorf("evaluationId が不正です")
	}
	if handoff.Outcome != "passed" && handoff.Outcome != "failed" {
		return fmt.Errorf("outcome が不正です")
	}
	if len(handoff.ReportSHA256) != 64 || !lowerHex(handoff.ReportSHA256) {
		return fmt.Errorf("reportSha256 が不正です")
	}
	return nil
}

func lowerHex(value string) bool {
	for _, character := range value {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
}

func parseFixedArguments(args []string) (candidateOptions, error) {
	want := []string{
		"--repository=" + fixedRepository,
		"--output-directory=" + fixedOutputDirectory,
	}
	if len(args) != len(want) || args[0] != want[0] || args[1] != want[1] {
		return candidateOptions{}, fmt.Errorf(
			"--repository=%s と --output-directory=%s をこの順で一回ずつ指定してください",
			fixedRepository,
			fixedOutputDirectory,
		)
	}
	return candidateOptions{
		Repository:      fixedRepository,
		OutputDirectory: fixedOutputDirectory,
	}, nil
}

func validateBuildEnvironment(environment []string) error {
	values, err := closedGoEnvironment(environment)
	if err != nil {
		return err
	}
	want := []struct {
		key   string
		value string
	}{
		{key: "GOWORK", value: "off"},
		{key: "GOENV", value: "off"},
		{key: "GOTOOLCHAIN", value: "local"},
		{key: "GOPROXY", value: "off"},
		{key: "GOSUMDB", value: "off"},
		{key: "GOFLAGS", value: "-mod=readonly -buildvcs=false"},
		{key: "GOOS", value: "linux"},
		{key: "GOARCH", value: "amd64"},
		{key: "GOAMD64", value: "v1"},
		{key: "GOEXPERIMENT", value: ""},
		{key: "CGO_ENABLED", value: "0"},
		{key: "GOMAXPROCS", value: "1"},
	}
	for _, expected := range want {
		actual, ok := values[expected.key]
		if !ok {
			return fmt.Errorf("%s が設定されていません", expected.key)
		}
		if actual != expected.value {
			return fmt.Errorf("%s は固定値 %q と一致しません", expected.key, expected.value)
		}
	}
	for _, key := range []string{"GOROOT", "GOMODCACHE", "GOCACHE", "TMPDIR"} {
		value, ok := values[key]
		if !ok || value == "" {
			return fmt.Errorf("%s が設定されていません", key)
		}
		if strings.ContainsRune(value, '\x00') || !filepath.IsAbs(value) {
			return fmt.Errorf("%s は絶対 path でなければなりません", key)
		}
	}
	pathValue, ok := values["PATH"]
	if !ok || pathValue == "" {
		return fmt.Errorf("PATH が設定されていません")
	}
	for _, entry := range filepath.SplitList(pathValue) {
		if entry == "" || strings.ContainsRune(entry, '\x00') || !filepath.IsAbs(entry) {
			return fmt.Errorf("PATH の各要素は絶対 path でなければなりません")
		}
	}
	return nil
}

func closedGoEnvironment(environment []string) (map[string]string, error) {
	allowed := map[string]struct{}{
		"GOWORK": {}, "GOENV": {}, "GOTOOLCHAIN": {}, "GOPROXY": {},
		"GOSUMDB": {}, "GOFLAGS": {}, "GOOS": {}, "GOARCH": {},
		"GOAMD64": {}, "GOEXPERIMENT": {}, "CGO_ENABLED": {}, "GOMAXPROCS": {},
		"PATH": {}, "GOROOT": {}, "GOMODCACHE": {}, "GOCACHE": {}, "TMPDIR": {},
	}
	result := make(map[string]string, len(allowed))
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			return nil, fmt.Errorf("形式が不正な環境変数があります")
		}
		_, relevant := allowed[key]
		if !relevant {
			return nil, fmt.Errorf("許可されていない環境変数 %s があります", key)
		}
		if _, duplicate := result[key]; duplicate {
			return nil, fmt.Errorf("環境変数 %s が重複しています", key)
		}
		result[key] = value
	}
	return result, nil
}

type commandRunner func(context.Context, []string, []string, io.Writer, io.Writer) error

func preparedHandoff(ctx context.Context, _ candidateOptions) (candidateHandoff, error) {
	return preparedHandoffWithRunner(
		ctx,
		fixedOutputDirectory,
		append([]string(nil), os.Environ()...),
		runCandidateWorker,
		readCandidateHandoff,
	)
}

func preparedHandoffWithRunner(
	ctx context.Context,
	outputDirectory string,
	environment []string,
	runner commandRunner,
	reader func(string) (candidateHandoff, error),
) (candidateHandoff, error) {
	ctx, cancel := context.WithTimeout(ctx, 25*time.Minute)
	defer cancel()
	if runner == nil {
		return candidateHandoff{}, fmt.Errorf("candidate worker の実行境界がありません")
	}
	if reader == nil {
		return candidateHandoff{}, fmt.Errorf("candidate handoff reader がありません")
	}
	var stderr tailBuffer
	stderr.max = maximumWorkerStderr
	err := runner(
		ctx,
		[]string{
			"./cmd/legal-query-candidate-worker",
			"--repository=" + fixedRepository,
			"--output-directory=" + fixedOutputDirectory,
		},
		append([]string(nil), environment...),
		io.Discard,
		&stderr,
	)
	if err != nil {
		return candidateHandoff{}, classifyWorkerFailure(err, stderr.String())
	}
	return reader(outputDirectory)
}

func runCandidateWorker(
	ctx context.Context,
	args []string,
	environment []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	commandArgs := append([]string{"run"}, args...)
	//nolint:gosec // SOT-ENG-038: 実行対象・引数・環境は固定 command と閉じた build 環境だけを受理する。
	command := exec.CommandContext(ctx, "go", commandArgs...)
	command.Dir = fixedRepository
	command.Env = append([]string(nil), environment...)
	command.Stdin = strings.NewReader("")
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

func classifyWorkerFailure(err error, stderr string) error {
	if code, ok := parseWorkerExitStatus(stderr); ok {
		return workerFailureError{code: code, err: err}
	}
	return workerFailureError{code: workerFailureSpawn, err: err}
}

func parseWorkerExitStatus(stderr string) (int, bool) {
	lines := strings.Split(stderr, "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" {
			continue
		}
		const prefix = "exit status "
		if !strings.HasPrefix(line, prefix) {
			return 0, false
		}
		value := line[len(prefix):]
		code := 0
		for _, character := range value {
			if character < '0' || character > '9' {
				return workerFailureUnknown, true
			}
			code = code*10 + int(character-'0')
		}
		if code == exitValidation {
			return workerFailureSpawn, true
		}
		if isWorkerFailureCode(code) {
			return code, true
		}
		return workerFailureUnknown, true
	}
	return 0, false
}

func isWorkerFailureCode(code int) bool {
	return code >= workerFailurePreparedLoad && code <= workerFailureUnknown
}

type tailBuffer struct {
	max  int
	data []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	if b.max <= 0 {
		return len(p), nil
	}
	if len(p) >= b.max {
		b.data = append(b.data[:0], p[len(p)-b.max:]...)
		return len(p), nil
	}
	total := len(b.data) + len(p)
	if total > b.max {
		drop := total - b.max
		b.data = append(b.data[:0], b.data[drop:]...)
	}
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *tailBuffer) String() string {
	return string(b.data)
}

type workerResultDocument struct {
	ArtifactKind  string `json:"artifactKind"`
	SchemaVersion int    `json:"schemaVersion"`
	EvaluationID  string `json:"evaluationId"`
	RequestSHA256 string `json:"requestSha256"`
	Outcome       string `json:"outcome"`
	ReportSHA256  string `json:"reportSha256"`
}

func readCandidateHandoff(outputRoot string) (candidateHandoff, error) {
	rootInfo, err := os.Lstat(outputRoot)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return candidateHandoff{}, fmt.Errorf("candidate output root が不正です")
	}
	entries, err := os.ReadDir(outputRoot)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() || entries[0].Type()&os.ModeSymlink != 0 {
		return candidateHandoff{}, fmt.Errorf("candidate output entry が不正です")
	}
	evaluationID := entries[0].Name()
	evaluationRoot := filepath.Join(outputRoot, evaluationID)
	children, err := os.ReadDir(evaluationRoot)
	if err != nil || len(children) != 2 ||
		children[0].Name() != "report.json" || children[1].Name() != "result.json" {
		return candidateHandoff{}, fmt.Errorf("candidate handoff file 集合が不正です")
	}
	for _, child := range children {
		info, infoErr := child.Info()
		if infoErr != nil || child.Type()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return candidateHandoff{}, fmt.Errorf("candidate handoff file が不正です")
		}
	}

	resultRaw, err := readBoundedFile(filepath.Join(evaluationRoot, "result.json"), 256<<10)
	if err != nil {
		return candidateHandoff{}, err
	}
	var result workerResultDocument
	decoder := json.NewDecoder(strings.NewReader(string(resultRaw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return candidateHandoff{}, fmt.Errorf("candidate result を decode できません")
	}
	reportRaw, err := readBoundedFile(filepath.Join(evaluationRoot, "report.json"), 4<<20)
	if err != nil {
		return candidateHandoff{}, err
	}
	reportDigest := sha256.Sum256(reportRaw)
	handoff := candidateHandoff{
		EvaluationID: result.EvaluationID,
		Outcome:      result.Outcome,
		ReportSHA256: result.ReportSHA256,
	}
	if result.ArtifactKind != "legal_query_candidate_evaluation_result" ||
		result.SchemaVersion != 2 || result.EvaluationID != evaluationID ||
		len(result.RequestSHA256) != 64 || !lowerHex(result.RequestSHA256) ||
		result.ReportSHA256 != hex.EncodeToString(reportDigest[:]) {
		return candidateHandoff{}, fmt.Errorf("candidate result binding が不正です")
	}
	if err := validateCandidateHandoff(handoff); err != nil {
		return candidateHandoff{}, err
	}
	return handoff, nil
}

func readBoundedFile(path string, maximum int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Size() < 1 || info.Size() > maximum {
		return nil, fmt.Errorf("candidate handoff file の size が不正です")
	}
	//nolint:gosec // SOT-ENG-038: Lstat 済みの固定 handoff subtree 内通常 file だけを読む。
	raw, err := os.ReadFile(path)
	if err != nil || int64(len(raw)) != info.Size() {
		return nil, fmt.Errorf("candidate handoff file を読めません")
	}
	return raw, nil
}
