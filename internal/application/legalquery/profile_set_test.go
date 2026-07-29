package legalquery

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"unicode/utf8"
)

func TestQueryProfileSetは共通ranking規則だけを受理する(t *testing.T) {
	t.Parallel()

	base := mustSelectorTestMetadata(
		t,
		"core",
		"core-profile-v1",
		selectorTestRankingVersion,
	)
	second := mustSelectorTestMetadata(
		t,
		"judicial",
		"judicial-profile-v7",
		selectorTestRankingVersion,
	)
	profiles := []QueryProfile{
		selectorTestProfile{metadata: base, selectionMode: QuerySelectionModeAutomatic},
		selectorTestProfile{metadata: second, selectionMode: QuerySelectionModeAutomatic},
	}
	profileSet, err := NewQueryProfileSet(profiles)
	if err != nil {
		t.Fatalf("SOT-ARCH-023: 共通 ranking 規則を拒否しました: %v", err)
	}
	result, err := profileSet.Collect(mustSelectorTestPreprocessResult(t))
	if err != nil {
		t.Fatalf("SOT-ARCH-023: profile contribution の回収に失敗しました: %v", err)
	}
	if result.RankingVersion() != selectorTestRankingVersion {
		t.Fatalf("rankingVersion = %q", result.RankingVersion())
	}

	t.Run("rankingVersionの不一致", func(t *testing.T) {
		t.Parallel()
		different := mustSelectorTestMetadata(
			t,
			"judicial",
			"judicial-profile-v7",
			"ranking-other-v1",
		)
		if _, err := NewQueryProfileSet([]QueryProfile{
			selectorTestProfile{metadata: base, selectionMode: QuerySelectionModeAutomatic},
			selectorTestProfile{metadata: different, selectionMode: QuerySelectionModeAutomatic},
		}); err == nil {
			t.Fatal("SOT-ARCH-023: 異なる rankingVersion の score 比較を受理しました")
		}
	})

	t.Run("score規則の不一致", func(t *testing.T) {
		t.Parallel()
		values := selectorTestMetadataValues(
			t,
			"judicial",
			"judicial-profile-v7",
			selectorTestRankingVersion,
		)
		scoreValues := queryScorePolicyValues()
		scoreValues.EvidenceWeights = append(
			[]QueryEvidenceWeight(nil),
			scoreValues.EvidenceWeights...,
		)
		scoreValues.EvidenceWeights[0].weight++
		scoreValues.Maximum++
		score, scoreErr := NewQueryScorePolicy(scoreValues)
		if scoreErr != nil {
			t.Fatalf("試験用 score policy を作成できません: %v", scoreErr)
		}
		selection, selectionErr := NewQuerySelectionPolicy(
			QuerySelectionPolicyValues{
				SingleThreshold:           120,
				MinimumExecutionThreshold: 80,
				SingleMargin:              25,
				HedgeMargin:               10,
				ScoreMinimum:              score.Minimum(),
				ScoreMaximum:              score.Maximum(),
			},
		)
		if selectionErr != nil {
			t.Fatalf("試験用 selection policy を作成できません: %v", selectionErr)
		}
		values.Score = score
		values.Selection = selection
		different, metadataErr := NewQueryProfileMetadata(values)
		if metadataErr != nil {
			t.Fatalf("試験用 metadata を作成できません: %v", metadataErr)
		}
		if _, err := NewQueryProfileSet([]QueryProfile{
			selectorTestProfile{metadata: base, selectionMode: QuerySelectionModeAutomatic},
			selectorTestProfile{metadata: different, selectionMode: QuerySelectionModeAutomatic},
		}); err == nil {
			t.Fatal("SOT-ARCH-023: 異なる score 規則の contribution を受理しました")
		}
	})

	t.Run("selection規則の不一致", func(t *testing.T) {
		t.Parallel()
		values := selectorTestMetadataValues(
			t,
			"judicial",
			"judicial-profile-v7",
			selectorTestRankingVersion,
		)
		selection, selectionErr := NewQuerySelectionPolicy(
			QuerySelectionPolicyValues{
				SingleThreshold:           121,
				MinimumExecutionThreshold: 80,
				SingleMargin:              25,
				HedgeMargin:               10,
				ScoreMinimum:              values.Score.Minimum(),
				ScoreMaximum:              values.Score.Maximum(),
			},
		)
		if selectionErr != nil {
			t.Fatalf("試験用 selection policy を作成できません: %v", selectionErr)
		}
		values.Selection = selection
		different, metadataErr := NewQueryProfileMetadata(values)
		if metadataErr != nil {
			t.Fatalf("試験用 metadata を作成できません: %v", metadataErr)
		}
		if _, err := NewQueryProfileSet([]QueryProfile{
			selectorTestProfile{metadata: base, selectionMode: QuerySelectionModeAutomatic},
			selectorTestProfile{metadata: different, selectionMode: QuerySelectionModeAutomatic},
		}); err == nil {
			t.Fatal("SOT-ARCH-023: 異なる threshold と margin を受理しました")
		}
	})

	t.Run("tie規則の不一致", func(t *testing.T) {
		t.Parallel()
		different := second
		different.tieBreak = []QueryTieBreak{
			QueryTieBreakStepCount,
			QueryTieBreakEvidenceSet,
			QueryTieBreakMeaningSignature,
			QueryTieBreakSourcePosition,
		}
		if _, err := NewQueryProfileSet([]QueryProfile{
			selectorTestProfile{metadata: base, selectionMode: QuerySelectionModeAutomatic},
			selectorTestProfile{metadata: different, selectionMode: QuerySelectionModeAutomatic},
		}); err == nil {
			t.Fatal("SOT-ARCH-023: 異なる tie-break の contribution を受理しました")
		}
	})
}

