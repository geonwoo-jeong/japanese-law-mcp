package compactipagenerate

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ikawaha/kagome-dict/dict"
)

func TestPOSClassOrderAndEmptyValues(t *testing.T) {
	table := dict.POSTable{NameList: []string{"", "名詞", "一般"}, POSs: []dict.POS{{1, 2}, {}, {1, 2}, {0}}}
	classes, ids, err := classifyPOS(table)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(classes, []dict.POS{{1, 2}, {}, {0}}) || !reflect.DeepEqual(ids, []uint16{0, 1, 0, 2}) {
		t.Fatal("SOT-ENG-051: 品詞分類が最初の出現順と一致しません")
	}
	data, count, err := encodePOS(table)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 || binary.LittleEndian.Uint32(data[8:12]) != 4 || !bytes.Equal(data[len(data)-8:], []byte{0, 0, 1, 0, 0, 0, 2, 0}) {
		t.Fatal("SOT-ENG-051: 元の語順に分類参照を保存していません")
	}
	for _, invalid := range []dict.POSTable{
		{}, {POSs: []dict.POS{{0}}}, {NameList: []string{"名詞"}, POSs: []dict.POS{{1}}},
		{NameList: []string{"\xff"}, POSs: []dict.POS{{0}}},
		{NameList: []string{strings.Repeat("x", 65536)}, POSs: []dict.POS{{0}}},
	} {
		if _, _, err := encodePOS(invalid); err == nil {
			t.Fatal("SOT-ENG-051: 不正な元品詞表を受理しました")
		}
	}
}

func TestTransformPreservesRawEntriesAndNotices(t *testing.T) {
	source := syntheticSource(t)
	generated, err := buildArtifacts(source)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := buildArtifacts(source)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(generated, repeated) {
		t.Fatal("SOT-ENG-051: 同じ固定入力から再現できません")
	}
	before := openTestZip(t, source.dictionary)
	after := openTestZip(t, generated.files["ipa-tokenization.dict"])
	var index int
	for _, original := range before.File {
		if original.Name == dict.POSDictFileName || original.Name == dict.ContentDictFileName {
			continue
		}
		copied := after.File[index]
		if original.Name != copied.Name {
			t.Fatal("SOT-ENG-051: entry の順序が変わりました")
		}
		originalRaw, err := original.OpenRaw()
		if err != nil {
			t.Fatal(err)
		}
		copiedRaw, err := copied.OpenRaw()
		if err != nil {
			t.Fatal(err)
		}
		var a, b bytes.Buffer
		if _, err := a.ReadFrom(originalRaw); err != nil {
			t.Fatal(err)
		}
		if _, err := b.ReadFrom(copiedRaw); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(a.Bytes(), b.Bytes()) {
			t.Fatal("SOT-ENG-051: 保持 entry が再圧縮されました")
		}
		index++
	}
	if !bytes.Equal(generated.files["IPA-LICENSE"], source.license) || !bytes.Equal(generated.files["IPA-NOTICE.txt"], source.notice) {
		t.Fatal("SOT-ENG-051: 上流の権利表示が変更されました")
	}
}

func TestSourceArchiveRejectsInvalidStructure(t *testing.T) {
	valid := openTestZip(t, syntheticSource(t).dictionary)
	variants := [][]*zip.File{
		valid.File[:len(valid.File)-1], append(append([]*zip.File{}, valid.File...), valid.File[0]),
		append(append([]*zip.File{}, valid.File...), &zip.File{FileHeader: zip.FileHeader{Name: "../unknown"}}),
	}
	for _, files := range variants {
		if err := validateSourceParts(&zip.Reader{File: files}); err == nil {
			t.Fatal("SOT-ENG-051: 不正な元辞書の構成を受理しました")
		}
	}
}

