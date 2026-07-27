package legalquerycorpus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	filesystemReadTestSchemaMaximumBytes       = 1 << 20
	filesystemReadTestManifestMaximumBytes     = 2 << 20
	filesystemReadTestFixtureMaximumBytes      = 256 << 10
	filesystemReadTestFixtureMaximumCount      = 4096
	filesystemReadTestFixtureMaximumTotalBytes = 64 << 20
)

type filesystemReadTestLayout struct {
	repositoryRoot string
	corpusPath     string
	schemaPath     string
	manifestPath   string
	setPaths       map[ManifestSetKind]string
}

func TestCorpusFilesystemはfileSize境界と原byteを保つ(t *testing.T) {
	tests := []struct {
		name      string
		maximum   int
		writeData func(
			t *testing.T,
			layout filesystemReadTestLayout,
			data []byte,
		)
		readData func(
			ctx context.Context,
			fs *corpusFilesystem,
		) ([]byte, error)
	}{
		{
			name:    "schema",
			maximum: filesystemReadTestSchemaMaximumBytes,
			writeData: func(
				t *testing.T,
				layout filesystemReadTestLayout,
				data []byte,
			) {
				filesystemReadTestWriteFile(t, layout.schemaPath, data)
			},
			readData: func(
				ctx context.Context,
				fs *corpusFilesystem,
			) ([]byte, error) {
				return fs.readSchemaV1(ctx)
			},
		},
		{
			name:    "manifest",
			maximum: filesystemReadTestManifestMaximumBytes,
			writeData: func(
				t *testing.T,
				layout filesystemReadTestLayout,
				data []byte,
			) {
				filesystemReadTestWriteFile(t, layout.manifestPath, data)
			},
			readData: func(
				_ context.Context,
				fs *corpusFilesystem,
			) ([]byte, error) {
				return fs.manifestBytes(), nil
			},
		},
		{
			name:    "fixture",
			maximum: filesystemReadTestFixtureMaximumBytes,
			writeData: func(
				t *testing.T,
				layout filesystemReadTestLayout,
				data []byte,
			) {
				filesystemReadTestWriteFixture(
					t,
					layout,
					ManifestSetDevelopment,
					"development-size-boundary",
					data,
				)
			},
			readData: func(
				ctx context.Context,
				fs *corpusFilesystem,
			) ([]byte, error) {
				return fs.newFixtureReader().read(
					ctx,
					ManifestSetDevelopment,
					"development-size-boundary",
				)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name+"上限丁度", func(t *testing.T) {
			layout := filesystemReadTestNewLayout(t)
			want := filesystemReadTestJSONDocument(test.maximum, "raw-boundary")
			test.writeData(t, layout, want)

			fs := filesystemReadTestOpen(t, layout)
			got, err := test.readData(context.Background(), fs)
			if err != nil {
				t.Fatalf(
					"SOT-ENG-026: %s の上限丁度を拒否した: %v",
					test.name,
					err,
				)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf(
					"SOT-ENG-026: %s の原 byte が一致しない",
					test.name,
				)
			}
		})

		t.Run(test.name+"上限超過", func(t *testing.T) {
			layout := filesystemReadTestNewLayout(t)
			test.writeData(
				t,
				layout,
				filesystemReadTestJSONDocument(test.maximum+1, "too-large"),
			)

			if test.name == "manifest" || test.name == "fixture" {
				fs, err := openCorpusFilesystem(
					context.Background(),
					layout.repositoryRoot,
					layout.corpusPath,
				)
				if fs != nil {
					_ = fs.close()
				}
				if err == nil {
					t.Fatalf(
						"SOT-ENG-026: %s の上限超過を受理した",
						test.name,
					)
				}
				return
			}

			fs := filesystemReadTestOpen(t, layout)
			if _, err := test.readData(context.Background(), fs); err == nil {
				t.Fatalf(
					"SOT-ENG-026: %s の上限超過を受理した",
					test.name,
				)
			}
		})
	}
}

func TestCorpusFilesystemはfixture原byte合計の宣言sizeを制限する(t *testing.T) {
	layout := filesystemReadTestNewLayout(t)
	const fixtureCountAtTotalLimit = filesystemReadTestFixtureMaximumTotalBytes /
		filesystemReadTestFixtureMaximumBytes

	for index := 0; index < fixtureCountAtTotalLimit; index++ {
		caseID := fmt.Sprintf("development-total-%04d", index)
		path := filesystemReadTestFixturePath(
			layout,
			ManifestSetDevelopment,
			caseID,
		)
		filesystemReadTestCreateSparseFile(
			t,
			path,
			filesystemReadTestFixtureMaximumBytes,
		)
	}

	fs, err := openCorpusFilesystem(
		context.Background(),
		layout.repositoryRoot,
		layout.corpusPath,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: fixture 合計 64 MiB 丁度を拒否した: %v", err)
	}
	_ = fs.close()

	filesystemReadTestCreateSparseFile(
		t,
		filesystemReadTestFixturePath(
			layout,
			ManifestSetDevelopment,
			"development-total-over",
		),
		1,
	)
	fs, err = openCorpusFilesystem(
		context.Background(),
		layout.repositoryRoot,
		layout.corpusPath,
	)
	if fs != nil {
		_ = fs.close()
	}
	if err == nil {
		t.Fatal("SOT-ENG-026: fixture 合計 64 MiB + 1 byte を受理した")
	}
}

func TestCorpusFilesystemはfixture総数を制限する(t *testing.T) {
	layout := filesystemReadTestNewLayout(t)
	for index := 0; index < filesystemReadTestFixtureMaximumCount; index++ {
		caseID := fmt.Sprintf("development-count-%04d", index)
		filesystemReadTestWriteFixture(
			t,
			layout,
			ManifestSetDevelopment,
			caseID,
			nil,
		)
	}

	fs, err := openCorpusFilesystem(
		context.Background(),
		layout.repositoryRoot,
		layout.corpusPath,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: fixture 4096 件丁度を拒否した: %v", err)
	}
	_ = fs.close()

	filesystemReadTestWriteFixture(
		t,
		layout,
		ManifestSetDevelopment,
		"development-count-over",
		nil,
	)
	fs, err = openCorpusFilesystem(
		context.Background(),
		layout.repositoryRoot,
		layout.corpusPath,
	)
	if fs != nil {
		_ = fs.close()
	}
	if err == nil {
		t.Fatal("SOT-ENG-026: fixture 4097 件を受理した")
	}
}

func TestFixtureReaderは集合とcaseIDから一度だけ読む(t *testing.T) {
	layout := filesystemReadTestNewLayout(t)
	want := []byte("{\n  \"raw\": \"fixture-byte-sequence\"\n}\n")
	filesystemReadTestWriteFixture(
		t,
		layout,
		ManifestSetDevelopment,
		"development-valid",
		want,
	)
	fs := filesystemReadTestOpen(t, layout)
	reader := fs.newFixtureReader()

	got, err := reader.read(
		context.Background(),
		ManifestSetDevelopment,
		"development-valid",
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: 有効な fixture を読めない: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("SOT-ENG-026: fixture の原 byte が変化した")
	}
	if _, err := reader.read(
		context.Background(),
		ManifestSetDevelopment,
		"development-valid",
	); err == nil {
		t.Fatal("SOT-ENG-026: 同じ fixture の二回目の読取りを受理した")
	}
}

func TestFixtureReaderは不正な集合とcaseIDを拒否する(t *testing.T) {
	layout := filesystemReadTestNewLayout(t)
	filesystemReadTestWriteFixture(
		t,
		layout,
		ManifestSetDevelopment,
		"development-present",
		[]byte(`{"present":true}`),
	)
	fs := filesystemReadTestOpen(t, layout)

	tests := []struct {
		name   string
		set    ManifestSetKind
		caseID string
	}{
		{
			name:   "未知の集合",
			set:    ManifestSetKind("unknown"),
			caseID: "development-present",
		},
		{
			name:   "集合prefix不一致",
			set:    ManifestSetDevelopment,
			caseID: "holdout-present",
		},
		{
			name:   "空caseID",
			set:    ManifestSetDevelopment,
			caseID: "",
		},
		{
			name:   "親directory",
			set:    ManifestSetDevelopment,
			caseID: "../development-present",
		},
		{
			name:   "slash",
			set:    ManifestSetDevelopment,
			caseID: "development-a/b",
		},
		{
			name:   "backslash",
			set:    ManifestSetDevelopment,
			caseID: `development-a\b`,
		},
		{
			name:   "絶対path",
			set:    ManifestSetDevelopment,
			caseID: "/development-present",
		},
		{
			name:   "未登録fixture",
			set:    ManifestSetDevelopment,
			caseID: "development-missing",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			reader := fs.newFixtureReader()
			if _, err := reader.read(
				context.Background(),
				test.set,
				test.caseID,
			); err == nil {
				t.Fatal("SOT-ENG-026: 不正または未登録の fixture 指定を受理した")
			}
		})
	}
}

