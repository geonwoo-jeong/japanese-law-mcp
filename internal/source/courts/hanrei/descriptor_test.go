package hanrei

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestDescriptorMatchesSOTIF043(t *testing.T) {
	t.Parallel()
	descriptor := Descriptor()
	if descriptor.ProviderID() != providerID ||
		descriptor.Source().ID() != sourceID ||
		descriptor.Source().Name() != "裁判所 裁判例検索" ||
		descriptor.Source().Publisher() != "最高裁判所" ||
		descriptor.Source().ServiceURL() != searchEndpoint ||
		descriptor.AdapterContractVersion() != "1.0.0" ||
		descriptor.VerifiedAt().String() != "2026-07-26" ||
		descriptor.InterfaceType() != model.InterfaceTypeHTML ||
		descriptor.CredentialRequired() {
		t.Errorf("SOT-IF-043: descriptor の固定値が一致しない")
	}
	capabilities := descriptor.Capabilities()
	if len(capabilities) != 2 ||
		capabilities[0].ID() != "judicial-decision.read" ||
		capabilities[1].ID() != "judicial-decision.search" {
		t.Fatalf("SOT-IF-043: capabilities = %#v", capabilities)
	}
	for _, capability := range capabilities {
		if capability.MajorVersion() != 1 ||
			capability.Level() != model.CapabilityLevelExtended ||
			capability.Stability() != model.CapabilityStabilityStable {
			t.Errorf("SOT-IF-043: capability = %#v", capability)
		}
	}
}
