package releasecontract_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

const (
	checkoutAction      = "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1"
	setupGoAction       = "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e"
	goReleaser          = "goreleaser/goreleaser-action@f06c13b6b1a9625abc9e6e439d9c05a8f2190e94"
	uploadArtifact      = "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a"
	downloadAction      = "actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c"
	releasePleaseAction = "googleapis/release-please-action@45996ed1f6d02564a971a2fa1b5860e934307cf7"
)

// SOT-DEL-010/SOT-DEL-004: 公式 archive の対象、内容、版および生成元を固定する。
func TestGoReleaserContract(t *testing.T) {
	t.Parallel()

	var config goReleaserConfig
	readYAML(t, ".goreleaser.yaml", &config)

	if config.Version != 2 || config.ProjectName != "japanese-law-mcp" {
		t.Fatalf("GoReleaser の基本設定 = version %d, project %q", config.Version, config.ProjectName)
	}
	if len(config.Builds) != 1 {
		t.Fatalf("build 数 = %d, want 1", len(config.Builds))
	}
	build := config.Builds[0]
	if build.ID != "japanese-law-mcp" ||
		build.Main != "./cmd/japanese-law-mcp" ||
		build.Binary != "japanese-law-mcp" {
		t.Fatalf("build = %#v", build)
	}
	assertStringSet(t, "goos", build.GOOS, []string{"darwin", "windows"})
	assertStringSet(t, "goarch", build.GOARCH, []string{"amd64", "arm64"})
	assertContains(t, "build env", build.Env, "CGO_ENABLED=0")
	assertContains(t, "build flags", build.Flags, "-trimpath")
	assertContains(
		t,
		"build ldflags",
		build.LDFlags,
		"-s -w -X github.com/geonwoo-jeong/japanese-law-mcp/internal/buildinfo.version={{ .Version }}",
	)
	if build.ModTimestamp != "{{ .CommitTimestamp }}" {
		t.Fatalf("mod_timestamp = %q", build.ModTimestamp)
	}

	if len(config.Archives) != 1 {
		t.Fatalf("archive 設定数 = %d, want 1", len(config.Archives))
	}
	archive := config.Archives[0]
	assertStringSet(t, "archive ids", archive.IDs, []string{"japanese-law-mcp"})
	assertStringSet(t, "archive formats", archive.Formats, []string{"tar.gz"})
	if archive.NameTemplate != "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}" {
		t.Fatalf("archive name_template = %q", archive.NameTemplate)
	}
	assertStringSet(t, "archive files", archive.Files, []string{"none*"})
	if len(archive.FormatOverrides) != 1 ||
		archive.FormatOverrides[0].GOOS != "windows" ||
		!reflect.DeepEqual(archive.FormatOverrides[0].Formats, []string{"zip"}) {
		t.Fatalf("format_overrides = %#v", archive.FormatOverrides)
	}

	if config.Checksum.Algorithm != "sha256" ||
		config.Checksum.NameTemplate != "{{ .ProjectName }}_{{ .Version }}_checksums.txt" {
		t.Fatalf("checksum = %#v", config.Checksum)
	}
	if config.Source.Enabled {
		t.Fatal("GoReleaser の source archive が有効です")
	}
	if !config.Release.Draft ||
		!config.Release.IncludeMeta ||
		config.Release.ReplaceExistingDraft ||
		!config.Release.UseExistingDraft ||
		!config.Release.ReplaceExistingArtifacts ||
		config.Release.Mode != "keep-existing" ||
		config.Release.TargetCommitish != "{{ .FullCommit }}" {
		t.Fatalf("release = %#v", config.Release)
	}
}

