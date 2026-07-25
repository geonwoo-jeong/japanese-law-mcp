package model

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var capabilityIDPattern = regexp.MustCompile(
	`^[a-z0-9]+(?:-[a-z0-9]+)*(?:\.[a-z0-9]+(?:-[a-z0-9]+)*)+$`,
)

// CapabilityLevel は、能力の提供範囲を表す。
type CapabilityLevel string

const (
	// CapabilityLevelCore は、製品の中核能力を表す。
	CapabilityLevelCore CapabilityLevel = "core"
	// CapabilityLevelExtended は、共通契約を持つ拡張能力を表す。
	CapabilityLevelExtended CapabilityLevel = "extended"
	// CapabilityLevelProviderSpecific は、一つのプロバイダー固有の能力を表す。
	CapabilityLevelProviderSpecific CapabilityLevel = "provider_specific"
)

// CapabilityStability は、能力契約の安定性を表す。
type CapabilityStability string

const (
	// CapabilityStabilityStable は、安定した能力契約を表す。
	CapabilityStabilityStable CapabilityStability = "stable"
	// CapabilityStabilityExperimental は、試行中の能力契約を表す。
	CapabilityStabilityExperimental CapabilityStability = "experimental"
)

// ProviderCapabilityValues は、ProviderCapability の作成に必要な値を保持する。
type ProviderCapabilityValues struct {
	ID           string
	MajorVersion int
	Level        CapabilityLevel
	Stability    CapabilityStability
}

// ProviderCapability は、プロバイダーが実装する能力契約を表す。
type ProviderCapability struct {
	id           string
	majorVersion int
	level        CapabilityLevel
	stability    CapabilityStability
}

// NewProviderCapability は、検証済みの ProviderCapability を返す。
func NewProviderCapability(values ProviderCapabilityValues) (ProviderCapability, error) {
	capability := ProviderCapability{
		id:           values.ID,
		majorVersion: values.MajorVersion,
		level:        values.Level,
		stability:    values.Stability,
	}
	if err := capability.Validate(); err != nil {
		return ProviderCapability{}, err
	}
	return capability, nil
}

// ID は、能力識別子を返す。
func (c ProviderCapability) ID() string {
	return c.id
}

// MajorVersion は、入出力の互換性境界を返す。
func (c ProviderCapability) MajorVersion() int {
	return c.majorVersion
}

// Level は、能力の提供範囲を返す。
func (c ProviderCapability) Level() CapabilityLevel {
	return c.level
}

// Stability は、能力契約の安定性を返す。
func (c ProviderCapability) Stability() CapabilityStability {
	return c.stability
}

// Validate は、能力識別子、版、提供範囲および安定性を確認する。
func (c ProviderCapability) Validate() error {
	if !capabilityIDPattern.MatchString(c.id) {
		return fmt.Errorf("能力 id は小文字の dot-separated segments でなければなりません")
	}
	if c.majorVersion < 1 {
		return fmt.Errorf("能力の majorVersion は 1 以上でなければなりません")
	}
	if c.level != CapabilityLevelCore &&
		c.level != CapabilityLevelExtended &&
		c.level != CapabilityLevelProviderSpecific {
		return fmt.Errorf("能力の level が定義されていません")
	}
	providerNamespace := strings.HasPrefix(c.id, "provider.")
	if c.level == CapabilityLevelProviderSpecific &&
		(!providerNamespace || len(strings.Split(c.id, ".")) < 3) {
		return fmt.Errorf("provider_specific の能力 id は provider.{providerId}.{operation} 形式でなければなりません")
	}
	if c.level != CapabilityLevelProviderSpecific && providerNamespace {
		return fmt.Errorf("provider namespace の能力は provider_specific でなければなりません")
	}
	if c.stability != CapabilityStabilityStable &&
		c.stability != CapabilityStabilityExperimental {
		return fmt.Errorf("能力の stability が定義されていません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-013 の項目名で能力を表す。
func (c ProviderCapability) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ID           string              `json:"id"`
		MajorVersion int                 `json:"majorVersion"`
		Level        CapabilityLevel     `json:"level"`
		Stability    CapabilityStability `json:"stability"`
	}{
		ID:           c.id,
		MajorVersion: c.majorVersion,
		Level:        c.level,
		Stability:    c.stability,
	})
}
