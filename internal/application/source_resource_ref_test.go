package application_test

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestProviderRegistryResolvesDescriptorForSourceResourceRef(t *testing.T) {
	t.Parallel()

	descriptor := newProviderDescriptor(t, providerDescriptorValues{
		providerID: "law-api-adapter-v2",
		sourceID:   "official-law-source",
		capabilities: []capabilityValues{
			{id: "law.document.read", majorVersion: 1},
		},
	})
	registry, err := application.NewProviderRegistry([]model.ProviderDescriptor{descriptor})
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewProviderRegistry() のエラー = %v", err)
	}
	ref := newSourceResourceRef(t, "law-api-adapter-v2", "official-law-source")

	got, err := registry.DescriptorForResourceRef(ref)
	if err != nil {
		t.Fatalf("SOT-MODEL-016/SOT-ARCH-012: DescriptorForResourceRef() のエラー = %v", err)
	}
	if got.ProviderID() != descriptor.ProviderID() ||
		got.Source().ID() != descriptor.Source().ID() {
		t.Fatalf("SOT-ARCH-012: DescriptorForResourceRef() = %#v、期待値 = %#v", got, descriptor)
	}
}

func TestProviderRegistryRejectsUnknownProviderInSourceResourceRef(t *testing.T) {
	t.Parallel()

	registry, err := application.NewProviderRegistry([]model.ProviderDescriptor{
		newProviderDescriptor(t, providerDescriptorValues{
			providerID: "law-api-adapter-v2",
			sourceID:   "official-law-source",
			capabilities: []capabilityValues{
				{id: "law.document.read", majorVersion: 1},
			},
		}),
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewProviderRegistry() のエラー = %v", err)
	}
	ref := newSourceResourceRef(t, "unknown-law-api", "official-law-source")

	if _, err := registry.DescriptorForResourceRef(ref); err == nil {
		t.Fatal("SOT-MODEL-016/SOT-ARCH-012: 未登録 provider の参照を解決できた")
	}
}

func TestProviderRegistryRejectsSourceMismatchInSourceResourceRef(t *testing.T) {
	t.Parallel()

	registry, err := application.NewProviderRegistry([]model.ProviderDescriptor{
		newProviderDescriptor(t, providerDescriptorValues{
			providerID: "law-api-adapter-v2",
			sourceID:   "official-law-source",
			capabilities: []capabilityValues{
				{id: "law.document.read", majorVersion: 1},
			},
		}),
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewProviderRegistry() のエラー = %v", err)
	}
	ref := newSourceResourceRef(t, "law-api-adapter-v2", "other-law-source")

	if _, err := registry.DescriptorForResourceRef(ref); err == nil {
		t.Fatal("SOT-MODEL-016/SOT-ARCH-012: provider と source が一致しない参照を解決できた")
	}
}

func TestProviderRegistryRejectsInvalidSourceResourceRef(t *testing.T) {
	t.Parallel()

	registry, err := application.NewProviderRegistry(nil)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewProviderRegistry() のエラー = %v", err)
	}

	if _, err := registry.DescriptorForResourceRef(model.SourceResourceRef{}); err == nil {
		t.Fatal("SOT-MODEL-016/SOT-ARCH-012: 無効な参照を解決できた")
	}
}

func newSourceResourceRef(
	t *testing.T,
	providerID string,
	sourceID string,
) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "law",
		ResourceID:   "001",
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	return ref
}
