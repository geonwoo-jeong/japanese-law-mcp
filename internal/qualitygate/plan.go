package qualitygate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
)

const readonlyGoFlags = "-mod=readonly"

const gitleaksLogOptions = "--full-history -m --diff-filter=ACMRT --no-textconv --no-ext-diff"

func buildPlan(input planInput) ([]step, error) {
	if _, err := ParseProfile(string(input.profile)); err != nil {
		return nil, err
	}
	if err := validateGitRanges(input.profile, input.gitRanges); err != nil {
		return nil, err
	}

	switch input.profile {
	case ProfilePreCommit:
		return buildPreCommitPlan(input)
	case ProfilePrePush:
		return buildPrePushPlan(input), nil
	case ProfileCI:
		return buildCIPlan(input), nil
	default:
		return nil, fmt.Errorf("品質ゲートのプロファイルが不正です: %q", input.profile)
	}
}

func buildPreCommitPlan(input planInput) ([]step, error) {
	paths, err := normalizeChangedPaths(input.changedPaths)
	if err != nil {
		return nil, err
	}
	goFiles, err := normalizeChangedPaths(input.goFormatPaths)
	if err != nil {
		return nil, err
	}

	steps := []step{
		checksumStep(input.snapshot),
		snapshotCachePolicyStep(input.snapshot),
	}
	if len(goFiles) > 0 {
		steps = append(steps, stagedFormatStep(input.snapshot, goFiles))
	}
	steps = append(steps, preCommitGitSteps(input.repository, input.snapshot)...)
	if needsSOTContract(paths) {
		steps = append(steps, sotContractStep(input.snapshot))
	}
	if needsActionLint(paths) {
		if actionLint, ok := actionLintStep(input.snapshot, false); ok {
			steps = append(steps, actionLint)
		}
	}
	if needsLintConfigVerification(paths) {
		steps = append(steps, lintConfigStep(input.snapshot, false))
	}
	return steps, nil
}

func buildPrePushPlan(input planInput) []step {
	steps := buildSnapshotPlan(
		input.snapshot,
		[]step{snapshotCachePolicyStep(input.snapshot)},
		false,
	)
	for index, gitRange := range input.gitRanges {
		steps = append(steps, gitSecretsStep(
			fmt.Sprintf("range-secrets-%d", index+1),
			fmt.Sprintf("送信対象 Git 範囲 %d の秘密情報", index+1),
			input.repository,
			input.snapshot,
			true,
			false,
			"--log-opts="+gitleaksLogOptions+" "+gitRange,
		))
	}
	return steps
}

func buildCIPlan(input planInput) []step {
	steps := buildSnapshotPlan(input.snapshot, ciCachePolicySteps(input.repository), true)
	steps = append(steps,
		productVulnerabilityStep(input.snapshot),
		commonToolVulnerabilityStep(input.snapshot),
		gitleaksVulnerabilityStep(input.snapshot),
		historyCompletenessStep(input.repository),
		gitSecretsStep(
			"history-secrets",
			"Git 全履歴の秘密情報",
			input.repository,
			input.snapshot,
			false,
			true,
			"--log-opts="+gitleaksLogOptions+" --all HEAD",
		),
	)
	return steps
}

func buildSnapshotPlan(snapshot string, cachePolicy []step, network bool) []step {
	steps := []step{checksumStep(snapshot)}
	steps = append(steps, cachePolicy...)
	steps = append(steps,
		lintConfigStep(snapshot, network),
		fullFormatStep(snapshot, network),
		vetStep(snapshot, network),
		sotAnalysisStep(snapshot, network),
		lintStep(snapshot, network),
		testStep(snapshot, network),
		coverageStep(snapshot),
		moduleTidyStep("root-tidy", "製品モジュールの整合性", snapshot, network),
		moduleVerifyStep("root-verify", "製品モジュールの依存物", snapshot, network),
		moduleTidyStep("tools-tidy", "共通検証ツールモジュールの整合性", filepath.Join(snapshot, "tools"), network),
		moduleVerifyStep("tools-verify", "共通検証ツールモジュールの依存物", filepath.Join(snapshot, "tools"), network),
		moduleTidyStep("gitleaks-tools-tidy", "秘密情報検査ツールモジュールの整合性", filepath.Join(snapshot, "tools", "gitleaks"), network),
		moduleVerifyStep("gitleaks-tools-verify", "秘密情報検査ツールモジュールの依存物", filepath.Join(snapshot, "tools", "gitleaks"), network),
	)
	if actionLint, ok := actionLintStep(snapshot, network); ok {
		steps = append(steps, actionLint)
	}
	steps = append(steps, snapshotSecretsStep(snapshot, network))
	return steps
}

