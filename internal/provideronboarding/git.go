package provideronboarding

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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
	changes, err := client.collectChanges(ctx, resolved, sources)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), changes.paths...), nil
}

func (client gitClient) collectChanges(
	ctx context.Context,
	resolved comparison,
	sources changeSources,
) (changedPathSet, error) {
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

	changes := changedPathSet{
		commit:      make(map[string]struct{}),
		index:       make(map[string]struct{}),
		workingTree: make(map[string]struct{}),
		untracked:   make(map[string]struct{}),
	}
	unique := make(map[string]struct{})
	for _, command := range commands {
		output, err := client.run(ctx, command.arguments...)
		if err != nil {
			return changedPathSet{}, fmt.Errorf(
				"git の変更パスを取得できませんでした: %w",
				err,
			)
		}
		paths, err := parseNULPaths(output)
		if err != nil {
			return changedPathSet{}, err
		}
		for _, changedPath := range paths {
			if command.source == "working-tree" {
				if _, divergent := changes.index[changedPath]; divergent {
					return changedPathSet{}, fmt.Errorf(
						"index と working tree で内容が異なる path は同時に検査できません: %s",
						changedPath,
					)
				}
			}
			switch command.source {
			case "commit":
				changes.commit[changedPath] = struct{}{}
			case "index":
				changes.index[changedPath] = struct{}{}
			case "working-tree":
				changes.workingTree[changedPath] = struct{}{}
			case "untracked":
				changes.untracked[changedPath] = struct{}{}
			}
			unique[changedPath] = struct{}{}
		}
	}
	changes.paths = make([]string, 0, len(unique))
	for changedPath := range unique {
		changes.paths = append(changes.paths, changedPath)
	}
	sort.Strings(changes.paths)
	return changes, nil
}

func (client gitClient) comparisonPathContents(
	ctx context.Context,
	repository string,
	resolved comparison,
	changes changedPathSet,
	changedPath string,
) ([]byte, []byte, bool, error) {
	previous, previousExists, err := client.commitPathContent(
		ctx,
		resolved.mergeBase,
		changedPath,
	)
	if err != nil {
		return nil, nil, false, fmt.Errorf(
			"比較元の path を読み取れません: %s: %w",
			changedPath,
			err,
		)
	}
	current, currentExists, err := client.currentPathContent(
		ctx,
		repository,
		resolved.headCommit,
		changes,
		changedPath,
	)
	if err != nil {
		return nil, nil, false, fmt.Errorf(
			"現在の検査対象 path を読み取れません: %s: %w",
			changedPath,
			err,
		)
	}
	if !previousExists || !currentExists {
		return nil, nil, false, nil
	}
	return previous, current, true, nil
}

func (client gitClient) currentPathContent(
	ctx context.Context,
	repository, headCommit string,
	changes changedPathSet,
	changedPath string,
) ([]byte, bool, error) {
	if _, exists := changes.untracked[changedPath]; exists {
		return readWorkingTreePath(repository, changedPath)
	}
	if _, exists := changes.workingTree[changedPath]; exists {
		return readWorkingTreePath(repository, changedPath)
	}
	if _, exists := changes.index[changedPath]; exists {
		return client.indexPathContent(ctx, changedPath)
	}
	if _, exists := changes.commit[changedPath]; exists {
		return client.commitPathContent(ctx, headCommit, changedPath)
	}
	return nil, false, errors.New("現在内容の由来を Git layer から確定できません")
}

func (client gitClient) commitPathContent(
	ctx context.Context,
	commit, repositoryPath string,
) ([]byte, bool, error) {
	if _, err := parseOID([]byte(commit)); err != nil {
		return nil, false, fmt.Errorf("commit object ID が不正です: %w", err)
	}
	if err := validateGitPath(repositoryPath); err != nil {
		return nil, false, err
	}
	output, err := client.run(
		ctx,
		"ls-tree",
		"-z",
		commit,
		"--",
		repositoryPath,
	)
	if err != nil {
		return nil, false, err
	}
	objectID, exists, err := parseTreeBlob(output, repositoryPath)
	if err != nil || !exists {
		return nil, exists, err
	}
	content, err := client.run(ctx, "cat-file", "blob", objectID)
	if err != nil {
		return nil, false, fmt.Errorf("blob を読み取れません: %w", err)
	}
	return content, true, nil
}

