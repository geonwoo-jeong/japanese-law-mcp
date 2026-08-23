package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestValidationErrorCanBeClassifiedAndUnwrapped(t *testing.T) {
	t.Parallel()

	cause := errors.New("provider schema が一致しません")
	got := NewValidationError(cause)
	if !IsValidationError(got) {
		t.Fatal("SOT-ARCH-015: validation error として判定できません")
	}
	if !errors.Is(got, cause) {
		t.Fatal("SOT-ARCH-015: validation error の原因を取得できません")
	}
	if got.Error() != cause.Error() {
		t.Fatalf("SOT-ARCH-015: validation error = %q、期待値 = %q", got, cause)
	}
	if NewValidationError(nil) != nil {
		t.Fatal("SOT-ARCH-015: NewValidationError(nil) != nil")
	}
}

func TestDefaultProviderConfiguration(t *testing.T) {
	t.Parallel()

	got := Default()
	providers := got.Providers()
	if len(providers) != 2 {
		t.Fatalf("SOT-IF-026: providers の件数 = %d、期待値 = 2", len(providers))
	}
	for _, providerID := range []string{"e-gov-law-api-v1", "e-gov-law-api-v2"} {
		provider, exists := providers[providerID]
		if !exists {
			t.Fatalf("SOT-IF-026: provider %q がありません", providerID)
		}
		if !provider.Enabled ||
			len(provider.Settings) != 0 ||
			len(provider.CredentialEnvRefs) != 0 {
			t.Fatalf("SOT-IF-026: provider %q = %#v", providerID, provider)
		}
	}

	wantRoutes := map[ProviderRouteKey]string{
		{CapabilityID: "law.article.read", MajorVersion: 1}:    "e-gov-law-api-v2",
		{CapabilityID: "law.content.search", MajorVersion: 1}:  "e-gov-law-api-v2",
		{CapabilityID: "law.document.read", MajorVersion: 1}:   "e-gov-law-api-v2",
		{CapabilityID: "law.version.compare", MajorVersion: 1}: "e-gov-law-api-v2",
		{CapabilityID: "law.revision.list", MajorVersion: 1}:   "e-gov-law-api-v2",
		{CapabilityID: "law.search", MajorVersion: 1}:          "e-gov-law-api-v2",
		{CapabilityID: "law.update.list", MajorVersion: 1}:     "e-gov-law-api-v1",
	}
	routes := got.ProviderRoutes()
	if len(routes) != len(wantRoutes) {
		t.Fatalf("SOT-IF-026: providerRoutes の件数 = %d、期待値 = %d", len(routes), len(wantRoutes))
	}
	for key, wantProviderID := range wantRoutes {
		route, exists := routes[key]
		if !exists {
			t.Fatalf("SOT-IF-026: route %s がありません", key)
		}
		if route.Selection != ProviderRouteSelectionPrimary ||
			route.DefaultProviderID != wantProviderID ||
			len(route.AggregateProviderIDs) != 0 ||
			route.RollbackProviderID != "" {
			t.Fatalf("SOT-IF-026: route %s = %#v", key, route)
		}
	}
}

