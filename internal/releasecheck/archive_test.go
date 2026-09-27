package releasecheck

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testArchiveEntry struct {
	name     string
	content  string
	typeflag byte
	mode     os.FileMode
}

func testArchiveNotices() archiveNotices {
	return archiveNotices{
		license: "合成した許諾文\n", notice: "合成した通知文\n",
		sdkLicense: "合成した SDK 許諾文\n", sdkPatch: "{\"patch\":\"合成した識別情報\"}\n",
	}
}

func testArchiveEntries(binaryName, binary string) []testArchiveEntry {
	notices := testArchiveNotices()
	return []testArchiveEntry{
		{name: binaryName, content: binary, typeflag: tar.TypeReg, mode: 0o755},
		{name: ipaLicenseName, content: notices.license, typeflag: tar.TypeReg, mode: 0o644},
		{name: ipaNoticeName, content: notices.notice, typeflag: tar.TypeReg, mode: 0o644},
		{name: sdkLicenseName, content: notices.sdkLicense, typeflag: tar.TypeReg, mode: 0o644},
		{name: sdkPatchName, content: notices.sdkPatch, typeflag: tar.TypeReg, mode: 0o644},
	}
}

func writeTestArchiveNotices(t *testing.T, repository string) {
	t.Helper()

	for relative, content := range testArchiveNoticeFiles() {
		name := filepath.Join(repository, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
			t.Fatalf("合成通知の directory を作成できません: %v", err)
		}
		if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
			t.Fatalf("合成通知を作成できません: %v", err)
		}
	}
}

func testArchiveNoticeFiles() map[string]string {
	notices := testArchiveNotices()
	return map[string]string{
		"internal/nlp/kagome/data/IPA-LICENSE":                             notices.license,
		"internal/nlp/kagome/data/IPA-NOTICE.txt":                          notices.notice,
		"third_party/modelcontextprotocol-go-sdk/LICENSE":                  notices.sdkLicense,
		"third_party/modelcontextprotocol-go-sdk-patch/MCP-SDK-LICENSE":    notices.sdkLicense,
		"third_party/modelcontextprotocol-go-sdk-patch/MCP-SDK-PATCH.json": notices.sdkPatch,
	}
}

func TestValidateArchive(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"tar.gz", "zip"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			binaryName := testArchiveBinaryName(format)
			for name, test := range archiveValidationCases(binaryName, format) {
				t.Run(name, func(t *testing.T) {
					t.Parallel()

					path := filepath.Join(t.TempDir(), "artifact."+format)
					writeTestArchive(t, path, format, test.entries)
					err := validateArchive(path, format, binaryName, testArchiveNotices())
					if test.wantErr == "" {
						if err != nil {
							t.Fatalf("validateArchive() のエラー = %v", err)
						}
						return
					}
					if err == nil || !strings.Contains(err.Error(), test.wantErr) {
						t.Fatalf("validateArchive() のエラー = %v, want %q", err, test.wantErr)
					}
				})
			}
		})
	}
}

type archiveValidationCase struct {
	entries []testArchiveEntry
	wantErr string
}

