package qualitygate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type osCommandExecutor struct {
	context     context.Context
	environment []string
	caches      cachePaths
}

func (e osCommandExecutor) run(spec commandSpec) ([]byte, []byte, error) {
	command := exec.CommandContext( //nolint:gosec // SOT-ENG-020: 実行ファイルと引数は固定した argv として渡す。
		e.context,
		spec.path,
		spec.args...,
	)
	command.Dir = spec.dir
	environment := gitEnvironment(
		e.environment,
		spec.preserveGitIndex,
		spec.preserveGitObjects,
		spec.isolateGitConfig,
	)
	if spec.goCommand {
		environment = goEnvironment(environment, spec.network, spec.goFlags, e.caches)
	}
	command.Env = environment

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func executeSteps(
	steps []step,
	executor commandExecutor,
	stdout io.Writer,
	stderr io.Writer,
) error {
	stdout = normalizeWriter(stdout)
	stderr = normalizeWriter(stderr)

	for _, current := range steps {
		_, _ = fmt.Fprintf(stdout, "開始: %s [%s]\n", current.name, current.sotID)
		if current.check != nil {
			if err := current.check(); err != nil {
				return reportStepFailure(current, err, stderr)
			}
			_, _ = fmt.Fprintf(stdout, "成功: %s [%s]\n", current.name, current.sotID)
			continue
		}
		if current.command == nil {
			return reportStepFailure(current, fmt.Errorf("実行内容がありません"), stderr)
		}

		commandStdout, commandStderr, err := executor.run(*current.command)
		_, _ = stdout.Write(commandStdout)
		_, _ = stderr.Write(commandStderr)
		if err == nil && current.command.validateOutput != nil {
			err = current.command.validateOutput(commandStdout)
		}
		if err != nil {
			if current.command.goCommand &&
				!current.command.network &&
				isOfflineDependencyFailure(commandStderr) {
				err = fmt.Errorf(
					"%w（pre-commit はオフラインです。モジュールが不足する場合は pre-push を一度実行してください）",
					err,
				)
			}
			return reportStepFailure(current, err, stderr)
		}
		_, _ = fmt.Fprintf(stdout, "成功: %s [%s]\n", current.name, current.sotID)
	}
	return nil
}

func isOfflineDependencyFailure(stderr []byte) bool {
	message := strings.ToLower(string(stderr))
	for _, pattern := range []string{
		"goproxy=off",
		"module lookup disabled",
		"missing go.sum entry",
		"module cache",
	} {
		if strings.Contains(message, pattern) {
			return true
		}
	}
	return false
}

func reportStepFailure(current step, cause error, stderr io.Writer) error {
	err := fmt.Errorf("%s [%s] が失敗しました: %w", current.name, current.sotID, cause)
	_, _ = fmt.Fprintf(stderr, "失敗: %v\n", err)
	return err
}

func newOSCommandExecutor(ctx context.Context, caches cachePaths) osCommandExecutor {
	return osCommandExecutor{
		context:     ctx,
		environment: os.Environ(),
		caches:      caches,
	}
}
