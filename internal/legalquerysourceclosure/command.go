package legalquerysourceclosure

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const maximumCommandErrorBytes = 64 << 10

// ToolchainInfrastructure は、意味入力に使わない固定 infrastructure path を保持する。
type ToolchainInfrastructure struct {
	GoBinary           string
	GoRoot             string
	ModuleCache        string
	BuildCache         string
	TemporaryDirectory string
}

// CommandToolchain は、継承環境を使わず固定 Go command を実行する。
type CommandToolchain struct {
	infrastructure ToolchainInfrastructure
}

// NewCommandToolchain は、command 実行前に全 infrastructure path を検証する。
func NewCommandToolchain(infrastructure ToolchainInfrastructure) (*CommandToolchain, error) {
	binary, err := validateGoBinary(infrastructure.GoBinary)
	if err != nil {
		return nil, err
	}
	goRoot, err := validateInfrastructureDirectory("GOROOT", infrastructure.GoRoot)
	if err != nil {
		return nil, err
	}
	moduleCache, err := validateInfrastructureDirectory("GOMODCACHE", infrastructure.ModuleCache)
	if err != nil {
		return nil, err
	}
	buildCache, err := validateInfrastructureDirectory("GOCACHE", infrastructure.BuildCache)
	if err != nil {
		return nil, err
	}
	temporaryDirectory, err := validateInfrastructureDirectory("TMPDIR", infrastructure.TemporaryDirectory)
	if err != nil {
		return nil, err
	}
	return &CommandToolchain{infrastructure: ToolchainInfrastructure{
		GoBinary:           binary,
		GoRoot:             goRoot,
		ModuleCache:        moduleCache,
		BuildCache:         buildCache,
		TemporaryDirectory: temporaryDirectory,
	}}, nil
}

// Version は、固定 executable の go version 出力から release version だけを返す。
func (t *CommandToolchain) Version(ctx context.Context, repositoryPath string) (string, error) {
	if t == nil {
		return "", fmt.Errorf("command toolchain が初期化されていません")
	}
	command := exec.CommandContext(ctx, t.infrastructure.GoBinary, "version") //nolint:gosec // SOT-ENG-038: descriptor 検証済み Go executable と固定 argv だけを起動する。
	command.Dir = repositoryPath
	command.Env = t.environment(FixedBuildContext())
	var stderr cappedBuffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return "", commandFailure("go version", err, stderr.String())
	}
	if len(output) > 128 {
		return "", fmt.Errorf("go version output が上限を超えています")
	}
	fields := strings.Fields(string(output))
	if len(fields) != 4 || fields[0] != "go" || fields[1] != "version" || strings.ContainsAny(fields[2], "\x00\r\n") {
		return "", fmt.Errorf("go version output が不正です")
	}
	return fields[2], nil
}

// ListDependencies は、固定環境で go list -deps -json -mod=readonly を起動する。
func (t *CommandToolchain) ListDependencies(ctx context.Context, request ListRequest) (io.ReadCloser, error) {
	if t == nil {
		return nil, fmt.Errorf("command toolchain が初期化されていません")
	}
	if err := request.BuildContext.validate(); err != nil {
		return nil, err
	}
	if len(request.PackageRoots) == 0 {
		return nil, fmt.Errorf("component package root が指定されていません")
	}
	arguments := []string{"list", "-deps", "-json", "-mod=readonly"}
	for _, root := range request.PackageRoots {
		if _, err := validateRepositoryRelativePath(root); err != nil {
			return nil, fmt.Errorf("component package root が不正です: %w", err)
		}
		arguments = append(arguments, "./"+root)
	}
	command := exec.CommandContext(ctx, t.infrastructure.GoBinary, arguments...) //nolint:gosec // SOT-ENG-038: descriptor 検証済み Go executable と閉じた package root の argv だけを起動する。
	command.Dir = request.RepositoryPath
	command.Env = t.environment(request.BuildContext)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("go list stdout を準備できません")
	}
	stderr := &cappedBuffer{}
	command.Stderr = stderr
	if err := command.Start(); err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("go list を起動できません: %w", err)
	}
	return &commandOutput{
		stdout:  stdout,
		command: command,
		stderr:  stderr,
	}, nil
}

func (t *CommandToolchain) environment(buildContext BuildContext) []string {
	return []string{
		"PATH=" + filepath.Dir(t.infrastructure.GoBinary),
		"GOROOT=" + t.infrastructure.GoRoot,
		"GOMODCACHE=" + t.infrastructure.ModuleCache,
		"GOCACHE=" + t.infrastructure.BuildCache,
		"TMPDIR=" + t.infrastructure.TemporaryDirectory,
		"GOWORK=off",
		"GOENV=off",
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GOFLAGS=-mod=readonly -buildvcs=false",
		"GOOS=" + buildContext.GOOS(),
		"GOARCH=" + buildContext.GOARCH(),
		"GOAMD64=" + buildContext.GOAMD64(),
		"GOEXPERIMENT=" + buildContext.GOEXPERIMENT(),
		fmt.Sprintf("CGO_ENABLED=%d", buildContext.CGOEnabled()),
		"GOMAXPROCS=1",
	}
}

type commandOutput struct {
	stdout  io.ReadCloser
	command *exec.Cmd
	stderr  *cappedBuffer
	once    sync.Once
	err     error
}

func (o *commandOutput) Read(target []byte) (int, error) {
	return o.stdout.Read(target)
}

func (o *commandOutput) Close() error {
	o.once.Do(func() {
		closeErr := o.stdout.Close()
		waitErr := o.command.Wait()
		if waitErr != nil {
			o.err = commandFailure("go list", waitErr, o.stderr.String())
		} else if closeErr != nil {
			o.err = fmt.Errorf("go list stdout を閉じられません: %w", closeErr)
		}
	})
	return o.err
}

type cappedBuffer struct {
	buffer    bytes.Buffer
	truncated bool
}

func (b *cappedBuffer) Write(value []byte) (int, error) {
	originalLength := len(value)
	remaining := maximumCommandErrorBytes - b.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
			b.truncated = true
		}
		_, _ = b.buffer.Write(value)
	} else if len(value) != 0 {
		b.truncated = true
	}
	return originalLength, nil
}

func (b *cappedBuffer) String() string {
	value := strings.TrimSpace(b.buffer.String())
	if b.truncated {
		value += "…"
	}
	return value
}

func commandFailure(operation string, err error, stderr string) error {
	if stderr == "" {
		return fmt.Errorf("%s が失敗しました: %w", operation, err)
	}
	return fmt.Errorf("%s が失敗しました: %w: %s", operation, err, stderr)
}

func validateGoBinary(value string) (string, error) {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("go executable を解決できません")
	}
	info, err := os.Lstat(absolute)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("go executable が symlink ではない実行可能 file ではありません")
	}
	return absolute, nil
}

func validateInfrastructureDirectory(name string, value string) (string, error) {
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("%s path を解決できません", name)
	}
	info, err := os.Lstat(absolute)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", fmt.Errorf("%s は symlink ではない directory でなければなりません", name)
	}
	return absolute, nil
}
