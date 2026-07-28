package application_test

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

var _ legalquery.SelectedCapabilityBinding = application.ProviderBindingMetadata{}

func TestProviderRoutesPrimaryBindingMetadataUsesEffectiveRollback(t *testing.T) {
	t.Parallel()

	primary := providerBindingsWithSource(
		t,
		newCompleteProviderBindings(t, "primary-provider"),
		"primary-source",
	)
	rollback := providerBindingsWithSource(
		t,
		newCompleteProviderBindings(t, "rollback-provider"),
		"rollback-source",
	)
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{primary, rollback},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: registry のエラー = %v", err)
	}
	values := completeProviderRouteValues("primary-provider")
	for index := range values {
		if values[index].CapabilityID == lawsearch.CapabilityID &&
			values[index].MajorVersion == lawsearch.MajorVersion {
			values[index].RollbackProviderID = "rollback-provider"
		}
	}
	routes, err := application.NewProviderRoutes(registry, values)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: NewProviderRoutes() のエラー = %v", err)
	}

	metadata, exists := routes.PrimaryBindingMetadata(
		lawsearch.CapabilityID,
		lawsearch.MajorVersion,
	)
	if !exists {
		t.Fatal("SOT-ARCH-026: 実効 rollback binding metadata がありません")
	}
	assertProviderBindingMetadata(
		t,
		metadata,
		"rollback-provider",
		"rollback-source",
		lawsearch.CapabilityID,
		lawsearch.MajorVersion,
	)
}

func TestProviderRoutesExplicitBindingMetadataIgnoresPrimaryRoute(t *testing.T) {
	t.Parallel()

	primary := providerBindingsWithSource(
		t,
		newCompleteProviderBindings(t, "primary-provider"),
		"primary-source",
	)
	explicit := providerBindingsWithSource(
		t,
		newCompleteProviderBindings(t, "explicit-provider"),
		"explicit-source",
	)
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{primary, explicit},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: registry のエラー = %v", err)
	}
	routes, err := application.NewProviderRoutes(
		registry,
		completeProviderRouteValues("primary-provider"),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: NewProviderRoutes() のエラー = %v", err)
	}

	primaryMetadata, primaryExists := routes.PrimaryBindingMetadata(
		lawdocumentread.CapabilityID,
		lawdocumentread.MajorVersion,
	)
	if !primaryExists {
		t.Fatal("SOT-ARCH-026: primary binding metadata がありません")
	}
	assertProviderBindingMetadata(
		t,
		primaryMetadata,
		"primary-provider",
		"primary-source",
		lawdocumentread.CapabilityID,
		lawdocumentread.MajorVersion,
	)

	explicitMetadata, explicitExists := routes.ExplicitBindingMetadata(
		"explicit-provider",
		lawdocumentread.CapabilityID,
		lawdocumentread.MajorVersion,
	)
	if !explicitExists {
		t.Fatal("SOT-ARCH-026: explicit binding metadata がありません")
	}
	assertProviderBindingMetadata(
		t,
		explicitMetadata,
		"explicit-provider",
		"explicit-source",
		lawdocumentread.CapabilityID,
		lawdocumentread.MajorVersion,
	)

	unroutedExplicitMetadata, unroutedExplicitExists := routes.ExplicitBindingMetadata(
		"explicit-provider",
		judicialdecisionsearch.CapabilityID,
		judicialdecisionsearch.MajorVersion,
	)
	if !unroutedExplicitExists {
		t.Fatal("SOT-ARCH-026: primary route がない explicit binding metadata がありません")
	}
	assertProviderBindingMetadata(
		t,
		unroutedExplicitMetadata,
		"explicit-provider",
		"explicit-source",
		judicialdecisionsearch.CapabilityID,
		judicialdecisionsearch.MajorVersion,
	)
}

