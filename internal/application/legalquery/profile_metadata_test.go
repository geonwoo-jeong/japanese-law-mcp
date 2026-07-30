package legalquery

import (
	"slices"
	"testing"
)

func TestQueryProfileMetadataはscore選択辞書版を不変に保持する(t *testing.T) {
	t.Parallel()

	score := mustTestQueryScorePolicy(t)
	selection, err := NewQuerySelectionPolicy(QuerySelectionPolicyValues{
		SingleThreshold:           120,
		MinimumExecutionThreshold: 80,
		SingleMargin:              25,
		HedgeMargin:               10,
		BranchRetentionMargin:     0,
		BranchRetentionPresent:    false,
		ScoreMinimum:              0,
		ScoreMaximum:              405,
	})
	if err != nil {
		t.Fatalf("selection policy を作成できません: %v", err)
	}
	target, err := NewQueryProfileTarget(QueryProfileTargetValues{
		Task:      TaskSearch,
		Resource:  ResourceLaw,
		InputKind: InputKindLawSearch,
	})
	if err != nil {
		t.Fatalf("target を作成できません: %v", err)
	}
	metadata, err := NewQueryProfileMetadata(QueryProfileMetadataValues{
		SchemaVersion:              1,
		ProfileID:                  "core",
		ProfileVersion:             "core-2026-07-28-1",
		RankingVersion:             "ranking-2026-07-28-1",
		CueSetVersion:              "core-cues-2026-07-28-1",
		LawNameLexiconVersion:      "law-name-2026-07-27",
		LegalConceptLexiconVersion: "legal-concept-2026-07-28-2",
		Targets:                    []QueryProfileTarget{target},
		Score:                      score,
		Selection:                  selection,
		TieBreak: []QueryTieBreak{
			QueryTieBreakEvidenceSet,
			QueryTieBreakStepCount,
			QueryTieBreakMeaningSignature,
			QueryTieBreakSourcePosition,
		},
		ConditionalTieBreaks: map[ConditionalTieBreakName][]QueryTieBreak{
			ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit: {
				QueryTieBreakEvidenceSet,
				QueryTieBreakStepCount,
				QueryTieBreakSourcePosition,
				QueryTieBreakMeaningSignature,
			},
		},
	})
	if err != nil {
		t.Fatalf("SOT-ENG-025: metadata のエラー = %v", err)
	}
	if metadata.ProfileID() != "core" ||
		metadata.ProfileVersion() != "core-2026-07-28-1" ||
		metadata.RankingVersion() != "ranking-2026-07-28-1" ||
		metadata.CueSetVersion() != "core-cues-2026-07-28-1" ||
		metadata.LawNameLexiconVersion() != "law-name-2026-07-27" ||
		metadata.LegalConceptLexiconVersion() != "legal-concept-2026-07-28-2" {
		t.Fatalf("metadata = %#v", metadata)
	}
	targets := metadata.Targets()
	tieBreak := metadata.TieBreak()
	conditional := metadata.ConditionalTieBreaks()
	conditionalTieBreak := conditional[ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit]
	targets[0] = QueryProfileTarget{}
	tieBreak[0] = QueryTieBreakSourcePosition
	conditionalTieBreak[0] = QueryTieBreakMeaningSignature
	nextConditional := metadata.ConditionalTieBreaks()
	nextConditionalTieBreak := nextConditional[ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit]
	if metadata.Targets()[0].InputKind() != InputKindLawSearch ||
		metadata.TieBreak()[0] != QueryTieBreakEvidenceSet ||
		nextConditionalTieBreak[0] != QueryTieBreakEvidenceSet {
		t.Fatal("SOT-ENG-025: getter から metadata を変更できました")
	}
	if got, exists := metadata.Score().Weight(EvidenceOfficialAlias); !exists ||
		got != 40 {
		t.Fatalf("official_alias weight = %d, %t", got, exists)
	}
	if value, present := metadata.Selection().BranchRetentionMargin(); present ||
		value != 0 {
		t.Fatalf("schemaVersion1 の branchRetentionMargin = (%d, %t)", value, present)
	}
}

