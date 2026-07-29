package legalquery

import "testing"

func TestCandidateGenerationはprofileの選択関係を不変に保持する(
	t *testing.T,
) {
	t.Parallel()

	first := planTestCandidate(
		t,
		"candidate-1-1",
		130,
		nil,
		planTestStep(t, "法令検索", "step-1-1-1"),
	)
	second := planTestCandidate(
		t,
		"candidate-1-2",
		125,
		nil,
		planTestStep(t, "法令本文検索", "step-1-2-1"),
	)
	pair, err := NewCandidateHedgePair(CandidateHedgePairValues{
		FirstCandidateID:  first.CandidateID(),
		SecondCandidateID: second.CandidateID(),
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-026: hedge pair を作成できません: %v", err)
	}
	generation, err := NewCandidateGeneration(CandidateGenerationValues{
		ProfileID:      "core",
		ProfileVersion: "core-v1",
		RankingVersion: "ranking-v1",
		Candidates:     []LegalQueryCandidate{first, second},
		SelectionMode:  QuerySelectionModeAutomatic,
		HedgePairs:     []CandidateHedgePair{pair},
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-026: contribution を作成できません: %v", err)
	}
	if generation.RankingVersion() != "ranking-v1" ||
		generation.SelectionMode() != QuerySelectionModeAutomatic {
		t.Fatalf(
			"SOT-MODEL-026: ranking/mode = %q/%q",
			generation.RankingVersion(),
			generation.SelectionMode(),
		)
	}
	pairs := generation.HedgePairs()
	if len(pairs) != 1 ||
		pairs[0].FirstCandidateID() != first.CandidateID() ||
		pairs[0].SecondCandidateID() != second.CandidateID() {
		t.Fatalf("SOT-MODEL-026: hedge pairs = %#v", pairs)
	}
	pairs[0] = CandidateHedgePair{}
	if len(generation.HedgePairs()) != 1 ||
		generation.HedgePairs()[0].FirstCandidateID() != first.CandidateID() {
		t.Fatal("SOT-MODEL-026: getter から hedge pair を変更できました")
	}
}

func TestCandidateGenerationは不正な選択関係を拒否する(t *testing.T) {
	t.Parallel()

	first := planTestCandidate(
		t,
		"candidate-1-1",
		130,
		nil,
		planTestStep(t, "法令検索", "step-1-1-1"),
	)
	second := planTestCandidate(
		t,
		"candidate-1-2",
		125,
		nil,
		planTestStep(t, "法令本文検索", "step-1-2-1"),
	)
	base := CandidateGenerationValues{
		ProfileID:      "core",
		ProfileVersion: "core-v1",
		RankingVersion: "ranking-v1",
		Candidates:     []LegalQueryCandidate{first, second},
		SelectionMode:  QuerySelectionModeAutomatic,
	}
	pair := func(firstID string, secondID string) CandidateHedgePair {
		value, err := NewCandidateHedgePair(CandidateHedgePairValues{
			FirstCandidateID:  firstID,
			SecondCandidateID: secondID,
		})
		if err != nil {
			t.Fatalf("試験用 hedge pair を作成できません: %v", err)
		}
		return value
	}

	tests := map[string]CandidateGenerationValues{
		"ranking version の欠落": func() CandidateGenerationValues {
			values := base
			values.RankingVersion = ""
			return values
		}(),
		"未知の selection mode": func() CandidateGenerationValues {
			values := base
			values.SelectionMode = QuerySelectionMode("unknown")
			return values
		}(),
		"存在しない候補": func() CandidateGenerationValues {
			values := base
			values.HedgePairs = []CandidateHedgePair{
				pair(first.CandidateID(), "candidate-1-3"),
			}
			return values
		}(),
		"逆順": func() CandidateGenerationValues {
			values := base
			values.HedgePairs = []CandidateHedgePair{
				pair(second.CandidateID(), first.CandidateID()),
			}
			return values
		}(),
		"重複": func() CandidateGenerationValues {
			values := base
			current := pair(first.CandidateID(), second.CandidateID())
			values.HedgePairs = []CandidateHedgePair{current, current}
			return values
		}(),
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewCandidateGeneration(values); err == nil {
				t.Fatal("SOT-MODEL-026: 不正な contribution を受理しました")
			}
		})
	}
}

func TestCandidateGenerationは五stepのhedgePairを拒否する(t *testing.T) {
	t.Parallel()

	first := planTestCandidate(
		t,
		"candidate-1-1",
		130,
		nil,
		planTestStep(t, "法令検索", "step-1-1-1"),
		planTestStep(t, "法令本文検索", "step-1-1-2"),
		planTestStep(t, "更新一覧", "step-1-1-3"),
	)
	second := planTestCandidate(
		t,
		"candidate-1-2",
		125,
		nil,
		planTestStep(t, "法令検索", "step-1-2-1"),
		planTestStep(t, "法令本文検索", "step-1-2-2"),
	)
	pair, err := NewCandidateHedgePair(CandidateHedgePairValues{
		FirstCandidateID:  first.CandidateID(),
		SecondCandidateID: second.CandidateID(),
	})
	if err != nil {
		t.Fatalf("試験用 hedge pair を作成できません: %v", err)
	}
	if _, err := NewCandidateGeneration(CandidateGenerationValues{
		ProfileID:      "core",
		ProfileVersion: "core-v1",
		RankingVersion: "ranking-v1",
		Candidates:     []LegalQueryCandidate{first, second},
		SelectionMode:  QuerySelectionModeAutomatic,
		HedgePairs:     []CandidateHedgePair{pair},
	}); err == nil {
		t.Fatal("SOT-MODEL-026: 五 step の hedge pair を受理しました")
	}
}

