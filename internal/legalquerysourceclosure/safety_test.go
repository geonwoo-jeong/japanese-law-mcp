package legalquerysourceclosure

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleArchiveRejectsUnsafeEntries(t *testing.T) {
	t.Parallel()
	tests := map[string][]zipFixtureEntry{
		"traversal": {{name: "example.com/dependency@v1.2.3/../outside.go", content: "package outside\n"}},
		"case collision": {
			{name: "example.com/dependency@v1.2.3/A.go", content: "package dependency\n"},
			{name: "example.com/dependency@v1.2.3/a.go", content: "package dependency\n"},
		},
		"symlink":      {{name: "example.com/dependency@v1.2.3/link", content: "target", mode: os.ModeSymlink | 0o777}},
		"wrong prefix": {{name: "example.com/other@v1.2.3/file.go", content: "package other\n"}},
	}
	for name, entries := range tests {
		name, entries := name, entries
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			raw := makeCustomModuleZip(t, entries)
			if _, _, _, err := inspectModuleZip(context.Background(), "example.com/dependency", "v1.2.3", raw); err == nil {
				t.Fatal("unsafe module zip entry を受理しました")
			}
		})
	}
}

func TestModuleArtifactRejectsUnboundCacheIdentity(t *testing.T) {
	t.Parallel()
	archive := makeModuleZip(t, "example.com/dependency", "v1.2.3", map[string]string{"dependency.go": "package dependency\n"})
	goMod := []byte("module example.com/dependency\n")
	identity := listedModule{
		path:     "example.com/dependency",
		version:  "v1.2.3",
		zipSum:   moduleZipH1ForTest(t, archive),
		goModSum: singleFileH1ForTest("go.mod", goMod),
	}
	if _, err := inspectModuleArtifact(context.Background(), identity, ModuleArtifact{
		Zip: archive, GoMod: goMod, ZipHash: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
	}); err == nil {
		t.Fatal("go list と一致しない cache ziphash を受理しました")
	}
	if _, err := inspectModuleArtifact(context.Background(), identity, ModuleArtifact{
		Zip: archive, GoMod: []byte("module example.com/other\n"), ZipHash: identity.zipSum,
	}); err == nil {
		t.Fatal("module path が異なる go.mod を受理しました")
	}
}

func TestBuilderRejectsBrokenGoListStream(t *testing.T) {
	t.Parallel()
	repository := t.TempDir()
	writeSourceClosureFixture(t, repository, "go.mod", "module example.com/main\n\ngo 1.25.0\n")
	writeSourceClosureFixture(t, repository, "internal/component/component.go", "package component\n")
	tests := map[string]string{
		"malformed":        `{`,
		"package error":    `{"ImportPath":"example.com/main/internal/component","Error":{}}`,
		"missing module":   `{"ImportPath":"example.com/main/internal/component","Dir":"` + filepath.Join(repository, "internal/component") + `"}`,
		"standard module":  `{"ImportPath":"fmt","Standard":true,"Module":{"Path":"example.com/dependency","Version":"v1.2.3"}}`,
		"duplicate import": goListPackageJSON(repository, "example.com/main/internal/component", "internal/component", []string{"component.go"}, nil) + "\n" + goListPackageJSON(repository, "example.com/main/internal/component", "internal/component", []string{"component.go"}, nil),
	}
	for name, output := range tests {
		name, output := name, output
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			builder := Builder{Toolchain: &fakeToolchain{version: "go1.26.5", output: []byte(output)}}
			if _, err := builder.Build(context.Background(), repository, []string{"internal/component"}); err == nil {
				t.Fatal("不正な go list stream を受理しました")
			}
		})
	}
}

func TestClosedHelpersRejectInvalidInputs(t *testing.T) {
	t.Parallel()
	if _, err := NewCommandToolchain(ToolchainInfrastructure{}); err == nil {
		t.Fatal("空の toolchain infrastructure を受理しました")
	}
	if err := (BuildContext{}).validate(); err == nil {
		t.Fatal("固定値ではない build context を受理しました")
	}
	if err := VerifyFiles(context.Background(), t.TempDir(), []SourceFile{
		{Path: "a.go", RawSHA256: strings.Repeat("a", 64)},
		{Path: "a.go", RawSHA256: strings.Repeat("a", 64)},
	}); err == nil {
		t.Fatal("重複 source path を受理しました")
	}
	if err := VerifyFiles(context.Background(), t.TempDir(), []SourceFile{{Path: "a.go", RawSHA256: "bad"}}); err == nil {
		t.Fatal("不正な source digest を受理しました")
	}

	var stderr cappedBuffer
	input := bytes.Repeat([]byte{'x'}, maximumCommandErrorBytes+5)
	if written, err := stderr.Write(input); err != nil || written != len(input) || !strings.HasSuffix(stderr.String(), "…") {
		t.Fatalf("cappedBuffer.Write() = %d, %v, %q", written, err, stderr.String())
	}
	if got := commandFailure("operation", errors.New("failure"), "").Error(); !strings.Contains(got, "operation") {
		t.Fatalf("commandFailure() = %q", got)
	}
	if got := commandFailure("operation", errors.New("failure"), "detail").Error(); !strings.Contains(got, "detail") {
		t.Fatalf("commandFailure(stderr) = %q", got)
	}
}

