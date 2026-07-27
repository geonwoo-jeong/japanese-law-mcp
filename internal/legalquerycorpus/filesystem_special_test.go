package legalquerycorpus

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestCorpusFilesystemはschemaFileのsymlinkと型違反を拒否する(
	t *testing.T,
) {
	t.Parallel()

	tests := map[string]func(*testing.T, filesystemReadTestLayout){
		"symlink": func(t *testing.T, layout filesystemReadTestLayout) {
			if err := os.Remove(layout.schemaPath); err != nil {
				t.Fatalf("schema file を削除できません: %v", err)
			}
			target := filepath.Join(t.TempDir(), "outside-schema.json")
			filesystemReadTestWriteFile(t, target, []byte(`{"outside":true}`))
			if err := os.Symlink(target, layout.schemaPath); err != nil {
				if errors.Is(err, os.ErrPermission) {
					t.Skipf("symlink を作成する権限がないため省略します: %v", err)
				}
				t.Fatalf("schema symlink を作成できません: %v", err)
			}
		},
		"directory": func(t *testing.T, layout filesystemReadTestLayout) {
			if err := os.Remove(layout.schemaPath); err != nil {
				t.Fatalf("schema file を削除できません: %v", err)
			}
			if err := os.Mkdir(layout.schemaPath, 0o700); err != nil {
				t.Fatalf("schema directory を作成できません: %v", err)
			}
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			layout := filesystemReadTestNewLayout(t)
			mutate(t, layout)
			filesystem := filesystemReadTestOpen(t, layout)
			if _, err := filesystem.readSchemaV1(
				context.Background(),
			); err == nil {
				t.Fatal("SOT-ENG-026: symlink または通常 file でない schema を受理した")
			}
		})
	}
}

func TestCorpusFilesystemはsocketFixtureを拒否する(t *testing.T) {
	t.Parallel()

	layout := filesystemReadTestNewLayout(t)
	socketPath := filepath.Join(
		layout.setPaths[ManifestSetDevelopment],
		"development-socket.json",
	)
	listener, err := net.ListenUnix("unix", &net.UnixAddr{
		Name: socketPath,
		Net:  "unix",
	})
	if err != nil {
		t.Skipf("Unix socket を作成できないため省略します: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	filesystem, err := openCorpusFilesystem(
		context.Background(),
		layout.repositoryRoot,
		layout.corpusPath,
	)
	if filesystem != nil {
		_ = filesystem.close()
	}
	if err == nil {
		t.Fatal("SOT-ENG-026: socket fixture を受理した")
	}
}
