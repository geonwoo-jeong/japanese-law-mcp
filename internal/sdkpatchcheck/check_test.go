package sdkpatchcheck

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/mod/sumdb/dirhash"
)

func Test固定SDK複製と許可差分を照合する(t *testing.T) {
	repository, manifest, archive := writeFixture(t)
	called := false
	err := runCheck(t.Context(), repository, func(ctx context.Context, actual string, identity moduleIdentity) ([]byte, error) {
		called = true
		if ctx != t.Context() || actual != repository || identity != manifest.Module {
			t.Fatal("SOT-ENG-020: 固定入力または context が取得処理へ渡されていません")
		}
		return archive, nil
	})
	if err != nil || !called {
		t.Fatalf("SOT-ENG-020: 正しい SDK copy を拒否しました: %v", err)
	}
}

func TestSDK複製の欠落追加改変と許諾不一致を拒否する(t *testing.T) {
	for _, name := range []string{"欠落", "改変", "追加", "余分なdirectory", "rootが通常file", "patch改変", "許諾改変", "symlink", "親symlink"} {
		t.Run(name, func(t *testing.T) {
			repository, manifest, archive := writeFixture(t)
			sdk := filepath.Join(repository, filepath.FromSlash(sourceDirectory))
			switch name {
			case "欠落":
				mustRemove(t, filepath.Join(sdk, "LICENSE"))
			case "改変":
				mustWrite(t, filepath.Join(sdk, "LICENSE"), []byte("別の許諾文"))
			case "追加":
				mustWrite(t, filepath.Join(sdk, "extra.go"), []byte("package extra"))
			case "余分なdirectory":
				if err := os.Mkdir(filepath.Join(sdk, "extra"), 0700); err != nil {
					t.Fatal(err)
				}
			case "rootが通常file":
				if err := os.Rename(sdk, sdk+"-退避"); err != nil {
					t.Fatal(err)
				}
				mustWrite(t, sdk, []byte("directory ではない入力"))
			case "patch改変":
				mustWrite(t, filepath.Join(repository, patchDirectory, manifest.PatchFile), []byte("別の patch"))
			case "許諾改変":
				mustWrite(t, filepath.Join(repository, patchDirectory, "MCP-SDK-LICENSE"), []byte("別の許諾文"))
			case "symlink":
				mustRemove(t, filepath.Join(sdk, "LICENSE"))
				if err := os.Symlink(filepath.Join(repository, patchDirectory, "MCP-SDK-LICENSE"), filepath.Join(sdk, "LICENSE")); err != nil {
					t.Skipf("symlink を作れません: %v", err)
				}
			case "親symlink":
				external := filepath.Join(repository, "sdk-退避")
				if err := os.Rename(sdk, external); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(external, sdk); err != nil {
					t.Skipf("symlink を作れません: %v", err)
				}
			}
			if err := checkFixture(t, repository, manifest, archive); err == nil {
				t.Fatal("SOT-ENG-020: 不正な SDK copy を受理しました")
			}
		})
	}
}

func TestSDK原ZIPと全file一覧の不一致を拒否する(t *testing.T) {
	_, initial, archive := writeFixture(t)
	for _, name := range []string{"ZIP原hash", "ZIPのh1", "go.mod原hash", "go.modのh1", "file件数", "filepath", "filehash", "filebytes"} {
		t.Run(name, func(t *testing.T) {
			manifest := initial
			manifest.UpstreamFiles = slices.Clone(initial.UpstreamFiles)
			switch name {
			case "ZIP原hash":
				manifest.Module.ZipSHA256 = strings.Repeat("0", 64)
			case "ZIPのh1":
				manifest.Module.ZipSum = "h1:別の値"
			case "go.mod原hash":
				manifest.Module.GoModSHA256 = strings.Repeat("0", 64)
			case "go.modのh1":
				manifest.Module.GoModSum = "h1:別の値"
			case "file件数":
				manifest.UpstreamFiles = manifest.UpstreamFiles[:1]
			case "filepath":
				manifest.UpstreamFiles[0].Path = "missing"
			case "filehash":
				manifest.UpstreamFiles[0].SHA256 = strings.Repeat("0", 64)
			case "filebytes":
				manifest.UpstreamFiles[0].Bytes++
			}
			if _, err := extractArchive(archive, manifest); err == nil {
				t.Fatal("SOT-ENG-020: 元 SDK と異なる identity を受理しました")
			}
		})
	}
}

