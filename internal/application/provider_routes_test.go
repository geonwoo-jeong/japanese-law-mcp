package application_test

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
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

func TestProviderRoutesResolveOptionalJudicialDecisionRoutes(t *testing.T) {
	t.Parallel()

	bindings := newCompleteProviderBindings(t, "judicial-provider")
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: registry のエラー = %v", err)
	}
	values := append(
		completeProviderRouteValues("judicial-provider"),
		application.ProviderRouteValues{
			CapabilityID:      judicialdecisionread.CapabilityID,
			MajorVersion:      judicialdecisionread.MajorVersion,
			Selection:         application.ProviderRouteSelectionPrimary,
			DefaultProviderID: "judicial-provider",
		},
		application.ProviderRouteValues{
			CapabilityID:      judicialdecisionsearch.CapabilityID,
			MajorVersion:      judicialdecisionsearch.MajorVersion,
			Selection:         application.ProviderRouteSelectionPrimary,
			DefaultProviderID: "judicial-provider",
		},
	)

	routes, routeErr := application.NewProviderRoutes(registry, values)
	if routeErr != nil {
		t.Fatalf("SOT-IF-041/SOT-IF-042: NewProviderRoutes() のエラー = %v", routeErr)
	}
	if port, exists := routes.JudicialDecisionSearch(); !exists ||
		port != bindings.JudicialDecisionSearch {
		t.Fatalf("SOT-IF-041: JudicialDecisionSearch() = %#v, %t", port, exists)
	}
	if port, exists := routes.JudicialDecisionRead(); !exists ||
		port != bindings.JudicialDecisionRead {
		t.Fatalf("SOT-IF-042: JudicialDecisionRead() = %#v, %t", port, exists)
	}
}

func TestProviderRoutesKeepJudicialDecisionRoutesOptional(t *testing.T) {
	t.Parallel()

	bindings := newCompleteProviderBindings(t, "core-provider")
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: registry のエラー = %v", err)
	}
	routes, routeErr := application.NewProviderRoutes(
		registry,
		completeProviderRouteValues("core-provider"),
	)
	if routeErr != nil {
		t.Fatalf("SOT-ARCH-019: 裁判例 route を必須として扱った: %v", routeErr)
	}
	if _, exists := routes.JudicialDecisionSearch(); exists {
		t.Fatal("SOT-ARCH-019: 未設定の judicial-decision.search route を返した")
	}
	if _, exists := routes.JudicialDecisionRead(); exists {
		t.Fatal("SOT-ARCH-019: 未設定の judicial-decision.read route を返した")
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
		"公開済み更新一覧 route の欠落": func(values []application.ProviderRouteValues) []application.ProviderRouteValues {
			return values[:len(values)-1]
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
