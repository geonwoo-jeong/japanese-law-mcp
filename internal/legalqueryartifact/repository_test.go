package legalqueryartifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRepositoryAndOpenChildRejectSymlink(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repository := filepath.Join(root, "repo")
	if err := os.MkdirAll(filepath.Join(repository, "child"), 0o750); err != nil {
		t.Fatalf("test repository を作れません: %v", err)
	}
	handle, err := OpenRepository(repository)
	if err != nil {
		t.Fatalf("repository root を開けません: %v", err)
	}
	t.Cleanup(func() { _ = handle.Close() })

	child, err := handle.OpenChild("child")
	if err != nil {
		t.Fatalf("child directory を開けません: %v", err)
	}
	_ = child.Close()

	link := filepath.Join(repository, "link")
	if err := os.Symlink(filepath.Join(repository, "child"), link); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("symlink を作成できない環境です")
		}
		t.Fatalf("symlink を作れません: %v", err)
	}
	if _, err := handle.OpenChild("link"); err == nil {
		t.Fatal("symlink child を受理しました")
	}
}

func TestReadRegularAndDirectoryBounds(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	repository := filepath.Join(root, "repo")
	if err := os.MkdirAll(repository, 0o750); err != nil {
		t.Fatalf("test repository を作れません: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repository, "ok.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("test file を書けません: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repository, "big.json"), []byte("12345"), 0o600); err != nil {
		t.Fatalf("test file を書けません: %v", err)
	}

	handle, err := OpenRepository(repository)
	if err != nil {
		t.Fatalf("repository root を開けません: %v", err)
	}
	t.Cleanup(func() { _ = handle.Close() })

	data, err := handle.ReadRegular("ok.json", 8)
	if err != nil {
		t.Fatalf("regular file を読めません: %v", err)
	}
	if string(data) != "{}\n" {
		t.Fatalf("data = %q", data)
	}
	if _, err := handle.ReadRegular("big.json", 4); err == nil {
		t.Fatal("size 超過 file を受理しました")
	}

	entries, err := handle.ReadDirectory(8, 16)
	if err != nil {
		t.Fatalf("directory を列挙できません: %v", err)
	}
	if len(entries) != 2 || entries[0].Name() != "big.json" || entries[1].Name() != "ok.json" {
		t.Fatalf("entries = %#v", entries)
	}
	if _, err := handle.ReadDirectory(1, 16); err == nil {
		t.Fatal("entry 数超過を受理しました")
	}
	if _, err := handle.ReadDirectory(8, 4); err == nil {
		t.Fatal("合計 size 超過を受理しました")
	}
}

func TestValidateSingleSegment(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"a/b", ".", "../x", ""} {
		if err := validateSingleSegment(name); err == nil {
			t.Fatalf("%q を受理しました", name)
		}
	}
	if err := validateSingleSegment("default-1.json"); err != nil {
		t.Fatalf("単一 segment を拒否しました: %v", err)
	}
}