func TestQueryProfileSetは集合全体の不透明な版を決定的に作る(
	t *testing.T,
) {
	t.Parallel()

	core := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"core",
			"core-profile-v1",
			selectorTestRankingVersion,
		),
		selectionMode: QuerySelectionModeAutomatic,
	}
	judicial := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"judicial",
			"judicial-profile-v1",
			selectorTestRankingVersion,
		),
		selectionMode: QuerySelectionModeAutomatic,
	}
	first, err := NewQueryProfileSet([]QueryProfile{core, judicial})
	if err != nil {
		t.Fatalf("profile set を作成できません: %v", err)
	}
	second, err := NewQueryProfileSet([]QueryProfile{core, judicial})
	if err != nil {
		t.Fatalf("同じ profile set を再作成できません: %v", err)
	}
	firstResult, err := first.Collect(mustSelectorTestPreprocessResult(t))
	if err != nil {
		t.Fatalf("profile set を集約できません: %v", err)
	}
	secondResult, err := second.Collect(mustSelectorTestPreprocessResult(t))
	if err != nil {
		t.Fatalf("profile set を再集約できません: %v", err)
	}
	if firstResult.ProfileVersion() != secondResult.ProfileVersion() ||
		firstResult.ProfileVersion() == selectorTestRankingVersion ||
		utf8.RuneCountInString(firstResult.ProfileVersion()) == 0 ||
		len(firstResult.ProfileVersion()) > maximumProfileVersionSize {
		t.Fatalf(
			"SOT-MODEL-026: profile set version = %q",
			firstResult.ProfileVersion(),
		)
	}

	judicial.metadata = mustSelectorTestMetadata(
		t,
		"judicial",
		"judicial-profile-v2",
		selectorTestRankingVersion,
	)
	changed, err := NewQueryProfileSet([]QueryProfile{core, judicial})
	if err != nil {
		t.Fatalf("profile 固有版を変更した set を作成できません: %v", err)
	}
	changedResult, err := changed.Collect(mustSelectorTestPreprocessResult(t))
	if err != nil {
		t.Fatalf("変更後の profile set を集約できません: %v", err)
	}
	if changedResult.ProfileVersion() == firstResult.ProfileVersion() {
		t.Fatal("SOT-MODEL-026: profile 固有版の変更が set version に反映されません")
	}

	firstComposition := queryProfileSetVersion(
		[]QueryProfileMetadata{core.metadata, judicial.metadata},
		"composition-test-v1",
	)
	secondComposition := queryProfileSetVersion(
		[]QueryProfileMetadata{core.metadata, judicial.metadata},
		"composition-test-v2",
	)
	if firstComposition == secondComposition {
		t.Fatal("SOT-ARCH-027: compositionVersion の変更が set version に反映されません")
	}
}

