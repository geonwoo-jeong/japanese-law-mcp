package provideronboarding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SOT-ENG-044: 初回導入は列挙された基盤成果物だけを許可する。
func TestValidateBootstrapChangesAllowsOnlyInitialAllowlist(t *testing.T) {
	t.Parallel()

	allowed := []string{
		"conformance/provider-capability.schema.json",
		"conformance/providers/e-gov-law-api-v2.yaml",
		"internal/providerconformance/loader.go",
		"internal/provideronboarding/run.go",
		"cmd/provider-onboarding-fit/main.go",
		"internal/githook/quality_gate.go",
		".github/workflows/quality.yml",
		"go.mod",
		"go.sum",
		"wiki/10-implementation-status.md",
	}
	if err := validateBootstrapChanges(allowed, []matrixRow{{status: "planned"}}); err != nil {
		t.Fatalf("初回導入の許可パスが拒否されました: %v", err)
	}

	rejected := []string{
		"internal/source/egov/lawv2/adapter.go",
		"internal/source/egov/lawv2/fixtures/search.json",
		"internal/model/provider_descriptor.go",
		"internal/application/provider_routes.go",
		"README.md",
	}
	for _, path := range rejected {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			err := validateBootstrapChanges(
				append(append([]string(nil), allowed...), path),
				[]matrixRow{{status: "planned"}},
			)
			if err == nil {
				t.Fatalf("初回導入の禁止パス %q が許可されました", path)
			}
			if !strings.Contains(err.Error(), path) {
				t.Fatalf("禁止パスを特定できないエラーです: %v", err)
			}
		})
	}
}

func TestValidateBootstrapChangesRequiresAtLeastOnePlannedRow(t *testing.T) {
	t.Parallel()

	paths := []string{"conformance/provider-capability.schema.json"}
	for _, rows := range [][]matrixRow{
		nil,
		{{status: "implemented"}},
		{{status: "planned"}, {status: "retired"}},
	} {
		if err := validateBootstrapChanges(paths, rows); err == nil {
			t.Fatalf("初回 matrix の不正な行が許可されました: %#v", rows)
		}
	}
}

// SOT-ENG-044: provider 変更と共通契約の変更を同じ変更単位に混在させない。
func TestValidateNormalChangesRejectsCommonContractChanges(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{
		{
			providerID:    "provider-a",
			implementedBy: "internal/source/a/v1",
			status:        "planned",
		},
	}
	for _, forbidden := range []string{
		"internal/model/provider_descriptor.go",
		"internal/capability/lawsearch/port.go",
		"internal/application/judicialdecisionread/port.go",
		"internal/application/judicialdecisionsearch/port.go",
		"internal/application/lawarticleread/port.go",
		"internal/application/lawcontentsearch/port.go",
		"internal/application/lawdocumentread/port.go",
		"internal/application/lawsearch/port.go",
		"internal/application/lawupdatelist/port.go",
		"sot/20-model/13-provider-capability.md",
		"sot/40-interfaces/22-law-search-capability.md",
	} {
		forbidden := forbidden
		t.Run(forbidden, func(t *testing.T) {
			t.Parallel()
			err := validateNormalChanges(
				[]string{
					"conformance/providers/provider-a.yaml",
					forbidden,
				},
				rows,
			)
			if err == nil {
				t.Fatalf("共通契約の変更 %q が許可されました", forbidden)
			}
		})
	}
}

func TestValidateNormalChangesRejectsAnotherProviderPackage(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{
		{
			providerID:    "provider-a",
			implementedBy: "internal/source/a/v1",
			status:        "planned",
		},
		{
			providerID:    "provider-b",
			implementedBy: "internal/source/b/v1",
			status:        "implemented",
		},
	}
	err := validateNormalChanges(
		[]string{
			"conformance/providers/provider-a.yaml",
			"internal/source/a/v1/parser.go",
			"internal/source/b/v1/parser.go",
		},
		rows,
	)
	if err == nil {
		t.Fatal("別 provider package の変更が許可されました")
	}
	if !strings.Contains(err.Error(), "internal/source/b/v1/parser.go") {
		t.Fatalf("対象外 provider のパスを特定できないエラーです: %v", err)
	}
}

