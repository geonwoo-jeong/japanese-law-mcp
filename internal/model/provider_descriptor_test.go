package model_test

import (
	"encoding/json"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestDescriptor(t *testing.T) {
	t.Parallel()

	source := newInformationSource(t)
	verifiedAt := newDate(t, "2026-07-25")
	capabilities := []model.ProviderCapability{
		newCapability(t, "law.document.read", 1, model.CapabilityLevelCore),
		newCapability(t, "law.search", 1, model.CapabilityLevelCore),
	}

	got, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "e-gov-law-api-v2",
		Source:                 source,
		AdapterContractVersion: "1.0.0",
		UpstreamSpecVersion:    "2",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		CredentialRequired:     false,
		Capabilities:           capabilities,
	})
	if err != nil {
		t.Fatalf("SOT-IF-014: NewProviderDescriptor() のエラー = %v", err)
	}

	if got.ProviderID() != "e-gov-law-api-v2" ||
		got.Source() != source ||
		got.AdapterContractVersion() != "1.0.0" ||
		got.VerifiedAt() != verifiedAt ||
		got.InterfaceType() != model.InterfaceTypeAPI ||
		got.CredentialRequired() {
		t.Fatalf("SOT-IF-014: Descriptor = %#v", got)
	}
	upstreamVersion, ok := got.UpstreamSpecVersion()
	if !ok || upstreamVersion != "2" {
		t.Fatalf("SOT-IF-014: UpstreamSpecVersion() = %q, %t", upstreamVersion, ok)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-IF-014: Validate() のエラー = %v", err)
	}

	capabilities[0] = newCapability(t, "law.article.read", 1, model.CapabilityLevelCore)
	fromDescriptor := got.Capabilities()
	fromDescriptor[0] = newCapability(t, "law.content.search", 1, model.CapabilityLevelCore)
	if got.Capabilities()[0].ID() != "law.document.read" {
		t.Fatalf("SOT-IF-014: capabilities が外部から変更された: %#v", got.Capabilities())
	}
}

func TestDescriptorJSONOmitsAbsentUpstreamVersion(t *testing.T) {
	t.Parallel()

	got, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "e-gov-law-api-v2",
		Source:                 newInformationSource(t),
		AdapterContractVersion: "1.2.3-alpha-beta.x-y+build-meta.5",
		VerifiedAt:             newDate(t, "2026-07-25"),
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities: []model.ProviderCapability{
			newCapability(t, "law.search", 1, model.CapabilityLevelCore),
		},
	})
	if err != nil {
		t.Fatalf("SOT-IF-014: NewProviderDescriptor() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/SOT-IF-014: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/SOT-IF-014: JSON を再解析できない: %v", err)
	}
	if _, exists := object["upstreamSpecVersion"]; exists {
		t.Fatalf("SOT-MODEL-009/SOT-IF-014: upstreamSpecVersion が省略されていない: %s", encoded)
	}
	if object["providerId"] != "e-gov-law-api-v2" ||
		object["adapterContractVersion"] != "1.2.3-alpha-beta.x-y+build-meta.5" ||
		object["verifiedAt"] != "2026-07-25" ||
		object["interfaceType"] != "api" ||
		object["credentialRequired"] != false {
		t.Fatalf("SOT-MODEL-009/SOT-IF-014: JSON の値が一致しない: %#v", object)
	}
	capabilities, ok := object["capabilities"].([]any)
	if !ok || len(capabilities) != 1 {
		t.Fatalf("SOT-MODEL-009/SOT-IF-014: capabilities = %#v", object["capabilities"])
	}

	wantKeys := []string{
		"adapterContractVersion",
		"capabilities",
		"credentialRequired",
		"interfaceType",
		"providerId",
		"source",
		"verifiedAt",
	}
	gotKeys := make([]string, 0, len(object))
	for key := range object {
		gotKeys = append(gotKeys, key)
	}
	if !sameStringSet(gotKeys, wantKeys) {
		t.Fatalf("SOT-MODEL-009/SOT-IF-014: JSON keys = %v、期待値 = %v", gotKeys, wantKeys)
	}
}

func TestDescriptorRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	source := newInformationSource(t)
	verifiedAt := newDate(t, "2026-07-25")
	lawSearch := newCapability(t, "law.search", 1, model.CapabilityLevelCore)
	lawSearchV2 := newCapability(t, "law.search", 2, model.CapabilityLevelCore)
	lawDocument := newCapability(t, "law.document.read", 1, model.CapabilityLevelCore)
	providerSpecific := newCapability(
		t,
		"provider.other-provider.special-search",
		1,
		model.CapabilityLevelProviderSpecific,
	)

	valid := model.ProviderDescriptorValues{
		ProviderID:             "e-gov-law-api-v2",
		Source:                 source,
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           []model.ProviderCapability{lawSearch},
	}

	tests := map[string]model.ProviderDescriptorValues{
		"providerId の形式不正": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.ProviderID = "E-Gov"
		}),
		"source のゼロ値": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.Source = model.InformationSource{}
		}),
		"SemVer ではない adapterContractVersion": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.AdapterContractVersion = "1.0"
		}),
		"verifiedAt のゼロ値": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.VerifiedAt = model.Date{}
		}),
		"未知の interfaceType": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.InterfaceType = model.InterfaceType("ftp")
		}),
		"capability の順序不正": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.Capabilities = []model.ProviderCapability{lawSearch, lawDocument}
		}),
		"capability の重複": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.Capabilities = []model.ProviderCapability{lawSearch, lawSearch}
		}),
		"同一 capability の版順序不正": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.Capabilities = []model.ProviderCapability{lawSearchV2, lawSearch}
		}),
		"capability のゼロ値": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.Capabilities = []model.ProviderCapability{{}}
		}),
		"provider 固有 namespace の不一致": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.Capabilities = []model.ProviderCapability{providerSpecific}
		}),
		"capabilities の欠落": withDescriptorChange(valid, func(values *model.ProviderDescriptorValues) {
			values.Capabilities = nil
		}),
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewProviderDescriptor(values); err == nil {
				t.Fatalf("SOT-IF-014: NewProviderDescriptor(%#v) が成功した", values)
			}
		})
	}
}

