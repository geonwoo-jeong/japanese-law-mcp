package providerconformance

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadはcanonicalArtifactsを読み込む(t *testing.T) {
	t.Parallel()

	catalog, err := Load(testRepositoryRoot(t))
	if err != nil {
		t.Fatalf("canonical artifacts を読み込めません: %v", err)
	}

	providers := catalog.Providers()
	if len(providers) != 2 {
		t.Fatalf("provider 数 = %d、期待値は 2 です", len(providers))
	}
	v1 := providerByID(t, providers, "e-gov-law-api-v1")
	v2 := providerByID(t, providers, "e-gov-law-api-v2")
	if v1.SchemaVersion != 1 || v2.SchemaVersion != 1 {
		t.Fatalf(
			"schemaVersion = v1:%d, v2:%d、期待値は 1 です",
			v1.SchemaVersion,
			v2.SchemaVersion,
		)
	}

	v1Rows := v1.Rows()
	if got := capabilityIDs(v1Rows); !slices.Equal(
		got,
		[]string{"law.update.list"},
	) {
		t.Fatalf("v1 capabilityId = %v", got)
	}
	v1Row := v1Rows[0]
	if !slices.Equal(
		v1Row.InterfaceSOTIDs,
		[]string{"SOT-IF-034", "SOT-IF-035", "SOT-IF-036", "SOT-IF-037"},
	) {
		t.Fatalf("v1 interfaceSotIds = %v", v1Row.InterfaceSOTIDs)
	}
	if v1Row.BudgetSOTID != "SOT-IF-035" ||
		v1Row.BudgetKey != "update-law-list-xml" ||
		v1Row.FixtureSet != "law-update-list-v1" ||
		v1Row.ImplementedBy !=
			"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/egov/lawv1" ||
		v1Row.ConformanceTarget != "./internal/source/egov/lawv1" ||
		v1Row.Status != "implemented" {
		t.Fatalf("v1 row = %#v", v1Row)
	}
	assertCasesAreExplicit(t, v1Row)
	assertCanonicalPublicErrors(t, v1Row)

	rows := v2.Rows()
	wantCapabilities := []string{
		"law.article.read",
		"law.content.search",
		"law.document.read",
		"law.search",
	}
	if got := capabilityIDs(rows); !slices.Equal(got, wantCapabilities) {
		t.Fatalf("capabilityId = %v、期待値は %v です", got, wantCapabilities)
	}

	wantSOTs := map[string][]string{
		"law.article.read":   {"SOT-IF-004", "SOT-IF-011", "SOT-IF-012", "SOT-IF-025"},
		"law.content.search": {"SOT-IF-004", "SOT-IF-010", "SOT-IF-023", "SOT-IF-028"},
		"law.document.read":  {"SOT-IF-004", "SOT-IF-011", "SOT-IF-024"},
		"law.search":         {"SOT-IF-004", "SOT-IF-009", "SOT-IF-022"},
	}
	wantFixtures := map[string]string{
		"law.article.read":   "law-article-read-v1",
		"law.content.search": "law-content-search-v1",
		"law.document.read":  "law-document-read-v1",
		"law.search":         "law-search-v1",
	}

	for _, row := range rows {
		if row.MajorVersion != 1 {
			t.Errorf("%s の majorVersion = %d、期待値は 1 です", row.CapabilityID, row.MajorVersion)
		}
		if !slices.Equal(row.InterfaceSOTIDs, wantSOTs[row.CapabilityID]) {
			t.Errorf("%s の interfaceSotIds = %v、期待値は %v です", row.CapabilityID, row.InterfaceSOTIDs, wantSOTs[row.CapabilityID])
		}
		if row.FixtureSet != wantFixtures[row.CapabilityID] {
			t.Errorf("%s の fixtureSet = %q、期待値は %q です", row.CapabilityID, row.FixtureSet, wantFixtures[row.CapabilityID])
		}
		if row.ParserContractVersion != "1.0.0" {
			t.Errorf("%s の parserContractVersion = %q、期待値は 1.0.0 です", row.CapabilityID, row.ParserContractVersion)
		}
		if row.ImplementedBy != "github.com/geonwoo-jeong/japanese-law-mcp/internal/source/egov/lawv2" {
			t.Errorf("%s の implementedBy = %q、期待値と異なります", row.CapabilityID, row.ImplementedBy)
		}
		if row.ConformanceTarget != "./internal/source/egov/lawv2" {
			t.Errorf("%s の conformanceTarget = %q、期待値と異なります", row.CapabilityID, row.ConformanceTarget)
		}
		if row.Status != "implemented" {
			t.Errorf("%s の status = %q、期待値は implemented です", row.CapabilityID, row.Status)
		}
		assertCasesAreExplicit(t, row)
		assertCanonicalPublicErrors(t, row)
	}

	if got := len(catalog.Rows()); got != 5 {
		t.Fatalf("Catalog.Rows() の件数 = %d、期待値は 5 です", got)
	}
}

