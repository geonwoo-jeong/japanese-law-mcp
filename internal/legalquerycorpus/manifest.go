package legalquerycorpus

import (
	"fmt"
	"regexp"
)

const (
	manifestSchemaVersion = 1
	manifestMaximumSeed   = 2147483647
	manifestMaximumCases  = 4096
)

var (
	manifestCorpusVersionPattern = regexp.MustCompile(`^corpus-v[1-9][0-9]*$`)
	manifestIdentifierPattern    = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	manifestSHA256Pattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ArtifactKind は、評価コーパス成果物の種別を表す。
type ArtifactKind string

const (
	// ArtifactKindCorpusManifest は、コーパス manifest を表す。
	ArtifactKindCorpusManifest ArtifactKind = "corpus_manifest"
)

// ManifestSetKind は、manifest が宣言する fixture 集合を表す。
type ManifestSetKind string

const (
	// ManifestSetDevelopment は、開発集合を表す。
	ManifestSetDevelopment ManifestSetKind = "development"
	// ManifestSetHoldout は、holdout 集合を表す。
	ManifestSetHoldout ManifestSetKind = "holdout"
	// ManifestSetExecution は、実行集合を表す。
	ManifestSetExecution ManifestSetKind = "execution"
)

// ManifestEntryValues は、一つの manifest entry の入力値を保持する。
type ManifestEntryValues struct {
	CaseID string
	SHA256 string
}

// ManifestEntry は、fixture の識別子と宣言 checksum を不変に保持する。
type ManifestEntry struct {
	caseID      string
	sha256      string
	initialized bool
}

// NewManifestEntry は、構造検証済みの manifest entry を返す。
func NewManifestEntry(values ManifestEntryValues) (ManifestEntry, error) {
	entry := ManifestEntry{
		caseID:      values.CaseID,
		sha256:      values.SHA256,
		initialized: true,
	}
	if err := entry.Validate(); err != nil {
		return ManifestEntry{}, err
	}
	return entry, nil
}

// CaseID は、fixture の case ID を返す。
func (e ManifestEntry) CaseID() string {
	return e.caseID
}

// SHA256 は、fixture 原 byte に対する宣言 checksum を返す。
func (e ManifestEntry) SHA256() string {
	return e.sha256
}

// Validate は、entry の識別子と checksum 制約を満たすか確認する。
func (e ManifestEntry) Validate() error {
	return e.validateAnySet()
}

func (e ManifestEntry) validateForSet(kind ManifestSetKind) error {
	if !e.initialized {
		return fmt.Errorf("ManifestEntry は NewManifestEntry で作成しなければなりません")
	}
	if !manifestSHA256Pattern.MatchString(e.sha256) {
		return fmt.Errorf("manifest entry の sha256 は小文字十六進六十四桁でなければなりません")
	}
	return validateManifestCaseID(kind, e.caseID)
}

func (e ManifestEntry) validateAnySet() error {
	if !e.initialized {
		return fmt.Errorf("ManifestEntry は NewManifestEntry で作成しなければなりません")
	}
	if !manifestSHA256Pattern.MatchString(e.sha256) {
		return fmt.Errorf("manifest entry の sha256 は小文字十六進六十四桁でなければなりません")
	}
	for _, kind := range manifestSetKinds() {
		if validateManifestCaseID(kind, e.caseID) == nil {
			return nil
		}
	}
	return fmt.Errorf("manifest entry の caseId が定義された集合形式ではありません")
}

// ManifestSetValues は、一つの fixture 集合の入力値を保持する。
type ManifestSetValues struct {
	Kind      ManifestSetKind
	CaseCount int
	Cases     []ManifestEntry
}

// ManifestSet は、集合種別と manifest 順の entry を不変に保持する。
type ManifestSet struct {
	kind        ManifestSetKind
	caseCount   int
	cases       []ManifestEntry
	initialized bool
}

// NewManifestSet は、件数と case ID 順を検証した集合を返す。
func NewManifestSet(values ManifestSetValues) (ManifestSet, error) {
	set := ManifestSet{
		kind:        values.Kind,
		caseCount:   values.CaseCount,
		cases:       cloneManifestEntries(values.Cases),
		initialized: true,
	}
	if err := set.Validate(); err != nil {
		return ManifestSet{}, err
	}
	return set, nil
}

// Kind は、fixture 集合の種別を返す。
func (s ManifestSet) Kind() ManifestSetKind {
	return s.kind
}

// CaseCount は、集合に宣言された fixture 件数を返す。
func (s ManifestSet) CaseCount() int {
	return s.caseCount
}

// Cases は、manifest 順の entry の複製を返す。
func (s ManifestSet) Cases() []ManifestEntry {
	return cloneManifestEntries(s.cases)
}

// Validate は、集合の種別、件数、entry と昇順を確認する。
func (s ManifestSet) Validate() error {
	if !s.initialized {
		return fmt.Errorf("ManifestSet は NewManifestSet で作成しなければなりません")
	}
	if !isManifestSetKind(s.kind) {
		return fmt.Errorf("manifest set の kind が定義されていません")
	}
	if s.caseCount < 0 || s.caseCount > manifestMaximumCases {
		return fmt.Errorf("manifest set の caseCount は 0 以上 4096 以下でなければなりません")
	}
	if s.caseCount != len(s.cases) {
		return fmt.Errorf("manifest set の caseCount と cases 件数が一致しません")
	}
	return validateManifestEntries(s.kind, s.cases)
}

func (s ManifestSet) clone() ManifestSet {
	return ManifestSet{
		kind:        s.kind,
		caseCount:   s.caseCount,
		cases:       cloneManifestEntries(s.cases),
		initialized: s.initialized,
	}
}

// ManifestValues は、manifest の作成に必要な値を保持する。
type ManifestValues struct {
	ArtifactKind                 ArtifactKind
	SchemaVersion                int
	CorpusVersion                string
	Seed                         int
	HoldoutDigest                string
	RequiredCategoryIDs          []string
	RequiredExecutionScenarioIDs []string
	Development                  ManifestSet
	Holdout                      ManifestSet
	Execution                    ManifestSet
}

// Manifest は、版、固定 catalog と三集合を不変に保持する。
type Manifest struct {
	artifactKind                 ArtifactKind
	schemaVersion                int
	corpusVersion                string
	seed                         int
	holdoutDigest                string
	requiredCategoryIDs          []string
	requiredExecutionScenarioIDs []string
	development                  ManifestSet
	holdout                      ManifestSet
	execution                    ManifestSet
	initialized                  bool
}

// NewManifest は、manifest 単体で確認できる構造を検証して返す。
func NewManifest(values ManifestValues) (Manifest, error) {
	manifest := Manifest{
		artifactKind:                 values.ArtifactKind,
		schemaVersion:                values.SchemaVersion,
		corpusVersion:                values.CorpusVersion,
		seed:                         values.Seed,
		holdoutDigest:                values.HoldoutDigest,
		requiredCategoryIDs:          cloneStrings(values.RequiredCategoryIDs),
		requiredExecutionScenarioIDs: cloneStrings(values.RequiredExecutionScenarioIDs),
		development:                  values.Development.clone(),
		holdout:                      values.Holdout.clone(),
		execution:                    values.Execution.clone(),
		initialized:                  true,
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// ArtifactKind は、corpus_manifest を返す。
func (m Manifest) ArtifactKind() ArtifactKind {
	return m.artifactKind
}

// SchemaVersion は、成果物 schema の version を返す。
func (m Manifest) SchemaVersion() int {
	return m.schemaVersion
}

// CorpusVersion は、コーパス directory の版を返す。
func (m Manifest) CorpusVersion() string {
	return m.corpusVersion
}

// Seed は、固定コーパス生成 seed を返す。
func (m Manifest) Seed() int {
	return m.seed
}

// HoldoutDigest は、holdout 順序付き entry の宣言 digest を返す。
func (m Manifest) HoldoutDigest() string {
	return m.holdoutDigest
}

// RequiredCategoryIDs は、必須 category ID の複製を返す。
func (m Manifest) RequiredCategoryIDs() []string {
	return cloneStrings(m.requiredCategoryIDs)
}

// RequiredExecutionScenarioIDs は、必須実行 scenario ID の複製を返す。
func (m Manifest) RequiredExecutionScenarioIDs() []string {
	return cloneStrings(m.requiredExecutionScenarioIDs)
}

// Development は、development 集合の複製を返す。
func (m Manifest) Development() ManifestSet {
	return m.development.clone()
}

// Holdout は、holdout 集合の複製を返す。
func (m Manifest) Holdout() ManifestSet {
	return m.holdout.clone()
}

// Execution は、execution 集合の複製を返す。
func (m Manifest) Execution() ManifestSet {
	return m.execution.clone()
}

// Validate は、manifest v1 の単一成果物内の構造を確認する。
func (m Manifest) Validate() error {
	if !m.initialized {
		return fmt.Errorf("Manifest は NewManifest で作成しなければなりません")
	}
	if err := m.validateHeader(); err != nil {
		return err
	}
	if !equalStringSequence(m.requiredCategoryIDs, manifestRequiredCategoryIDs()) {
		return fmt.Errorf("requiredCategoryIds は定義された十二件を正しい順序で保持しなければなりません")
	}
	if !equalStringSequence(
		m.requiredExecutionScenarioIDs,
		manifestRequiredExecutionScenarioIDs(),
	) {
		return fmt.Errorf("requiredExecutionScenarioIds は定義された七件を正しい順序で保持しなければなりません")
	}
	return m.validateSets()
}

func (m Manifest) validateHeader() error {
	switch {
	case m.artifactKind != ArtifactKindCorpusManifest:
		return fmt.Errorf("artifactKind は corpus_manifest でなければなりません")
	case m.schemaVersion != manifestSchemaVersion:
		return fmt.Errorf("schemaVersion は 1 でなければなりません")
	case !manifestCorpusVersionPattern.MatchString(m.corpusVersion):
		return fmt.Errorf("corpusVersion は corpus-v と正の十進整数の正規形でなければなりません")
	case m.seed < 0 || m.seed > manifestMaximumSeed:
		return fmt.Errorf("seed は 0 以上 2147483647 以下でなければなりません")
	case !manifestSHA256Pattern.MatchString(m.holdoutDigest):
		return fmt.Errorf("holdoutDigest は小文字十六進六十四桁でなければなりません")
	default:
		return nil
	}
}

func (m Manifest) validateSets() error {
	sets := []struct {
		expected ManifestSetKind
		value    ManifestSet
	}{
		{expected: ManifestSetDevelopment, value: m.development},
		{expected: ManifestSetHoldout, value: m.holdout},
		{expected: ManifestSetExecution, value: m.execution},
	}
	for _, set := range sets {
		if set.value.kind != set.expected {
			return fmt.Errorf("manifest の集合種別が配置と一致しません")
		}
		if err := set.value.Validate(); err != nil {
			return fmt.Errorf("manifest の集合が有効ではありません: %w", err)
		}
	}
	return nil
}

func validateManifestEntries(kind ManifestSetKind, entries []ManifestEntry) error {
	previous := ""
	for index, entry := range entries {
		if err := entry.validateForSet(kind); err != nil {
			return fmt.Errorf("manifest set の entry が有効ではありません: %w", err)
		}
		if index > 0 && previous >= entry.caseID {
			return fmt.Errorf("manifest set の cases は caseId の昇順で重複なく保持しなければなりません")
		}
		previous = entry.caseID
	}
	return nil
}

func validateManifestCaseID(kind ManifestSetKind, value string) error {
	if len(value) < 1 || len(value) > 64 || !manifestIdentifierPattern.MatchString(value) {
		return fmt.Errorf("caseId は 1 byte 以上 64 byte 以下の正規形でなければなりません")
	}
	prefix := string(kind) + "-"
	if len(value) <= len(prefix) || value[:len(prefix)] != prefix {
		return fmt.Errorf("caseId は所属集合の prefix で始まらなければなりません")
	}
	return nil
}

func cloneManifestEntries(values []ManifestEntry) []ManifestEntry {
	return append([]ManifestEntry{}, values...)
}

func cloneStrings(values []string) []string {
	return append([]string{}, values...)
}

func equalStringSequence(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func manifestSetKinds() []ManifestSetKind {
	return []ManifestSetKind{
		ManifestSetDevelopment,
		ManifestSetHoldout,
		ManifestSetExecution,
	}
}

func isManifestSetKind(value ManifestSetKind) bool {
	for _, kind := range manifestSetKinds() {
		if value == kind {
			return true
		}
	}
	return false
}

func manifestRequiredCategoryIDs() []string {
	return []string{
		"ambiguity",
		"budget-boundary",
		"capability-intent",
		"input-boundary",
		"law-name-and-concept",
		"official-reference",
		"pack-state",
		"safety-execution-boundary",
		"structured-location-and-date",
		"surface-variation",
		"typo-variation",
		"unsupported-scope",
	}
}

func manifestRequiredExecutionScenarioIDs() []string {
	return []string{
		"execution-all-failed",
		"execution-empty",
		"execution-item-budget",
		"execution-nonempty",
		"execution-partial-failure",
		"execution-reversed-completion",
		"execution-timeout",
	}
}
