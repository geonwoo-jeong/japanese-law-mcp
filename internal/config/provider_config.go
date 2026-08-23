package config

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	keyProviders      = "providers"
	keyProviderRoutes = "providerRoutes"
)

var (
	providerIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	capabilityPattern = regexp.MustCompile(
		`^[a-z0-9]+(?:-[a-z0-9]+)*(?:\.[a-z0-9]+(?:-[a-z0-9]+)*)+$`,
	)
	environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// ProviderRouteSelection は、能力に使用する provider の選択方法を表す。
type ProviderRouteSelection string

const (
	ProviderRouteSelectionExplicit  ProviderRouteSelection = "explicit"
	ProviderRouteSelectionPrimary   ProviderRouteSelection = "primary"
	ProviderRouteSelectionAggregate ProviderRouteSelection = "aggregate"
)

// CredentialEnvRef は、秘密値を読み取る環境変数への参照を表す。
type CredentialEnvRef struct {
	Type string
	Name string
}

// ProviderConfig は、一つの provider の起動時設定を表す。
type ProviderConfig struct {
	Enabled           bool
	Settings          map[string]any
	CredentialEnvRefs map[string]CredentialEnvRef
}

// ProviderRouteKey は、一つの能力と major version を表す。
type ProviderRouteKey struct {
	CapabilityID string
	MajorVersion int
}

// ProviderRoute は、一つの能力に対する provider 選択を表す。
type ProviderRoute struct {
	Selection            ProviderRouteSelection
	DefaultProviderID    string
	AggregateProviderIDs []string
	RollbackProviderID   string
}

// ParseProviderRouteKey は、{capabilityId}@{majorVersion} を型付き key に変換する。
func ParseProviderRouteKey(value string) (ProviderRouteKey, error) {
	capabilityID, versionText, found := strings.Cut(value, "@")
	if !found || strings.Contains(versionText, "@") ||
		!capabilityPattern.MatchString(capabilityID) {
		return ProviderRouteKey{}, fmt.Errorf(
			"providerRoutes の key は {capabilityId}@{majorVersion} 形式でなければなりません",
		)
	}
	majorVersion, err := strconv.Atoi(versionText)
	if err != nil || majorVersion < 1 || versionText != strconv.Itoa(majorVersion) {
		return ProviderRouteKey{}, fmt.Errorf(
			"providerRoutes の majorVersion は 1 以上でなければなりません",
		)
	}
	return ProviderRouteKey{
		CapabilityID: capabilityID,
		MajorVersion: majorVersion,
	}, nil
}

// String は、route key を canonical な文字列へ戻す。
func (k ProviderRouteKey) String() string {
	return k.CapabilityID + "@" + strconv.Itoa(k.MajorVersion)
}

func defaultProviders(
	judicialCasesEnabled bool,
) map[string]ProviderConfig {
	providers := map[string]ProviderConfig{
		"e-gov-law-api-v1": {
			Enabled:           true,
			Settings:          make(map[string]any),
			CredentialEnvRefs: make(map[string]CredentialEnvRef),
		},
		"e-gov-law-api-v2": {
			Enabled:           true,
			Settings:          make(map[string]any),
			CredentialEnvRefs: make(map[string]CredentialEnvRef),
		},
	}
	if judicialCasesEnabled {
		providers["courts-hanrei-html"] = ProviderConfig{
			Enabled:           true,
			Settings:          make(map[string]any),
			CredentialEnvRefs: make(map[string]CredentialEnvRef),
		}
	}
	return providers
}

func defaultProviderRoutes(
	judicialCasesEnabled bool,
) map[ProviderRouteKey]ProviderRoute {
	const (
		courtsProviderID = "courts-hanrei-html"
		v1ProviderID     = "e-gov-law-api-v1"
		v2ProviderID     = "e-gov-law-api-v2"
	)
	routes := map[ProviderRouteKey]ProviderRoute{
		{CapabilityID: "law.article.read", MajorVersion: 1}: {
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: v2ProviderID,
		},
		{CapabilityID: "law.content.search", MajorVersion: 1}: {
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: v2ProviderID,
		},
		{CapabilityID: "law.document.read", MajorVersion: 1}: {
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: v2ProviderID,
		},
		{CapabilityID: "law.version.compare", MajorVersion: 1}: {
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: v2ProviderID,
		},
		{CapabilityID: "law.revision.list", MajorVersion: 1}: {
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: v2ProviderID,
		},
		{CapabilityID: "law.search", MajorVersion: 1}: {
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: v2ProviderID,
		},
		{CapabilityID: "law.update.list", MajorVersion: 1}: {
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: v1ProviderID,
		},
	}
	if judicialCasesEnabled {
		routes[ProviderRouteKey{
			CapabilityID: "judicial-decision.read",
			MajorVersion: 1,
		}] = ProviderRoute{
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: courtsProviderID,
		}
		routes[ProviderRouteKey{
			CapabilityID: "judicial-decision.search",
			MajorVersion: 1,
		}] = ProviderRoute{
			Selection:         ProviderRouteSelectionPrimary,
			DefaultProviderID: courtsProviderID,
		}
	}
	return routes
}

func resolveProviderConfigs(
	values map[string]ProviderConfig,
	judicialCasesEnabled bool,
) (map[string]ProviderConfig, error) {
	if values == nil {
		return cloneProviderConfigs(defaultProviders(judicialCasesEnabled)), nil
	}
	resolved := make(map[string]ProviderConfig)
	if len(values) != 0 {
		resolved = cloneProviderConfigs(defaultProviders(judicialCasesEnabled))
	}
	providerIDs := make([]string, 0, len(values))
	for providerID := range values {
		providerIDs = append(providerIDs, providerID)
	}
	sort.Strings(providerIDs)
	for _, providerID := range providerIDs {
		if !providerIDPattern.MatchString(providerID) {
			return nil, fmt.Errorf(
				"providerId は小文字の ASCII 英数字と内部のハイフンで構成しなければなりません",
			)
		}
		provider := cloneProviderConfig(values[providerID])
		if err := validateCredentialEnvRefs(provider.CredentialEnvRefs); err != nil {
			return nil, fmt.Errorf("provider %q: %w", providerID, err)
		}
		if provider.Settings == nil {
			provider.Settings = make(map[string]any)
		}
		if provider.CredentialEnvRefs == nil {
			provider.CredentialEnvRefs = make(map[string]CredentialEnvRef)
		}
		resolved[providerID] = provider
	}
	return resolved, nil
}

func resolveProviderRoutes(
	values map[string]ProviderRoute,
	judicialCasesEnabled bool,
) (map[ProviderRouteKey]ProviderRoute, error) {
	if values == nil {
		return cloneProviderRoutes(defaultProviderRoutes(judicialCasesEnabled)), nil
	}
	resolved := make(map[ProviderRouteKey]ProviderRoute)
	if len(values) != 0 {
		resolved = cloneProviderRoutes(defaultProviderRoutes(judicialCasesEnabled))
	}
	rawKeys := make([]string, 0, len(values))
	for key := range values {
		rawKeys = append(rawKeys, key)
	}
	sort.Strings(rawKeys)
	for _, rawKey := range rawKeys {
		key, err := ParseProviderRouteKey(rawKey)
		if err != nil {
			return nil, err
		}
		route := cloneProviderRoute(values[rawKey])
		if err := validateProviderRoute(route); err != nil {
			return nil, fmt.Errorf("providerRoutes.%s: %w", rawKey, err)
		}
		resolved[key] = route
	}
	return resolved, nil
}

func validateCredentialEnvRefs(values map[string]CredentialEnvRef) error {
	slots := make([]string, 0, len(values))
	for slot := range values {
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	for _, slot := range slots {
		if slot == "" {
			return fmt.Errorf("credentialEnvRefs の slot は空にできません")
		}
		ref := values[slot]
		if ref.Type != "env" {
			return fmt.Errorf("credentialEnvRefs の type は env でなければなりません")
		}
		if len(ref.Name) == 0 || len(ref.Name) > 128 ||
			!environmentNamePattern.MatchString(ref.Name) {
			return fmt.Errorf("credentialEnvRefs の name が環境変数名の形式ではありません")
		}
	}
	return nil
}

func validateProviderRoute(route ProviderRoute) error {
	if route.DefaultProviderID != "" &&
		!providerIDPattern.MatchString(route.DefaultProviderID) {
		return fmt.Errorf("defaultProviderId が providerId の形式ではありません")
	}
	if route.RollbackProviderID != "" &&
		!providerIDPattern.MatchString(route.RollbackProviderID) {
		return fmt.Errorf("rollbackProviderId が providerId の形式ではありません")
	}
	for _, providerID := range route.AggregateProviderIDs {
		if !providerIDPattern.MatchString(providerID) {
			return fmt.Errorf("aggregateProviderIds に不正な providerId があります")
		}
	}

	switch route.Selection {
	case ProviderRouteSelectionExplicit:
		if route.DefaultProviderID != "" ||
			len(route.AggregateProviderIDs) != 0 ||
			route.RollbackProviderID != "" {
			return fmt.Errorf("explicit route に provider 選択項目を併用できません")
		}
	case ProviderRouteSelectionPrimary:
		if route.DefaultProviderID == "" {
			return fmt.Errorf("primary route には defaultProviderId が必要です")
		}
		if len(route.AggregateProviderIDs) != 0 {
			return fmt.Errorf("primary route に aggregateProviderIds を指定できません")
		}
	case ProviderRouteSelectionAggregate:
		if route.DefaultProviderID != "" || route.RollbackProviderID != "" {
			return fmt.Errorf("aggregate route に primary の項目を指定できません")
		}
		if len(route.AggregateProviderIDs) == 0 {
			return fmt.Errorf("aggregate route には aggregateProviderIds が必要です")
		}
		seen := make(map[string]struct{}, len(route.AggregateProviderIDs))
		for _, providerID := range route.AggregateProviderIDs {
			if _, exists := seen[providerID]; exists {
				return fmt.Errorf("aggregateProviderIds に重複があります")
			}
			seen[providerID] = struct{}{}
		}
	default:
		return fmt.Errorf("selection は explicit、primary または aggregate でなければなりません")
	}
	return nil
}

func cloneProviderConfigs(values map[string]ProviderConfig) map[string]ProviderConfig {
	cloned := make(map[string]ProviderConfig, len(values))
	for providerID, provider := range values {
		cloned[providerID] = cloneProviderConfig(provider)
	}
	return cloned
}

func cloneProviderConfig(value ProviderConfig) ProviderConfig {
	settings := make(map[string]any, len(value.Settings))
	for key, setting := range value.Settings {
		settings[key] = cloneSettingValue(setting)
	}
	refs := make(map[string]CredentialEnvRef, len(value.CredentialEnvRefs))
	for slot, ref := range value.CredentialEnvRefs {
		refs[slot] = ref
	}
	return ProviderConfig{
		Enabled:           value.Enabled,
		Settings:          settings,
		CredentialEnvRefs: refs,
	}
}

func cloneSettingValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, item := range typed {
			cloned[key] = cloneSettingValue(item)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for index, item := range typed {
			cloned[index] = cloneSettingValue(item)
		}
		return cloned
	case []string:
		return append([]string(nil), typed...)
	default:
		return typed
	}
}

func cloneProviderRoutes(
	values map[ProviderRouteKey]ProviderRoute,
) map[ProviderRouteKey]ProviderRoute {
	cloned := make(map[ProviderRouteKey]ProviderRoute, len(values))
	for key, route := range values {
		cloned[key] = cloneProviderRoute(route)
	}
	return cloned
}

func cloneProviderRoute(value ProviderRoute) ProviderRoute {
	value.AggregateProviderIDs = append([]string(nil), value.AggregateProviderIDs...)
	return value
}
