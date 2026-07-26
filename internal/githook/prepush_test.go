package githook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/provideronboarding"
)

func TestPrePushChecksUniqueCommitsForMultipleRefsAndSkipsDeletion(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "順序.txt", "A\n")
	commitA := commitAll(t, repository, "A")
	writeFile(t, repository, "順序.txt", "B\n")
	commitB := commitAll(t, repository, "B")
	writeFile(t, repository, "順序.txt", "C\n")
	commitC := commitAll(t, repository, "C")
	writeFile(t, repository, "順序.txt", "作業ツリーのみ\n")
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/main", commitC, "refs/heads/main", commitA) +
			oidLine("refs/heads/copy", commitC, "refs/heads/copy", commitA) +
			oidLine("(delete)", zeroOID(len(commitC)), "refs/heads/old", commitB),
	)
	var contents []string
	var ranges [][]string
	var snapshots []string
	app.qualityGate = func(
		_ context.Context,
		profile, snapshot, gotRepository string,
		_ string,
		gotRanges []string,
		_, _ io.Writer,
	) error {
		if profile != "pre-push" || gotRepository != repository {
			return fmt.Errorf("引数が不正です: %s %s", profile, gotRepository)
		}
		content := readTestFile(t, filepath.Join(snapshot, "順序.txt"))
		contents = append(contents, string(content))
		ranges = append(ranges, append([]string(nil), gotRanges...))
		snapshots = append(snapshots, snapshot)
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push", "origin", "example.invalid"})

	if code != 0 {
		t.Fatalf("pre-push が失敗しました: %s", stderr)
	}
	if got, want := strings.Join(contents, ""), "C\n"; got != want {
		t.Fatalf("検査した commit = %q, want %q", got, want)
	}
	if got, want := strings.Join(ranges[0], ","), commitA+".."+commitC; got != want {
		t.Fatalf("秘密情報検査の range = %q, want %q", got, want)
	}
	for _, snapshot := range snapshots {
		assertNotExists(t, snapshot)
	}
	if got := string(readTestFile(t, filepath.Join(repository, "順序.txt"))); got != "作業ツリーのみ\n" {
		t.Fatalf("作業ツリーが変更されました: %q", got)
	}
}

func TestPrePushRunsProviderGateForUniqueTipBasePairsBeforeQualityGate(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "順序.txt", "A\n")
	commitA := commitAll(t, repository, "A")
	writeFile(t, repository, "順序.txt", "B\n")
	commitB := commitAll(t, repository, "B")
	writeFile(t, repository, "順序.txt", "C\n")
	tip := commitAll(t, repository, "C")
	writeFile(t, repository, "順序.txt", "作業ツリーのみ\n")
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/first", tip, "refs/heads/first", commitA) +
			oidLine("refs/heads/first-copy", tip, "refs/heads/first-copy", commitA) +
			oidLine("refs/heads/second", tip, "refs/heads/second", commitB),
	)

	var calls []string
	var snapshots []string
	app.providerOnboarding = func(
		_ context.Context,
		options provideronboarding.Options,
	) error {
		calls = append(calls, "provider:"+options.BaseRef)
		snapshots = append(snapshots, options.Repository)
		if options.GitRepository != repository {
			return fmt.Errorf(
				"Git repository = %q, want %q",
				options.GitRepository,
				repository,
			)
		}
		if options.HeadRef != tip {
			return fmt.Errorf("head commit = %q, want %q", options.HeadRef, tip)
		}
		if options.IncludeIndex ||
			options.IncludeWorkingTree ||
			options.IncludeUntracked {
			return errors.New(
				"commit snapshot に index、working tree または未追跡 file が含まれています",
			)
		}
		if options.Stdout != app.stdout || options.Stderr != app.stderr {
			return errors.New("provider gate の出力先が hook と一致しません")
		}
		content := readTestFile(t, filepath.Join(options.Repository, "順序.txt"))
		if got, want := string(content), "C\n"; got != want {
			return fmt.Errorf("provider gate の source snapshot = %q, want %q", got, want)
		}
		return nil
	}
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		_ string,
		_ []string,
		_, _ io.Writer,
	) error {
		calls = append(calls, "quality")
		snapshots = append(snapshots, snapshot)
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push"})

	if code != 0 {
		t.Fatalf("pre-push が失敗しました: %s", stderr)
	}
	wantCalls := []string{"provider:" + commitA, "provider:" + commitB, "quality"}
	if got, want := strings.Join(calls, ","), strings.Join(wantCalls, ","); got != want {
		t.Fatalf("gate 実行順 = %q, want %q", got, want)
	}
	for _, snapshot := range snapshots {
		assertNotExists(t, snapshot)
	}
	if got := string(readTestFile(t, filepath.Join(repository, "順序.txt"))); got != "作業ツリーのみ\n" {
		t.Fatalf("作業ツリーが変更されました: %q", got)
	}
}

