package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
)

const (
	ExitSuccess = 0
	ExitRuntime = 1
	ExitUsage   = 2
)

// Options は、一回の CLI 実行に必要な依存関係を保持する。
type Options struct {
	Context       context.Context
	Args          []string
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	Version       string
	UserConfigDir func() (string, error)
	Run           Runner
}

// Execute は、新しいコマンドツリーを生成して一度だけ実行する。
func Execute(options Options) int {
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	in := options.Stdin
	if in == nil {
		in = &emptyReader{}
	}
	out := options.Stdout
	if out == nil {
		out = io.Discard
	}
	errOut := options.Stderr
	if errOut == nil {
		errOut = io.Discard
	}

	command := newCommand(commandOptions{
		in:            in,
		out:           out,
		errOut:        errOut,
		version:       options.Version,
		userConfigDir: options.UserConfigDir,
		run:           options.Run,
	})
	command.SetArgs(options.Args)

	_, err := command.ExecuteContextC(ctx)
	if err == nil {
		return ExitSuccess
	}

	var classified *commandError
	if !errors.As(err, &classified) {
		classified = invalidArgumentsError().(*commandError)
	}
	_, _ = fmt.Fprintf(errOut, "エラー: %s\n", classified.message)
	return classified.code
}

type emptyReader struct{}

func (*emptyReader) Read([]byte) (int, error) {
	return 0, io.EOF
}
