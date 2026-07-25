package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/buildinfo"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/cli"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/config"
	projectmcp "github.com/japanese-law-mcp/japanese-law-mcp/internal/mcp"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/transport/stdio"
)

var errStreamableHTTPNotImplemented = errors.New("Streamable HTTP はまだ実装されていません")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	code := cli.Execute(cli.Options{
		Context:       ctx,
		Args:          os.Args[1:],
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		Version:       buildinfo.Version(),
		UserConfigDir: os.UserConfigDir,
		Run:           newServerRunner(buildinfo.Version(), stdio.Run),
	})
	os.Exit(code)
}

type stdioRunner func(context.Context, stdio.Server) error

func newServerRunner(version string, runStdio stdioRunner) cli.Runner {
	return func(ctx context.Context, cfg config.Config) error {
		switch cfg.Transport() {
		case config.TransportStdio:
			return runStdio(ctx, projectmcp.NewServer(version))
		case config.TransportStreamableHTTP:
			return errStreamableHTTPNotImplemented
		default:
			return errors.New("対応していないトランスポートです")
		}
	}
}
