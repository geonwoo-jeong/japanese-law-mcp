package legalquery

import "testing"

func TestQueryCandidateCompositionMemberは位置付きstepを不変に保持する(
	t *testing.T,
) {
	t.Parallel()

	origins := []QueryCandidateStepOrigin{
		mustCandidateStepOrigin(t, "step-1-1-1", 3),
		mustCandidateStepOrigin(t, "step-1-1-2", 21),
	}
	member, err := NewQueryCandidateCompositionMember(
		QueryCandidateCompositionMemberValues{
			CandidateID: "candidate-1-1",
			Role:        QueryCandidateCompositionRoleRequiredMember,
			StepOrigins: origins,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-028: composition member を作成できません: %v", err)
	}
	origins[0] = QueryCandidateStepOrigin{}

	if member.CandidateID() != "candidate-1-1" ||
		member.Role() != QueryCandidateCompositionRoleRequiredMember {
		t.Fatalf(
			"SOT-MODEL-028: candidate/role = %q/%q",
			member.CandidateID(),
			member.Role(),
		)
	}
	got := member.StepOrigins()
	if len(got) != 2 ||
		got[0].StepID() != "step-1-1-1" ||
		got[0].SourceStartByte() != 3 ||
		got[1].StepID() != "step-1-1-2" ||
		got[1].SourceStartByte() != 21 {
		t.Fatalf("SOT-MODEL-028: step origins = %#v", got)
	}
	got[0] = QueryCandidateStepOrigin{}
	if member.StepOrigins()[0].StepID() != "step-1-1-1" {
		t.Fatal("SOT-MODEL-028: getter から step origin を変更できました")
	}
}

func TestQueryCandidateCompositionMemberは構造境界を検証する(
	t *testing.T,
) {
	t.Parallel()

	if _, err := NewQueryCandidateStepOrigin(QueryCandidateStepOriginValues{
		StepID:          "step-1-1-1",
		SourceStartByte: -1,
	}); err == nil {
		t.Fatal("SOT-MODEL-028: 負の sourceStartByte を受理しました")
	}
	large, err := NewQueryCandidateStepOrigin(QueryCandidateStepOriginValues{
		StepID:          "step-1-1-1",
		SourceStartByte: 1 << 30,
	})
	if err != nil || large.SourceStartByte() != 1<<30 {
		t.Fatalf(
			"SOT-MODEL-028: Collect 前の query 境界を model で拒否しました: %v",
			err,
		)
	}

	validOrigin := mustCandidateStepOrigin(t, "step-1-1-1", 0)
	tests := map[string]QueryCandidateCompositionMemberValues{
		"candidate ID の欠落": {
			Role:        QueryCandidateCompositionRoleRequiredMember,
			StepOrigins: []QueryCandidateStepOrigin{validOrigin},
		},
		"未知の role": {
			CandidateID: "candidate-1-1",
			Role:        QueryCandidateCompositionRole("optional"),
			StepOrigins: []QueryCandidateStepOrigin{validOrigin},
		},
		"step origin の欠落": {
			CandidateID: "candidate-1-1",
			Role:        QueryCandidateCompositionRoleRequiredMember,
		},
		"step ID の重複": {
			CandidateID: "candidate-1-1",
			Role:        QueryCandidateCompositionRoleRequiredMember,
			StepOrigins: []QueryCandidateStepOrigin{validOrigin, validOrigin},
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewQueryCandidateCompositionMember(values); err == nil {
				t.Fatal("SOT-MODEL-028: 不正な composition member を受理しました")
			}
		})
	}

	if (QueryCandidateStepOrigin{}).Validate() == nil {
		t.Fatal("SOT-MODEL-028: zero-value step origin を受理しました")
	}
	if (QueryCandidateCompositionMember{}).Validate() == nil {
		t.Fatal("SOT-MODEL-028: zero-value composition member を受理しました")
	}
}

func TestCandidateGenerationはcompositionMemberを候補とstepへ対応させる(
	t *testing.T,
) {
	t.Parallel()

	candidate := planTestCandidate(
		t,
		"candidate-1-1",
		130,
		nil,
		planTestStep(t, "法令検索", "step-1-1-1"),
		planTestStep(t, "法令本文検索", "step-1-1-2"),
	)
	member := mustCandidateCompositionMember(
		t,
		candidate.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(t, "step-1-1-1", 6),
			mustCandidateStepOrigin(t, "step-1-1-2", 24),
		},
	)
	members := []QueryCandidateCompositionMember{member}
	generation, err := NewCandidateGeneration(CandidateGenerationValues{
		ProfileID:          "core",
		ProfileVersion:     "core-v1",
		RankingVersion:     "ranking-v1",
		Candidates:         []LegalQueryCandidate{candidate},
		SelectionMode:      QuerySelectionModeAutomatic,
		CompositionMembers: members,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-026: composition member を拒否しました: %v", err)
	}
	members[0].stepOrigins[0] = QueryCandidateStepOrigin{}
	members[0] = QueryCandidateCompositionMember{}

	got := generation.CompositionMembers()
	if len(got) != 1 ||
		got[0].CandidateID() != candidate.CandidateID() ||
		len(got[0].StepOrigins()) != 2 ||
		got[0].StepOrigins()[1].SourceStartByte() != 24 {
		t.Fatalf("SOT-MODEL-026: composition members = %#v", got)
	}
	got[0].stepOrigins[0] = QueryCandidateStepOrigin{}
	got[0] = QueryCandidateCompositionMember{}
	if generation.CompositionMembers()[0].CandidateID() !=
		candidate.CandidateID() {
		t.Fatal("SOT-MODEL-026: getter から composition member を変更できました")
	}
}

