package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

type providerFileValues struct {
	providers      map[string]ProviderConfig
	providerRoutes map[string]ProviderRoute
}

func loadProviderFileValues(path string) (providerFileValues, error) {
	if path == "" {
		return providerFileValues{}, nil
	}
	data, err := os.ReadFile(path) //nolint:gosec // SOT-IF-020: 利用者が明示した設定ファイルを読み取るための入力境界であり、内容は直後に形式と項目を検証する。
	if err != nil {
		return providerFileValues{}, fmt.Errorf("設定ファイルを読み込めません")
	}
	document, err := decodeConfigDocument(filepath.Ext(path), data)
	if err != nil {
		return providerFileValues{}, fmt.Errorf("設定ファイルの構造を解釈できません: %w", err)
	}
	if err := validateTopLevelDocument(document); err != nil {
		return providerFileValues{}, err
	}

	providersValue, providersPresent := findDocumentValue(document, keyProviders)
	providers, err := decodeProviders(providersValue, providersPresent)
	if err != nil {
		return providerFileValues{}, err
	}
	routesValue, routesPresent := findDocumentValue(document, keyProviderRoutes)
	routes, err := decodeProviderRoutes(routesValue, routesPresent)
	if err != nil {
		return providerFileValues{}, err
	}
	return providerFileValues{
		providers:      providers,
		providerRoutes: routes,
	}, nil
}

func decodeConfigDocument(extension string, data []byte) (map[string]any, error) {
	document := make(map[string]any)
	switch strings.ToLower(extension) {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &document); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(data, &document); err != nil {
			return nil, err
		}
	case ".toml":
		if err := toml.Unmarshal(data, &document); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("対応していない設定ファイル形式です")
	}
	if document == nil {
		return nil, fmt.Errorf("最上位は object でなければなりません")
	}
	return document, nil
}

