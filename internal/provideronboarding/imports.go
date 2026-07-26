package provideronboarding

import (
	"bufio"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type providerPackage struct {
	providerID string
	path       string
}

func validateProviderImports(repository string, rows []matrixRow) error {
	packages, err := collectProviderPackages(rows)
	if err != nil {
		return err
	}
	if len(packages) == 0 {
		return nil
	}
	module, err := readModulePath(repository)
	if err != nil {
		return err
	}
	for _, source := range packages {
		root := filepath.Join(repository, filepath.FromSlash(source.path))
		info, err := os.Lstat(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("provider package を確認できません: %s: %w", source.path, err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("provider package が実体のあるディレクトリではありません: %s", source.path)
		}
		if err := inspectProviderPackage(repository, root, module, source, packages); err != nil {
			return err
		}
	}
	return nil
}

func inspectProviderPackage(
	repository, root, module string,
	source providerPackage,
	packages []providerPackage,
) error {
	return filepath.WalkDir(root, func(target string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("provider package 内にシンボリックリンクがあります: %s", target)
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("provider の Go file を確認できません: %s: %w", target, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("provider の Go file が通常ファイルではありません: %s", target)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), target, nil, parser.ImportsOnly)
		if err != nil {
			relative, _ := filepath.Rel(repository, target)
			return fmt.Errorf(
				"provider の Go import を解析できません: %s: %w",
				filepath.ToSlash(relative),
				err,
			)
		}
		for _, imported := range parsed.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return fmt.Errorf("go import path を解釈できません: %s: %w", target, err)
			}
			targetPackage, found := importedProviderPackage(importPath, module, packages)
			if found && targetPackage.providerID != source.providerID {
				return fmt.Errorf(
					"provider package 間の import は禁止されています: %s から %s",
					source.providerID,
					targetPackage.providerID,
				)
			}
		}
		return nil
	})
}

func collectProviderPackages(rows []matrixRow) ([]providerPackage, error) {
	seen := make(map[providerPackage]struct{})
	result := make([]providerPackage, 0)
	for _, row := range rows {
		if row.implementedBy == "" {
			continue
		}
		if row.providerID == "" {
			return nil, errors.New("implementedBy を持つ matrix row の providerId が空です")
		}
		if err := validateProviderPackagePath(row.implementedBy); err != nil {
			return nil, fmt.Errorf(
				"providerId=%s の implementedBy が不正です: %w",
				row.providerID,
				err,
			)
		}
		item := providerPackage{providerID: row.providerID, path: row.implementedBy}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].path != result[right].path {
			return result[left].path < result[right].path
		}
		return result[left].providerID < result[right].providerID
	})
	for left := range result {
		for right := left + 1; right < len(result); right++ {
			if result[left].providerID == result[right].providerID {
				continue
			}
			if hasPathPrefix(result[left].path, result[right].path) ||
				hasPathPrefix(result[right].path, result[left].path) {
				return nil, fmt.Errorf(
					"異なる provider の package path が重複しています: %s, %s",
					result[left].path,
					result[right].path,
				)
			}
		}
	}
	return result, nil
}

func validateProviderPackagePath(value string) error {
	if !hasPathPrefix(value, "internal/source") ||
		value == "internal/source" ||
		filepath.ToSlash(filepath.Clean(filepath.FromSlash(value))) != value ||
		strings.ContainsAny(value, `\:`) {
		return fmt.Errorf("provider package path ではありません: %q", value)
	}
	return nil
}

func providerPackageOwner(
	changedPath string,
	packages []providerPackage,
) (providerPackage, bool) {
	var owner providerPackage
	found := false
	for _, candidate := range packages {
		if !hasPathPrefix(changedPath, candidate.path) {
			continue
		}
		if !found || len(candidate.path) > len(owner.path) {
			owner = candidate
			found = true
		}
	}
	return owner, found
}

func importedProviderPackage(
	importPath, module string,
	packages []providerPackage,
) (providerPackage, bool) {
	prefix := module + "/"
	if !strings.HasPrefix(importPath, prefix) {
		return providerPackage{}, false
	}
	return providerPackageOwner(strings.TrimPrefix(importPath, prefix), packages)
}

func readModulePath(repository string) (string, error) {
	target := filepath.Join(repository, "go.mod")
	info, err := os.Lstat(target)
	if err != nil {
		return "", fmt.Errorf("go.mod を確認できません: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("go.mod が通常ファイルではありません")
	}
	//nolint:gosec // SOT-ENG-018: target は検証済み repository root と固定名 go.mod の結合結果に限定する。
	file, err := os.Open(target)
	if err != nil {
		return "", fmt.Errorf("go.mod を開けません: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.HasPrefix(fields[0], "//") {
			continue
		}
		if len(fields) != 2 || fields[0] != "module" || fields[1] == "" {
			return "", errors.New("go.mod の module 宣言が不正です")
		}
		return fields[1], nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("go.mod を読み取れません: %w", err)
	}
	return "", errors.New("go.mod に module 宣言がありません")
}