func TestCandidateGenerationは不正なcompositionMember対応を拒否する(
	t *testing.T,
) {
	t.Parallel()

	first := planTestCandidate(
		t,
		"candidate-1-1",
		130,
		nil,
		planTestStep(t, "法令検索", "step-1-1-1"),
		planTestStep(t, "法令本文検索", "step-1-1-2"),
	)
	second := planTestCandidate(
		t,
		"candidate-1-2",
		125,
		nil,
		planTestStep(t, "法令検索", "step-1-2-1"),
	)
	validMember := mustCandidateCompositionMember(
		t,
		first.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(t, "step-1-1-1", 0),
			mustCandidateStepOrigin(t, "step-1-1-2", 12),
		},
	)
	base := CandidateGenerationValues{
		ProfileID:          "core",
		ProfileVersion:     "core-v1",
		RankingVersion:     "ranking-v1",
		Candidates:         []LegalQueryCandidate{first},
		SelectionMode:      QuerySelectionModeAutomatic,
		CompositionMembers: []QueryCandidateCompositionMember{validMember},
	}
	hedge, err := NewCandidateHedgePair(CandidateHedgePairValues{
		FirstCandidateID:  first.CandidateID(),
		SecondCandidateID: second.CandidateID(),
	})
	if err != nil {
		t.Fatalf("試験用 hedge pair を作成できません: %v", err)
	}

	tests := map[string]CandidateGenerationValues{
		"存在しない候補": withCompositionMembers(
			base,
			mustCandidateCompositionMember(
				t,
				"candidate-1-9",
				validMember.StepOrigins(),
			),
		),
		"step の不足": withCompositionMembers(
			base,
			mustCandidateCompositionMember(
				t,
				first.CandidateID(),
				[]QueryCandidateStepOrigin{
					mustCandidateStepOrigin(t, "step-1-1-1", 0),
				},
			),
		),
		"step の逆順": withCompositionMembers(
			base,
			mustCandidateCompositionMember(
				t,
				first.CandidateID(),
				[]QueryCandidateStepOrigin{
					mustCandidateStepOrigin(t, "step-1-1-2", 12),
					mustCandidateStepOrigin(t, "step-1-1-1", 0),
				},
			),
		),
		"clarification required": func() CandidateGenerationValues {
			values := base
			values.SelectionMode = QuerySelectionModeClarificationRequired
			return values
		}(),
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewCandidateGeneration(values); err == nil {
				t.Fatal("SOT-MODEL-026: 不正な composition member 対応を受理しました")
			}
		})
	}

	hedgeValues := base
	hedgeValues.Candidates = []LegalQueryCandidate{first, second}
	hedgeValues.HedgePairs = []CandidateHedgePair{hedge}
	generation, err := NewCandidateGeneration(hedgeValues)
	if err != nil {
		t.Fatalf(
			"SOT-ARCH-027: hedge 候補の member を構造として保持できません: %v",
			err,
		)
	}
	if len(generation.CompositionMembers()) != 1 ||
		len(generation.HedgePairs()) != 1 {
		t.Fatalf(
			"SOT-ARCH-027: hedge/member = %#v/%#v",
			generation.HedgePairs(),
			generation.CompositionMembers(),
		)
	}
}

func TestCandidateGenerationは非Member候補と複数Memberを構造として保持する(
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
	firstMember := mustCandidateCompositionMember(
		t,
		first.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(t, "step-1-1-1", 0),
		},
	)
	secondMember := mustCandidateCompositionMember(
		t,
		second.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(t, "step-1-2-1", 9),
		},
	)
	base := CandidateGenerationValues{
		ProfileID:      "core",
		ProfileVersion: "core-v1",
		RankingVersion: "ranking-v1",
		Candidates:     []LegalQueryCandidate{first, second},
		SelectionMode:  QuerySelectionModeAutomatic,
	}
	tests := map[string][]QueryCandidateCompositionMember{
		"一つの member と非 member 代替候補": {firstMember},
		"複数 member": {firstMember, secondMember},
	}
	for name, members := range tests {
		name, members := name, members
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			values := base
			values.CompositionMembers = members
			generation, err := NewCandidateGeneration(values)
			if err != nil {
				t.Fatalf(
					"SOT-MODEL-026: 構造として有効な composition members = %v",
					err,
				)
			}
			if len(generation.Candidates()) != 2 ||
				len(generation.CompositionMembers()) != len(members) {
				t.Fatalf(
					"SOT-MODEL-026: candidates/members = %d/%d",
					len(generation.Candidates()),
					len(generation.CompositionMembers()),
				)
			}
		})
	}
}

func mustCandidateStepOrigin(
	t *testing.T,
	stepID string,
	sourceStartByte int,
) QueryCandidateStepOrigin {
	t.Helper()
	origin, err := NewQueryCandidateStepOrigin(QueryCandidateStepOriginValues{
		StepID:          stepID,
		SourceStartByte: sourceStartByte,
	})
	if err != nil {
		t.Fatalf("試験用 step origin を作成できません: %v", err)
	}
	return origin
}

func mustCandidateCompositionMember(
	t *testing.T,
	candidateID string,
	origins []QueryCandidateStepOrigin,
) QueryCandidateCompositionMember {
	t.Helper()
	member, err := NewQueryCandidateCompositionMember(
		QueryCandidateCompositionMemberValues{
			CandidateID: candidateID,
			Role:        QueryCandidateCompositionRoleRequiredMember,
			StepOrigins: origins,
		},
	)
	if err != nil {
		t.Fatalf("試験用 composition member を作成できません: %v", err)
	}
	return member
}

func withCompositionMembers(
	values CandidateGenerationValues,
	members ...QueryCandidateCompositionMember,
) CandidateGenerationValues {
	values.CompositionMembers = members
	return values
}
