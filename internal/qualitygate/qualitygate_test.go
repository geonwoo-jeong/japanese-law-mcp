package qualitygate

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestBuildPlanPreCommitIsFastOfflineAndPathAware(t *testing.T) {
	t.Parallel()

	snapshot := writeValidPrinciples(t)
	writeWorkflowFile(t, snapshot, "quality.yml")

	steps, err := buildPlan(planInput{
		profile:    ProfilePreCommit,
		repository: "/repo",
		snapshot:   snapshot,
		changedPaths: []string{
			"internal/example/example.go",
			"sot/50-engineering/20-verification-gate.md",
			".github/workflows/quality.yml",
			".golangci.yml",
			"tools/gitleaks/go.mod",
		},
		goFormatPaths: []string{"internal/example/example.go"},
	})
	if err != nil {
		t.Fatalf("計画の作成に失敗しました: %v", err)
	}

	got := stepSignatures(steps)
	want := []string{
		"checksum|SOT-ENG-020|internal",
		"snapshot-cache-policy|SOT-ENG-019|internal",
		"format|SOT-ENG-019|" + snapshot + "|gofmt|-l|--|internal/example/example.go",
		"cached-diff|SOT-ENG-020|/repo|git|diff|--cached|--check",
		"staged-secrets|SOT-ENG-020|" + snapshot + "|go|tool|-modfile=tools/gitleaks/go.mod|gitleaks|dir|--config=.gitleaks.toml|--gitleaks-ignore-path=" + os.DevNull + "|--ignore-gitleaks-allow|--redact|--no-banner|.",
		"sot-contract|SOT-ENG-020|" + snapshot + "|go|test|-count=1|./internal/sotcheck",
		"actions-lint|SOT-ENG-019|" + snapshot + "|go|tool|-modfile=tools/go.mod|actionlint|-shellcheck=|-pyflakes=|.github/workflows/quality.yml",
		"lint-config|SOT-ENG-019|" + snapshot + "|go|tool|-modfile=tools/go.mod|golangci-lint|config|verify",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pre-commit 計画が一致しません:\n got: %q\nwant: %q", got, want)
	}

	for _, step := range steps {
		if step.command != nil && step.command.path == "go" && step.command.network {
			t.Fatalf("pre-commit の Go コマンドがネットワークを許可しています: %s", step.key)
		}
	}
}

func TestBuildPlanDeletedPathsTriggerChecksButNotGofmt(t *testing.T) {
	t.Parallel()

	snapshot := writeValidPrinciples(t)
	writeWorkflowFile(t, snapshot, "remaining.yml")

	steps, err := buildPlan(planInput{
		profile:       ProfilePreCommit,
		repository:    "/repo",
		snapshot:      snapshot,
		changedPaths:  []string{".github/workflows/deleted.yml", "internal/deleted.go"},
		goFormatPaths: []string{},
	})
	if err != nil {
		t.Fatalf("計画の作成に失敗しました: %v", err)
	}
	signatures := stepSignatures(steps)
	for _, signature := range signatures {
		if strings.HasPrefix(signature, "format|") {
			t.Fatalf("削除済み Go ファイルが gofmt 対象です: %q", signatures)
		}
	}
	if !slices.ContainsFunc(signatures, func(signature string) bool {
		return strings.HasPrefix(signature, "actions-lint|")
	}) {
		t.Fatalf("削除済み workflow が actionlint を起動していません: %q", signatures)
	}
}

func TestBuildPlanSkipsActionlintWhenNoWorkflowRemains(t *testing.T) {
	t.Parallel()

	snapshot := writeValidPrinciples(t)
	steps, err := buildPlan(planInput{
		profile:      ProfilePreCommit,
		repository:   "/repo",
		snapshot:     snapshot,
		changedPaths: []string{".github/workflows/deleted.yml"},
	})
	if err != nil {
		t.Fatalf("計画の作成に失敗しました: %v", err)
	}
	if slices.ContainsFunc(stepSignatures(steps), func(signature string) bool {
		return strings.HasPrefix(signature, "actions-lint|")
	}) {
		t.Fatal("workflow が残っていない snapshot で actionlint が起動しました")
	}
}

