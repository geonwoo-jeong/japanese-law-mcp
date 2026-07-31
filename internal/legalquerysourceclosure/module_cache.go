package legalquerysourceclosure

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"golang.org/x/mod/module"
)

// ModuleCacheProvider は、既存 module download cache を取得物の読取り元にする。
// 評価時の fresh cache materialization はこの型の責務ではない。
type ModuleCacheProvider struct {
	cacheRoot string
}

// NewModuleCacheProvider は、symlink ではない module cache root を固定する。
func NewModuleCacheProvider(cacheRoot string) (*ModuleCacheProvider, error) {
	absolute, err := filepath.Abs(cacheRoot)
	if err != nil {
		return nil, fmt.Errorf("module cache root を解決できません")
	}
	reader, err := openRepositoryReader(absolute)
	if err != nil {
		return nil, fmt.Errorf("module cache root を検証できません: %w", err)
	}
	if err := reader.Close(); err != nil {
		return nil, err
	}
	return &ModuleCacheProvider{cacheRoot: absolute}, nil
}

// Load は、exact module path/version の zip と go.mod を上限付きで一回ずつ読む。
func (p *ModuleCacheProvider) Load(ctx context.Context, modulePath string, version string) (ModuleArtifact, error) {
	if p == nil || p.cacheRoot == "" {
		return ModuleArtifact{}, fmt.Errorf("module cache provider が初期化されていません")
	}
	if module.Check(modulePath, version) != nil {
		return ModuleArtifact{}, fmt.Errorf("module path または version が不正です")
	}
	escapedPath, err := module.EscapePath(modulePath)
	if err != nil {
		return ModuleArtifact{}, fmt.Errorf("module path を escape できません")
	}
	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return ModuleArtifact{}, fmt.Errorf("module version を escape できません")
	}
	reader, err := openRepositoryReader(p.cacheRoot)
	if err != nil {
		return ModuleArtifact{}, fmt.Errorf("module cache root を開けません: %w", err)
	}
	defer func() { _ = reader.Close() }()
	base := path.Join("cache/download", escapedPath, "@v", escapedVersion)
	if err := ctx.Err(); err != nil {
		return ModuleArtifact{}, fmt.Errorf("module artifact の読取りが中止されました: %w", err)
	}
	zipRaw, err := reader.readRegularContext(ctx, base+".zip", MaximumModuleZipBytes)
	if err != nil {
		return ModuleArtifact{}, fmt.Errorf("module zip を読めません: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return ModuleArtifact{}, fmt.Errorf("module artifact の読取りが中止されました: %w", err)
	}
	goModRaw, err := reader.readRegularContext(ctx, base+".mod", MaximumModuleGoModBytes)
	if err != nil {
		return ModuleArtifact{}, fmt.Errorf("module go.mod を読めません: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return ModuleArtifact{}, fmt.Errorf("module artifact の読取りが中止されました: %w", err)
	}
	zipHashRaw, err := reader.readRegularContext(ctx, base+".ziphash", 128)
	if err != nil {
		return ModuleArtifact{}, fmt.Errorf("module ziphash を読めません: %w", err)
	}
	zipHash := string(zipHashRaw)
	if !moduleSumPattern.MatchString(zipHash) {
		return ModuleArtifact{}, fmt.Errorf("module ziphash の形式が不正です")
	}
	return ModuleArtifact{Zip: zipRaw, GoMod: goModRaw, ZipHash: zipHash}, nil
}