func TestCommandAndCacheBoundariesFailClosed(t *testing.T) {
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Skip("Go command がありません")
	}
	goBinary, err = filepath.EvalSymlinks(goBinary)
	if err != nil {
		t.Fatal(err)
	}
	goRoot := testRuntimeGoRoot(t, goBinary)
	toolchain, err := NewCommandToolchain(ToolchainInfrastructure{
		GoBinary:           goBinary,
		GoRoot:             goRoot,
		ModuleCache:        t.TempDir(),
		BuildCache:         t.TempDir(),
		TemporaryDirectory: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	validDirectory := t.TempDir()
	invalidDirectory := filepath.Join(validDirectory, "missing")
	invalidInfrastructures := []ToolchainInfrastructure{
		{GoBinary: goBinary, GoRoot: invalidDirectory, ModuleCache: validDirectory, BuildCache: validDirectory, TemporaryDirectory: validDirectory},
		{GoBinary: goBinary, GoRoot: goRoot, ModuleCache: invalidDirectory, BuildCache: validDirectory, TemporaryDirectory: validDirectory},
		{GoBinary: goBinary, GoRoot: goRoot, ModuleCache: validDirectory, BuildCache: invalidDirectory, TemporaryDirectory: validDirectory},
		{GoBinary: goBinary, GoRoot: goRoot, ModuleCache: validDirectory, BuildCache: validDirectory, TemporaryDirectory: invalidDirectory},
	}
	for _, infrastructure := range invalidInfrastructures {
		if _, err := NewCommandToolchain(infrastructure); err == nil {
			t.Fatal("不正な infrastructure directory を受理しました")
		}
	}
	if _, err := toolchain.ListDependencies(context.Background(), ListRequest{BuildContext: FixedBuildContext()}); err == nil {
		t.Fatal("空の package roots を受理しました")
	}
	if _, err := toolchain.ListDependencies(context.Background(), ListRequest{PackageRoots: []string{"internal/component"}}); err == nil {
		t.Fatal("固定値ではない build context を受理しました")
	}
	if _, err := toolchain.ListDependencies(context.Background(), ListRequest{PackageRoots: []string{"../outside"}, BuildContext: FixedBuildContext()}); err == nil {
		t.Fatal("不正な package root を受理しました")
	}
	if _, err := toolchain.Version(context.Background(), filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("存在しない working directory を受理しました")
	}
	var nilToolchain *CommandToolchain
	if _, err := nilToolchain.Version(context.Background(), t.TempDir()); err == nil {
		t.Fatal("未初期化 toolchain を受理しました")
	}
	if _, err := nilToolchain.ListDependencies(context.Background(), ListRequest{}); err == nil {
		t.Fatal("未初期化 toolchain の list request を受理しました")
	}

	if _, err := NewModuleCacheProvider(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("存在しない module cache root を受理しました")
	}
	var nilProvider *ModuleCacheProvider
	if _, err := nilProvider.Load(context.Background(), "example.com/dependency", "v1.2.3"); err == nil {
		t.Fatal("未初期化 module provider を受理しました")
	}
	cache := t.TempDir()
	provider, err := NewModuleCacheProvider(cache)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Load(context.Background(), "example.com/dependency", "v1.2.3"); err == nil {
		t.Fatal("欠落した module zip を受理しました")
	}
	if _, err := provider.Load(context.Background(), "../invalid", "v1.2.3"); err == nil {
		t.Fatal("不正な module identity を受理しました")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Load(cancelled, "example.com/dependency", "v1.2.3"); err == nil {
		t.Fatal("cancel 済み module 読取りを継続しました")
	}
	writeSourceClosureFixture(t, cache, "cache/download/example.com/dependency/@v/v1.2.3.zip", "zip")
	writeSourceClosureFixture(t, cache, "cache/download/example.com/dependency/@v/v1.2.3.mod", "module example.com/dependency\n")
	writeSourceClosureFixture(t, cache, "cache/download/example.com/dependency/@v/v1.2.3.ziphash", "invalid")
	if _, err := provider.Load(context.Background(), "example.com/dependency", "v1.2.3"); err == nil {
		t.Fatal("不正な module ziphash を受理しました")
	}
	if _, err := openRepositoryReader(""); err == nil {
		t.Fatal("空の repository root を受理しました")
	}
}

type zipFixtureEntry struct {
	name    string
	content string
	mode    os.FileMode
}

func makeCustomModuleZip(t *testing.T, entries []zipFixtureEntry) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, fixture := range entries {
		header := &zip.FileHeader{Name: fixture.name, Method: zip.Deflate}
		if fixture.mode != 0 {
			header.SetMode(fixture.mode)
		}
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, fixture.content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
