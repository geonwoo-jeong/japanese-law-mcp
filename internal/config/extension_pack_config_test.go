package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultDisablesJudicialCasesExtensionPack(t *testing.T) {
	t.Parallel()

	got := Default()
	if got.JudicialCasesEnabled() {
		t.Fatal("SOT-IF-040: judicial-cases の既定値が有効です")
	}
	if len(got.ExtensionPacks()) != 0 {
		t.Fatalf("SOT-IF-040: extensionPacks = %#v、期待値 = 空", got.ExtensionPacks())
	}
}

func TestDefaultDisablesLegislativeHistoryExtensionPack(t *testing.T) {
	t.Parallel()

	got := Default()
	if got.LegislativeHistoryEnabled() {
		t.Fatal("SOT-IF-061: legislative-history の既定値が有効です")
	}
}

func TestDefaultDisablesJudicialCitationsExtensionPack(t *testing.T) {
	t.Parallel()

	got := Default()
	if got.JudicialCitationsEnabled() {
		t.Fatal("SOT-IF-067: judicial-citations の既定値が有効です")
	}
}

func TestConfigOwnsExtensionPackSnapshot(t *testing.T) {
	t.Parallel()

	values := validProviderValues()
	values.ExtensionPacks = map[string]ExtensionPackConfig{
		ExtensionPackJudicialCases: {Enabled: true},
	}
	got, err := New(values)
	if err != nil {
		t.Fatalf("SOT-ARCH-019: New() のエラー = %v", err)
	}

	delete(values.ExtensionPacks, ExtensionPackJudicialCases)
	snapshot := got.ExtensionPacks()
	delete(snapshot, ExtensionPackJudicialCases)

	if !got.JudicialCasesEnabled() {
		t.Fatal("SOT-ARCH-019: 外部変更により judicial-cases が無効になりました")
	}
}

func TestNewRejectsUnknownExtensionPack(t *testing.T) {
	t.Parallel()

	values := validProviderValues()
	values.ExtensionPacks = map[string]ExtensionPackConfig{
		"unknown-pack": {Enabled: true},
	}
	if _, err := New(values); err == nil {
		t.Fatal("SOT-IF-040: 未知の extension pack を受理しました")
	}
}

func TestNewAddsJudicialCasesConditionalProviderAndRoutes(t *testing.T) {
	t.Parallel()

	values := validProviderValues()
	values.ExtensionPacks = map[string]ExtensionPackConfig{
		ExtensionPackJudicialCases: {Enabled: true},
	}
	got, err := New(values)
	if err != nil {
		t.Fatalf("SOT-IF-040/SOT-IF-046: New() のエラー = %v", err)
	}

	provider, exists := got.Provider("courts-hanrei-html")
	if !exists ||
		!provider.Enabled ||
		len(provider.Settings) != 0 ||
		len(provider.CredentialEnvRefs) != 0 {
		t.Fatalf("SOT-IF-046: courts-hanrei-html = %#v, %t", provider, exists)
	}
	for _, capabilityID := range []string{
		"judicial-decision.read",
		"judicial-decision.search",
	} {
		key := ProviderRouteKey{CapabilityID: capabilityID, MajorVersion: 1}
		route, routeExists := got.ProviderRoute(key)
		if !routeExists ||
			route.Selection != ProviderRouteSelectionPrimary ||
			route.DefaultProviderID != "courts-hanrei-html" {
			t.Fatalf("SOT-IF-046: route %s = %#v, %t", key, route, routeExists)
		}
	}
	if len(got.Providers()) != 3 || len(got.ProviderRoutes()) != 9 {
		t.Fatalf(
			"SOT-IF-046: providers = %d, routes = %d",
			len(got.Providers()),
			len(got.ProviderRoutes()),
		)
	}
}

func TestNewAddsLegislativeHistoryConditionalProviderAndRoutes(t *testing.T) {
	t.Parallel()

	values := validProviderValues()
	values.ExtensionPacks = map[string]ExtensionPackConfig{
		ExtensionPackLegislativeHistory: {Enabled: true},
	}
	got, err := New(values)
	if err != nil {
		t.Fatalf("SOT-IF-061/SOT-IF-065: New() のエラー = %v", err)
	}

	provider, exists := got.Provider("ndl-diet-speech-api")
	if !exists ||
		!provider.Enabled ||
		len(provider.Settings) != 0 ||
		len(provider.CredentialEnvRefs) != 0 {
		t.Fatalf("SOT-IF-065: ndl-diet-speech-api = %#v, %t", provider, exists)
	}
	key := ProviderRouteKey{CapabilityID: "parliament.speech.search", MajorVersion: 1}
	route, routeExists := got.ProviderRoute(key)
	if !routeExists ||
		route.Selection != ProviderRouteSelectionPrimary ||
		route.DefaultProviderID != "ndl-diet-speech-api" {
		t.Fatalf("SOT-IF-065: route %s = %#v, %t", key, route, routeExists)
	}
	if len(got.Providers()) != 3 || len(got.ProviderRoutes()) != 8 {
		t.Fatalf(
			"SOT-IF-065: providers = %d, routes = %d",
			len(got.Providers()),
			len(got.ProviderRoutes()),
		)
	}
}