func TestConfigOwnsProviderConfigurationSnapshots(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"nested": map[string]any{
			"regions": []any{"jp", "us"},
		},
	}
	credentials := map[string]CredentialEnvRef{
		"apiKey": {Type: "env", Name: "CUSTOM_API_KEY"},
	}
	aggregateProviderIDs := []string{"custom-provider", "secondary-provider"}

	values := validProviderValues()
	values.Providers = map[string]ProviderConfig{
		"custom-provider": {
			Enabled:           true,
			Settings:          settings,
			CredentialEnvRefs: credentials,
		},
	}
	values.ProviderRoutes = map[string]ProviderRoute{
		"provider.custom.read@2": {
			Selection:            ProviderRouteSelectionAggregate,
			AggregateProviderIDs: aggregateProviderIDs,
		},
	}

	got, err := New(values)
	if err != nil {
		t.Fatalf("SOT-ARCH-015: New() のエラー = %v", err)
	}

	settings["nested"].(map[string]any)["regions"].([]any)[0] = "changed"
	credentials["apiKey"] = CredentialEnvRef{Type: "env", Name: "CHANGED"}
	aggregateProviderIDs[0] = "changed-provider"

	provider, exists := got.Provider("custom-provider")
	if !exists {
		t.Fatal("SOT-ARCH-015: custom-provider がありません")
	}
	provider.Settings["nested"].(map[string]any)["regions"].([]any)[0] = "also-changed"
	provider.CredentialEnvRefs["apiKey"] = CredentialEnvRef{Type: "env", Name: "ALSO_CHANGED"}

	routeKey := ProviderRouteKey{CapabilityID: "provider.custom.read", MajorVersion: 2}
	route, exists := got.ProviderRoute(routeKey)
	if !exists {
		t.Fatalf("SOT-ARCH-015: route %s がありません", routeKey)
	}
	route.AggregateProviderIDs[0] = "also-changed-provider"

	allProviders := got.Providers()
	delete(allProviders, "custom-provider")
	allRoutes := got.ProviderRoutes()
	delete(allRoutes, routeKey)

	provider, exists = got.Provider("custom-provider")
	if !exists {
		t.Fatal("SOT-ARCH-015: snapshot の変更で provider が削除されました")
	}
	regions := provider.Settings["nested"].(map[string]any)["regions"].([]any)
	if !slices.Equal(regions, []any{"jp", "us"}) {
		t.Fatalf("SOT-ARCH-015: settings が外部から変更されました: %v", regions)
	}
	if provider.CredentialEnvRefs["apiKey"].Name != "CUSTOM_API_KEY" {
		t.Fatalf("SOT-ARCH-015: credentialEnvRefs が外部から変更されました: %#v", provider.CredentialEnvRefs)
	}

	route, exists = got.ProviderRoute(routeKey)
	if !exists {
		t.Fatal("SOT-ARCH-015: snapshot の変更で route が削除されました")
	}
	if !slices.Equal(route.AggregateProviderIDs, []string{"custom-provider", "secondary-provider"}) {
		t.Fatalf("SOT-ARCH-015: aggregateProviderIds が外部から変更されました: %v", route.AggregateProviderIDs)
	}
}

func TestParseProviderRouteKey(t *testing.T) {
	t.Parallel()

	got, err := ParseProviderRouteKey("provider.custom-read.document@12")
	if err != nil {
		t.Fatalf("SOT-IF-026: ParseProviderRouteKey() のエラー = %v", err)
	}
	want := ProviderRouteKey{
		CapabilityID: "provider.custom-read.document",
		MajorVersion: 12,
	}
	if got != want || got.String() != "provider.custom-read.document@12" {
		t.Fatalf("SOT-IF-026: route key = %#v / %q、期待値 = %#v", got, got.String(), want)
	}

	for _, input := range []string{
		"",
		"law@1",
		"Law.search@1",
		"law.-search@1",
		"law.search-@1",
		"law.search",
		"law.search@0",
		"law.search@01",
		"law.search@-1",
		"law.search@1@2",
	} {
		if _, parseErr := ParseProviderRouteKey(input); parseErr == nil {
			t.Errorf("SOT-IF-026: ParseProviderRouteKey(%q) のエラー = nil", input)
		}
	}
}

