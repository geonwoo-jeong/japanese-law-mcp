package main

import (
	"errors"
	"fmt"
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitingcandidatesearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/config"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/courts/hanrei"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/courts/hanreipdf"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/egov/lawv1"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/egov/lawv2"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/ndl/kokkai"
)

func validateBuiltInProviderConfig(
	providerID string,
	provider config.ProviderConfig,
	judicialCasesEnabled bool,
	judicialCitationsEnabled bool,
	legislativeHistoryEnabled bool,
) error {
	switch providerID {
	case hanrei.Descriptor().ProviderID():
		if !judicialCasesEnabled {
			return config.NewValidationError(
				errors.New(
					"judicial-cases が無効なため courts-hanrei-html を構成できません",
				),
			)
		}
	case hanreipdf.Descriptor().ProviderID():
		if !judicialCitationsEnabled {
			return config.NewValidationError(
				errors.New(
					"judicial-citations が無効なため courts-hanrei-pdf を構成できません",
				),
			)
		}
	case kokkai.Descriptor().ProviderID():
		if !legislativeHistoryEnabled {
			return config.NewValidationError(
				errors.New(
					"legislative-history が無効なため ndl-diet-speech-api を構成できません",
				),
			)
		}
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

func validateExtensionPackProviderScope(cfg config.Config) error {
	if !cfg.JudicialCasesEnabled() {
		if _, exists := cfg.Provider(hanrei.Descriptor().ProviderID()); exists {
			return config.NewValidationError(
				errors.New(
					"judicial-cases が無効なため courts-hanrei-html を指定できません",
				),
			)
		}
		for _, key := range []config.ProviderRouteKey{
			{
				CapabilityID: judicialdecisionread.CapabilityID,
				MajorVersion: judicialdecisionread.MajorVersion,
			},
			{
				CapabilityID: judicialdecisionsearch.CapabilityID,
				MajorVersion: judicialdecisionsearch.MajorVersion,
			},
		} {
			if _, exists := cfg.ProviderRoute(key); exists {
				return config.NewValidationError(
					fmt.Errorf(
						"judicial-cases が無効なため providerRoutes.%s を指定できません",
						key,
					),
				)
			}
		}
	}
	if !cfg.JudicialCitationsEnabled() {
		if _, exists := cfg.Provider(hanreipdf.Descriptor().ProviderID()); exists {
			return config.NewValidationError(
				errors.New(
					"judicial-citations が無効なため courts-hanrei-pdf を指定できません",
				),
			)
		}
		for _, key := range []config.ProviderRouteKey{
			{
				CapabilityID: judicialcasecitationextract.CapabilityID,
				MajorVersion: judicialcasecitationextract.MajorVersion,
			},
			{
				CapabilityID: judicialcitingcandidatesearch.CapabilityID,
				MajorVersion: judicialcitingcandidatesearch.MajorVersion,
			},
		} {
			if _, exists := cfg.ProviderRoute(key); exists {
				return config.NewValidationError(
					fmt.Errorf(
						"judicial-citations が無効なため providerRoutes.%s を指定できません",
						key,
					),
				)
			}
		}
	}
	if !cfg.LegislativeHistoryEnabled() {
		if _, exists := cfg.Provider(kokkai.Descriptor().ProviderID()); exists {
			return config.NewValidationError(
				errors.New(
					"legislative-history が無効なため ndl-diet-speech-api を指定できません",
				),
			)
		}
		key := config.ProviderRouteKey{
			CapabilityID: parliamentspeechsearch.CapabilityID,
			MajorVersion: parliamentspeechsearch.MajorVersion,
		}
		if _, exists := cfg.ProviderRoute(key); exists {
			return config.NewValidationError(
				fmt.Errorf(
					"legislative-history が無効なため providerRoutes.%s を指定できません",
					key,
				),
			)
		}
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
