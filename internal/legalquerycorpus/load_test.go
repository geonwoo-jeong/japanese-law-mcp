package legalquerycorpus

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadは実在成果物を検証してmanifest順のCorpusを返す(t *testing.T) {
	layout := loadTestWriteValidCorpus(t)

	for _, corpusDirectory := range []string{
		layout.corpusPath,
		filepath.Join(layout.repositoryRoot, layout.corpusPath),
	} {
		corpusDirectory := corpusDirectory
		t.Run(corpusDirectory, func(t *testing.T) {
			got, err := Load(
				context.Background(),
				layout.repositoryRoot,
				corpusDirectory,
			)
			if err != nil {
				t.Fatalf("SOT-ENG-026: Load() error = %v", err)
			}
			if err := got.Validate(); err != nil {
				t.Fatalf("SOT-ENG-026: Corpus.Validate() error = %v", err)
			}
			if len(got.Development()) != 8 ||
				len(got.Holdout()) != 240 ||
				len(got.Execution()) != 8 {
				t.Fatalf(
					"SOT-ENG-026: collection size = (%d, %d, %d)",
					len(got.Development()),
					len(got.Holdout()),
					len(got.Execution()),
				)
			}
			if got.Development()[0].CaseID() != "development-all-failed" ||
				got.Holdout()[0].CaseID() != "holdout-case-000" ||
				got.Execution()[0].CaseID() != "execution-all-failed" {
				t.Fatal("SOT-ENG-026: Load が manifest 順を保持しませんでした")
			}

			development := got.Development()
			development[0] = SemanticCase{}
			manifest := got.Manifest()
			cases := manifest.Development().Cases()
			cases[0] = ManifestEntry{}
			if got.Development()[0].CaseID() != "development-all-failed" ||
				got.Manifest().Development().Cases()[0].CaseID() !=
					"development-all-failed" {
				t.Fatal("SOT-ENG-026: Load の戻り値が外部から変更されました")
			}
		})
	}
}

func TestLoadは取消と各段階の失敗でCorpusの零値を返す(t *testing.T) {
	t.Run("取消", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		got, err := Load(ctx, t.TempDir(), "testdata/legalquery/corpus-v1")
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("SOT-ENG-026: cancellation error = %v", err)
		}
		if !reflect.DeepEqual(got, Corpus{}) {
			t.Fatalf("SOT-ENG-026: 取消時の部分結果 = %#v", got)
		}
	})

	t.Run("manifest headerをschemaより先に検証", func(t *testing.T) {
		layout := loadTestWriteValidCorpus(t)
		filesystemReadTestWriteFile(
			t,
			layout.manifestPath,
			[]byte(`{"artifactKind":"corpus_manifest","schemaVersion":3}`),
		)
		if err := os.Remove(layout.schemaPath); err != nil {
			t.Fatalf("SOT-ENG-026: schema を削除できません: %v", err)
		}

		got, err := Load(
			context.Background(),
			layout.repositoryRoot,
			layout.corpusPath,
		)
		if err == nil || !strings.Contains(err.Error(), "schemaVersion") {
			t.Fatalf("SOT-ENG-026: bootstrap error = %v", err)
		}
		if strings.Contains(err.Error(), layout.repositoryRoot) {
			t.Fatalf("SOT-ENG-026: error が repository path を含みます: %v", err)
		}
		if !reflect.DeepEqual(got, Corpus{}) {
			t.Fatalf("SOT-ENG-026: bootstrap 失敗時の部分結果 = %#v", got)
		}
	})

	t.Run("manifest不整合", func(t *testing.T) {
		layout := loadTestWriteValidCorpus(t)
		fixturePath := filesystemReadTestFixturePath(
			layout,
			ManifestSetDevelopment,
			"development-all-failed",
		)
		filesystemReadTestWriteFile(
			t,
			fixturePath,
			[]byte(`{"secret":"照会原文を返さない"}`),
		)

		got, err := Load(
			context.Background(),
			layout.repositoryRoot,
			layout.corpusPath,
		)
		if err == nil {
			t.Fatal("SOT-ENG-026: fixture 不整合を受理しました")
		}
		if strings.Contains(err.Error(), "照会原文を返さない") {
			t.Fatalf("SOT-ENG-026: error が fixture 原文を含みます: %v", err)
		}
		if !reflect.DeepEqual(got, Corpus{}) {
			t.Fatalf("SOT-ENG-026: fixture 失敗時の部分結果 = %#v", got)
		}
	})
}

