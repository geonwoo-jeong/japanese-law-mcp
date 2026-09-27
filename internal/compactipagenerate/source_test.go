package compactipagenerate

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"golang.org/x/mod/sumdb/dirhash"
)

func TestExtractSourceRequiresVerifiedSameModuleBytes(t *testing.T) {
	identity := moduleIdentity{Path: modulePath, Version: "v1.0.0"}
	prefix := identity.Path + "@" + identity.Version + "/"
	source := syntheticSource(t)
	files := map[string][]byte{
		prefix + "ipa.dict": source.dictionary, prefix + "LICENSE": source.license,
		prefix + "NOTICE.txt": source.notice, prefix + "go.mod": []byte("module " + modulePath + "\n"),
	}
	var names []string
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	var err error
	identity.Sum, err = dirhash.Hash1(names, func(name string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(files[name])), nil })
	if err != nil {
		t.Fatal(err)
	}
	identity.GoModSum, err = dirhash.Hash1([]string{"go.mod"}, func(string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(files[prefix+"go.mod"])), nil })
	if err != nil {
		t.Fatal(err)
	}
	data := moduleZip(t, names, files)
	got, err := extractSource(data, identity)
	if err != nil {
		t.Fatal(err)
	}
	source.module = identity
	if !reflect.DeepEqual(got, source) {
		t.Fatal("SOT-ENG-051: 検証した module の入力を保持していません")
	}
	changed := identity
	changed.Sum = "h1:不一致"
	if _, err := extractSource(data, changed); err == nil {
		t.Fatal("SOT-ENG-051: checksum の異なる module を受理しました")
	}
	changed = identity
	changed.GoModSum = "h1:不一致"
	if _, err := extractSource(data, changed); err == nil {
		t.Fatal("SOT-ENG-051: go.mod の checksum 不一致を受理しました")
	}
	duplicate := append(append([]string{}, names...), names[0])
	if _, err := extractSource(moduleZip(t, duplicate, files), identity); err == nil {
		t.Fatal("SOT-ENG-051: 重複した module entry を受理しました")
	}
	files[prefix+"../ipa.dict"] = source.dictionary
	if _, err := extractSource(moduleZip(t, append(names, prefix+"../ipa.dict"), files), identity); err == nil {
		t.Fatal("SOT-ENG-051: 不正な module path を受理しました")
	}
}

func TestPinnedModuleMetadataMustExistBeforeDownload(t *testing.T) {
	repository := t.TempDir()
	mod := []byte("module example.invalid/test\n\ngo 1.25.0\n\nrequire " + modulePath + " v1.0.0\n")
	if err := os.WriteFile(filepath.Join(repository, "go.mod"), mod, 0600); err != nil {
		t.Fatal(err)
	}
	version, err := requiredVersion(repository)
	if err != nil || version != "v1.0.0" {
		t.Fatalf("SOT-ENG-051: go.mod の固定版を読めません: %v", err)
	}
	if _, err := readGoSum(repository, version); err == nil {
		t.Fatal("SOT-ENG-051: 固定 checksum のない module を受理しました")
	}
	sum := []byte(modulePath + " v1.0.0 h1:合成\n" + modulePath + " v1.0.0/go.mod h1:合成\n")
	if err := os.WriteFile(filepath.Join(repository, "go.sum"), sum, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readGoSum(repository, version); err != nil {
		t.Fatal(err)
	}
	mod = append(mod, []byte("replace "+modulePath+" => ../local\n")...)
	if err := os.WriteFile(filepath.Join(repository, "go.mod"), mod, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := requiredVersion(repository); err == nil {
		t.Fatal("SOT-ENG-051: 任意の置換 module を受理しました")
	}
}

func moduleZip(t *testing.T, names []string, files map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range names {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
