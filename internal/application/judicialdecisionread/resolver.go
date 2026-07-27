package judicialdecisionread

import (
	"fmt"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// ProviderBinding は、一つの provider 記述子、構成状態および詳細取得 port を結び付ける。
type ProviderBinding struct {
	Descriptor model.ProviderDescriptor
	Enabled    bool
	Port       Port
}

type resolvedBinding struct {
	descriptor model.ProviderDescriptor
	enabled    bool
	port       Port
}

// ResolutionError は、参照または provider 状態により外部呼出し前に失敗したことを表す。
type ResolutionError struct {
	code       model.ErrorCode
	providerID string
}

// Error は、分類に対応する安全な日本語メッセージを返す。
func (e ResolutionError) Error() string {
	switch e.code {
	case model.ErrorCodeInvalidArgument:
		return "裁判例の資源参照が登録済みプロバイダーと一致しません"
	case model.ErrorCodeConfigurationRequired:
		return "裁判例のプロバイダーが無効化されています"
	case model.ErrorCodeUnsupportedCapability:
		return "裁判例のプロバイダーは詳細取得に対応していません"
	default:
		return "裁判例のプロバイダーを解決できません"
	}
}

// Code は、公開エラーへ対応できる分類を返す。
func (e ResolutionError) Code() model.ErrorCode {
	return e.code
}

// ProviderID は、解決しようとした providerId を返す。
func (e ResolutionError) ProviderID() string {
	return e.providerID
}

// Resolver は、起動時に検証した binding を providerId ごとに保持する。
type Resolver struct {
	bindings    map[string]resolvedBinding
	initialized bool
}

// NewResolver は、能力宣言と型付き port が一致する不変な resolver を返す。
func NewResolver(values []ProviderBinding) (Resolver, error) {
	bindings := make(map[string]resolvedBinding, len(values))
	for index, value := range values {
		if err := value.Descriptor.Validate(); err != nil {
			return Resolver{}, fmt.Errorf(
				"providers[%d] の ProviderDescriptor が有効ではありません: %w",
				index,
				err,
			)
		}
		providerID := value.Descriptor.ProviderID()
		if _, exists := bindings[providerID]; exists {
			return Resolver{}, fmt.Errorf(
				"providerId %q が重複しています",
				providerID,
			)
		}
		declared := declaresReadCapability(value.Descriptor)
		hasPort := !isNilPort(value.Port)
		switch {
		case declared && !hasPort:
			return Resolver{}, fmt.Errorf(
				"providerId %q は %s@%d を宣言していますが port がありません",
				providerID,
				CapabilityID,
				MajorVersion,
			)
		case !declared && hasPort:
			return Resolver{}, fmt.Errorf(
				"providerId %q の port が ProviderDescriptor に宣言されていません",
				providerID,
			)
		}
		bindings[providerID] = resolvedBinding{
			descriptor: value.Descriptor,
			enabled:    value.Enabled,
			port:       value.Port,
		}
	}
	return Resolver{
		bindings:    bindings,
		initialized: true,
	}, nil
}

// Resolve は、既定 route を使わず ref.providerId と完全一致する port を返す。
func (r Resolver) Resolve(request Request) (Port, error) {
	if !r.initialized {
		return nil, fmt.Errorf(
			"Resolver は NewResolver で作成しなければなりません",
		)
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}

	ref := request.Ref()
	providerID := ref.ProviderID()
	binding, exists := r.bindings[providerID]
	if !exists {
		return nil, newResolutionError(
			model.ErrorCodeInvalidArgument,
			providerID,
		)
	}
	if binding.descriptor.Source().ID() != ref.Key().SourceID() {
		return nil, newResolutionError(
			model.ErrorCodeInvalidArgument,
			providerID,
		)
	}
	if !binding.enabled {
		return nil, newResolutionError(
			model.ErrorCodeConfigurationRequired,
			providerID,
		)
	}
	if !declaresReadCapability(binding.descriptor) || isNilPort(binding.port) {
		return nil, newResolutionError(
			model.ErrorCodeUnsupportedCapability,
			providerID,
		)
	}
	return binding.port, nil
}

func newResolutionError(
	code model.ErrorCode,
	providerID string,
) ResolutionError {
	return ResolutionError{
		code:       code,
		providerID: providerID,
	}
}

func declaresReadCapability(descriptor model.ProviderDescriptor) bool {
	for _, capability := range descriptor.Capabilities() {
		if capability.ID() == CapabilityID &&
			capability.MajorVersion() == MajorVersion {
			return true
		}
	}
	return false
}

func isNilPort(port Port) bool {
	if port == nil {
		return true
	}
	value := reflect.ValueOf(port)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