func TestLoadはManifestとSchemaの各失敗を零値にする(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, layout filesystemReadTestLayout)
	}{
		{
			name: "manifestの不正JSON",
			mutate: func(t *testing.T, layout filesystemReadTestLayout) {
				filesystemReadTestWriteFile(
					t,
					layout.manifestPath,
					[]byte(`{"artifactKind"`),
				)
			},
		},
		{
			name: "manifest以外のartifactKind",
			mutate: func(t *testing.T, layout filesystemReadTestLayout) {
				filesystemReadTestWriteFile(
					t,
					layout.manifestPath,
					[]byte(
						`{"artifactKind":"semantic_case","schemaVersion":1}`,
					),
				)
			},
		},
		{
			name: "schemaの欠落",
			mutate: func(t *testing.T, layout filesystemReadTestLayout) {
				filesystemReadTestWriteFile(
					t,
					layout.manifestPath,
					mustJSONBytes(t, validManifest()),
				)
				if err := os.Remove(layout.schemaPath); err != nil {
					t.Fatalf("SOT-ENG-026: schema を削除できません: %v", err)
				}
			},
		},
		{
			name: "schemaの安全境界違反",
			mutate: func(t *testing.T, layout filesystemReadTestLayout) {
				filesystemReadTestWriteFile(
					t,
					layout.manifestPath,
					mustJSONBytes(t, validManifest()),
				)
			},
		},
		{
			name: "manifestのschema違反",
			mutate: func(t *testing.T, layout filesystemReadTestLayout) {
				filesystemReadTestWriteFile(
					t,
					layout.schemaPath,
					schemaRuntimeTestFixedSchema(t),
				)
				filesystemReadTestWriteFile(
					t,
					layout.manifestPath,
					[]byte(
						`{"artifactKind":"corpus_manifest","schemaVersion":1}`,
					),
				)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			layout := filesystemReadTestNewLayout(t)
			test.mutate(t, layout)

			got, err := Load(
				context.Background(),
				layout.repositoryRoot,
				layout.corpusPath,
			)
			if err == nil {
				t.Fatal("SOT-ENG-026: 不正な manifest または schema を受理しました")
			}
			if strings.Contains(err.Error(), layout.repositoryRoot) {
				t.Fatalf("SOT-ENG-026: error が repository path を含みます: %v", err)
			}
			if !reflect.DeepEqual(got, Corpus{}) {
				t.Fatalf("SOT-ENG-026: 段階失敗時の部分結果 = %#v", got)
			}
		})
	}
}

func TestLoadの後始末は一回だけ実行し失敗時に零値へ戻す(t *testing.T) {
	loadFailure := errors.New("load failure")
	closeFailure := errors.New("close failure")
	tests := []struct {
		name      string
		loadError error
		closeErr  error
	}{
		{name: "成功"},
		{name: "本体失敗", loadError: loadFailure},
		{name: "取消", loadError: context.Canceled},
		{name: "close失敗", closeErr: closeFailure},
		{
			name:      "本体とcloseの両方が失敗",
			loadError: loadFailure,
			closeErr:  closeFailure,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			sentinel := Corpus{initialized: true}
			loadCalls := 0
			closeCalls := 0

			got, err := loadAndCloseCorpus(
				ctx,
				&corpusFilesystem{},
				func(
					received context.Context,
					_ *corpusFilesystem,
				) (Corpus, error) {
					loadCalls++
					if received != ctx {
						t.Fatal("SOT-ENG-026: Load の context が置換されました")
					}
					return sentinel, test.loadError
				},
				func() error {
					closeCalls++
					return test.closeErr
				},
			)
			if loadCalls != 1 || closeCalls != 1 {
				t.Fatalf(
					"SOT-ENG-026: call count = (%d, %d)",
					loadCalls,
					closeCalls,
				)
			}
			if test.loadError == nil && test.closeErr == nil {
				if err != nil || !reflect.DeepEqual(got, sentinel) {
					t.Fatalf("SOT-ENG-026: cleanup success = (%#v, %v)", got, err)
				}
				return
			}
			if !reflect.DeepEqual(got, Corpus{}) {
				t.Fatalf("SOT-ENG-026: cleanup 失敗時の部分結果 = %#v", got)
			}
			if test.loadError != nil && !errors.Is(err, test.loadError) {
				t.Fatalf("SOT-ENG-026: load error が保持されません: %v", err)
			}
			if test.closeErr != nil && !errors.Is(err, test.closeErr) {
				t.Fatalf("SOT-ENG-026: close error が保持されません: %v", err)
			}
		})
	}
}
