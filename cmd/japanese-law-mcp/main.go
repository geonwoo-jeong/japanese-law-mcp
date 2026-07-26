package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/getlaw"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/searchlaws"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/buildinfo"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/cli"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/config"
	projectmcp "github.com/japanese-law-mcp/japanese-law-mcp/internal/mcp"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/source/egov/lawv2"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/transport/stdio"
)

var errStreamableHTTPNotImplemented = errors.New("トランスポート Streamable HTTP はまだ実装されていません")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx)
	stop()
	os.Exit(code)
}

func run(ctx context.Context) int {
	return cli.Execute(cli.Options{ //nolint:contextcheck // SOT-ENG-010: ctx は Options.Context を介して Cobra の ExecuteContextC へ渡す。
		Context:       ctx,
		Args:          os.Args[1:],
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		Version:       buildinfo.Version(),
		UserConfigDir: os.UserConfigDir,
		Run:           newServerRunner(buildinfo.Version(), stdio.Run),
	})
}

type stdioRunner func(context.Context, stdio.Server) error

func newServerRunner(version string, runStdio stdioRunner) cli.Runner {
	return func(ctx context.Context, cfg config.Config) error {
		switch cfg.Transport() {
		case config.TransportStdio:
			provider, err := lawv2.NewSearchLawsFacade()
			if err != nil {
				return fmt.Errorf("公開 search_laws facade を初期化できません: %w", err)
			}
			searchLaws, err := searchlaws.NewService(
				provider,
				cfg.RequestTimeout(),
			)
			if err != nil {
				return fmt.Errorf("公開 search_laws service を初期化できません: %w", err)
			}
			documentReader, err := lawv2.NewLawDocumentAdapter()
			if err != nil {
				return fmt.Errorf("e-Gov law.document.read adapter を初期化できません: %w", err)
			}
			getLaw, err := getlaw.NewService(
				documentReader,
				lawv2.Descriptor(),
				cfg.RequestTimeout(),
			)
			if err != nil {
				return fmt.Errorf("公開 get_law service を初期化できません: %w", err)
			}
			return runStdio(ctx, projectmcp.NewServerWithDependencies(
				version,
				projectmcp.Dependencies{
					SearchLaws: searchLaws,
					GetLaw:     getLaw,
				},
			))
		case config.TransportStreamableHTTP:
			return errStreamableHTTPNotImplemented
		default:
			return errors.New("対応していないトランスポートです")
		}
	}
}
