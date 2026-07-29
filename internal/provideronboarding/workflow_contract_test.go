package provideronboarding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
