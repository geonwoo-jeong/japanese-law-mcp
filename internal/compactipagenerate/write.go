package compactipagenerate

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// SOT-ENG-051: 検証済みの一式を repository 内へ準備し、既存集合を退避して置き換える。
func replaceArtifacts(ctx context.Context, repository string, files map[string][]byte) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := os.OpenRoot(repository)
	if err != nil {
		return fmt.Errorf("repository を開けません: %w", err)
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	if err := checkRootDirectoryEntries(root, dataDirectory, files, true); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := root.MkdirAll(filepath.Dir(dataDirectory), 0750); err != nil {
		return fmt.Errorf("IPA 生成先を作れません: %w", err)
	}
	temporary := filepath.Join(filepath.Dir(dataDirectory), ".ipa-generated-"+rand.Text())
	if err := root.Mkdir(temporary, 0700); err != nil {
		return fmt.Errorf("IPA 生成領域を作れません: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			err = errors.Join(err, root.RemoveAll(temporary))
		}
	}()
	ready := filepath.Join(temporary, "ready")
	if err := root.Mkdir(ready, 0750); err != nil {
		return fmt.Errorf("IPA 生成領域を準備できません: %w", err)
	}
	for name, data := range files {
		if err := root.WriteFile(filepath.Join(ready, name), data, 0600); err != nil {
			return fmt.Errorf("IPA 生成物を保存できません: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	backup := filepath.Join(temporary, "previous")
	moved := false
	if err := root.Rename(dataDirectory, backup); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("IPA 生成物を退避できません: %w", err)
		}
	} else {
		moved = true
	}
	if err := root.Rename(ready, dataDirectory); err != nil {
		if moved {
			if restoreErr := root.Rename(backup, dataDirectory); restoreErr != nil {
				cleanup = false
				return errors.Join(err, fmt.Errorf("IPA 生成物を復元できません。退避先 %s を保持します: %w", backup, restoreErr))
			}
		}
		return fmt.Errorf("IPA 生成物を置き換えられません: %w", err)
	}
	return nil
}

func checkDirectoryEntries(repository string, expected map[string][]byte) (err error) {
	root, err := os.OpenRoot(repository)
	if err != nil {
		return fmt.Errorf("repository を開けません: %w", err)
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	return checkRootDirectoryEntries(root, dataDirectory, expected, true)
}

func checkRootDirectoryEntries(root *os.Root, directory string, expected map[string][]byte, allowManifest bool) error {
	entries, err := fs.ReadDir(root.FS(), filepath.ToSlash(directory))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		_, known := expected[entry.Name()]
		if entry.Name() == manifestName && allowManifest {
			known = true
		}
		if !known || !entry.Type().IsRegular() {
			return fmt.Errorf("IPA 生成先に未定義の file %s があります", entry.Name())
		}
	}
	return nil
}

func readRepositoryFile(repository, relative string) (data []byte, err error) {
	root, err := os.OpenRoot(repository)
	if err != nil {
		return nil, fmt.Errorf("repository を開けません: %w", err)
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	return root.ReadFile(filepath.FromSlash(relative))
}
