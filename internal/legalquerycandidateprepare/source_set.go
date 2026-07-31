package legalquerycandidateprepare

import (
	"context"
	"fmt"
	"math"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerysourceclosure"
)

var semanticComponentRoots = []string{
	"internal/querypreprocess",
	"internal/queryprofile/core",
	"internal/queryprofile/judicialcases",
	"internal/application/legalquery",
}

type semanticSourceIdentity interface {
	MainModulePath() string
	GoLanguageVersion() string
	GoToolchainVersion() string
	GoDebugSettings() []legalquerysourceclosure.GoDebugSetting
	GOOS() string
	GOARCH() string
	GOAMD64() string
	GOEXPERIMENT() string
	CGOEnabled() int
	BuildTags() []string
	Files() []legalquerysourceclosure.SourceFile
	ModuleDependencies() []legalquerysourceclosure.ModuleDependency
}

// BuildSemanticSourceSet は、固定 component roots の Go dependency closure を構築する。
func BuildSemanticSourceSet(
	ctx context.Context,
	repositoryRoot string,
) (legalquerycandidateeval.SemanticSourceSet, error) {
	if err := checkPrepareContext(ctx); err != nil {
		return legalquerycandidateeval.SemanticSourceSet{}, err
	}
	builder, err := legalquerysourceclosure.NewLocalBuilder(ctx)
	if err != nil {
		return legalquerycandidateeval.SemanticSourceSet{}, err
	}
	closure, err := builder.Build(ctx, repositoryRoot, semanticComponentRoots)
	if err != nil {
		return legalquerycandidateeval.SemanticSourceSet{}, err
	}
	return projectSemanticSourceSet(closure)
}

func projectSemanticSourceSet(
	source semanticSourceIdentity,
) (legalquerycandidateeval.SemanticSourceSet, error) {
	files := source.Files()
	projectedFiles := make([]legalquerycandidateeval.FileDigest, 0, len(files))
	for _, file := range files {
		projectedFiles = append(projectedFiles, legalquerycandidateeval.FileDigest{
			Path: file.Path, RawSHA256: file.RawSHA256,
		})
	}
	modules, err := projectModuleDependencies(source.ModuleDependencies())
	if err != nil {
		return legalquerycandidateeval.SemanticSourceSet{}, err
	}
	debug := source.GoDebugSettings()
	projectedDebug := make([]legalquerycandidateeval.GoDebugSetting, 0, len(debug))
	for _, setting := range debug {
		projectedDebug = append(projectedDebug, legalquerycandidateeval.GoDebugSetting{
			Name: setting.Name, Value: setting.Value,
		})
	}
	buildTags := source.BuildTags()
	result := legalquerycandidateeval.SemanticSourceSet{
		MainModulePath:     source.MainModulePath(),
		GoLanguageVersion:  source.GoLanguageVersion(),
		GoToolchainVersion: source.GoToolchainVersion(),
		GoDebugSettings:    projectedDebug,
		GOOS:               source.GOOS(),
		GOARCH:             source.GOARCH(),
		GOAMD64:            source.GOAMD64(),
		GOEXPERIMENT:       source.GOEXPERIMENT(),
		CGOEnabled:         source.CGOEnabled(),
		BuildTags:          append(make([]string, 0, len(buildTags)), buildTags...),
		Files:              projectedFiles,
		ModuleDependencies: modules,
	}
	result.SourceSetSHA256, err =
		legalquerycandidateeval.CanonicalSourceSetSHA256(result)
	if err != nil {
		return legalquerycandidateeval.SemanticSourceSet{}, err
	}
	return result, nil
}

func projectModuleDependencies(
	values []legalquerysourceclosure.ModuleDependency,
) ([]legalquerycandidateeval.ModuleDependency, error) {
	result := make([]legalquerycandidateeval.ModuleDependency, 0, len(values))
	for _, value := range values {
		if value.ModuleZipByteLength > math.MaxInt ||
			value.ModuleExpandedByteLength > math.MaxInt {
			return nil, fmt.Errorf("module dependency の byte 長が int 上限を超えています")
		}
		result = append(result, legalquerycandidateeval.ModuleDependency{
			ModulePath:               value.ModulePath,
			Version:                  value.Version,
			ModuleZipSum:             value.ModuleZipSum,
			ModuleZipRawSHA256:       value.ModuleZipRawSHA256,
			ModuleZipByteLength:      int(value.ModuleZipByteLength),
			ModuleZipEntryCount:      value.ModuleZipEntryCount,
			ModuleExpandedByteLength: int(value.ModuleExpandedByteLength),
			ModuleGoModSum:           value.ModuleGoModSum,
			ModuleGoModRawSHA256:     value.ModuleGoModRawSHA256,
		})
	}
	return result, nil
}
