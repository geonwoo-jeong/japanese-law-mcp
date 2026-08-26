package judicialcitingcandidatesearch

import (
	"fmt"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const candidateTestProviderID = "test-candidate-provider"

type candidateTestOperation string

const candidateSearchTestOperation candidateTestOperation = "GET /candidate-test"

func (candidateTestOperation) SourceOperationProviderID() string {
	return candidateTestProviderID
}

func (operation candidateTestOperation) SourceOperationName() string {
	return string(operation)
}

func (operation candidateTestOperation) ValidateSourceOperation() error {
	if operation != candidateSearchTestOperation {
		return fmt.Errorf("試験用の候補検索 operation が定義されていません")
	}
	return nil
}

func mustCandidateSourceError(
	t *testing.T,
	code model.SourceErrorCode,
) model.SourceError {
	t.Helper()
	return mustTestSourceError(t, code, CapabilityID)
}

func mustTestSourceError(
	t *testing.T,
	code model.SourceErrorCode,
	capabilityID string,
) model.SourceError {
	t.Helper()

	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           capabilityID,
		MajorVersion: MajorVersion,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("capability: %v", err)
	}
	verifiedAt, err := model.NewDate("2026-08-26")
	if err != nil {
		t.Fatalf("verifiedAt: %v", err)
	}
	capabilities := []model.ProviderCapability{capability}
	if code == model.SourceErrorCodeUnsupportedCapability {
		capabilities = []model.ProviderCapability{}
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             candidateTestProviderID,
		Source:                 hanreiSource(t),
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeHTML,
		Capabilities:           capabilities,
	})
	if err != nil {
		t.Fatalf("descriptor: %v", err)
	}
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   descriptor,
		Capability: capability,
		Operation:  candidateSearchTestOperation,
	})
	if err != nil {
		t.Fatalf("source error: %v", err)
	}
	return sourceError
}
