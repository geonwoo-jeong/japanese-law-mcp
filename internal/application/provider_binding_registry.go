package application

import (
	"fmt"
	"reflect"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawsearch"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

// ProviderBindings は、一つの記述子と実装済みの型付き能力ポートを結び付ける。
type ProviderBindings struct {
	Descriptor       model.ProviderDescriptor
	LawSearch        lawsearch.Port
	LawContentSearch lawcontentsearch.Port
	LawDocumentRead  lawdocumentread.Port
	LawArticleRead   lawarticleread.Port
}

// ProviderBindingRegistry は、検証済みの型付き binding を providerId ごとに保持する。
type ProviderBindingRegistry struct {
	bindings    map[string]ProviderBindings
	initialized bool
}

// NewProviderBindingRegistry は、能力宣言と型付き port が正確に一致する registry を返す。
func NewProviderBindingRegistry(
	values []ProviderBindings,
) (ProviderBindingRegistry, error) {
	bindings := make(map[string]ProviderBindings, len(values))
	for index, value := range values {
		if err := value.Descriptor.Validate(); err != nil {
			return ProviderBindingRegistry{}, fmt.Errorf(
				"providers[%d] の ProviderDescriptor が有効ではありません: %w",
				index,
				err,
			)
		}
		providerID := value.Descriptor.ProviderID()
		if _, exists := bindings[providerID]; exists {
			return ProviderBindingRegistry{}, fmt.Errorf(
				"providerId %q が重複しています",
				providerID,
			)
		}
		if err := validateProviderBindings(value); err != nil {
			return ProviderBindingRegistry{}, fmt.Errorf(
				"providerId %q の binding が有効ではありません: %w",
				providerID,
				err,
			)
		}
		bindings[providerID] = value
	}
	return ProviderBindingRegistry{
		bindings:    bindings,
		initialized: true,
	}, nil
}

// Descriptor は、providerId が一致する不変な記述子を返す。
func (r ProviderBindingRegistry) Descriptor(
	providerID string,
) (model.ProviderDescriptor, bool) {
	binding, exists := r.bindings[providerID]
	if !exists {
		return model.ProviderDescriptor{}, false
	}
	return binding.Descriptor, true
}

// LawSearch は、providerId の law.search@1 port を返す。
func (r ProviderBindingRegistry) LawSearch(
	providerID string,
) (lawsearch.Port, bool) {
	binding, exists := r.bindings[providerID]
	if !exists || isNilTypedPort(binding.LawSearch) {
		return nil, false
	}
	return binding.LawSearch, true
}

// LawContentSearch は、providerId の law.content.search@1 port を返す。
func (r ProviderBindingRegistry) LawContentSearch(
	providerID string,
) (lawcontentsearch.Port, bool) {
	binding, exists := r.bindings[providerID]
	if !exists || isNilTypedPort(binding.LawContentSearch) {
		return nil, false
	}
	return binding.LawContentSearch, true
}

// LawDocumentRead は、providerId の law.document.read@1 port を返す。
func (r ProviderBindingRegistry) LawDocumentRead(
	providerID string,
) (lawdocumentread.Port, bool) {
	binding, exists := r.bindings[providerID]
	if !exists || isNilTypedPort(binding.LawDocumentRead) {
		return nil, false
	}
	return binding.LawDocumentRead, true
}

// LawArticleRead は、providerId の law.article.read@1 port を返す。
func (r ProviderBindingRegistry) LawArticleRead(
	providerID string,
) (lawarticleread.Port, bool) {
	binding, exists := r.bindings[providerID]
	if !exists || isNilTypedPort(binding.LawArticleRead) {
		return nil, false
	}
	return binding.LawArticleRead, true
}

func validateProviderBindings(value ProviderBindings) error {
	declared := make(map[providerRouteKey]struct{}, len(value.Descriptor.Capabilities()))
	for _, capability := range value.Descriptor.Capabilities() {
		key := providerRouteKey{
			capabilityID: capability.ID(),
			majorVersion: capability.MajorVersion(),
		}
		if !isSupportedProviderRouteKey(key) {
			return fmt.Errorf(
				"capability %s@%d に対応する型付き port がありません",
				capability.ID(),
				capability.MajorVersion(),
			)
		}
		if !hasPortForCapability(value, key) {
			return fmt.Errorf(
				"宣言した capability %s@%d の port がありません",
				capability.ID(),
				capability.MajorVersion(),
			)
		}
		declared[key] = struct{}{}
	}

	for _, key := range supportedProviderRouteKeys() {
		if !hasPortForCapability(value, key) {
			continue
		}
		if _, exists := declared[key]; !exists {
			return fmt.Errorf(
				"port %s@%d が ProviderDescriptor に宣言されていません",
				key.capabilityID,
				key.majorVersion,
			)
		}
	}
	return nil
}

func hasPortForCapability(
	value ProviderBindings,
	key providerRouteKey,
) bool {
	switch key {
	case lawSearchProviderRouteKey():
		return !isNilTypedPort(value.LawSearch)
	case lawContentSearchProviderRouteKey():
		return !isNilTypedPort(value.LawContentSearch)
	case lawDocumentReadProviderRouteKey():
		return !isNilTypedPort(value.LawDocumentRead)
	case lawArticleReadProviderRouteKey():
		return !isNilTypedPort(value.LawArticleRead)
	default:
		return false
	}
}

func isNilTypedPort(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
