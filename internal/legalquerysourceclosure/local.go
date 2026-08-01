package legalquerysourceclosure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const maximumGoEnvironmentBytes = 32 << 10

// NewLocalBuilder は、PATH 上の Go executable と固定済み cache を使う builder を返す。
// HOME のない閉じた CI 環境では allowlist 済み infrastructure path を使い、
// 通常環境では既定 cache を閉じた go env command で発見する。
func NewLocalBuilder(ctx context.Context) (Builder, error) {
	if ctx == nil {
		return Builder{}, fmt.Errorf("go cache discovery context は nil にできません")
	}
	if err := ctx.Err(); err != nil {
		return Builder{}, fmt.Errorf("go cache discovery が中止されました: %w", err)
	}
	binary, err := exec.LookPath("go")
	if err != nil {
		return Builder{}, fmt.Errorf("PATH から Go executable を解決できません")
	}
	binary, err = filepath.EvalSymlinks(binary)
	if err != nil {
		return Builder{}, fmt.Errorf("go executable の symlink を解決できません")
	}
	binary, err = validateGoBinary(binary)
	if err != nil {
		return Builder{}, err
	}
	if os.Getenv("HOME") == "" {
		return newClosedEnvironmentBuilder(binary)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Builder{}, fmt.Errorf("go cache discovery 用 home directory を解決できません")
	}
	command := exec.CommandContext(ctx, binary, "env", "-json", "GOROOT", "GOMODCACHE", "GOCACHE") //nolint:gosec // SOT-ENG-038: descriptor 検証済み Go executable と固定 go env argv だけを起動する。
	command.Env = []string{
		"PATH=" + filepath.Dir(binary),
		"HOME=" + home,
		"GOWORK=off",
		"GOENV=off",
		"GOTOOLCHAIN=local",
		"GOMAXPROCS=1",
	}
	var stderr cappedBuffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return Builder{}, commandFailure("go env cache discovery", err, stderr.String())
	}
	if len(output) > maximumGoEnvironmentBytes {
		return Builder{}, fmt.Errorf("go env cache discovery output が上限を超えています")
	}
	var environment struct {
		GoRoot      string `json:"GOROOT"`
		ModuleCache string `json:"GOMODCACHE"`
		BuildCache  string `json:"GOCACHE"`
	}
	if err := json.Unmarshal(output, &environment); err != nil {
		return Builder{}, fmt.Errorf("go env cache discovery output を解析できません")
	}
	return newInfrastructureBuilder(ToolchainInfrastructure{
		GoBinary:           binary,
		GoRoot:             environment.GoRoot,
		ModuleCache:        environment.ModuleCache,
		BuildCache:         environment.BuildCache,
		TemporaryDirectory: os.TempDir(),
	})
}

func newClosedEnvironmentBuilder(goBinary string) (Builder, error) {
	required := []struct {
		name  string
		value string
	}{
		{name: "GOROOT", value: os.Getenv("GOROOT")},
		{name: "GOMODCACHE", value: os.Getenv("GOMODCACHE")},
		{name: "GOCACHE", value: os.Getenv("GOCACHE")},
		{name: "TMPDIR", value: os.Getenv("TMPDIR")},
	}
	for _, variable := range required {
		if variable.value == "" {
			return Builder{}, fmt.Errorf(
				"閉じた go cache 環境に %s がありません",
				variable.name,
			)
		}
	}
	return newInfrastructureBuilder(ToolchainInfrastructure{
		GoBinary:           goBinary,
		GoRoot:             required[0].value,
		ModuleCache:        required[1].value,
		BuildCache:         required[2].value,
		TemporaryDirectory: required[3].value,
	})
}

func newInfrastructureBuilder(infrastructure ToolchainInfrastructure) (Builder, error) {
	toolchain, err := NewCommandToolchain(infrastructure)
	if err != nil {
		return Builder{}, err
	}
	modules, err := NewModuleCacheProvider(infrastructure.ModuleCache)
	if err != nil {
		return Builder{}, err
	}
	return Builder{Toolchain: toolchain, Modules: modules}, nil
}
