package legalquerycorpus

import (
	"fmt"
)

const maximumSemanticCaseListItems = 64

// SafetyVariant は、安全境界 case の通常入力または敵対的入力を表す。
type SafetyVariant string

const (
	// SafetyVariantOrdinary は、通常の入力による安全境界確認を表す。
	SafetyVariantOrdinary SafetyVariant = "ordinary"
	// SafetyVariantAdversarial は、敵対的な入力による安全境界確認を表す。
	SafetyVariantAdversarial SafetyVariant = "adversarial"
)

// SemanticCaseValues は、意味判定 fixture の作成に必要な値を保持する。
type SemanticCaseValues struct {
	ArtifactKind   ArtifactKind
	SchemaVersion  int
	CaseID         string
	LeakageGroupID string
	CoverageIDs    []string
	SafetyVariant  *SafetyVariant
	EnabledPacks   []string
	Request        Request
	Expected       SemanticExpected
}

// SemanticCase は、原 request と意味判定の期待投影を不変に保持する。
type SemanticCase struct {
	artifactKind   ArtifactKind
	schemaVersion  int
	caseID         string
	leakageGroupID string
	coverageIDs    []string
	safetyVariant  *SafetyVariant
	enabledPacks   []string
	request        Request
	expected       SemanticExpected
	initialized    bool
}

// NewSemanticCase は、単一 fixture 内の構造と cross-field 制約を検証して返す。
func NewSemanticCase(values SemanticCaseValues) (SemanticCase, error) {
	expected, err := cloneSemanticExpected(values.Expected)
	if err != nil {
		return SemanticCase{}, err
	}
	semanticCase := SemanticCase{
		artifactKind:   values.ArtifactKind,
		schemaVersion:  values.SchemaVersion,
		caseID:         values.CaseID,
		leakageGroupID: values.LeakageGroupID,
		coverageIDs:    cloneStrings(values.CoverageIDs),
		safetyVariant:  cloneSafetyVariant(values.SafetyVariant),
		enabledPacks:   cloneStrings(values.EnabledPacks),
		request:        values.Request.clone(),
		expected:       expected,
		initialized:    true,
	}
	if err := semanticCase.Validate(); err != nil {
		return SemanticCase{}, err
	}
	return semanticCase, nil
}

// ArtifactKind は、semantic_case を返す。
func (c SemanticCase) ArtifactKind() ArtifactKind {
	return c.artifactKind
}

// SchemaVersion は、成果物 schema の version を返す。
func (c SemanticCase) SchemaVersion() int {
	return c.schemaVersion
}

// CaseID は、development または holdout の fixture ID を返す。
func (c SemanticCase) CaseID() string {
	return c.caseID
}

// LeakageGroupID は、同じ発話と変形群を束ねる ID を返す。
func (c SemanticCase) LeakageGroupID() string {
	return c.leakageGroupID
}

// CoverageIDs は、昇順の詳細 coverage ID の複製を返す。
func (c SemanticCase) CoverageIDs() []string {
	return cloneStrings(c.coverageIDs)
}

// SafetyVariant は、安全境界 variant と項目の存在を返す。
func (c SemanticCase) SafetyVariant() (SafetyVariant, bool) {
	if c.safetyVariant == nil {
		return "", false
	}
	return *c.safetyVariant, true
}

// EnabledPacks は、case で有効な pack ID の複製を返す。
func (c SemanticCase) EnabledPacks() []string {
	return cloneStrings(c.enabledPacks)
}

// Request は、製品境界へ矯正していない原 request の複製を返す。
func (c SemanticCase) Request() Request {
	return c.request.clone()
}

// Expected は、request error または意味判定の期待投影の複製を返す。
func (c SemanticCase) Expected() SemanticExpected {
	expected, err := cloneSemanticExpected(c.expected)
	if err != nil {
		panic(fmt.Sprintf("検証済み SemanticCase の expected 複製に失敗しました: %v", err))
	}
	return expected
}

// Validate は、semantic case 単体で確認できる構造と期待値の整合を確認する。
func (c SemanticCase) Validate() error {
	if !c.initialized {
		return fmt.Errorf("SemanticCase は NewSemanticCase で作成しなければなりません")
	}
	if err := c.validateHeader(); err != nil {
		return err
	}
	if err := validateSemanticCoverage(c.coverageIDs, c.safetyVariant); err != nil {
		return err
	}
	if err := validateSemanticEnabledPacks(c.enabledPacks); err != nil {
		return err
	}
	if err := c.request.Validate(); err != nil {
		return fmt.Errorf("semantic case の request が初期化されていません")
	}
	if _, err := cloneSemanticExpected(c.expected); err != nil {
		return fmt.Errorf("semantic case の expected が有効ではありません: %w", err)
	}
	if err := validateSemanticRequestExpected(c.request, c.expected); err != nil {
		return err
	}
	if plan, ok := c.expected.(ExpectedPlan); ok {
		return validateSemanticPlanAvailability(plan, c.enabledPacks)
	}
	return nil
}

func (c SemanticCase) validateHeader() error {
	switch {
	case c.artifactKind != ArtifactKindSemanticCase:
		return fmt.Errorf("artifactKind は semantic_case でなければなりません")
	case c.schemaVersion != corpusSchemaVersion:
		return fmt.Errorf("schemaVersion は 1 でなければなりません")
	case !isSemanticCaseID(c.caseID):
		return fmt.Errorf(
			"semantic case の caseId は development または holdout の正規形でなければなりません",
		)
	case validateExpectedIdentifier("leakageGroupId", c.leakageGroupID) != nil:
		return fmt.Errorf("leakageGroupId は 1 byte 以上 64 byte 以下の正規形でなければなりません")
	default:
		return nil
	}
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*SemanticCase) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"SemanticCase は JSON から直接復元できません。version 別 DTO を使用してください",
	)
}

func isSemanticCaseID(value string) bool {
	return validateManifestCaseID(ManifestSetDevelopment, value) == nil ||
		validateManifestCaseID(ManifestSetHoldout, value) == nil
}

func cloneSafetyVariant(value *SafetyVariant) *SafetyVariant {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