// SOT-DEL-014: Release Please が更新する版、変更履歴およびリリース契約を固定する。
func TestReleasePleaseConfigurationContract(t *testing.T) {
	t.Parallel()

	var manifest map[string]string
	readJSON(t, ".release-please-manifest.json", &manifest)
	if !reflect.DeepEqual(manifest, map[string]string{".": "0.0.0"}) {
		t.Fatalf("release manifest = %#v", manifest)
	}

	var config releasePleaseConfig
	readJSON(t, "release-please-config.json", &config)
	if config.Schema !=
		"https://raw.githubusercontent.com/googleapis/release-please/v17.6.0/schemas/config.json" {
		t.Fatalf("release-please schema = %q", config.Schema)
	}
	if !config.AlwaysUpdate ||
		config.GroupPullRequestTitlePattern != "chore: ${version} をリリースする" {
		t.Fatalf("release-please root config = %#v", config)
	}

	root, exists := config.Packages["."]
	if !exists || len(config.Packages) != 1 {
		t.Fatalf("release-please packages = %#v", config.Packages)
	}
	if root.ReleaseType != "go" ||
		root.IncludeComponentInTag ||
		!root.IncludeVInTag ||
		!root.Draft ||
		!root.ForceTagCreation ||
		root.ChangelogPath != "CHANGELOG.md" ||
		root.PullRequestTitlePattern != "chore${scope}: ${version} をリリースする" {
		t.Fatalf("release-please root package = %#v", root)
	}
	if strings.TrimSpace(root.PullRequestHeader) == "" ||
		strings.TrimSpace(root.PullRequestFooter) == "" {
		t.Fatalf("Release PR の日本語説明がありません: %#v", root)
	}
	if len(root.ExtraFiles) != 1 ||
		root.ExtraFiles[0].Type != "generic" ||
		root.ExtraFiles[0].Path != "release-notes/CURRENT.md" {
		t.Fatalf("release-please extra-files = %#v", root.ExtraFiles)
	}

	gotSections := make([]string, 0, len(root.ChangelogSections))
	for _, section := range root.ChangelogSections {
		if !section.Hidden {
			gotSections = append(gotSections, section.Type+"|"+section.Section)
		}
	}
	assertStringSet(
		t,
		"公開する変更履歴セクション",
		gotSections,
		[]string{
			"docs|文書",
			"feat|機能",
			"fix|修正",
			"perf|性能",
			"refactor|内部改善",
			"revert|取り消し",
		},
	)

	current, err := os.ReadFile(filepath.Join(repositoryRoot(t), "release-notes", "CURRENT.md"))
	if err != nil {
		t.Fatalf("現在のリリース契約を読み込めません: %v", err)
	}
	if !strings.Contains(
		string(current),
		"# Japanese Law MCP v0.0.0 <!-- x-release-please-version -->",
	) {
		t.Fatalf("現在のリリース契約に版更新注釈がありません: %s", current)
	}
}

// SOT-DEL-004/SOT-DEL-010/SOT-DEL-014/SOT-ENG-020:
// Release PR の merge 後だけ配布物を作り、4対象を確認してから公開する。
func TestReleaseWorkflowContract(t *testing.T) {
	t.Parallel()

	var workflow releaseWorkflow
	readYAML(t, ".github/workflows/release.yml", &workflow)

	if !reflect.DeepEqual(workflow.On.Push.Branches, []string{"main"}) ||
		len(workflow.On.Push.Tags) != 0 {
		t.Fatalf("release push 条件 = %#v", workflow.On.Push)
	}
	if workflow.Permissions["contents"] != "read" {
		t.Fatalf("workflow permissions = %#v", workflow.Permissions)
	}
	for _, name := range []string{"release-please", "package", "smoke", "publish"} {
		if _, ok := workflow.Jobs[name]; !ok {
			t.Fatalf("job %q がありません", name)
		}
	}
	if len(workflow.Jobs) != 4 {
		t.Fatalf("job 数 = %d, want 4", len(workflow.Jobs))
	}

	assertReleasePleaseJob(t, workflow.Jobs["release-please"])
	assertPackageJob(t, workflow.Jobs["package"])
	assertSmokeJob(t, workflow.Jobs["smoke"])
	assertPublishJob(t, workflow.Jobs["publish"])
}