func TestLoadの返却値は外部変更から分離される(t *testing.T) {
	t.Parallel()

	catalog, err := Load(testRepositoryRoot(t))
	if err != nil {
		t.Fatalf("canonical artifacts を読み込めません: %v", err)
	}

	providers := catalog.Providers()
	rows := providerByID(t, providers, "e-gov-law-api-v2").Rows()
	providers[0].ProviderID = "changed"
	rows[0].InterfaceSOTIDs[0] = "SOT-IF-999"
	rows[0].RequiredCases[0] = "changed"
	rows[0].PublicErrorSet[0] = "changed"
	rows[0].NotApplicableCases[0].Reason = "変更済み"

	again := catalog.Providers()
	againV2 := providerByID(t, again, "e-gov-law-api-v2")
	againRows := againV2.Rows()
	if len(again) != 2 {
		t.Fatal("Providers() の変更が Catalog 内部へ反映されました")
	}
	if againRows[0].InterfaceSOTIDs[0] == "SOT-IF-999" ||
		againRows[0].RequiredCases[0] == "changed" ||
		againRows[0].PublicErrorSet[0] == "changed" ||
		againRows[0].NotApplicableCases[0].Reason == "変更済み" {
		t.Fatal("Rows() の nested slice 変更が Catalog 内部へ反映されました")
	}

	flat := catalog.Rows()
	flat[0].InterfaceSOTIDs[0] = "SOT-IF-999"
	if catalog.Rows()[0].InterfaceSOTIDs[0] == "SOT-IF-999" {
		t.Fatal("Catalog.Rows() の nested slice 変更が内部へ反映されました")
	}
}

func providerByID(
	t *testing.T,
	providers []ProviderMatrix,
	providerID string,
) ProviderMatrix {
	t.Helper()
	for _, provider := range providers {
		if provider.ProviderID == providerID {
			return provider
		}
	}
	t.Fatalf("providerId %q がありません", providerID)
	return ProviderMatrix{}
}

func TestLoadは厳格でないYAMLを拒否する(t *testing.T) {
	t.Parallel()

	valid := string(validProviderYAML("test-provider", validRowYAML("test-provider", "law.search", "GET /laws", "laws-json")))
	tests := map[string][]byte{
		"UTF-8不正":      append([]byte(valid), 0xff),
		"複数document":   []byte(valid + "\n---\n{}\n"),
		"空の第二document": []byte(valid + "\n---\n"),
		"anchor": []byte(strings.Replace(
			valid,
			"providerId: test-provider",
			"providerId: &provider test-provider",
			1,
		)),
		"alias": []byte(strings.Replace(
			strings.Replace(valid, "providerId: test-provider", "providerId: &provider test-provider", 1),
			"operation: GET /laws",
			"operation: *provider",
			1,
		)),
		"merge key": []byte(strings.Replace(
			valid,
			"  - providerId: test-provider",
			"  - <<: &defaults\n      providerId: test-provider\n    providerId: test-provider",
			1,
		)),
		"custom tag": []byte(strings.Replace(
			valid,
			"providerId: test-provider",
			"providerId: !provider test-provider",
			1,
		)),
		"重複key": []byte(strings.Replace(
			valid,
			"    capabilityId: law.search",
			"    providerId: test-provider\n    capabilityId: law.search",
			1,
		)),
		"timestamp scalar": []byte(strings.Replace(
			valid,
			"parserContractVersion: 1.0.0",
			"parserContractVersion: 2026-07-25",
			1,
		)),
		"非有限float": []byte(strings.Replace(
			valid,
			"majorVersion: 1",
			"majorVersion: .inf",
			1,
		)),
		"16進integer": []byte(strings.Replace(
			valid,
			"majorVersion: 1",
			"majorVersion: 0x1",
			1,
		)),
		"非文字列key": []byte(strings.Replace(
			valid,
			"schemaVersion: 1",
			"? [schemaVersion]\n: 1",
			1,
		)),
		"未知の最上位key": []byte(valid + "unknown: true\n"),
	}

	for name, content := range tests {
		name, content := name, content
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository := newTestRepository(t, map[string][]byte{"test-provider.yaml": content})
			if _, err := Load(repository); err == nil {
				t.Fatal("不正な YAML を受理しました")
			}
		})
	}
}