func TestFixtureReaderは列挙後のfixture差替えを拒否する(t *testing.T) {
	tests := []struct {
		name   string
		change func(t *testing.T, path string)
	}{
		{
			name: "通常fileへの差替え",
			change: func(t *testing.T, path string) {
				held, err := os.Open(path) //nolint:gosec // SOT-ENG-026: test 一時 directory 内の検証対象だけを開く。
				if err != nil {
					t.Fatalf("SOT-ENG-026: 差替え前 file を保持できない: %v", err)
				}
				t.Cleanup(func() {
					_ = held.Close()
				})
				if err := os.Remove(path); err != nil {
					t.Fatalf("SOT-ENG-026: fixture を削除できない: %v", err)
				}
				filesystemReadTestWriteFile(t, path, []byte(`{"replaced":true}`))
			},
		},
		{
			name: "列挙時sizeからの増加",
			change: func(t *testing.T, path string) {
				file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0) //nolint:gosec // SOT-ENG-026: test 一時 directory 内の検証対象だけを開く。
				if err != nil {
					t.Fatalf("SOT-ENG-026: fixture を追記用に開けない: %v", err)
				}
				if _, err := file.Write([]byte("x")); err != nil {
					_ = file.Close()
					t.Fatalf("SOT-ENG-026: fixture を増加できない: %v", err)
				}
				if err := file.Close(); err != nil {
					t.Fatalf("SOT-ENG-026: 増加した fixture を閉じられない: %v", err)
				}
			},
		},
		{
			name: "symlinkへの差替え",
			change: func(t *testing.T, path string) {
				target := filepath.Join(t.TempDir(), "outside.json")
				filesystemReadTestWriteFile(t, target, []byte(`{"replaced":true}`))
				if err := os.Remove(path); err != nil {
					t.Fatalf("SOT-ENG-026: fixture を削除できない: %v", err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Fatalf("SOT-ENG-026: fixture symlink を作成できない: %v", err)
				}
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			layout := filesystemReadTestNewLayout(t)
			const caseID = "development-replaced"
			filesystemReadTestWriteFixture(
				t,
				layout,
				ManifestSetDevelopment,
				caseID,
				[]byte(`{"original":true}`),
			)
			fs := filesystemReadTestOpen(t, layout)
			path := filesystemReadTestFixturePath(
				layout,
				ManifestSetDevelopment,
				caseID,
			)
			test.change(t, path)

			if _, err := fs.newFixtureReader().read(
				context.Background(),
				ManifestSetDevelopment,
				caseID,
			); err == nil {
				t.Fatal("SOT-ENG-026: 列挙後に変化した fixture を受理した")
			}
		})
	}
}

