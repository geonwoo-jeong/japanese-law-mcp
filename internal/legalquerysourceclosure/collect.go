package legalquerysourceclosure

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/module"
)

func collectGoListPackage(
	ctx context.Context,
	repositoryPath string,
	repository *repositoryReader,
	mainModulePath string,
	packageRoots []string,
	pkg goListPackage,
	seenImports map[string]struct{},
	foundRoots map[string]struct{},
	files map[string]SourceFile,
	modules map[string]listedModule,
	sourceBytes *int64,
) error {
	if pkg.ImportPath == "" {
		return fmt.Errorf("go dependency package の import path がありません")
	}
	if _, exists := seenImports[pkg.ImportPath]; exists {
		return fmt.Errorf("go dependency package %q が複数回解決されました", pkg.ImportPath)
	}
	seenImports[pkg.ImportPath] = struct{}{}
	if pkg.Error != nil || len(pkg.DepsErrors) != 0 {
		return fmt.Errorf("go dependency package %q に読込み error があります", pkg.ImportPath)
	}
	if pkg.Standard {
		if pkg.Module != nil {
			return fmt.Errorf("standard package %q に外部 module が設定されています", pkg.ImportPath)
		}
		return nil
	}
	if pkg.Module == nil {
		return fmt.Errorf("非 standard package %q の module identity がありません", pkg.ImportPath)
	}
	if pkg.Module.Main {
		return collectLocalPackage(ctx, repositoryPath, repository, mainModulePath, packageRoots, pkg, foundRoots, files, sourceBytes)
	}
	return collectExternalModule(pkg, mainModulePath, modules)
}

func collectLocalPackage(
	ctx context.Context,
	repositoryPath string,
	repository *repositoryReader,
	mainModulePath string,
	packageRoots []string,
	pkg goListPackage,
	foundRoots map[string]struct{},
	files map[string]SourceFile,
	sourceBytes *int64,
) error {
	if pkg.Module.Path != mainModulePath || pkg.Module.Version != "" || pkg.Module.Sum != "" || pkg.Module.GoModSum != "" || pkg.Module.Replace != nil {
		return fmt.Errorf("local package %q の main module identity が一致しません", pkg.ImportPath)
	}
	if !filepath.IsAbs(pkg.Dir) {
		return fmt.Errorf("local package %q の directory が絶対 path ではありません", pkg.ImportPath)
	}
	relativeDirectory, err := filepath.Rel(repositoryPath, filepath.Clean(pkg.Dir))
	if err != nil || relativeDirectory == ".." || strings.HasPrefix(relativeDirectory, ".."+string(filepath.Separator)) || filepath.IsAbs(relativeDirectory) {
		return fmt.Errorf("local package %q が repository 外を参照しています", pkg.ImportPath)
	}
	posixDirectory := filepath.ToSlash(relativeDirectory)
	wantImportPath := mainModulePath
	if posixDirectory != "." {
		if _, err := validateRepositoryRelativePath(posixDirectory); err != nil {
			return fmt.Errorf("local package %q の directory が不正です: %w", pkg.ImportPath, err)
		}
		if err := repository.validateDirectory(posixDirectory); err != nil {
			return fmt.Errorf("local package %q の directory を検証できません: %w", pkg.ImportPath, err)
		}
		wantImportPath += "/" + posixDirectory
	}
	if pkg.ImportPath != wantImportPath {
		return fmt.Errorf("local package %q の import path と directory が一致しません", pkg.ImportPath)
	}
	for _, root := range packageRoots {
		if posixDirectory == root {
			foundRoots[root] = struct{}{}
		}
	}
	for _, selected := range pkg.selectedFiles() {
		if err := collectLocalFile(ctx, repository, posixDirectory, selected, files, sourceBytes); err != nil {
			return fmt.Errorf("local package %q: %w", pkg.ImportPath, err)
		}
	}
	return nil
}

