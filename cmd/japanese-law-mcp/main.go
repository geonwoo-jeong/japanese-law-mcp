package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/continuation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/getarticle"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/getlaw"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/listlawupdates"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlawcontent"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchlaws"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/buildinfo"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/cli"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/config"
	projectmcp "github.com/geonwoo-jeong/japanese-law-mcp/internal/mcp"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/egov/lawv1"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/egov/lawv2"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/transport/stdio"
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
		registry, routes, err := newProviderRoutes(cfg)
		if err != nil {
			return err
		}
		if err := validateCompatibilityFacadeRoutes(routes); err != nil {
			return err
		}

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
			updateLister, exists := routes.LawUpdateList()
			if !exists {
				return errors.New("primary law.update.list binding がありません")
			}
			listLawUpdates, err := listlawupdates.NewService(
				updateLister,
				cfg.RequestTimeout(),
			)
			if err != nil {
				return fmt.Errorf("公開 list_law_updates service を初期化できません: %w", err)
			}
			return runStdio(ctx, projectmcp.NewServerWithDependencies(
				version,
				projectmcp.Dependencies{
					SearchLaws:       searchLaws,
					SearchLawContent: searchLawContent,
					GetLaw:           getLaw,
					GetArticle:       getArticle,
					ListLawUpdates:   listLawUpdates,
				},
			))
		case config.TransportStreamableHTTP:
			return errStreamableHTTPNotImplemented
		default:
			return errors.New("対応していないトランスポートです")
		}
	}
}

func validateCompatibilityFacadeRoutes(routes application.ProviderRoutes) error {
	v2ProviderID := lawv2.Descriptor().ProviderID()
	for _, capability := range []struct {
		id      string
		version int
		tool    string
	}{
		{
			id:      lawcontentsearch.CapabilityID,
			version: lawcontentsearch.MajorVersion,
			tool:    "search_law_content",
		},
		{
			id:      lawsearch.CapabilityID,
			version: lawsearch.MajorVersion,
			tool:    "search_laws",
		},
	} {
		providerID, exists := routes.ProviderID(capability.id, capability.version)
		if !exists || providerID != v2ProviderID {
			return config.NewValidationError(fmt.Errorf(
				"%s は provider %q の互換 facade を必要とします",
				capability.tool,
				v2ProviderID,
			))
		}
	}
	return nil
}

func newProviderRoutes(cfg config.Config) (
	application.ProviderBindingRegistry,
	application.ProviderRoutes,
	error,
) {
	bindings, err := newEnabledProviderBindings(cfg.Providers())
	if err != nil {
		return application.ProviderBindingRegistry{},
			application.ProviderRoutes{},
			err
	}
	registry, err := application.NewProviderBindingRegistry(
		bindings,
	)
	if err != nil {
		return application.ProviderBindingRegistry{},
			application.ProviderRoutes{},
			fmt.Errorf("provider binding registry を初期化できません: %w", err)
	}
	routes, err := application.NewProviderRoutes(
		registry,
		configuredProviderRouteValues(cfg.ProviderRoutes()),
	)
	if err != nil {
		return application.ProviderBindingRegistry{},
			application.ProviderRoutes{},
			config.NewValidationError(
				fmt.Errorf("provider route を初期化できません: %w", err),
			)
	}
	return registry, routes, nil
}

func newEnabledProviderBindings(
	providers map[string]config.ProviderConfig,
) ([]application.ProviderBindings, error) {
	providerIDs := make([]string, 0, len(providers))
	for providerID := range providers {
		providerIDs = append(providerIDs, providerID)
	}
	sort.Strings(providerIDs)

	for _, providerID := range providerIDs {
		if err := validateBuiltInProviderConfig(providerID, providers[providerID]); err != nil {
			return nil, err
		}
	}

	bindings := make([]application.ProviderBindings, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		provider := providers[providerID]
		if !provider.Enabled {
			continue
		}

		var (
			binding application.ProviderBindings
			err     error
		)
		switch providerID {
		case lawv1.Descriptor().ProviderID():
			binding, err = lawv1.NewProviderBindings()
		case lawv2.Descriptor().ProviderID():
			var manager *continuation.Manager
			manager, err = continuation.NewManager()
			if err == nil {
				binding, err = lawv2.NewProviderBindings(manager)
			}
		default:
			return nil, config.NewValidationError(
				fmt.Errorf("providerId %q は組込みプロバイダーではありません", providerID),
			)
		}
		if err != nil {
			return nil, fmt.Errorf("provider %q の binding を初期化できません: %w", providerID, err)
		}
		bindings = append(bindings, binding)
	}
	return bindings, nil
}

func validateBuiltInProviderConfig(
	providerID string,
	provider config.ProviderConfig,
) error {
	switch providerID {
	case lawv1.Descriptor().ProviderID(), lawv2.Descriptor().ProviderID():
	default:
		return config.NewValidationError(
			fmt.Errorf("providerId %q は組込みプロバイダーではありません", providerID),
		)
	}
	if len(provider.Settings) != 0 {
		return config.NewValidationError(
			fmt.Errorf("provider %q は settings を受け付けません", providerID),
		)
	}
	if len(provider.CredentialEnvRefs) != 0 {
		return config.NewValidationError(
			fmt.Errorf("provider %q は credentialEnvRefs を受け付けません", providerID),
		)
	}
	return nil
}

func configuredProviderRouteValues(
	routes map[config.ProviderRouteKey]config.ProviderRoute,
) []application.ProviderRouteValues {
	keys := make([]config.ProviderRouteKey, 0, len(routes))
	for key := range routes {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(left, right int) bool {
		if keys[left].CapabilityID != keys[right].CapabilityID {
			return keys[left].CapabilityID < keys[right].CapabilityID
		}
		return keys[left].MajorVersion < keys[right].MajorVersion
	})

	values := make([]application.ProviderRouteValues, 0, len(keys))
	for _, key := range keys {
		route := routes[key]
		values = append(values, application.ProviderRouteValues{
			CapabilityID:       key.CapabilityID,
			MajorVersion:       key.MajorVersion,
			Selection:          string(route.Selection),
			DefaultProviderID:  route.DefaultProviderID,
			RollbackProviderID: route.RollbackProviderID,
		})
	}
	return values
}