func TestCorpusFilesystemは読取り前のcontext取消を返す(t *testing.T) {
	t.Run("schema", func(t *testing.T) {
		layout := filesystemReadTestNewLayout(t)
		fs := filesystemReadTestOpen(t, layout)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := fs.readSchemaV1(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("SOT-ENG-026: schema の取消 error = %v", err)
		}
	})

	t.Run("fixture", func(t *testing.T) {
		layout := filesystemReadTestNewLayout(t)
		filesystemReadTestWriteFixture(
			t,
			layout,
			ManifestSetDevelopment,
			"development-canceled",
			[]byte(`{"secret":"must-not-be-read"}`),
		)
		fs := filesystemReadTestOpen(t, layout)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := fs.newFixtureReader().read(
			ctx,
			ManifestSetDevelopment,
			"development-canceled",
		); !errors.Is(err, context.Canceled) {
			t.Fatalf("SOT-ENG-026: fixture の取消 error = %v", err)
		}
	})
}

func TestFixtureReaderはfixture間のcontext取消を返す(t *testing.T) {
	layout := filesystemReadTestNewLayout(t)
	for _, caseID := range []string{"development-first", "development-second"} {
		filesystemReadTestWriteFixture(
			t,
			layout,
			ManifestSetDevelopment,
			caseID,
			[]byte(fmt.Sprintf(`{"caseId":%q}`, caseID)),
		)
	}
	fs := filesystemReadTestOpen(t, layout)
	reader := fs.newFixtureReader()
	ctx, cancel := context.WithCancel(context.Background())

	if _, err := reader.read(
		ctx,
		ManifestSetDevelopment,
		"development-first",
	); err != nil {
		t.Fatalf("SOT-ENG-026: 一件目の fixture を読めない: %v", err)
	}
	cancel()
	if _, err := reader.read(
		ctx,
		ManifestSetDevelopment,
		"development-second",
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("SOT-ENG-026: fixture 間の取消 error = %v", err)
	}
}