func (client gitClient) indexPathContent(
	ctx context.Context,
	repositoryPath string,
) ([]byte, bool, error) {
	if err := validateGitPath(repositoryPath); err != nil {
		return nil, false, err
	}
	output, err := client.run(
		ctx,
		"ls-files",
		"--stage",
		"-z",
		"--",
		repositoryPath,
	)
	if err != nil {
		return nil, false, err
	}
	objectID, exists, err := parseIndexBlob(output, repositoryPath)
	if err != nil || !exists {
		return nil, exists, err
	}
	content, err := client.run(ctx, "cat-file", "blob", objectID)
	if err != nil {
		return nil, false, fmt.Errorf("index blob を読み取れません: %w", err)
	}
	return content, true, nil
}

func readWorkingTreePath(
	repository, repositoryPath string,
) ([]byte, bool, error) {
	if err := validateGitPath(repositoryPath); err != nil {
		return nil, false, err
	}
	target := filepath.Join(repository, filepath.FromSlash(repositoryPath))
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, errors.New("通常ファイルではありません")
	}
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(target))
	if err != nil {
		return nil, false, fmt.Errorf("親ディレクトリを解決できません: %w", err)
	}
	resolvedRepository, err := filepath.EvalSymlinks(repository)
	if err != nil {
		return nil, false, fmt.Errorf("repository を解決できません: %w", err)
	}
	relativeParent, err := filepath.Rel(resolvedRepository, resolvedParent)
	if err != nil || relativeParent == ".." ||
		strings.HasPrefix(relativeParent, ".."+string(filepath.Separator)) {
		return nil, false, errors.New("repository 外の path は読み取れません")
	}
	content, err := os.ReadFile(target) //nolint:gosec // SOT-ENG-044: Git が列挙し、repository 内と通常ファイルであることを確認した matrix だけを読む。
	if err != nil {
		return nil, false, err
	}
	return content, true, nil
}

func parseTreeBlob(
	output []byte,
	expectedPath string,
) (string, bool, error) {
	metadata, actualPath, exists, err := parseSingleGitEntry(output)
	if err != nil || !exists {
		return "", exists, err
	}
	if actualPath != expectedPath {
		return "", false, fmt.Errorf(
			"git tree の path が一致しません: got=%q want=%q",
			actualPath,
			expectedPath,
		)
	}
	fields := strings.Fields(metadata)
	if len(fields) != 3 || fields[1] != "blob" || !regularGitMode(fields[0]) {
		return "", false, fmt.Errorf("通常 blob ではありません: %q", metadata)
	}
	objectID, err := parseOID([]byte(fields[2]))
	if err != nil {
		return "", false, err
	}
	return objectID, true, nil
}

func parseIndexBlob(
	output []byte,
	expectedPath string,
) (string, bool, error) {
	metadata, actualPath, exists, err := parseSingleGitEntry(output)
	if err != nil || !exists {
		return "", exists, err
	}
	if actualPath != expectedPath {
		return "", false, fmt.Errorf(
			"git index の path が一致しません: got=%q want=%q",
			actualPath,
			expectedPath,
		)
	}
	fields := strings.Fields(metadata)
	if len(fields) != 3 || fields[2] != "0" || !regularGitMode(fields[0]) {
		return "", false, fmt.Errorf("通常の stage 0 blob ではありません: %q", metadata)
	}
	objectID, err := parseOID([]byte(fields[1]))
	if err != nil {
		return "", false, err
	}
	return objectID, true, nil
}

func parseSingleGitEntry(output []byte) (string, string, bool, error) {
	if len(output) == 0 {
		return "", "", false, nil
	}
	if output[len(output)-1] != 0 {
		return "", "", false, errors.New("git entry が NUL 終端ではありません")
	}
	records := bytes.Split(output[:len(output)-1], []byte{0})
	if len(records) != 1 {
		return "", "", false, errors.New("git entry が一つに確定しません")
	}
	metadata, pathBytes, found := bytes.Cut(records[0], []byte{'\t'})
	if !found {
		return "", "", false, errors.New("git entry の形式が不正です")
	}
	repositoryPath := string(pathBytes)
	if err := validateGitPath(repositoryPath); err != nil {
		return "", "", false, err
	}
	return string(metadata), repositoryPath, true, nil
}

func regularGitMode(value string) bool {
	return value == "100644" || value == "100755"
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
	//nolint:gosec // SOT-ENG-044: 実行ファイルは git に固定し、値は shell を介さず argv で渡す。
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
