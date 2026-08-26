package config

import (
	"fmt"
	"sort"
)

const (
	keyExtensionPacks = "extensionPacks"

	// ExtensionPackJudicialCases は、裁判例拡張パックの識別子である。
	ExtensionPackJudicialCases = "judicial-cases"
	// ExtensionPackJudicialCitations は、判例引用追跡拡張パックの識別子である。
	ExtensionPackJudicialCitations = "judicial-citations"
	// ExtensionPackLegislativeHistory は、立法過程拡張パックの識別子である。
	ExtensionPackLegislativeHistory = "legislative-history"
)

// ExtensionPackConfig は、一つの拡張パックの起動時設定を表す。
type ExtensionPackConfig struct {
	Enabled bool
}

func resolveExtensionPacks(
	values map[string]ExtensionPackConfig,
) (map[string]ExtensionPackConfig, error) {
	if values == nil {
		return make(map[string]ExtensionPackConfig), nil
	}
	packIDs := make([]string, 0, len(values))
	for packID := range values {
		packIDs = append(packIDs, packID)
	}
	sort.Strings(packIDs)

	resolved := make(map[string]ExtensionPackConfig, len(values))
	for _, packID := range packIDs {
		if packID != ExtensionPackJudicialCases &&
			packID != ExtensionPackJudicialCitations &&
			packID != ExtensionPackLegislativeHistory {
			return nil, fmt.Errorf("未知の extension pack %q が指定されています", packID)
		}
		resolved[packID] = values[packID]
	}
	return resolved, nil
}

func cloneExtensionPacks(
	values map[string]ExtensionPackConfig,
) map[string]ExtensionPackConfig {
	cloned := make(map[string]ExtensionPackConfig, len(values))
	for packID, pack := range values {
		cloned[packID] = pack
	}
	return cloned
}
