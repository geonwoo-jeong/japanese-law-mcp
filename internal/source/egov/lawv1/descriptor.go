package lawv1

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	providerID             = "e-gov-law-api-v1"
	adapterContractVersion = "1.0.0"
	upstreamSpecVersion    = "1"
)

// Descriptor は、e-Gov 法令 API Version 1 更新一覧の記述子を返す。
func Descriptor() model.ProviderDescriptor {
	capability := lawUpdateListCapability()
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             providerID,
		Source:                 informationSource(),
		AdapterContractVersion: adapterContractVersion,
		UpstreamSpecVersion:    upstreamSpecVersion,
		VerifiedAt:             mustDate("2026-07-26"),
		InterfaceType:          model.InterfaceTypeAPI,
		CredentialRequired:     false,
		Capabilities:           []model.ProviderCapability{capability},
	})
	if err != nil {
		panic(fmt.Sprintf("e-Gov Version 1 provider descriptor の固定値が不正です: %v", err))
	}
	return descriptor
}

func informationSource() model.InformationSource {
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         providerID,
		Name:       "e-Gov 法令 API Version 1",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/docs/law-data-basic/8529371-law-api-v1/",
	})
	if err != nil {
		panic(fmt.Sprintf("e-Gov Version 1 information source の固定値が不正です: %v", err))
	}
	return source
}

func lawUpdateListCapability() model.ProviderCapability {
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           lawupdatelist.CapabilityID,
		MajorVersion: lawupdatelist.MajorVersion,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(fmt.Sprintf("e-Gov Version 1 capability の固定値が不正です: %v", err))
	}
	return capability
}

func mustDate(value string) model.Date {
	date, err := model.NewDate(value)
	if err != nil {
		panic(fmt.Sprintf("e-Gov Version 1 descriptor の固定日付が不正です: %v", err))
	}
	return date
}