func assertReleasePleaseJob(t *testing.T, job workflowJob) {
	t.Helper()

	if job.Permissions["contents"] != "write" ||
		job.Permissions["issues"] != "write" ||
		job.Permissions["pull-requests"] != "write" {
		t.Fatalf("release-please permissions = %#v", job.Permissions)
	}
	step := requireActionStep(t, job.Steps, releasePleaseAction)
	if step.ID != "release" ||
		step.With["config-file"] != "release-please-config.json" ||
		step.With["manifest-file"] != ".release-please-manifest.json" ||
		step.With["target-branch"] != "main" {
		t.Fatalf("release-please action = %#v", step)
	}
	for name, want := range map[string]string{
		"release_ready": "${{ steps.resolve.outputs.release_ready }}",
		"sha":           "${{ steps.resolve.outputs.sha }}",
		"tag_name":      "${{ steps.resolve.outputs.tag_name }}",
		"upload_url":    "${{ steps.resolve.outputs.upload_url }}",
		"version":       "${{ steps.resolve.outputs.version }}",
	} {
		if job.Outputs[name] != want {
			t.Fatalf("release-please output %q = %q, want %q", name, job.Outputs[name], want)
		}
	}

	checkout := requireActionStep(t, job.Steps, checkoutAction)
	if checkout.With["ref"] != "${{ github.sha }}" ||
		checkout.With["fetch-depth"] != 0 ||
		checkout.With["persist-credentials"] != false {
		t.Fatalf("release-please checkout inputs = %#v", checkout.With)
	}

	resolveIndex := requireRunStep(t, job.Steps, "release_ready=")
	resolve := job.Steps[resolveIndex]
	if resolve.ID != "resolve" ||
		resolve.Env["GH_TOKEN"] != "${{ secrets.GITHUB_TOKEN }}" ||
		resolve.Env["ACTION_RELEASE_CREATED"] != "${{ steps.release.outputs.release_created }}" {
		t.Fatalf("release の再開判定 step = %#v", resolve)
	}
	for _, fragment := range []string{
		".release-please-manifest.json",
		"$GITHUB_SHA",
		"releases/tags/",
		"/git/ref/tags/",
		".draft == true",
		".upload_url",
		"ACTION_RELEASE_CREATED",
		"GITHUB_OUTPUT",
	} {
		if !strings.Contains(resolve.Run, fragment) {
			t.Fatalf("release の再開判定に %q がありません: %s", fragment, resolve.Run)
		}
	}
}

func assertPackageJob(t *testing.T, job workflowJob) {
	t.Helper()

	if job.Permissions["contents"] != "write" ||
		!needsInclude(job.Needs, "release-please") ||
		!strings.Contains(job.If, "release_ready == 'true'") {
		t.Fatalf(
			"package 境界 = needs %#v, if %q, permissions %#v",
			job.Needs,
			job.If,
			job.Permissions,
		)
	}
	if job.Outputs["upload-url"] != "${{ needs.release-please.outputs.upload_url }}" ||
		job.Env["RELEASE_UPLOAD_URL"] != "${{ needs.release-please.outputs.upload_url }}" {
		t.Fatalf("package の release 識別子 = outputs %#v, env %#v", job.Outputs, job.Env)
	}
	checkout := requireActionStep(t, job.Steps, checkoutAction)
	if checkout.With["fetch-depth"] != 0 ||
		checkout.With["persist-credentials"] != false ||
		checkout.With["ref"] != "${{ needs.release-please.outputs.sha }}" {
		t.Fatalf("checkout inputs = %#v", checkout.With)
	}
	setup := requireActionStep(t, job.Steps, setupGoAction)
	if setup.With["go-version"] != "1.26.5" {
		t.Fatalf("Go version = %#v", setup.With["go-version"])
	}
	releaser := requireActionStep(t, job.Steps, goReleaser)
	if releaser.With["distribution"] != "goreleaser" ||
		releaser.With["version"] != "v2.17.0" ||
		asString(releaser.With["args"]) != "release --clean" {
		t.Fatalf("GoReleaser inputs = %#v", releaser.With)
	}
	upload := requireActionStep(t, job.Steps, uploadArtifact)
	if upload.With["name"] != "release-dist-${{ needs.release-please.outputs.sha }}" ||
		upload.With["overwrite"] != true {
		t.Fatalf("smoke test 用 artifact = %#v", upload.With)
	}

	qualityIndex := requireRunStep(t, job.Steps, "go run ./cmd/quality-gate")
	releaseIndex := indexActionStep(job.Steps, goReleaser)
	artifactCheckIndex := requireRunStep(t, job.Steps, "--dist=dist")
	immutableGuardIndex := requireRunStep(t, job.Steps, ".draft == true")
	releaseContractIndex := requireRunStep(t, job.Steps, "release-notes/CURRENT.md")
	sourceIndex := requireRunStep(t, job.Steps, "git rev-parse")
	if releaseContractIndex >= immutableGuardIndex ||
		sourceIndex >= immutableGuardIndex ||
		qualityIndex >= releaseIndex ||
		artifactCheckIndex <= releaseIndex ||
		immutableGuardIndex >= releaseIndex {
		t.Fatalf(
			"step 順序が不正です: contract=%d, source=%d, guard=%d, quality=%d, release=%d, check=%d",
			releaseContractIndex,
			sourceIndex,
			immutableGuardIndex,
			qualityIndex,
			releaseIndex,
			artifactCheckIndex,
		)
	}
	guard := job.Steps[immutableGuardIndex].Run
	for _, fragment := range []string{
		"${GITHUB_API_URL}",
		"${RELEASE_TAG}",
		"${RELEASE_SHA}",
		"${RELEASE_UPLOAD_URL}",
		"200",
		".draft == true",
		".upload_url == $upload_url",
	} {
		if !strings.Contains(guard, fragment) {
			t.Fatalf("draft release guard に %q がありません: %s", fragment, guard)
		}
	}
	for _, step := range job.Steps {
		if strings.Contains(step.Run, "go run ./cmd/release-check") &&
			!strings.Contains(step.Run, "--repository=.") {
			t.Fatalf("release-check に repository がありません: %q", step.Run)
		}
	}
}

