package legalquery

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

type fixedQueryProfile struct {
	metadata   QueryProfileMetadata
	generation CandidateGeneration
}

func (p fixedQueryProfile) Metadata() QueryProfileMetadata {
	return p.metadata
}

func (fixedQueryProfile) CueVocabulary() []CueVocabularyEntry {
	return nil
}

func (p fixedQueryProfile) Generate(
	CandidateGenerationInput,
	CandidateIDScope,
) (CandidateGeneration, error) {
	return p.generation, nil
}

func TestCollectProfileCandidatesはprofile識別子の不一致を拒否する(
	t *testing.T,
) {
	t.Parallel()

	metadata := mustCollectionProfileMetadata(t)
	preprocessed, err := NewPreprocessResult(PreprocessResultValues{
		Query:         "検索してください",
		ComparisonKey: querynormalization.ComparisonKey("検索してください"),
	})
	if err != nil {
		t.Fatalf("試験用 preprocess result を作成できません: %v", err)
	}
	scope, err := NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("試験用 ID scope を作成できません: %v", err)
	}

	tests := []CandidateGenerationValues{
		{
			ProfileID:      "other",
			ProfileVersion: metadata.ProfileVersion(),
		},
		{
			ProfileID:      metadata.ProfileID(),
			ProfileVersion: "core-v2",
		},
	}
	for _, values := range tests {
		generation, generationErr := NewCandidateGeneration(values)
		if generationErr != nil {
			t.Fatalf("試験用 generation を作成できません: %v", generationErr)
		}
		profile := fixedQueryProfile{
			metadata:   metadata,
			generation: generation,
		}
		if _, collectErr := CollectProfileCandidates(
			profile,
			preprocessed,
			scope,
		); collectErr == nil {
			t.Fatalf(
				"profile と異なる generation を受理しました: %#v",
				values,
			)
		}
	}
}

func mustCollectionProfileMetadata(t *testing.T) QueryProfileMetadata {
	t.Helper()
	target, err := NewQueryProfileTarget(QueryProfileTargetValues{
		Task:      TaskSearch,
		Resource:  ResourceLaw,
		InputKind: InputKindLawSearch,
	})
	if err != nil {
		t.Fatalf("試験用 profile target を作成できません: %v", err)
	}
	score := mustTestQueryScorePolicy(t)
	selection, err := NewQuerySelectionPolicy(QuerySelectionPolicyValues{
		SingleThreshold:           120,
		MinimumExecutionThreshold: 80,
		SingleMargin:              25,
		HedgeMargin:               10,
		ScoreMinimum:              score.Minimum(),
		ScoreMaximum:              score.Maximum(),
	})
	if err != nil {
		t.Fatalf("試験用 selection policy を作成できません: %v", err)
	}
	metadata, err := NewQueryProfileMetadata(QueryProfileMetadataValues{
		SchemaVersion:              1,
		ProfileID:                  "core",
		ProfileVersion:             "core-v1",
		CueSetVersion:              "core-cues-v1",
		LawNameLexiconVersion:      "law-name-v1",
		LegalConceptLexiconVersion: "legal-concept-v1",
		Targets:                    []QueryProfileTarget{target},
		Score:                      score,
		Selection:                  selection,
		TieBreak: []QueryTieBreak{
			QueryTieBreakEvidenceSet,
			QueryTieBreakStepCount,
			QueryTieBreakMeaningSignature,
			QueryTieBreakSourcePosition,
		},
	})
	if err != nil {
		t.Fatalf("試験用 profile metadata を作成できません: %v", err)
	}
	return metadata
}