func TestPrePushUsesFullHistoryForNewRef(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "順序.txt", "A\n")
	parent := commitAll(t, repository, "A")
	writeFile(t, repository, "順序.txt", "B\n")
	tip := commitAll(t, repository, "B")
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/main", tip, "refs/heads/main", zeroOID(len(tip))),
	)
	var contents []string
	var bases []string
	app.providerOnboarding = func(
		_ context.Context,
		options provideronboarding.Options,
	) error {
		if options.HeadRef != tip {
			return fmt.Errorf("head commit = %q, want %q", options.HeadRef, tip)
		}
		bases = append(bases, options.BaseRef)
		return nil
	}
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		_ string,
		gotRanges []string,
		_, _ io.Writer,
	) error {
		content := readTestFile(t, filepath.Join(snapshot, "順序.txt"))
		contents = append(contents, string(content))
		if got, want := strings.Join(gotRanges, ","), tip; got != want {
			return fmt.Errorf("git range = %q, want %q", got, want)
		}
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push"})

	if code != 0 {
		t.Fatalf("new ref の検査が失敗しました: %s", stderr)
	}
	if got, want := strings.Join(contents, ""), "B\n"; got != want {
		t.Fatalf("検査した履歴 = %q, want %q", got, want)
	}
	if got, want := strings.Join(bases, ","), parent; got != want {
		t.Fatalf("provider gate の base commit = %q, want %q", got, want)
	}
}

func TestPrePushPeelsAnnotatedTagToCommit(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "対象.txt", "parent\n")
	commitAll(t, repository, "parent")
	writeFile(t, repository, "対象.txt", "tagged commit\n")
	commit := commitAll(t, repository, "tag target")
	runGit(t, repository, "tag", "-a", "v1.0.0", "-m", "リリース")
	tagOID := strings.TrimSpace(runGit(t, repository, "rev-parse", "refs/tags/v1.0.0"))
	if tagOID == commit {
		t.Fatal("annotated tag object が作成されませんでした")
	}
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/tags/v1.0.0", tagOID, "refs/tags/v1.0.0", zeroOID(len(tagOID))),
	)
	var ranges []string
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		_ string,
		gotRanges []string,
		_, _ io.Writer,
	) error {
		content := readTestFile(t, filepath.Join(snapshot, "対象.txt"))
		if string(content) != "tagged commit\n" {
			return errors.New("tag の commit snapshot が一致しません")
		}
		ranges = append(ranges, gotRanges...)
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push"})

	if code != 0 {
		t.Fatalf("annotated tag push が失敗しました: %s", stderr)
	}
	if got, want := strings.Join(ranges, ","), commit; got != want {
		t.Fatalf("peeled git range = %q, want %q", got, want)
	}
}

func TestPrePushAcceptsSHA256ObjectIDsAndZeroOID(t *testing.T) {
	repository := t.TempDir()
	runGit(t, repository, "init", "--quiet", "--object-format=sha256")
	runGit(t, repository, "config", "user.name", "テスト利用者")
	runGit(t, repository, "config", "user.email", "test@example.invalid")
	writeFile(t, repository, "対象.txt", "parent\n")
	commitAll(t, repository, "parent")
	writeFile(t, repository, "対象.txt", "sha256\n")
	commit := commitAll(t, repository, "sha256")
	if len(commit) != 64 {
		t.Fatalf("SHA-256 commit ID の長さ = %d, want 64", len(commit))
	}
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/main", commit, "refs/heads/main", zeroOID(64)),
	)
	called := false
	app.qualityGate = func(
		_ context.Context,
		_, _, _ string,
		_ string,
		ranges []string,
		_, _ io.Writer,
	) error {
		called = true
		if got, want := strings.Join(ranges, ","), commit; got != want {
			return fmt.Errorf("SHA-256 range = %q, want %q", got, want)
		}
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push"})

	if code != 0 {
		t.Fatalf("SHA-256 push の検査が失敗しました: %s", stderr)
	}
	if !called {
		t.Fatal("SHA-256 tip に quality gate が実行されませんでした")
	}
}

