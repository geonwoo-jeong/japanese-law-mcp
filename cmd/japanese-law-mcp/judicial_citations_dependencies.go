package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationnormalize"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationtrace"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/config"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	projectmcp "github.com/geonwoo-jeong/japanese-law-mcp/internal/mcp"
)

var loadJudicialCitationLawAliasResolver = sync.OnceValues(
	func() (judicialcitationnormalize.ExactLawAliasResolver, error) {
		lexicon, err := lawnamelexicon.LoadEmbedded()
		if err != nil {
			return judicialcitationnormalize.ExactLawAliasResolver{}, err
		}
		return judicialcitationnormalize.NewExactLawAliasResolver(lexicon.Entries())
	},
)

func newJudicialCitationLawAliasResolver() (
	judicialcitationnormalize.ExactLawAliasResolver,
	error,
) {
	return loadJudicialCitationLawAliasResolver()
}

func newJudicialCitationsDependencies(
	cfg config.Config,
	registry application.ProviderBindingRegistry,
	routes application.ProviderRoutes,
) (projectmcp.JudicialCitationsDependencies, error) {
	if !cfg.JudicialCitationsEnabled() {
		return projectmcp.JudicialCitationsDependencies{}, nil
	}
	detailReader, err := newJudicialCitationDetailReader(cfg, registry, routes)
	if err != nil {
		return projectmcp.JudicialCitationsDependencies{}, err
	}
	extractor, exists := routes.JudicialCaseCitationExtract()
	if !exists {
		return projectmcp.JudicialCitationsDependencies{},
			config.NewValidationError(
				errors.New("primary judicial-decision.case-citation.extract binding がありません"),
			)
	}
	candidateSearcher, exists := routes.JudicialCitingCandidateSearch()
	if !exists {
		return projectmcp.JudicialCitationsDependencies{},
			config.NewValidationError(
				errors.New("primary judicial-decision.citing-candidate.search binding がありません"),
			)
	}
	resolver, err := newJudicialCitationLawAliasResolver()
	if err != nil {
		return projectmcp.JudicialCitationsDependencies{},
			fmt.Errorf("判例引用追跡の法令別名解決器を初期化できません: %w", err)
	}
	service, err := judicialcitationtrace.NewService(
		detailReader,
		extractor,
		candidateSearcher,
		resolver,
	)
	if err != nil {
		return projectmcp.JudicialCitationsDependencies{},
			config.NewValidationError(fmt.Errorf(
				"judicial-citations の公開依存関係を初期化できません: %w",
				err,
			))
	}
	dependencies, err := projectmcp.NewJudicialCitationsDependencies(service)
	if err != nil {
		return projectmcp.JudicialCitationsDependencies{},
			config.NewValidationError(fmt.Errorf(
				"judicial-citations の公開依存関係を初期化できません: %w",
				err,
			))
	}
	return dependencies, nil
}

func newJudicialCitationDetailReader(
	cfg config.Config,
	registry application.ProviderBindingRegistry,
	routes application.ProviderRoutes,
) (judicialdecisionread.Port, error) {
	readProvider, exists := routes.JudicialDecisionRead()
	if !exists {
		return nil, config.NewValidationError(
			errors.New("primary judicial-decision.read binding がありません"),
		)
	}
	providerID, exists := routes.ProviderID(
		judicialdecisionread.CapabilityID,
		judicialdecisionread.MajorVersion,
	)
	if !exists {
		return nil, config.NewValidationError(
			errors.New("primary judicial-decision.read provider がありません"),
		)
	}
	descriptor, exists := registry.Descriptor(providerID)
	if !exists {
		return nil, config.NewValidationError(
			errors.New("primary judicial-decision.read descriptor がありません"),
		)
	}
	resolver, err := judicialdecisionread.NewResolver(
		[]judicialdecisionread.ProviderBinding{{
			Descriptor: descriptor,
			Enabled:    true,
			Port:       readProvider,
		}},
	)
	if err != nil {
		return nil, config.NewValidationError(fmt.Errorf(
			"判例引用追跡の judicial-decision.read resolver を初期化できません: %w",
			err,
		))
	}
	service, err := judicialdecisionread.NewService(resolver, cfg.RequestTimeout())
	if err != nil {
		return nil, config.NewValidationError(fmt.Errorf(
			"判例引用追跡の judicial-decision.read service を初期化できません: %w",
			err,
		))
	}
	return service, nil
}
