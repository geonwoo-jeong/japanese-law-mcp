package legalquerycorpus

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const filesystemPathTestManifestContent = `{"artifactKind":"corpus_manifest","schemaVersion":1}`

func TestCorpusFilesystemは相対絶対pathの正常layoutを開く(t *testing.T) {
	tests := []struct {
		name           string
		relativeRepo   bool
		absoluteCorpus bool
	}{
		{name: "absolute repo relative corpus"},
		{name: "absolute repo absolute corpus", absoluteCorpus: true},
		{name: "relative repo relative corpus", relativeRepo: true},
		{
			name:           "relative repo absolute corpus",
			relativeRepo:   true,
			absoluteCorpus: true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			layout := filesystemPathTestCreateLayout(t, test.relativeRepo)
			corpusInput := filepath.Join(
				"testdata",
				"legalquery",
				"corpus-v1",
			)
			if test.absoluteCorpus {
				var err error
				corpusInput, err = filepath.Abs(layout.corpusRoot)
				if err != nil {
					t.Fatalf("絶対 corpus path を作成できません: %v", err)
				}
			}

			filesystem, err := openCorpusFilesystem(
				context.Background(),
				layout.repositoryRoot,
				corpusInput,
			)
			if err != nil {
				t.Fatalf("SOT-ENG-026: 正常な corpus filesystem error = %v", err)
			}
			t.Cleanup(func() {
				if err := filesystem.close(); err != nil {
					t.Errorf("SOT-ENG-026: corpus filesystem close error = %v", err)
				}
			})

			manifest := filesystem.manifestBytes()
			if string(manifest) != filesystemPathTestManifestContent {
				t.Fatalf("SOT-ENG-026: manifest bytes = %q", manifest)
			}
			manifest[0] = 'X'
			if string(filesystem.manifestBytes()) !=
				filesystemPathTestManifestContent {
				t.Fatal("SOT-ENG-026: manifest bytes が getter から変更された")
			}
		})
	}
}

func TestCorpusFilesystemはfixture名を昇順の複製で返す(t *testing.T) {
	t.Parallel()

	layout := filesystemPathTestCreateLayout(t, false)
	filesystem, err := openCorpusFilesystem(
		context.Background(),
		layout.repositoryRoot,
		layout.corpusRoot,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: corpus filesystem error = %v", err)
	}
	t.Cleanup(func() {
		if err := filesystem.close(); err != nil {
			t.Errorf("SOT-ENG-026: corpus filesystem close error = %v", err)
		}
	})

	tests := []struct {
		kind ManifestSetKind
		want []string
	}{
		{
			kind: ManifestSetDevelopment,
			want: []string{"development-a.json", "development-z.json"},
		},
		{
			kind: ManifestSetHoldout,
			want: []string{"holdout-a.json", "holdout-z.json"},
		},
		{
			kind: ManifestSetExecution,
			want: []string{"execution-a.json", "execution-z.json"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(string(test.kind), func(t *testing.T) {
			got, err := filesystem.fixtureFileNames(test.kind)
			if err != nil {
				t.Fatalf("SOT-ENG-026: fixture 列挙 error = %v", err)
			}
			if !filesystemPathTestEqualStrings(got, test.want) {
				t.Fatalf("SOT-ENG-026: fixture names = %#v", got)
			}
			got[0] = "changed.json"
			again, err := filesystem.fixtureFileNames(test.kind)
			if err != nil {
				t.Fatalf("SOT-ENG-026: fixture 再列挙 error = %v", err)
			}
			if !filesystemPathTestEqualStrings(again, test.want) {
				t.Fatal("SOT-ENG-026: fixture names が getter から変更された")
			}
		})
	}
}

func TestCorpusFilesystemはcanonicalでないcorpusPathを拒否する(t *testing.T) {
	t.Parallel()

	layout := filesystemPathTestCreateLayout(t, false)
	parent := filepath.Dir(layout.corpusRoot)
	placeholder := filepath.Join(parent, "placeholder")
	filesystemPathTestMkdirAll(t, placeholder)

	outside := filesystemPathTestCreateLayout(t, false)
	different := filepath.Join(layout.repositoryRoot, "other", "corpus-v1")
	filesystemPathTestPopulateCorpusRoot(t, different)
	versionZero := filepath.Join(parent, "corpus-v0")
	versionLeadingZero := filepath.Join(parent, "corpus-v01")
	filesystemPathTestPopulateCorpusRoot(t, versionZero)
	filesystemPathTestPopulateCorpusRoot(t, versionLeadingZero)

	separator := string(os.PathSeparator)
	tests := map[string]string{
		"repository外": outside.corpusRoot,
		"別subtree":    different,
		"dot":         parent + separator + "." + separator + "corpus-v1",
		"dotdot": parent + separator + "placeholder" + separator +
			".." + separator + "corpus-v1",
		"末尾separator": layout.corpusRoot + separator,
		"corpus-v0":   versionZero,
		"corpus-v01":  versionLeadingZero,
	}
	for name, corpusInput := range tests {
		name, corpusInput := name, corpusInput
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := filesystemPathTestOpenAndScan(
				context.Background(),
				layout.repositoryRoot,
				corpusInput,
			); err == nil {
				t.Fatal("SOT-ENG-026: canonical でない corpus path を受理した")
			}
		})
	}
}