func TestValidateNormalChangesAllowsSingleProviderPackage(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{{
		providerID:    "provider-a",
		implementedBy: "internal/source/a/v1",
		status:        "planned",
	}}
	err := validateNormalChanges(
		[]string{
			"conformance/providers/provider-a.yaml",
			"internal/source/a/v1/parser.go",
			"internal/source/a/v1/fixtures/search.json",
		},
		rows,
	)
	if err != nil {
		t.Fatalf("一つの provider に閉じた変更が拒否されました: %v", err)
	}
}

func TestValidateNormalChangesAllowsAdoptedCoupledProviderUnit(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{
		{
			providerID:    "courts-hanrei-html",
			implementedBy: "internal/source/courts/hanrei",
			status:        "implemented",
		},
		{
			providerID:    "courts-hanrei-pdf",
			implementedBy: "internal/source/courts/hanreipdf",
			status:        "implemented",
		},
	}
	err := validateNormalChanges(
		[]string{
			"conformance/providers/courts-hanrei-html.yaml",
			"conformance/providers/courts-hanrei-pdf.yaml",
			"internal/source/courts/hanrei/provider_bindings.go",
			"internal/source/courts/hanreipdf/provider_bindings.go",
			"internal/application/provider_routes.go",
			"internal/config/provider_config.go",
			"cmd/japanese-law-mcp/main.go",
		},
		rows,
	)
	if err != nil {
		t.Fatalf("採用済みの従属 provider 単位が拒否されました: %v", err)
	}
}

func TestValidateNormalChangesRejectsCoupledProviderSourceOnlyMaintenance(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{
		{
			providerID:    "courts-hanrei-html",
			implementedBy: "internal/source/courts/hanrei",
			status:        "implemented",
		},
		{
			providerID:    "courts-hanrei-pdf",
			implementedBy: "internal/source/courts/hanreipdf",
			status:        "implemented",
		},
	}
	err := validateNormalChanges(
		[]string{
			"internal/source/courts/hanrei/parser.go",
			"internal/source/courts/hanreipdf/parser.go",
		},
		rows,
	)
	if err == nil {
		t.Fatal("matrix を変更しない複数 provider の maintenance が許可されました")
	}
}

func TestValidateNormalChangesRejectsProviderOutsideAdoptedCoupledUnit(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{
		{
			providerID:    "courts-hanrei-html",
			implementedBy: "internal/source/courts/hanrei",
		},
		{
			providerID:    "courts-hanrei-pdf",
			implementedBy: "internal/source/courts/hanreipdf",
		},
		{
			providerID:    "provider-c",
			implementedBy: "internal/source/provider/c",
		},
	}
	err := validateNormalChanges(
		[]string{
			"conformance/providers/courts-hanrei-html.yaml",
			"conformance/providers/courts-hanrei-pdf.yaml",
			"conformance/providers/provider-c.yaml",
		},
		rows,
	)
	if err == nil {
		t.Fatal("採用済みの従属 provider 単位へ第三の provider が混在しました")
	}
}

func TestRepositoryProviderPathUsesCurrentModule(t *testing.T) {
	t.Parallel()

	path, err := repositoryProviderPath(
		"github.com/example/project",
		"github.com/example/project/internal/source/a/v1",
	)
	if err != nil {
		t.Fatalf("module 内 package を解決できませんでした: %v", err)
	}
	if path != "internal/source/a/v1" {
		t.Fatalf("provider package path = %q", path)
	}
	if _, err := repositoryProviderPath(
		"github.com/example/project",
		"github.com/other/project/internal/source/a/v1",
	); err == nil {
		t.Fatal("module 外 package が許可されました")
	}
}

