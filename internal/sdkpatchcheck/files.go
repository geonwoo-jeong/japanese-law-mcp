package sdkpatchcheck

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func checkComponents(root *os.Root, name string) error {
	if !validRelativePath(name) {
		return fmt.Errorf("SDK 照合 path が不正です")
	}
	parts := strings.Split(name, "/")
	for index := range parts {
		info, err := root.Lstat(strings.Join(parts[:index+1], "/"))
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (index+1 < len(parts) && !info.IsDir()) {
			return fmt.Errorf("SDK 照合 path に symlink または不正な directory があります")
		}
	}
	return nil
}

func readRegular(root *os.Root, name string, limit int64) (data []byte, err error) {
	if err := checkComponents(root, name); err != nil {
		return nil, err
	}
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() < 0 || before.Size() > limit {
		return nil, fmt.Errorf("SDK 照合 file は上限内の通常 file でなければなりません: %s", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, fmt.Errorf("SDK 照合 file が読取り中に置き換えられました")
	}
	data, err = io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("SDK 照合 file が上限を超えています")
	}
	return data, nil
}
