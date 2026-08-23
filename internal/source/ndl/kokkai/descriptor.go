// Package kokkai は、国立国会図書館の国会会議録検索 API provider を実装する。
package kokkai

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	providerID             = "ndl-diet-speech-api"
	sourceID               = "ndl-diet-records"
	serviceURL             = "https://kokkai.ndl.go.jp/"
	adapterContractVersion = "1.0.0"
)

// Descriptor は、SOT-IF-063 の provider descriptor を返す。
func Descriptor() model.ProviderDescriptor {
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             providerID,
		Source:                 informationSource(),
		AdapterContractVersion: adapterContractVersion,
		VerifiedAt:             mustDescriptorDate("2026-08-23"),
		InterfaceType:          model.InterfaceTypeAPI,
		CredentialRequired:     false,
		Capabilities: []model.ProviderCapability{
			mustParliamentSpeechSearchCapability(),
		},
	})
	if err != nil {
		panic(fmt.Sprintf("国会発言 provider descriptor の固定値が不正です: %v", err))
	}
	return descriptor
}

func informationSource() model.InformationSource {
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "国会会議録検索システム",
		Publisher:  "国立国会図書館",
		Authority:  model.AuthorityOfficial,
		ServiceURL: serviceURL,
	})
	if err != nil {
		panic(fmt.Sprintf("国会発言 information source の固定値が不正です: %v", err))
	}
	return source
}

func mustParliamentSpeechSearchCapability() model.ProviderCapability {
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           parliamentspeechsearch.CapabilityID,
		MajorVersion: parliamentspeechsearch.MajorVersion,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(fmt.Sprintf("国会発言 capability の固定値が不正です: %v", err))
	}
	return capability
}

func mustDescriptorDate(value string) model.Date {
	date, err := model.NewDate(value)
	if err != nil {
		panic(fmt.Sprintf("国会発言 descriptor の固定日付が不正です: %v", err))
	}
	return date
}
