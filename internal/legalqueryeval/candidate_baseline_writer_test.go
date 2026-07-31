package legalqueryeval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteCandidateBaselineは検証済みbyteだけを予約版へ新規作成する(
	t *testing.T,
) {
	t.Parallel()

	const verificationID = "evaluation-baseline-candidate-isolation"
	repositoryRoot, current := prepareCandidateBaselineWriterRepository(t)
	candidate := candidateBaselineRaw(t, "default-2")

	if err := WriteCandidateBaseline(
		context.Background(),
		repositoryRoot,
		"default-2",
		candidate,
	); err != nil {
		t.Fatalf("%s: 候補 baseline を作成できません: %v", verificationID, err)
	}

	versionPath := filepath.Join(
		repositoryRoot,
		"testdata",
		"legalquery",
		"baselines",
		"versions",
		"default-2.json",
	)
	written, err := readCandidateBaselineRootFile(
		repositoryRoot,
		filepath.Join("testdata", "legalquery", "baselines", "versions", "default-2.json"),
	)
	if err != nil {
		t.Fatalf("%s: 作成済み候補を読めません: %v", verificationID, err)
	}
	if !bytes.Equal(written, candidate) {
		t.Fatal(verificationID + ": caller が渡した report byte から変化しました")
	}
	info, err := os.Lstat(versionPath)
	if err != nil {
		t.Fatalf("%s: 候補 baseline を確認できません: %v", verificationID, err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("%s: 候補 baseline の mode = %v", verificationID, info.Mode())
	}

	after, err := readCandidateBaselineRootFile(
		repositoryRoot,
		filepath.Join("testdata", "legalquery", "baselines", "default.json"),
	)
	if err != nil {
		t.Fatalf("%s: default baseline を確認できません: %v", verificationID, err)
	}
	if !bytes.Equal(after, current) {
		t.Fatal(verificationID + ": default.json を変更しました")
	}

	if err := WriteCandidateBaseline(
		context.Background(),
		repositoryRoot,
		"default-2",
		candidate,
	); err == nil {
		t.Fatal(verificationID + ": 使用済み予約版を再利用しました")
	}
	writtenAgain, err := readCandidateBaselineRootFile(
		repositoryRoot,
		filepath.Join("testdata", "legalquery", "baselines", "versions", "default-2.json"),
	)
	if err != nil {
		t.Fatalf("%s: 既存候補を再確認できません: %v", verificationID, err)
	}
	if !bytes.Equal(writtenAgain, candidate) {
		t.Fatal(verificationID + ": 既存候補を上書きしました")
	}
}

func TestWriteCandidateBaselineは不正な版とreportByteを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "evaluation-baseline-candidate-isolation"
	candidate := candidateBaselineRaw(t, "default-2")
	tests := []struct {
		name    string
		version string
		raw     []byte
		cancel  bool
	}{
		{name: "path traversal", version: "../default-2", raw: candidate},
		{name: "non canonical version", version: "default-02", raw: candidate},
		{name: "report version mismatch", version: "default-3", raw: candidate},
		{name: "failed acceptance", version: "default-2", raw: failedCandidateBaselineRaw(t, "default-2")},
		{name: "non canonical bytes", version: "default-2", raw: append(append([]byte{}, candidate...), ' ')},
		{name: "empty bytes", version: "default-2", raw: nil},
		{name: "oversized bytes", version: "default-2", raw: bytes.Repeat([]byte{'x'}, maximumStandardBaselineBytes+1)},
		{name: "cancelled", version: "default-2", raw: candidate, cancel: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			repositoryRoot, _ := prepareCandidateBaselineWriterRepository(t)
			ctx := context.Background()
			if test.cancel {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			}
			if err := WriteCandidateBaseline(
				ctx,
				repositoryRoot,
				test.version,
				test.raw,
			); err == nil {
				t.Fatalf("%s: 不正入力 %q を受理しました", verificationID, test.name)
			}
			assertCandidateBaselineAbsent(t, repositoryRoot, "default-2")
			assertCandidateBaselineAbsent(t, repositoryRoot, "default-3")
		})
	}
}