func TestQueryScorePolicyは全根拠の固定順と境界を要求する(t *testing.T) {
	t.Parallel()

	valid := queryScorePolicyValues()
	if _, err := NewQueryScorePolicy(valid); err != nil {
		t.Fatalf("SOT-ARCH-023: valid score policy のエラー = %v", err)
	}

	tests := map[string]func(QueryScorePolicyValues) QueryScorePolicyValues{
		"根拠欠落": func(values QueryScorePolicyValues) QueryScorePolicyValues {
			values.EvidenceWeights = values.EvidenceWeights[:len(values.EvidenceWeights)-1]
			return values
		},
		"根拠順違反": func(values QueryScorePolicyValues) QueryScorePolicyValues {
			values.EvidenceWeights[0], values.EvidenceWeights[1] =
				values.EvidenceWeights[1], values.EvidenceWeights[0]
			return values
		},
		"weight 非正": func(values QueryScorePolicyValues) QueryScorePolicyValues {
			values.EvidenceWeights[0].weight = 0
			return values
		},
		"confidence 逆転": func(values QueryScorePolicyValues) QueryScorePolicyValues {
			values.HighConfidenceAt = values.MediumConfidenceAt - 1
			return values
		},
		"最大値超過": func(values QueryScorePolicyValues) QueryScorePolicyValues {
			values.HighConfidenceAt = values.Maximum + 1
			return values
		},
	}
	for name, mutate := range tests {
		name := name
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewQueryScorePolicy(
				mutate(queryScorePolicyValues()),
			); err == nil {
				t.Fatal("不正な score policy を受理しました")
			}
		})
	}
}

func TestQueryProfileMetadataは重複targetと不完全tieBreakを拒否する(t *testing.T) {
	t.Parallel()

	target, err := NewQueryProfileTarget(QueryProfileTargetValues{
		Task:      TaskSearch,
		Resource:  ResourceLaw,
		InputKind: InputKindLawSearch,
	})
	if err != nil {
		t.Fatalf("target を作成できません: %v", err)
	}
	score := mustTestQueryScorePolicy(t)
	selection, err := NewQuerySelectionPolicy(QuerySelectionPolicyValues{
		SingleThreshold:           120,
		MinimumExecutionThreshold: 80,
		SingleMargin:              25,
		HedgeMargin:               10,
		BranchRetentionMargin:     0,
		BranchRetentionPresent:    false,
		ScoreMinimum:              score.Minimum(),
		ScoreMaximum:              score.Maximum(),
	})
	if err != nil {
		t.Fatalf("selection policy を作成できません: %v", err)
	}
	base := QueryProfileMetadataValues{
		SchemaVersion:              1,
		ProfileID:                  "core",
		ProfileVersion:             "core-v1",
		RankingVersion:             "ranking-v1",
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
	}
	duplicated := base
	duplicated.Targets = []QueryProfileTarget{target, target}
	if _, err := NewQueryProfileMetadata(duplicated); err == nil {
		t.Fatal("重複 target を受理しました")
	}
	incomplete := base
	incomplete.TieBreak = []QueryTieBreak{QueryTieBreakEvidenceSet}
	if _, err := NewQueryProfileMetadata(incomplete); err == nil {
		t.Fatal("不完全な tie-break を受理しました")
	}
}