func TestValidateNormalChangesFailsClosedForAmbiguousProviderScope(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{
		{providerID: "provider-a", implementedBy: "internal/source/a/v1"},
		{providerID: "provider-b", implementedBy: "internal/source/b/v1"},
	}
	tests := []struct {
		name  string
		paths []string
	}{
		{
			name: "unknown matrix provider",
			paths: []string{
				"conformance/providers/provider-c.yaml",
			},
		},
		{
			name: "multiple matrix providers",
			paths: []string{
				"conformance/providers/provider-a.yaml",
				"conformance/providers/provider-b.yaml",
			},
		},
		{
			name: "provider neutral source",
			paths: []string{
				"conformance/providers/provider-a.yaml",
				"internal/source/shared/http.go",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if err := validateNormalChanges(tt.paths, rows); err == nil {
				t.Fatalf("曖昧な provider 変更が許可されました: %#v", tt.paths)
			}
		})
	}

	if err := validateNormalChanges(
		[]string{"docs/provider-guide.md"},
		rows,
	); err != nil {
		t.Fatalf("provider 対象外の変更が拒否されました: %v", err)
	}
}

func TestValidateNormalChangesRejectsProviderControlWithoutMatrixTarget(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{{
		providerID:    "provider-a",
		implementedBy: "internal/source/a/v1",
	}}
	for _, changedPath := range []string{
		"internal/model/provider_descriptor.go",
		"internal/application/provider_routes.go",
		"internal/application/provider_binding_registry.go",
		"internal/application/provider_registry.go",
		"internal/config/provider_config.go",
	} {
		if err := validateNormalChanges([]string{changedPath}, rows); err == nil {
			t.Fatalf("matrix target のない provider 制御変更が許可されました: %s", changedPath)
		}
	}

	if err := validateNormalChanges(
		[]string{
			"conformance/providers/provider-a.yaml",
			"internal/application/provider_routes.go",
		},
		rows,
	); err != nil {
		t.Fatalf("matrix target と同じ変更の provider 制御変更が拒否されました: %v", err)
	}

	if err := validateNormalChanges(
		[]string{
			"internal/application/legalquery/candidate_composition_member.go",
		},
		rows,
	); err != nil {
		t.Fatalf("provider と無関係な候補合成が provider 制御変更になりました: %v", err)
	}
}

func TestProviderControlPathUsesProductionResponsibilityBoundary(t *testing.T) {
	t.Parallel()

	productionControls := []string{
		"internal/model/provider_descriptor.go",
		"internal/application/provider_binding_registry.go",
		"internal/application/provider_registry.go",
		"internal/application/provider_routes.go",
		"internal/config/provider_config.go",
	}
	neutralFiles := []string{
		"internal/config/provider_loader.go",
		"internal/config/provider_config_test.go",
		"internal/config/config.go",
		"internal/config/extension_pack_loader.go",
		"internal/application/legalquery/candidate_composition_member.go",
	}
	allFiles := append(append([]string(nil), productionControls...), neutralFiles...)
	for _, repositoryPath := range allFiles {
		localPath := filepath.Join("..", "..", filepath.FromSlash(repositoryPath))
		info, err := os.Lstat(localPath)
		if err != nil {
			t.Fatalf("責任境界を確認する実在 file がありません: %s: %v", repositoryPath, err)
		}
		if !info.Mode().IsRegular() {
			t.Fatalf("責任境界を確認する対象が通常 file ではありません: %s", repositoryPath)
		}
	}
	for _, repositoryPath := range productionControls {
		if !providerControlPath(repositoryPath) {
			t.Errorf("provider 制御の production file が対象外です: %s", repositoryPath)
		}
	}
	for _, repositoryPath := range neutralFiles {
		if providerControlPath(repositoryPath) {
			t.Errorf("共有または test file が provider 制御に誤分類されました: %s", repositoryPath)
		}
	}
}

