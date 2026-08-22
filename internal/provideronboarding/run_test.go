package provideronboarding

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// SOT-ENG-018: 初回導入では canonical loader と通常 test を再利用する。
func TestRunBootstrapLoadsMatrixAndRunsProviderConformanceTests(t *testing.T) {
	t.Parallel()

	repository, base := newBootstrapRepository(t)
	loadCalled := false
	testCalled := false
	err := runWithDependencies(
		context.Background(),
		testOptions(repository, base),
		dependencies{
			load: func(gotRepository string) ([]matrixRow, error) {
				loadCalled = true
				if gotRepository != resolvedTestPath(t, repository) {
					t.Fatalf(
						"loader repository = %q, want %q",
						gotRepository,
						resolvedTestPath(t, repository),
					)
				}
				return []matrixRow{{
					providerID: "provider-a",
					status:     "planned",
				}}, nil
			},
			test: func(_ context.Context, gotRepository string, _, _ io.Writer) error {
				testCalled = true
				if gotRepository != resolvedTestPath(t, repository) {
					t.Fatalf(
						"test repository = %q, want %q",
						gotRepository,
						resolvedTestPath(t, repository),
					)
				}
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("初回導入 gate が失敗しました: %v", err)
	}
	if !loadCalled || !testCalled {
		t.Fatalf("loader/test の呼出しが不足しています: load=%t test=%t", loadCalled, testCalled)
	}
}

func TestRunBootstrapRejectsProviderImplementationAndNonPlannedRows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		rows []matrixRow
	}{
		{
			name: "provider adapter",
			path: "internal/source/egov/lawv2/adapter.go",
			rows: []matrixRow{{providerID: "e-gov-law-api-v2", status: "planned"}},
		},
		{
			name: "implemented row",
			rows: []matrixRow{{providerID: "e-gov-law-api-v2", status: "implemented"}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repository, base := newBootstrapRepository(t)
			if tt.path != "" {
				writeTestFile(t, repository, tt.path, "package lawv2\n")
			}
			testCalled := false
			err := runWithDependencies(
				context.Background(),
				testOptions(repository, base),
				dependencies{
					load: func(string) ([]matrixRow, error) {
						return append([]matrixRow(nil), tt.rows...), nil
					},
					test: func(context.Context, string, io.Writer, io.Writer) error {
						testCalled = true
						return nil
					},
				},
			)
			if err == nil {
				t.Fatal("不正な初回導入が成功しました")
			}
			if testCalled {
				t.Fatal("静的検査の失敗後に test が実行されました")
			}
		})
	}
}

func TestRunPropagatesLoaderAndTestFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		dependencies dependencies
		want         string
	}{
		{
			name: "loader",
			dependencies: dependencies{
				load: func(string) ([]matrixRow, error) {
					return nil, errors.New("matrix failure")
				},
				test: func(context.Context, string, io.Writer, io.Writer) error {
					t.Fatal("loader failure の後に test が実行されました")
					return nil
				},
			},
			want: "matrix failure",
		},
		{
			name: "go test",
			dependencies: dependencies{
				load: func(string) ([]matrixRow, error) {
					return []matrixRow{{providerID: "provider-a", status: "planned"}}, nil
				},
				test: func(context.Context, string, io.Writer, io.Writer) error {
					return errors.New("go test failure")
				},
			},
			want: "go test failure",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repository, base := newBootstrapRepository(t)
			err := runWithDependencies(
				context.Background(),
				testOptions(repository, base),
				tt.dependencies,
			)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("失敗が伝播しませんでした: %v", err)
			}
		})
	}
}