func TestCorpusFilesystemはrepositoryRootの不正な種類を拒否する(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	fileRoot := filepath.Join(base, "repository-file")
	filesystemPathTestWriteFile(t, fileRoot, []byte("file"))
	missingRoot := filepath.Join(base, "repository-missing")

	target := filesystemPathTestCreateLayout(t, false)
	symlinkRoot := filepath.Join(base, "repository-symlink")
	filesystemPathTestSymlink(t, target.repositoryRoot, symlinkRoot)

	for name, repositoryRoot := range map[string]string{
		"file":    fileRoot,
		"missing": missingRoot,
		"symlink": symlinkRoot,
	} {
		name, repositoryRoot := name, repositoryRoot
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := filesystemPathTestOpenAndScan(
				context.Background(),
				repositoryRoot,
				filepath.Join("testdata", "legalquery", "corpus-v1"),
			); err == nil {
				t.Fatal("SOT-ENG-026: 不正な repository root を受理した")
			}
		})
	}
}

func TestCorpusFilesystemは各構成要素のsymlinkを拒否する(t *testing.T) {
	t.Parallel()

	components := map[string]func(filesystemPathTestLayout) string{
		"testdata": func(layout filesystemPathTestLayout) string {
			return filepath.Join(layout.repositoryRoot, "testdata")
		},
		"legalquery": func(layout filesystemPathTestLayout) string {
			return filepath.Join(layout.repositoryRoot, "testdata", "legalquery")
		},
		"schemas": func(layout filesystemPathTestLayout) string {
			return filepath.Join(
				layout.repositoryRoot,
				"testdata",
				"legalquery",
				"schemas",
			)
		},
		"corpus root": func(layout filesystemPathTestLayout) string {
			return layout.corpusRoot
		},
		"manifest": func(layout filesystemPathTestLayout) string {
			return filepath.Join(layout.corpusRoot, "manifest.json")
		},
		"development set": func(layout filesystemPathTestLayout) string {
			return filepath.Join(layout.corpusRoot, "development")
		},
		"holdout set": func(layout filesystemPathTestLayout) string {
			return filepath.Join(layout.corpusRoot, "holdout")
		},
		"execution set": func(layout filesystemPathTestLayout) string {
			return filepath.Join(layout.corpusRoot, "execution")
		},
		"fixture": func(layout filesystemPathTestLayout) string {
			return filepath.Join(
				layout.corpusRoot,
				"development",
				"development-a.json",
			)
		},
	}
	for component, selectPath := range components {
		component, selectPath := component, selectPath
		for _, targetScope := range []string{"internal", "external"} {
			targetScope := targetScope
			t.Run(component+" "+targetScope, func(t *testing.T) {
				t.Parallel()
				layout := filesystemPathTestCreateLayout(t, false)
				targetParent := layout.repositoryRoot
				if targetScope == "external" {
					targetParent = t.TempDir()
				}
				filesystemPathTestReplaceWithSymlink(
					t,
					selectPath(layout),
					targetParent,
				)
				if err := filesystemPathTestOpenAndScan(
					context.Background(),
					layout.repositoryRoot,
					layout.corpusRoot,
				); err == nil {
					t.Fatal("SOT-ENG-026: symlink の構成要素を受理した")
				}
			})
		}
	}
}

