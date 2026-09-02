package provideronboarding

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

func validateBootstrapChanges(paths []string, rows []matrixRow) error {
	for _, changedPath := range paths {
		if !bootstrapPathAllowed(changedPath) {
			return fmt.Errorf("初回導入の許可範囲外の変更です: %s", changedPath)
		}
	}
	if len(rows) == 0 {
		return fmt.Errorf("初回 provider matrix に row がありません")
	}
	for _, row := range rows {
		if row.status != "planned" {
			return fmt.Errorf(
				"初回 provider matrix の row は planned でなければなりません: providerId=%s status=%s",
				row.providerID,
				row.status,
			)
		}
	}
	return nil
}

func bootstrapPathAllowed(changedPath string) bool {
	for _, prefix := range []string{
		"conformance",
		"internal/providerconformance",
		"internal/provideronboarding",
		"cmd/provider-onboarding-fit",
		"internal/githook",
	} {
		if hasPathPrefix(changedPath, prefix) {
			return true
		}
	}
	switch changedPath {
	case ".github/workflows/quality.yml",
		"go.mod",
		"go.sum",
		"wiki/10-implementation-status.md":
		return true
	default:
		return false
	}
}

func validateNormalChanges(paths []string, rows []matrixRow) error {
	_, err := evaluateNormalChanges(paths, rows)
	return err
}

func evaluateNormalChanges(
	paths []string,
	rows []matrixRow,
) (bool, error) {
	return evaluateNormalChangesWithTraceOnlyMatrices(paths, rows, nil)
}

func evaluateNormalChangesWithTraceOnlyMatrices(
	paths []string,
	rows []matrixRow,
	traceOnlyMatrixPaths map[string]struct{},
) (bool, error) {
	packages, err := collectProviderPackages(rows)
	if err != nil {
		return false, err
	}
	if err := validateTraceOnlyMatrixIsolation(paths, traceOnlyMatrixPaths); err != nil {
		return false, err
	}
	matrixTargets := make(map[string]struct{})
	sourceTargets := make(map[string]struct{})
	unknownSourcePaths := make([]string, 0)
	infrastructureChanged := false
	for _, changedPath := range paths {
		if providerConformanceInfrastructurePath(changedPath) {
			infrastructureChanged = true
		}
		_, traceOnlyMatrix := traceOnlyMatrixPaths[changedPath]
		if providerID, ok := matrixProviderID(changedPath); ok && !traceOnlyMatrix {
			matrixTargets[providerID] = struct{}{}
		}
		if !hasPathPrefix(changedPath, "internal/source") {
			continue
		}
		owner, found := providerPackageOwner(changedPath, packages)
		if !found {
			unknownSourcePaths = append(unknownSourcePaths, changedPath)
			continue
		}
		sourceTargets[owner.providerID] = struct{}{}
	}
	if len(unknownSourcePaths) != 0 {
		sort.Strings(unknownSourcePaths)
		return false, fmt.Errorf(
			"provider-neutral または未登録の source package を同時に変更できません: %s",
			unknownSourcePaths[0],
		)
	}
	targets := matrixTargets
	matrixScoped := len(targets) != 0
	if !matrixScoped {
		targets = sourceTargets
	}
	if len(targets) == 0 {
		for _, changedPath := range paths {
			if providerControlPath(changedPath) {
				return false, fmt.Errorf(
					"provider 制御変更には対象 provider の matrix 変更が必要です: %s",
					changedPath,
				)
			}
			if infrastructureChanged && commonContractPath(changedPath) {
				return false, fmt.Errorf(
					"共通 model または capability の変更を分離してください: %s",
					changedPath,
				)
			}
		}
		if infrastructureChanged {
			return true, nil
		}
		return false, nil
	}
	allowedTargets, adoptedGroup := adoptedProviderTargetGroup(targets, matrixScoped)
	if !adoptedGroup && len(targets) != 1 {
		return false, fmt.Errorf(
			"一つの provider 変更に複数の provider が含まれます: %s",
			strings.Join(sortedSet(targets), ", "),
		)
	}
	if len(allowedTargets) == 0 {
		allowedTargets = sortedSet(targets)
	}
	for _, target := range allowedTargets {
		if !providerKnown(target, packages) {
			return false, fmt.Errorf(
				"matrix の providerId に対応する provider package がありません: %s",
				target,
			)
		}
	}
	for _, changedPath := range paths {
		if commonContractPath(changedPath) {
			return false, fmt.Errorf(
				"共通 model または capability の変更を分離してください: %s",
				changedPath,
			)
		}
		owner, found := providerPackageOwner(changedPath, packages)
		if found && !containsString(allowedTargets, owner.providerID) {
			return false, fmt.Errorf(
				"対象外 provider package の変更です: %s",
				changedPath,
			)
		}
	}
	return true, nil
}

