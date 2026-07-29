package githook

import (
	"context"
	"io"
	"os"
	"os/exec"
)

func runQualityGate(
	ctx context.Context,
	profile, repository, gitRepository, gitIndexFile string,
	gitRanges []string,
	stdout io.Writer,
	stderr io.Writer,
) error {
	caches, err := existingHookCachePaths(ctx, gitRepository)
	if err != nil {
		return err
	}
	arguments := []string{
		"run",
		"./cmd/quality-gate",
		"--profile=" + profile,
		"--repository=" + repository,
		"--git-repository=" + gitRepository,
	}
	for _, gitRange := range gitRanges {
		arguments = append(arguments, "--git-range="+gitRange)
	}
	//nolint:gosec // SOT-ENG-027: 実行ファイルとサブコマンドを固定し、検証済み値を argv として渡す。
	command := exec.CommandContext(ctx, "go", arguments...)
	command.Dir = repository
	command.Env = environmentWithHookCaches(
		controlledGoEnvironment(os.Environ(), false),
		caches,
	)
	if gitIndexFile != "" {
		command.Env = environmentWithValue(command.Env, "GIT_INDEX_FILE", gitIndexFile)
	}
	command.Stdin = nil
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}