func TestPrePushRejectsMalformedMissingAndNonCommitOIDs(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "対象.txt", "commit\n")
	commit := commitAll(t, repository, "commit")
	blob := strings.TrimSpace(runGit(t, repository, "hash-object", "対象.txt"))
	missing := strings.Repeat("a", len(commit))

	tests := []struct {
		name  string
		stdin string
	}{
		{
			name:  "field count",
			stdin: "refs/heads/main three fields\n",
		},
		{
			name:  "malformed oid",
			stdin: oidLine("refs/heads/main", "not-an-oid", "refs/heads/main", zeroOID(len(commit))),
		},
		{
			name:  "different oid lengths",
			stdin: oidLine("refs/heads/main", commit, "refs/heads/main", zeroOID(64)),
		},
		{
			name:  "missing local object",
			stdin: oidLine("refs/heads/main", missing, "refs/heads/main", zeroOID(len(commit))),
		},
		{
			name:  "noncommit local object",
			stdin: oidLine("refs/heads/main", blob, "refs/heads/main", zeroOID(len(commit))),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApplication(repository)
			app.stdin = strings.NewReader(test.stdin)
			called := false
			app.qualityGate = func(
				context.Context,
				string,
				string,
				string,
				string,
				[]string,
				io.Writer,
				io.Writer,
			) error {
				called = true
				return nil
			}

			code, _, _ := executeForTest(t, app, []string{"pre-push"})

			if code == 0 {
				t.Fatal("不正な入力が受理されました")
			}
			if called {
				t.Fatal("入力検証失敗後に quality gate が実行されました")
			}
		})
	}
}

func TestPrePushFallsBackToTipHistoryWhenRemoteOIDIsMissing(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "対象.txt", "parent\n")
	parent := commitAll(t, repository, "parent")
	writeFile(t, repository, "対象.txt", "commit\n")
	commit := commitAll(t, repository, "commit")
	missing := strings.Repeat("a", len(commit))
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/main", commit, "refs/heads/main", missing),
	)
	var ranges []string
	var bases []string
	app.providerOnboarding = func(
		_ context.Context,
		options provideronboarding.Options,
	) error {
		bases = append(bases, options.BaseRef)
		return nil
	}
	app.qualityGate = func(
		_ context.Context,
		_, _, _ string,
		_ string,
		gotRanges []string,
		_, _ io.Writer,
	) error {
		ranges = append(ranges, gotRanges...)
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push"})

	if code != 0 {
		t.Fatalf("remote object がローカルにない push を拒否しました: %s", stderr)
	}
	if got, want := strings.Join(ranges, ","), commit; got != want {
		t.Fatalf("fallback range = %q, want %q", got, want)
	}
	if got, want := strings.Join(bases, ","), parent; got != want {
		t.Fatalf("provider gate の fallback base = %q, want %q", got, want)
	}
}

func TestPrePushRejectsRootCommitWithoutProviderComparisonBase(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "対象.txt", "root\n")
	root := commitAll(t, repository, "root")
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/main", root, "refs/heads/main", zeroOID(len(root))),
	)
	providerCalled := false
	qualityCalled := false
	app.providerOnboarding = func(
		context.Context,
		provideronboarding.Options,
	) error {
		providerCalled = true
		return nil
	}
	app.qualityGate = func(
		context.Context,
		string,
		string,
		string,
		string,
		[]string,
		io.Writer,
		io.Writer,
	) error {
		qualityCalled = true
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push"})

	if code == 0 {
		t.Fatal("比較元のない root commit push が成功しました")
	}
	if !strings.Contains(stderr, "第一親") || !strings.Contains(stderr, "比較元") {
		t.Fatalf("root commit を拒否した理由が不明確です: %s", stderr)
	}
	if providerCalled || qualityCalled {
		t.Fatal("比較元の検証失敗後に gate が実行されました")
	}
}

