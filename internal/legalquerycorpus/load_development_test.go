package legalquerycorpus

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDevelopmentはmanifestとholdoutなしでdevelopmentだけを読む(
	t *testing.T,
) {
	t.Parallel()

	repositoryRoot := developmentTestRepositoryRoot(t)
	root := t.TempDir()
	copyDevelopmentTree(
		t,
		filepath.Join(repositoryRoot, "testdata", "legalquery", "schemas"),
		filepath.Join(root, "testdata", "legalquery", "schemas"),
	)
	copyDevelopmentTree(
		t,
		filepath.Join(
			repositoryRoot,
			"testdata",
			"legalquery",
			"corpus-v10",
			"development",
		),
		filepath.Join(
			root,
			"testdata",
			"legalquery",
			"corpus-v10",
			"development",
		),
	)

	development, err := LoadDevelopment(
		context.Background(),
		root,
		"testdata/legalquery/corpus-v10/development",
	)
	if err != nil {
		t.Fatalf(
			"SOT-ENG-024/SOT-ENG-026: LoadDevelopment() error = %v",
			err,
		)
	}
	if development.CorpusVersion() != "corpus-v10" ||
		development.SchemaVersion() != corpusSchemaVersionV2 ||
		!manifestSHA256Pattern.MatchString(development.ContentDigest()) ||
		len(development.Cases()) != repositoryCorpusV10Development {
		t.Fatalf(
			"SOT-ENG-024/SOT-ENG-026: development identity = %q/%d/%q/%d",
			development.CorpusVersion(),
			development.SchemaVersion(),
			development.ContentDigest(),
			len(development.Cases()),
		)
	}
	cases := development.Cases()
	firstCaseID := cases[0].CaseID()
	cases[0] = cases[1]
	if development.Cases()[0].CaseID() != firstCaseID {
		t.Fatal("SOT-ENG-024/SOT-ENG-026: Cases() が共有 slice を返しました")
	}
	fixtureRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatalf("SOT-ENG-024: test repository を開けません: %v", err)
	}
	fixture := filepath.Join(
		"testdata",
		"legalquery",
		"corpus-v10",
		"development",
		firstCaseID+".json",
	)
	file, err := fixtureRoot.OpenFile(fixture, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		_ = fixtureRoot.Close()
		t.Fatalf("SOT-ENG-024: development fixture を開けません: %v", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		_ = file.Close()
		t.Fatalf("SOT-ENG-024: development fixture を変更できません: %v", err)
	}
	if err := file.Close(); err != nil {
		_ = fixtureRoot.Close()
		t.Fatalf("SOT-ENG-024: development fixture を閉じられません: %v", err)
	}
	if err := fixtureRoot.Close(); err != nil {
		t.Fatalf("SOT-ENG-024: test repository を閉じられません: %v", err)
	}
	changed, err := LoadDevelopment(
		context.Background(),
		root,
		"testdata/legalquery/corpus-v10/development",
	)
	if err != nil {
		t.Fatalf("SOT-ENG-024: 空白変更後の fixture を読めません: %v", err)
	}
	if changed.ContentDigest() == development.ContentDigest() ||
		changed.Cases()[0].CaseID() != firstCaseID {
		t.Fatal("SOT-ENG-024: 原 byte の変更が content digest に反映されません")
	}
}

func TestLoadDevelopmentは取消済みContextを拒否する(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LoadDevelopment(
		ctx,
		developmentTestRepositoryRoot(t),
		"testdata/legalquery/corpus-v10/development",
	); err == nil {
		t.Fatal("SOT-ENG-024/SOT-ENG-026: 取消済み context を受理しました")
	}
}

func TestResolveDevelopmentFilesystemPathsはdevelopmentDirectoryだけを受理する(
	t *testing.T,
) {
	t.Parallel()

	repositoryRoot := developmentTestRepositoryRoot(t)
	tests := []string{
		"",
		"development",
		"testdata/legalquery/corpus-v10",
		"testdata/legalquery/corpus-v10/holdout",
		"testdata/legalquery/corpus-v10/development/..",
		"testdata/legalquery/corpus-v10/development/child",
	}
	for _, testCase := range tests {
		if _, err := resolveDevelopmentFilesystemPaths(
			repositoryRoot,
			testCase,
		); err == nil {
			t.Fatalf(
				"SOT-ENG-026: resolveDevelopmentFilesystemPaths(%q) が成功しました",
				testCase,
			)
		}
	}
}

func TestCopyDevelopmentTreeはSymlinkを拒否する(t *testing.T) {
	t.Parallel()

	source := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte("{}"), 0o600); err != nil {
		t.Fatalf("SOT-ENG-024: symlink target を作成できません: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(source, "linked.json")); err != nil {
		t.Fatalf("SOT-ENG-024: symlink fixture を作成できません: %v", err)
	}
	if err := copyDevelopmentTreeFiles(source, t.TempDir()); err == nil {
		t.Fatal("SOT-ENG-024: development copy が symlink を受理しました")
	}
}

func developmentTestRepositoryRoot(t *testing.T) string {
	t.Helper()

	return filepath.Clean(filepath.Join(
		filepath.Dir(corpusSchemaV1Path(t)),
		"..",
		"..",
		"..",
	))
}

func copyDevelopmentTree(t *testing.T, source string, target string) {
	t.Helper()

	if err := copyDevelopmentTreeFiles(source, target); err != nil {
		t.Fatalf("SOT-ENG-026: development tree を複製できません: %v", err)
	}
}

func copyDevelopmentTreeFiles(source string, target string) (err error) {
	if err := os.MkdirAll(target, 0o750); err != nil {
		return err
	}
	sourceInfo, err := os.Lstat(source)
	if err != nil || sourceInfo.Mode()&os.ModeSymlink != 0 ||
		!sourceInfo.IsDir() {
		return fmt.Errorf("development tree の root が有効ではありません")
	}
	sourceRoot, err := os.OpenRoot(source)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, sourceRoot.Close()) }()
	openedSource, err := sourceRoot.Stat(".")
	if err != nil || !os.SameFile(sourceInfo, openedSource) {
		return fmt.Errorf("development tree の root が変化しました")
	}
	targetInfo, err := os.Lstat(target)
	if err != nil || targetInfo.Mode()&os.ModeSymlink != 0 ||
		!targetInfo.IsDir() {
		return fmt.Errorf("development copy の root が有効ではありません")
	}
	targetRoot, err := os.OpenRoot(target)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, targetRoot.Close()) }()
	openedTarget, err := targetRoot.Stat(".")
	if err != nil || !os.SameFile(targetInfo, openedTarget) {
		return fmt.Errorf("development copy の root が変化しました")
	}
	return fs.WalkDir(
		sourceRoot.FS(),
		".",
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("development tree に symlink があります")
			}
			relative := filepath.FromSlash(path)
			if entry.IsDir() {
				if relative == "." {
					return nil
				}
				return targetRoot.MkdirAll(relative, 0o750)
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return fmt.Errorf("development tree に通常 file 以外があります")
			}
			data, err := sourceRoot.ReadFile(relative)
			if err != nil {
				return err
			}
			current, err := sourceRoot.Lstat(relative)
			if err != nil || !current.Mode().IsRegular() ||
				!os.SameFile(info, current) {
				return fmt.Errorf("development tree の file が変化しました")
			}
			return targetRoot.WriteFile(relative, data, 0o600)
		},
	)
}
