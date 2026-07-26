package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/continuation"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/getarticle"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/getlaw"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/searchlawcontent"
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
			registry, routes, err := newDefaultProviderRoutes()
			if err != nil {
				return err
			}
			if _, exists := routes.LawSearch(); !exists {
				return errors.New("primary law.search binding がありません")
			}
			if _, exists := routes.LawContentSearch(); !exists {
				return errors.New("primary law.content.search binding がありません")
			}
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
			searchLawContentProvider, err := lawv2.NewSearchLawContentFacade()
			if err != nil {
				return fmt.Errorf("公開 search_law_content facade を初期化できません: %w", err)
			}
			searchLawContent, err := searchlawcontent.NewService(
				searchLawContentProvider,
				cfg.RequestTimeout(),
			)
			if err != nil {
				return fmt.Errorf("公開 search_law_content service を初期化できません: %w", err)
			}
			documentReader, exists := routes.LawDocumentRead()
			if !exists {
				return errors.New("primary law.document.read binding がありません")
			}
			documentProviderID, exists := routes.ProviderID(
				lawdocumentread.CapabilityID,
				lawdocumentread.MajorVersion,
			)
			if !exists {
				return errors.New("primary law.document.read provider がありません")
			}
			documentProvider, exists := registry.Descriptor(documentProviderID)
			if !exists {
				return errors.New("primary law.document.read descriptor がありません")
			}
			getLaw, err := getlaw.NewService(
				documentReader,
				documentProvider,
				cfg.RequestTimeout(),
			)
			if err != nil {
				return fmt.Errorf("公開 get_law service を初期化できません: %w", err)
			}
			articleReader, exists := routes.LawArticleRead()
			if !exists {
				return errors.New("primary law.article.read binding がありません")
			}
			articleProviderID, exists := routes.ProviderID(
				lawarticleread.CapabilityID,
				lawarticleread.MajorVersion,
			)
			if !exists {
				return errors.New("primary law.article.read provider がありません")
			}
			articleProvider, exists := registry.Descriptor(articleProviderID)
			if !exists {
				return errors.New("primary law.article.read descriptor がありません")
			}
			getArticle, err := getarticle.NewService(
				articleReader,
				articleProvider,
				cfg.RequestTimeout(),
			)
			if err != nil {
				return fmt.Errorf("公開 get_article service を初期化できません: %w", err)
			}
			return runStdio(ctx, projectmcp.NewServerWithDependencies(
				version,
				projectmcp.Dependencies{
					SearchLaws:       searchLaws,
					SearchLawContent: searchLawContent,
					GetLaw:           getLaw,
					GetArticle:       getArticle,
				},
			))
		case config.TransportStreamableHTTP:
			return errStreamableHTTPNotImplemented
		default:
			return errors.New("対応していないトランスポートです")
		}
	}
}

func newDefaultProviderRoutes() (
	application.ProviderBindingRegistry,
	application.ProviderRoutes,
	error,
) {
	manager, err := continuation.NewManager()
	if err != nil {
		return application.ProviderBindingRegistry{},
			application.ProviderRoutes{},
			fmt.Errorf("continuation manager を初期化できません: %w", err)
	}
	bindings, err := lawv2.NewProviderBindings(manager)
	if err != nil {
		return application.ProviderBindingRegistry{},
			application.ProviderRoutes{},
			fmt.Errorf("e-Gov provider binding を初期化できません: %w", err)
	}
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err != nil {
		return application.ProviderBindingRegistry{},
			application.ProviderRoutes{},
			fmt.Errorf("provider binding registry を初期化できません: %w", err)
	}
	routes, err := application.NewProviderRoutes(
		registry,
		defaultProviderRouteValues(),
	)
	if err != nil {
		return application.ProviderBindingRegistry{},
			application.ProviderRoutes{},
			fmt.Errorf("provider route を初期化できません: %w", err)
	}
	return registry, routes, nil
}

func defaultProviderRouteValues() []application.ProviderRouteValues {
	const providerID = "e-gov-law-api-v2"
	return []application.ProviderRouteValues{
		{
			CapabilityID:      lawarticleread.CapabilityID,
			MajorVersion:      lawarticleread.MajorVersion,
			Selection:         application.ProviderRouteSelectionPrimary,
			DefaultProviderID: providerID,
		},
		{
			CapabilityID:      lawcontentsearch.CapabilityID,
			MajorVersion:      lawcontentsearch.MajorVersion,
			Selection:         application.ProviderRouteSelectionPrimary,
			DefaultProviderID: providerID,
		},
		{
			CapabilityID:      lawdocumentread.CapabilityID,
			MajorVersion:      lawdocumentread.MajorVersion,
			Selection:         application.ProviderRouteSelectionPrimary,
			DefaultProviderID: providerID,
		},
		{
			CapabilityID:      lawsearch.CapabilityID,
			MajorVersion:      lawsearch.MajorVersion,
			Selection:         application.ProviderRouteSelectionPrimary,
			DefaultProviderID: providerID,
		},
	}
}
