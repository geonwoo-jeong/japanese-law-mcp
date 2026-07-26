package provideronboarding

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// SOT-ENG-017/018: command は通常テストと同じ canonical loader を再利用する。
func TestLoadCanonicalRowsUsesProviderConformanceCatalog(t *testing.T) {
	t.Parallel()

	rows, err := loadCanonicalRows(testRepositoryRoot(t))
	if err != nil {
		t.Fatalf("canonical matrix を読み込めませんでした: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("row 数 = %d, want 4", len(rows))
	}
	for _, row := range rows {
		if row.providerID != "e-gov-law-api-v2" {
			t.Fatalf("providerId = %q", row.providerID)
		}
		if row.implementedBy != "internal/source/egov/lawv2" {
			t.Fatalf("implementedBy = %q", row.implementedBy)
		}
		if row.status != "planned" {
			t.Fatalf("status = %q", row.status)
		}
	}
}

// SOT-ENG-018: 再帰を避け、providerconformance package の通常テストだけを実行する。
func TestRunProviderConformanceTests(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runProviderConformanceTests(
		context.Background(),
		testRepositoryRoot(t),
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("providerconformance test が失敗しました: %v\n%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "/internal/providerconformance") {
		t.Fatalf("実行した package を確認できません: %q", stdout.String())
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