func TestNewAddsJudicialCitationsConditionalProviderAndRoutes(t *testing.T) {
	t.Parallel()

	values := validProviderValues()
	values.ExtensionPacks = map[string]ExtensionPackConfig{
		ExtensionPackJudicialCases:     {Enabled: true},
		ExtensionPackJudicialCitations: {Enabled: true},
	}
	got, err := New(values)
	if err != nil {
		t.Fatalf("SOT-IF-067/SOT-IF-074: New() のエラー = %v", err)
	}

	provider, exists := got.Provider("courts-hanrei-pdf")
	if !exists ||
		!provider.Enabled ||
		len(provider.Settings) != 0 ||
		len(provider.CredentialEnvRefs) != 0 {
		t.Fatalf("SOT-IF-074: courts-hanrei-pdf = %#v, %t", provider, exists)
	}
	for _, capabilityID := range []string{
		"judicial-decision.case-citation.extract",
		"judicial-decision.citing-candidate.search",
	} {
		key := ProviderRouteKey{CapabilityID: capabilityID, MajorVersion: 1}
		route, routeExists := got.ProviderRoute(key)
		if !routeExists {
			t.Fatalf("SOT-IF-074: route %s がありません", key)
		}
		switch capabilityID {
		case "judicial-decision.case-citation.extract":
			if route.DefaultProviderID != "courts-hanrei-pdf" {
				t.Fatalf("SOT-IF-074: route %s = %#v", key, route)
			}
		case "judicial-decision.citing-candidate.search":
			if route.DefaultProviderID != "courts-hanrei-html" {
				t.Fatalf("SOT-IF-074: route %s = %#v", key, route)
			}
		}
		if route.Selection != ProviderRouteSelectionPrimary {
			t.Fatalf("SOT-IF-074: route %s = %#v", key, route)
		}
	}
	if len(got.Providers()) != 4 || len(got.ProviderRoutes()) != 11 {
		t.Fatalf(
			"SOT-IF-067/SOT-IF-074: providers = %d, routes = %d",
			len(got.Providers()),
			len(got.ProviderRoutes()),
		)
	}
}

func TestNewResolvesAllValidExtensionPackCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		judicialCases     bool
		judicialCitations bool
		legislative       bool
		providerCount     int
		routeCount        int
	}{
		{name: "全 pack 無効", providerCount: 2, routeCount: 7},
		{name: "立法過程", legislative: true, providerCount: 3, routeCount: 8},
		{name: "裁判例", judicialCases: true, providerCount: 3, routeCount: 9},
		{name: "裁判例と立法過程", judicialCases: true, legislative: true, providerCount: 4, routeCount: 10},
		{name: "裁判例と引用", judicialCases: true, judicialCitations: true, providerCount: 4, routeCount: 11},
		{name: "全 pack 有効", judicialCases: true, judicialCitations: true, legislative: true, providerCount: 5, routeCount: 12},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			values := validProviderValues()
			values.ExtensionPacks = map[string]ExtensionPackConfig{
				ExtensionPackJudicialCases:      {Enabled: test.judicialCases},
				ExtensionPackJudicialCitations:  {Enabled: test.judicialCitations},
				ExtensionPackLegislativeHistory: {Enabled: test.legislative},
			}
			got, err := New(values)
			if err != nil {
				t.Fatalf("SOT-IF-067: New() のエラー = %v", err)
			}
			if len(got.Providers()) != test.providerCount ||
				len(got.ProviderRoutes()) != test.routeCount {
				t.Fatalf(
					"SOT-IF-067: providers = %d, routes = %d",
					len(got.Providers()),
					len(got.ProviderRoutes()),
				)
			}
		})
	}
}

func TestNewRejectsJudicialCitationsWithoutJudicialCases(t *testing.T) {
	t.Parallel()

	values := validProviderValues()
	values.ExtensionPacks = map[string]ExtensionPackConfig{
		ExtensionPackJudicialCitations: {Enabled: true},
	}
	if _, err := New(values); err == nil {
		t.Fatal("SOT-IF-067: judicial-citations 単独有効化を受理しました")
	}
}