func archiveValidationCases(binaryName, format string) map[string]archiveValidationCase {
	entries := testArchiveEntries(binaryName, "binary")
	binary, license, notice, sdkLicense, sdkPatch := entries[0], entries[1], entries[2], entries[3], entries[4]
	changed := func(name, content string, typeflag byte, mode os.FileMode) []testArchiveEntry {
		return []testArchiveEntry{binary, {name: name, content: content, typeflag: typeflag, mode: mode}, notice, sdkLicense, sdkPatch}
	}
	tests := map[string]archiveValidationCase{
		"実行ファイルと通知":   {entries: entries},
		"通知が先頭":       {entries: []testArchiveEntry{notice, license, sdkPatch, sdkLicense, binary}},
		"実行ファイル不足":    {entries: []testArchiveEntry{license, notice, sdkLicense, sdkPatch}, wantErr: "二つが必要"},
		"許諾文不足":       {entries: []testArchiveEntry{binary, notice, sdkLicense, sdkPatch}, wantErr: "二つが必要"},
		"通知文不足":       {entries: []testArchiveEntry{binary, license, sdkLicense, sdkPatch}, wantErr: "二つが必要"},
		"実行ファイル重複":    {entries: []testArchiveEntry{binary, license, notice, sdkLicense, sdkPatch, binary}, wantErr: "重複"},
		"通知重複":        {entries: []testArchiveEntry{binary, license, notice, sdkLicense, sdkPatch, license}, wantErr: "重複"},
		"未知ファイル":      {entries: changed("README.md", "extra", tar.TypeReg, 0o644), wantErr: "予期しない"},
		"パストラバーサル":    {entries: changed("../IPA-LICENSE", license.content, tar.TypeReg, 0o644), wantErr: "不正なパス"},
		"絶対パス":        {entries: changed("/IPA-LICENSE", license.content, tar.TypeReg, 0o644), wantErr: "不正なパス"},
		"逆スラッシュ":      {entries: changed(`..\IPA-LICENSE`, license.content, tar.TypeReg, 0o644), wantErr: "不正なパス"},
		"通知シンボリックリンク": {entries: changed(ipaLicenseName, "", tar.TypeSymlink, os.ModeSymlink|0o777), wantErr: "通常ファイル"},
		"通知ディレクトリ":    {entries: changed(ipaLicenseName, "", tar.TypeDir, os.ModeDir|0o755), wantErr: "通常ファイル"},
		"空の通知":        {entries: changed(ipaLicenseName, "", tar.TypeReg, 0o644), wantErr: "原文と一致"},
		"異なる通知":       {entries: changed(ipaLicenseName, "別の合成文", tar.TypeReg, 0o644), wantErr: "原文と一致"},
		"通知上限超過":      {entries: changed(ipaLicenseName, strings.Repeat("x", maxNoticeBytes+1), tar.TypeReg, 0o644), wantErr: "大きすぎ"},
		"通知の実行権限":     {entries: changed(ipaLicenseName, license.content, tar.TypeReg, 0o755), wantErr: "実行権限"},
	}
	if format == "tar.gz" {
		tests["通知ハードリンク"] = archiveValidationCase{
			entries: changed(ipaLicenseName, "", tar.TypeLink, 0o644), wantErr: "通常ファイル",
		}
		binary.mode = 0o644
		tests["実行権限なし"] = archiveValidationCase{
			entries: []testArchiveEntry{binary, license, notice, sdkLicense, sdkPatch}, wantErr: "実行権限",
		}
	}
	addSDKArchiveValidationCases(tests, entries, format)
	return tests
}

func addSDKArchiveValidationCases(tests map[string]archiveValidationCase, entries []testArchiveEntry, format string) {
	for _, index := range []int{3, 4} {
		original := entries[index]
		changed := func(content string, typeflag byte, mode os.FileMode) []testArchiveEntry {
			result := append([]testArchiveEntry(nil), entries...)
			result[index] = testArchiveEntry{name: original.name, content: content, typeflag: typeflag, mode: mode}
			return result
		}
		missing := append([]testArchiveEntry(nil), entries[:index]...)
		missing = append(missing, entries[index+1:]...)
		duplicate := append([]testArchiveEntry(nil), entries...)
		duplicate = append(duplicate, original)
		tests[original.name+"不足"] = archiveValidationCase{entries: missing, wantErr: "二つが必要"}
		tests[original.name+"重複"] = archiveValidationCase{entries: duplicate, wantErr: "重複"}
		for name, test := range map[string]archiveValidationCase{
			"空":         {entries: changed("", tar.TypeReg, 0o644), wantErr: "原文と一致"},
			"内容不一致":     {entries: changed("異なる付属文書", tar.TypeReg, 0o644), wantErr: "原文と一致"},
			"上限超過":      {entries: changed(strings.Repeat("x", maxNoticeBytes+1), tar.TypeReg, 0o644), wantErr: "大きすぎ"},
			"実行権限":      {entries: changed(original.content, tar.TypeReg, 0o755), wantErr: "実行権限"},
			"シンボリックリンク": {entries: changed("", tar.TypeSymlink, os.ModeSymlink|0o777), wantErr: "通常ファイル"},
			"ディレクトリ":    {entries: changed("", tar.TypeDir, os.ModeDir|0o755), wantErr: "通常ファイル"},
		} {
			tests[original.name+name] = test
		}
		if format == "tar.gz" {
			tests[original.name+"ハードリンク"] = archiveValidationCase{
				entries: changed("", tar.TypeLink, 0o644), wantErr: "通常ファイル",
			}
		}
	}
}

