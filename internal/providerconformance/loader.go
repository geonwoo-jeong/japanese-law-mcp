package providerconformance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

const (
	canonicalSchemaRelativePath   = "conformance/provider-capability.schema.json"
	providerDirectoryRelativePath = "conformance/providers"
	draft202012Schema             = "https://json-schema.org/draft/2020-12/schema"
)

var (
	providerIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	standardCases     = [...]string{
		"descriptor",
		"capability-binding",
		"outbound-request",
		"authentication",
		"provenance",
		"resource-ref-roundtrip",
		"empty-vs-not-found",
		"unsupported-query",
		"page-invariants",
		"continuation-roundtrip",
		"continuation-tamper",
		"continuation-expired",
		"error-normalization",
		"secret-non-exposure",
		"response-bytes-limit",
		"decompressed-bytes-limit",
		"entries-or-objects-limit",
		"depth-limit",
		"parse-timeout",
		"concurrency-limit",
		"cancellation",
		"contract-changed",
	}
)

type matrixDocument struct {
	SchemaVersion int   `json:"schemaVersion"`
	Rows          []Row `json:"rows"`
}

type tuple struct {
	providerID   string
	capabilityID string
	majorVersion int
	operation    string
}

// Load は canonical schema と全 provider matrix を読み込む。
func Load(repository string) (Catalog, error) {
	if repository == "" {
		return Catalog{}, fmt.Errorf("repository path を指定してください")
	}
	root, err := filepath.Abs(repository)
	if err != nil {
		return Catalog{}, fmt.Errorf("repository path を解決できません: %w", err)
	}
	if err := requireDirectory(root, "repository"); err != nil {
		return Catalog{}, err
	}

	schemaPath := filepath.Join(root, filepath.FromSlash(canonicalSchemaRelativePath))
	schema, err := loadSchema(schemaPath)
	if err != nil {
		return Catalog{}, err
	}

	providerDirectory := filepath.Join(root, filepath.FromSlash(providerDirectoryRelativePath))
	if err := requireDirectory(providerDirectory, "provider matrix directory"); err != nil {
		return Catalog{}, err
	}
	entries, err := os.ReadDir(providerDirectory)
	if err != nil {
		return Catalog{}, fmt.Errorf("provider matrix directory を読み込めません: %w", err)
	}
	if len(entries) == 0 {
		return Catalog{}, fmt.Errorf("provider matrix file がありません")
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	providers := make([]ProviderMatrix, 0, len(entries))
	seenTuples := make(map[tuple]string)
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(providerDirectory, name)
		if entry.Type()&os.ModeSymlink != 0 {
			return Catalog{}, fmt.Errorf("provider matrix %q に symlink は使用できません", name)
		}
		info, err := entry.Info()
		if err != nil {
			return Catalog{}, fmt.Errorf("provider matrix %q の情報を取得できません: %w", name, err)
		}
		if !info.Mode().IsRegular() || filepath.Ext(name) != ".yaml" {
			return Catalog{}, fmt.Errorf("provider matrix directory に未知の entry %q があります", name)
		}
		providerID := strings.TrimSuffix(name, ".yaml")
		if !providerIDPattern.MatchString(providerID) {
			return Catalog{}, fmt.Errorf("provider matrix file 名 %q が providerId 形式ではありません", name)
		}

		matrix, err := loadProviderMatrix(path, providerID, schema)
		if err != nil {
			return Catalog{}, fmt.Errorf("provider matrix %q: %w", name, err)
		}
		for _, row := range matrix.rows {
			key := tuple{
				providerID:   row.ProviderID,
				capabilityID: row.CapabilityID,
				majorVersion: row.MajorVersion,
				operation:    row.Operation,
			}
			if previous, exists := seenTuples[key]; exists {
				return Catalog{}, fmt.Errorf("row tuple が %q と %q で重複しています", previous, name)
			}
			seenTuples[key] = name
		}
		providers = append(providers, matrix)
	}

	return Catalog{providers: providers}, nil
}

func loadSchema(path string) (*jsonschema.Resolved, error) {
	data, err := readRegularFile(path, "canonical schema")
	if err != nil {
		return nil, err
	}

	var declaration struct {
		Schema string `json:"$schema"`
	}
	if err := json.Unmarshal(data, &declaration); err != nil {
		return nil, fmt.Errorf("canonical schema は正しい JSON ではありません: %w", err)
	}
	if declaration.Schema != draft202012Schema {
		return nil, fmt.Errorf("canonical schema の $schema は %q でなければなりません", draft202012Schema)
	}

	var schema jsonschema.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("canonical schema を解釈できません: %w", err)
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("canonical schema を解決できません: %w", err)
	}
	return resolved, nil
}