func assertSmokeJob(t *testing.T, job workflowJob) {
	t.Helper()

	if !needsInclude(job.Needs, "package") || job.RunsOn != "${{ matrix.runner }}" {
		t.Fatalf("smoke job = needs %#v, runs-on %q", job.Needs, job.RunsOn)
	}
	checkout := requireActionStep(t, job.Steps, checkoutAction)
	if checkout.With["ref"] != "${{ needs.package.outputs.sha }}" {
		t.Fatalf("smoke checkout inputs = %#v", checkout.With)
	}
	requireActionStep(t, job.Steps, setupGoAction)
	download := requireActionStep(t, job.Steps, downloadAction)
	if download.With["name"] != "release-dist-${{ needs.package.outputs.sha }}" {
		t.Fatalf("smoke が取得する artifact 名 = %#v", download.With["name"])
	}
	smokeIndex := requireRunStep(t, job.Steps, "go run ./cmd/release-check")
	if !strings.Contains(job.Steps[smokeIndex].Run, "--repository=.") ||
		!strings.Contains(job.Steps[smokeIndex].Run, "release-notes/CURRENT.md") {
		t.Fatalf("smoke release-check に repository がありません: %q", job.Steps[smokeIndex].Run)
	}

	got := make([]string, 0, len(job.Strategy.Matrix.Include))
	for _, row := range job.Strategy.Matrix.Include {
		got = append(got, row["runner"]+"|"+row["os"]+"|"+row["arch"])
	}
	want := []string{
		"macos-15-intel|darwin|amd64",
		"macos-15|darwin|arm64",
		"windows-2025|windows|amd64",
		"windows-11-arm|windows|arm64",
	}
	assertStringSet(t, "smoke matrix", got, want)
}