func TestRunNormalChangeSkipsConformanceTestOutsideProviderScope(t *testing.T) {
	t.Parallel()

	repository := newTestGitRepository(t, map[string]string{
		"go.mod":                 "module github.com/example/project\n\ngo 1.25.0\n",
		canonicalSchemaPath:      "{}\n",
		canonicalLoaderPath:      "package providerconformance\n",
		canonicalCommandPath:     "package main\n",
		"README.md":              "変更前\n",
		"internal/source/a/a.go": "package a\n",
	})
	base := gitOutput(t, repository, "rev-parse", "HEAD")
	writeTestFile(t, repository, "README.md", "変更後\n")

	loadCalled := false
	testCalled := false
	err := runWithDependencies(
		context.Background(),
		testOptions(repository, base),
		dependencies{
			load: func(string) ([]matrixRow, error) {
				loadCalled = true
				return []matrixRow{{
					providerID:    "provider-a",
					implementedBy: "internal/source/a",
					status:        "implemented",
				}}, nil
			},
			test: func(context.Context, string, io.Writer, io.Writer) error {
				testCalled = true
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("provider 対象外の通常変更が失敗しました: %v", err)
	}
	if !loadCalled {
		t.Fatal("適用範囲の判定前に matrix が読み込まれていません")
	}
	if testCalled {
		t.Fatal("provider 対象外の通常変更で conformance test が実行されました")
	}
}

func TestRunNormalProviderInfrastructureChangeRunsConformanceTest(t *testing.T) {
	t.Parallel()

	repository := newTestGitRepository(t, map[string]string{
		"go.mod":             "module github.com/example/project\n\ngo 1.25.0\n",
		canonicalSchemaPath:  "{}\n",
		canonicalLoaderPath:  "package providerconformance\n",
		canonicalCommandPath: "package main\n",
		"internal/provideronboarding/conformance.go": "package provideronboarding\n",
		"internal/source/a/a.go":                     "package a\n",
	})
	base := gitOutput(t, repository, "rev-parse", "HEAD")
	writeTestFile(
		t,
		repository,
		"internal/provideronboarding/conformance.go",
		"package provideronboarding\n\nconst changed = true\n",
	)

	testCalled := false
	err := runWithDependencies(
		context.Background(),
		testOptions(repository, base),
		dependencies{
			load: func(string) ([]matrixRow, error) {
				return []matrixRow{{
					providerID:    "provider-a",
					implementedBy: "internal/source/a",
					status:        "implemented",
				}}, nil
			},
			test: func(context.Context, string, io.Writer, io.Writer) error {
				testCalled = true
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("provider conformance 基盤変更が失敗しました: %v", err)
	}
	if !testCalled {
		t.Fatal("provider conformance 基盤変更で conformance test が実行されませんでした")
	}
}

func TestRunNormalProviderChangeRunsConformanceTest(t *testing.T) {
	t.Parallel()

	repository := newTestGitRepository(t, map[string]string{
		"go.mod":                 "module github.com/example/project\n\ngo 1.25.0\n",
		canonicalSchemaPath:      "{}\n",
		canonicalLoaderPath:      "package providerconformance\n",
		canonicalCommandPath:     "package main\n",
		"internal/source/a/a.go": "package a\n",
	})
	base := gitOutput(t, repository, "rev-parse", "HEAD")
	writeTestFile(t, repository, "internal/source/a/a.go", "package a\n\nconst changed = true\n")

	testCalled := false
	err := runWithDependencies(
		context.Background(),
		testOptions(repository, base),
		dependencies{
			load: func(string) ([]matrixRow, error) {
				return []matrixRow{{
					providerID:    "provider-a",
					implementedBy: "internal/source/a",
					status:        "implemented",
				}}, nil
			},
			test: func(context.Context, string, io.Writer, io.Writer) error {
				testCalled = true
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("provider 通常変更が失敗しました: %v", err)
	}
	if !testCalled {
		t.Fatal("provider 通常変更で conformance test が実行されませんでした")
	}
}

func TestRunTreatsInvalidBaseRefAsUsageError(t *testing.T) {
	t.Parallel()

	repository, _ := newBootstrapRepository(t)
	options := testOptions(repository, "not-a-revision")
	err := runWithDependencies(
		context.Background(),
		options,
		dependencies{
			load: func(string) ([]matrixRow, error) {
				t.Fatal("不正な base ref で loader が実行されました")
				return nil, nil
			},
			test: func(context.Context, string, io.Writer, io.Writer) error {
				t.Fatal("不正な base ref で test が実行されました")
				return nil
			},
		},
	)
	if !errors.Is(err, ErrInvalidBaseRef) {
		t.Fatalf("usage error ではありません: %v", err)
	}
}

func TestRunFailsClosedOutsideGitRepository(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	err := runWithDependencies(
		context.Background(),
		testOptions(repository, "HEAD"),
		dependencies{
			load: func(string) ([]matrixRow, error) {
				t.Fatal("VCS failure の後に loader が実行されました")
				return nil, nil
			},
			test: func(context.Context, string, io.Writer, io.Writer) error {
				t.Fatal("VCS failure の後に test が実行されました")
				return nil
			},
		},
	)
	if err == nil {
		t.Fatal("Git repository 外で gate が成功しました")
	}
	if errors.Is(err, ErrInvalidBaseRef) {
		t.Fatalf("VCS failure が usage error になりました: %v", err)
	}
}

func newBootstrapRepository(t *testing.T) (string, string) {
	t.Helper()

	repository := newTestGitRepository(t, map[string]string{
		"go.mod": "module github.com/example/bootstrap\n\ngo 1.25.0\n",
	})
	base := gitOutput(t, repository, "rev-parse", "HEAD")
	writeTestFile(t, repository, canonicalSchemaPath, "{}\n")
	writeTestFile(t, repository, canonicalLoaderPath, "package providerconformance\n")
	writeTestFile(t, repository, canonicalCommandPath, "package main\n")
	writeTestFile(
		t,
		repository,
		"conformance/providers/provider-a.yaml",
		"schemaVersion: 1\nrows: []\n",
	)
	return repository, base
}

func testOptions(repository, base string) Options {
	return Options{
		Repository:         repository,
		GitRepository:      repository,
		BaseRef:            base,
		HeadRef:            "HEAD",
		IncludeIndex:       true,
		IncludeWorkingTree: true,
		IncludeUntracked:   true,
		Stdout:             &bytes.Buffer{},
		Stderr:             &bytes.Buffer{},
	}
}

func testCommand(ctx context.Context, repository, name string, args ...string) *exec.Cmd {
	//nolint:gosec // SOT-ENG-018: テストは固定した git 実行ファイルを argv で起動する。
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = repository
	return command
}

func trimCommandOutput(output []byte) string {
	return strings.TrimSuffix(strings.TrimSuffix(string(output), "\n"), "\r")
}

func resolvedTestPath(t *testing.T, value string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(value)
	if err != nil {
		t.Fatalf("テストパスを解決できませんでした: %v", err)
	}
	return resolved
}
