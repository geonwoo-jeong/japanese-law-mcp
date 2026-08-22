package provideronboarding

import (
	"encoding/json"
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
	exactGoRootEnv       = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_GOROOT"
	exactGoModuleEnv     = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_GOMODCACHE"
	exactGoBuildEnv      = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_GOCACHE"
	exactGoTempEnv       = "JAPANESE_LAW_MCP_EXACT_CANDIDATE_TMPDIR"
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
	candidateVersion := currentCandidateGoVersion(t, workingDirectory)
	candidateSetup := requireQualityActionStepByID(
		t,
		verify.Steps,
		"candidate-go",
		qualitySetupGoAction,
	)
	if candidateSetup.With["go-version"] != strings.TrimPrefix(candidateVersion, "go") ||
		candidateSetup.With["cache"] != false {
		t.Fatalf("候補再現用 Go setup = %#v", candidateSetup.With)
	}
	record := requireQualityRunStepByID(t, verify.Steps, "record-candidate-go")
	for _, required := range []string{
		exactGoRootEnv,
		exactGoModuleEnv,
		exactGoBuildEnv,
		exactGoTempEnv,
		candidateVersion,
	} {
		if !strings.Contains(record.Run, required) {
			t.Fatalf("候補再現用 Go infrastructure の固定 step に %q がありません", required)
		}
	}
	setup := requireQualityActionStepByID(t, verify.Steps, "primary-go", qualitySetupGoAction)
	if setup.With["go-version"] != qualityGoVersion {
		t.Fatalf("verify job の Go version = %#v", setup.With["go-version"])
	}
	candidateIndex := qualityStepIndex(verify.Steps, "candidate-go")
	recordIndex := qualityStepIndex(verify.Steps, "record-candidate-go")
	primaryIndex := qualityStepIndex(verify.Steps, "primary-go")
	if candidateIndex >= recordIndex || recordIndex >= primaryIndex {
		t.Fatalf(
			"候補再現用 Go の準備順 = candidate %d, record %d, primary %d",
			candidateIndex,
			recordIndex,
			primaryIndex,
		)
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
	for _, obsolete := range []string{"quality-go1.26.5-"} {
		if strings.Contains(source, obsolete) {
			t.Fatalf("脆弱な Go patch 版の設定 %q が残っています", obsolete)
		}
	}
}

// SOT-ENG-020/038: network 取得は修正済み Go、候補実行は manifest の exact Go を使う。
func TestCandidateWorkflowSeparatesBootstrapAndExactToolchains(t *testing.T) {
	t.Parallel()

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("テストの作業ディレクトリを取得できませんでした: %v", err)
	}
	workflow := readQualityWorkflow(
		t,
		workingDirectory,
		"candidate-evaluation.yml",
	)
	job, exists := workflow.Jobs["candidate-evaluation"]
	if !exists {
		t.Fatal("候補評価 workflow に candidate-evaluation job がありません")
	}
	bootstrap := requireQualityActionStepByID(t, job.Steps, "bootstrap-go", qualitySetupGoAction)
	if bootstrap.With["go-version"] != qualityGoVersion || bootstrap.With["cache"] != false {
		t.Fatalf("module 取得用 Go setup = %#v", bootstrap.With)
	}
	exactVersion := strings.TrimPrefix(currentCandidateGoVersion(t, workingDirectory), "go")
	exact := requireQualityActionStepByID(t, job.Steps, "candidate-go", qualitySetupGoAction)
	if exact.With["go-version"] != exactVersion || exact.With["cache"] != false {
		t.Fatalf("候補評価用 exact Go setup = %#v", exact.With)
	}
	bootstrapIndex := qualityStepIndex(job.Steps, "bootstrap-go")
	moduleIndex := qualityNamedStepIndex(job.Steps, "固定 module archive を準備する")
	exactIndex := qualityStepIndex(job.Steps, "candidate-go")
	commandIndex := qualityNamedStepIndex(job.Steps, "閉じた候補評価 command を実行する")
	if bootstrapIndex >= moduleIndex || moduleIndex >= exactIndex || exactIndex >= commandIndex {
		t.Fatalf(
			"候補評価の toolchain 順序 = bootstrap %d, module %d, exact %d, command %d",
			bootstrapIndex,
			moduleIndex,
			exactIndex,
			commandIndex,
		)
	}
	if !strings.Contains(job.Steps[moduleIndex].Run, "GOPROXY=https://proxy.golang.org") ||
		!strings.Contains(job.Steps[commandIndex].Run, "GOPROXY=off GOSUMDB=off") {
		t.Fatal("候補評価の network 準備と閉じた実行が分離されていません")
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

func requireQualityActionStepByID(
	t *testing.T,
	steps []qualityWorkflowStep,
	id string,
	action string,
) qualityWorkflowStep {
	t.Helper()
	for _, step := range steps {
		if step.ID == id && step.Uses == action {
			return step
		}
	}
	t.Fatalf("job に ID %q の action %q がありません", id, action)
	return qualityWorkflowStep{}
}

func requireQualityRunStepByID(
	t *testing.T,
	steps []qualityWorkflowStep,
	id string,
) qualityWorkflowStep {
	t.Helper()
	for _, step := range steps {
		if step.ID == id && step.Run != "" {
			return step
		}
	}
	t.Fatalf("job に ID %q の run step がありません", id)
	return qualityWorkflowStep{}
}

func qualityStepIndex(steps []qualityWorkflowStep, id string) int {
	for index, step := range steps {
		if step.ID == id {
			return index
		}
	}
	return -1
}

func qualityNamedStepIndex(steps []qualityWorkflowStep, name string) int {
	for index, step := range steps {
		if step.Name == name {
			return index
		}
	}
	return -1
}

func readQualityWorkflow(
	t *testing.T,
	workingDirectory string,
	fileName string,
) qualityWorkflow {
	t.Helper()
	workflowPath := filepath.Join(workingDirectory, "..", "..", ".github", "workflows", fileName)
	//nolint:gosec // SOT-ENG-020/038: repository 内の固定 workflow だけを読み取る。
	content, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("workflow %q を読み取れませんでした: %v", fileName, err)
	}
	var workflow qualityWorkflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("workflow %q を解析できませんでした: %v", fileName, err)
	}
	return workflow
}