func TestQueryProfileSetは構築後のmetadata変更を拒否する(t *testing.T) {
	t.Parallel()

	profile := &selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"core",
			"core-profile-v1",
			selectorTestRankingVersion,
		),
		selectionMode: QuerySelectionModeAutomatic,
	}
	profileSet, err := NewQueryProfileSet([]QueryProfile{profile})
	if err != nil {
		t.Fatalf("profile set を作成できません: %v", err)
	}
	profile.metadata = mustSelectorTestMetadata(
		t,
		"core",
		"core-profile-v2",
		selectorTestRankingVersion,
	)
	if _, err := profileSet.Collect(
		mustSelectorTestPreprocessResult(t),
	); err == nil {
		t.Fatal("SOT-MODEL-026: 構築後に変更された metadata を受理しました")
	}
}

func TestQueryProfileSetはscoreを安定集計する(t *testing.T) {
	t.Parallel()

	firstHigh := mustSelectorTestCandidate(t, "candidate-first-high", 130, nil, 1)
	firstLow := mustSelectorTestCandidate(t, "candidate-first-low", 100, nil, 1)
	secondHigh := mustSelectorTestCandidate(t, "candidate-second-high", 130, nil, 1)
	secondMiddle := mustSelectorTestCandidate(t, "candidate-second-middle", 110, nil, 1)
	first := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"core",
			"core-profile-v1",
			selectorTestRankingVersion,
		),
		candidates:    []LegalQueryCandidate{firstHigh, firstLow},
		selectionMode: QuerySelectionModeAutomatic,
	}
	second := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"judicial",
			"judicial-profile-v1",
			selectorTestRankingVersion,
		),
		candidates:    []LegalQueryCandidate{secondHigh, secondMiddle},
		selectionMode: QuerySelectionModeAutomatic,
	}
	profileSet, err := NewQueryProfileSet([]QueryProfile{first, second})
	if err != nil {
		t.Fatalf("profile set を作成できません: %v", err)
	}
	result, err := profileSet.Collect(mustSelectorTestPreprocessResult(t))
	if err != nil {
		t.Fatalf("SOT-ARCH-023: contribution の安定集計に失敗しました: %v", err)
	}
	got := result.RankedCandidates()
	want := []string{
		"candidate-first-high",
		"candidate-second-high",
		"candidate-second-middle",
		"candidate-first-low",
	}
	if len(got) != len(want) {
		t.Fatalf("rankedCandidates の件数 = %d", len(got))
	}
	for index, candidate := range got {
		if candidate.CandidateID() != want[index] {
			t.Fatalf(
				"SOT-ARCH-023: rankedCandidates[%d] = %q; want %q",
				index,
				candidate.CandidateID(),
				want[index],
			)
		}
	}

	got[0].candidateID = "changed"
	if result.RankedCandidates()[0].CandidateID() != want[0] {
		t.Fatal("SOT-ENG-025: getter から集計済み候補を変更できました")
	}
}

func TestQueryProfileSetはprofile間の同一意味を拒否する(t *testing.T) {
	t.Parallel()

	input, err := NewLawSearchIntentV1(LawSearchIntentV1Values{
		Query: "同一の法令検索",
	})
	if err != nil {
		t.Fatalf("試験用 logical input を作成できません: %v", err)
	}
	first := mustProfileSetCandidateWithInput(
		t,
		"candidate-first",
		"step-first",
		130,
		ConfidenceHigh,
		input,
	)
	second := mustProfileSetCandidateWithInput(
		t,
		"candidate-second",
		"step-second",
		120,
		ConfidenceMedium,
		input,
	)
	profileSet, err := NewQueryProfileSet([]QueryProfile{
		selectorTestProfile{
			metadata: mustSelectorTestMetadata(
				t,
				"core",
				"core-profile-v1",
				selectorTestRankingVersion,
			),
			candidates:    []LegalQueryCandidate{first},
			selectionMode: QuerySelectionModeAutomatic,
		},
		selectorTestProfile{
			metadata: mustSelectorTestMetadata(
				t,
				"judicial",
				"judicial-profile-v1",
				selectorTestRankingVersion,
			),
			candidates:    []LegalQueryCandidate{second},
			selectionMode: QuerySelectionModeAutomatic,
		},
	})
	if err != nil {
		t.Fatalf("意味衝突試験用 profile set を作成できません: %v", err)
	}
	if _, err := profileSet.Collect(
		mustSelectorTestPreprocessResult(t),
	); err == nil {
		t.Fatal("SOT-MODEL-026: profile 間の同一意味を黙って統合しました")
	}
}