func TestQueryProfileMetadataはschemaVersionとbranchRetentionの存在状態を区別する(
	t *testing.T,
) {
	t.Parallel()

	score := mustTestQueryScorePolicy(t)
	target, err := NewQueryProfileTarget(QueryProfileTargetValues{
		Task:      TaskSearch,
		Resource:  ResourceLaw,
		InputKind: InputKindLawSearch,
	})
	if err != nil {
		t.Fatalf("target を作成できません: %v", err)
	}
	base := QueryProfileMetadataValues{
		SchemaVersion:              1,
		ProfileID:                  "core",
		ProfileVersion:             "core-v1",
		RankingVersion:             "ranking-v1",
		CueSetVersion:              "core-cues-v1",
		LawNameLexiconVersion:      "law-name-v1",
		LegalConceptLexiconVersion: "legal-concept-v1",
		Targets:                    []QueryProfileTarget{target},
		Score:                      score,
		TieBreak: []QueryTieBreak{
			QueryTieBreakEvidenceSet,
			QueryTieBreakStepCount,
			QueryTieBreakMeaningSignature,
			QueryTieBreakSourcePosition,
		},
	}

	v1Selection, err := NewQuerySelectionPolicy(QuerySelectionPolicyValues{
		SingleThreshold:           120,
		MinimumExecutionThreshold: 80,
		SingleMargin:              25,
		HedgeMargin:               10,
		BranchRetentionMargin:     0,
		BranchRetentionPresent:    false,
		ScoreMinimum:              score.Minimum(),
		ScoreMaximum:              score.Maximum(),
	})
	if err != nil {
		t.Fatalf("schemaVersion1 selection を作成できません: %v", err)
	}
	base.Selection = v1Selection
	v1, err := NewQueryProfileMetadata(base)
	if err != nil {
		t.Fatalf("schemaVersion1 metadata を作成できません: %v", err)
	}
	if value, present := v1.Selection().BranchRetentionMargin(); present ||
		value != 0 {
		t.Fatalf("schemaVersion1 の branchRetentionMargin = (%d, %t)", value, present)
	}

	v1WithBranch := base
	v1WithBranch.Selection, err = NewQuerySelectionPolicy(QuerySelectionPolicyValues{
		SingleThreshold:           120,
		MinimumExecutionThreshold: 80,
		SingleMargin:              25,
		HedgeMargin:               10,
		BranchRetentionMargin:     0,
		BranchRetentionPresent:    true,
		ScoreMinimum:              score.Minimum(),
		ScoreMaximum:              score.Maximum(),
	})
	if err != nil {
		t.Fatalf("schemaVersion1 branch selection を作成できません: %v", err)
	}
	if _, err := NewQueryProfileMetadata(v1WithBranch); err == nil {
		t.Fatal("schemaVersion1 に branchRetentionMargin を受理しました")
	}

	v2Selection, err := NewQuerySelectionPolicy(QuerySelectionPolicyValues{
		SingleThreshold:           120,
		MinimumExecutionThreshold: 80,
		SingleMargin:              25,
		HedgeMargin:               10,
		BranchRetentionMargin:     0,
		BranchRetentionPresent:    true,
		ScoreMinimum:              score.Minimum(),
		ScoreMaximum:              score.Maximum(),
	})
	if err != nil {
		t.Fatalf("schemaVersion2 selection を作成できません: %v", err)
	}
	v2Values := base
	v2Values.SchemaVersion = 2
	v2Values.Selection = v2Selection
	v2, err := NewQueryProfileMetadata(v2Values)
	if err != nil {
		t.Fatalf("schemaVersion2 metadata を作成できません: %v", err)
	}
	if value, present := v2.Selection().BranchRetentionMargin(); !present ||
		value != 0 {
		t.Fatalf("schemaVersion2 の branchRetentionMargin = (%d, %t)", value, present)
	}

	v2MissingBranch := v2Values
	v2MissingBranch.Selection = v1Selection
	if _, err := NewQueryProfileMetadata(v2MissingBranch); err == nil {
		t.Fatal("schemaVersion2 に branchRetentionMargin 欠落を受理しました")
	}

	hiddenBranch, err := NewQuerySelectionPolicy(QuerySelectionPolicyValues{
		SingleThreshold:           120,
		MinimumExecutionThreshold: 80,
		SingleMargin:              25,
		HedgeMargin:               10,
		BranchRetentionMargin:     1,
		BranchRetentionPresent:    false,
		ScoreMinimum:              score.Minimum(),
		ScoreMaximum:              score.Maximum(),
	})
	if err == nil {
		t.Fatalf(
			"profile-metadata-branch-retention-presence: 隠れた値を受理しました: %#v",
			hiddenBranch,
		)
	}
}

func mustTestQueryScorePolicy(t *testing.T) QueryScorePolicy {
	t.Helper()
	policy, err := NewQueryScorePolicy(queryScorePolicyValues())
	if err != nil {
		t.Fatalf("score policy を作成できません: %v", err)
	}
	return policy
}

func queryScorePolicyValues() QueryScorePolicyValues {
	codes := []EvidenceCode{
		EvidenceOfficialIdentifier,
		EvidenceStructuredReference,
		EvidenceExplicitTask,
		EvidenceExplicitResource,
		EvidenceOfficialAlias,
		EvidenceLegalConcept,
		EvidenceMorphologicalContext,
		EvidenceUniqueTypoCorrection,
		EvidenceGeneralTerm,
	}
	weights := []int{90, 80, 60, 50, 40, 35, 25, 15, 10}
	values := make([]QueryEvidenceWeight, 0, len(codes))
	for index, code := range codes {
		values = append(values, QueryEvidenceWeight{
			code:   code,
			weight: weights[index],
		})
	}
	if !slices.IsSortedFunc(values, func(left, right QueryEvidenceWeight) int {
		leftRank, _ := evidenceRank(left.code)
		rightRank, _ := evidenceRank(right.code)
		return leftRank - rightRank
	}) {
		panic("試験用 evidence weight が固定順ではありません")
	}
	return QueryScorePolicyValues{
		Minimum:            0,
		Maximum:            405,
		EvidenceWeights:    values,
		HighConfidenceAt:   130,
		MediumConfidenceAt: 80,
	}
}
