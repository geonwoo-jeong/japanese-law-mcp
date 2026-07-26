package releasecontract_test

import (
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
	checkoutAction = "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1"
	setupGoAction  = "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e"
	goReleaser     = "goreleaser/goreleaser-action@f06c13b6b1a9625abc9e6e439d9c05a8f2190e94"
	uploadArtifact = "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a"
	downloadAction = "actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c"
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
		!config.Release.ReplaceExistingDraft ||
		config.Release.UseExistingDraft ||
		config.Release.ReplaceExistingArtifacts ||
		config.Release.Mode != "" ||
		config.Release.TargetCommitish != "{{ .FullCommit }}" {
		t.Fatalf("release = %#v", config.Release)
	}
}

// SOT-DEL-004/SOT-DEL-010/SOT-ENG-020: 品質ゲート後に draft を作り、4対象を確認してから公開する。
func TestReleaseWorkflowContract(t *testing.T) {
	t.Parallel()

	var workflow releaseWorkflow
	readYAML(t, ".github/workflows/release.yml", &workflow)

	if !reflect.DeepEqual(workflow.On.Push.Tags, []string{"v*"}) {
		t.Fatalf("release tag = %#v", workflow.On.Push.Tags)
	}
	if workflow.Permissions["contents"] != "read" {
		t.Fatalf("workflow permissions = %#v", workflow.Permissions)
	}
	for _, name := range []string{"package", "smoke", "publish"} {
		if _, ok := workflow.Jobs[name]; !ok {
			t.Fatalf("job %q がありません", name)
		}
	}
	if len(workflow.Jobs) != 3 {
		t.Fatalf("job 数 = %d, want 3", len(workflow.Jobs))
	}

	assertPackageJob(t, workflow.Jobs["package"])
	assertSmokeJob(t, workflow.Jobs["smoke"])
	assertPublishJob(t, workflow.Jobs["publish"])
}

func assertPackageJob(t *testing.T, job workflowJob) {
	t.Helper()

	if job.Permissions["contents"] != "write" {
		t.Fatalf("package permissions = %#v", job.Permissions)
	}
	checkout := requireActionStep(t, job.Steps, checkoutAction)
	if checkout.With["fetch-depth"] != 0 || checkout.With["persist-credentials"] != false {
		t.Fatalf("checkout inputs = %#v", checkout.With)
	}
	setup := requireActionStep(t, job.Steps, setupGoAction)
	if setup.With["go-version"] != "1.26.5" {
		t.Fatalf("Go version = %#v", setup.With["go-version"])
	}
	releaser := requireActionStep(t, job.Steps, goReleaser)
	if releaser.With["distribution"] != "goreleaser" ||
		releaser.With["version"] != "v2.17.0" ||
		!strings.Contains(asString(releaser.With["args"]), "--release-notes=") {
		t.Fatalf("GoReleaser inputs = %#v", releaser.With)
	}
	requireActionStep(t, job.Steps, uploadArtifact)

	qualityIndex := requireRunStep(t, job.Steps, "go run ./cmd/quality-gate")
	releaseIndex := indexActionStep(job.Steps, goReleaser)
	artifactCheckIndex := requireRunStep(t, job.Steps, "--dist=dist")
	immutableGuardIndex := requireRunStep(t, job.Steps, ".draft == true")
	releaseNotesIndex := requireRunStep(t, job.Steps, "--release-notes=")
	if releaseNotesIndex >= immutableGuardIndex ||
		qualityIndex >= releaseIndex ||
		artifactCheckIndex <= releaseIndex ||
		immutableGuardIndex >= releaseIndex {
		t.Fatalf(
			"step 順序が不正です: notes=%d, guard=%d, quality=%d, release=%d, check=%d",
			releaseNotesIndex,
			immutableGuardIndex,
			qualityIndex,
			releaseIndex,
			artifactCheckIndex,
		)
	}
	guard := job.Steps[immutableGuardIndex].Run
	for _, fragment := range []string{
		"${GITHUB_API_URL}",
		"${GITHUB_REF_NAME}",
		"404",
		"200",
	} {
		if !strings.Contains(guard, fragment) {
			t.Fatalf("公開済み release guard に %q がありません: %s", fragment, guard)
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

	if asString(job.Needs) != "package" || job.RunsOn != "${{ matrix.runner }}" {
		t.Fatalf("smoke job = needs %#v, runs-on %q", job.Needs, job.RunsOn)
	}
	requireActionStep(t, job.Steps, checkoutAction)
	requireActionStep(t, job.Steps, setupGoAction)
	requireActionStep(t, job.Steps, downloadAction)
	smokeIndex := requireRunStep(t, job.Steps, "go run ./cmd/release-check")
	if !strings.Contains(job.Steps[smokeIndex].Run, "--repository=.") {
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

	if asString(job.Needs) != "smoke" || job.Permissions["contents"] != "write" {
		t.Fatalf("publish job = needs %#v, permissions %#v", job.Needs, job.Permissions)
	}
	index := requireRunStep(t, job.Steps, "gh release edit")
	if !strings.Contains(job.Steps[index].Run, "--draft=false") {
		t.Fatalf("release 公開 command = %q", job.Steps[index].Run)
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
			Tags []string `yaml:"tags"`
		} `yaml:"push"`
	} `yaml:"on"`
	Permissions map[string]string      `yaml:"permissions"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

type workflowJob struct {
	RunsOn      string            `yaml:"runs-on"`
	Needs       any               `yaml:"needs"`
	Permissions map[string]string `yaml:"permissions"`
	Steps       []workflowStep    `yaml:"steps"`
	Strategy    struct {
		Matrix struct {
			Include []map[string]string `yaml:"include"`
		} `yaml:"matrix"`
	} `yaml:"strategy"`
}

type workflowStep struct {
	Uses string         `yaml:"uses"`
	Run  string         `yaml:"run"`
	With map[string]any `yaml:"with"`
}