func TestLoadはproviderDirectoryの境界を検証する(t *testing.T) {
	t.Parallel()

	valid := validProviderYAML("test-provider", validRowYAML("test-provider", "law.search", "GET /laws", "laws-json"))

	t.Run("provider fileなし", func(t *testing.T) {
		repository := newTestRepository(t, nil)
		if _, err := Load(repository); err == nil {
			t.Fatal("provider file がない repository を受理しました")
		}
	})

	t.Run("未知file", func(t *testing.T) {
		repository := newTestRepository(t, map[string][]byte{
			"test-provider.yaml": valid,
			"README.txt":         []byte("説明"),
		})
		if _, err := Load(repository); err == nil {
			t.Fatal("未知 file を受理しました")
		}
	})

	t.Run("yml拡張子", func(t *testing.T) {
		repository := newTestRepository(t, map[string][]byte{"test-provider.yml": valid})
		if _, err := Load(repository); err == nil {
			t.Fatal(".yml file を受理しました")
		}
	})

	t.Run("directory", func(t *testing.T) {
		repository := newTestRepository(t, map[string][]byte{"test-provider.yaml": valid})
		if err := os.Mkdir(filepath.Join(repository, "conformance", "providers", "nested"), 0o750); err != nil {
			t.Fatalf("test directory を作成できません: %v", err)
		}
		if _, err := Load(repository); err == nil {
			t.Fatal("provider directory 内の directory を受理しました")
		}
	})

	t.Run("symlink", func(t *testing.T) {
		repository := newTestRepository(t, nil)
		target := filepath.Join(repository, "outside.yaml")
		if err := os.WriteFile(target, valid, 0o600); err != nil {
			t.Fatalf("symlink target を作成できません: %v", err)
		}
		link := filepath.Join(repository, "conformance", "providers", "test-provider.yaml")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symlink を作成できないため省略します: %v", err)
		}
		if _, err := Load(repository); err == nil {
			t.Fatal("symlink provider file を受理しました")
		}
	})

	t.Run("schema symlink", func(t *testing.T) {
		repository := newTestRepository(t, map[string][]byte{"test-provider.yaml": valid})
		schemaPath := filepath.Join(repository, "conformance", "provider-capability.schema.json")
		target := filepath.Join(repository, "schema.json")
		copyTestFile(t, filepath.Join(testRepositoryRoot(t), "conformance", "provider-capability.schema.json"), target)
		if err := os.Remove(schemaPath); err != nil {
			t.Fatalf("schema を置換できません: %v", err)
		}
		if err := os.Symlink(target, schemaPath); err != nil {
			t.Skipf("symlink を作成できないため省略します: %v", err)
		}
		if _, err := Load(repository); err == nil {
			t.Fatal("symlink schema を受理しました")
		}
	})
}

func TestLoadはfile名とrowの整列重複を検証する(t *testing.T) {
	t.Parallel()

	tests := map[string]map[string][]byte{
		"file名とproviderId不一致": {
			"other-provider.yaml": validProviderYAML(
				"test-provider",
				validRowYAML("test-provider", "law.search", "GET /laws", "laws-json"),
			),
		},
		"row未整列": {
			"test-provider.yaml": validProviderYAML(
				"test-provider",
				validRowYAML("test-provider", "law.search", "GET /laws", "laws-json"),
				validRowYAML("test-provider", "law.article.read", "GET /law", "law-data-xml"),
			),
		},
		"tuple重複": {
			"test-provider.yaml": validProviderYAML(
				"test-provider",
				validRowYAML("test-provider", "law.search", "GET /laws", "laws-json"),
				validRowYAML("test-provider", "law.search", "GET /laws", "other-budget"),
			),
		},
	}

	for name, files := range tests {
		name, files := name, files
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository := newTestRepository(t, files)
			if _, err := Load(repository); err == nil {
				t.Fatal("不正な provider matrix を受理しました")
			}
		})
	}
}

func TestLoadはschemaの版と妥当性を検証する(t *testing.T) {
	t.Parallel()

	valid := validProviderYAML("test-provider", validRowYAML("test-provider", "law.search", "GET /laws", "laws-json"))
	tests := map[string]string{
		"Draft宣言なし": `{"type":"object"}`,
		"Draft違い":   `{"$schema":"http://json-schema.org/draft-07/schema#","type":"object"}`,
		"不正JSON":    `{`,
		"不正schema":  `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":42}`,
	}

	for name, schema := range tests {
		name, schema := name, schema
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository := newTestRepository(t, map[string][]byte{"test-provider.yaml": valid})
			path := filepath.Join(repository, "conformance", "provider-capability.schema.json")
			if err := os.WriteFile(path, []byte(schema), 0o600); err != nil {
				t.Fatalf("test schema を書き込めません: %v", err)
			}
			if _, err := Load(repository); err == nil {
				t.Fatal("不正な schema を受理しました")
			}
		})
	}
}

