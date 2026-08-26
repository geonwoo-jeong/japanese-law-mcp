// Package hanreipdf は、裁判所「裁判例検索」の全文 PDF から判例引用を抽出する。
package hanreipdf

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	providerID             = "courts-hanrei-pdf"
	sourceID               = "courts-hanrei"
	serviceURL             = "https://www.courts.go.jp/hanrei/search1/index.html"
	documentPathPrefix     = "/assets/hanrei/"
	adapterContractVersion = "1.0.0"
)

// Descriptor は、裁判所 PDF provider の固定 descriptor を返す。
func Descriptor() model.ProviderDescriptor {
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             providerID,
		Source:                 informationSource(),
		AdapterContractVersion: adapterContractVersion,
		VerifiedAt:             mustDescriptorDate("2026-08-26"),
		InterfaceType:          model.InterfaceTypeDownload,
		CredentialRequired:     false,
		Capabilities: []model.ProviderCapability{
			mustCapability(judicialcasecitationextract.CapabilityID),
		},
	})
	if err != nil {
		panic(fmt.Sprintf("裁判所 PDF provider descriptor の固定値が不正です: %v", err))
	}
	return descriptor
}

func informationSource() model.InformationSource {
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: serviceURL,
	})
	if err != nil {
		panic(fmt.Sprintf("裁判所 PDF information source の固定値が不正です: %v", err))
	}
	return source
}

func mustCapability(id string) model.ProviderCapability {
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           id,
		MajorVersion: judicialcasecitationextract.MajorVersion,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		panic(fmt.Sprintf("裁判所 PDF capability の固定値が不正です: %v", err))
	}
	return capability
}

func mustDescriptorDate(value string) model.Date {
	date, err := model.NewDate(value)
	if err != nil {
		panic(fmt.Sprintf("裁判所 PDF descriptor の固定日付が不正です: %v", err))
	}
	return date
}
