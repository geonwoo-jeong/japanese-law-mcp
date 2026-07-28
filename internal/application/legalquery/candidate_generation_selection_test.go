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
