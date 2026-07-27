package legalquerycorpus

import "fmt"

func cloneCorpusManifest(value Manifest) (Manifest, error) {
	if err := value.Validate(); err != nil {
		return Manifest{}, err
	}
	return NewManifest(ManifestValues{
		ArtifactKind:                 value.ArtifactKind(),
		SchemaVersion:                value.SchemaVersion(),
		CorpusVersion:                value.CorpusVersion(),
		Seed:                         value.Seed(),
		HoldoutDigest:                value.HoldoutDigest(),
		RequiredCategoryIDs:          value.RequiredCategoryIDs(),
		RequiredExecutionScenarioIDs: value.RequiredExecutionScenarioIDs(),
		Development:                  value.Development(),
		Holdout:                      value.Holdout(),
		Execution:                    value.Execution(),
	})
}

func cloneCorpusSemanticCase(value SemanticCase) (SemanticCase, error) {
	if err := value.Validate(); err != nil {
		return SemanticCase{}, err
	}
	var safetyVariant *SafetyVariant
	if variant, exists := value.SafetyVariant(); exists {
		safetyVariant = &variant
	}
	return NewSemanticCase(SemanticCaseValues{
		ArtifactKind:   value.ArtifactKind(),
		SchemaVersion:  value.SchemaVersion(),
		CaseID:         value.CaseID(),
		LeakageGroupID: value.LeakageGroupID(),
		CoverageIDs:    value.CoverageIDs(),
		SafetyVariant:  safetyVariant,
		EnabledPacks:   value.EnabledPacks(),
		Request:        value.Request(),
		Expected:       value.Expected(),
	})
}

func cloneCorpusSemanticCases(values []SemanticCase) ([]SemanticCase, error) {
	cloned := make([]SemanticCase, 0, len(values))
	for _, value := range values {
		semanticCase, err := cloneCorpusSemanticCase(value)
		if err != nil {
			return nil, fmt.Errorf("semantic case を複製できません: %w", err)
		}
		cloned = append(cloned, semanticCase)
	}
	return cloned, nil
}

func cloneCorpusExecutionCase(value ExecutionCase) (ExecutionCase, error) {
	if err := value.Validate(); err != nil {
		return ExecutionCase{}, err
	}
	return NewExecutionCase(ExecutionCaseValues{
		ArtifactKind:   value.ArtifactKind(),
		SchemaVersion:  value.SchemaVersion(),
		CaseID:         value.CaseID(),
		ScenarioIDs:    value.ScenarioIDs(),
		SemanticCaseID: value.SemanticCaseID(),
		Actions:        value.Actions(),
		Expected:       value.Expected(),
	})
}

func cloneCorpusExecutionCases(values []ExecutionCase) ([]ExecutionCase, error) {
	cloned := make([]ExecutionCase, 0, len(values))
	for _, value := range values {
		executionCase, err := cloneCorpusExecutionCase(value)
		if err != nil {
			return nil, fmt.Errorf("execution case を複製できません: %w", err)
		}
		cloned = append(cloned, executionCase)
	}
	return cloned, nil
}

func mustCloneCorpusSemanticCases(values []SemanticCase) []SemanticCase {
	cloned, err := cloneCorpusSemanticCases(values)
	if err != nil {
		panic(fmt.Sprintf("検証済み Corpus の semantic cases 複製に失敗しました: %v", err))
	}
	return cloned
}

func mustCloneCorpusExecutionCases(values []ExecutionCase) []ExecutionCase {
	cloned, err := cloneCorpusExecutionCases(values)
	if err != nil {
		panic(fmt.Sprintf("検証済み Corpus の execution cases 複製に失敗しました: %v", err))
	}
	return cloned
}
