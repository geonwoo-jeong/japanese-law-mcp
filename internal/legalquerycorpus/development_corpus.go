package legalquerycorpus

import "fmt"

// DevelopmentCorpus は、校正入力の development case だけを不変に保持する。
type DevelopmentCorpus struct {
	corpusVersion string
	schemaVersion int
	contentDigest string
	cases         []SemanticCase
	initialized   bool
}

func newDevelopmentCorpus(
	corpusVersion string,
	schemaVersion int,
	contentDigest string,
	cases []SemanticCase,
) (DevelopmentCorpus, error) {
	cloned, err := cloneCorpusSemanticCases(cases)
	if err != nil {
		return DevelopmentCorpus{}, err
	}
	development := DevelopmentCorpus{
		corpusVersion: corpusVersion,
		schemaVersion: schemaVersion,
		contentDigest: contentDigest,
		cases:         cloned,
		initialized:   true,
	}
	if err := development.Validate(); err != nil {
		return DevelopmentCorpus{}, err
	}
	return development, nil
}

// CorpusVersion は、development directory の corpus version を返す。
func (d DevelopmentCorpus) CorpusVersion() string { return d.corpusVersion }

// SchemaVersion は、development fixture 共通の schema version を返す。
func (d DevelopmentCorpus) SchemaVersion() int { return d.schemaVersion }

// ContentDigest は、file 名順の development fixture 原 byte を結ぶ digest を返す。
func (d DevelopmentCorpus) ContentDigest() string { return d.contentDigest }

// Cases は、file 名順の development case の複製を返す。
func (d DevelopmentCorpus) Cases() []SemanticCase {
	return mustCloneCorpusSemanticCases(d.cases)
}

// Validate は、development だけで閉じる版、種別および順序を確認する。
func (d DevelopmentCorpus) Validate() error {
	if !d.initialized {
		return fmt.Errorf("DevelopmentCorpus は LoadDevelopment を介して作成してください")
	}
	if !manifestCorpusVersionPattern.MatchString(d.corpusVersion) ||
		!isSupportedCorpusSchemaVersion(d.schemaVersion) ||
		!manifestSHA256Pattern.MatchString(d.contentDigest) ||
		len(d.cases) == 0 {
		return fmt.Errorf("development corpus の identity が有効ではありません")
	}
	previous := ""
	for _, semanticCase := range d.cases {
		if err := semanticCase.Validate(); err != nil {
			return fmt.Errorf("development semantic case が有効ではありません: %w", err)
		}
		if semanticCase.ArtifactKind() != ArtifactKindSemanticCase ||
			semanticCase.SchemaVersion() != d.schemaVersion ||
			validateManifestCaseID(
				ManifestSetDevelopment,
				semanticCase.CaseID(),
			) != nil ||
			(previous != "" && previous >= semanticCase.CaseID()) {
			return fmt.Errorf("development semantic case の版、集合または順序が不正です")
		}
		previous = semanticCase.CaseID()
	}
	return nil
}