func TestCheckReproducesBytesAndRetainsGenerationToolchain(t *testing.T) {
	expected, err := buildArtifacts(syntheticSource(t))
	if err != nil {
		t.Fatal(err)
	}
	expected.manifest.Generator = generatorIdentity{Path: "cmd/compact-ipa-generate", Sources: []fileDigest{digest("internal/compactipagenerate/example.go", []byte("合成入力"))}, Toolchain: "go1.25.0"}
	repository := t.TempDir()
	writeTestArtifacts(t, repository, expected)
	expected.manifest.Generator.Toolchain = "go1.26.7"
	if err := checkArtifacts(repository, expected); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repository, dataDirectory, "ipa-pos.bin")
	if err := os.WriteFile(path, []byte("変更"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkArtifacts(repository, expected); err == nil {
		t.Fatal("SOT-ENG-051: 変更した生成 byte を受理しました")
	}
}

func TestManifestRejectsUnknownDuplicateAndMismatchedFields(t *testing.T) {
	expected, err := buildArtifacts(syntheticSource(t))
	if err != nil {
		t.Fatal(err)
	}
	expected.manifest.Generator = generatorIdentity{Path: "cmd/compact-ipa-generate", Sources: []fileDigest{}, Toolchain: "go1.26.6"}
	raw, err := encodeManifest(expected.manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{
		append([]byte(`{"unknown":0,`), raw[1:]...), append([]byte(`{"formatVersion":1,`), raw[1:]...),
		bytes.Replace(raw, []byte(`"formatVersion": 1`), []byte(`"formatVersion": 2`), 1),
	} {
		if _, err := decodeManifest(data); err == nil {
			t.Fatal("SOT-ENG-051: 閉じていない manifest を受理しました")
		}
	}
	mutations := []func(*manifest){
		func(m *manifest) { m.Outputs[0].Path = "../outside" }, func(m *manifest) { m.Words++ },
		func(m *manifest) { m.Module.Sum = "h1:不一致" }, func(m *manifest) { m.Generator.Sources = []fileDigest{digest("/absolute", nil)} },
	}
	for _, mutate := range mutations {
		repository := t.TempDir()
		writeTestArtifacts(t, repository, expected)
		var changed manifest
		if err := json.Unmarshal(raw, &changed); err != nil {
			t.Fatal(err)
		}
		mutate(&changed)
		data, err := encodeManifest(changed)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repository, dataDirectory, manifestName), data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := checkArtifacts(repository, expected); err == nil {
			t.Fatal("SOT-ENG-051: 由来と矛盾する manifest を受理しました")
		}
	}
}

func TestRunRejectsMissingBoundaryInputs(t *testing.T) {
	if err := Run(context.Background(), Options{}); err == nil {
		t.Fatal("SOT-ENG-051: repository のない生成を受理しました")
	}
}

func syntheticSource(t *testing.T) sourceData {
	t.Helper()
	table := dict.POSTable{NameList: []string{"", "名詞", "一般"}, POSs: []dict.POS{{1, 2}, {1}, {1, 2}}}
	var pos bytes.Buffer
	if _, err := table.WriteTo(&pos); err != nil {
		t.Fatal(err)
	}
	names := []string{dict.MorphDictFileName, dict.ContentMetaFileName, dict.ContentDictFileName, dict.POSDictFileName, dict.IndexDictFileName, dict.ConnectionDictFileName, dict.CharDefDictFileName, dict.UnkDictFileName, dict.DictInfoFileName}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range names {
		entry, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err != nil {
			t.Fatal(err)
		}
		data := []byte("合成辞書部分:" + name)
		if name == dict.POSDictFileName {
			data = pos.Bytes()
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return sourceData{module: moduleIdentity{Path: modulePath, Version: "v1.0.0", Sum: "h1:合成", GoModSum: "h1:合成"}, dictionary: buffer.Bytes(), license: []byte("合成許諾文\n"), notice: []byte("合成通知文\n")}
}

func openTestZip(t *testing.T, data []byte) *zip.Reader {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	return reader
}

func writeTestArtifacts(t *testing.T, repository string, expected artifactSet) {
	t.Helper()
	encoded, err := encodeManifest(expected.manifest)
	if err != nil {
		t.Fatal(err)
	}
	files := make(map[string][]byte, len(expected.files)+1)
	for name, data := range expected.files {
		files[name] = data
	}
	files[manifestName] = encoded
	if err := replaceArtifacts(context.Background(), repository, files); err != nil {
		t.Fatal(err)
	}
}

func TestCancelledGenerationPreservesAdoptedFiles(t *testing.T) {
	repository := t.TempDir()
	files := map[string][]byte{"IPA-LICENSE": []byte("合成保存値")}
	if err := replaceArtifacts(context.Background(), repository, files); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	changed := map[string][]byte{"IPA-LICENSE": []byte("変更値")}
	if err := replaceArtifacts(ctx, repository, changed); err == nil {
		t.Fatal("SOT-ENG-010: 取消した生成を保存しました")
	}
	got, err := readRepositoryFile(repository, dataDirectory+"/IPA-LICENSE")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, files["IPA-LICENSE"]) {
		t.Fatal("SOT-ENG-051: 取消後に採用済み集合が変更されました")
	}
}

func TestGenerationRejectsOutputAncestorOutsideRepository(t *testing.T) {
	repository, outside := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(repository, "internal", "nlp"), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(outside, "data"), 0750); err != nil {
		t.Fatal(err)
	}
	protected := filepath.Join(outside, "data", "IPA-LICENSE")
	if err := os.WriteFile(protected, []byte("合成保存値"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repository, "internal", "nlp", "kagome")); err != nil {
		t.Skipf("合成 symlink を作成できません: %v", err)
	}
	files := map[string][]byte{"IPA-LICENSE": []byte("変更値")}
	if err := replaceArtifacts(context.Background(), repository, files); err == nil {
		t.Fatal("SOT-ENG-051: repository 外の生成先を受理しました")
	}
	root, err := os.OpenRoot(outside)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	}()
	got, err := root.ReadFile("data/IPA-LICENSE")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "合成保存値" {
		t.Fatal("SOT-ENG-051: repository 外の file を変更しました")
	}
}
