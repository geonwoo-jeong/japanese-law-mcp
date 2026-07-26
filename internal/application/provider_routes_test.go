package application_test

import (
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawupdatelist"
)

func TestProviderRoutesResolvePrimaryAndRollbackTypedPorts(t *testing.T) {
	t.Parallel()

	primary := newCompleteProviderBindings(t, "primary-provider")
	rollback := newCompleteProviderBindings(t, "rollback-provider")
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{primary, rollback},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: registry のエラー = %v", err)
	}
	values := completeProviderRouteValues("primary-provider")
	values[3].RollbackProviderID = "rollback-provider"

	routes, err := application.NewProviderRoutes(registry, values)
	if err != nil {
		t.Fatalf("SOT-ARCH-012/SOT-IF-026: NewProviderRoutes() のエラー = %v", err)
	}
	if providerID, exists := routes.ProviderID(
		lawsearch.CapabilityID,
		lawsearch.MajorVersion,
	); !exists || providerID != "rollback-provider" {
		t.Fatalf("SOT-IF-026: law.search providerId = %q, %t", providerID, exists)
	}
	if providerID, exists := routes.ProviderID(
		lawdocumentread.CapabilityID,
		lawdocumentread.MajorVersion,
	); !exists || providerID != "primary-provider" {
		t.Fatalf("SOT-IF-026: law.document.read providerId = %q, %t", providerID, exists)
	}

	if port, exists := routes.LawSearch(); !exists || port != rollback.LawSearch {
		t.Fatalf("SOT-ARCH-012: LawSearch() = %#v, %t", port, exists)
	}
	if port, exists := routes.LawContentSearch(); !exists ||
		port != primary.LawContentSearch {
		t.Fatalf("SOT-ARCH-012: LawContentSearch() = %#v, %t", port, exists)
	}
	if port, exists := routes.LawDocumentRead(); !exists ||
		port != primary.LawDocumentRead {
		t.Fatalf("SOT-ARCH-012: LawDocumentRead() = %#v, %t", port, exists)
	}
	if port, exists := routes.LawArticleRead(); !exists ||
		port != primary.LawArticleRead {
		t.Fatalf("SOT-ARCH-012: LawArticleRead() = %#v, %t", port, exists)
	}
	if port, exists := routes.LawUpdateList(); !exists ||
		port != primary.LawUpdateList {
		t.Fatalf("SOT-ARCH-012: LawUpdateList() = %#v, %t", port, exists)
	}
}

func TestProviderRoutesRejectInvalidRoutes(t *testing.T) {
	t.Parallel()

	complete := newCompleteProviderBindings(t, "primary-provider")
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{complete},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: registry のエラー = %v", err)
	}

	tests := map[string]func([]application.ProviderRouteValues) []application.ProviderRouteValues{
		"必須 route の欠落": func(values []application.ProviderRouteValues) []application.ProviderRouteValues {
			return values[1:]
		},
		"route の重複": func(values []application.ProviderRouteValues) []application.ProviderRouteValues {
			return append(values, values[0])
		},
		"aggregate": func(values []application.ProviderRouteValues) []application.ProviderRouteValues {
			values[0].Selection = "aggregate"
			return values
		},
		"未知の capability": func(values []application.ProviderRouteValues) []application.ProviderRouteValues {
			values[0].CapabilityID = "law.revision.list"
			return values
		},
		"未対応 majorVersion": func(values []application.ProviderRouteValues) []application.ProviderRouteValues {
			values[0].MajorVersion = 2
			return values
		},
		"default binding の欠落": func(values []application.ProviderRouteValues) []application.ProviderRouteValues {
			values[0].DefaultProviderID = "missing-provider"
			return values
		},
		"rollback binding の欠落": func(values []application.ProviderRouteValues) []application.ProviderRouteValues {
			values[0].RollbackProviderID = "missing-provider"
			return values
		},
	}
	for name, change := range tests {
		name, change := name, change
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			values := completeProviderRouteValues("primary-provider")
			if _, routeErr := application.NewProviderRoutes(
				registry,
				change(values),
			); routeErr == nil {
				t.Fatalf("SOT-ARCH-012/SOT-IF-026: %s を受理した", name)
			}
		})
	}
}

func completeProviderRouteValues(
	providerID string,
) []application.ProviderRouteValues {
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
		{
			CapabilityID:      lawupdatelist.CapabilityID,
			MajorVersion:      lawupdatelist.MajorVersion,
			Selection:         application.ProviderRouteSelectionPrimary,
			DefaultProviderID: providerID,
		},
	}
}