func TestWriteCandidateBaselineは既存と非通常fileを上書きしない(t *testing.T) {
	t.Parallel()

	const verificationID = "evaluation-baseline-candidate-isolation"
	candidate := candidateBaselineRaw(t, "default-2")
	t.Run("existing regular", func(t *testing.T) {
		t.Parallel()
		repositoryRoot, _ := prepareCandidateBaselineWriterRepository(t)
		target := candidateBaselineTestPath(repositoryRoot, "default-2")
		existing := []byte("既存予約\n")
		if err := os.WriteFile(target, existing, 0o600); err != nil {
			t.Fatalf("既存予約を準備できません: %v", err)
		}
		if err := WriteCandidateBaseline(
			context.Background(), repositoryRoot, "default-2", candidate,
		); err == nil {
			t.Fatal(verificationID + ": 既存 file を上書きしました")
		}
		got, err := readCandidateBaselineRootFile(
			repositoryRoot,
			filepath.Join("testdata", "legalquery", "baselines", "versions", "default-2.json"),
		)
		if err != nil || !bytes.Equal(got, existing) {
			t.Fatal(verificationID + ": 既存 file の byte が変化しました")
		}
	})

	t.Run("directory", func(t *testing.T) {
		t.Parallel()
		repositoryRoot, _ := prepareCandidateBaselineWriterRepository(t)
		target := candidateBaselineTestPath(repositoryRoot, "default-2")
		if err := os.Mkdir(target, 0o750); err != nil {
			t.Fatalf("非通常 path を準備できません: %v", err)
		}
		if err := WriteCandidateBaseline(
			context.Background(), repositoryRoot, "default-2", candidate,
		); err == nil {
			t.Fatal(verificationID + ": directory を上書きしました")
		}
		info, err := os.Lstat(target)
		if err != nil || !info.IsDir() {
			t.Fatal(verificationID + ": 既存 directory が変化しました")
		}
	})

	t.Run("symlink", func(t *testing.T) {
		t.Parallel()
		repositoryRoot, _ := prepareCandidateBaselineWriterRepository(t)
		outside := filepath.Join(repositoryRoot, "outside.json")
		outsideBytes := []byte("外部 file\n")
		if err := os.WriteFile(outside, outsideBytes, 0o600); err != nil {
			t.Fatalf("symlink target を準備できません: %v", err)
		}
		target := candidateBaselineTestPath(repositoryRoot, "default-2")
		if err := os.Symlink(outside, target); err != nil {
			if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "operation not permitted") {
				t.Skip("symlink を作成できない環境です")
			}
			t.Fatalf("symlink を準備できません: %v", err)
		}
		if err := WriteCandidateBaseline(
			context.Background(), repositoryRoot, "default-2", candidate,
		); err == nil {
			t.Fatal(verificationID + ": symlink を上書きしました")
		}
		got, err := readCandidateBaselineRootFile(repositoryRoot, "outside.json")
		if err != nil || !bytes.Equal(got, outsideBytes) {
			t.Fatal(verificationID + ": repository 外 file が変化しました")
		}
	})
}

func TestWriteCandidateBaselineはsymlinkのpathComponentを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "evaluation-baseline-candidate-isolation"
	repositoryRoot, _ := prepareCandidateBaselineWriterRepository(t)
	versions := filepath.Dir(candidateBaselineTestPath(repositoryRoot, "default-2"))
	original := versions + "-original"
	if err := os.Rename(versions, original); err != nil {
		t.Fatalf("versions directory を退避できません: %v", err)
	}
	if err := os.Symlink(original, versions); err != nil {
		if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("symlink を作成できない環境です")
		}
		t.Fatalf("versions symlink を準備できません: %v", err)
	}
	if err := WriteCandidateBaseline(
		context.Background(),
		repositoryRoot,
		"default-2",
		candidateBaselineRaw(t, "default-2"),
	); err == nil {
		t.Fatal(verificationID + ": symlink の path component を受理しました")
	}
	if _, err := os.Lstat(filepath.Join(original, "default-2.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(verificationID + ": symlink 配下へ候補を作成しました")
	}
}

func TestWriteCandidateBaselineはsymlinkのrepositoryRootを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "evaluation-baseline-candidate-isolation"
	repositoryRoot, _ := prepareCandidateBaselineWriterRepository(t)
	rootLink := filepath.Join(t.TempDir(), "repository-link")
	if err := os.Symlink(repositoryRoot, rootLink); err != nil {
		if errors.Is(err, os.ErrPermission) || strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("symlink を作成できない環境です")
		}
		t.Fatalf("repository symlink を準備できません: %v", err)
	}
	if err := WriteCandidateBaseline(
		context.Background(),
		rootLink,
		"default-2",
		candidateBaselineRaw(t, "default-2"),
	); err == nil {
		t.Fatal(verificationID + ": symlink の repository root を受理しました")
	}
	assertCandidateBaselineAbsent(t, repositoryRoot, "default-2")
}