func TestBuildPlanPreCommitSkipsUnrelatedConditionalChecks(t *testing.T) {
	t.Parallel()

	snapshot := writeValidPrinciples(t)

	steps, err := buildPlan(planInput{
		profile:      ProfilePreCommit,
		repository:   "/repo",
		snapshot:     snapshot,
		changedPaths: []string{".gitignore"},
	})
	if err != nil {
		t.Fatalf("計画の作成に失敗しました: %v", err)
	}

	got := stepSignatures(steps)
	want := []string{
		"checksum|SOT-ENG-020|internal",
		"snapshot-cache-policy|SOT-ENG-019|internal",
		"cached-diff|SOT-ENG-020|/repo|git|diff|--cached|--check",
		"staged-secrets|SOT-ENG-020|" + snapshot + "|go|tool|-modfile=tools/gitleaks/go.mod|gitleaks|dir|--config=.gitleaks.toml|--gitleaks-ignore-path=" + os.DevNull + "|--ignore-gitleaks-allow|--redact|--no-banner|.",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("不要な条件付き検査が含まれています:\n got: %q\nwant: %q", got, want)
	}
}

func TestBuildPlanPreCommitChecksReadmeLinks(t *testing.T) {
	t.Parallel()

	snapshot := writeValidPrinciples(t)

	steps, err := buildPlan(planInput{
		profile:      ProfilePreCommit,
		repository:   "/repo",
		snapshot:     snapshot,
		changedPaths: []string{"README.md"},
	})
	if err != nil {
		t.Fatalf("計画の作成に失敗しました: %v", err)
	}

	signatures := stepSignatures(steps)
	if !slices.ContainsFunc(signatures, func(signature string) bool {
		return strings.HasPrefix(signature, "sot-contract|")
	}) {
		t.Fatalf("README.md の変更で SOT とリンクの検査が起動しません: %q", signatures)
	}
}

func TestBuildPlanPrePushIsMinimalAndDeterministic(t *testing.T) {
	t.Parallel()

	snapshot := writeValidPrinciples(t)
	writeWorkflowFile(t, snapshot, "quality.yml")

	steps, err := buildPlan(planInput{
		profile:    ProfilePrePush,
		repository: "/repo",
		snapshot:   snapshot,
		gitRanges:  []string{"origin/main..HEAD", "abc123..def456"},
	})
	if err != nil {
		t.Fatalf("計画の作成に失敗しました: %v", err)
	}

	got := stepSignatures(steps)
	want := []string{
		"checksum|SOT-ENG-020|internal",
		"snapshot-cache-policy|SOT-ENG-019|internal",
		"range-secrets-1|SOT-ENG-020|" + snapshot + "|go|tool|-modfile=tools/gitleaks/go.mod|gitleaks|git|--config=.gitleaks.toml|--gitleaks-ignore-path=" + os.DevNull + "|--ignore-gitleaks-allow|--redact|--no-banner|--log-opts=--full-history -m --diff-filter=ACMRT --no-textconv --no-ext-diff origin/main..HEAD|/repo",
		"range-secrets-2|SOT-ENG-020|" + snapshot + "|go|tool|-modfile=tools/gitleaks/go.mod|gitleaks|git|--config=.gitleaks.toml|--gitleaks-ignore-path=" + os.DevNull + "|--ignore-gitleaks-allow|--redact|--no-banner|--log-opts=--full-history -m --diff-filter=ACMRT --no-textconv --no-ext-diff abc123..def456|/repo",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pre-push 計画が一致しません:\n got: %q\nwant: %q", got, want)
	}

	for _, forbidden := range []string{"|go|test|", "|go|vet|", "golangci-lint"} {
		if slices.ContainsFunc(got, func(signature string) bool {
			return strings.Contains(signature, forbidden)
		}) {
			t.Fatalf("pre-push に高負荷検査が含まれています: %s", forbidden)
		}
	}
	for _, current := range steps[len(steps)-2:] {
		if current.command == nil || !current.command.preserveGitObjects {
			t.Fatalf("pre-push range が Git object 環境を保存していません: %s", current.key)
		}
	}
}

