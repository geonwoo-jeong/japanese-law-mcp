package application

// ProviderBindingMetadata は、選択済み capability binding の
// provider 非依存 metadata snapshot である。
type ProviderBindingMetadata struct {
	providerID             string
	sourceID               string
	capabilityID           string
	capabilityMajorVersion int
}

// ProviderID は、選択した provider の識別子を返す。
func (m ProviderBindingMetadata) ProviderID() string {
	return m.providerID
}

// SourceID は、選択した provider descriptor の情報源識別子を返す。
func (m ProviderBindingMetadata) SourceID() string {
	return m.sourceID
}

// CapabilityID は、選択した capability の識別子を返す。
func (m ProviderBindingMetadata) CapabilityID() string {
	return m.capabilityID
}

// CapabilityMajorVersion は、選択した capability の major version を返す。
func (m ProviderBindingMetadata) CapabilityMajorVersion() int {
	return m.capabilityMajorVersion
}

// PrimaryBindingMetadata は、実効 primary route の binding metadata を返す。
func (r ProviderRoutes) PrimaryBindingMetadata(
	capabilityID string,
	majorVersion int,
) (ProviderBindingMetadata, bool) {
	if !r.initialized {
		return ProviderBindingMetadata{}, false
	}
	providerID, exists := r.ProviderID(capabilityID, majorVersion)
	if !exists {
		return ProviderBindingMetadata{}, false
	}
	return r.bindingMetadata(providerID, capabilityID, majorVersion)
}

// ExplicitBindingMetadata は、明示した provider の binding metadata を返す。
func (r ProviderRoutes) ExplicitBindingMetadata(
	providerID string,
	capabilityID string,
	majorVersion int,
) (ProviderBindingMetadata, bool) {
	if !r.initialized {
		return ProviderBindingMetadata{}, false
	}
	return r.bindingMetadata(providerID, capabilityID, majorVersion)
}

func (r ProviderRoutes) bindingMetadata(
	providerID string,
	capabilityID string,
	majorVersion int,
) (ProviderBindingMetadata, bool) {
	key := providerRouteKey{
		capabilityID: capabilityID,
		majorVersion: majorVersion,
	}
	if !isSupportedProviderRouteKey(key) ||
		!r.registry.hasBinding(providerID, key) {
		return ProviderBindingMetadata{}, false
	}
	descriptor, exists := r.registry.Descriptor(providerID)
	if !exists {
		return ProviderBindingMetadata{}, false
	}
	return ProviderBindingMetadata{
		providerID:             providerID,
		sourceID:               descriptor.Source().ID(),
		capabilityID:           capabilityID,
		capabilityMajorVersion: majorVersion,
	}, true
}
