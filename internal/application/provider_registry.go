package application

import (
	"fmt"
	"sort"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type providerCapabilityKey struct {
	providerID   string
	capabilityID string
	majorVersion int
}

// ProviderRegistry は、起動時に確定したプロバイダー記述子と能力宣言を保持する。
type ProviderRegistry struct {
	descriptors  map[string]model.ProviderDescriptor
	ordered      []model.ProviderDescriptor
	declarations map[providerCapabilityKey]model.ProviderCapability
}

// NewProviderRegistry は、記述子と能力宣言を検証して不変な registry を返す。
func NewProviderRegistry(descriptors []model.ProviderDescriptor) (ProviderRegistry, error) {
	byProviderID := make(map[string]model.ProviderDescriptor, len(descriptors))
	ordered := make([]model.ProviderDescriptor, 0, len(descriptors))
	declarations := make(map[providerCapabilityKey]model.ProviderCapability)

	for index, descriptor := range descriptors {
		if err := descriptor.Validate(); err != nil {
			return ProviderRegistry{}, fmt.Errorf(
				"providers[%d] の ProviderDescriptor が有効ではありません: %w",
				index,
				err,
			)
		}
		providerID := descriptor.ProviderID()
		if _, exists := byProviderID[providerID]; exists {
			return ProviderRegistry{}, fmt.Errorf(
				"providerId %q が重複しています",
				providerID,
			)
		}

		byProviderID[providerID] = descriptor
		ordered = append(ordered, descriptor)
		for _, capability := range descriptor.Capabilities() {
			key := providerCapabilityKey{
				providerID:   providerID,
				capabilityID: capability.ID(),
				majorVersion: capability.MajorVersion(),
			}
			declarations[key] = capability
		}
	}

	sort.Slice(ordered, func(left, right int) bool {
		return ordered[left].ProviderID() < ordered[right].ProviderID()
	})

	return ProviderRegistry{
		descriptors:  byProviderID,
		ordered:      ordered,
		declarations: declarations,
	}, nil
}

// Descriptors は、providerId 順の記述子の複製を返す。
func (r ProviderRegistry) Descriptors() []model.ProviderDescriptor {
	descriptors := make([]model.ProviderDescriptor, len(r.ordered))
	copy(descriptors, r.ordered)
	return descriptors
}

// Descriptor は、providerId が一致する記述子を返す。
func (r ProviderRegistry) Descriptor(providerID string) (model.ProviderDescriptor, bool) {
	descriptor, exists := r.descriptors[providerID]
	return descriptor, exists
}

// DeclaredCapability は、完全一致する能力宣言を返す。
func (r ProviderRegistry) DeclaredCapability(
	providerID string,
	capabilityID string,
	majorVersion int,
) (model.ProviderCapability, bool) {
	capability, exists := r.declarations[providerCapabilityKey{
		providerID:   providerID,
		capabilityID: capabilityID,
		majorVersion: majorVersion,
	}]
	return capability, exists
}