func TestNewRejectsJudicialCitationConfigurationWhilePackDisabled(t *testing.T) {
	t.Parallel()

	tests := map[string]func(Values) Values{
		"PDF provider": func(values Values) Values {
			values.Providers = map[string]ProviderConfig{
				"courts-hanrei-pdf": {Enabled: true},
			}
			return values
		},
		"引用抽出 route": func(values Values) Values {
			values.ProviderRoutes = map[string]ProviderRoute{
				"judicial-decision.case-citation.extract@1": {
					Selection:         ProviderRouteSelectionPrimary,
					DefaultProviderID: "courts-hanrei-pdf",
				},
			}
			return values
		},
		"候補検索 route": func(values Values) Values {
			values.ProviderRoutes = map[string]ProviderRoute{
				"judicial-decision.citing-candidate.search@1": {
					Selection:         ProviderRouteSelectionPrimary,
					DefaultProviderID: "courts-hanrei-html",
				},
			}
			return values
		},
	}
	for name, change := range tests {
		name, change := name, change
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			values := change(validProviderValues())
			if _, err := New(values); err == nil {
				t.Fatalf("SOT-IF-074: judicial-citations 無効時の明示的な %s を受理しました", name)
			}
		})
	}
}

func TestNewAcceptsJudicialCitationOverridesWhilePackEnabled(t *testing.T) {
	t.Parallel()

	values := validProviderValues()
	values.ExtensionPacks = map[string]ExtensionPackConfig{
		ExtensionPackJudicialCases:     {Enabled: true},
		ExtensionPackJudicialCitations: {Enabled: true},
	}
	values.Providers = map[string]ProviderConfig{
		"courts-hanrei-pdf": {Enabled: false},
	}
	values.ProviderRoutes = map[string]ProviderRoute{
		"judicial-decision.case-citation.extract@1": {
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: "courts-hanrei-pdf",
		},
	}
	got, err := New(values)
	if err != nil {
		t.Fatalf("SOT-IF-074: 有効 pack の明示設定を拒否しました: %v", err)
	}
	provider, exists := got.Provider("courts-hanrei-pdf")
	if !exists || provider.Enabled {
		t.Fatalf("SOT-IF-026/SOT-IF-074: PDF provider override = %#v, %t", provider, exists)
	}
}

func TestNewAppliesUserOverrideToConditionalProvider(t *testing.T) {
	t.Parallel()

	values := validProviderValues()
	values.ExtensionPacks = map[string]ExtensionPackConfig{
		ExtensionPackJudicialCases: {Enabled: true},
	}
	values.Providers = map[string]ProviderConfig{
		"courts-hanrei-html": {Enabled: false},
	}
	got, err := New(values)
	if err != nil {
		t.Fatalf("SOT-IF-026/SOT-IF-046: New() のエラー = %v", err)
	}
	provider, exists := got.Provider("courts-hanrei-html")
	if !exists || provider.Enabled {
		t.Fatalf("SOT-IF-026: provider override = %#v, %t", provider, exists)
	}
}

func TestLoadJudicialCasesExtensionPackFormats(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]string{
		"config.yaml": `
extensionPacks:
  judicial-cases:
    enabled: true
`,
		"config.json": `{
  "extensionPacks": {
    "judicial-cases": {
      "enabled": true
    }
  }
}`,
		"config.toml": `
[extensionPacks.judicial-cases]
enabled = true
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
				t.Fatalf("SOT-IF-039/SOT-IF-040: Load() のエラー = %v", err)
			}
			if !got.JudicialCasesEnabled() {
				t.Fatal("SOT-IF-040: judicial-cases が有効ではありません")
			}
		})
	}
}

func TestLoadLegislativeHistoryExtensionPackFormats(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]string{
		"config.yaml": `
extensionPacks:
  legislative-history:
    enabled: true
`,
		"config.json": `{
  "extensionPacks": {
    "legislative-history": {
      "enabled": true
    }
  }
}`,
		"config.toml": `
[extensionPacks.legislative-history]
enabled = true
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
				t.Fatalf("SOT-IF-039/SOT-IF-061: Load() のエラー = %v", err)
			}
			if !got.LegislativeHistoryEnabled() {
				t.Fatal("SOT-IF-061: legislative-history が有効ではありません")
			}
		})
	}
}

