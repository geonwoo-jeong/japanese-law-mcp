package legalquerysourceclosure

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestModuleCacheProviderReadsExactRawArtifacts(t *testing.T) {
	t.Parallel()
	cache := t.TempDir()
	writeSourceClosureFixture(t, cache, "cache/download/example.com/dependency/@v/v1.2.3.zip", "zip-byte")
	writeSourceClosureFixture(t, cache, "cache/download/example.com/dependency/@v/v1.2.3.mod", "module example.com/dependency\n")
	writeSourceClosureFixture(t, cache, "cache/download/example.com/dependency/@v/v1.2.3.ziphash", "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	provider, err := NewModuleCacheProvider(cache)
	if err != nil {
		t.Fatalf("NewModuleCacheProvider() error = %v", err)
	}
	artifact, err := provider.Load(context.Background(), "example.com/dependency", "v1.2.3")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if string(artifact.Zip) != "zip-byte" || string(artifact.GoMod) != "module example.com/dependency\n" || artifact.ZipHash != "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" {
		t.Fatalf("artifact = %#v", artifact)
	}
	artifact.Zip[0] = 'X'
	reloaded, err := provider.Load(context.Background(), "example.com/dependency", "v1.2.3")
	if err != nil || string(reloaded.Zip) != "zip-byte" {
		t.Fatalf("再読込み artifact = %q, error = %v", reloaded.Zip, err)
	}
}

func TestCommandToolchainListsOnlyFixedContext(t *testing.T) {
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Skip("Go command がありません")
	}
	goBinary, err = filepath.EvalSymlinks(goBinary)
	if err != nil {
		t.Fatal(err)
	}
	repository := t.TempDir()
	writeSourceClosureFixture(t, repository, "go.mod", "module example.com/main\n\ngo 1.25.0\n")
	writeSourceClosureFixture(t, repository, "internal/component/component.go", "package component\n")
	moduleCache := t.TempDir()
	buildCache := t.TempDir()
	temporaryDirectory := t.TempDir()
	toolchain, err := NewCommandToolchain(ToolchainInfrastructure{
		GoBinary:           goBinary,
		GoRoot:             testRuntimeGoRoot(t, goBinary),
		ModuleCache:        moduleCache,
		BuildCache:         buildCache,
		TemporaryDirectory: temporaryDirectory,
	})
	if err != nil {
		t.Fatalf("NewCommandToolchain() error = %v", err)
	}
	version, err := toolchain.Version(context.Background(), repository)
	if err != nil || !goToolchainVersionPattern.MatchString(version) {
		t.Fatalf("Version() = %q, %v", version, err)
	}
	output, err := toolchain.ListDependencies(context.Background(), ListRequest{
		RepositoryPath: repository,
		PackageRoots:   []string{"internal/component"},
		BuildContext:   FixedBuildContext(),
	})
	if err != nil {
		t.Fatalf("ListDependencies() error = %v", err)
	}
	raw, err := io.ReadAll(output)
	if err != nil {
		t.Fatal(err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("command close error = %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("go list output が空です")
	}
}

func TestNewLocalBuilderCreatesConcreteDependencies(t *testing.T) {
	builder, err := NewLocalBuilder(context.Background())
	if err != nil {
		t.Fatalf("NewLocalBuilder() error = %v", err)
	}
	if builder.Toolchain == nil || builder.Modules == nil {
		t.Fatal("local builder の concrete dependency がありません")
	}
}

func TestNewLocalBuilderはHOMEなしの閉じた環境だけで構成する(t *testing.T) {
	const childProcessName = "candidate-evaluation-build-context-isolation"

	if os.Args[0] == childProcessName {
		builder, err := NewLocalBuilder(t.Context())
		if err != nil {
			t.Fatalf("candidate-evaluation-build-context-isolation: NewLocalBuilder() error = %v", err)
		}
		toolchain, ok := builder.Toolchain.(*CommandToolchain)
		if !ok {
			t.Fatalf("candidate-evaluation-build-context-isolation: toolchain = %T", builder.Toolchain)
		}
		goBinary, err := exec.LookPath("go")
		if err != nil {
			t.Fatalf("candidate-evaluation-build-context-isolation: Go command を解決できません: %v", err)
		}
		goBinary, err = filepath.EvalSymlinks(goBinary)
		if err != nil {
			t.Fatalf("candidate-evaluation-build-context-isolation: Go command の symlink を解決できません: %v", err)
		}
		wantInfrastructure := ToolchainInfrastructure{
			GoBinary:           goBinary,
			GoRoot:             os.Getenv("GOROOT"),
			ModuleCache:        os.Getenv("GOMODCACHE"),
			BuildCache:         os.Getenv("GOCACHE"),
			TemporaryDirectory: os.Getenv("TMPDIR"),
		}
		if toolchain.infrastructure != wantInfrastructure {
			t.Fatalf("candidate-evaluation-build-context-isolation: infrastructure = %#v, want %#v", toolchain.infrastructure, wantInfrastructure)
		}
		modules, ok := builder.Modules.(*ModuleCacheProvider)
		if !ok {
			t.Fatalf("candidate-evaluation-build-context-isolation: module provider = %T", builder.Modules)
		}
		if modules.cacheRoot != wantInfrastructure.ModuleCache {
			t.Fatalf("candidate-evaluation-build-context-isolation: module cache = %q, want %q", modules.cacheRoot, wantInfrastructure.ModuleCache)
		}
		return
	}

	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Skip("Go command がありません")
	}
	goBinary, err = filepath.EvalSymlinks(goBinary)
	if err != nil {
		t.Fatal(err)
	}
	goRoot := testRuntimeGoRoot(t, goBinary)
	moduleCache := t.TempDir()
	buildCache := t.TempDir()
	temporaryDirectory := t.TempDir()
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("test executable を解決できません: %v", err)
	}
	command := exec.CommandContext( //nolint:gosec // SOT-ENG-038: os.Executable で得た同じ test binary と固定 test 名だけを子 process へ渡す。
		t.Context(),
		testBinary,
		"-test.run=^TestNewLocalBuilderはHOMEなしの閉じた環境だけで構成する$",
	)
	command.Args[0] = childProcessName
	command.Env = []string{
		"PATH=" + filepath.Dir(goBinary),
		"GOROOT=" + goRoot,
		"GOMODCACHE=" + moduleCache,
		"GOCACHE=" + buildCache,
		"TMPDIR=" + temporaryDirectory,
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("candidate-evaluation-build-context-isolation: HOME なしの閉じた環境で builder を構成できません: %v\n%s", err, output)
	}
}

