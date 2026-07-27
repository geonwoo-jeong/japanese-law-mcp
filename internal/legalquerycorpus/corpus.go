package legalquerycorpus

import "fmt"

// Corpus は、manifest 順の三つの評価集合を不変に保持する。
type Corpus struct {
	manifest    Manifest
	development []SemanticCase
	holdout     []SemanticCase
	execution   []ExecutionCase
	initialized bool
}

// newCorpus は、loader が検証した成果物を manifest 順の aggregate にする。
func newCorpus(
	manifest Manifest,
	development []SemanticCase,
	holdout []SemanticCase,
	execution []ExecutionCase,
) (Corpus, error) {
	clonedManifest, err := cloneCorpusManifest(manifest)
	if err != nil {
		return Corpus{}, err
	}
	clonedDevelopment, err := cloneCorpusSemanticCases(development)
	if err != nil {
		return Corpus{}, err
	}
	clonedHoldout, err := cloneCorpusSemanticCases(holdout)
	if err != nil {
		return Corpus{}, err
	}
	clonedExecution, err := cloneCorpusExecutionCases(execution)
	if err != nil {
		return Corpus{}, err
	}
	corpus := Corpus{
		manifest:    clonedManifest,
		development: clonedDevelopment,
		holdout:     clonedHoldout,
		execution:   clonedExecution,
		initialized: true,
	}
	if err := corpus.Validate(); err != nil {
		return Corpus{}, err
	}
	return corpus, nil
}

// Manifest は、成果物一覧と版を持つ manifest の複製を返す。
func (c Corpus) Manifest() Manifest {
	manifest, err := cloneCorpusManifest(c.manifest)
	if err != nil {
		panic(fmt.Sprintf("検証済み Corpus の manifest 複製に失敗しました: %v", err))
	}
	return manifest
}

// Development は、manifest 順の development case の複製を返す。
func (c Corpus) Development() []SemanticCase {
	return mustCloneCorpusSemanticCases(c.development)
}

// Holdout は、manifest 順の holdout case の複製を返す。
func (c Corpus) Holdout() []SemanticCase {
	return mustCloneCorpusSemanticCases(c.holdout)
}

// Execution は、manifest 順の execution case の複製を返す。
func (c Corpus) Execution() []ExecutionCase {
	return mustCloneCorpusExecutionCases(c.execution)
}

// Validate は、manifest と三集合の件数、版、種別および順序を確認する。
func (c Corpus) Validate() error {
	if !c.initialized {
		return fmt.Errorf("Corpus は Load を介して作成しなければなりません")
	}
	if err := c.manifest.Validate(); err != nil {
		return fmt.Errorf("corpus の manifest が有効ではありません: %w", err)
	}
	schemaVersion := c.manifest.SchemaVersion()
	if err := validateCorpusSemanticSet(
		c.manifest.Development(),
		ManifestSetDevelopment,
		schemaVersion,
		c.development,
	); err != nil {
		return err
	}
	if err := validateCorpusSemanticSet(
		c.manifest.Holdout(),
		ManifestSetHoldout,
		schemaVersion,
		c.holdout,
	); err != nil {
		return err
	}
	return validateCorpusExecutionSet(
		c.manifest.Execution(),
		schemaVersion,
		c.execution,
	)
}

func validateCorpusSemanticSet(
	set ManifestSet,
	expectedSet ManifestSetKind,
	schemaVersion int,
	cases []SemanticCase,
) error {
	if set.Kind() != expectedSet || set.CaseCount() != len(cases) {
		return fmt.Errorf("semantic case 集合が manifest の宣言と一致しません")
	}
	entries := set.Cases()
	for index, semanticCase := range cases {
		if err := semanticCase.Validate(); err != nil {
			return fmt.Errorf("corpus の semantic case が有効ではありません: %w", err)
		}
		if semanticCase.ArtifactKind() != ArtifactKindSemanticCase ||
			semanticCase.SchemaVersion() != schemaVersion ||
			validateManifestCaseID(expectedSet, semanticCase.CaseID()) != nil ||
			entries[index].CaseID() != semanticCase.CaseID() {
			return fmt.Errorf("semantic case が manifest の集合、版または順序と一致しません")
		}
	}
	return nil
}

func validateCorpusExecutionSet(
	set ManifestSet,
	schemaVersion int,
	cases []ExecutionCase,
) error {
	if set.Kind() != ManifestSetExecution || set.CaseCount() != len(cases) {
		return fmt.Errorf("execution case 集合が manifest の宣言と一致しません")
	}
	entries := set.Cases()
	for index, executionCase := range cases {
		if err := executionCase.Validate(); err != nil {
			return fmt.Errorf("corpus の execution case が有効ではありません: %w", err)
		}
		if executionCase.ArtifactKind() != ArtifactKindExecutionCase ||
			executionCase.SchemaVersion() != schemaVersion ||
			entries[index].CaseID() != executionCase.CaseID() {
			return fmt.Errorf("execution case が manifest の版または順序と一致しません")
		}
	}
	return nil
}

// UnmarshalJSON は、loader を介さない直接復元を拒否する。
func (*Corpus) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Corpus は JSON から直接復元できません。Load を使用してください")
}