func TestCorpusFilesystemのerrorはfile内容を含まない(t *testing.T) {
	const secret = "secret-query-content-must-not-leak"

	t.Run("schema", func(t *testing.T) {
		layout := filesystemReadTestNewLayout(t)
		filesystemReadTestWriteFile(
			t,
			layout.schemaPath,
			filesystemReadTestJSONDocument(
				filesystemReadTestSchemaMaximumBytes+1,
				secret,
			),
		)
		fs := filesystemReadTestOpen(t, layout)
		_, err := fs.readSchemaV1(context.Background())
		filesystemReadTestRequireSafeError(t, err, secret)
	})

	t.Run("manifest", func(t *testing.T) {
		layout := filesystemReadTestNewLayout(t)
		filesystemReadTestWriteFile(
			t,
			layout.manifestPath,
			filesystemReadTestJSONDocument(
				filesystemReadTestManifestMaximumBytes+1,
				secret,
			),
		)
		fs, err := openCorpusFilesystem(
			context.Background(),
			layout.repositoryRoot,
			layout.corpusPath,
		)
		if fs != nil {
			_ = fs.close()
		}
		filesystemReadTestRequireSafeError(t, err, secret)
	})

	t.Run("fixture", func(t *testing.T) {
		layout := filesystemReadTestNewLayout(t)
		filesystemReadTestWriteFixture(
			t,
			layout,
			ManifestSetDevelopment,
			"development-secret",
			filesystemReadTestJSONDocument(
				filesystemReadTestFixtureMaximumBytes+1,
				secret,
			),
		)
		fs, err := openCorpusFilesystem(
			context.Background(),
			layout.repositoryRoot,
			layout.corpusPath,
		)
		if fs != nil {
			_ = fs.close()
		}
		filesystemReadTestRequireSafeError(t, err, secret)
	})
}