func TestSDKmanifestの閉じた構造と許可範囲(t *testing.T) {
	_, manifest, _ := writeFixture(t)
	raw := encodeTestManifest(t, manifest)
	if _, err := decodeManifest(raw); err != nil {
		t.Fatal(err)
	}
	for _, input := range [][]byte{
		append(slices.Clone(raw), []byte("{}")...),
		bytes.Replace(raw, []byte(`"schemaVersion":1`), []byte(`"schemaVersion":1,"schemaVersion":1`), 1),
		bytes.Replace(raw, []byte(`"schemaVersion":1`), []byte(`"schemaVersion":1,"unknown":1`), 1),
		[]byte("null"), nil,
	} {
		if _, err := decodeManifest(input); err == nil {
			t.Fatal("SOT-ENG-020: 閉じていない manifest を受理しました")
		}
	}
	for _, mutate := range []func(*patchManifest){
		func(m *patchManifest) { m.SchemaVersion++ },
		func(m *patchManifest) { m.Module.Version = "v1.6.2" },
		func(m *patchManifest) { m.Module.Path = "example.com/other" },
		func(m *patchManifest) { m.SourceDirectory = "../outside" },
		func(m *patchManifest) { m.PatchFile = "../patch" },
		func(m *patchManifest) { m.PatchID = "別の patch" },
		func(m *patchManifest) { m.Changes[0].Path = "mcp/server.go" },
		func(m *patchManifest) { m.Changes[0].BeforeSHA256 = strings.Repeat("0", 64) },
		func(m *patchManifest) { m.Additions[0].Path = "mcp/extra.go" },
		func(m *patchManifest) { m.UpstreamFiles[1] = m.UpstreamFiles[0] },
		func(m *patchManifest) { m.UpstreamFiles[0].Path = "../outside" },
	} {
		copyManifest, err := decodeManifest(raw)
		if err != nil {
			t.Fatal(err)
		}
		mutate(&copyManifest)
		if _, err := decodeManifest(encodeTestManifest(t, copyManifest)); err == nil {
			t.Fatal("SOT-ENG-020: manifest の許可範囲外の変更を受理しました")
		}
	}
}

func TestSDKversion限定replaceを維持する(t *testing.T) {
	valid := []byte(fixtureGoMod())
	if err := verifyModuleReplacement(valid); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{
		strings.Replace(fixtureGoMod(), " v1.6.1 =>", " =>", 1),
		strings.Replace(fixtureGoMod(), "=> ./third_party/modelcontextprotocol-go-sdk", "=> ../outside", 1),
		strings.Replace(fixtureGoMod(), "require "+ModulePath+" v1.6.1", "require "+ModulePath+" v1.6.2", 1),
		"module example.com/test\n",
	} {
		if err := verifyModuleReplacement([]byte(input)); err == nil {
			t.Fatal("SOT-ENG-020: SDK の固定版以外を受理しました")
		}
	}
}

func TestSDK照合の中断と取得失敗(t *testing.T) {
	//nolint:staticcheck // SOT-ENG-052: nil context を拒否する境界契約を直接確認する。
	if err := Run(nil, "."); err == nil {
		t.Fatal("nil context を受理しました")
	}
	if err := Run(t.Context(), ""); err == nil {
		t.Fatal("空 repository を受理しました")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := Run(ctx, "."); !errors.Is(err, context.Canceled) {
		t.Fatalf("中断が伝わりません: %v", err)
	}
	repository, _, _ := writeFixture(t)
	failure := errors.New("取得できない固定入力")
	err := runCheck(t.Context(), repository, func(context.Context, string, moduleIdentity) ([]byte, error) { return nil, failure })
	if !errors.Is(err, failure) {
		t.Fatalf("取得失敗を保存しません: %v", err)
	}
}

func TestSDK取得環境はworkspaceと可変設定を引き継がない(t *testing.T) {
	actual := metadataEnvironment([]string{"PATH=保持", "GOPROXY=off", "GOMODCACHE=保持", "GOWORK=../別project", "GOENV=設定", "GOFLAGS=-mod=mod", "GOTOOLCHAIN=auto", "GO111MODULE=off"}, true)
	want := []string{"PATH=保持", "GOPROXY=off", "GOMODCACHE=保持", "GOENV=off", "GOWORK=off", "GOTOOLCHAIN=local", "GOFLAGS=-mod=readonly -p=1", "GO111MODULE=on"}
	if !slices.Equal(actual, want) {
		t.Fatalf("SOT-ENG-020: 元 SDK 取得環境の固定が不正です: %v", actual)
	}
	isolated := metadataEnvironment([]string{"GOPROXY=off", "GOFLAGS=任意"}, false)
	if !slices.Contains(isolated, "GOPROXY=off") || !slices.Contains(isolated, "GOFLAGS=-mod=mod -p=1") || slices.Contains(isolated, "GOFLAGS=任意") {
		t.Fatal("SOT-ENG-020: 一時 module の取得環境が固定されていません")
	}
}

func TestSDK原ZIPの危険なentryを拒否する(t *testing.T) {
	_, manifest, _ := writeFixture(t)
	for _, name := range []string{"../outside", "other/module@v1.6.1/a", ModulePath + "@" + UpstreamVersion + "/../outside"} {
		var buffer bytes.Buffer
		writer := zip.NewWriter(&buffer)
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("入力")); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		manifest.Module.ZipSHA256 = digest(buffer.Bytes())
		if _, err := extractArchive(buffer.Bytes(), manifest); err == nil {
			t.Fatal("SOT-ENG-020: 元 ZIP の危険な path を受理しました")
		}
	}
}

