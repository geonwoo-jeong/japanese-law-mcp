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
)

var errServerNotImplemented = errors.New("サーバー機能はまだ実装されていません")

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
		Run:           runServer,
	})
	os.Exit(code)
}

func runServer(context.Context, config.Config) error {
	return errServerNotImplemented
}