func TestCandidateGenerationは構造だけの信号を候補と言語信号から分離する(
	t *testing.T,
) {
	t.Parallel()

	base := CandidateGenerationValues{
		ProfileID:      "core",
		ProfileVersion: "core-v2",
		RankingVersion: "ranking-v1",
		Signals: []CandidateGenerationSignal{
			CandidateSignalStandaloneStructuredQuery,
		},
		SelectionMode: QuerySelectionModeAutomatic,
	}
	generation, err := NewCandidateGeneration(base)
	if err != nil {
		t.Fatalf("SOT-MODEL-026: standalone contribution = %v", err)
	}
	if len(generation.Candidates()) != 0 ||
		len(generation.Signals()) != 1 ||
		generation.Signals()[0] !=
			CandidateSignalStandaloneStructuredQuery {
		t.Fatalf("SOT-MODEL-026: generation = %#v", generation)
	}

	withCandidate := base
	withCandidate.Candidates = []LegalQueryCandidate{
		mustServiceSingleCandidate(t),
	}
	if _, err := NewCandidateGeneration(withCandidate); err == nil {
		t.Fatal("SOT-MODEL-026: standalone signal と候補の併存を受理しました")
	}
	withLanguage := base
	withLanguage.Signals = []CandidateGenerationSignal{
		CandidateSignalNonJapaneseQuery,
		CandidateSignalStandaloneStructuredQuery,
	}
	if _, err := NewCandidateGeneration(withLanguage); err == nil {
		t.Fatal("SOT-MODEL-026: standalone signal と言語信号の併存を受理しました")
	}
	withOther := base
	withOther.Signals = []CandidateGenerationSignal{
		CandidateSignalStandaloneStructuredQuery,
		CandidateSignalUnsupportedTaskOrResource,
	}
	if _, err := NewCandidateGeneration(withOther); err == nil {
		t.Fatal("SOT-MODEL-026: standalone signal と他 signal の併存を受理しました")
	}
}

func TestCandidateGenerationはProfile内のStep上限制約を型付きで保持する(
	t *testing.T,
) {
	t.Parallel()

	generation, err := NewCandidateGeneration(CandidateGenerationValues{
		ProfileID:             "core",
		ProfileVersion:        "core-v1",
		RankingVersion:        "ranking-v1",
		SelectionMode:         QuerySelectionModeClarificationRequired,
		CompositionConstraint: QueryCompositionConstraintStepLimitExceeded,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-026: step 上限制約を持つ contribution = %v", err)
	}
	if generation.CompositionConstraint() !=
		QueryCompositionConstraintStepLimitExceeded {
		t.Fatalf(
			"SOT-MODEL-026: compositionConstraint = %q",
			generation.CompositionConstraint(),
		)
	}

	ordinary, err := NewCandidateGeneration(CandidateGenerationValues{
		ProfileID:      "core",
		ProfileVersion: "core-v1",
		RankingVersion: "ranking-v1",
		SelectionMode:  QuerySelectionModeAutomatic,
	})
	if err != nil {
		t.Fatalf("通常 contribution を作成できません: %v", err)
	}
	if ordinary.CompositionConstraint() != QueryCompositionConstraintNone {
		t.Fatalf(
			"未指定の compositionConstraint = %q",
			ordinary.CompositionConstraint(),
		)
	}
}

func TestCandidateGenerationはStep上限制約の部分実行状態を拒否する(
	t *testing.T,
) {
	t.Parallel()

	first := planTestCandidate(
		t,
		"candidate-1-1",
		130,
		nil,
		planTestStep(t, "法令検索", "step-1-1-1"),
	)
	second := planTestCandidate(
		t,
		"candidate-1-2",
		125,
		nil,
		planTestStep(t, "法令本文検索", "step-1-2-1"),
	)
	pair, err := NewCandidateHedgePair(CandidateHedgePairValues{
		FirstCandidateID:  first.CandidateID(),
		SecondCandidateID: second.CandidateID(),
	})
	if err != nil {
		t.Fatalf("試験用 hedge pair を作成できません: %v", err)
	}
	member := mustCandidateCompositionMember(
		t,
		first.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(t, "step-1-1-1", 0),
		},
	)
	base := CandidateGenerationValues{
		ProfileID:             "core",
		ProfileVersion:        "core-v1",
		RankingVersion:        "ranking-v1",
		SelectionMode:         QuerySelectionModeClarificationRequired,
		CompositionConstraint: QueryCompositionConstraintStepLimitExceeded,
	}
	tests := map[string]CandidateGenerationValues{
		"automatic": func() CandidateGenerationValues {
			values := base
			values.SelectionMode = QuerySelectionModeAutomatic
			return values
		}(),
		"candidate": func() CandidateGenerationValues {
			values := base
			values.Candidates = []LegalQueryCandidate{first}
			return values
		}(),
		"hedge pair": func() CandidateGenerationValues {
			values := base
			values.Candidates = []LegalQueryCandidate{first, second}
			values.HedgePairs = []CandidateHedgePair{pair}
			return values
		}(),
		"composition member": func() CandidateGenerationValues {
			values := base
			values.Candidates = []LegalQueryCandidate{first}
			values.CompositionMembers = []QueryCandidateCompositionMember{member}
			return values
		}(),
		"未知の制約": func() CandidateGenerationValues {
			values := base
			values.CompositionConstraint = QueryCompositionConstraint("unknown")
			return values
		}(),
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewCandidateGeneration(values); err == nil {
				t.Fatal("SOT-MODEL-026: step 上限制約の不正な状態を受理しました")
			}
		})
	}
}