func TestPrePushMaterializesFilesIgnoredByExportAttributes(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "親.txt", "parent\n")
	commitAll(t, repository, "parent")
	writeFile(t, repository, ".gitattributes", "検査対象.txt export-ignore\n")
	writeFile(t, repository, "検査対象.txt", "archive から消える内容\n")
	commit := commitAll(t, repository, "export-ignore")
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/main", commit, "refs/heads/main", zeroOID(len(commit))),
	)
	called := false
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		_ string,
		_ []string,
		_, _ io.Writer,
	) error {
		called = true
		content := readTestFile(t, filepath.Join(snapshot, "検査対象.txt"))
		if got, want := string(content), "archive から消える内容\n"; got != want {
			return fmt.Errorf("commit tree の raw blob と一致しません: %q", got)
		}
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push"})

	if code != 0 {
		t.Fatalf("export-ignore を含む commit snapshot の検査が失敗しました: %s", stderr)
	}
	if !called {
		t.Fatal("完全な commit snapshot に quality gate が実行されませんでした")
	}
}

func TestPrePushIgnoresLocalReplaceObjects(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "対象.txt", "置換先\n")
	replacement := commitAll(t, repository, "replacement")
	writeFile(t, repository, "対象.txt", "送信対象\n")
	tip := commitAll(t, repository, "actual")
	runGit(t, repository, "replace", tip, replacement)
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/main", tip, "refs/heads/main", zeroOID(len(tip))),
	)
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		_ string,
		_ []string,
		_, _ io.Writer,
	) error {
		content := readTestFile(t, filepath.Join(snapshot, "対象.txt"))
		if got, want := string(content), "送信対象\n"; got != want {
			return fmt.Errorf("replace object の内容が混入しました: %q", got)
		}
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push"})

	if code != 0 {
		t.Fatalf("replace object を無効化した push 検査が失敗しました: %s", stderr)
	}
}

func TestPrePushFailsFastAndCleansSnapshot(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "順序.txt", "A\n")
	commitA := commitAll(t, repository, "A")
	writeFile(t, repository, "順序.txt", "B\n")
	commitB := commitAll(t, repository, "B")
	writeFile(t, repository, "順序.txt", "C\n")
	commitC := commitAll(t, repository, "C")
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/first", commitB, "refs/heads/first", commitA) +
			oidLine("refs/heads/second", commitC, "refs/heads/second", commitA),
	)
	var snapshots []string
	app.qualityGate = func(
		_ context.Context,
		_, snapshot, _ string,
		_ string,
		_ []string,
		_, _ io.Writer,
	) error {
		snapshots = append(snapshots, snapshot)
		return errors.New("意図した失敗")
	}

	code, _, _ := executeForTest(t, app, []string{"pre-push"})

	if code == 0 {
		t.Fatal("quality gate 失敗時に pre-push が成功しました")
	}
	if len(snapshots) != 1 {
		t.Fatalf("fail-fast ではありません: %d 回実行", len(snapshots))
	}
	assertNotExists(t, snapshots[0])
}

func TestPrePushStopsBeforeQualityGateWhenProviderGateFails(t *testing.T) {
	repository := newGitRepository(t)
	writeFile(t, repository, "順序.txt", "A\n")
	commitA := commitAll(t, repository, "A")
	writeFile(t, repository, "順序.txt", "B\n")
	commitB := commitAll(t, repository, "B")
	app := newTestApplication(repository)
	app.stdin = strings.NewReader(
		oidLine("refs/heads/main", commitB, "refs/heads/main", commitA),
	)
	var snapshot string
	app.providerOnboarding = func(
		_ context.Context,
		options provideronboarding.Options,
	) error {
		snapshot = options.Repository
		return errors.New("意図した provider gate の失敗")
	}
	qualityCalled := false
	app.qualityGate = func(
		context.Context,
		string,
		string,
		string,
		string,
		[]string,
		io.Writer,
		io.Writer,
	) error {
		qualityCalled = true
		return nil
	}

	code, _, stderr := executeForTest(t, app, []string{"pre-push"})

	if code == 0 {
		t.Fatal("provider gate 失敗時に pre-push が成功しました")
	}
	if !strings.Contains(stderr, "意図した provider gate の失敗") {
		t.Fatalf("provider gate の失敗理由が保持されませんでした: %s", stderr)
	}
	if qualityCalled {
		t.Fatal("provider gate 失敗後に quality gate が実行されました")
	}
	assertNotExists(t, snapshot)
}