func TestCorpusFilesystemはcorpusRootの未知欠落型違反を拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*testing.T, filesystemPathTestLayout){
		"未知entry": func(t *testing.T, layout filesystemPathTestLayout) {
			filesystemPathTestWriteFile(
				t,
				filepath.Join(layout.corpusRoot, "unknown"),
				[]byte("unknown"),
			)
		},
		"manifest欠落": func(t *testing.T, layout filesystemPathTestLayout) {
			filesystemPathTestRemove(t, filepath.Join(layout.corpusRoot, "manifest.json"))
		},
		"manifestがdirectory": func(
			t *testing.T,
			layout filesystemPathTestLayout,
		) {
			path := filepath.Join(layout.corpusRoot, "manifest.json")
			filesystemPathTestRemove(t, path)
			filesystemPathTestMkdirAll(t, path)
		},
		"development欠落": func(t *testing.T, layout filesystemPathTestLayout) {
			filesystemPathTestRemove(
				t,
				filepath.Join(layout.corpusRoot, "development"),
			)
		},
		"holdout欠落": func(t *testing.T, layout filesystemPathTestLayout) {
			filesystemPathTestRemove(t, filepath.Join(layout.corpusRoot, "holdout"))
		},
		"execution欠落": func(t *testing.T, layout filesystemPathTestLayout) {
			filesystemPathTestRemove(t, filepath.Join(layout.corpusRoot, "execution"))
		},
		"developmentがfile": func(t *testing.T, layout filesystemPathTestLayout) {
			path := filepath.Join(layout.corpusRoot, "development")
			filesystemPathTestRemove(t, path)
			filesystemPathTestWriteFile(t, path, []byte("file"))
		},
		"holdoutがfile": func(t *testing.T, layout filesystemPathTestLayout) {
			path := filepath.Join(layout.corpusRoot, "holdout")
			filesystemPathTestRemove(t, path)
			filesystemPathTestWriteFile(t, path, []byte("file"))
		},
		"executionがfile": func(t *testing.T, layout filesystemPathTestLayout) {
			path := filepath.Join(layout.corpusRoot, "execution")
			filesystemPathTestRemove(t, path)
			filesystemPathTestWriteFile(t, path, []byte("file"))
		},
		"corpus root欠落": func(t *testing.T, layout filesystemPathTestLayout) {
			filesystemPathTestRemove(t, layout.corpusRoot)
		},
		"corpus rootがfile": func(t *testing.T, layout filesystemPathTestLayout) {
			filesystemPathTestRemove(t, layout.corpusRoot)
			filesystemPathTestWriteFile(t, layout.corpusRoot, []byte("file"))
		},
		"schemas欠落": func(t *testing.T, layout filesystemPathTestLayout) {
			filesystemPathTestRemove(
				t,
				filepath.Join(
					layout.repositoryRoot,
					"testdata",
					"legalquery",
					"schemas",
				),
			)
		},
		"schemasがfile": func(t *testing.T, layout filesystemPathTestLayout) {
			path := filepath.Join(
				layout.repositoryRoot,
				"testdata",
				"legalquery",
				"schemas",
			)
			filesystemPathTestRemove(t, path)
			filesystemPathTestWriteFile(t, path, []byte("file"))
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			layout := filesystemPathTestCreateLayout(t, false)
			mutate(t, layout)
			if err := filesystemPathTestOpenAndScan(
				context.Background(),
				layout.repositoryRoot,
				layout.corpusRoot,
			); err == nil {
				t.Fatal("SOT-ENG-026: 欠落、未知または型違反の root entry を受理した")
			}
		})
	}
}

