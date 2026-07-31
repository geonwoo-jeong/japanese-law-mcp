// Package legalquerysourceclosure は、候補 query composition の意味 source 集合を固定する。
package legalquerysourceclosure

import (
	"context"
	"fmt"
	"io"
	"slices"
)

const (
	MaximumSourceFiles          = 8192
	MaximumSourceFileBytes      = 8 << 20
	MaximumSourceTotalBytes     = 128 << 20
	MaximumModuleCount          = 1024
	MaximumModuleZipBytes       = 64 << 20
	MaximumModuleGoModBytes     = 1 << 20
	MaximumModuleZipEntries     = 16384
	MaximumModuleEntryBytes     = 16 << 20
	MaximumModuleExpandBytes    = 128 << 20
	MaximumAllModuleZipBytes    = 512 << 20
	MaximumAllModuleModBytes    = 16 << 20
	MaximumAllModuleEntries     = 131072
	MaximumAllModuleExpandBytes = 512 << 20
	maximumGoListBytes          = 64 << 20
	maximumGoListPackages       = 65536
)

// GoDebugSetting は、go.mod が明示した一つの godebug 設定である。
type GoDebugSetting struct {
	Name  string
	Value string
}

// SourceFile は、repository-relative POSIX path と検証済み原 byte digest を保持する。
type SourceFile struct {
	Path      string
	RawSHA256 string
}

// ModuleDependency は、一外部 module の取得物と展開量を固定する。
type ModuleDependency struct {
	ModulePath               string
	Version                  string
	ModuleZipSum             string
	ModuleZipRawSHA256       string
	ModuleZipByteLength      int64
	ModuleZipEntryCount      int
	ModuleExpandedByteLength int64
	ModuleGoModSum           string
	ModuleGoModRawSHA256     string
}

// BuildContext は、候補 source を選ぶ閉じた Go build context である。
type BuildContext struct {
	goos         string
	goarch       string
	goamd64      string
	goexperiment string
	cgoEnabled   int
	buildTags    []string
}

// FixedBuildContext は、SOT-ENG-038 が固定した build context を返す。
func FixedBuildContext() BuildContext {
	return BuildContext{
		goos:       "linux",
		goarch:     "amd64",
		goamd64:    "v1",
		cgoEnabled: 0,
		buildTags:  []string{},
	}
}

func (c BuildContext) GOOS() string         { return c.goos }
func (c BuildContext) GOARCH() string       { return c.goarch }
func (c BuildContext) GOAMD64() string      { return c.goamd64 }
func (c BuildContext) GOEXPERIMENT() string { return c.goexperiment }
func (c BuildContext) CGOEnabled() int      { return c.cgoEnabled }
func (c BuildContext) BuildTags() []string  { return slices.Clone(c.buildTags) }

func (c BuildContext) validate() error {
	fixed := FixedBuildContext()
	if c.goos != fixed.goos || c.goarch != fixed.goarch || c.goamd64 != fixed.goamd64 ||
		c.goexperiment != fixed.goexperiment || c.cgoEnabled != fixed.cgoEnabled || len(c.buildTags) != 0 {
		return fmt.Errorf("go build context が SOT-ENG-038 の固定値ではありません")
	}
	return nil
}

// SourceSet は、manifest へ投影する検証済み semantic source closure である。
type SourceSet struct {
	mainModulePath     string
	goLanguageVersion  string
	goToolchainVersion string
	goDebugSettings    []GoDebugSetting
	buildContext       BuildContext
	files              []SourceFile
	modules            []ModuleDependency
}

func (s SourceSet) MainModulePath() string                 { return s.mainModulePath }
func (s SourceSet) GoLanguageVersion() string              { return s.goLanguageVersion }
func (s SourceSet) GoToolchainVersion() string             { return s.goToolchainVersion }
func (s SourceSet) GoDebugSettings() []GoDebugSetting      { return slices.Clone(s.goDebugSettings) }
func (s SourceSet) GOOS() string                           { return s.buildContext.GOOS() }
func (s SourceSet) GOARCH() string                         { return s.buildContext.GOARCH() }
func (s SourceSet) GOAMD64() string                        { return s.buildContext.GOAMD64() }
func (s SourceSet) GOEXPERIMENT() string                   { return s.buildContext.GOEXPERIMENT() }
func (s SourceSet) CGOEnabled() int                        { return s.buildContext.CGOEnabled() }
func (s SourceSet) BuildTags() []string                    { return s.buildContext.BuildTags() }
func (s SourceSet) Files() []SourceFile                    { return slices.Clone(s.files) }
func (s SourceSet) ModuleDependencies() []ModuleDependency { return slices.Clone(s.modules) }

// ListRequest は、toolchain へ渡す閉じた dependency-list request である。
type ListRequest struct {
	RepositoryPath string
	PackageRoots   []string
	BuildContext   BuildContext
}

// Toolchain は、固定 toolchain version と dependency closure を提供する。
type Toolchain interface {
	Version(ctx context.Context, repositoryPath string) (string, error)
	ListDependencies(ctx context.Context, request ListRequest) (io.ReadCloser, error)
}

// ModuleArtifact は、同一取得で検証する module zip と go.mod の原 byte である。
type ModuleArtifact struct {
	Zip     []byte
	GoMod   []byte
	ZipHash string
}

// ModuleProvider は、列挙済み exact module identity の原取得物だけを返す。
type ModuleProvider interface {
	Load(ctx context.Context, modulePath string, version string) (ModuleArtifact, error)
}