func TestQueryProfileSetは全profileを横断した候補上限を守る(t *testing.T) {
	t.Parallel()

	firstCandidates := selectorTestCandidates(t, "first", 8, 300)
	secondCandidates := selectorTestCandidates(t, "second", 8, 200)
	first := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"core",
			"core-profile-v1",
			selectorTestRankingVersion,
		),
		candidates:    firstCandidates,
		selectionMode: QuerySelectionModeAutomatic,
	}
	second := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"judicial",
			"judicial-profile-v1",
			selectorTestRankingVersion,
		),
		candidates:    secondCandidates,
		selectionMode: QuerySelectionModeAutomatic,
	}
	profileSet, err := NewQueryProfileSet([]QueryProfile{first, second})
	if err != nil {
		t.Fatalf("profile set を作成できません: %v", err)
	}
	result, err := profileSet.Collect(mustSelectorTestPreprocessResult(t))
	if err != nil {
		t.Fatalf("SOT-MODEL-023: 全体で十六候補を拒否しました: %v", err)
	}
	if len(result.RankedCandidates()) != MaxRankedCandidates {
		t.Fatalf("rankedCandidates の件数 = %d", len(result.RankedCandidates()))
	}

	second.candidates = selectorTestCandidates(t, "second", 9, 200)
	overLimit, err := NewQueryProfileSet([]QueryProfile{first, second})
	if err != nil {
		t.Fatalf("上限試験用 profile set を作成できません: %v", err)
	}
	if _, err := overLimit.Collect(mustSelectorTestPreprocessResult(t)); err == nil {
		t.Fatal("SOT-MODEL-023: 十七候補を切り捨てて受理しました")
	}
}

func TestQueryProfileSetは全profileを横断したID重複を拒否する(t *testing.T) {
	t.Parallel()

	firstMetadata := mustSelectorTestMetadata(
		t,
		"core",
		"core-profile-v1",
		selectorTestRankingVersion,
	)
	secondMetadata := mustSelectorTestMetadata(
		t,
		"judicial",
		"judicial-profile-v1",
		selectorTestRankingVersion,
	)
	t.Run("candidateId", func(t *testing.T) {
		t.Parallel()
		duplicate := mustSelectorTestCandidate(t, "candidate-duplicate", 120, nil, 1)
		profileSet, err := NewQueryProfileSet([]QueryProfile{
			selectorTestProfile{
				metadata:      firstMetadata,
				candidates:    []LegalQueryCandidate{duplicate},
				selectionMode: QuerySelectionModeAutomatic,
			},
			selectorTestProfile{
				metadata:      secondMetadata,
				candidates:    []LegalQueryCandidate{duplicate},
				selectionMode: QuerySelectionModeAutomatic,
			},
		})
		if err != nil {
			t.Fatalf("profile set を作成できません: %v", err)
		}
		if _, err := profileSet.Collect(mustSelectorTestPreprocessResult(t)); err == nil {
			t.Fatal("SOT-MODEL-023: profile 間の candidateId 重複を受理しました")
		}
	})

	t.Run("stepId", func(t *testing.T) {
		t.Parallel()
		first := planTestCandidate(
			t,
			"candidate-first",
			130,
			nil,
			planTestStep(t, "法令検索", "step-duplicate"),
		)
		second := planTestCandidate(
			t,
			"candidate-second",
			120,
			nil,
			planTestStep(t, "法令検索", "step-duplicate"),
		)
		profileSet, err := NewQueryProfileSet([]QueryProfile{
			selectorTestProfile{
				metadata:      firstMetadata,
				candidates:    []LegalQueryCandidate{first},
				selectionMode: QuerySelectionModeAutomatic,
			},
			selectorTestProfile{
				metadata:      secondMetadata,
				candidates:    []LegalQueryCandidate{second},
				selectionMode: QuerySelectionModeAutomatic,
			},
		})
		if err != nil {
			t.Fatalf("profile set を作成できません: %v", err)
		}
		if _, err := profileSet.Collect(mustSelectorTestPreprocessResult(t)); err == nil {
			t.Fatal("SOT-MODEL-023: profile 間の stepId 重複を受理しました")
		}
	})
}