func TestLoadProviderConfigurationFormatsAndOverlay(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]string{
		"config.yaml": `
providers:
  custom-provider:
    enabled: false
    settings:
      region: jp
    credentialEnvRefs:
      apiKey:
        type: env
        name: CUSTOM_API_KEY
providerRoutes:
  provider.custom.read@2:
    selection: aggregate
    aggregateProviderIds:
      - custom-provider
      - secondary-provider
`,
		"config.json": `{
  "providers": {
    "custom-provider": {
      "enabled": false,
      "settings": {"region": "jp"},
      "credentialEnvRefs": {
        "apiKey": {"type": "env", "name": "CUSTOM_API_KEY"}
      }
    }
  },
  "providerRoutes": {
    "provider.custom.read@2": {
      "selection": "aggregate",
      "aggregateProviderIds": ["custom-provider", "secondary-provider"]
    }
  }
}`,
		"config.toml": `
[providers.custom-provider]
enabled = false

[providers.custom-provider.settings]
region = "jp"

[providers.custom-provider.credentialEnvRefs.apiKey]
type = "env"
name = "CUSTOM_API_KEY"

[providerRoutes."provider.custom.read@2"]
selection = "aggregate"
aggregateProviderIds = ["custom-provider", "secondary-provider"]
`,
	}

	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			writeFile(t, path, content)

			got, err := Load(LoadOptions{
				Flags:         newFlagSet(t),
				ConfigFile:    path,
				UserConfigDir: fixedUserConfigDir(t.TempDir()),
			})
			if err != nil {
				t.Fatalf("SOT-IF-020/SOT-IF-026: Load() のエラー = %v", err)
			}

			providers := got.Providers()
			if len(providers) != 3 {
				t.Fatalf("SOT-IF-026: providers の件数 = %d、期待値 = 3", len(providers))
			}
			custom := providers["custom-provider"]
			if custom.Enabled ||
				custom.Settings["region"] != "jp" ||
				custom.CredentialEnvRefs["apiKey"] != (CredentialEnvRef{Type: "env", Name: "CUSTOM_API_KEY"}) {
				t.Fatalf("SOT-IF-026: custom-provider = %#v", custom)
			}

			routes := got.ProviderRoutes()
			if len(routes) != 8 {
				t.Fatalf("SOT-IF-026: providerRoutes の件数 = %d、期待値 = 8", len(routes))
			}
			customRoute := routes[ProviderRouteKey{CapabilityID: "provider.custom.read", MajorVersion: 2}]
			if customRoute.Selection != ProviderRouteSelectionAggregate ||
				!slices.Equal(customRoute.AggregateProviderIDs, []string{"custom-provider", "secondary-provider"}) {
				t.Fatalf("SOT-IF-026: custom route = %#v", customRoute)
			}
		})
	}
}

func TestLoadProviderConfigurationUsesAtomicKeyOverlay(t *testing.T) {
	clearKnownEnvironment(t)

	t.Run("route object は atomic", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		writeFile(t, path, `
providerRoutes:
  law.search@1:
    selection: explicit
`)

		got, err := Load(LoadOptions{
			Flags:         newFlagSet(t),
			ConfigFile:    path,
			UserConfigDir: fixedUserConfigDir(t.TempDir()),
		})
		if err != nil {
			t.Fatalf("SOT-IF-026: Load() のエラー = %v", err)
		}
		routes := got.ProviderRoutes()
		if len(routes) != 7 {
			t.Fatalf("SOT-IF-026: providerRoutes の件数 = %d、期待値 = 7", len(routes))
		}
		route := routes[ProviderRouteKey{CapabilityID: "law.search", MajorVersion: 1}]
		if route.Selection != ProviderRouteSelectionExplicit ||
			route.DefaultProviderID != "" {
			t.Fatalf("SOT-IF-026: atomic replacement 後の route = %#v", route)
		}
	})

	t.Run("provider object は atomic", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		writeFile(t, path, `
providers:
  e-gov-law-api-v2:
    settings: {}
`)

		_, err := Load(LoadOptions{
			Flags:         newFlagSet(t),
			ConfigFile:    path,
			UserConfigDir: fixedUserConfigDir(t.TempDir()),
		})
		if err == nil || !strings.Contains(err.Error(), "enabled") {
			t.Fatalf("SOT-IF-026: Load() のエラー = %v", err)
		}
	})
}

func TestLoadExplicitEmptyProviderNamespacesRemoveDefaults(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]string{
		"config.yaml": "providers: {}\nproviderRoutes: {}\n",
		"config.json": `{"providers": {}, "providerRoutes": {}}`,
		"config.toml": "[providers]\n[providerRoutes]\n",
	}

	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			writeFile(t, path, content)

			got, err := Load(LoadOptions{
				Flags:         newFlagSet(t),
				ConfigFile:    path,
				UserConfigDir: fixedUserConfigDir(t.TempDir()),
			})
			if err != nil {
				t.Fatalf("SOT-IF-020/SOT-IF-026: Load() のエラー = %v", err)
			}
			if len(got.Providers()) != 0 || len(got.ProviderRoutes()) != 0 {
				t.Fatalf(
					"SOT-IF-026: providers = %v、providerRoutes = %v",
					got.Providers(),
					got.ProviderRoutes(),
				)
			}
		})
	}
}

