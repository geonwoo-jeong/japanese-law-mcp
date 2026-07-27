package legalquerycorpus

import "fmt"

const maximumExecutionCaseActions = 4

// ExecutionCaseValues は、実行 fixture の作成に必要な値を保持する。
type ExecutionCaseValues struct {
	ArtifactKind   ArtifactKind
	SchemaVersion  int
	CaseID         string
	ScenarioIDs    []string
	SemanticCaseID string
	Actions        []ExecutionAction
	Expected       ExecutionExpected
}

// ExecutionCase は、fake action と公開結果の期待投影を不変に保持する。
type ExecutionCase struct {
	artifactKind   ArtifactKind
	schemaVersion  int
	caseID         string
	scenarioIDs    []string
	semanticCaseID string
	actions        []ExecutionAction
	expected       ExecutionExpected
	initialized    bool
}

// NewExecutionCase は、fixture 単体で確認できる構造と投影を検証して返す。
func NewExecutionCase(values ExecutionCaseValues) (ExecutionCase, error) {
	actions, err := cloneExecutionActions(values.Actions)
	if err != nil {
		return ExecutionCase{}, err
	}
	expected, err := cloneExecutionExpected(values.Expected)
	if err != nil {
		return ExecutionCase{}, err
	}
	executionCase := ExecutionCase{
		artifactKind:   values.ArtifactKind,
		schemaVersion:  values.SchemaVersion,
		caseID:         values.CaseID,
		scenarioIDs:    cloneStrings(values.ScenarioIDs),
		semanticCaseID: values.SemanticCaseID,
		actions:        actions,
		expected:       expected,
		initialized:    true,
	}
	if err := executionCase.Validate(); err != nil {
		return ExecutionCase{}, err
	}
	return executionCase, nil
}

// ArtifactKind は、execution_case を返す。
func (c ExecutionCase) ArtifactKind() ArtifactKind {
	return c.artifactKind
}

// SchemaVersion は、成果物 schema の version を返す。
func (c ExecutionCase) SchemaVersion() int {
	return c.schemaVersion
}

// CaseID は、execution fixture の ID を返す。
func (c ExecutionCase) CaseID() string {
	return c.caseID
}

// ScenarioIDs は、昇順の実行 scenario ID の複製を返す。
func (c ExecutionCase) ScenarioIDs() []string {
	return cloneStrings(c.scenarioIDs)
}

// SemanticCaseID は、参照する development fixture の ID を返す。
func (c ExecutionCase) SemanticCaseID() string {
	return c.semanticCaseID
}

// Actions は、plan 順の fake action の複製を返す。
func (c ExecutionCase) Actions() []ExecutionAction {
	return mustCloneExecutionActions(c.actions)
}

// Expected は、公開結果または error の期待投影の複製を返す。
func (c ExecutionCase) Expected() ExecutionExpected {
	expected, err := cloneExecutionExpected(c.expected)
	if err != nil {
		panic(fmt.Sprintf("検証済み ExecutionCase の expected 複製に失敗しました: %v", err))
	}
	return expected
}

// Validate は、execution case 単体で閉じる順序、投影および scenario を確認する。
func (c ExecutionCase) Validate() error {
	if !c.initialized {
		return fmt.Errorf("ExecutionCase は NewExecutionCase で作成しなければなりません")
	}
	if err := c.validateHeader(); err != nil {
		return err
	}
	if err := validateExecutionScenarioIDs(c.scenarioIDs); err != nil {
		return err
	}
	if err := validateExecutionActions(c.actions); err != nil {
		return err
	}
	if _, err := cloneExecutionExpected(c.expected); err != nil {
		return fmt.Errorf("execution case の expected が有効ではありません: %w", err)
	}
	if err := validateExecutionActionProjection(c.actions, c.expected); err != nil {
		return err
	}
	return validateExecutionScenarios(c.scenarioIDs, c.actions, c.expected)
}

func (c ExecutionCase) validateHeader() error {
	switch {
	case c.artifactKind != ArtifactKindExecutionCase:
		return fmt.Errorf("artifactKind は execution_case でなければなりません")
	case c.schemaVersion != corpusSchemaVersion:
		return fmt.Errorf("schemaVersion は 1 でなければなりません")
	case validateManifestCaseID(ManifestSetExecution, c.caseID) != nil:
		return fmt.Errorf("execution case の caseId は execution 集合の正規形でなければなりません")
	case validateManifestCaseID(ManifestSetDevelopment, c.semanticCaseID) != nil:
		return fmt.Errorf("semanticCaseId は development 集合の正規形でなければなりません")
	default:
		return nil
	}
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExecutionCase) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ExecutionCase は JSON から直接復元できません。version 別 DTO を使用してください",
	)
}
