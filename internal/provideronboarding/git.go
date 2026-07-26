package provideronboarding

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
)

type gitClient struct {
	repository string
}

func newGitClient(repository string) gitClient {
	return gitClient{repository: repository}
}

func (client gitClient) resolveComparison(
	ctx context.Context,
	baseRef, headRef string,
) (comparison, error) {
	if _, err := client.run(ctx, "rev-parse", "--git-dir"); err != nil {
		return comparison{}, fmt.Errorf("git リポジトリ情報を取得できませんでした: %w", err)
	}
	baseCommit, err := client.resolveCommit(ctx, baseRef)
	if err != nil {
		return comparison{}, fmt.Errorf("%w: %q", ErrInvalidBaseRef, baseRef)
	}
	headCommit, err := client.resolveCommit(ctx, headRef)
	if err != nil {
		return comparison{}, fmt.Errorf("検査対象 commit を解決できませんでした: %q: %w", headRef, err)
	}
	output, err := client.run(ctx, "merge-base", baseCommit, headCommit)
	if err != nil {
		return comparison{}, fmt.Errorf("比較開始点の merge base を取得できませんでした: %w", err)
	}
	mergeBase, err := parseOID(output)
	if err != nil {
		return comparison{}, fmt.Errorf("merge base の応答が不正です: %w", err)
	}
	return comparison{
		baseCommit: baseCommit,
		headCommit: headCommit,
		mergeBase:  mergeBase,
	}, nil
}

func (client gitClient) resolveCommit(ctx context.Context, revision string) (string, error) {
	if revision == "" || strings.ContainsAny(revision, "\x00\r\n") {
		return "", errors.New("git revision が空または不正です")
	}
	output, err := client.run(
		ctx,
		"rev-parse",
		"--verify",
		"--quiet",
		"--end-of-options",
		revision+"^{commit}",
	)
	if err != nil {
		return "", err
	}
	return parseOID(output)
}

func (client gitClient) collectChangedPaths(
	ctx context.Context,
	resolved comparison,
	sources changeSources,
) ([]string, error) {
	type pathCommand struct {
		source    string
		arguments []string
	}
	commands := []pathCommand{{
		source: "commit",
		arguments: []string{
			"diff",
			"--name-only",
			"--no-renames",
			"-z",
			resolved.mergeBase,
			resolved.headCommit,
			"--",
		},
	}}
	if sources.index {
		commands = append(commands, pathCommand{
			source: "index",
			arguments: []string{
				"diff",
				"--cached",
				"--name-only",
				"--no-renames",
				"-z",
				"--",
			},
		})
	}
	if sources.workingTree {
		commands = append(commands, pathCommand{
			source: "working-tree",
			arguments: []string{
				"diff",
				"--name-only",
				"--no-renames",
				"-z",
				"--",
			},
		})
	}
	if sources.untracked {
		commands = append(commands, pathCommand{
			source: "untracked",
			arguments: []string{
				"ls-files",
				"--others",
				"--exclude-standard",
				"-z",
				"--",
			},
		})
	}

	unique := make(map[string]struct{})
	indexPaths := make(map[string]struct{})
	for _, command := range commands {
		output, err := client.run(ctx, command.arguments...)
		if err != nil {
			return nil, fmt.Errorf("git の変更パスを取得できませんでした: %w", err)
		}
		paths, err := parseNULPaths(output)
		if err != nil {
			return nil, err
		}
		for _, changedPath := range paths {
			if command.source == "working-tree" {
				if _, divergent := indexPaths[changedPath]; divergent {
					return nil, fmt.Errorf(
						"index と working tree で内容が異なる path は同時に検査できません: %s",
						changedPath,
					)
				}
			}
			if command.source == "index" {
				indexPaths[changedPath] = struct{}{}
			}
			unique[changedPath] = struct{}{}
		}
	}
	result := make([]string, 0, len(unique))
	for changedPath := range unique {
		result = append(result, changedPath)
	}
	sort.Strings(result)
	return result, nil
}

func (client gitClient) treePaths(
	ctx context.Context,
	commit string,
) (map[string]struct{}, error) {
	output, err := client.run(
		ctx,
		"ls-tree",
		"-r",
		"--name-only",
		"-z",
		commit,
		"--",
	)
	if err != nil {
		return nil, fmt.Errorf("比較元 commit の tree を取得できませんでした: %w", err)
	}
	paths, err := parseNULPaths(output)
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(paths))
	for _, treePath := range paths {
		result[treePath] = struct{}{}
	}
	return result, nil
}

func (client gitClient) run(ctx context.Context, arguments ...string) ([]byte, error) {
	commandArguments := append([]string{"-C", client.repository}, arguments...)
	//nolint:gosec // SOT-ENG-018: 実行ファイルは git に固定し、値は shell を介さず argv で渡す。
	command := exec.CommandContext(ctx, "git", commandArguments...)
	command.Env = environmentWithValue(os.Environ(), "GIT_NO_REPLACE_OBJECTS", "1")
	output, err := command.Output()
	if err == nil {
		return output, nil
	}
	message := err.Error()
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		stderr := strings.TrimSpace(string(exitError.Stderr))
		if stderr != "" {
			message = stderr
		}
	}
	return nil, errors.New(message)
}

func parseOID(output []byte) (string, error) {
	value := strings.ToLower(singleLineOutput(output))
	if len(value) != 40 && len(value) != 64 {
		return "", fmt.Errorf("object ID の長さが不正です: %q", value)
	}
	for _, character := range value {
		if !strings.ContainsRune("0123456789abcdef", character) {
			return "", fmt.Errorf("object ID に十六進数以外が含まれます: %q", value)
		}
	}
	return value, nil
}

func parseNULPaths(output []byte) ([]string, error) {
	if len(output) == 0 {
		return nil, nil
	}
	if output[len(output)-1] != 0 {
		return nil, errors.New("git の path 応答が NUL 終端ではありません")
	}
	records := bytes.Split(output[:len(output)-1], []byte{0})
	result := make([]string, 0, len(records))
	for _, record := range records {
		value := string(record)
		if err := validateGitPath(value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func validateGitPath(value string) error {
	if value == "" ||
		path.IsAbs(value) ||
		path.Clean(value) != value ||
		value == ".." ||
		strings.HasPrefix(value, "../") ||
		strings.ContainsAny(value, `\:`) {
		return fmt.Errorf("git が不正な repository path を返しました: %q", value)
	}
	return nil
}

func singleLineOutput(output []byte) string {
	return strings.TrimSuffix(strings.TrimSuffix(string(output), "\n"), "\r")
}

func environmentWithValue(environment []string, key, value string) []string {
	result := make([]string, 0, len(environment))
	for _, item := range environment {
		name, _, _ := strings.Cut(item, "=")
		if strings.EqualFold(name, key) {
			continue
		}
		result = append(result, item)
	}
	return append(result, key+"="+value)
}
