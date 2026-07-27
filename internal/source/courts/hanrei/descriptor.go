// Package hanrei は、裁判所「裁判例検索」HTML の型付き adapter を提供する。
package hanrei

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	providerID             = "courts-hanrei-html"
	sourceID               = "courts-hanrei"
	searchEndpoint         = "https://www.courts.go.jp/hanrei/search1/index.html"
	adapterContractVersion = "1.0.0"
)

// Descriptor は、SOT-IF-043 の provider descriptor を返す。
func Descriptor() model.ProviderDescriptor {
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             providerID,
		Source:                 informationSource(),
		AdapterContractVersion: adapterContractVersion,
		VerifiedAt:             mustDescriptorDate("2026-07-26"),
		InterfaceType:          model.InterfaceTypeHTML,
		CredentialRequired:     false,
		Capabilities: []model.ProviderCapability{
			mustJudicialCapability("judicial-decision.read"),
			mustJudicialCapability("judicial-decision.search"),
		},
	})
	if err != nil {
		panic(fmt.Sprintf("裁判所 provider descriptor の固定値が不正です: %v", err))
	}
	return descriptor
}

func informationSource() model.InformationSource {
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: searchEndpoint,
	})
	if err != nil {
		panic(fmt.Sprintf("裁判所 information source の固定値が不正です: %v", err))
	}
	return source
}

func judicialDecisionSearchCapability() model.ProviderCapability {
	return mustJudicialCapability("judicial-decision.search")
}

func mustJudicialCapability(id string) model.ProviderCapability {
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           id,
		MajorVersion: 1,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(fmt.Sprintf("裁判所 capability の固定値が不正です: %v", err))
	}
	return capability
}

func mustDescriptorDate(value string) model.Date {
	date, err := model.NewDate(value)
	if err != nil {
		panic(fmt.Sprintf("裁判所 descriptor の固定日付が不正です: %v", err))
	}
	return date
}
