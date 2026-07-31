package legalqueryeval

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBaselineRepositoryは初回Bootstrapのbyte一致だけを受理する(t *testing.T) {
	t.Parallel()

	const verificationID = "evaluation-baseline-initial-bootstrap"
	root := t.TempDir()
	baseline := prepareTestBaselineRepository(t, root)

	artifact, err := loadBaselineRepository(
		context.Background(),
		root,
		"default-1",
	)
	if err != nil {
		t.Fatalf("%s: bootstrap baseline を読み込めません: %v", verificationID, err)
	}
	if got := artifact.RawBytes(); string(got) != string(baseline) {
		t.Fatalf("%s: raw bytes が現行 baseline と一致しません", verificationID)
	}
	if got := artifact.SHA256(); got == "" || len(got) != 64 {
		t.Fatalf("%s: sha256 = %q", verificationID, got)
	}
	if artifact.Report().BaselineVersion() != "default-1" {
		t.Fatalf(
			"%s: baselineVersion = %q",
			verificationID,
			artifact.Report().BaselineVersion(),
		)
	}

	writeTestBaselineFile(
		t,
		root,
		"testdata/legalquery/baselines/versions/default-1.json",
		append(append([]byte{}, baseline...), ' '),
	)
	if _, err := loadBaselineRepository(
		context.Background(),
		root,
		"default-1",
	); err == nil {
		t.Fatal(verificationID + ": default.json と version file の byte 不一致を受理しました")
	}
}

func TestValidateBaselineHistoryBudgetは上限超過を拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "evaluation-baseline-resource-maximum"
	if err := validateBaselineHistoryBudget(4096, 256<<20); err != nil {
		t.Fatalf("%s: 上限内を拒否しました: %v", verificationID, err)
	}
	if err := validateBaselineHistoryBudget(4097, 256<<20); err == nil {
		t.Fatal(verificationID + ": version file 件数 +1 を受理しました")
	}
	if err := validateBaselineHistoryBudget(4096, (256<<20)+1); err == nil {
		t.Fatal(verificationID + ": history byte 合計 +1 を受理しました")
	}
}

func TestLoadBaselineRepositoryは未知entryとsymlinkを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "evaluation-baseline-history-bounds"
	root := t.TempDir()
	prepareTestBaselineRepository(t, root)
	writeTestBaselineFile(
		t,
		root,
		"testdata/legalquery/baselines/unknown.txt",
		[]byte("unexpected\n"),
	)
	if _, err := loadBaselineRepository(
		context.Background(),
		root,
		"default-1",
	); err == nil {
		t.Fatal(verificationID + ": baselines/ の未知 entry を受理しました")
	}

	root = t.TempDir()
	prepareTestBaselineRepository(t, root)
	target := filepath.Join(root, "outside.json")
	if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("%s: symlink target を書けません: %v", verificationID, err)
	}
	link := filepath.Join(
		root,
		"testdata/legalquery/baselines/versions/default-2.json",
	)
	if err := os.MkdirAll(filepath.Dir(link), 0o750); err != nil {
		t.Fatalf("%s: symlink dir を作れません: %v", verificationID, err)
	}
	if err := os.Symlink(target, link); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("symlink を作成できない環境です")
		}
		t.Fatalf("%s: symlink を作れません: %v", verificationID, err)
	}
	if _, err := loadBaselineRepository(
		context.Background(),
		root,
		"default-1",
	); err == nil {
		t.Fatal(verificationID + ": versions/ の symlink を受理しました")
	}
}

func prepareTestBaselineRepository(t *testing.T, root string) []byte {
	t.Helper()

	baseline := readRepositoryTestFile(
		t,
		"testdata/legalquery/baselines/default.json",
	)
	schema := readRepositoryTestFile(
		t,
		"testdata/legalquery/schemas/legal-query-baseline-v1.schema.json",
	)
	writeTestBaselineFile(
		t,
		root,
		"testdata/legalquery/schemas/legal-query-baseline-v1.schema.json",
		schema,
	)
	writeTestBaselineFile(
		t,
		root,
		"testdata/legalquery/baselines/default.json",
		baseline,
	)
	writeTestBaselineFile(
		t,
		root,
		"testdata/legalquery/baselines/versions/default-1.json",
		baseline,
	)
	return baseline
}

func readRepositoryTestFile(t *testing.T, relative string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("repository test fixture を読めません: %v", err)
	}
	return data
}

func writeTestBaselineFile(
	t *testing.T,
	root string,
	relative string,
	content []byte,
) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("test fixture dir を作れません: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("test fixture file を書けません: %v", err)
	}
}