func TestBuildPlanAssignsGoNetworkPolicyByProfile(t *testing.T) {
	t.Parallel()

	snapshot := writeValidPrinciples(t)
	writeWorkflowFile(t, snapshot, "quality.yml")
	prePush, err := buildPlan(planInput{
		profile:    ProfilePrePush,
		repository: "/repo",
		snapshot:   snapshot,
		gitRanges:  []string{"main..HEAD"},
	})
	if err != nil {
		t.Fatalf("pre-push 計画の作成に失敗しました: %v", err)
	}
	for _, current := range prePush {
		if current.command != nil && current.command.path == "go" && current.command.network {
			t.Fatalf("pre-push の Go コマンドがネットワークを許可しています: %s", current.key)
		}
	}

	ci, err := buildPlan(planInput{
		profile:    ProfileCI,
		repository: "/repo",
		snapshot:   snapshot,
	})
	if err != nil {
		t.Fatalf("CI 計画の作成に失敗しました: %v", err)
	}
	for _, current := range ci {
		if current.key == "legal-query-eval" {
			if current.command == nil ||
				current.command.path != "go" ||
				current.command.network {
				t.Fatalf(
					"統合照会の固定評価が offline ではありません: %#v",
					current.command,
				)
			}
			continue
		}
		if current.command != nil && current.command.path == "go" && !current.command.network {
			t.Fatalf("CI の Go コマンドがネットワークを許可していません: %s", current.key)
		}
	}
}

func TestBuildPlanCIAddsAllVulnerabilityAndHistoryChecks(t *testing.T) {
	t.Parallel()

	snapshot := writeValidPrinciples(t)
	writeWorkflowFile(t, snapshot, "quality.yml")

	steps, err := buildPlan(planInput{
		profile:    ProfileCI,
		repository: "/repo",
		snapshot:   snapshot,
	})
	if err != nil {
		t.Fatalf("計画の作成に失敗しました: %v", err)
	}

	got := stepSignatures(steps)
	wantPrefix := []string{
		"checksum|SOT-ENG-020|internal",
		"snapshot-cache-index|SOT-ENG-019|/repo|git|ls-files|-z|--cached|--|.cache|.tmp|coverage.out",
		"snapshot-cache-history|SOT-ENG-019|/repo|git|log|--all|HEAD|-m|--format=|--name-only|-z|--|.cache|.tmp|coverage.out",
	}
	if len(got) < len(wantPrefix) || !reflect.DeepEqual(got[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("CI の cache policy が一致しません:\n got: %q\nwant: %q", got, wantPrefix)
	}
	if !slices.Contains(
		got,
		"test|SOT-ENG-020|"+snapshot+"|go|test|-p=1|-count=1|-covermode=set|-coverprofile=coverage.out|./...",
	) {
		t.Fatalf("CI の省資源テスト設定がありません: %q", got)
	}
	wantSuffix := []string{
		"legal-query-eval|SOT-ENG-024|" + snapshot + "|go|run|./cmd/legal-query-eval|--adoption=./testdata/legalquery/adoptions/current.json|--format=json",
		"product-vulnerabilities|SOT-ENG-020|" + snapshot + "|go|tool|-modfile=tools/go.mod|govulncheck|-test|./...",
		"tool-vulnerabilities|SOT-ENG-020|" + snapshot + "|go|tool|govulncheck|github.com/golangci/golangci-lint/v2/cmd/golangci-lint|github.com/rhysd/actionlint/cmd/actionlint|golang.org/x/vuln/cmd/govulncheck",
		"gitleaks-vulnerabilities|SOT-ENG-020|" + snapshot + "|go|tool|govulncheck|github.com/zricethezav/gitleaks/v8",
		"history-completeness|SOT-ENG-027|/repo|git|rev-parse|--is-shallow-repository",
		"history-secrets|SOT-ENG-020|" + snapshot + "|go|tool|-modfile=tools/gitleaks/go.mod|gitleaks|git|--config=.gitleaks.toml|--gitleaks-ignore-path=" + os.DevNull + "|--ignore-gitleaks-allow|--redact|--no-banner|--log-opts=--full-history -m --diff-filter=ACMRT --no-textconv --no-ext-diff --all HEAD|/repo",
	}
	if len(got) < len(wantSuffix) {
		t.Fatalf("CI 計画の検査数が不足しています: %q", got)
	}
	if tail := got[len(got)-len(wantSuffix):]; !reflect.DeepEqual(tail, wantSuffix) {
		t.Fatalf("CI 固有検査が一致しません:\n got: %q\nwant: %q", tail, wantSuffix)
	}
	history := steps[len(steps)-1]
	if history.command == nil || history.command.preserveGitObjects {
		t.Fatal("CI 全履歴検査が ambient Git object 環境を保存しています")
	}
}

func TestBuildPlanRejectsUnsafeOrMisplacedGitRanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile Profile
		ranges  []string
	}{
		{name: "pre-commit では受け付けない", profile: ProfilePreCommit, ranges: []string{"main..HEAD"}},
		{name: "CI では受け付けない", profile: ProfileCI, ranges: []string{"main..HEAD"}},
		{name: "空文字", profile: ProfilePrePush, ranges: []string{""}},
		{name: "先頭ハイフン", profile: ProfilePrePush, ranges: []string{"--all"}},
		{name: "空白", profile: ProfilePrePush, ranges: []string{"main..HEAD --all"}},
		{name: "制御文字", profile: ProfilePrePush, ranges: []string{"main..\nHEAD"}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := buildPlan(planInput{
				profile:    tt.profile,
				repository: "/repo",
				snapshot:   "/snapshot",
				gitRanges:  tt.ranges,
			})
			if err == nil {
				t.Fatal("不正な Git 範囲が受理されました")
			}
		})
	}
}