func checkFixture(t *testing.T, repository string, manifest patchManifest, archive []byte) error {
	t.Helper()
	root, err := os.OpenRoot(repository)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	return verifyCopy(t.Context(), root, manifest, archive)
}

func fixtureGoMod() string {
	return "module example.com/test\n\ngo 1.25.0\n\nrequire " + ModulePath + " " + UpstreamVersion + "\n\nreplace " + ModulePath + " " + UpstreamVersion + " => ./" + sourceDirectory + "\n"
}

func writeFixture(t *testing.T) (string, patchManifest, []byte) {
	t.Helper()
	repository := t.TempDir()
	files := map[string][]byte{"LICENSE": []byte("元の許諾文\n"), "go.mod": []byte("module " + ModulePath + "\n\ngo 1.25.0\n"), implementationPath: []byte("package json\n")}
	names := []string{"LICENSE", "go.mod", implementationPath}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	prefix := ModulePath + "@" + UpstreamVersion + "/"
	fullNames := make([]string, 0, len(names))
	manifest := patchManifest{SchemaVersion: 1, SourceDirectory: sourceDirectory, PatchFile: "json-unmarshal.patch", Reason: "合成 fixture の照合", Module: moduleIdentity{Path: ModulePath, Version: UpstreamVersion, OriginCommit: strings.Repeat("1", 40)}}
	for _, name := range names {
		entry, err := writer.Create(prefix + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(files[name]); err != nil {
			t.Fatal(err)
		}
		fullNames = append(fullNames, prefix+name)
		manifest.UpstreamFiles = append(manifest.UpstreamFiles, fileDigest{Path: name, Bytes: int64(len(files[name])), SHA256: digest(files[name])})
		mustWrite(t, filepath.Join(repository, sourceDirectory, filepath.FromSlash(name)), files[name])
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	manifest.Module.ZipSHA256 = digest(buffer.Bytes())
	var err error
	manifest.Module.ZipSum, err = dirhash.Hash1(fullNames, func(name string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(files[strings.TrimPrefix(name, prefix)])), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest.Module.GoModSum, err = dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(files["go.mod"])), nil })
	if err != nil {
		t.Fatal(err)
	}
	manifest.Module.GoModSHA256 = digest(files["go.mod"])
	changed, added, patch := []byte("package json\n// 許可差分\n"), []byte("package json\n// 差分試験\n"), []byte("適用する固定差分\n")
	manifest.Changes = []changedFile{{Path: implementationPath, BeforeSHA256: digest(files[implementationPath]), AfterSHA256: digest(changed), Bytes: int64(len(changed))}}
	manifest.Additions = []fileDigest{{Path: equivalenceTestPath, SHA256: digest(added), Bytes: int64(len(added))}}
	manifest.PatchSHA256 = digest(patch)
	manifest.PatchID = "sha256-" + manifest.PatchSHA256
	mustWrite(t, filepath.Join(repository, sourceDirectory, implementationPath), changed)
	mustWrite(t, filepath.Join(repository, sourceDirectory, equivalenceTestPath), added)
	mustWrite(t, filepath.Join(repository, patchDirectory, manifest.PatchFile), patch)
	mustWrite(t, filepath.Join(repository, patchDirectory, "MCP-SDK-LICENSE"), files["LICENSE"])
	mustWrite(t, filepath.Join(repository, patchDirectory, manifestName), encodeTestManifest(t, manifest))
	mustWrite(t, filepath.Join(repository, "go.mod"), []byte(fixtureGoMod()))
	return repository, manifest, buffer.Bytes()
}

func encodeTestManifest(t *testing.T, value patchManifest) []byte {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func mustWrite(t *testing.T, filename string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func mustRemove(t *testing.T, filename string) {
	t.Helper()
	if err := os.Remove(filename); err != nil {
		t.Fatal(err)
	}
}
