package hanreipdf

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestDescriptorMatchesSOTIF070(t *testing.T) {
	t.Parallel()

	descriptor := Descriptor()
	if descriptor.ProviderID() != providerID ||
		descriptor.Source().ID() != sourceID ||
		descriptor.Source().Publisher() != "最高裁判所" ||
		descriptor.AdapterContractVersion() != adapterContractVersion ||
		descriptor.VerifiedAt().String() != "2026-08-26" ||
		descriptor.InterfaceType() != model.InterfaceTypeDownload {
		t.Fatalf("descriptor = %#v", descriptor)
	}
	capabilities := descriptor.Capabilities()
	if len(capabilities) != 1 || capabilities[0].ID() != "judicial-decision.case-citation.extract" {
		t.Fatalf("capabilities = %#v", capabilities)
	}
}