func TestCorpusFilesystemはset内の不正entryを拒否する(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*testing.T, string){
		"subdirectory": func(t *testing.T, setRoot string) {
			filesystemPathTestMkdirAll(t, filepath.Join(setRoot, "nested"))
		},
		"json subdirectory": func(t *testing.T, setRoot string) {
			filesystemPathTestMkdirAll(t, filepath.Join(setRoot, "nested.json"))
		},
		"non-json": func(t *testing.T, setRoot string) {
			filesystemPathTestWriteFile(
				t,
				filepath.Join(setRoot, "notes.txt"),
				[]byte("notes"),
			)
		},
	}
	for _, kind := range []ManifestSetKind{
		ManifestSetDevelopment,
		ManifestSetHoldout,
		ManifestSetExecution,
	} {
		kind := kind
		for name, mutate := range tests {
			name, mutate := name, mutate
			t.Run(string(kind)+" "+name, func(t *testing.T) {
				t.Parallel()
				layout := filesystemPathTestCreateLayout(t, false)
				mutate(t, filepath.Join(layout.corpusRoot, string(kind)))
				if err := filesystemPathTestOpenAndScan(
					context.Background(),
					layout.repositoryRoot,
					layout.corpusRoot,
				); err == nil {
					t.Fatal("SOT-ENG-026: set 内の不正 entry を受理した")
				}
			})
		}
	}
}

func TestCorpusFilesystemはcancel済みとnilContextを拒否する(t *testing.T) {
	t.Parallel()

	layout := filesystemPathTestCreateLayout(t, false)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	for name, ctx := range map[string]context.Context{
		"pre-canceled": canceled,
		"nil":          nil,
	} {
		name, ctx := name, ctx
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := filesystemPathTestOpenAndScan(
				ctx,
				layout.repositoryRoot,
				layout.corpusRoot,
			); err == nil {
				t.Fatal("SOT-ENG-026: 無効な context を受理した")
			}
		})
	}
}

func TestCorpusFilesystemErrorは入力pathとmanifest内容を含まない(t *testing.T) {
	t.Parallel()

	const (
		pathSecret     = "filesystem-path-test-secret-repository"
		manifestSecret = "filesystem-path-test-secret-manifest-content"
	)
	repositoryRoot := filepath.Join(t.TempDir(), pathSecret)
	layout := filesystemPathTestCreateLayoutAt(t, repositoryRoot)
	filesystemPathTestWriteFile(
		t,
		filepath.Join(layout.corpusRoot, "manifest.json"),
		[]byte(manifestSecret),
	)
	filesystemPathTestWriteFile(
		t,
		filepath.Join(layout.corpusRoot, "development", "forbidden.txt"),
		[]byte("forbidden"),
	)

	err := filesystemPathTestOpenAndScan(
		context.Background(),
		layout.repositoryRoot,
		layout.corpusRoot,
	)
	if err == nil {
		t.Fatal("SOT-ENG-026: 不正 entry を受理した")
	}
	if strings.Contains(err.Error(), pathSecret) ||
		strings.Contains(err.Error(), manifestSecret) {
		t.Fatalf("SOT-ENG-026: error が入力 path または manifest 内容を含む: %v", err)
	}
}

type filesystemPathTestLayout struct {
	repositoryRoot string
	corpusRoot     string
}

