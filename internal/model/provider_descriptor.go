package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var providerIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// InterfaceType は、外部情報源の提供方式を表す。
type InterfaceType string

const (
	// InterfaceTypeAPI は、API による提供を表す。
	InterfaceTypeAPI InterfaceType = "api"
	// InterfaceTypeHTML は、HTML による提供を表す。
	InterfaceTypeHTML InterfaceType = "html"
	// InterfaceTypeDownload は、ファイルの取得による提供を表す。
	InterfaceTypeDownload InterfaceType = "download"
	// InterfaceTypeHybrid は、複数方式を組み合わせた提供を表す。
	InterfaceTypeHybrid InterfaceType = "hybrid"
)

// ProviderDescriptorValues は、ProviderDescriptor の作成に必要な値を保持する。
type ProviderDescriptorValues struct {
	ProviderID             string
	Source                 InformationSource
	AdapterContractVersion string
	UpstreamSpecVersion    string
	VerifiedAt             Date
	InterfaceType          InterfaceType
	CredentialRequired     bool
	Capabilities           []ProviderCapability
}

// ProviderDescriptor は、一つのプロバイダーアダプターの不変な記述子を表す。
type ProviderDescriptor struct {
	providerID             string
	source                 InformationSource
	adapterContractVersion string
	upstreamSpecVersion    string
	verifiedAt             Date
	interfaceType          InterfaceType
	credentialRequired     bool
	capabilities           []ProviderCapability
}

// NewProviderDescriptor は、入力を複製して検証済みの ProviderDescriptor を返す。
func NewProviderDescriptor(values ProviderDescriptorValues) (ProviderDescriptor, error) {
	capabilities := cloneCapabilities(values.Capabilities)
	descriptor := ProviderDescriptor{
		providerID:             values.ProviderID,
		source:                 values.Source,
		adapterContractVersion: values.AdapterContractVersion,
		upstreamSpecVersion:    values.UpstreamSpecVersion,
		verifiedAt:             values.VerifiedAt,
		interfaceType:          values.InterfaceType,
		credentialRequired:     values.CredentialRequired,
		capabilities:           capabilities,
	}
	if err := descriptor.Validate(); err != nil {
		return ProviderDescriptor{}, err
	}
	return descriptor, nil
}

// ProviderID は、プロジェクト内で不変のプロバイダー識別子を返す。
func (d ProviderDescriptor) ProviderID() string {
	return d.providerID
}

// Source は、アダプターが取得する情報源を返す。
func (d ProviderDescriptor) Source() InformationSource {
	return d.source
}

// AdapterContractVersion は、アダプター境界の SemVer を返す。
func (d ProviderDescriptor) AdapterContractVersion() string {
	return d.adapterContractVersion
}

// UpstreamSpecVersion は、確認した外部仕様の版と有無を返す。
func (d ProviderDescriptor) UpstreamSpecVersion() (string, bool) {
	return d.upstreamSpecVersion, d.upstreamSpecVersion != ""
}

// VerifiedAt は、宣言した契約を再確認した最も古い日付を返す。
func (d ProviderDescriptor) VerifiedAt() Date {
	return d.verifiedAt
}

// InterfaceType は、外部情報源の提供方式を返す。
func (d ProviderDescriptor) InterfaceType() InterfaceType {
	return d.interfaceType
}

// CredentialRequired は、外部情報源の利用に認証情報が必要かを返す。
func (d ProviderDescriptor) CredentialRequired() bool {
	return d.credentialRequired
}

// Capabilities は、宣言された能力の複製を返す。
func (d ProviderDescriptor) Capabilities() []ProviderCapability {
	return cloneCapabilities(d.capabilities)
}

// Validate は、記述子の必須項目、版、列挙値および能力順を確認する。
func (d ProviderDescriptor) Validate() error {
	if !providerIDPattern.MatchString(d.providerID) {
		return fmt.Errorf("providerId は小文字の ASCII 英数字と内部のハイフンで構成しなければなりません")
	}
	if err := d.source.Validate(); err != nil {
		return fmt.Errorf("source が有効ではありません: %w", err)
	}
	if !isSemVer(d.adapterContractVersion) {
		return fmt.Errorf("adapterContractVersion は SemVer でなければなりません")
	}
	if err := d.verifiedAt.Validate(); err != nil {
		return fmt.Errorf("verifiedAt が有効ではありません: %w", err)
	}
	if !isInterfaceType(d.interfaceType) {
		return fmt.Errorf("interfaceType が定義されていません")
	}
	if d.capabilities == nil {
		return fmt.Errorf("capabilities は必須です")
	}
	return validateCapabilities(d.providerID, d.capabilities)
}

// MarshalJSON は、SOT-IF-014 の項目名で記述子を表す。
func (d ProviderDescriptor) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ProviderID             string               `json:"providerId"`
		Source                 InformationSource    `json:"source"`
		AdapterContractVersion string               `json:"adapterContractVersion"`
		UpstreamSpecVersion    string               `json:"upstreamSpecVersion,omitempty"`
		VerifiedAt             Date                 `json:"verifiedAt"`
		InterfaceType          InterfaceType        `json:"interfaceType"`
		CredentialRequired     bool                 `json:"credentialRequired"`
		Capabilities           []ProviderCapability `json:"capabilities"`
	}{
		ProviderID:             d.providerID,
		Source:                 d.source,
		AdapterContractVersion: d.adapterContractVersion,
		UpstreamSpecVersion:    d.upstreamSpecVersion,
		VerifiedAt:             d.verifiedAt,
		InterfaceType:          d.interfaceType,
		CredentialRequired:     d.credentialRequired,
		Capabilities:           cloneCapabilities(d.capabilities),
	})
}

func cloneCapabilities(values []ProviderCapability) []ProviderCapability {
	if values == nil {
		return nil
	}
	cloned := make([]ProviderCapability, len(values))
	copy(cloned, values)
	return cloned
}

func isInterfaceType(value InterfaceType) bool {
	return value == InterfaceTypeAPI ||
		value == InterfaceTypeHTML ||
		value == InterfaceTypeDownload ||
		value == InterfaceTypeHybrid
}

func validateCapabilities(providerID string, capabilities []ProviderCapability) error {
	for index, capability := range capabilities {
		if err := capability.Validate(); err != nil {
			return fmt.Errorf("capabilities[%d] が有効ではありません: %w", index, err)
		}
		if err := validateCapabilityNamespace(providerID, capability); err != nil {
			return err
		}
		if index == 0 {
			continue
		}

		previous := capabilities[index-1]
		switch {
		case capability.ID() < previous.ID():
			return fmt.Errorf("capabilities は id と majorVersion の昇順でなければなりません")
		case capability.ID() == previous.ID() &&
			capability.MajorVersion() == previous.MajorVersion():
			return fmt.Errorf("capability %s@%d が重複しています", capability.ID(), capability.MajorVersion())
		case capability.ID() == previous.ID() &&
			capability.MajorVersion() < previous.MajorVersion():
			return fmt.Errorf("capabilities は id と majorVersion の昇順でなければなりません")
		}
	}
	return nil
}

func validateCapabilityNamespace(
	providerID string,
	capability ProviderCapability,
) error {
	prefix := "provider." + providerID + "."
	if capability.Level() == CapabilityLevelProviderSpecific && !strings.HasPrefix(capability.ID(), prefix) {
		return fmt.Errorf(
			"provider 固有 capability %q は %q namespace を使用しなければなりません",
			capability.ID(),
			prefix,
		)
	}
	return nil
}
