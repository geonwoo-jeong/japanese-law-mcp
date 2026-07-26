package provideronboarding

import (
	"strings"
	"testing"
)

// SOT-ENG-017/018: provider package 間の import は Go AST で拒否する。
func TestValidateProviderImportsRejectsAnotherProvider(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	writeTestFile(
		t,
		repository,
		"go.mod",
		"module github.com/example/provider-test\n\ngo 1.25.0\n",
	)
	writeTestFile(
		t,
		repository,
		"internal/source/a/v1/adapter.go",
		"package v1\n\nimport _ \"github.com/example/provider-test/internal/source/b/v1\"\n",
	)
	writeTestFile(
		t,
		repository,
		"internal/source/b/v1/adapter.go",
		"package v1\n",
	)

	rows := []matrixRow{
		{providerID: "provider-a", implementedBy: "internal/source/a/v1"},
		{providerID: "provider-b", implementedBy: "internal/source/b/v1"},
	}
	err := validateProviderImports(repository, rows)
	if err == nil {
		t.Fatal("provider package 間の import が許可されました")
	}
	if !strings.Contains(err.Error(), "provider-a") ||
		!strings.Contains(err.Error(), "provider-b") {
		t.Fatalf("import 元と先を特定できないエラーです: %v", err)
	}
}

func TestValidateProviderImportsAllowsProviderNeutralPackage(t *testing.T) {
	t.Parallel()

	repository := t.TempDir()
	writeTestFile(
		t,
		repository,
		"go.mod",
		"module github.com/example/provider-test\n\ngo 1.25.0\n",
	)
	writeTestFile(
		t,
		repository,
		"internal/source/a/v1/adapter.go",
		"package v1\n\nimport _ \"github.com/example/provider-test/internal/source/shared\"\n",
	)

	rows := []matrixRow{{
		providerID:    "provider-a",
		implementedBy: "internal/source/a/v1",
	}}
	if err := validateProviderImports(repository, rows); err != nil {
		t.Fatalf("provider-neutral package の import が拒否されました: %v", err)
	}
}

func TestCollectProviderPackagesRejectsInvalidOrOverlappingOwnership(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rows []matrixRow
	}{
		{
			name: "empty provider id",
			rows: []matrixRow{{implementedBy: "internal/source/a/v1"}},
		},
		{
			name: "outside provider root",
			rows: []matrixRow{{
				providerID:    "provider-a",
				implementedBy: "internal/model",
			}},
		},
		{
			name: "overlapping providers",
			rows: []matrixRow{
				{providerID: "provider-a", implementedBy: "internal/source/a"},
				{providerID: "provider-b", implementedBy: "internal/source/a/v1"},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := collectProviderPackages(tt.rows); err == nil {
				t.Fatalf("不正な provider package 所有関係が許可されました: %#v", tt.rows)
			}
		})
	}
}
