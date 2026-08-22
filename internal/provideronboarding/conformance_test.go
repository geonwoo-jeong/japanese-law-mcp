package provideronboarding

import (
	"context"
	"io"
	"path/filepath"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/providerconformance"
)

// SOT-ENG-017/018: command は通常テストと同じ canonical loader を再利用する。
func TestLoadCanonicalRowsUsesProviderConformanceCatalog(t *testing.T) {
	t.Parallel()

	rows, err := loadCanonicalRows(testRepositoryRoot(t))
	if err != nil {
		t.Fatalf("canonical matrix を読み込めませんでした: %v", err)
	}
	if len(rows) != 7 {
		t.Fatalf("row 数 = %d, want 7", len(rows))
	}
	counts := map[string]int{}
	for _, row := range rows {
		counts[row.providerID]++
		switch row.providerID {
		case "courts-hanrei-html":
			if row.implementedBy != "internal/source/courts/hanrei" {
				t.Fatalf("courts implementedBy = %q", row.implementedBy)
			}
			if row.status != "implemented" {
				t.Fatalf("courts status = %q", row.status)
			}
		case "e-gov-law-api-v1":
			if row.implementedBy != "internal/source/egov/lawv1" {
				t.Fatalf("v1 implementedBy = %q", row.implementedBy)
			}
			if row.status != "implemented" {
				t.Fatalf("v1 status = %q", row.status)
			}
		case "e-gov-law-api-v2":
			if row.implementedBy != "internal/source/egov/lawv2" {
				t.Fatalf("v2 implementedBy = %q", row.implementedBy)
			}
			if row.status != "implemented" {
				t.Fatalf("v2 status = %q", row.status)
			}
		default:
			t.Fatalf("providerId = %q", row.providerID)
		}
	}
	if counts["courts-hanrei-html"] != 2 ||
		counts["e-gov-law-api-v1"] != 1 ||
		counts["e-gov-law-api-v2"] != 4 {
		t.Fatalf("provider row counts = %v", counts)
	}
}

// SOT-ENG-017/018: matrix の implemented target を通常テストとして実行する。
func TestRunProviderConformanceTestsUsesImplementedMatrixTargets(t *testing.T) {
	t.Parallel()

	var arguments []string
	err := runProviderConformanceTestsWith(
		context.Background(),
		testRepositoryRoot(t),
		io.Discard,
		io.Discard,
		func(
			_ context.Context,
			_ string,
			got []string,
			_, _ io.Writer,
		) error {
			arguments = append([]string(nil), got...)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("provider conformance test の実行準備が失敗しました: %v", err)
	}
	want := []string{
		"test",
		"-count=1",
		"./internal/providerconformance",
		"./internal/source/courts/hanrei",
		"./internal/source/egov/lawv1",
		"./internal/source/egov/lawv2",
	}
	if !slices.Equal(arguments, want) {
		t.Fatalf("go arguments = %q, want %q", arguments, want)
	}
}

func TestProviderConformanceTestTargetsUseImplementedRowsOnce(t *testing.T) {
	t.Parallel()

	rows := []providerconformance.Row{
		{Status: "implemented", ConformanceTarget: "./internal/source/provider-b"},
		{Status: "implemented", ConformanceTarget: "./internal/source/provider-b"},
		{Status: "planned", ConformanceTarget: "./internal/source/provider-c"},
		{Status: "retired", ConformanceTarget: "./internal/source/provider-d"},
		{Status: "implemented", ConformanceTarget: "./internal/source/provider-a"},
	}
	want := []string{
		"./internal/providerconformance",
		"./internal/source/provider-a",
		"./internal/source/provider-b",
	}
	if got := providerConformanceTestTargets(rows); !slices.Equal(got, want) {
		t.Fatalf("conformance targets = %q, want %q", got, want)
	}
}

func testRepositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repository root を解決できません: %v", err)
	}
	return root
}
