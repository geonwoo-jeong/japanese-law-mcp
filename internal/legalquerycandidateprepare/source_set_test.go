package legalquerycandidateprepare

import (
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerysourceclosure"
)

func TestBuildSemanticSourceSetは実MarkerSourceを固定する(t *testing.T) {
	if !useExactCandidateToolchain(t) {
		t.Skip("候補再現用 Go 環境がないため local では実行しません")
	}
	t.Setenv("GOOS", "linux")
	t.Setenv("GOARCH", "amd64")
	t.Setenv("GOAMD64", "v1")
	t.Setenv("CGO_ENABLED", "0")
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("candidate-evaluation-schema-v3-marker-content-binding: repository を解決できません: %v", err)
	}
	source, err := BuildSemanticSourceSet(t.Context(), repository)
	if err != nil {
		t.Fatalf("candidate-evaluation-schema-v3-marker-content-binding: source set を構築できません: %v", err)
	}
	want := map[string]bool{
		"internal/querypreprocess/candidate_case_failure.go": false,
		"internal/querypreprocess/preprocess.go":             false,
	}
	for _, file := range source.Files {
		if _, exists := want[file.Path]; exists {
			want[file.Path] = true
		}
	}
	for path, found := range want {
		if !found {
			t.Fatalf("candidate-evaluation-schema-v3-marker-content-binding: %s が source set にありません", path)
		}
	}
}

func TestProjectSemanticSourceSetはClosureの全Identityを保持する(t *testing.T) {
	t.Parallel()

	input := semanticSourceSetFixture{
		files: []legalquerysourceclosure.SourceFile{{
			Path:      "internal/application/legalquery/selector.go",
			RawSHA256: "0000000000000000000000000000000000000000000000000000000000000000",
		}},
		modules: []legalquerysourceclosure.ModuleDependency{{
			ModulePath:               "example.invalid/module",
			Version:                  "v1.0.0",
			ModuleZipSum:             "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
			ModuleZipRawSHA256:       "1111111111111111111111111111111111111111111111111111111111111111",
			ModuleZipByteLength:      10,
			ModuleZipEntryCount:      1,
			ModuleExpandedByteLength: 5,
			ModuleGoModSum:           "h1:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=",
			ModuleGoModRawSHA256:     "2222222222222222222222222222222222222222222222222222222222222222",
		}},
	}
	projected, err := projectSemanticSourceSet(input)
	if err != nil {
		t.Fatalf("candidate-evaluation-build-context-isolation: source set を投影できません: %v", err)
	}
	if projected.MainModulePath != "github.com/geonwoo-jeong/japanese-law-mcp" ||
		projected.GoLanguageVersion != "1.25.0" ||
		projected.GoToolchainVersion != "go1.26.5" ||
		projected.GOOS != "linux" || projected.GOARCH != "amd64" ||
		projected.GOAMD64 != "v1" || projected.CGOEnabled != 0 ||
		len(projected.Files) != 1 || len(projected.ModuleDependencies) != 1 ||
		len(projected.SourceSetSHA256) != 64 {
		t.Fatalf("candidate-evaluation-build-context-isolation: source set = %#v", projected)
	}
}

type semanticSourceSetFixture struct {
	files   []legalquerysourceclosure.SourceFile
	modules []legalquerysourceclosure.ModuleDependency
}

func (semanticSourceSetFixture) MainModulePath() string {
	return "github.com/geonwoo-jeong/japanese-law-mcp"
}
func (semanticSourceSetFixture) GoLanguageVersion() string  { return "1.25.0" }
func (semanticSourceSetFixture) GoToolchainVersion() string { return "go1.26.5" }
func (semanticSourceSetFixture) GoDebugSettings() []legalquerysourceclosure.GoDebugSetting {
	return []legalquerysourceclosure.GoDebugSetting{}
}
func (semanticSourceSetFixture) GOOS() string         { return "linux" }
func (semanticSourceSetFixture) GOARCH() string       { return "amd64" }
func (semanticSourceSetFixture) GOAMD64() string      { return "v1" }
func (semanticSourceSetFixture) GOEXPERIMENT() string { return "" }
func (semanticSourceSetFixture) CGOEnabled() int      { return 0 }
func (semanticSourceSetFixture) BuildTags() []string  { return []string{} }
func (f semanticSourceSetFixture) Files() []legalquerysourceclosure.SourceFile {
	return append([]legalquerysourceclosure.SourceFile(nil), f.files...)
}
func (f semanticSourceSetFixture) ModuleDependencies() []legalquerysourceclosure.ModuleDependency {
	return append([]legalquerysourceclosure.ModuleDependency(nil), f.modules...)
}
