package provideronboarding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const (
	qualitySetupGoAction = "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e"
	qualityCacheAction   = "actions/cache@55cc8345863c7cc4c66a329aec7e433d2d1c52a9"
	qualityGoVersion     = "1.26.7"
)

// SOT-ENG-018/020: 新規 ref の provider 差分は既定 branch との分岐点から検査する。
func TestQualityWorkflowUsesDefaultBranchForNewRefProviderBase(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("テストの作業ディレクトリを取得できませんでした: %v", err)
	}
	workflowPath := filepath.Join(
		workingDirectory,
		"..",
		"..",
		".github",
		"workflows",
		"quality.yml",
	)
	//nolint:gosec // SOT-ENG-018/020: テストの作業ディレクトリから固定した repository 内 workflow だけを読み取る。
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("品質 workflow を読み取れませんでした: %v", err)
	}
	source := string(content)
	for _, required := range []string{
		"PROVIDER_DEFAULT_BRANCH: ${{ github.event.repository.default_branch }}",
		"refs/remotes/origin/${PROVIDER_DEFAULT_BRANCH}",
		"git check-ref-format --branch",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("新規 ref の比較元検証に %q がありません", required)
		}
	}
	if strings.Contains(source, "HEAD^1") {
		t.Fatal("新規 ref の provider 差分が第一親だけに限定されています")
	}
}

// SOT-ENG-020/027: 権威ゲートは既知の標準ライブラリ脆弱性を修正した固定 patch 版を使う。
func TestQualityWorkflowUsesPatchedGoToolchain(t *testing.T) {
	t.Parallel()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("テストの作業ディレクトリを取得できませんでした: %v", err)
	}
	workflowPath := filepath.Join(
		workingDirectory,
		"..",
		"..",
		".github",
		"workflows",
		"quality.yml",
	)
	//nolint:gosec // SOT-ENG-020/027: repository 内の固定 workflow だけを読み取る。
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("品質 workflow を読み取れませんでした: %v", err)
	}
	var workflow qualityWorkflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("品質 workflow を解析できませんでした: %v", err)
	}
	verify, exists := workflow.Jobs["verify"]
	if !exists {
		t.Fatal("品質 workflow に verify job がありません")
	}
	setup := requireQualityActionStep(t, verify.Steps, qualitySetupGoAction)
	if setup.With["go-version"] != qualityGoVersion {
		t.Fatalf("verify job の Go version = %#v", setup.With["go-version"])
	}
	cache := requireQualityActionStep(t, verify.Steps, qualityCacheAction)
	wantCacheKey := "${{ runner.os }}-${{ runner.arch }}-quality-go1.26.7-${{ hashFiles('go.sum', 'tools/go.sum', 'tools/gitleaks/go.sum', '.golangci.yml') }}"
	if cache.With["key"] != wantCacheKey {
		t.Fatalf("verify job の品質 cache key = %#v", cache.With["key"])
	}
	wantRestoreKey := "${{ runner.os }}-${{ runner.arch }}-quality-go1.26.7-"
	if strings.TrimSpace(stringValue(cache.With["restore-keys"])) != wantRestoreKey {
		t.Fatalf("verify job の品質 cache restore key = %#v", cache.With["restore-keys"])
	}

	source := string(content)
	for _, obsolete := range []string{
		`go-version: "1.26.5"`,
		"quality-go1.26.5-",
	} {
		if strings.Contains(source, obsolete) {
			t.Fatalf("脆弱な Go patch 版の設定 %q が残っています", obsolete)
		}
	}
}

func requireQualityActionStep(
	t *testing.T,
	steps []qualityWorkflowStep,
	action string,
) qualityWorkflowStep {
	t.Helper()
	for _, step := range steps {
		if step.Uses == action {
			return step
		}
	}
	t.Fatalf("verify job に action %q がありません", action)
	return qualityWorkflowStep{}
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

type qualityWorkflow struct {
	Jobs map[string]qualityWorkflowJob `yaml:"jobs"`
}

type qualityWorkflowJob struct {
	Steps []qualityWorkflowStep `yaml:"steps"`
}

type qualityWorkflowStep struct {
	Uses string         `yaml:"uses"`
	With map[string]any `yaml:"with"`
}