func TestLoadRejectsInvalidProviderStructures(t *testing.T) {
	clearKnownEnvironment(t)

	longEnvironmentName := "A" + strings.Repeat("B", 128)
	tests := map[string]string{ // #nosec G101 -- SOT-IF-026: 認証情報ではなく credentialEnvRefs の拒否条件を確認する固定テスト値である。
		"providers の型": `
providers: []
`,
		"provider object の型": `
providers:
  custom-provider: true
`,
		"provider の未知フィールド": `
providers:
  custom-provider:
    enabled: true
    unknown: value
`,
		"enabled の欠落": `
providers:
  custom-provider:
    settings: {}
`,
		"enabled の型": `
providers:
  custom-provider:
    enabled: "true"
`,
		"settings の型": `
providers:
  custom-provider:
    enabled: true
    settings: []
`,
		"credentialEnvRefs の型": `
providers:
  custom-provider:
    enabled: true
    credentialEnvRefs: []
`,
		"credential ref の未知フィールド": `
providers:
  custom-provider:
    enabled: true
    credentialEnvRefs:
      apiKey:
        type: env
        name: CUSTOM_API_KEY
        value: fixture
`,
		"credential ref の型": `
providers:
  custom-provider:
    enabled: true
    credentialEnvRefs:
      apiKey: CUSTOM_API_KEY
`,
		"credential type の欠落": `
providers:
  custom-provider:
    enabled: true
    credentialEnvRefs:
      apiKey:
        name: CUSTOM_API_KEY
`,
		"credential type の値": `
providers:
  custom-provider:
    enabled: true
    credentialEnvRefs:
      apiKey:
        type: file
        name: CUSTOM_API_KEY
`,
		"credential name の欠落": `
providers:
  custom-provider:
    enabled: true
    credentialEnvRefs:
      apiKey:
        type: env
`,
		"credential name の先頭": `
providers:
  custom-provider:
    enabled: true
    credentialEnvRefs:
      apiKey:
        type: env
        name: 1CUSTOM_API_KEY
`,
		"credential name の記号": `
providers:
  custom-provider:
    enabled: true
    credentialEnvRefs:
      apiKey:
        type: env
        name: CUSTOM-API-KEY
`,
		"credential name の長さ": fmt.Sprintf(`
providers:
  custom-provider:
    enabled: true
    credentialEnvRefs:
      apiKey:
        type: env
        name: %s
`, longEnvironmentName),
		"providerId の形式": `
providers:
  Custom_Provider:
    enabled: true
`,
	}

	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			writeFile(t, path, content)
			if _, err := Load(LoadOptions{
				Flags:         newFlagSet(t),
				ConfigFile:    path,
				UserConfigDir: fixedUserConfigDir(t.TempDir()),
			}); err == nil {
				t.Fatal("SOT-IF-014/SOT-IF-018/SOT-IF-026: Load() のエラー = nil")
			}
		})
	}
}

