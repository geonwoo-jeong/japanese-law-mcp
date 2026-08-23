package kokkai

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestDescriptorMatchesSOTIF063(t *testing.T) {
	t.Parallel()

	descriptor := Descriptor()
	if descriptor.ProviderID() != providerID ||
		descriptor.Source().ID() != sourceID ||
		descriptor.Source().Name() != "国会会議録検索システム" ||
		descriptor.Source().Publisher() != "国立国会図書館" ||
		descriptor.Source().ServiceURL() != serviceURL ||
		descriptor.AdapterContractVersion() != adapterContractVersion ||
		descriptor.VerifiedAt().String() != "2026-08-23" ||
		descriptor.InterfaceType() != model.InterfaceTypeAPI ||
		descriptor.CredentialRequired() {
		t.Fatalf("SOT-IF-063: descriptor = %#v", descriptor)
	}

	capabilities := descriptor.Capabilities()
	if len(capabilities) != 1 {
		t.Fatalf("SOT-IF-063: capabilities = %#v", capabilities)
	}
	capability := capabilities[0]
	if capability.ID() != parliamentspeechsearch.CapabilityID ||
		capability.MajorVersion() != parliamentspeechsearch.MajorVersion ||
		capability.Level() != model.CapabilityLevelExtended ||
		capability.Stability() != model.CapabilityStabilityStable {
		t.Fatalf("SOT-IF-062/063: capability = %#v", capability)
	}
}