func ciCachePolicySteps(repository string) []step {
	return []step{
		commandStep(
			"snapshot-cache-index",
			"Git index のキャッシュ追跡禁止",
			"SOT-ENG-019",
			commandSpec{
				path: "git",
				args: []string{
					"ls-files",
					"-z",
					"--cached",
					"--",
					".cache",
					".tmp",
					"coverage.out",
				},
				dir:              repository,
				isolateGitConfig: true,
				validateOutput:   requireNoTrackedCache,
			},
		),
		commandStep(
			"snapshot-cache-history",
			"Git 履歴のキャッシュ追跡禁止",
			"SOT-ENG-019",
			commandSpec{
				path: "git",
				args: []string{
					"log",
					"--all",
					"HEAD",
					"-m",
					"--format=",
					"--name-only",
					"-z",
					"--",
					".cache",
					".tmp",
					"coverage.out",
				},
				dir:              repository,
				isolateGitConfig: true,
				validateOutput:   requireNoTrackedCache,
			},
		),
	}
}

func stagedFormatStep(snapshot string, goFiles []string) step {
	args := append([]string{"-l", "--"}, goFiles...)
	return commandStep(
		"format",
		"ステージ済み Go ファイルの整形",
		"SOT-ENG-019",
		commandSpec{
			path:           "gofmt",
			args:           args,
			dir:            snapshot,
			validateOutput: requireEmptyOutput,
		},
	)
}

func preCommitGitSteps(repository, snapshot string) []step {
	return []step{
		commandStep(
			"cached-diff",
			"ステージ済み差分の空白エラー",
			"SOT-ENG-020",
			commandSpec{
				path:               "git",
				args:               []string{"diff", "--cached", "--check"},
				dir:                repository,
				preserveGitIndex:   true,
				preserveGitObjects: true,
				isolateGitConfig:   true,
			},
		),
		commandStep(
			"staged-secrets",
			"ステージ済み内容の秘密情報",
			"SOT-ENG-020",
			goToolCommand(snapshot, false, directoryGitleaksArgs()),
		),
	}
}

func directoryGitleaksArgs() []string {
	return []string{
		"tool",
		"-modfile=tools/gitleaks/go.mod",
		"gitleaks",
		"dir",
		"--config=.gitleaks.toml",
		"--gitleaks-ignore-path=" + os.DevNull,
		"--ignore-gitleaks-allow",
		"--redact",
		"--no-banner",
		".",
	}
}

func sotContractStep(snapshot string) step {
	return commandStep(
		"sot-contract",
		"SOT 構造とリンク",
		"SOT-ENG-020",
		goCommand(snapshot, false, "test", "-count=1", "./internal/sotcheck"),
	)
}

func lintConfigStep(snapshot string, network bool) step {
	return commandStep(
		"lint-config",
		"リンター設定",
		"SOT-ENG-019",
		goToolCommand(
			snapshot,
			network,
			[]string{"tool", "-modfile=tools/go.mod", "golangci-lint", "config", "verify"},
		),
	)
}

func actionLintStep(snapshot string, network bool) (step, bool) {
	workflowFiles := make([]string, 0)
	for _, extension := range []string{".yml", ".yaml"} {
		matches, _ := filepath.Glob(filepath.Join(snapshot, ".github", "workflows", "*"+extension))
		for _, match := range matches {
			relative, err := filepath.Rel(snapshot, match)
			if err != nil {
				continue
			}
			workflowFiles = append(workflowFiles, filepath.ToSlash(relative))
		}
	}
	slices.Sort(workflowFiles)
	if len(workflowFiles) == 0 {
		return step{}, false
	}
	args := []string{
		"tool",
		"-modfile=tools/go.mod",
		"actionlint",
		// SOT-ENG-019: 固定していない PATH 上の外部解析器を呼ばない。
		// shell wrapper は統合テストで構文と入出力を検証し、Python workflow は現在存在しない。
		"-shellcheck=",
		"-pyflakes=",
	}
	args = append(args, workflowFiles...)
	return commandStep(
		"actions-lint",
		"GitHub Actions の静的解析",
		"SOT-ENG-019",
		goToolCommand(snapshot, network, args),
	), true
}