func TestLoadRejectsInvalidProviderRouteStructures(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]string{
		"providerRoutes の型": `
providerRoutes: []
`,
		"route object の型": `
providerRoutes:
  law.search@1: primary
`,
		"route の未知フィールド": `
providerRoutes:
  law.search@1:
    selection: primary
    defaultProviderId: custom-provider
    unknown: value
`,
		"route key の形式": `
providerRoutes:
  law.search:
    selection: explicit
`,
		"selection の欠落": `
providerRoutes:
  law.search@1:
    defaultProviderId: custom-provider
`,
		"selection の型": `
providerRoutes:
  law.search@1:
    selection: true
`,
		"selection の値": `
providerRoutes:
  law.search@1:
    selection: fallback
`,
		"explicit と defaultProviderId": `
providerRoutes:
  law.search@1:
    selection: explicit
    defaultProviderId: custom-provider
`,
		"explicit と aggregateProviderIds": `
providerRoutes:
  law.search@1:
    selection: explicit
    aggregateProviderIds: [custom-provider]
`,
		"explicit と rollbackProviderId": `
providerRoutes:
  law.search@1:
    selection: explicit
    rollbackProviderId: custom-provider
`,
		"primary の defaultProviderId 欠落": `
providerRoutes:
  law.search@1:
    selection: primary
`,
		"primary の aggregateProviderIds": `
providerRoutes:
  law.search@1:
    selection: primary
    defaultProviderId: custom-provider
    aggregateProviderIds: [custom-provider]
`,
		"primary の providerId 型": `
providerRoutes:
  law.search@1:
    selection: primary
    defaultProviderId: true
`,
		"aggregate の配列欠落": `
providerRoutes:
  law.search@1:
    selection: aggregate
`,
		"aggregate の空配列": `
providerRoutes:
  law.search@1:
    selection: aggregate
    aggregateProviderIds: []
`,
		"aggregate の型": `
providerRoutes:
  law.search@1:
    selection: aggregate
    aggregateProviderIds: custom-provider
`,
		"aggregate の要素型": `
providerRoutes:
  law.search@1:
    selection: aggregate
    aggregateProviderIds: [custom-provider, true]
`,
		"aggregate の重複": `
providerRoutes:
  law.search@1:
    selection: aggregate
    aggregateProviderIds: [custom-provider, custom-provider]
`,
		"aggregate と defaultProviderId": `
providerRoutes:
  law.search@1:
    selection: aggregate
    defaultProviderId: custom-provider
    aggregateProviderIds: [custom-provider]
`,
		"aggregate と rollbackProviderId": `
providerRoutes:
  law.search@1:
    selection: aggregate
    aggregateProviderIds: [custom-provider]
    rollbackProviderId: custom-provider
`,
	}

	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			writeFile(t, path, content)
			if _, err := Load(LoadOptions{
				Flags:         newFlagSet(t),
				ConfigFile:    path,
				UserConfigDir: fixedUserConfigDir(t.TempDir()),
			}); err == nil {
				t.Fatal("SOT-MODEL-013/SOT-IF-026: Load() のエラー = nil")
			}
		})
	}
}

func TestLoadKeepsProviderSpecificValidationAtCompositionRoot(t *testing.T) {
	clearKnownEnvironment(t)

	path := filepath.Join(t.TempDir(), "config.yaml")
	writeFile(t, path, `
providers:
  unregistered-provider:
    enabled: true
    settings:
      providerSpecificValue:
        nested: true
    credentialEnvRefs:
      providerSpecificSlot:
        type: env
        name: PROVIDER_SPECIFIC_SECRET
providerRoutes:
  provider.unregistered.custom@3:
    selection: primary
    defaultProviderId: unregistered-provider
    rollbackProviderId: another-unregistered-provider
`)

	got, err := Load(LoadOptions{
		Flags:         newFlagSet(t),
		ConfigFile:    path,
		UserConfigDir: fixedUserConfigDir(t.TempDir()),
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-015/SOT-IF-026: generic 構造の Load() のエラー = %v", err)
	}

	provider := got.Providers()["unregistered-provider"]
	if provider.Settings["providerSpecificValue"] == nil ||
		provider.CredentialEnvRefs["providerSpecificSlot"].Name != "PROVIDER_SPECIFIC_SECRET" {
		t.Fatalf("SOT-IF-018: provider 固有設定が保持されていません: %#v", provider)
	}
	route := got.ProviderRoutes()[ProviderRouteKey{
		CapabilityID: "provider.unregistered.custom",
		MajorVersion: 3,
	}]
	if route.DefaultProviderID != "unregistered-provider" ||
		route.RollbackProviderID != "another-unregistered-provider" {
		t.Fatalf("SOT-IF-026: generic route が保持されていません: %#v", route)
	}
}

func validProviderValues() Values {
	return Values{
		Transport:      string(TransportStdio),
		RequestTimeout: 30 * time.Second,
		ListenAddress:  "127.0.0.1:8080",
		AllowedOrigins: make([]string, 0),
	}
}