func filesystemReadTestNewLayout(t *testing.T) filesystemReadTestLayout {
	t.Helper()

	repositoryRoot := t.TempDir()
	corpusPath := filepath.Join("testdata", "legalquery", "corpus-v1")
	corpusRoot := filepath.Join(repositoryRoot, corpusPath)
	schemaPath := filepath.Join(
		repositoryRoot,
		"testdata",
		"legalquery",
		"schemas",
		"legal-query-corpus-v1.schema.json",
	)
	setPaths := map[ManifestSetKind]string{
		ManifestSetDevelopment: filepath.Join(corpusRoot, "development"),
		ManifestSetHoldout:     filepath.Join(corpusRoot, "holdout"),
		ManifestSetExecution:   filepath.Join(corpusRoot, "execution"),
	}
	for _, path := range setPaths {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatalf("SOT-ENG-026: 集合 directory を作成できない: %v", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(schemaPath), 0o700); err != nil {
		t.Fatalf("SOT-ENG-026: schema directory を作成できない: %v", err)
	}
	manifestPath := filepath.Join(corpusRoot, "manifest.json")
	filesystemReadTestWriteFile(t, schemaPath, []byte(`{"schema":"v1"}`))
	filesystemReadTestWriteFile(t, manifestPath, []byte(`{"manifest":"v1"}`))

	return filesystemReadTestLayout{
		repositoryRoot: repositoryRoot,
		corpusPath:     corpusPath,
		schemaPath:     schemaPath,
		manifestPath:   manifestPath,
		setPaths:       setPaths,
	}
}

func filesystemReadTestOpen(
	t *testing.T,
	layout filesystemReadTestLayout,
) *corpusFilesystem {
	t.Helper()

	fs, err := openCorpusFilesystem(
		context.Background(),
		layout.repositoryRoot,
		layout.corpusPath,
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: filesystem を開けない: %v", err)
	}
	t.Cleanup(func() {
		_ = fs.close()
	})
	return fs
}

func filesystemReadTestWriteFixture(
	t *testing.T,
	layout filesystemReadTestLayout,
	set ManifestSetKind,
	caseID string,
	data []byte,
) {
	t.Helper()

	filesystemReadTestWriteFile(
		t,
		filesystemReadTestFixturePath(layout, set, caseID),
		data,
	)
}

func filesystemReadTestFixturePath(
	layout filesystemReadTestLayout,
	set ManifestSetKind,
	caseID string,
) string {
	return filepath.Join(layout.setPaths[set], caseID+".json")
}

func filesystemReadTestWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("SOT-ENG-026: test file を書き込めない: %v", err)
	}
}

func filesystemReadTestCreateSparseFile(
	t *testing.T,
	path string,
	size int,
) {
	t.Helper()

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // SOT-ENG-026: test 一時 directory 内に sparse fixture を作る。
	if err != nil {
		t.Fatalf("SOT-ENG-026: sparse fixture を作成できない: %v", err)
	}
	if err := file.Truncate(int64(size)); err != nil {
		_ = file.Close()
		t.Fatalf("SOT-ENG-026: sparse fixture を拡張できない: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("SOT-ENG-026: sparse fixture を閉じられない: %v", err)
	}
}

func filesystemReadTestJSONDocument(size int, marker string) []byte {
	prefix := []byte(`{"marker":"` + marker + `","padding":"`)
	suffix := []byte(`"}`)
	if size < len(prefix)+len(suffix) {
		panic("test JSON document の size が小さすぎます")
	}
	padding := bytes.Repeat([]byte("x"), size-len(prefix)-len(suffix))
	return append(append(prefix, padding...), suffix...)
}

func filesystemReadTestRequireSafeError(
	t *testing.T,
	err error,
	secret string,
) {
	t.Helper()

	if err == nil {
		t.Fatal("SOT-ENG-026: 境界違反を受理した")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("SOT-ENG-026: error が file 内容を含む: %v", err)
	}
}