func validateTraceOnlyMatrixIsolation(
	paths []string,
	traceOnlyMatrixPaths map[string]struct{},
) error {
	if len(traceOnlyMatrixPaths) == 0 {
		return nil
	}
	changed := make(map[string]struct{}, len(paths))
	for _, changedPath := range paths {
		changed[changedPath] = struct{}{}
	}
	for traceOnlyPath := range traceOnlyMatrixPaths {
		if _, exists := changed[traceOnlyPath]; !exists {
			return fmt.Errorf(
				"変更されていない matrix が SOT 参照更新に分類されました: %s",
				traceOnlyPath,
			)
		}
		if _, matrixPath := matrixProviderID(traceOnlyPath); !matrixPath {
			return fmt.Errorf(
				"provider matrix ではない path が SOT 参照更新に分類されました: %s",
				traceOnlyPath,
			)
		}
	}
	for _, changedPath := range paths {
		if _, traceOnly := traceOnlyMatrixPaths[changedPath]; traceOnly {
			continue
		}
		if hasPathPrefix(changedPath, "internal/source") ||
			providerControlPath(changedPath) ||
			commonContractPath(changedPath) ||
			providerConformanceInfrastructurePath(changedPath) {
			return fmt.Errorf(
				"matrix の SOT 参照更新を provider または conformance 変更と混在させられません: %s",
				changedPath,
			)
		}
	}
	return nil
}

func providerConformanceInfrastructurePath(changedPath string) bool {
	for _, prefix := range []string{
		"conformance",
		"internal/providerconformance",
		"internal/provideronboarding",
		"cmd/provider-onboarding-fit",
	} {
		if hasPathPrefix(changedPath, prefix) {
			return true
		}
	}
	return changedPath == ".github/workflows/quality.yml"
}

func matrixProviderID(changedPath string) (string, bool) {
	const prefix = "conformance/providers/"
	if !strings.HasPrefix(changedPath, prefix) ||
		strings.Contains(strings.TrimPrefix(changedPath, prefix), "/") ||
		path.Ext(changedPath) != ".yaml" {
		return "", false
	}
	providerID := strings.TrimSuffix(path.Base(changedPath), ".yaml")
	if providerID == "" {
		return "", false
	}
	return providerID, true
}

func commonContractPath(changedPath string) bool {
	for _, prefix := range []string{
		"internal/model",
		"internal/capability",
		"internal/application/capability",
		"internal/application/ports",
		"internal/application/judicialdecisionread",
		"internal/application/judicialdecisionsearch",
		"internal/application/lawarticleread",
		"internal/application/lawcontentsearch",
		"internal/application/lawdocumentread",
		"internal/application/lawsearch",
		"internal/application/lawupdatelist",
		"sot/20-model",
	} {
		if hasPathPrefix(changedPath, prefix) {
			return true
		}
	}
	return hasPathPrefix(changedPath, "sot/40-interfaces") &&
		strings.Contains(strings.ToLower(path.Base(changedPath)), "capability")
}

func providerControlPath(changedPath string) bool {
	switch changedPath {
	case "internal/model/provider_descriptor.go",
		"internal/application/provider_binding_registry.go",
		"internal/application/provider_registry.go",
		"internal/application/provider_routes.go",
		"internal/application/composition_root.go",
		"internal/config/provider_config.go":
		return true
	default:
		return false
	}
}

func hasPathPrefix(value, prefix string) bool {
	return value == prefix || strings.HasPrefix(value, prefix+"/")
}

func providerKnown(providerID string, packages []providerPackage) bool {
	for _, providerPackage := range packages {
		if providerPackage.providerID == providerID {
			return true
		}
	}
	return false
}

func adoptedProviderTargetGroup(
	targets map[string]struct{},
	matrixScoped bool,
) ([]string, bool) {
	if !matrixScoped {
		return nil, false
	}
	for _, group := range [][]string{
		{"courts-hanrei-html", "courts-hanrei-pdf"},
	} {
		if len(targets) != len(group) {
			continue
		}
		matched := true
		for _, target := range group {
			if _, exists := targets[target]; !exists {
				matched = false
				break
			}
		}
		if matched {
			return append([]string(nil), group...), true
		}
	}
	return nil, false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sortedSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