func TestNewLocalBuilderはHOMEなしの閉じた環境でCachePath欠落を拒否する(t *testing.T) {
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Skip("Go command がありません")
	}
	goBinary, err = filepath.EvalSymlinks(goBinary)
	if err != nil {
		t.Fatal(err)
	}
	for _, missing := range []string{"GOROOT", "GOMODCACHE", "GOCACHE", "TMPDIR"} {
		t.Run(missing, func(t *testing.T) {
			infrastructureRoot := t.TempDir()
			t.Setenv("GOROOT", infrastructureRoot)
			t.Setenv("GOMODCACHE", infrastructureRoot)
			t.Setenv("GOCACHE", infrastructureRoot)
			t.Setenv("TMPDIR", infrastructureRoot)
			t.Setenv(missing, "")

			_, err := newClosedEnvironmentBuilder(goBinary)
			if err == nil || !strings.Contains(err.Error(), missing) {
				t.Fatalf(
					"candidate-evaluation-build-context-isolation: %s 欠落の error = %v",
					missing,
					err,
				)
			}
		})
	}
}

func TestNewLocalBuilderは取消済みContextを拒否する(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewLocalBuilder(ctx); err == nil {
		t.Fatal("candidate-evaluation-build-context-isolation: 取消済み discovery を受理しました")
	}
}

func TestModuleCacheProviderRejectsSymlinkSegment(t *testing.T) {
	t.Parallel()
	cache := t.TempDir()
	outside := t.TempDir()
	writeSourceClosureFixture(t, outside, "v1.2.3.zip", "zip-byte")
	writeSourceClosureFixture(t, outside, "v1.2.3.mod", "module example.com/dependency\n")
	writeSourceClosureFixture(t, outside, "v1.2.3.ziphash", "h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	parent := filepath.Join(cache, "cache", "download", "example.com", "dependency")
	if err := os.MkdirAll(parent, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(parent, "@v")); err != nil {
		t.Fatal(err)
	}
	provider, err := NewModuleCacheProvider(cache)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Load(context.Background(), "example.com/dependency", "v1.2.3"); err == nil {
		t.Fatal("module cache の symlink segment を受理しました")
	}
}

func testRuntimeGoRoot(t *testing.T, goBinary string) string {
	t.Helper()
	command := exec.CommandContext(
		t.Context(),
		goBinary,
		"env",
		"GOROOT",
	)
	output, err := command.Output()
	if err != nil {
		t.Fatalf("go env GOROOT を取得できません: %v", err)
	}
	return strings.TrimSpace(string(output))
}