func TestDescriptorAllowsEmptyCapabilities(t *testing.T) {
	t.Parallel()

	got, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "metadata-only-provider",
		Source:                 newInformationSource(t),
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             newDate(t, "2026-07-25"),
		InterfaceType:          model.InterfaceTypeDownload,
		Capabilities:           []model.ProviderCapability{},
	})
	if err != nil {
		t.Fatalf("SOT-IF-014: 空の capabilities を拒否した: %v", err)
	}
	if got.Capabilities() == nil || len(got.Capabilities()) != 0 {
		t.Fatalf("SOT-IF-014: capabilities = %#v、期待値 = 空配列", got.Capabilities())
	}
}

func TestProviderDescriptorAcceptsMatchingProviderSpecificCapability(t *testing.T) {
	t.Parallel()

	values := validDescriptorValues(t)
	values.Capabilities = []model.ProviderCapability{
		newCapability(
			t,
			"provider.e-gov-law-api-v2.special-search",
			1,
			model.CapabilityLevelProviderSpecific,
		),
	}
	if _, err := model.NewProviderDescriptor(values); err != nil {
		t.Fatalf("SOT-MODEL-013/SOT-IF-014: 一致する provider namespace を拒否した: %v", err)
	}
}

func TestProviderDescriptorAcceptsInterfaceTypes(t *testing.T) {
	t.Parallel()

	for _, interfaceType := range []model.InterfaceType{
		model.InterfaceTypeAPI,
		model.InterfaceTypeHTML,
		model.InterfaceTypeDownload,
		model.InterfaceTypeHybrid,
	} {
		interfaceType := interfaceType
		t.Run(string(interfaceType), func(t *testing.T) {
			t.Parallel()

			values := validDescriptorValues(t)
			values.InterfaceType = interfaceType
			if _, err := model.NewProviderDescriptor(values); err != nil {
				t.Fatalf("SOT-IF-014: interfaceType %q を拒否した: %v", interfaceType, err)
			}
		})
	}
}

func TestZeroProviderDescriptorCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.ProviderDescriptor{}); err == nil {
		t.Fatal("SOT-MODEL-009/SOT-IF-014: ProviderDescriptor のゼロ値を JSON に変換できた")
	}
}

func TestProviderDescriptorSemVer(t *testing.T) {
	t.Parallel()

	for _, version := range []string{
		"0.0.0",
		"1.2.3-alpha-beta",
		"1.2.3-x-y.z",
		"1.2.3+build-meta.5",
		"1.2.3-alpha-beta.x-y+build-meta.5",
	} {
		version := version
		t.Run("有効_"+version, func(t *testing.T) {
			t.Parallel()

			values := validDescriptorValues(t)
			values.AdapterContractVersion = version
			if _, err := model.NewProviderDescriptor(values); err != nil {
				t.Fatalf("SOT-IF-014: 有効な SemVer %q を拒否した: %v", version, err)
			}
		})
	}

	for _, version := range []string{
		"",
		"v1.2.3",
		"1.2",
		"01.2.3",
		"1.02.3",
		"1.2.03",
		"1.2.3-",
		"1.2.3+",
		"1.2.3-01",
		"1.2.3-alpha..1",
		"1.2.3+build..1",
		"1.2.3-α",
		".1.2",
	} {
		version := version
		t.Run("無効_"+version, func(t *testing.T) {
			t.Parallel()

			values := validDescriptorValues(t)
			values.AdapterContractVersion = version
			if _, err := model.NewProviderDescriptor(values); err == nil {
				t.Fatalf("SOT-IF-014: 無効な SemVer %q を受理した", version)
			}
		})
	}
}

func newInformationSource(t *testing.T) model.InformationSource {
	t.Helper()

	got, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できない: %v", err)
	}
	return got
}

func validDescriptorValues(t *testing.T) model.ProviderDescriptorValues {
	t.Helper()

	return model.ProviderDescriptorValues{
		ProviderID:             "e-gov-law-api-v2",
		Source:                 newInformationSource(t),
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             newDate(t, "2026-07-25"),
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities: []model.ProviderCapability{
			newCapability(t, "law.search", 1, model.CapabilityLevelCore),
		},
	}
}

func newDate(t *testing.T, value string) model.Date {
	t.Helper()

	got, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("Date を作成できない: %v", err)
	}
	return got
}

func newCapability(
	t *testing.T,
	id string,
	majorVersion int,
	level model.CapabilityLevel,
) model.ProviderCapability {
	t.Helper()

	got, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           id,
		MajorVersion: majorVersion,
		Level:        level,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("ProviderCapability を作成できない: %v", err)
	}
	return got
}

func withDescriptorChange(
	base model.ProviderDescriptorValues,
	change func(*model.ProviderDescriptorValues),
) model.ProviderDescriptorValues {
	capabilities := make([]model.ProviderCapability, len(base.Capabilities))
	copy(capabilities, base.Capabilities)
	values := base
	values.Capabilities = capabilities
	change(&values)
	return values
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
