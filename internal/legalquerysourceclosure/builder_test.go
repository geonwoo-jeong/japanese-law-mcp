package legalquerysourceclosure

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestBuilderBuildsFixedSemanticSourceClosure(t *testing.T) {
	t.Parallel()
	repository := t.TempDir()
	writeSourceClosureFixture(t, repository, "go.mod", "module example.com/main\n\ngo 1.25.0\n\ngodebug panicnil=1\ngodebug asynctimerchan=0\n")
	writeSourceClosureFixture(t, repository, "internal/component/component.go", "package component\n")
	writeSourceClosureFixture(t, repository, "internal/shared/shared.go", "package shared\n")
	writeSourceClosureFixture(t, repository, "internal/component/assets/prompt.txt", "prompt\n")

	moduleZip := makeModuleZip(t, "example.com/dependency", "v1.2.3", map[string]string{
		"dependency.go": "package dependency\n",
		"LICENSE":       "license\n",
	})
	moduleMod := []byte("module example.com/dependency\n\ngo 1.24\n")
	zipSum := moduleZipH1ForTest(t, moduleZip)
	modSum := singleFileH1ForTest("go.mod", moduleMod)
	listOutput := strings.Join([]string{
		goListPackageJSON(repository, "example.com/main/internal/shared", "internal/shared", []string{"shared.go"}, nil),
		goListPackageJSON(repository, "example.com/main/internal/component", "internal/component", []string{"component.go"}, []string{"assets/prompt.txt"}),
		`{"ImportPath":"example.com/dependency","Dir":"/module-cache/example.com/dependency@v1.2.3","Module":{"Path":"example.com/dependency","Version":"v1.2.3","Sum":"` + zipSum + `","GoModSum":"` + modSum + `"}}`,
		`{"ImportPath":"fmt","Standard":true}`,
	}, "\n")

	builder := Builder{
		Toolchain: &fakeToolchain{version: "go1.26.5", output: []byte(listOutput)},
		Modules: fakeModuleProvider{artifacts: map[string]ModuleArtifact{
			"example.com/dependency@v1.2.3": {Zip: moduleZip, GoMod: moduleMod, ZipHash: zipSum},
		}},
	}
	set, err := builder.Build(context.Background(), repository, []string{"internal/component", "internal/component"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if set.MainModulePath() != "example.com/main" || set.GoLanguageVersion() != "1.25.0" || set.GoToolchainVersion() != "go1.26.5" {
		t.Fatalf("module/toolchain identity = %q %q %q", set.MainModulePath(), set.GoLanguageVersion(), set.GoToolchainVersion())
	}
	if set.GOOS() != "linux" || set.GOARCH() != "amd64" || set.GOAMD64() != "v1" || set.GOEXPERIMENT() != "" || set.CGOEnabled() != 0 || len(set.BuildTags()) != 0 {
		t.Fatalf("build context が固定値ではありません: %+v", set)
	}
	if got := set.GoDebugSettings(); !slices.Equal(got, []GoDebugSetting{{Name: "asynctimerchan", Value: "0"}, {Name: "panicnil", Value: "1"}}) {
		t.Fatalf("godebug = %#v", got)
	}
	wantPaths := []string{
		"internal/component/assets/prompt.txt",
		"internal/component/component.go",
		"internal/shared/shared.go",
	}
	files := set.Files()
	if got := sourceFilePaths(files); !slices.Equal(got, wantPaths) {
		t.Fatalf("files = %#v, want %#v", got, wantPaths)
	}
	for _, file := range files {
		raw, readErr := os.ReadFile(filepath.Join(repository, filepath.FromSlash(file.Path)))
		if readErr != nil {
			t.Fatal(readErr)
		}
		if file.RawSHA256 != sha256HexForTest(raw) {
			t.Fatalf("%s rawSha256 = %q", file.Path, file.RawSHA256)
		}
	}
	modules := set.ModuleDependencies()
	if len(modules) != 1 {
		t.Fatalf("moduleDependencies = %#v", modules)
	}
	dependency := modules[0]
	if dependency.ModulePath != "example.com/dependency" || dependency.Version != "v1.2.3" ||
		dependency.ModuleZipSum != zipSum || dependency.ModuleZipRawSHA256 != sha256HexForTest(moduleZip) ||
		dependency.ModuleZipByteLength != int64(len(moduleZip)) || dependency.ModuleZipEntryCount != 2 ||
		dependency.ModuleExpandedByteLength != int64(len("package dependency\n")+len("license\n")) ||
		dependency.ModuleGoModSum != modSum || dependency.ModuleGoModRawSHA256 != sha256HexForTest(moduleMod) {
		t.Fatalf("module identity = %#v", dependency)
	}

	// 呼出し元が返却 slice を変更しても、閉じた集合は変化しない。
	files[0].Path = "changed"
	modules[0].ModulePath = "changed"
	debug := set.GoDebugSettings()
	debug[0].Name = "changed"
	if set.Files()[0].Path != wantPaths[0] || set.ModuleDependencies()[0].ModulePath != "example.com/dependency" || set.GoDebugSettings()[0].Name != "asynctimerchan" {
		t.Fatal("SourceSet の返却 slice が内部状態を共有しています")
	}
}

func TestBuilderRejectsUnclosedSourceAndModuleInputs(t *testing.T) {
	t.Parallel()
	baseRepository := func(t *testing.T) string {
		t.Helper()
		repository := t.TempDir()
		writeSourceClosureFixture(t, repository, "go.mod", "module example.com/main\n\ngo 1.25.0\n")
		writeSourceClosureFixture(t, repository, "internal/component/component.go", "package component\n")
		return repository
	}

	t.Run("repository traversal", func(t *testing.T) {
		repository := baseRepository(t)
		builder := Builder{Toolchain: &fakeToolchain{version: "go1.26.5"}, Modules: fakeModuleProvider{}}
		if _, err := builder.Build(context.Background(), repository, []string{"../outside"}); err == nil {
			t.Fatal("repository 外 package root を受理しました")
		}
	})

	t.Run("missing component root", func(t *testing.T) {
		repository := baseRepository(t)
		builder := Builder{Toolchain: &fakeToolchain{version: "go1.26.5", output: []byte(`{"ImportPath":"fmt","Standard":true}`)}, Modules: fakeModuleProvider{}}
		if _, err := builder.Build(context.Background(), repository, []string{"internal/component"}); err == nil {
			t.Fatal("component package の欠落を受理しました")
		}
	})

	t.Run("local source outside repository", func(t *testing.T) {
		repository := baseRepository(t)
		outside := t.TempDir()
		builder := Builder{Toolchain: &fakeToolchain{version: "go1.26.5", output: []byte(fmt.Sprintf(
			`{"ImportPath":"example.com/main/internal/component","Dir":%q,"Module":{"Path":"example.com/main","Main":true},"GoFiles":["component.go"]}`,
			outside,
		))}, Modules: fakeModuleProvider{}}
		if _, err := builder.Build(context.Background(), repository, []string{"internal/component"}); err == nil {
			t.Fatal("repository 外 local source を受理しました")
		}
	})

	t.Run("external module replace", func(t *testing.T) {
		repository := baseRepository(t)
		output := goListPackageJSON(repository, "example.com/main/internal/component", "internal/component", []string{"component.go"}, nil) + "\n" +
			`{"ImportPath":"example.com/dependency","Module":{"Path":"example.com/dependency","Version":"v1.2.3","Sum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","GoModSum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","Replace":{"Path":"../dependency"}}}`
		builder := Builder{Toolchain: &fakeToolchain{version: "go1.26.5", output: []byte(output)}, Modules: fakeModuleProvider{}}
		if _, err := builder.Build(context.Background(), repository, []string{"internal/component"}); err == nil {
			t.Fatal("replace 済み external module を受理しました")
		}
	})

	t.Run("module archive checksum", func(t *testing.T) {
		repository := baseRepository(t)
		archive := makeModuleZip(t, "example.com/dependency", "v1.2.3", map[string]string{"dependency.go": "package dependency\n"})
		goMod := []byte("module example.com/dependency\n")
		output := goListPackageJSON(repository, "example.com/main/internal/component", "internal/component", []string{"component.go"}, nil) + "\n" +
			`{"ImportPath":"example.com/dependency","Module":{"Path":"example.com/dependency","Version":"v1.2.3","Sum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","GoModSum":"` + singleFileH1ForTest("go.mod", goMod) + `"}}`
		builder := Builder{
			Toolchain: &fakeToolchain{version: "go1.26.5", output: []byte(output)},
			Modules:   fakeModuleProvider{artifacts: map[string]ModuleArtifact{"example.com/dependency@v1.2.3": {Zip: archive, GoMod: goMod, ZipHash: "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="}}},
		}
		if _, err := builder.Build(context.Background(), repository, []string{"internal/component"}); err == nil {
			t.Fatal("module zip sum の不一致を受理しました")
		}
	})
}

func TestVerifyFilesUsesClosedRepositoryBytes(t *testing.T) {
	t.Parallel()
	repository := t.TempDir()
	writeSourceClosureFixture(t, repository, "internal/component/component.go", "package component\n")
	raw := []byte("package component\n")
	want := []SourceFile{{Path: "internal/component/component.go", RawSHA256: sha256HexForTest(raw)}}
	if err := VerifyFiles(context.Background(), repository, want); err != nil {
		t.Fatalf("VerifyFiles() error = %v", err)
	}

	writeSourceClosureFixture(t, repository, "internal/component/component.go", "package changed\n")
	if err := VerifyFiles(context.Background(), repository, want); err == nil {
		t.Fatal("変更された source byte を受理しました")
	}
	if err := VerifyFiles(context.Background(), repository, []SourceFile{{Path: "b.go", RawSHA256: strings.Repeat("a", 64)}, {Path: "a.go", RawSHA256: strings.Repeat("b", 64)}}); err == nil {
		t.Fatal("未整列 file 集合を受理しました")
	}
}

func writeSourceClosureFixture(t *testing.T, root string, relative string, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

type fakeToolchain struct {
	version  string
	output   []byte
	err      error
	requests []ListRequest
}

func (f *fakeToolchain) Version(context.Context, string) (string, error) { return f.version, f.err }

func (f *fakeToolchain) ListDependencies(_ context.Context, request ListRequest) (io.ReadCloser, error) {
	f.requests = append(f.requests, request)
	if f.err != nil {
		return nil, f.err
	}
	return io.NopCloser(bytes.NewReader(f.output)), nil
}

type fakeModuleProvider struct {
	artifacts map[string]ModuleArtifact
}

func (f fakeModuleProvider) Load(_ context.Context, modulePath string, version string) (ModuleArtifact, error) {
	artifact, ok := f.artifacts[modulePath+"@"+version]
	if !ok {
		return ModuleArtifact{}, fmt.Errorf("module がありません")
	}
	return ModuleArtifact{Zip: slices.Clone(artifact.Zip), GoMod: slices.Clone(artifact.GoMod), ZipHash: artifact.ZipHash}, nil
}

func goListPackageJSON(repository string, importPath string, relativeDir string, goFiles []string, embedFiles []string) string {
	return fmt.Sprintf(
		`{"ImportPath":%q,"Dir":%q,"Module":{"Path":%q,"Main":true},"GoFiles":[%s],"EmbedFiles":[%s]}`,
		importPath,
		filepath.Join(repository, filepath.FromSlash(relativeDir)),
		"example.com/main",
		quotedStrings(goFiles),
		quotedStrings(embedFiles),
	)
}

func quotedStrings(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = fmt.Sprintf("%q", value)
	}
	return strings.Join(quoted, ",")
}

func sourceFilePaths(files []SourceFile) []string {
	paths := make([]string, len(files))
	for index, file := range files {
		paths[index] = file.Path
	}
	return paths
}

func makeModuleZip(t *testing.T, modulePath string, version string, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		entry, err := writer.Create(modulePath + "@" + version + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func moduleZipH1ForTest(t *testing.T, raw []byte) string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	type namedDigest struct {
		name   string
		digest string
	}
	digests := make([]namedDigest, 0, len(reader.File))
	for _, file := range reader.File {
		opened, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		content, readErr := io.ReadAll(opened)
		_ = opened.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		digest := sha256.Sum256(content)
		digests = append(digests, namedDigest{name: file.Name, digest: hex.EncodeToString(digest[:])})
	}
	slices.SortFunc(digests, func(left, right namedDigest) int { return strings.Compare(left.name, right.name) })
	aggregate := sha256.New()
	for _, digest := range digests {
		_, _ = fmt.Fprintf(aggregate, "%s  %s\n", digest.digest, digest.name)
	}
	return "h1:" + base64.StdEncoding.EncodeToString(aggregate.Sum(nil))
}

func singleFileH1ForTest(name string, raw []byte) string {
	digest := sha256.Sum256(raw)
	aggregate := sha256.Sum256([]byte(hex.EncodeToString(digest[:]) + "  " + name + "\n"))
	return "h1:" + base64.StdEncoding.EncodeToString(aggregate[:])
}

func sha256HexForTest(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}