func loadProviderMatrix(path, providerID string, schema *jsonschema.Resolved) (ProviderMatrix, error) {
	data, err := readRegularFile(path, "provider matrix")
	if err != nil {
		return ProviderMatrix{}, err
	}
	value, err := decodeStrictYAML(data)
	if err != nil {
		return ProviderMatrix{}, err
	}
	if err := schema.Validate(value); err != nil {
		return ProviderMatrix{}, fmt.Errorf("schema に適合しません: %w", err)
	}

	jsonData, err := json.Marshal(value)
	if err != nil {
		return ProviderMatrix{}, fmt.Errorf("JSON 互換値へ変換できません: %w", err)
	}
	var document matrixDocument
	if err := json.Unmarshal(jsonData, &document); err != nil {
		return ProviderMatrix{}, fmt.Errorf("matrix row を読み込めません: %w", err)
	}
	for index, row := range document.Rows {
		if row.ProviderID != providerID {
			return ProviderMatrix{}, fmt.Errorf("rows[%d].providerId %q が file 名の %q と一致しません", index, row.ProviderID, providerID)
		}
		if err := validateCasePartition(row); err != nil {
			return ProviderMatrix{}, fmt.Errorf("rows[%d]: %w", index, err)
		}
		if index > 0 && compareRows(document.Rows[index-1], row) > 0 {
			return ProviderMatrix{}, fmt.Errorf("rows[%d] は capabilityId、majorVersion、operation、budgetKey の順に整列していません", index)
		}
	}
	if err := rejectDuplicateTuples(document.Rows); err != nil {
		return ProviderMatrix{}, err
	}

	return ProviderMatrix{
		ProviderID:    providerID,
		SchemaVersion: document.SchemaVersion,
		rows:          cloneRows(document.Rows),
	}, nil
}

func validateCasePartition(row Row) error {
	required := make(map[string]struct{}, len(row.RequiredCases))
	for _, name := range row.RequiredCases {
		required[name] = struct{}{}
	}
	notApplicable := make(map[string]struct{}, len(row.NotApplicableCases))
	for _, item := range row.NotApplicableCases {
		if _, exists := notApplicable[item.Case]; exists {
			return fmt.Errorf("notApplicableCases の case %q が重複しています", item.Case)
		}
		if _, exists := required[item.Case]; exists {
			return fmt.Errorf("case %q が requiredCases と notApplicableCases に重複しています", item.Case)
		}
		notApplicable[item.Case] = struct{}{}
	}
	for _, name := range standardCases {
		_, requiredCase := required[name]
		_, notApplicableCase := notApplicable[name]
		if requiredCase == notApplicableCase {
			return fmt.Errorf("標準 case %q は requiredCases と notApplicableCases の片方だけに記述してください", name)
		}
	}
	return nil
}

func rejectDuplicateTuples(rows []Row) error {
	seen := make(map[tuple]int, len(rows))
	for index, row := range rows {
		key := tuple{
			providerID:   row.ProviderID,
			capabilityID: row.CapabilityID,
			majorVersion: row.MajorVersion,
			operation:    row.Operation,
		}
		if previous, exists := seen[key]; exists {
			return fmt.Errorf("rows[%d] と rows[%d] の tuple が重複しています", previous, index)
		}
		seen[key] = index
	}
	return nil
}

func compareRows(left, right Row) int {
	switch {
	case left.CapabilityID < right.CapabilityID:
		return -1
	case left.CapabilityID > right.CapabilityID:
		return 1
	case left.MajorVersion < right.MajorVersion:
		return -1
	case left.MajorVersion > right.MajorVersion:
		return 1
	case left.Operation < right.Operation:
		return -1
	case left.Operation > right.Operation:
		return 1
	case left.BudgetKey < right.BudgetKey:
		return -1
	case left.BudgetKey > right.BudgetKey:
		return 1
	default:
		return 0
	}
}

func requireDirectory(path, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%s %q を確認できません: %w", label, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%s %q は通常の directory でなければなりません", label, path)
	}
	return nil
}

func readRegularFile(path, label string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("%s %q を確認できません: %w", label, path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s %q は通常 file でなければなりません", label, path)
	}
	//nolint:gosec // SOT-ENG-017: path は repository 内の固定 canonical path で、直前に通常ファイルか確認する。
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s %q を読み込めません: %w", label, path, err)
	}
	return data, nil
}