func collectLocalFile(
	ctx context.Context,
	repository *repositoryReader,
	packageDirectory string,
	selected string,
	files map[string]SourceFile,
	sourceBytes *int64,
) error {
	if _, err := validateRepositoryRelativePath(selected); err != nil {
		return fmt.Errorf("選択 source file %q が不正です: %w", selected, err)
	}
	relative := selected
	if packageDirectory != "." {
		relative = path.Join(packageDirectory, selected)
	}
	if _, exists := files[relative]; exists {
		return nil
	}
	if len(files) == MaximumSourceFiles {
		return fmt.Errorf("semantic source file 数が上限を超えています")
	}
	raw, err := repository.readRegularContext(ctx, relative, MaximumSourceFileBytes)
	if err != nil {
		return fmt.Errorf("semantic source file %q を読めません: %w", relative, err)
	}
	if int64(len(raw)) > MaximumSourceTotalBytes-*sourceBytes {
		return fmt.Errorf("semantic source file の合計 size が上限を超えています")
	}
	*sourceBytes += int64(len(raw))
	files[relative] = SourceFile{Path: relative, RawSHA256: rawSHA256(raw)}
	return nil
}

func collectExternalModule(pkg goListPackage, mainModulePath string, modules map[string]listedModule) error {
	dependency := pkg.Module
	if dependency.Path == mainModulePath || dependency.Replace != nil ||
		module.Check(dependency.Path, dependency.Version) != nil ||
		!moduleSumPattern.MatchString(dependency.Sum) || !moduleSumPattern.MatchString(dependency.GoModSum) {
		return fmt.Errorf("external package %q の module identity が不正です", pkg.ImportPath)
	}
	listed := listedModule{
		path:     dependency.Path,
		version:  dependency.Version,
		zipSum:   dependency.Sum,
		goModSum: dependency.GoModSum,
	}
	if current, exists := modules[dependency.Path]; exists {
		if current != listed {
			return fmt.Errorf("external module %q が複数 identity へ解決されました", dependency.Path)
		}
		return nil
	}
	if len(modules) == MaximumModuleCount {
		return fmt.Errorf("external module 数が上限を超えています")
	}
	modules[dependency.Path] = listed
	return nil
}

func (b Builder) inspectModules(ctx context.Context, listed []listedModule) ([]ModuleDependency, error) {
	if len(listed) != 0 && b.Modules == nil {
		return nil, fmt.Errorf("external module artifact provider が指定されていません")
	}
	dependencies := make([]ModuleDependency, 0, len(listed))
	var totalZipBytes int64
	var totalModBytes int64
	var totalEntries int
	var totalExpandedBytes int64
	for _, identity := range listed {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("external module 検証が中止されました: %w", err)
		}
		artifact, err := b.Modules.Load(ctx, identity.path, identity.version)
		if err != nil {
			return nil, fmt.Errorf("external module %s@%s の取得物を読めません: %w", identity.path, identity.version, err)
		}
		dependency, err := inspectModuleArtifact(ctx, identity, artifact)
		if err != nil {
			return nil, fmt.Errorf("external module %s@%s を検証できません: %w", identity.path, identity.version, err)
		}
		if dependency.ModuleZipByteLength > MaximumAllModuleZipBytes-totalZipBytes ||
			int64(len(artifact.GoMod)) > MaximumAllModuleModBytes-totalModBytes ||
			dependency.ModuleZipEntryCount > MaximumAllModuleEntries-totalEntries ||
			dependency.ModuleExpandedByteLength > MaximumAllModuleExpandBytes-totalExpandedBytes {
			return nil, fmt.Errorf("external module 取得物の合計資源量が上限を超えています")
		}
		totalZipBytes += dependency.ModuleZipByteLength
		totalModBytes += int64(len(artifact.GoMod))
		totalEntries += dependency.ModuleZipEntryCount
		totalExpandedBytes += dependency.ModuleExpandedByteLength
		dependencies = append(dependencies, dependency)
	}
	slices.SortFunc(dependencies, func(left, right ModuleDependency) int {
		if compared := strings.Compare(left.ModulePath, right.ModulePath); compared != 0 {
			return compared
		}
		return strings.Compare(left.Version, right.Version)
	})
	return dependencies, nil
}