func filesystemPathTestCreateLayout(
	t *testing.T,
	relativeRepository bool,
) filesystemPathTestLayout {
	t.Helper()
	if !relativeRepository {
		return filesystemPathTestCreateLayoutAt(
			t,
			filepath.Join(t.TempDir(), "repository"),
		)
	}
	workspace, err := os.MkdirTemp(".", "filesystem-path-test-")
	if err != nil {
		t.Fatalf("相対 path 用一時 directory を作成できません: %v", err)
	}
	workspace = filepath.Clean(workspace)
	t.Cleanup(func() {
		if err := os.RemoveAll(workspace); err != nil {
			t.Errorf("相対 path 用一時 directory を削除できません: %v", err)
		}
	})
	return filesystemPathTestCreateLayoutAt(
		t,
		filepath.Join(workspace, "repository"),
	)
}

func filesystemPathTestCreateLayoutAt(
	t *testing.T,
	repositoryRoot string,
) filesystemPathTestLayout {
	t.Helper()
	schemas := filepath.Join(
		repositoryRoot,
		"testdata",
		"legalquery",
		"schemas",
	)
	filesystemPathTestMkdirAll(t, schemas)
	corpusRoot := filepath.Join(
		repositoryRoot,
		"testdata",
		"legalquery",
		"corpus-v1",
	)
	filesystemPathTestPopulateCorpusRoot(t, corpusRoot)
	return filesystemPathTestLayout{
		repositoryRoot: repositoryRoot,
		corpusRoot:     corpusRoot,
	}
}

func filesystemPathTestPopulateCorpusRoot(t *testing.T, corpusRoot string) {
	t.Helper()
	filesystemPathTestMkdirAll(t, corpusRoot)
	filesystemPathTestWriteFile(
		t,
		filepath.Join(corpusRoot, "manifest.json"),
		[]byte(filesystemPathTestManifestContent),
	)
	for _, kind := range []ManifestSetKind{
		ManifestSetDevelopment,
		ManifestSetHoldout,
		ManifestSetExecution,
	} {
		setRoot := filepath.Join(corpusRoot, string(kind))
		filesystemPathTestMkdirAll(t, setRoot)
		filesystemPathTestWriteFile(
			t,
			filepath.Join(setRoot, string(kind)+"-z.json"),
			[]byte(`{}`),
		)
		filesystemPathTestWriteFile(
			t,
			filepath.Join(setRoot, string(kind)+"-a.json"),
			[]byte(`{}`),
		)
	}
}

func filesystemPathTestOpenAndScan(
	ctx context.Context,
	repositoryRoot string,
	corpusDirectory string,
) error {
	filesystem, err := openCorpusFilesystem(
		ctx,
		repositoryRoot,
		corpusDirectory,
	)
	if err != nil {
		return err
	}
	defer func() {
		_ = filesystem.close()
	}()
	for _, kind := range []ManifestSetKind{
		ManifestSetDevelopment,
		ManifestSetHoldout,
		ManifestSetExecution,
	} {
		if _, err := filesystem.fixtureFileNames(kind); err != nil {
			return err
		}
	}
	return nil
}

func filesystemPathTestReplaceWithSymlink(
	t *testing.T,
	originalPath string,
	targetParent string,
) {
	t.Helper()
	targetPath := filepath.Join(targetParent, "filesystem-path-test-link-target")
	if err := os.Rename(originalPath, targetPath); err != nil {
		t.Fatalf("symlink 対象を移動できません: %v", err)
	}
	filesystemPathTestSymlink(t, targetPath, originalPath)
}

func filesystemPathTestSymlink(
	t *testing.T,
	targetPath string,
	linkPath string,
) {
	t.Helper()
	if err := os.Symlink(targetPath, linkPath); err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink を作成する権限がないため省略します: %v", err)
		}
		t.Fatalf("symlink を作成できません: %v", err)
	}
}

func filesystemPathTestMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatalf("directory を作成できません: %v", err)
	}
}

func filesystemPathTestWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("file を作成できません: %v", err)
	}
}

func filesystemPathTestRemove(t *testing.T, path string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("path を削除できません: %v", err)
	}
}

func filesystemPathTestEqualStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
