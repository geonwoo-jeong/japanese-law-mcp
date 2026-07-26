package githook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/provideronboarding"
)

const hooksPath = ".githooks"

type qualityGateRunner func(
	context.Context,
	string,
	string,
	string,
	string,
	[]string,
	io.Writer,
	io.Writer,
) error

type providerOnboardingRunner func(
	context.Context,
	provideronboarding.Options,
) error

type application struct {
	repository         string
	stdin              io.Reader
	stdout             io.Writer
	stderr             io.Writer
	warmUp             func(context.Context, string) error
	qualityGate        qualityGateRunner
	providerOnboarding providerOnboardingRunner
	indexPinned        func(string)
}

// Execute は、リポジトリ用 Git hook の操作を実行して終了コードを返す。
func Execute(
	ctx context.Context,
	args []string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
) int {
	repository, err := findRepository(ctx)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Git リポジトリを特定できませんでした: %v\n", err)
		return exitCode(err)
	}

	app := &application{
		repository:         repository,
		stdin:              stdin,
		stdout:             stdout,
		stderr:             stderr,
		warmUp:             warmUpTools,
		qualityGate:        runQualityGate,
		providerOnboarding: provideronboarding.Run,
	}
	return app.run(ctx, args)
}

func (app *application) run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return app.reportError(errors.New("操作を指定してください"))
	}

	var err error
	switch args[0] {
	case "install":
		err = requireArgumentCount(args, 1)
		if err == nil {
			err = app.install(ctx)
		}
	case "uninstall":
		err = requireArgumentCount(args, 1)
		if err == nil {
			err = app.uninstall(ctx)
		}
	case "check":
		err = requireArgumentCount(args, 1)
		if err == nil {
			err = app.check(ctx)
		}
	case "pre-commit":
		err = requireArgumentCount(args, 1)
		if err == nil {
			err = app.preCommit(ctx)
		}
	case "pre-push":
		err = validatePrePushArguments(args)
		if err == nil {
			err = app.prePush(ctx)
		}
	default:
		err = fmt.Errorf("未対応の操作です: %s", args[0])
	}
	if err != nil {
		return app.reportError(err)
	}
	return 0
}

func (app *application) reportError(err error) int {
	_, _ = fmt.Fprintf(app.stderr, "Git hook の検証に失敗しました: %v\n", err)
	return exitCode(err)
}

func requireArgumentCount(args []string, count int) error {
	if len(args) != count {
		return fmt.Errorf("%s の引数数が不正です", args[0])
	}
	return nil
}

func validatePrePushArguments(args []string) error {
	if len(args) != 1 && len(args) != 3 {
		return errors.New("pre-push には引数なし、または remote name と location を指定します")
	}
	return nil
}

func findRepository(ctx context.Context) (string, error) {
	command := gitCommand(ctx, "", nil, "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return stringWithoutLineEnding(output), nil
}

func gitCommand(
	ctx context.Context,
	repository string,
	stdin io.Reader,
	args ...string,
) *exec.Cmd {
	commandArgs := args
	if repository != "" {
		commandArgs = append([]string{"-C", repository}, args...)
	}
	//nolint:gosec // SOT-ENG-021: 実行ファイルは git に固定し、引数は argv として渡して shell 解釈を行わない。
	command := exec.CommandContext(ctx, "git", commandArgs...)
	command.Env = environmentWithValue(os.Environ(), "GIT_NO_REPLACE_OBJECTS", "1")
	command.Stdin = stdin
	return command
}

func environmentWithValue(environment []string, key, value string) []string {
	result := make([]string, 0, len(environment))
	for _, item := range environment {
		name, _, _ := strings.Cut(item, "=")
		if strings.EqualFold(name, key) {
			continue
		}
		result = append(result, item)
	}
	return append(result, key+"="+value)
}

func controlledGoEnvironment(environment []string, network bool) []string {
	environment = environmentWithoutKeys(
		environment,
		"GOOS",
		"GOARCH",
		"GOEXPERIMENT",
		"GO111MODULE",
		"CGO_ENABLED",
	)
	values := map[string]string{
		"GOENV":       "off",
		"GOTOOLCHAIN": "local",
		"GOWORK":      "off",
		"GOFLAGS":     "-mod=readonly",
		"GOPROXY":     "off",
		"GOSUMDB":     "sum.golang.org",
		"GOPRIVATE":   "",
		"GONOPROXY":   "",
		"GONOSUMDB":   "",
		"GOINSECURE":  "",
		"GOVCS":       "public:git,private:off",
	}
	if network {
		values["GOPROXY"] = "https://proxy.golang.org"
	}
	result := append([]string(nil), environment...)
	for key, value := range values {
		result = environmentWithValue(result, key, value)
	}
	return result
}

func environmentWithoutKeys(environment []string, keys ...string) []string {
	result := make([]string, 0, len(environment))
	for _, item := range environment {
		name, _, _ := strings.Cut(item, "=")
		remove := false
		for _, key := range keys {
			if strings.EqualFold(name, key) {
				remove = true
				break
			}
		}
		if !remove {
			result = append(result, item)
		}
	}
	return result
}

func hasPOSIXPermissionBits() bool {
	return runtime.GOOS != "windows"
}

func stringWithoutLineEnding(value []byte) string {
	return strings.TrimSuffix(strings.TrimSuffix(string(value), "\n"), "\r")
}

func exitCode(err error) int {
	type exitCoder interface {
		ExitCode() int
	}

	var coded exitCoder
	if errors.As(err, &coded) {
		code := coded.ExitCode()
		if 0 < code && code <= 255 {
			return code
		}
	}
	return 1
}