func currentCandidateGoVersion(t *testing.T, workingDirectory string) string {
	t.Helper()
	root := filepath.Join(workingDirectory, "..", "..")
	base := filepath.Join(root, "testdata", "legalquery", "candidate-evaluations")
	var pointer struct {
		EvaluationID string `json:"evaluationId"`
	}
	readClosedJSON(t, filepath.Join(base, "current.json"), &pointer)
	if !hasSHA256ID(pointer.EvaluationID, "evaluation-sha256-") {
		t.Fatalf("候補評価 pointer の ID が不正です: %q", pointer.EvaluationID)
	}
	var request struct {
		CandidateContentID string `json:"candidateContentId"`
	}
	readClosedJSON(t, filepath.Join(base, "requests", pointer.EvaluationID+".json"), &request)
	if !hasSHA256ID(request.CandidateContentID, "candidate-content-sha256-") {
		t.Fatalf("候補評価 request の content ID が不正です: %q", request.CandidateContentID)
	}
	var manifest struct {
		SemanticSourceSet struct {
			GoToolchainVersion string `json:"goToolchainVersion"`
		} `json:"semanticSourceSet"`
	}
	readClosedJSON(
		t,
		filepath.Join(base, "content-manifests", request.CandidateContentID+".json"),
		&manifest,
	)
	version := manifest.SemanticSourceSet.GoToolchainVersion
	if !strings.HasPrefix(version, "go") || strings.Count(version, ".") != 2 {
		t.Fatalf("候補 content manifest の Go version が不正です: %q", version)
	}
	return version
}

func readClosedJSON(t *testing.T, path string, target any) {
	t.Helper()
	//nolint:gosec // SOT-ENG-038: 検証済み ID から repository 内の固定 artifact だけを読む。
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("候補評価 artifact を読み取れませんでした: %v", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("候補評価 artifact を解析できませんでした: %v", err)
	}
}

func hasSHA256ID(value string, prefix string) bool {
	if len(value) != len(prefix)+64 || !strings.HasPrefix(value, prefix) {
		return false
	}
	for _, character := range value[len(prefix):] {
		if (character < '0' || character > '9') && (character < 'a' || character > 'f') {
			return false
		}
	}
	return true
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
	ID   string         `yaml:"id"`
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}