func assertPublishJob(t *testing.T, job workflowJob) {
	t.Helper()

	if !needsInclude(job.Needs, "package") ||
		!needsInclude(job.Needs, "smoke") ||
		job.Permissions["contents"] != "write" {
		t.Fatalf("publish job = needs %#v, permissions %#v", job.Needs, job.Permissions)
	}
	checkout := requireActionStep(t, job.Steps, checkoutAction)
	if checkout.With["ref"] != "${{ needs.package.outputs.sha }}" ||
		checkout.With["fetch-depth"] != 0 ||
		checkout.With["persist-credentials"] != false {
		t.Fatalf("publish checkout inputs = %#v", checkout.With)
	}
	setup := requireActionStep(t, job.Steps, setupGoAction)
	if setup.With["go-version"] != "1.26.5" {
		t.Fatalf("publish の Go version = %#v", setup.With["go-version"])
	}
	if job.Env["RELEASE_UPLOAD_URL"] != "${{ needs.package.outputs.upload-url }}" {
		t.Fatalf("publish の release 識別子 = %#v", job.Env)
	}
	index := requireRunStep(t, job.Steps, "gh release edit")
	for _, fragment := range []string{
		"gh api",
		"go run ./cmd/release-notes",
		"CHANGELOG.md",
		"release-notes/CURRENT.md",
		"git rev-parse",
		"/git/ref/tags/",
		"${RELEASE_SHA}",
		"${RELEASE_UPLOAD_URL}",
		".upload_url == $upload_url",
		"--notes-file",
		"--draft=false",
	} {
		if !strings.Contains(job.Steps[index].Run, fragment) {
			t.Fatalf("release 公開 command に %q がありません: %q", fragment, job.Steps[index].Run)
		}
	}
	if strings.Contains(job.Steps[index].Run, "'.body") ||
		strings.Contains(job.Steps[index].Run, `".body`) {
		t.Fatalf("公開する release notes が可変な remote body を参照しています: %q", job.Steps[index].Run)
	}
}