func TestExtractArchiveBinaryByName(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"tar.gz", "zip"} {
		for name, order := range map[string][5]int{"先頭": {0, 1, 2, 3, 4}, "中間": {1, 2, 0, 3, 4}, "末尾": {4, 3, 2, 1, 0}} {
			t.Run(format+"/"+name, func(t *testing.T) {
				t.Parallel()

				tempDirectory := t.TempDir()
				archive := filepath.Join(tempDirectory, "artifact."+format)
				binaryName := testArchiveBinaryName(format)
				entries := testArchiveEntries(binaryName, "binary")
				ordered := make([]testArchiveEntry, len(order))
				for index, source := range order {
					ordered[index] = entries[source]
				}
				writeTestArchive(t, archive, format, ordered)
				target := newReleaseTarget("1.2.3", "darwin", "arm64", format)
				if format == "zip" {
					target = newReleaseTarget("1.2.3", "windows", "amd64", format)
				}
				destinationDirectory := t.TempDir()
				destination, err := extractArchiveBinary(archive, target, destinationDirectory, testArchiveNotices())
				if err != nil {
					t.Fatalf("extractArchiveBinary() のエラー = %v", err)
				}
				content, err := os.ReadFile(destination) //nolint:gosec // SOT-ENG-019: テスト専用 TempDir の展開結果を読む。
				if err != nil || string(content) != "binary" {
					t.Fatalf("展開内容 = %q, エラー = %v", content, err)
				}
				files, err := os.ReadDir(destinationDirectory)
				if err != nil || len(files) != 1 || files[0].Name() != binaryName {
					t.Fatalf("展開したファイル = %v, エラー = %v", files, err)
				}
			})
		}
	}
}

func TestLoadArchiveNotices(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	writeTestArchiveNotices(t, repository)
	notices, err := loadArchiveNotices(repository)
	if err != nil || notices != testArchiveNotices() {
		t.Fatalf("通知 = %#v, エラー = %v", notices, err)
	}
}

func TestLoadArchiveNoticesRejectsInvalidFiles(t *testing.T) {
	t.Parallel()

	for relative := range testArchiveNoticeFiles() {
		for _, mutation := range []string{"不足", "空", "上限超過", "ディレクトリ", "シンボリックリンク"} {
			t.Run(relative+"/"+mutation, func(t *testing.T) {
				t.Parallel()

				repository := t.TempDir()
				writeTestArchiveNotices(t, repository)
				name := filepath.Join(repository, filepath.FromSlash(relative))
				mutateArchiveNoticeFile(t, name, mutation)
				if _, err := loadArchiveNotices(repository); err == nil {
					t.Fatal("不正な付属文書の正本を受理しました")
				}
			})
		}
	}
}

