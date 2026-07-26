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

func TestValidateArchive(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		format  string
		entries []testArchiveEntry
		wantErr string
	}{
		"tar.gz の単一実行ファイル": {
			format: "tar.gz",
			entries: []testArchiveEntry{{
				name: "japanese-law-mcp", content: "binary", typeflag: tar.TypeReg, mode: 0o755,
			}},
		},
		"zip の単一実行ファイル": {
			format: "zip",
			entries: []testArchiveEntry{{
				name: "japanese-law-mcp.exe", content: "binary", mode: 0o755,
			}},
		},
		"tar.gz のパストラバーサル": {
			format: "tar.gz",
			entries: []testArchiveEntry{{
				name: "../japanese-law-mcp", content: "binary", typeflag: tar.TypeReg, mode: 0o755,
			}},
			wantErr: "不正なパス",
		},
		"tar.gz のシンボリックリンク": {
			format: "tar.gz",
			entries: []testArchiveEntry{{
				name: "japanese-law-mcp", typeflag: tar.TypeSymlink, mode: 0o777,
			}},
			wantErr: "通常ファイル",
		},
		"tar.gz の実行権限なし": {
			format: "tar.gz",
			entries: []testArchiveEntry{{
				name: "japanese-law-mcp", content: "binary", typeflag: tar.TypeReg, mode: 0o644,
			}},
			wantErr: "実行権限",
		},
		"tar.gz の追加ファイル": {
			format: "tar.gz",
			entries: []testArchiveEntry{
				{name: "japanese-law-mcp", content: "binary", typeflag: tar.TypeReg, mode: 0o755},
				{name: "README.md", content: "extra", typeflag: tar.TypeReg, mode: 0o644},
			},
			wantErr: "一つだけ",
		},
		"zip の追加ディレクトリ": {
			format: "zip",
			entries: []testArchiveEntry{
				{name: "japanese-law-mcp.exe", content: "binary", mode: 0o755},
				{name: "docs/", mode: os.ModeDir | 0o755},
			},
			wantErr: "一つだけ",
		},
		"zip のシンボリックリンク": {
			format: "zip",
			entries: []testArchiveEntry{{
				name: "japanese-law-mcp.exe", content: "target", mode: os.ModeSymlink | 0o777,
			}},
			wantErr: "通常ファイル",
		},
	}
	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "artifact."+test.format)
			switch test.format {
			case "tar.gz":
				writeTestTarGz(t, path, test.entries)
			case "zip":
				writeTestZip(t, path, test.entries)
			default:
				t.Fatalf("未知の形式です: %s", test.format)
			}
			expected := "japanese-law-mcp"
			if test.format == "zip" {
				expected += ".exe"
			}
			err := validateArchive(path, test.format, expected)
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
}

func TestExtractZipBinary(t *testing.T) {
	t.Parallel()

	tempDirectory := t.TempDir()
	archive := filepath.Join(tempDirectory, "artifact.zip")
	writeTestZip(t, archive, []testArchiveEntry{{
		name: "japanese-law-mcp.exe", content: "binary", mode: 0o755,
	}})
	target := newReleaseTarget("1.2.3", "windows", "amd64", "zip")
	destinationDirectory := filepath.Join(tempDirectory, "output")
	if err := os.Mkdir(destinationDirectory, 0o700); err != nil {
		t.Fatalf("展開先を作成できません: %v", err)
	}
	destination, err := extractArchiveBinary(archive, target, destinationDirectory)
	if err != nil {
		t.Fatalf("extractArchiveBinary() のエラー = %v", err)
	}
	content, err := os.ReadFile(destination) //nolint:gosec // SOT-ENG-019: テスト専用 TempDir の展開結果を読む。
	if err != nil {
		t.Fatalf("展開した実行ファイルを読めません: %v", err)
	}
	if string(content) != "binary" {
		t.Fatalf("展開内容 = %q", content)
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
