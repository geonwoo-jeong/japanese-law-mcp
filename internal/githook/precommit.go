package githook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (app *application) preCommit(ctx context.Context) (result error) {
	treeOID, err := app.pinIndex(ctx)
	if err != nil {
		return err
	}

	directory, err := os.MkdirTemp("", "japanese-law-mcp-pre-commit-")
	if err != nil {
		return fmt.Errorf("index snapshot 用の一時ディレクトリを作成できませんでした: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(directory); result == nil && cleanupErr != nil {
			result = fmt.Errorf("index snapshot を削除できませんでした: %w", cleanupErr)
		}
	}()

	indexFile := filepath.Join(directory, "index")
	snapshot := filepath.Join(directory, "snapshot")
	if err := os.Mkdir(snapshot, 0o700); err != nil {
		return fmt.Errorf("index snapshot を作成できませんでした: %w", err)
	}
	if app.indexPinned != nil {
		app.indexPinned(treeOID)
	}
	if err := app.checkoutTree(ctx, treeOID, indexFile, snapshot); err != nil {
		return err
	}
	return app.qualityGate(
		ctx,
		"pre-commit",
		snapshot,
		app.repository,
		indexFile,
		nil,
		app.stdout,
		app.stderr,
	)
}

func (app *application) pinIndex(ctx context.Context) (string, error) {
	command := gitCommand(ctx, app.repository, nil, "write-tree")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("対象の Git index を immutable tree に固定できませんでした: %w", err)
	}
	treeOID := stringWithoutLineEnding(output)
	if !validOID(treeOID) {
		return "", fmt.Errorf("write-tree が不正な object ID を返しました: %q", treeOID)
	}
	if err := app.validateTree(ctx, treeOID); err != nil {
		return "", err
	}
	return treeOID, nil
}

func (app *application) validateTree(ctx context.Context, treeOID string) error {
	command := gitCommand(ctx, app.repository, nil, "ls-tree", "-r", "-z", treeOID)
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("固定済み Git tree を確認できませんでした: %w", err)
	}
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		if err := validateTreeRecord(record); err != nil {
			return err
		}
	}
	return nil
}

func validateTreeRecord(record []byte) error {
	metadata, filename, found := bytes.Cut(record, []byte{'\t'})
	fields := bytes.Fields(metadata)
	if !found || len(fields) != 3 || len(filename) == 0 {
		return errors.New("固定済み Git tree に解釈できない entry があります")
	}
	if err := validateRepositoryPath(string(filename)); err != nil {
		return err
	}

	switch string(fields[0]) {
	case "100644", "100755":
		if string(fields[1]) != "blob" || !validOID(string(fields[2])) {
			return fmt.Errorf("固定済み Git tree の通常ファイル entry が不正です: %s", filename)
		}
		return nil
	case "120000":
		return fmt.Errorf("シンボリックリンクは snapshot に含められません: %s", filename)
	case "160000":
		return fmt.Errorf("submodule は snapshot に含められません: %s", filename)
	default:
		return fmt.Errorf("未対応の Git index mode %s です: %s", fields[0], filename)
	}
}

func (app *application) checkoutTree(
	ctx context.Context,
	treeOID, indexFile, snapshot string,
) error {
	readTree := gitCommandWithIndex(ctx, app.repository, indexFile, "read-tree", treeOID)
	if output, err := readTree.CombinedOutput(); err != nil {
		return fmt.Errorf("固定済み tree を一時 index へ読み込めませんでした: %w: %s", err, output)
	}
	return app.materializeTreeSnapshot(ctx, treeOID, snapshot)
}

func gitCommandWithIndex(
	ctx context.Context,
	repository, indexFile string,
	args ...string,
) *exec.Cmd {
	command := gitCommand(ctx, repository, nil, args...)
	command.Env = environmentWithValue(command.Env, "GIT_INDEX_FILE", indexFile)
	return command
}
