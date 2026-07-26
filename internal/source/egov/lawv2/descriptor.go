package lawv2

import (
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

const (
	providerID             = "e-gov-law-api-v2"
	adapterContractVersion = "1.0.0"
	upstreamSpecVersion    = "2.1.139"
)

// Descriptor は、e-Gov 法令 API Version 2 の固定された記述子を返す。
//
// この段階では runtime registry へ登録せず、planned binding の契約テストだけで使う。
func Descriptor() model.ProviderDescriptor {
	source := informationSource()
	verifiedAt := mustDate("2026-07-25")
	capabilities := []model.ProviderCapability{
		mustCapability("law.article.read"),
		mustCapability("law.content.search"),
		mustCapability("law.document.read"),
		mustCapability("law.search"),
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             providerID,
		Source:                 source,
		AdapterContractVersion: adapterContractVersion,
		UpstreamSpecVersion:    upstreamSpecVersion,
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		CredentialRequired:     false,
		Capabilities:           capabilities,
	})
	if err != nil {
		panic(fmt.Sprintf("e-Gov provider descriptor の固定値が不正です: %v", err))
	}
	return descriptor
}

func informationSource() model.InformationSource {
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         providerID,
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/redoc/",
	})
	if err != nil {
		panic(fmt.Sprintf("e-Gov information source の固定値が不正です: %v", err))
	}
	return source
}

func lawSearchCapability() model.ProviderCapability {
	return mustCapability("law.search")
}

func mustCapability(id string) model.ProviderCapability {
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           id,
		MajorVersion: 1,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(fmt.Sprintf("e-Gov capability の固定値が不正です: %v", err))
	}
	return capability
}

func mustDate(value string) model.Date {
	date, err := model.NewDate(value)
	if err != nil {
		panic(fmt.Sprintf("e-Gov descriptor の固定日付が不正です: %v", err))
	}
	return date
}