func TestLoadJudicialCitationsExtensionPackFormats(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]string{
		"config.yaml": `
extensionPacks:
  judicial-cases:
    enabled: true
  judicial-citations:
    enabled: true
`,
		"config.json": `{
  "extensionPacks": {
    "judicial-cases": {
      "enabled": true
    },
    "judicial-citations": {
      "enabled": true
    }
  }
}`,
		"config.toml": `
[extensionPacks.judicial-cases]
enabled = true

[extensionPacks.judicial-citations]
enabled = true
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
				t.Fatalf("SOT-IF-039/SOT-IF-067: Load() のエラー = %v", err)
			}
			if !got.JudicialCasesEnabled() || !got.JudicialCitationsEnabled() {
				t.Fatal("SOT-IF-067: judicial-citations 構成が有効ではありません")
			}
		})
	}
}

func TestLoadDefaultsMissingOrFalseJudicialCasesToDisabled(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]string{
		"extensionPacks 省略": "",
		"空の extensionPacks": `
extensionPacks: {}
`,
		"judicial-cases の空 object": `
extensionPacks:
  judicial-cases: {}
`,
		"enabled false": `
extensionPacks:
  judicial-cases:
    enabled: false
`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			options := LoadOptions{
				Flags:         newFlagSet(t),
				UserConfigDir: fixedUserConfigDir(t.TempDir()),
			}
			if content != "" {
				path := filepath.Join(t.TempDir(), "config.yaml")
				writeFile(t, path, content)
				options.ConfigFile = path
			}

			got, err := Load(options)
			if err != nil {
				t.Fatalf("SOT-IF-040: Load() のエラー = %v", err)
			}
			if got.JudicialCasesEnabled() {
				t.Fatal("SOT-IF-040: judicial-cases が有効です")
			}
		})
	}
}

func TestLoadRejectsInvalidExtensionPackStructures(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]string{
		"extensionPacks の型": `
extensionPacks: []
`,
		"extensionPacks の null": `
extensionPacks: null
`,
		"未知の pack": `
extensionPacks:
  unknown-pack:
    enabled: true
`,
		"pack object の型": `
extensionPacks:
  judicial-cases: true
`,
		"pack object の null": `
extensionPacks:
  judicial-cases: null
`,
		"未知の pack 設定": `
extensionPacks:
  judicial-cases:
    enabled: true
    unknown: true
`,
		"enabled の型": `
extensionPacks:
  judicial-cases:
    enabled: "true"
`,
		"enabled の null": `
extensionPacks:
  judicial-cases:
    enabled: null
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
				t.Fatal("SOT-IF-039/SOT-IF-040: 不正な extensionPacks を受理しました")
			}
		})
	}
}

func TestLoadRejectsInvalidExtensionPackStructuresAcrossFormats(t *testing.T) {
	clearKnownEnvironment(t)

	tests := map[string]struct {
		filename string
		content  string
	}{
		"JSON の型": {
			filename: "config.json",
			content:  `{"extensionPacks":{"judicial-cases":{"enabled":"true"}}}`,
		},
		"JSON の null": {
			filename: "config.json",
			content:  `{"extensionPacks":{"judicial-cases":{"enabled":null}}}`,
		},
		"JSON の未知項目": {
			filename: "config.json",
			content:  `{"extensionPacks":{"judicial-cases":{"enabled":true,"unknown":true}}}`,
		},
		"TOML の型": {
			filename: "config.toml",
			content: `
[extensionPacks.judicial-cases]
enabled = "true"
`,
		},
		"TOML の未知項目": {
			filename: "config.toml",
			content: `
[extensionPacks.judicial-cases]
enabled = true
unknown = true
`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), test.filename)
			writeFile(t, path, test.content)

			if _, err := Load(LoadOptions{
				Flags:         newFlagSet(t),
				ConfigFile:    path,
				UserConfigDir: fixedUserConfigDir(t.TempDir()),
			}); err == nil {
				t.Fatal("SOT-IF-039/SOT-IF-040: 不正な extensionPacks を受理しました")
			}
		})
	}
}

func TestLoadDoesNotReadExtensionPacksFromEnvironment(t *testing.T) {
	clearKnownEnvironment(t)
	t.Setenv(
		"JAPANESE_LAW_MCP_EXTENSION_PACKS",
		`{"judicial-cases":{"enabled":true}}`,
	)

	got, err := Load(LoadOptions{
		Flags:         newFlagSet(t),
		UserConfigDir: fixedUserConfigDir(t.TempDir()),
	})
	if err != nil {
		t.Fatalf("SOT-IF-039: Load() のエラー = %v", err)
	}
	if got.JudicialCasesEnabled() {
		t.Fatal("SOT-IF-039: 環境変数から extensionPacks を有効にしました")
	}
}
