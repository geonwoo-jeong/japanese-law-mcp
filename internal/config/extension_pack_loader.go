package config

import (
	"fmt"
	"sort"
)

func decodeExtensionPacks(
	value any,
	present bool,
) (map[string]ExtensionPackConfig, error) {
	if !present {
		return nil, nil
	}
	object, err := requireObject(value, keyExtensionPacks)
	if err != nil {
		return nil, err
	}
	packIDs := make([]string, 0, len(object))
	for packID := range object {
		packIDs = append(packIDs, packID)
	}
	sort.Strings(packIDs)

	packs := make(map[string]ExtensionPackConfig, len(object))
	for _, packID := range packIDs {
		if packID != ExtensionPackJudicialCases &&
			packID != ExtensionPackLegislativeHistory {
			return nil, fmt.Errorf("未知の extension pack %q が指定されています", packID)
		}
		packObject, objectErr := requireObject(
			object[packID],
			"extensionPacks."+packID,
		)
		if objectErr != nil {
			return nil, objectErr
		}
		if err := rejectUnknownFields(
			packObject,
			"extensionPacks."+packID,
			"enabled",
		); err != nil {
			return nil, err
		}
		enabled := false
		if enabledValue, exists := packObject["enabled"]; exists {
			enabled, err = requireExtensionPackEnabled(packID, enabledValue)
			if err != nil {
				return nil, err
			}
		}
		packs[packID] = ExtensionPackConfig{Enabled: enabled}
	}
	return packs, nil
}

func requireExtensionPackEnabled(packID string, value any) (bool, error) {
	enabled, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf(
			"extensionPacks.%s.enabled は boolean でなければなりません",
			packID,
		)
	}
	return enabled, nil
}