func fullFormatStep(snapshot string, network bool) step {
	return commandStep(
		"format",
		"Go ファイルの整形",
		"SOT-ENG-019",
		goToolCommand(
			snapshot,
			network,
			[]string{"tool", "-modfile=tools/go.mod", "golangci-lint", "fmt", "--diff"},
		),
	)
}

func vetStep(snapshot string, network bool) step {
	return commandStep(
		"vet",
		"Go 標準静的解析",
		"SOT-ENG-020",
		goCommand(snapshot, network, "vet", "./..."),
	)
}

func sotAnalysisStep(snapshot string, network bool) step {
	return commandStep(
		"sot-analysis",
		"SOT 固有の静的解析",
		"SOT-ENG-019",
		goCommand(snapshot, network, "run", "./cmd/sotvet", "--", "./..."),
	)
}

func lintStep(snapshot string, network bool) step {
	return commandStep(
		"lint",
		"汎用リンター",
		"SOT-ENG-019",
		goToolCommand(
			snapshot,
			network,
			[]string{"tool", "-modfile=tools/go.mod", "golangci-lint", "run", "./..."},
		),
	)
}

func testStep(snapshot string, network bool) step {
	return commandStep(
		"test",
		"テストとカバレッジ計測",
		"SOT-ENG-020",
		goCommand(
			snapshot,
			network,
			"test",
			"-count=1",
			"-covermode=atomic",
			"-coverpkg=./...",
			"-coverprofile=coverage.out",
			"./...",
		),
	)
}

func coverageStep(snapshot string) step {
	return internalStep(
		"coverage",
		"カバレッジ下限",
		"SOT-ENG-020",
		func() error {
			return verifyTotalCoverage(filepath.Join(snapshot, "coverage.out"), 80)
		},
	)
}

func snapshotSecretsStep(snapshot string, network bool) step {
	return commandStep(
		"snapshot-secrets",
		"検査スナップショットの秘密情報",
		"SOT-ENG-020",
		goToolCommand(snapshot, network, directoryGitleaksArgs()),
	)
}

func productVulnerabilityStep(snapshot string) step {
	return commandStep(
		"product-vulnerabilities",
		"製品コードの既知の脆弱性",
		"SOT-ENG-020",
		goToolCommand(
			snapshot,
			true,
			[]string{"tool", "-modfile=tools/go.mod", "govulncheck", "-test", "./..."},
		),
	)
}

func commonToolVulnerabilityStep(snapshot string) step {
	return commandStep(
		"tool-vulnerabilities",
		"共通検証ツールの既知の脆弱性",
		"SOT-ENG-020",
		goToolVulnerabilityCommand(
			snapshot,
			"tools/go.mod",
			"github.com/golangci/golangci-lint/v2/cmd/golangci-lint",
			"github.com/rhysd/actionlint/cmd/actionlint",
			"golang.org/x/vuln/cmd/govulncheck",
		),
	)
}

func gitleaksVulnerabilityStep(snapshot string) step {
	return commandStep(
		"gitleaks-vulnerabilities",
		"秘密情報検査ツールの既知の脆弱性",
		"SOT-ENG-020",
		goToolVulnerabilityCommand(
			snapshot,
			"tools/gitleaks/go.mod",
			"github.com/zricethezav/gitleaks/v8",
		),
	)
}

func historyCompletenessStep(repository string) step {
	return commandStep(
		"history-completeness",
		"Git 全履歴の取得状態",
		"SOT-ENG-021",
		commandSpec{
			path:             "git",
			args:             []string{"rev-parse", "--is-shallow-repository"},
			dir:              repository,
			isolateGitConfig: true,
			validateOutput:   requireNonShallowRepository,
		},
	)
}

func checksumStep(snapshot string) step {
	return internalStep(
		"checksum",
		"開発原則のチェックサム",
		"SOT-ENG-020",
		func() error { return verifyDevelopmentPrinciples(snapshot) },
	)
}

func snapshotCachePolicyStep(snapshot string) step {
	return internalStep(
		"snapshot-cache-policy",
		"snapshot のキャッシュ追跡禁止",
		"SOT-ENG-019",
		func() error { return verifySnapshotCachePolicy(snapshot) },
	)
}

func moduleTidyStep(key, name, directory string, network bool) step {
	return commandStep(
		key,
		name,
		"SOT-ENG-020",
		goCommand(directory, network, "mod", "tidy", "-diff"),
	)
}