func needsInclude(value any, want string) bool {
	switch needs := value.(type) {
	case string:
		return needs == want
	case []any:
		for _, need := range needs {
			if asString(need) == want {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func requireActionStep(t *testing.T, steps []workflowStep, action string) workflowStep {
	t.Helper()

	index := indexActionStep(steps, action)
	if index < 0 {
		t.Fatalf("action %q がありません", action)
	}
	return steps[index]
}

func indexActionStep(steps []workflowStep, action string) int {
	for index, step := range steps {
		if step.Uses == action {
			return index
		}
	}
	return -1
}

func requireRunStep(t *testing.T, steps []workflowStep, fragment string) int {
	t.Helper()

	for index, step := range steps {
		if strings.Contains(step.Run, fragment) {
			return index
		}
	}
	t.Fatalf("run step %q がありません", fragment)
	return -1
}

func assertContains(t *testing.T, name string, values []string, want string) {
	t.Helper()

	if !slices.Contains(values, want) {
		t.Fatalf("%s = %#v, want %q", name, values, want)
	}
}

func assertStringSet(t *testing.T, name string, got, want []string) {
	t.Helper()

	gotCopy := append([]string(nil), got...)
	wantCopy := append([]string(nil), want...)
	slices.Sort(gotCopy)
	slices.Sort(wantCopy)
	if !reflect.DeepEqual(gotCopy, wantCopy) {
		t.Fatalf("%s = %#v, want %#v", name, gotCopy, wantCopy)
	}
}

func asString(value any) string {
	text, _ := value.(string)
	return text
}

func readYAML(t *testing.T, relativePath string, target any) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("%s を読み込めません: %v", relativePath, err)
	}
	if err := yaml.Unmarshal(content, target); err != nil {
		t.Fatalf("%s を YAML として解釈できません: %v", relativePath, err)
	}
}

func readJSON(t *testing.T, relativePath string, target any) {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(repositoryRoot(t), filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("%s を読み込めません: %v", relativePath, err)
	}
	if err := json.Unmarshal(content, target); err != nil {
		t.Fatalf("%s を JSON として解釈できません: %v", relativePath, err)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("テストファイルの位置を確認できません")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

type goReleaserConfig struct {
	Version     int              `yaml:"version"`
	ProjectName string           `yaml:"project_name"`
	Builds      []releaseBuild   `yaml:"builds"`
	Archives    []releaseArchive `yaml:"archives"`
	Checksum    releaseChecksum  `yaml:"checksum"`
	Source      releaseSource    `yaml:"source"`
	Release     releaseSettings  `yaml:"release"`
}

type releaseBuild struct {
	ID           string   `yaml:"id"`
	Main         string   `yaml:"main"`
	Binary       string   `yaml:"binary"`
	Env          []string `yaml:"env"`
	GOOS         []string `yaml:"goos"`
	GOARCH       []string `yaml:"goarch"`
	Flags        []string `yaml:"flags"`
	LDFlags      []string `yaml:"ldflags"`
	ModTimestamp string   `yaml:"mod_timestamp"`
}

type releaseArchive struct {
	IDs             []string         `yaml:"ids"`
	Formats         []string         `yaml:"formats"`
	NameTemplate    string           `yaml:"name_template"`
	FormatOverrides []formatOverride `yaml:"format_overrides"`
	Files           []string         `yaml:"files"`
}

type formatOverride struct {
	GOOS    string   `yaml:"goos"`
	Formats []string `yaml:"formats"`
}

type releaseChecksum struct {
	NameTemplate string `yaml:"name_template"`
	Algorithm    string `yaml:"algorithm"`
}

type releaseSource struct {
	Enabled bool `yaml:"enabled"`
}

type releaseSettings struct {
	Draft                    bool   `yaml:"draft"`
	ReplaceExistingDraft     bool   `yaml:"replace_existing_draft"`
	UseExistingDraft         bool   `yaml:"use_existing_draft"`
	ReplaceExistingArtifacts bool   `yaml:"replace_existing_artifacts"`
	Mode                     string `yaml:"mode"`
	TargetCommitish          string `yaml:"target_commitish"`
	IncludeMeta              bool   `yaml:"include_meta"`
}

type releaseWorkflow struct {
	On struct {
		Push struct {
			Branches []string `yaml:"branches"`
			Tags     []string `yaml:"tags"`
		} `yaml:"push"`
	} `yaml:"on"`
	Permissions map[string]string      `yaml:"permissions"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

type workflowJob struct {
	RunsOn      string            `yaml:"runs-on"`
	Needs       any               `yaml:"needs"`
	If          string            `yaml:"if"`
	Permissions map[string]string `yaml:"permissions"`
	Outputs     map[string]string `yaml:"outputs"`
	Env         map[string]string `yaml:"env"`
	Steps       []workflowStep    `yaml:"steps"`
	Strategy    struct {
		Matrix struct {
			Include []map[string]string `yaml:"include"`
		} `yaml:"matrix"`
	} `yaml:"strategy"`
}

type workflowStep struct {
	ID   string            `yaml:"id"`
	Uses string            `yaml:"uses"`
	If   string            `yaml:"if"`
	Run  string            `yaml:"run"`
	Env  map[string]string `yaml:"env"`
	With map[string]any    `yaml:"with"`
}

type releasePleaseConfig struct {
	Schema                       string                          `json:"$schema"`
	AlwaysUpdate                 bool                            `json:"always-update"`
	GroupPullRequestTitlePattern string                          `json:"group-pull-request-title-pattern"`
	Packages                     map[string]releasePleasePackage `json:"packages"`
}

type releasePleasePackage struct {
	ReleaseType             string                          `json:"release-type"`
	IncludeComponentInTag   bool                            `json:"include-component-in-tag"`
	IncludeVInTag           bool                            `json:"include-v-in-tag"`
	Draft                   bool                            `json:"draft"`
	ForceTagCreation        bool                            `json:"force-tag-creation"`
	ChangelogPath           string                          `json:"changelog-path"`
	PullRequestTitlePattern string                          `json:"pull-request-title-pattern"`
	PullRequestHeader       string                          `json:"pull-request-header"`
	PullRequestFooter       string                          `json:"pull-request-footer"`
	ChangelogSections       []releasePleaseChangelogSection `json:"changelog-sections"`
	ExtraFiles              []releasePleaseExtraFile        `json:"extra-files"`
}

type releasePleaseChangelogSection struct {
	Type    string `json:"type"`
	Section string `json:"section"`
	Hidden  bool   `json:"hidden"`
}

type releasePleaseExtraFile struct {
	Type string `json:"type"`
	Path string `json:"path"`
}
