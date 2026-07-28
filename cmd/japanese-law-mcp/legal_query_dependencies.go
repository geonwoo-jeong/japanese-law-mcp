package main

import (
	"fmt"
	"sync"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/config"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
	coreprofile "github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/core"
)

const judicialCasesPackID = "judicial-cases"

type coreLegalQueryPlanningDependencies struct {
	preprocessor legalquery.QueryPreprocessor
	profiles     legalquery.QueryProfileSet
}

var loadCoreLegalQueryPlanningDependencies = sync.OnceValues(
	func() (coreLegalQueryPlanningDependencies, error) {
		profile, err := coreprofile.LoadEmbedded()
		if err != nil {
			return coreLegalQueryPlanningDependencies{},
				fmt.Errorf("法令コア query profile を初期化できません: %w", err)
		}
		preprocessor, err := querypreprocess.NewEmbedded(
			profile.CueVocabulary(),
		)
		if err != nil {
			return coreLegalQueryPlanningDependencies{},
				fmt.Errorf("統合照会の前処理器を初期化できません: %w", err)
		}
		profiles, err := legalquery.NewQueryProfileSet(
			[]legalquery.QueryProfile{profile},
		)
		if err != nil {
			return coreLegalQueryPlanningDependencies{},
				fmt.Errorf("法令コア query profile set を初期化できません: %w", err)
		}
		return coreLegalQueryPlanningDependencies{
			preprocessor: preprocessor,
			profiles:     profiles,
		}, nil
	},
)

func newLegalQueryService(
	cfg config.Config,
	routes application.ProviderRoutes,
) (*legalquery.Service, error) {
	planning, err := loadCoreLegalQueryPlanningDependencies()
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
		enabledPacks = []string{judicialCasesPackID}
	}
	packState, err := legalquery.NewStaticPackState(
		[]string{judicialCasesPackID},
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