func mutateArchiveNoticeFile(t *testing.T, name, mutation string) {
	t.Helper()

	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}
	switch mutation {
	case "空":
		if err := os.WriteFile(name, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	case "上限超過":
		if err := os.WriteFile(name, []byte(strings.Repeat("x", maxNoticeBytes+1)), 0o600); err != nil {
			t.Fatal(err)
		}
	case "ディレクトリ":
		if err := os.Mkdir(name, 0o700); err != nil {
			t.Fatal(err)
		}
	case "シンボリックリンク":
		target := name + ".target"
		if err := os.WriteFile(target, []byte("合成したリンク先"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, name); err != nil {
			t.Skipf("symlink を作成できません: %v", err)
		}
	}
}

func TestLoadArchiveNoticesRejectsSDKLicenseMismatch(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	writeTestArchiveNotices(t, repository)
	name := filepath.Join(repository, "third_party", "modelcontextprotocol-go-sdk-patch", sdkLicenseName)
	if err := os.WriteFile(name, []byte("異なる SDK 許諾文"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadArchiveNotices(repository); err == nil || !strings.Contains(err.Error(), "許諾原文と一致") {
		t.Fatalf("SDK 許諾文の不一致を拒否しませんでした: %v", err)
	}
}

func TestValidateArchiveRequiresCanonicalNotices(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "artifact.zip")
	writeTestZip(t, path, testArchiveEntries("japanese-law-mcp.exe", "binary"))
	if err := validateArchive(path, "zip", "japanese-law-mcp.exe", archiveNotices{}); err == nil {
		t.Fatal("正本なしで通知を受理しました")
	}
}

func TestValidateArchiveSizeBounds(t *testing.T) {
	t.Parallel()

	if err := validateArchiveEntry(ipaNoticeName, true, maxNoticeBytes, "binary"); err != nil {
		t.Fatalf("上限と同じ通知サイズを拒否しました: %v", err)
	}
	if err := validateArchiveEntry(ipaNoticeName, true, -1, "binary"); err == nil {
		t.Fatal("負のサイズを受理しました")
	}
	entry := &zip.File{FileHeader: zip.FileHeader{Name: "binary", UncompressedSize64: ^uint64(0)}}
	if err := validateZipEntry(entry, "binary", testArchiveNotices()); err == nil {
		t.Fatal("整数変換範囲を超える ZIP entry を受理しました")
	}
}

func testArchiveBinaryName(format string) string {
	if format == "zip" {
		return "japanese-law-mcp.exe"
	}
	return "japanese-law-mcp"
}

func writeTestArchive(t *testing.T, path, format string, entries []testArchiveEntry) {
	t.Helper()

	if format == "tar.gz" {
		writeTestTarGz(t, path, entries)
	} else {
		writeTestZip(t, path, entries)
	}
}

func writeTestTarGz(t *testing.T, path string, entries []testArchiveEntry) {
	t.Helper()

	file, err := os.Create(path) //nolint:gosec // SOT-ENG-019: テスト専用の TempDir 配下へ固定名で作成する。
	if err != nil {
		t.Fatalf("tar.gz を作成できません: %v", err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Mode:     int64(entry.mode.Perm()),
			Size:     int64(len(entry.content)),
			Typeflag: entry.typeflag,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("tar header を書き込めません: %v", err)
		}
		if entry.content != "" {
			if _, err := io.WriteString(tarWriter, entry.content); err != nil {
				t.Fatalf("tar entry を書き込めません: %v", err)
			}
		}
	}
	closeTestWriters(t, tarWriter, gzipWriter, file)
}

func writeTestZip(t *testing.T, path string, entries []testArchiveEntry) {
	t.Helper()

	file, err := os.Create(path) //nolint:gosec // SOT-ENG-019: テスト専用の TempDir 配下へ固定名で作成する。
	if err != nil {
		t.Fatalf("zip を作成できません: %v", err)
	}
	writer := zip.NewWriter(file)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Store}
		header.SetMode(entry.mode)
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("zip header を書き込めません: %v", err)
		}
		if _, err := io.WriteString(entryWriter, entry.content); err != nil {
			t.Fatalf("zip entry を書き込めません: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("zip writer を閉じられません: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("zip file を閉じられません: %v", err)
	}
}

func closeTestWriters(
	t *testing.T,
	tarWriter *tar.Writer,
	gzipWriter *gzip.Writer,
	file *os.File,
) {
	t.Helper()

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tar writer を閉じられません: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzip writer を閉じられません: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("tar.gz file を閉じられません: %v", err)
	}
}
