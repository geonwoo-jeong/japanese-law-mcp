package application_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestProviderRegistryRegistersDescriptorsAndDeclarations(t *testing.T) {
	t.Parallel()

	egov := newProviderDescriptor(t, providerDescriptorValues{
		providerID: "e-gov-law-api-v2",
		sourceID:   "shared-law-source",
		capabilities: []capabilityValues{
			{id: "law.document.read", majorVersion: 1},
			{id: "law.search", majorVersion: 1},
		},
	})
	secondary := newProviderDescriptor(t, providerDescriptorValues{
		providerID: "secondary-law-api",
		sourceID:   "shared-law-source",
		capabilities: []capabilityValues{
			{
				id:           "law.search",
				majorVersion: 1,
				stability:    model.CapabilityStabilityExperimental,
			},
		},
	})
	input := []model.ProviderDescriptor{secondary, egov}

	registry, err := application.NewProviderRegistry(input)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewProviderRegistry() のエラー = %v", err)
	}

	input[0] = egov
	descriptors := registry.Descriptors()
	if len(descriptors) != 2 ||
		descriptors[0].ProviderID() != "e-gov-law-api-v2" ||
		descriptors[1].ProviderID() != "secondary-law-api" {
		t.Fatalf("SOT-ARCH-012: Descriptors() = %#v", descriptors)
	}
	descriptors[0] = secondary
	if got, ok := registry.Descriptor("e-gov-law-api-v2"); !ok || got.ProviderID() != "e-gov-law-api-v2" {
		t.Fatalf("SOT-ARCH-012: Descriptor() = %#v, %t", got, ok)
	}

	capability, ok := registry.DeclaredCapability("e-gov-law-api-v2", "law.search", 1)
	if !ok ||
		capability.ID() != "law.search" ||
		capability.MajorVersion() != 1 ||
		capability.Level() != model.CapabilityLevelCore ||
		capability.Stability() != model.CapabilityStabilityStable {
		t.Fatalf("SOT-ARCH-012: DeclaredCapability() = %#v, %t", capability, ok)
	}
	secondaryCapability, ok := registry.DeclaredCapability(
		"secondary-law-api",
		"law.search",
		1,
	)
	if !ok || secondaryCapability.Stability() != model.CapabilityStabilityExperimental {
		t.Fatalf(
			"SOT-ARCH-012: secondary の DeclaredCapability() = %#v, %t",
			secondaryCapability,
			ok,
		)
	}
}

func TestProviderRegistryUsesExactBindingKey(t *testing.T) {
	t.Parallel()

	registry, err := application.NewProviderRegistry([]model.ProviderDescriptor{
		newProviderDescriptor(t, providerDescriptorValues{
			providerID: "e-gov-law-api-v2",
			sourceID:   "e-gov-law-api-v2",
			capabilities: []capabilityValues{
				{id: "law.search", majorVersion: 1},
				{id: "law.search", majorVersion: 2},
			},
		}),
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewProviderRegistry() のエラー = %v", err)
	}

	for name, query := range map[string]struct {
		providerID   string
		capabilityID string
		majorVersion int
	}{
		"未知の provider": {
			providerID:   "unknown",
			capabilityID: "law.search",
			majorVersion: 1,
		},
		"未知の capability": {
			providerID:   "e-gov-law-api-v2",
			capabilityID: "law.document.read",
			majorVersion: 1,
		},
		"未知の majorVersion": {
			providerID:   "e-gov-law-api-v2",
			capabilityID: "law.search",
			majorVersion: 3,
		},
	} {
		name, query := name, query
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, ok := registry.DeclaredCapability(query.providerID, query.capabilityID, query.majorVersion); ok {
				t.Fatalf("SOT-ARCH-012: 未登録 binding %#v を取得できた", query)
			}
		})
	}
}

func TestProviderRegistryRejectsDuplicateProviderID(t *testing.T) {
	t.Parallel()

	descriptor := newProviderDescriptor(t, providerDescriptorValues{
		providerID: "e-gov-law-api-v2",
		sourceID:   "e-gov-law-api-v2",
		capabilities: []capabilityValues{
			{id: "law.search", majorVersion: 1},
		},
	})
	if _, err := application.NewProviderRegistry([]model.ProviderDescriptor{
		descriptor,
		descriptor,
	}); err == nil {
		t.Fatal("SOT-ARCH-012: 同じ providerId の重複登録が成功した")
	}
}

