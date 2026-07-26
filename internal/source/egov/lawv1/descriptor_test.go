package lawv1

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestDescriptorはVersion1更新一覧だけを宣言する(t *testing.T) {
	t.Parallel()

	descriptor := Descriptor()
	if descriptor.ProviderID() != "e-gov-law-api-v1" {
		t.Fatalf("providerId = %q", descriptor.ProviderID())
	}
	if descriptor.Source().ID() != descriptor.ProviderID() {
		t.Fatalf("sourceId = %q", descriptor.Source().ID())
	}
	if descriptor.AdapterContractVersion() != "1.0.0" {
		t.Fatalf("adapterContractVersion = %q", descriptor.AdapterContractVersion())
	}
	if version, exists := descriptor.UpstreamSpecVersion(); !exists || version != "1" {
		t.Fatalf("upstreamSpecVersion = %q, %t", version, exists)
	}
	if descriptor.VerifiedAt().String() != "2026-07-26" {
		t.Fatalf("verifiedAt = %q", descriptor.VerifiedAt())
	}
	if descriptor.InterfaceType() != model.InterfaceTypeAPI ||
		descriptor.CredentialRequired() {
		t.Fatal("interfaceType または credentialRequired が不正です")
	}
	capabilities := descriptor.Capabilities()
	if len(capabilities) != 1 {
		t.Fatalf("capabilities = %d", len(capabilities))
	}
	capability := capabilities[0]
	if capability.ID() != lawupdatelist.CapabilityID ||
		capability.MajorVersion() != lawupdatelist.MajorVersion ||
		capability.Level() != model.CapabilityLevelCore ||
		capability.Stability() != model.CapabilityStabilityStable {
		t.Fatalf("SOT-IF-034/035: capability = %#v", capability)
	}
}

func TestNewProviderBindingsは型付きPortを構成する(t *testing.T) {
	t.Parallel()

	bindings, err := NewProviderBindings()
	if err != nil {
		t.Fatalf("NewProviderBindings() error = %v", err)
	}
	if bindings.Descriptor.ProviderID() != Descriptor().ProviderID() {
		t.Fatalf("providerId = %q", bindings.Descriptor.ProviderID())
	}
	if bindings.LawUpdateList == nil {
		t.Fatal("LawUpdateList port がありません")
	}
}