func TestEvaluateNormalChangesExcludesOnlyIsolatedTraceabilityMatrices(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{
		{providerID: "provider-a", implementedBy: "internal/source/a"},
		{providerID: "provider-b", implementedBy: "internal/source/b"},
		{providerID: "provider-c", implementedBy: "internal/source/c"},
	}
	traceOnly := map[string]struct{}{
		"conformance/providers/provider-a.yaml": {},
		"conformance/providers/provider-b.yaml": {},
		"conformance/providers/provider-c.yaml": {},
	}
	paths := []string{
		"README.md",
		"conformance/providers/provider-a.yaml",
		"conformance/providers/provider-b.yaml",
		"conformance/providers/provider-c.yaml",
		"internal/config/provider_loader.go",
		"internal/config/provider_config_test.go",
		"sot/40-interfaces/77-mcp-tool-exposure-and-extension-packs.md",
	}
	applicable, err := evaluateNormalChangesWithTraceOnlyMatrices(paths, rows, traceOnly)
	if err != nil {
		t.Fatalf("追跡可能性だけの matrix 更新が拒否されました: %v", err)
	}
	if !applicable {
		t.Fatal("追跡可能性だけの matrix 更新で conformance test が省略されました")
	}
}

func TestEvaluateNormalChangesRejectsTraceabilityMatrixMixedWithProviderScope(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{
		{providerID: "provider-a", implementedBy: "internal/source/a"},
		{providerID: "provider-b", implementedBy: "internal/source/b"},
	}
	tracePath := "conformance/providers/provider-a.yaml"
	traceOnly := map[string]struct{}{tracePath: {}}
	for _, forbidden := range []string{
		"conformance/provider-capability.schema.json",
		"conformance/providers/provider-b.yaml",
		"internal/providerconformance/loader_test.go",
		"internal/provideronboarding/changes.go",
		"internal/source/a/provider.go",
		"internal/application/provider_routes.go",
		"internal/application/lawsearch/port.go",
	} {
		forbidden := forbidden
		t.Run(forbidden, func(t *testing.T) {
			t.Parallel()
			_, err := evaluateNormalChangesWithTraceOnlyMatrices(
				[]string{tracePath, forbidden},
				rows,
				traceOnly,
			)
			if err == nil || !strings.Contains(err.Error(), forbidden) {
				t.Fatalf("混在変更 %q が拒否されませんでした: %v", forbidden, err)
			}
		})
	}
}

func TestEvaluateNormalChangesRejectsInvalidTraceabilityClassification(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{{
		providerID:    "provider-a",
		implementedBy: "internal/source/a",
	}}
	for name, traceOnly := range map[string]map[string]struct{}{
		"unchanged":  {"conformance/providers/provider-b.yaml": {}},
		"not matrix": {"README.md": {}},
	} {
		name := name
		traceOnly := traceOnly
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := evaluateNormalChangesWithTraceOnlyMatrices(
				[]string{"conformance/providers/provider-a.yaml"},
				rows,
				traceOnly,
			); err == nil {
				t.Fatalf("不正な追跡可能性分類が許可されました: %#v", traceOnly)
			}
		})
	}
}

func TestValidateNormalChangesRejectsInfrastructureScopeBypass(t *testing.T) {
	t.Parallel()

	rows := []matrixRow{{
		providerID:    "provider-a",
		implementedBy: "internal/source/a/v1",
	}}
	tests := []struct {
		name string
		path string
	}{
		{
			name: "unregistered provider source",
			path: "internal/source/new-provider/adapter.go",
		},
		{
			name: "provider control without matrix",
			path: "internal/application/provider_routes.go",
		},
		{
			name: "common capability contract",
			path: "internal/application/lawsearch/port.go",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			paths := []string{
				"internal/provideronboarding/conformance.go",
				test.path,
			}
			err := validateNormalChanges(paths, rows)
			if err == nil {
				t.Fatalf("conformance 基盤との混在変更が許可されました: %q", paths)
			}
			if !strings.Contains(err.Error(), test.path) {
				t.Fatalf("拒否した path を特定できないエラーです: %v", err)
			}
		})
	}
}