func moduleVerifyStep(key, name, directory string, network bool) step {
	return commandStep(
		key,
		name,
		"SOT-ENG-020",
		goCommand(directory, network, "mod", "verify"),
	)
}

func gitSecretsStep(
	key, name, repository, snapshot string,
	preserveObjects bool,
	network bool,
	extra ...string,
) step {
	args := []string{
		"tool",
		"-modfile=tools/gitleaks/go.mod",
		"gitleaks",
		"git",
		"--config=.gitleaks.toml",
		"--gitleaks-ignore-path=" + os.DevNull,
		"--ignore-gitleaks-allow",
		"--redact",
		"--no-banner",
	}
	args = append(args, extra...)
	args = append(args, repository)
	command := withIsolatedGitConfig(goToolCommand(snapshot, network, args))
	command.preserveGitObjects = preserveObjects
	return commandStep(
		key,
		name,
		"SOT-ENG-020",
		command,
	)
}

func goCommand(directory string, network bool, args ...string) commandSpec {
	return commandSpec{
		path:      "go",
		args:      slices.Clone(args),
		dir:       directory,
		goCommand: true,
		network:   network,
		goFlags:   readonlyGoFlags,
	}
}

func goToolCommand(directory string, network bool, args []string) commandSpec {
	return goCommand(directory, network, args...)
}

func goToolVulnerabilityCommand(
	directory string,
	moduleFile string,
	packages ...string,
) commandSpec {
	args := []string{"tool", "govulncheck"}
	args = append(args, packages...)
	command := goCommand(directory, true, args...)
	command.goFlags = readonlyGoFlags + " -modfile=" + moduleFile
	return command
}

func withIsolatedGitConfig(command commandSpec) commandSpec {
	command.isolateGitConfig = true
	return command
}

func validateGitRanges(profile Profile, gitRanges []string) error {
	if len(gitRanges) > 0 && profile != ProfilePrePush {
		return errors.New("--git-range は pre-push プロファイルでだけ指定できます")
	}
	if profile == ProfilePrePush && len(gitRanges) == 0 {
		return errors.New("pre-push プロファイルには --git-range が一つ以上必要です")
	}
	for _, gitRange := range gitRanges {
		if gitRange == "" ||
			strings.HasPrefix(gitRange, "-") ||
			strings.IndexFunc(gitRange, unicode.IsSpace) >= 0 ||
			strings.IndexFunc(gitRange, unicode.IsControl) >= 0 {
			return fmt.Errorf("指定した Git 範囲が不正です: %q", gitRange)
		}
	}
	return nil
}

func normalizeChangedPaths(paths []string) ([]string, error) {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
		if path == "" ||
			cleaned == "." ||
			cleaned == ".." ||
			strings.HasPrefix(cleaned, "../") ||
			filepath.IsAbs(path) {
			return nil, fmt.Errorf("変更パスが不正です: %q", path)
		}
		normalized = append(normalized, cleaned)
	}
	slices.Sort(normalized)
	return slices.Compact(normalized), nil
}

func needsSOTContract(paths []string) bool {
	for _, path := range paths {
		if path == "README.md" ||
			path == "AGENTS.md" ||
			strings.HasPrefix(path, "docs/") ||
			strings.HasPrefix(path, "sot/") ||
			strings.HasPrefix(path, "wiki/") ||
			strings.HasPrefix(path, "internal/sotcheck/") {
			return true
		}
	}
	return false
}

func needsActionLint(paths []string) bool {
	for _, path := range paths {
		if strings.HasPrefix(path, ".github/workflows/") ||
			strings.HasPrefix(path, ".github/actions/") {
			return true
		}
	}
	return false
}

func needsLintConfigVerification(paths []string) bool {
	for _, path := range paths {
		if path == ".golangci.yml" ||
			path == "tools/go.mod" ||
			path == "tools/go.sum" {
			return true
		}
	}
	return false
}

func requireEmptyOutput(output []byte) error {
	if len(strings.TrimSpace(string(output))) > 0 {
		return errors.New("gofmt が未整形の Go ファイルを検出しました")
	}
	return nil
}

func requireNonShallowRepository(output []byte) error {
	if strings.TrimSpace(string(output)) != "false" {
		return errors.New("CI の Git リポジトリが shallow であり、全履歴を検査できません")
	}
	return nil
}

func requireNoTrackedCache(output []byte) error {
	if len(output) > 0 {
		return errors.New("SOT-ENG-019: .cache、.tmp または coverage.out に Git 追跡対象があります")
	}
	return nil
}
