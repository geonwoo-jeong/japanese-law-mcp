package githook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type expectedFile struct {
	oid        string
	executable bool
}

func (app *application) validateTreeSnapshot(
	ctx context.Context,
	treeish, snapshot string,
) error {
	expected, err := app.expectedTreeFiles(ctx, treeish)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(expected))
	err = filepath.WalkDir(snapshot, func(
		target string,
		entry os.DirEntry,
		walkErr error,
	) error {
		return app.validateSnapshotEntry(ctx, snapshot, target, entry, walkErr, expected, seen)
	})
	if err != nil {
		return err
	}
	for filename := range expected {
		if _, ok := seen[filename]; !ok {
			return fmt.Errorf("archive snapshot に commit のファイルがありません: %s", filename)
		}
	}
	return nil
}

func (app *application) expectedTreeFiles(
	ctx context.Context,
	treeish string,
) (map[string]expectedFile, error) {
	command := gitCommand(ctx, app.repository, nil, "ls-tree", "-r", "-z", treeish)
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("commit tree を取得できませんでした: %w", err)
	}
	files := make(map[string]expectedFile)
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		filename, file, parseErr := parseExpectedFile(record)
		if parseErr != nil {
			return nil, parseErr
		}
		if _, exists := files[filename]; exists {
			return nil, fmt.Errorf("commit tree に path が重複しています: %s", filename)
		}
		files[filename] = file
	}
	return files, nil
}

func parseExpectedFile(record []byte) (string, expectedFile, error) {
	metadata, rawFilename, found := bytes.Cut(record, []byte{'\t'})
	fields := bytes.Fields(metadata)
	if !found || len(fields) != 3 || len(rawFilename) == 0 {
		return "", expectedFile{}, errors.New("commit tree に解釈できない entry があります")
	}
	filename := string(rawFilename)
	if err := validateRepositoryPath(filename); err != nil {
		return "", expectedFile{}, err
	}
	mode, objectType, oid := string(fields[0]), string(fields[1]), string(fields[2])
	if objectType != "blob" || !validOID(oid) {
		return "", expectedFile{}, fmt.Errorf("commit tree に未対応の entry があります: %s", filename)
	}
	switch mode {
	case "100644":
		return filename, expectedFile{oid: oid}, nil
	case "100755":
		return filename, expectedFile{oid: oid, executable: true}, nil
	case "120000":
		return "", expectedFile{}, fmt.Errorf("commit tree にシンボリックリンクがあります: %s", filename)
	default:
		return "", expectedFile{}, fmt.Errorf("commit tree mode %s は未対応です: %s", mode, filename)
	}
}

func (app *application) validateSnapshotEntry(
	ctx context.Context,
	root, target string,
	entry os.DirEntry,
	walkErr error,
	expected map[string]expectedFile,
	seen map[string]struct{},
) error {
	if walkErr != nil {
		return walkErr
	}
	if target == root || entry.IsDir() {
		return nil
	}
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("archive snapshot に通常ファイル以外があります: %s", target)
	}
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	filename := filepath.ToSlash(relative)
	file, ok := expected[filename]
	if !ok {
		return fmt.Errorf("archive snapshot に commit tree 外のファイルがあります: %s", filename)
	}
	if _, duplicate := seen[filename]; duplicate {
		return fmt.Errorf("archive snapshot に path が重複しています: %s", filename)
	}
	seen[filename] = struct{}{}
	if hasPOSIXPermissionBits() &&
		(info.Mode().Perm()&0o111 != 0) != file.executable {
		return fmt.Errorf("archive snapshot の実行権限が一致しません: %s", filename)
	}
	return app.validateSnapshotBlob(ctx, target, filename, file.oid)
}

func (app *application) validateSnapshotBlob(
	ctx context.Context,
	target, filename, expectedOID string,
) error {
	command := gitCommand(
		ctx,
		app.repository,
		nil,
		"hash-object",
		"--no-filters",
		"--",
		target,
	)
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("archive snapshot の blob を計算できませんでした: %s: %w", filename, err)
	}
	actualOID := strings.ToLower(stringWithoutLineEnding(output))
	if actualOID != strings.ToLower(expectedOID) {
		return fmt.Errorf("archive snapshot の内容が commit tree と一致しません: %s", filename)
	}
	return nil
}

func validateRepositoryPath(name string) error {
	if name == "" ||
		path.IsAbs(name) ||
		path.Clean(name) != name ||
		strings.ContainsAny(name, `\:`) {
		return fmt.Errorf("リポジトリ path が不正です: %q", name)
	}
	if name == ".." || strings.HasPrefix(name, "../") {
		return fmt.Errorf("リポジトリ外を指す path です: %q", name)
	}
	return nil
}