func validateTopLevelDocument(document map[string]any) error {
	keys := make([]string, 0, len(document))
	for key := range document {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	seen := make(map[string]string, len(keys))
	for _, key := range keys {
		canonical := canonicalConfigKey(key)
		if canonical == "" {
			return fmt.Errorf("未知の設定項目 %q が指定されています", key)
		}
		if previous, exists := seen[canonical]; exists {
			return fmt.Errorf(
				"設定項目 %q と %q は同じ項目を重複して指定しています",
				previous,
				key,
			)
		}
		seen[canonical] = key
	}
	return nil
}

func findDocumentValue(document map[string]any, canonical string) (any, bool) {
	for key, value := range document {
		if strings.EqualFold(key, canonical) {
			return value, true
		}
	}
	return nil, false
}

func decodeProviders(value any, present bool) (map[string]ProviderConfig, error) {
	if !present {
		return nil, nil
	}
	object, err := requireObject(value, keyProviders)
	if err != nil {
		return nil, err
	}
	providers := make(map[string]ProviderConfig, len(object))
	for providerID, providerValue := range object {
		providerObject, objectErr := requireObject(
			providerValue,
			"providers."+providerID,
		)
		if objectErr != nil {
			return nil, objectErr
		}
		if err := rejectUnknownFields(
			providerObject,
			"providers."+providerID,
			"enabled",
			"settings",
			"credentialEnvRefs",
		); err != nil {
			return nil, err
		}
		enabledValue, exists := providerObject["enabled"]
		if !exists {
			return nil, fmt.Errorf("providers.%s.enabled は必須です", providerID)
		}
		enabled, ok := enabledValue.(bool)
		if !ok {
			return nil, fmt.Errorf("providers.%s.enabled は boolean でなければなりません", providerID)
		}
		settings := make(map[string]any)
		if settingsValue, settingsExist := providerObject["settings"]; settingsExist {
			settings, err = requireObject(settingsValue, "providers."+providerID+".settings")
			if err != nil {
				return nil, err
			}
		}
		refs := make(map[string]CredentialEnvRef)
		if refsValue, refsExist := providerObject["credentialEnvRefs"]; refsExist {
			refs, err = decodeCredentialEnvRefs(providerID, refsValue)
			if err != nil {
				return nil, err
			}
		}
		providers[providerID] = ProviderConfig{
			Enabled:           enabled,
			Settings:          settings,
			CredentialEnvRefs: refs,
		}
	}
	return providers, nil
}

func decodeCredentialEnvRefs(
	providerID string,
	value any,
) (map[string]CredentialEnvRef, error) {
	object, err := requireObject(
		value,
		"providers."+providerID+".credentialEnvRefs",
	)
	if err != nil {
		return nil, err
	}
	refs := make(map[string]CredentialEnvRef, len(object))
	for slot, refValue := range object {
		refObject, objectErr := requireObject(
			refValue,
			"providers."+providerID+".credentialEnvRefs."+slot,
		)
		if objectErr != nil {
			return nil, objectErr
		}
		if err := rejectUnknownFields(
			refObject,
			"providers."+providerID+".credentialEnvRefs."+slot,
			"type",
			"name",
		); err != nil {
			return nil, err
		}
		typeValue, typeExists := refObject["type"]
		nameValue, nameExists := refObject["name"]
		refType, typeOK := typeValue.(string)
		name, nameOK := nameValue.(string)
		if !typeExists || !typeOK {
			return nil, fmt.Errorf("credentialEnvRefs の type は string で必須です")
		}
		if !nameExists || !nameOK {
			return nil, fmt.Errorf("credentialEnvRefs の name は string で必須です")
		}
		refs[slot] = CredentialEnvRef{Type: refType, Name: name}
	}
	return refs, nil
}

func decodeProviderRoutes(value any, present bool) (map[string]ProviderRoute, error) {
	if !present {
		return nil, nil
	}
	object, err := requireObject(value, keyProviderRoutes)
	if err != nil {
		return nil, err
	}
	routes := make(map[string]ProviderRoute, len(object))
	for rawKey, routeValue := range object {
		routeObject, objectErr := requireObject(
			routeValue,
			"providerRoutes."+rawKey,
		)
		if objectErr != nil {
			return nil, objectErr
		}
		if err := rejectUnknownFields(
			routeObject,
			"providerRoutes."+rawKey,
			"selection",
			"defaultProviderId",
			"aggregateProviderIds",
			"rollbackProviderId",
		); err != nil {
			return nil, err
		}
		selectionValue, exists := routeObject["selection"]
		selection, ok := selectionValue.(string)
		if !exists || !ok {
			return nil, fmt.Errorf(
				"providerRoutes.%s.selection は string で必須です",
				rawKey,
			)
		}
		defaultProviderID, err := optionalStringField(
			routeObject,
			"defaultProviderId",
			"providerRoutes."+rawKey,
		)
		if err != nil {
			return nil, err
		}
		rollbackProviderID, err := optionalStringField(
			routeObject,
			"rollbackProviderId",
			"providerRoutes."+rawKey,
		)
		if err != nil {
			return nil, err
		}
		aggregateProviderIDs, err := optionalStringSliceField(
			routeObject,
			"aggregateProviderIds",
			"providerRoutes."+rawKey,
		)
		if err != nil {
			return nil, err
		}
		routes[rawKey] = ProviderRoute{
			Selection:            ProviderRouteSelection(selection),
			DefaultProviderID:    defaultProviderID,
			AggregateProviderIDs: aggregateProviderIDs,
			RollbackProviderID:   rollbackProviderID,
		}
	}
	return routes, nil
}

func requireObject(value any, path string) (map[string]any, error) {
	object, ok := value.(map[string]any)
	if !ok || object == nil {
		return nil, fmt.Errorf("%s は object でなければなりません", path)
	}
	return cloneStringMap(object), nil
}

func cloneStringMap(value map[string]any) map[string]any {
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = cloneSettingValue(item)
	}
	return cloned
}

func rejectUnknownFields(object map[string]any, path string, allowed ...string) error {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, field := range allowed {
		allowedSet[field] = struct{}{}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, exists := allowedSet[key]; !exists {
			return fmt.Errorf("%s に未知の項目 %q があります", path, key)
		}
	}
	return nil
}

func optionalStringField(
	object map[string]any,
	field string,
	path string,
) (string, error) {
	value, exists := object[field]
	if !exists {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s.%s は string でなければなりません", path, field)
	}
	return text, nil
}

func optionalStringSliceField(
	object map[string]any,
	field string,
	path string,
) ([]string, error) {
	value, exists := object[field]
	if !exists {
		return nil, nil
	}
	switch values := value.(type) {
	case []string:
		return append([]string(nil), values...), nil
	case []any:
		result := make([]string, 0, len(values))
		for _, item := range values {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf(
					"%s.%s は string の配列でなければなりません",
					path,
					field,
				)
			}
			result = append(result, text)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("%s.%s は配列でなければなりません", path, field)
	}
}