func TestProviderRegistryRejectsInvalidDescriptor(t *testing.T) {
	t.Parallel()

	if _, err := application.NewProviderRegistry([]model.ProviderDescriptor{{}}); err == nil {
		t.Fatal("SOT-ARCH-012/SOT-IF-014: 無効な ProviderDescriptor の登録が成功した")
	}
}

func TestProviderRegistryAllowsNoProviders(t *testing.T) {
	t.Parallel()

	registry, err := application.NewProviderRegistry(nil)
	if err != nil {
		t.Fatalf("SOT-ARCH-012: 空の registry を作成できない: %v", err)
	}
	if registry.Descriptors() == nil || len(registry.Descriptors()) != 0 {
		t.Fatalf("SOT-ARCH-012: Descriptors() = %#v、期待値 = 空配列", registry.Descriptors())
	}
	if _, ok := registry.Descriptor("e-gov-law-api-v2"); ok {
		t.Fatal("SOT-ARCH-012: 空の registry から descriptor を取得できた")
	}
}

func TestProviderRegistrySupportsConcurrentReads(t *testing.T) {
	t.Parallel()

	registry, err := application.NewProviderRegistry([]model.ProviderDescriptor{
		newProviderDescriptor(t, providerDescriptorValues{
			providerID: "e-gov-law-api-v2",
			sourceID:   "e-gov-law-api-v2",
			capabilities: []capabilityValues{
				{id: "law.search", majorVersion: 1},
			},
		}),
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-012: NewProviderRegistry() のエラー = %v", err)
	}

	const readerCount = 32
	start := make(chan struct{})
	failures := make(chan error, readerCount)
	var waitGroup sync.WaitGroup
	waitGroup.Add(readerCount)

	for reader := 0; reader < readerCount; reader++ {
		go func() {
			defer waitGroup.Done()
			<-start

			for attempt := 0; attempt < 100; attempt++ {
				if len(registry.Descriptors()) != 1 {
					failures <- fmt.Errorf("Descriptors() の件数が一致しない")
					return
				}
				if _, ok := registry.Descriptor("e-gov-law-api-v2"); !ok {
					failures <- fmt.Errorf("Descriptor() が登録値を返さない")
					return
				}
				if _, ok := registry.DeclaredCapability(
					"e-gov-law-api-v2",
					"law.search",
					1,
				); !ok {
					failures <- fmt.Errorf("DeclaredCapability() が登録値を返さない")
					return
				}
			}
		}()
	}

	close(start)
	waitGroup.Wait()
	close(failures)
	for failure := range failures {
		t.Fatalf("SOT-ARCH-012: 並行読込みに失敗した: %v", failure)
	}
}

type providerDescriptorValues struct {
	providerID   string
	sourceID     string
	capabilities []capabilityValues
}

type capabilityValues struct {
	id           string
	majorVersion int
	stability    model.CapabilityStability
}

func newProviderDescriptor(
	t *testing.T,
	values providerDescriptorValues,
) model.ProviderDescriptor {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         values.sourceID,
		Name:       values.providerID,
		Publisher:  "公的機関",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://example.test/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できない: %v", err)
	}
	verifiedAt, err := model.NewDate("2026-07-25")
	if err != nil {
		t.Fatalf("Date を作成できない: %v", err)
	}
	capabilities := make([]model.ProviderCapability, 0, len(values.capabilities))
	for _, capabilityValues := range values.capabilities {
		stability := capabilityValues.stability
		if stability == "" {
			stability = model.CapabilityStabilityStable
		}
		capability, capabilityErr := model.NewProviderCapability(model.ProviderCapabilityValues{
			ID:           capabilityValues.id,
			MajorVersion: capabilityValues.majorVersion,
			Level:        model.CapabilityLevelCore,
			Stability:    stability,
		})
		if capabilityErr != nil {
			t.Fatalf("ProviderCapability を作成できない: %v", capabilityErr)
		}
		capabilities = append(capabilities, capability)
	}

	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             values.providerID,
		Source:                 source,
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           capabilities,
	})
	if err != nil {
		t.Fatalf("ProviderDescriptor を作成できない: %v", err)
	}
	return descriptor
}
