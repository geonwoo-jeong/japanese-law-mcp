package legalquerysourceclosure

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

var (
	goLanguageVersionPattern  = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?$`)
	goToolchainVersionPattern = regexp.MustCompile(`^go[0-9]+\.[0-9]+\.[0-9]+$`)
	moduleSumPattern          = regexp.MustCompile(`^h1:[A-Za-z0-9+/]{43}=$`)
)

// Builder は、固定 package roots から semantic source closure を構築する。
type Builder struct {
	Toolchain Toolchain
	Modules   ModuleProvider
}

// Build は、SOT-ENG-038 の固定 context で local file と外部 module を閉じる。
func (b Builder) Build(ctx context.Context, repositoryPath string, packageRoots []string) (SourceSet, error) {
	if b.Toolchain == nil {
		return SourceSet{}, fmt.Errorf("go toolchain が指定されていません")
	}
	if err := ctx.Err(); err != nil {
		return SourceSet{}, fmt.Errorf("semantic source closure の構築が中止されました: %w", err)
	}
	absoluteRepository, err := filepath.Abs(repositoryPath)
	if err != nil {
		return SourceSet{}, fmt.Errorf("repository root を解決できません")
	}
	repository, err := openRepositoryReader(absoluteRepository)
	if err != nil {
		return SourceSet{}, err
	}
	defer func() { _ = repository.Close() }()
	roots, err := normalizePackageRoots(repository, packageRoots)
	if err != nil {
		return SourceSet{}, err
	}
	moduleIdentity, err := readMainModule(ctx, repository)
	if err != nil {
		return SourceSet{}, err
	}
	toolchainVersion, err := b.Toolchain.Version(ctx, absoluteRepository)
	if err != nil {
		return SourceSet{}, fmt.Errorf("go toolchain version を取得できません: %w", err)
	}
	if !goToolchainVersionPattern.MatchString(toolchainVersion) {
		return SourceSet{}, fmt.Errorf("go toolchain version が固定可能な release 形式ではありません")
	}
	buildContext := FixedBuildContext()
	if err := buildContext.validate(); err != nil {
		return SourceSet{}, err
	}
	request := ListRequest{
		RepositoryPath: absoluteRepository,
		PackageRoots:   slices.Clone(roots),
		BuildContext:   buildContext,
	}
	output, err := b.Toolchain.ListDependencies(ctx, request)
	if err != nil {
		return SourceSet{}, fmt.Errorf("go dependency closure を列挙できません: %w", err)
	}
	closure, decodeErr := decodeGoList(ctx, output, absoluteRepository, repository, moduleIdentity.path, roots)
	closeErr := output.Close()
	if decodeErr != nil {
		return SourceSet{}, decodeErr
	}
	if closeErr != nil {
		return SourceSet{}, fmt.Errorf("go dependency closure command が失敗しました: %w", closeErr)
	}
	modules, err := b.inspectModules(ctx, closure.modules)
	if err != nil {
		return SourceSet{}, err
	}
	return SourceSet{
		mainModulePath:     moduleIdentity.path,
		goLanguageVersion:  moduleIdentity.goVersion,
		goToolchainVersion: toolchainVersion,
		goDebugSettings:    slices.Clone(moduleIdentity.goDebug),
		buildContext:       buildContext,
		files:              slices.Clone(closure.files),
		modules:            modules,
	}, nil
}

type mainModuleIdentity struct {
	path      string
	goVersion string
	goDebug   []GoDebugSetting
}

func readMainModule(ctx context.Context, repository *repositoryReader) (mainModuleIdentity, error) {
	raw, err := repository.readRegularContext(ctx, "go.mod", MaximumModuleGoModBytes)
	if err != nil {
		return mainModuleIdentity{}, fmt.Errorf("root go.mod を読めません: %w", err)
	}
	parsed, err := modfile.Parse("go.mod", raw, nil)
	if err != nil {
		return mainModuleIdentity{}, fmt.Errorf("root go.mod を解析できません: %w", err)
	}
	if parsed.Module == nil || module.CheckPath(parsed.Module.Mod.Path) != nil {
		return mainModuleIdentity{}, fmt.Errorf("root go.mod の module path が不正です")
	}
	if parsed.Go == nil || !goLanguageVersionPattern.MatchString(parsed.Go.Version) {
		return mainModuleIdentity{}, fmt.Errorf("root go.mod の go language version が不正です")
	}
	settings := make([]GoDebugSetting, 0, len(parsed.Godebug))
	seen := make(map[string]struct{}, len(parsed.Godebug))
	for _, setting := range parsed.Godebug {
		if setting == nil || setting.Key == "" || setting.Value == "" || strings.ContainsAny(setting.Key+setting.Value, "\x00\r\n") {
			return mainModuleIdentity{}, fmt.Errorf("root go.mod の godebug 設定が不正です")
		}
		if _, exists := seen[setting.Key]; exists {
			return mainModuleIdentity{}, fmt.Errorf("root go.mod の godebug 名が重複しています")
		}
		seen[setting.Key] = struct{}{}
		settings = append(settings, GoDebugSetting{Name: setting.Key, Value: setting.Value})
	}
	slices.SortFunc(settings, func(left, right GoDebugSetting) int {
		return strings.Compare(left.Name, right.Name)
	})
	return mainModuleIdentity{path: parsed.Module.Mod.Path, goVersion: parsed.Go.Version, goDebug: settings}, nil
}

func normalizePackageRoots(repository *repositoryReader, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("component package root が指定されていません")
	}
	unique := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, err := validateRepositoryRelativePath(value); err != nil {
			return nil, fmt.Errorf("component package root %q が不正です: %w", value, err)
		}
		if err := repository.validateDirectory(value); err != nil {
			return nil, fmt.Errorf("component package root %q を検証できません: %w", value, err)
		}
		unique[value] = struct{}{}
	}
	roots := make([]string, 0, len(unique))
	for value := range unique {
		roots = append(roots, value)
	}
	slices.Sort(roots)
	return roots, nil
}

type goListModule struct {
	Path     string        `json:"Path"`
	Version  string        `json:"Version"`
	Main     bool          `json:"Main"`
	Sum      string        `json:"Sum"`
	GoModSum string        `json:"GoModSum"`
	Replace  *goListModule `json:"Replace"`
}

type goListPackage struct {
	Dir          string        `json:"Dir"`
	ImportPath   string        `json:"ImportPath"`
	Standard     bool          `json:"Standard"`
	Module       *goListModule `json:"Module"`
	Error        *struct{}     `json:"Error"`
	DepsErrors   []struct{}    `json:"DepsErrors"`
	GoFiles      []string      `json:"GoFiles"`
	CgoFiles     []string      `json:"CgoFiles"`
	CFiles       []string      `json:"CFiles"`
	CXXFiles     []string      `json:"CXXFiles"`
	MFiles       []string      `json:"MFiles"`
	HFiles       []string      `json:"HFiles"`
	FFiles       []string      `json:"FFiles"`
	SFiles       []string      `json:"SFiles"`
	SwigFiles    []string      `json:"SwigFiles"`
	SwigCXXFiles []string      `json:"SwigCXXFiles"`
	SysoFiles    []string      `json:"SysoFiles"`
	EmbedFiles   []string      `json:"EmbedFiles"`
}

func (p goListPackage) selectedFiles() []string {
	total := len(p.GoFiles) + len(p.CgoFiles) + len(p.CFiles) + len(p.CXXFiles) + len(p.MFiles) +
		len(p.HFiles) + len(p.FFiles) + len(p.SFiles) + len(p.SwigFiles) + len(p.SwigCXXFiles) +
		len(p.SysoFiles) + len(p.EmbedFiles)
	files := make([]string, 0, total)
	files = append(files, p.GoFiles...)
	files = append(files, p.CgoFiles...)
	files = append(files, p.CFiles...)
	files = append(files, p.CXXFiles...)
	files = append(files, p.MFiles...)
	files = append(files, p.HFiles...)
	files = append(files, p.FFiles...)
	files = append(files, p.SFiles...)
	files = append(files, p.SwigFiles...)
	files = append(files, p.SwigCXXFiles...)
	files = append(files, p.SysoFiles...)
	files = append(files, p.EmbedFiles...)
	return files
}

type listedModule struct {
	path     string
	version  string
	zipSum   string
	goModSum string
}

type decodedClosure struct {
	files   []SourceFile
	modules []listedModule
}

func decodeGoList(
	ctx context.Context,
	output io.Reader,
	repositoryPath string,
	repository *repositoryReader,
	mainModulePath string,
	packageRoots []string,
) (decodedClosure, error) {
	limited := &io.LimitedReader{R: contextReader{ctx: ctx, reader: output}, N: maximumGoListBytes + 1}
	decoder := json.NewDecoder(bufio.NewReaderSize(limited, 32<<10))
	seenImports := make(map[string]struct{})
	files := make(map[string]SourceFile)
	modules := make(map[string]listedModule)
	foundRoots := make(map[string]struct{}, len(packageRoots))
	var sourceBytes int64
	packageCount := 0
	for {
		if err := ctx.Err(); err != nil {
			return decodedClosure{}, fmt.Errorf("go dependency closure の読取りが中止されました: %w", err)
		}
		var pkg goListPackage
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return decodedClosure{}, fmt.Errorf("go dependency closure JSON を解析できません: %w", err)
		}
		if limited.N <= 0 {
			return decodedClosure{}, fmt.Errorf("go dependency closure JSON が上限を超えています")
		}
		packageCount++
		if packageCount > maximumGoListPackages {
			return decodedClosure{}, fmt.Errorf("go dependency package 数が上限を超えています")
		}
		if err := collectGoListPackage(ctx, repositoryPath, repository, mainModulePath, packageRoots, pkg, seenImports, foundRoots, files, modules, &sourceBytes); err != nil {
			return decodedClosure{}, err
		}
	}
	for _, root := range packageRoots {
		if _, found := foundRoots[root]; !found {
			return decodedClosure{}, fmt.Errorf("component package root %q が dependency closure にありません", root)
		}
	}
	resultFiles := make([]SourceFile, 0, len(files))
	for _, file := range files {
		resultFiles = append(resultFiles, file)
	}
	slices.SortFunc(resultFiles, func(left, right SourceFile) int { return strings.Compare(left.Path, right.Path) })
	resultModules := make([]listedModule, 0, len(modules))
	for _, dependency := range modules {
		resultModules = append(resultModules, dependency)
	}
	slices.SortFunc(resultModules, func(left, right listedModule) int {
		if compared := strings.Compare(left.path, right.path); compared != 0 {
			return compared
		}
		return strings.Compare(left.version, right.version)
	})
	return decodedClosure{files: resultFiles, modules: resultModules}, nil
}