func TestQueryProfileSetはprofile順のIDscopeを割り当てる(t *testing.T) {
	t.Parallel()

	profiles := make([]QueryProfile, 0, 2)
	for index, profileID := range []string{"core", "judicial"} {
		profileID := profileID
		profileVersion := fmt.Sprintf("%s-profile-v1", profileID)
		profile := selectorTestProfile{
			metadata: mustSelectorTestMetadata(
				t,
				profileID,
				profileVersion,
				selectorTestRankingVersion,
			),
			selectionMode: QuerySelectionModeAutomatic,
		}
		profile.generate = func(
			scope CandidateIDScope,
		) (QueryProfileContribution, error) {
			input, err := NewLawSearchIntentV1(LawSearchIntentV1Values{
				Query: profileID + "の試験検索",
			})
			if err != nil {
				return QueryProfileContribution{}, err
			}
			candidate, err := AssembleLegalQueryCandidate(CandidateAssemblyValues{
				IDScope:          scope,
				CandidateOrdinal: 1,
				SemanticScore:    120 - index,
				Confidence:       ConfidenceMedium,
				EvidenceCodes:    []EvidenceCode{EvidenceExplicitTask},
				LogicalInputs:    []LogicalInput{input},
			})
			if err != nil {
				return QueryProfileContribution{}, err
			}
			return NewCandidateGeneration(QueryProfileContributionValues{
				ProfileID:      profileID,
				ProfileVersion: profileVersion,
				RankingVersion: selectorTestRankingVersion,
				Candidates:     []LegalQueryCandidate{candidate},
				SelectionMode:  QuerySelectionModeAutomatic,
			})
		}
		profiles = append(profiles, profile)
	}
	profileSet, err := NewQueryProfileSet(profiles)
	if err != nil {
		t.Fatalf("profile set を作成できません: %v", err)
	}
	result, err := profileSet.Collect(mustSelectorTestPreprocessResult(t))
	if err != nil {
		t.Fatalf("SOT-MODEL-023: profile scope の回収に失敗しました: %v", err)
	}
	ids := []string{
		result.RankedCandidates()[0].CandidateID(),
		result.RankedCandidates()[1].CandidateID(),
	}
	slices.Sort(ids)
	if !slices.Equal(ids, []string{"candidate-1-1", "candidate-2-1"}) {
		t.Fatalf("SOT-MODEL-023: profile ごとの candidate ID = %#v", ids)
	}
}

func TestQueryProfileSetはselectionModeをfailClosedで集約する(t *testing.T) {
	t.Parallel()

	auto := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"core",
			"core-profile-v1",
			selectorTestRankingVersion,
		),
		selectionMode: QuerySelectionModeAutomatic,
	}
	clarification := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"judicial",
			"judicial-profile-v1",
			selectorTestRankingVersion,
		),
		selectionMode: QuerySelectionModeClarificationRequired,
	}
	profileSet, err := NewQueryProfileSet([]QueryProfile{auto, clarification})
	if err != nil {
		t.Fatalf("profile set を作成できません: %v", err)
	}
	result, err := profileSet.Collect(mustSelectorTestPreprocessResult(t))
	if err != nil {
		t.Fatalf("selection mode の集約に失敗しました: %v", err)
	}
	if result.SelectionMode() != QuerySelectionModeClarificationRequired {
		t.Fatalf("selectionMode = %q", result.SelectionMode())
	}

	invalid := auto
	invalid.selectionMode = QuerySelectionMode("unknown")
	invalidSet, err := NewQueryProfileSet([]QueryProfile{invalid})
	if err != nil {
		t.Fatalf("不正 mode 試験用 profile set を作成できません: %v", err)
	}
	if _, err := invalidSet.Collect(mustSelectorTestPreprocessResult(t)); err == nil {
		t.Fatal("SOT-ARCH-025: 未知の selection mode を受理しました")
	}
}

