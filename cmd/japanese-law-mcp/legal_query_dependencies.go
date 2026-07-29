package main

import (
	"fmt"
	"sync"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/config"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"
)

type legalQueryPlanningDependencies struct {
	preprocessor legalquery.QueryPreprocessor
	profiles     legalquery.QueryProfileSet
}

// 意味認識用の profile と cue は pack 状態より先に固定し、実行 facade だけを pack 有効時に構成する。
var loadLegalQueryPlanningDependencies = sync.OnceValues(
	func() (legalQueryPlanningDependencies, error) {
		planning, err := legalqueryplanning.LoadEmbedded()
		if err != nil {
			return legalQueryPlanningDependencies{},
				fmt.Errorf("統合照会 planning 依存を初期化できません: %w", err)
		}
		return legalQueryPlanningDependencies{
			preprocessor: planning.Preprocessor(),
			profiles:     planning.Profiles(),
		}, nil
	},
)

func newLegalQueryService(
	cfg config.Config,
	routes application.ProviderRoutes,
) (*legalquery.Service, error) {
	planning, err := loadLegalQueryPlanningDependencies()
	if err != nil {
		return nil, err
	}
	coreFacade, err := application.NewCoreLegalQueryFacade(
		routes,
		legalquery.NewCoreMaterializer(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"統合照会の法令コア facade を初期化できません: %w",
			err,
		)
	}
	judicialFacade, err := newJudicialCasesLegalQueryFacade(cfg, routes)
	if err != nil {
		return nil, err
	}
	executor, err := legalquery.NewExecutor(coreFacade, judicialFacade)
	if err != nil {
		return nil, fmt.Errorf("統合照会の executor を初期化できません: %w", err)
	}
	enabledPacks := []string(nil)
	if cfg.JudicialCasesEnabled() {
		enabledPacks = []string{legalqueryplanning.JudicialCasesPackID}
	}
	packState, err := legalquery.NewStaticPackState(
		[]string{legalqueryplanning.JudicialCasesPackID},
		enabledPacks,
	)
	if err != nil {
		return nil, fmt.Errorf("統合照会の pack 状態を初期化できません: %w", err)
	}
	service, err := legalquery.NewService(
		planning.preprocessor,
		planning.profiles,
		packState,
		executor,
		cfg.RequestTimeout(),
	)
	if err != nil {
		return nil, fmt.Errorf("統合照会 service を初期化できません: %w", err)
	}
	return service, nil
}

func newJudicialCasesLegalQueryFacade(
	cfg config.Config,
	routes application.ProviderRoutes,
) (legalquery.JudicialCasesCapabilityFacade, error) {
	if !cfg.JudicialCasesEnabled() {
		return nil, nil
	}
	facade, err := application.NewJudicialCasesLegalQueryFacade(
		routes,
		legalquery.NewJudicialCasesMaterializer(),
	)
	if err != nil {
		return nil, config.NewValidationError(fmt.Errorf(
			"統合照会の裁判例 facade を初期化できません: %w",
			err,
		))
	}
	return facade, nil
}
