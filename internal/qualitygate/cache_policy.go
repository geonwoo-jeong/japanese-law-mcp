package qualitygate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func verifySnapshotCachePolicy(snapshot string) error {
	for _, name := range []string{".cache", ".tmp"} {
		root := filepath.Join(snapshot, name)
		info, err := os.Lstat(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("snapshot の %s を確認できません: %w", name, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("snapshot の %s は実体のあるディレクトリではありません", name)
		}
		if err := rejectCacheContent(root, name); err != nil {
			return err
		}
	}
	coveragePath := filepath.Join(snapshot, "coverage.out")
	if _, err := os.Lstat(coveragePath); err == nil {
		return fmt.Errorf("SOT-ENG-019: snapshot の coverage.out を追跡対象にできません")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("snapshot の coverage.out を確認できません: %w", err)
	}
	return nil
}

func rejectCacheContent(root, name string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root && !entry.IsDir() {
			return fmt.Errorf(
				"SOT-ENG-019: snapshot の %s 配下を追跡対象にできません: %s",
				name,
				path,
			)
		}
		return nil
	})
}