func TestQueryProfileSetは不正なhedge組を拒否する(t *testing.T) {
	t.Parallel()

	first := mustSelectorTestCandidate(t, "candidate-first", 110, nil, 3)
	second := mustSelectorTestCandidate(t, "candidate-second", 105, nil, 2)
	pair := mustSelectorTestHedgePair(t, first, second)
	if _, err := selectorTestProfileSetResult(
		t,
		[]LegalQueryCandidate{first, second},
		nil,
		QuerySelectionModeAutomatic,
		[]CandidateHedgePair{pair},
	); err == nil {
		t.Fatal("SOT-MODEL-023: 合計五 step の hedge 組を受理しました")
	}

	missing, err := NewCandidateHedgePair(CandidateHedgePairValues{
		FirstCandidateID:  first.CandidateID(),
		SecondCandidateID: "candidate-missing",
	})
	if err != nil {
		t.Fatalf("参照不整合試験用 hedge pair を作成できません: %v", err)
	}
	if _, err := selectorTestProfileSetResult(
		t,
		[]LegalQueryCandidate{first},
		nil,
		QuerySelectionModeAutomatic,
		[]CandidateHedgePair{missing},
	); err == nil {
		t.Fatal("SOT-ARCH-023: 存在しない候補を参照する hedge 組を受理しました")
	}
}

func TestQueryProfileSetはprofile失敗時に部分結果を返さない(t *testing.T) {
	t.Parallel()

	first := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"core",
			"core-profile-v1",
			selectorTestRankingVersion,
		),
		candidates: []LegalQueryCandidate{
			mustSelectorTestCandidate(t, "candidate-core", 120, nil, 1),
		},
		selectionMode: QuerySelectionModeAutomatic,
	}
	second := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"judicial",
			"judicial-profile-v1",
			selectorTestRankingVersion,
		),
		selectionMode:   QuerySelectionModeAutomatic,
		generationError: errors.New("試験用 profile エラー"),
	}
	profileSet, err := NewQueryProfileSet([]QueryProfile{first, second})
	if err != nil {
		t.Fatalf("profile set を作成できません: %v", err)
	}
	result, err := profileSet.Collect(mustSelectorTestPreprocessResult(t))
	if err == nil {
		t.Fatal("SOT-ARCH-022: profile エラーから部分計画を作成しました")
	}
	if len(result.RankedCandidates()) != 0 {
		t.Fatal("SOT-ARCH-022: 失敗時に部分候補を返しました")
	}
}

func selectorTestCandidates(
	t *testing.T,
	prefix string,
	count int,
	firstScore int,
) []LegalQueryCandidate {
	t.Helper()
	candidates := make([]LegalQueryCandidate, 0, count)
	for index := 0; index < count; index++ {
		candidates = append(candidates, mustSelectorTestCandidate(
			t,
			fmt.Sprintf("candidate-%s-%02d", prefix, index+1),
			firstScore-index,
			nil,
			1,
		))
	}
	return candidates
}

func mustProfileSetCandidateWithInput(
	t *testing.T,
	candidateID string,
	stepID string,
	score int,
	confidence Confidence,
	input LogicalInput,
) LegalQueryCandidate {
	t.Helper()
	specification, exists := stepSpecificationFor(input.InputKind())
	if !exists {
		t.Fatal("試験用 logical input の能力対応がありません")
	}
	step, err := NewLegalQueryCandidateStep(LegalQueryCandidateStepValues{
		StepID:                 stepID,
		Task:                   specification.task,
		Resource:               specification.resource,
		CapabilityID:           specification.capabilityID,
		CapabilityMajorVersion: specification.majorVersion,
		InputKind:              input.InputKind(),
		LogicalInput:           input,
	})
	if err != nil {
		t.Fatalf("試験用 step を作成できません: %v", err)
	}
	candidate, err := NewLegalQueryCandidate(LegalQueryCandidateValues{
		CandidateID:    candidateID,
		SemanticScore:  score,
		Confidence:     confidence,
		EvidenceCodes:  []EvidenceCode{EvidenceExplicitTask},
		ConceptSources: []LegalConceptSource{},
		RequiredPacks:  []string{},
		Steps:          []LegalQueryCandidateStep{step},
	})
	if err != nil {
		t.Fatalf("試験用 candidate を作成できません: %v", err)
	}
	return candidate
}