func assertCasesAreExplicit(t *testing.T, row Row) {
	t.Helper()

	required := make(map[string]struct{}, len(row.RequiredCases))
	for _, name := range row.RequiredCases {
		required[name] = struct{}{}
	}
	notApplicable := make(map[string]struct{}, len(row.NotApplicableCases))
	for _, item := range row.NotApplicableCases {
		if item.Reason == "" {
			t.Errorf("%s の %s に日本語の理由がありません", row.CapabilityID, item.Case)
		}
		notApplicable[item.Case] = struct{}{}
	}
	for _, name := range standardCases {
		_, isRequired := required[name]
		_, isNotApplicable := notApplicable[name]
		if isRequired == isNotApplicable {
			t.Errorf("%s の case %q は requiredCases と notApplicableCases の片方だけに必要です", row.CapabilityID, name)
		}
	}
}

func assertCanonicalPublicErrors(t *testing.T, row Row) {
	t.Helper()

	want := []string{"invalid_argument"}
	switch row.CapabilityID {
	case "law.document.read":
		want = append(want, "not_found")
	case "law.article.read":
		want = append(want, "not_found", "ambiguous_location")
	}
	want = append(want,
		"unsupported_capability",
		"unsupported_query",
		"configuration_required",
		"source_auth_failed",
		"rate_limited",
		"source_timeout",
		"source_unavailable",
		"source_busy",
		"source_contract_changed",
		"invalid_source_response",
		"source_response_too_large",
		"source_processing_limit",
		"unsafe_source_content",
	)
	if !slices.Equal(row.PublicErrorSet, want) {
		t.Errorf("%s の publicErrorSet = %v、期待値は %v です", row.CapabilityID, row.PublicErrorSet, want)
	}
}

func capabilityIDs(rows []Row) []string {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.CapabilityID
	}
	return ids
}

func newTestRepository(t *testing.T, providers map[string][]byte) string {
	t.Helper()

	repository := t.TempDir()
	providerDirectory := filepath.Join(repository, "conformance", "providers")
	if err := os.MkdirAll(providerDirectory, 0o750); err != nil {
		t.Fatalf("test repository を作成できません: %v", err)
	}
	copyTestFile(
		t,
		filepath.Join(testRepositoryRoot(t), "conformance", "provider-capability.schema.json"),
		filepath.Join(repository, "conformance", "provider-capability.schema.json"),
	)
	for name, content := range providers {
		if err := os.WriteFile(filepath.Join(providerDirectory, name), content, 0o600); err != nil {
			t.Fatalf("provider matrix %q を書き込めません: %v", name, err)
		}
	}
	return repository
}

func copyTestFile(t *testing.T, source, destination string) {
	t.Helper()

	//nolint:gosec // SOT-ENG-017: test helper の source は各 test が repository 内の既知の fixture path として構成する。
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("test file %q を読み込めません: %v", source, err)
	}
	//nolint:gosec // SOT-ENG-017: test helper の destination は t.TempDir 配下の既知の canonical path に限定する。
	if err := os.WriteFile(destination, data, 0o600); err != nil {
		t.Fatalf("test file %q を書き込めません: %v", destination, err)
	}
}

func validProviderYAML(providerID string, rows ...string) []byte {
	return []byte(fmt.Sprintf("schemaVersion: 1\nrows:\n%s\n", strings.Join(rows, "\n")))
}

func validRowYAML(providerID, capabilityID, operation, budgetKey string) string {
	return fmt.Sprintf(`  - providerId: %s
    capabilityId: %s
    majorVersion: 1
    operation: %s
    interfaceSotIds:
      - SOT-IF-004
      - SOT-IF-009
      - SOT-IF-022
    budgetSotId: SOT-IF-004
    budgetKey: %s
    concurrencyGroup: provider-http
    artifactType: JSON
    fixtureSet: law-search-v1
    requiredCases:
      - descriptor
      - capability-binding
      - outbound-request
      - authentication
      - provenance
      - resource-ref-roundtrip
      - empty-vs-not-found
      - unsupported-query
      - page-invariants
      - continuation-roundtrip
      - continuation-tamper
      - continuation-expired
      - error-normalization
      - secret-non-exposure
      - response-bytes-limit
      - decompressed-bytes-limit
      - entries-or-objects-limit
      - depth-limit
      - parse-timeout
      - concurrency-limit
      - cancellation
      - contract-changed
    notApplicableCases: []
    supportsContinuation: true
    supportsAuth: true
    publicErrorSet:
      - invalid_argument
    parserContractVersion: 1.0.0
    implementedBy: github.com/geonwoo-jeong/japanese-law-mcp/internal/source/test/provider
    conformanceTarget: ./internal/source/test/provider
    status: planned`, providerID, capabilityID, operation, budgetKey)
}