func writeValidPrinciples(t *testing.T) string {
	t.Helper()

	repository := t.TempDir()
	documentPath := filepath.Join(repository, filepath.FromSlash(developmentPrinciplesPath))
	if err := os.MkdirAll(filepath.Dir(documentPath), 0o750); err != nil {
		t.Fatalf("docs の作成に失敗しました: %v", err)
	}
	content := []byte("固定した開発原則\n")
	if err := os.WriteFile(documentPath, content, 0o600); err != nil {
		t.Fatalf("開発原則の作成に失敗しました: %v", err)
	}
	sum := sha256.Sum256(content)
	manifest := hex.EncodeToString(sum[:]) + "  " + developmentPrinciplesPath + "\n"
	if err := os.WriteFile(
		filepath.Join(repository, "docs", "development-principles.sha256"),
		[]byte(manifest),
		0o600,
	); err != nil {
		t.Fatalf("チェックサムの作成に失敗しました: %v", err)
	}
	for _, relativePath := range []string{"tools/go.mod", "tools/gitleaks/go.mod"} {
		path := filepath.Join(repository, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("ツールモジュールのディレクトリを作成できません: %v", err)
		}
		if err := os.WriteFile(
			path,
			[]byte("module example.invalid/tools\n\ngo 1.25.0\n"),
			0o600,
		); err != nil {
			t.Fatalf("ツールモジュールを作成できません: %v", err)
		}
	}
	return repository
}

func writeWorkflowFile(t *testing.T, repository string, name string) {
	t.Helper()

	path := filepath.Join(repository, ".github", "workflows", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("workflow directory の作成に失敗しました: %v", err)
	}
	if err := os.WriteFile(path, []byte("name: quality\non: push\njobs: {}\n"), 0o600); err != nil {
		t.Fatalf("workflow file の作成に失敗しました: %v", err)
	}
}

func assertCommandBoundary(t *testing.T, spec commandSpec, directory, path string) {
	t.Helper()
	if spec.dir != directory || spec.path != path {
		t.Fatalf(
			"コマンド境界 = dir %q, path %q; want dir %q, path %q",
			spec.dir,
			spec.path,
			directory,
			path,
		)
	}
}

func resolvedPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("パスを解決できません: %v", err)
	}
	return resolved
}

func stepSignatures(steps []step) []string {
	signatures := make([]string, 0, len(steps))
	for _, step := range steps {
		if step.command == nil {
			signatures = append(signatures, step.key+"|"+step.sotID+"|internal")
			continue
		}
		fields := []string{step.key, step.sotID, step.command.dir, step.command.path}
		fields = append(fields, step.command.args...)
		signatures = append(signatures, strings.Join(fields, "|"))
	}
	return signatures
}

func environmentMap(environment []string) map[string]string {
	result := make(map[string]string, len(environment))
	for _, entry := range environment {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			result[key] = value
		}
	}
	return result
}

type executionResult struct {
	stdout []byte
	stderr []byte
	err    error
}

type recordingExecutor struct {
	paths   []string
	specs   []commandSpec
	results []executionResult
}

func (e *recordingExecutor) run(spec commandSpec) ([]byte, []byte, error) {
	e.paths = append(e.paths, spec.path)
	e.specs = append(e.specs, spec)
	index := len(e.paths) - 1
	if index >= len(e.results) {
		return nil, nil, nil
	}
	result := e.results[index]
	return result.stdout, result.stderr, result.err
}
