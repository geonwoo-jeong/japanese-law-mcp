// legal-query-candidate-eval は、統合照会候補を CI へ handoff する専用 bootstrap である。
//
// 第 4 段階 4 では引数と build context だけを固定し、第 4 段階 5 の閉じた worker
// registry が接続されるまで候補評価を開始しない。
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	fixedRepository      = "."
	fixedOutputDirectory = "./.artifacts/legal-query-candidate-evaluation"

	exitSuccess    = 0
	exitValidation = 1
	exitUsage      = 2
)

var errWorkerNotConnected = errors.New("第4.5段階の閉じた worker registry は未接続です")

type candidateOptions struct {
	Repository      string
	OutputDirectory string
}

type candidateExecutor func(candidateOptions) error

func main() {
	os.Exit(run(
		os.Args[1:],
		os.Environ(),
		os.Stdout,
		os.Stderr,
		preparedHandoff,
	))
}

func run(
	args []string,
	environment []string,
	stdout, stderr io.Writer,
	execute candidateExecutor,
) int {
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
	if err := execute(options); err != nil {
		_, _ = fmt.Fprintf(stderr, "候補評価 bootstrap は準備状態です: %v\n", err)
		return exitValidation
	}
	_, _ = fmt.Fprintln(stdout, "候補評価 handoff が完了しました")
	return exitSuccess
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
	for _, key := range []string{"GOROOT", "GOMODCACHE", "GOCACHE"} {
		value, ok := values[key]
		if !ok || value == "" {
			continue
		}
		if strings.ContainsRune(value, '\x00') || !filepath.IsAbs(value) {
			return fmt.Errorf("%s は絶対 path でなければなりません", key)
		}
	}
	return nil
}

func closedGoEnvironment(environment []string) (map[string]string, error) {
	allowed := map[string]struct{}{
		"GOWORK": {}, "GOENV": {}, "GOTOOLCHAIN": {}, "GOPROXY": {},
		"GOSUMDB": {}, "GOFLAGS": {}, "GOOS": {}, "GOARCH": {},
		"GOAMD64": {}, "GOEXPERIMENT": {}, "CGO_ENABLED": {}, "GOMAXPROCS": {},
		"GOROOT": {}, "GOMODCACHE": {}, "GOCACHE": {},
	}
	result := make(map[string]string, len(allowed))
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		_, relevant := allowed[key]
		if !relevant && !strings.HasPrefix(key, "GO") {
			continue
		}
		if !relevant {
			return nil, fmt.Errorf("許可されていない Go 環境変数 %s があります", key)
		}
		if _, duplicate := result[key]; duplicate {
			return nil, fmt.Errorf("環境変数 %s が重複しています", key)
		}
		result[key] = value
	}
	return result, nil
}

func preparedHandoff(candidateOptions) error {
	return errWorkerNotConnected
}