func TestWriteCandidateBaselineはhistoryの未知entryを拒否する(t *testing.T) {
	t.Parallel()

	const verificationID = "evaluation-baseline-history-bounds"
	repositoryRoot, _ := prepareCandidateBaselineWriterRepository(t)
	unknown := filepath.Join(
		filepath.Dir(candidateBaselineTestPath(repositoryRoot, "default-2")),
		"unknown.txt",
	)
	if err := os.WriteFile(unknown, []byte("unknown\n"), 0o600); err != nil {
		t.Fatalf("未知 entry を準備できません: %v", err)
	}
	if err := WriteCandidateBaseline(
		context.Background(),
		repositoryRoot,
		"default-2",
		candidateBaselineRaw(t, "default-2"),
	); err == nil {
		t.Fatal(verificationID + ": history の未知 entry を無視しました")
	}
	assertCandidateBaselineAbsent(t, repositoryRoot, "default-2")
}

func TestValidateCandidateBaselineHistoryBudgetは追加後の上限超過を拒否する(
	t *testing.T,
) {
	t.Parallel()

	const verificationID = "evaluation-baseline-resource-maximum"
	if err := validateCandidateBaselineHistoryBudget(4095, (256<<20)-1, 1); err != nil {
		t.Fatalf("%s: 上限内の追加を拒否しました: %v", verificationID, err)
	}
	if err := validateCandidateBaselineHistoryBudget(4096, 1, 1); err == nil {
		t.Fatal(verificationID + ": version file 件数 +1 を受理しました")
	}
	if err := validateCandidateBaselineHistoryBudget(1, 256<<20, 1); err == nil {
		t.Fatal(verificationID + ": history byte 合計 +1 を受理しました")
	}
}

func prepareCandidateBaselineWriterRepository(t *testing.T) (string, []byte) {
	t.Helper()

	repositoryRoot := t.TempDir()
	schema := readRepositoryTestFile(
		t,
		"testdata/legalquery/schemas/legal-query-baseline-v1.schema.json",
	)
	current := candidateBaselineRaw(t, "default-1")
	writeTestBaselineFile(
		t,
		repositoryRoot,
		"testdata/legalquery/schemas/legal-query-baseline-v1.schema.json",
		schema,
	)
	writeTestBaselineFile(
		t,
		repositoryRoot,
		"testdata/legalquery/baselines/default.json",
		current,
	)
	writeTestBaselineFile(
		t,
		repositoryRoot,
		"testdata/legalquery/baselines/versions/default-1.json",
		current,
	)
	return repositoryRoot, current
}

func candidateBaselineRaw(t *testing.T, version string) []byte {
	t.Helper()

	report := mustStandardReport(t, 1, []ExecutionCaseEvaluation{
		mustExecutionCaseEvaluation(t, "execution-intent-01"),
	})
	return candidateBaselineRawFromReport(t, report, version)
}

func failedCandidateBaselineRaw(t *testing.T, version string) []byte {
	t.Helper()

	evaluation, err := NewExecutionCaseEvaluation(ExecutionCaseEvaluationValues{
		CaseID:              "execution-intent-01",
		ExpectedMatched:     false,
		AttemptOrderMatched: true,
	})
	if err != nil {
		t.Fatalf("不合格候補の execution evaluation を作成できません: %v", err)
	}
	report := mustStandardReport(t, 1, []ExecutionCaseEvaluation{evaluation})
	return candidateBaselineRawFromReport(t, report, version)
}

func candidateBaselineRawFromReport(
	t *testing.T,
	report StandardReport,
	version string,
) []byte {
	t.Helper()

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("候補 report を JSON 化できません: %v", err)
	}
	raw = bytes.Replace(
		raw,
		[]byte(`"baselineVersion":"default-1"`),
		[]byte(`"baselineVersion":"`+version+`"`),
		1,
	)
	return append(raw, '\n')
}

func candidateBaselineTestPath(repositoryRoot, version string) string {
	return filepath.Join(
		repositoryRoot,
		"testdata",
		"legalquery",
		"baselines",
		"versions",
		version+".json",
	)
}

func readCandidateBaselineRootFile(repositoryRoot, relative string) ([]byte, error) {
	root, err := os.OpenRoot(repositoryRoot)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	return root.ReadFile(filepath.ToSlash(relative))
}

func assertCandidateBaselineAbsent(t *testing.T, repositoryRoot, version string) {
	t.Helper()
	if _, err := os.Lstat(candidateBaselineTestPath(repositoryRoot, version)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("不正入力で %s.json が作成されました", version)
	}
}