func TestProviderRoutesBindingMetadataRejectsUnknownOrMissingSelection(t *testing.T) {
	t.Parallel()

	bindings := providerBindingsWithSource(
		t,
		newCompleteProviderBindings(t, "registered-provider"),
		"registered-source",
	)
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: registry のエラー = %v", err)
	}
	routes, err := application.NewProviderRoutes(
		registry,
		completeProviderRouteValues("registered-provider"),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: NewProviderRoutes() のエラー = %v", err)
	}

	tests := map[string]func() bool{
		"primary の未知 capability": func() bool {
			_, exists := routes.PrimaryBindingMetadata("law.unknown", 1)
			return exists
		},
		"primary の未対応 majorVersion": func() bool {
			_, exists := routes.PrimaryBindingMetadata(
				lawsearch.CapabilityID,
				lawsearch.MajorVersion+1,
			)
			return exists
		},
		"primary route の欠落": func() bool {
			_, exists := routes.PrimaryBindingMetadata(
				judicialdecisionsearch.CapabilityID,
				judicialdecisionsearch.MajorVersion,
			)
			return exists
		},
		"explicit の未知 provider": func() bool {
			_, exists := routes.ExplicitBindingMetadata(
				"unknown-provider",
				lawsearch.CapabilityID,
				lawsearch.MajorVersion,
			)
			return exists
		},
		"explicit の未知 capability": func() bool {
			_, exists := routes.ExplicitBindingMetadata(
				"registered-provider",
				"law.unknown",
				1,
			)
			return exists
		},
		"explicit の未対応 majorVersion": func() bool {
			_, exists := routes.ExplicitBindingMetadata(
				"registered-provider",
				lawsearch.CapabilityID,
				lawsearch.MajorVersion+1,
			)
			return exists
		},
	}
	for name, selectInvalid := range tests {
		name, selectInvalid := name, selectInvalid
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if selectInvalid() {
				t.Fatalf("SOT-ARCH-026: %s から metadata を作成した", name)
			}
		})
	}
}

func TestProviderRoutesBindingMetadataRejectsUninitializedRoutes(t *testing.T) {
	t.Parallel()

	var routes application.ProviderRoutes
	if _, exists := routes.PrimaryBindingMetadata(
		lawsearch.CapabilityID,
		lawsearch.MajorVersion,
	); exists {
		t.Fatal("SOT-ARCH-026: 未初期化 route から primary metadata を作成した")
	}
	if _, exists := routes.ExplicitBindingMetadata(
		"registered-provider",
		lawsearch.CapabilityID,
		lawsearch.MajorVersion,
	); exists {
		t.Fatal("SOT-ARCH-026: 未初期化 route から explicit metadata を作成した")
	}
}

func TestProviderRoutesBindingMetadataCannotBypassTypedPortValidation(t *testing.T) {
	t.Parallel()

	bindings := newCompleteProviderBindings(t, "invalid-provider")
	var typedNil *fakeLawSearchBinding
	bindings.LawSearch = typedNil

	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err == nil {
		t.Fatal("SOT-ARCH-026: typed nil port を持つ registry を受理した")
	}
	if _, routeErr := application.NewProviderRoutes(
		registry,
		completeProviderRouteValues("invalid-provider"),
	); routeErr == nil {
		t.Fatal("SOT-ARCH-026: 検証に失敗した registry から route を作成した")
	}
}

func providerBindingsWithSource(
	t *testing.T,
	bindings application.ProviderBindings,
	sourceID string,
) application.ProviderBindings {
	t.Helper()

	capabilities := bindings.Descriptor.Capabilities()
	values := make([]capabilityValues, len(capabilities))
	for index, capability := range capabilities {
		values[index] = capabilityValues{
			id:           capability.ID(),
			majorVersion: capability.MajorVersion(),
			stability:    capability.Stability(),
		}
	}
	bindings.Descriptor = newProviderDescriptor(t, providerDescriptorValues{
		providerID:   bindings.Descriptor.ProviderID(),
		sourceID:     sourceID,
		capabilities: values,
	})
	return bindings
}

func assertProviderBindingMetadata(
	t *testing.T,
	metadata application.ProviderBindingMetadata,
	providerID string,
	sourceID string,
	capabilityID string,
	capabilityMajorVersion int,
) {
	t.Helper()

	if metadata.ProviderID() != providerID ||
		metadata.SourceID() != sourceID ||
		metadata.CapabilityID() != capabilityID ||
		metadata.CapabilityMajorVersion() != capabilityMajorVersion {
		t.Fatalf(
			"SOT-ARCH-026: binding metadata = (%q, %q, %q, %d)、期待値 = (%q, %q, %q, %d)",
			metadata.ProviderID(),
			metadata.SourceID(),
			metadata.CapabilityID(),
			metadata.CapabilityMajorVersion(),
			providerID,
			sourceID,
			capabilityID,
			capabilityMajorVersion,
		)
	}
}
