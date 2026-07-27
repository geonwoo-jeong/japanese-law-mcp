package legalquery

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestLegalQueryPlanAcceptsFiveDecisionShapes(t *testing.T) {
	t.Parallel()

	core := planTestCandidate(t, "candidate-core", 900, nil,
		planTestStep(t, "法令検索", "step-core"))
	secondary := planTestCandidate(t, "candidate-secondary", 850, nil,
		planTestStep(t, "法令本文検索", "step-secondary"))
	pack := planTestCandidate(t, "candidate-pack", 800, []string{"judicial-cases"},
		planTestStep(t, "裁判例検索", "step-pack"))

	tests := map[string]LegalQueryPlanValues{
		"single": {
			ProfileVersion:   "profile-v1",
			Decision:         PlanDecisionSingle,
			RankedCandidates: []LegalQueryCandidate{core, secondary},
			Selected: []LegalQueryPlanSelection{
				planTestSelection(t, core, SelectionAvailabilityAvailable),
			},
			ReasonCodes:     []ReasonCode{ReasonCodeSingleClearCandidate},
			LimitPerAttempt: 10,
		},
		"hedged": {
			ProfileVersion:   "profile-v1",
			Decision:         PlanDecisionHedged,
			RankedCandidates: []LegalQueryCandidate{core, secondary},
			Selected: []LegalQueryPlanSelection{
				planTestSelection(t, core, SelectionAvailabilityAvailable),
				planTestSelection(t, secondary, SelectionAvailabilityAvailable),
			},
			ReasonCodes:     []ReasonCode{ReasonCodeHedgedCloseCandidates},
			LimitPerAttempt: 10,
		},
		"needs_clarification": {
			ProfileVersion:   "profile-v1",
			Decision:         PlanDecisionNeedsClarification,
			RankedCandidates: []LegalQueryCandidate{core, secondary},
			Selected: []LegalQueryPlanSelection{
				planTestSelection(t, core, SelectionAvailabilityAvailable),
				planTestSelection(t, secondary, SelectionAvailabilityAvailable),
			},
			ReasonCodes: []ReasonCode{
				ReasonCodeBelowExecutionThreshold,
				ReasonCodeAmbiguousCandidates,
			},
			LimitPerAttempt: 10,
		},
		"capability_unavailable": {
			ProfileVersion:   "profile-v1",
			Decision:         PlanDecisionCapabilityUnavailable,
			RankedCandidates: []LegalQueryCandidate{pack},
			Selected: []LegalQueryPlanSelection{
				planTestSelection(t, pack, SelectionAvailabilityPackDisabled),
			},
			ReasonCodes:     []ReasonCode{ReasonCodeRequiredPackDisabled},
			LimitPerAttempt: 10,
		},
		"unsupported": {
			ProfileVersion:   "profile-v1",
			Decision:         PlanDecisionUnsupported,
			RankedCandidates: []LegalQueryCandidate{core},
			Selected:         []LegalQueryPlanSelection{},
			ReasonCodes: []ReasonCode{
				ReasonCodeNonJapaneseQuery,
				ReasonCodeMixedUnsupportedIntent,
				ReasonCodeUnsupportedTaskOrResource,
			},
			LimitPerAttempt: 10,
		},
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			plan, err := NewLegalQueryPlan(values)
			if err != nil {
				t.Fatalf("SOT-MODEL-023: plan を作成できません: %v", err)
			}
			if plan.Language() != "ja" ||
				plan.ProfileVersion() != values.ProfileVersion ||
				plan.Decision() != values.Decision ||
				len(plan.RankedCandidates()) != len(values.RankedCandidates) ||
				len(plan.Selected()) != len(values.Selected) {
				t.Fatalf("SOT-MODEL-023: plan = %#v", plan)
			}
			if err := plan.Validate(); err != nil {
				t.Fatalf("SOT-MODEL-023: plan が無効です: %v", err)
			}
		})
	}
}

func TestLegalQueryPlanRejectsDecisionShapeMismatch(t *testing.T) {
	t.Parallel()

	core := planTestCandidate(t, "candidate-core", 900, nil,
		planTestStep(t, "法令検索", "step-core"))
	other := planTestCandidate(t, "candidate-other", 800, nil,
		planTestStep(t, "法令本文検索", "step-other"))
	pack := planTestCandidate(t, "candidate-pack", 700, []string{"judicial-cases"},
		planTestStep(t, "裁判例検索", "step-pack"))
	available := planTestSelection(t, core, SelectionAvailabilityAvailable)
	otherAvailable := planTestSelection(t, other, SelectionAvailabilityAvailable)
	disabled := planTestSelection(t, pack, SelectionAvailabilityPackDisabled)

	tests := map[string]LegalQueryPlanValues{
		"未知の decision": planTestValues(
			PlanDecision("unknown"), []LegalQueryCandidate{core}, []LegalQueryPlanSelection{available},
			[]ReasonCode{ReasonCodeSingleClearCandidate},
		),
		"single の選択なし": planTestValues(
			PlanDecisionSingle, []LegalQueryCandidate{core}, nil,
			[]ReasonCode{ReasonCodeSingleClearCandidate},
		),
		"single の二候補": planTestValues(
			PlanDecisionSingle, []LegalQueryCandidate{core, other},
			[]LegalQueryPlanSelection{available, otherAvailable},
			[]ReasonCode{ReasonCodeSingleClearCandidate},
		),
		"single の pack 無効": planTestValues(
			PlanDecisionSingle, []LegalQueryCandidate{pack},
			[]LegalQueryPlanSelection{disabled},
			[]ReasonCode{ReasonCodeSingleClearCandidate},
		),
		"hedged の一候補": planTestValues(
			PlanDecisionHedged, []LegalQueryCandidate{core},
			[]LegalQueryPlanSelection{available},
			[]ReasonCode{ReasonCodeHedgedCloseCandidates},
		),
		"hedged の pack 無効": planTestValues(
			PlanDecisionHedged, []LegalQueryCandidate{core, pack},
			[]LegalQueryPlanSelection{available, disabled},
			[]ReasonCode{ReasonCodeHedgedCloseCandidates},
		),
		"明確化の pack 無効": planTestValues(
			PlanDecisionNeedsClarification, []LegalQueryCandidate{pack},
			[]LegalQueryPlanSelection{disabled},
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		),
		"利用不可の選択なし": planTestValues(
			PlanDecisionCapabilityUnavailable, []LegalQueryCandidate{pack}, nil,
			[]ReasonCode{ReasonCodeRequiredPackDisabled},
		),
		"利用不可の available": planTestValues(
			PlanDecisionCapabilityUnavailable, []LegalQueryCandidate{core},
			[]LegalQueryPlanSelection{available},
			[]ReasonCode{ReasonCodeRequiredPackDisabled},
		),
		"対象外の選択あり": planTestValues(
			PlanDecisionUnsupported, []LegalQueryCandidate{core},
			[]LegalQueryPlanSelection{available},
			[]ReasonCode{ReasonCodeUnsupportedTaskOrResource},
		),
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewLegalQueryPlan(values); err == nil {
				t.Fatal("SOT-MODEL-023: decision と矛盾する plan を受理しました")
			}
		})
	}
}

func TestLegalQueryPlanEnforcesReasonCodes(t *testing.T) {
	t.Parallel()

	base := planTestSingleValues(t)
	tests := map[string]LegalQueryPlanValues{
		"欠落": func() LegalQueryPlanValues {
			values := base
			values.ReasonCodes = []ReasonCode{}
			return values
		}(),
		"未知": func() LegalQueryPlanValues {
			values := base
			values.ReasonCodes = []ReasonCode{ReasonCode("unknown")}
			return values
		}(),
		"decision と不一致": func() LegalQueryPlanValues {
			values := base
			values.ReasonCodes = []ReasonCode{ReasonCodeHedgedCloseCandidates}
			return values
		}(),
		"重複": planTestValues(
			PlanDecisionUnsupported,
			base.RankedCandidates,
			nil,
			[]ReasonCode{
				ReasonCodeNonJapaneseQuery,
				ReasonCodeNonJapaneseQuery,
			},
		),
		"順序違反": planTestValues(
			PlanDecisionUnsupported,
			base.RankedCandidates,
			nil,
			[]ReasonCode{
				ReasonCodeUnsupportedTaskOrResource,
				ReasonCodeNonJapaneseQuery,
			},
		),
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewLegalQueryPlan(values); err == nil {
				t.Fatal("SOT-MODEL-023: 不正な reasonCodes を受理しました")
			}
		})
	}
}

func TestLegalQueryPlanSelectionMatchesCandidateAndAvailability(t *testing.T) {
	t.Parallel()

	pack := planTestCandidate(t, "candidate-pack", 900,
		[]string{"judicial-cases", "tax"},
		planTestStep(t, "裁判例検索", "step-pack"))
	valid := LegalQueryPlanSelectionValues{
		CandidateID:   pack.CandidateID(),
		Availability:  SelectionAvailabilityPackDisabled,
		RequiredPacks: pack.RequiredPacks(),
	}
	selection, err := NewLegalQueryPlanSelection(valid)
	if err != nil {
		t.Fatalf("SOT-MODEL-023: selection を作成できません: %v", err)
	}
	requiredPacks := selection.RequiredPacks()
	requiredPacks[0] = "changed"
	if selection.CandidateID() != pack.CandidateID() ||
		selection.Availability() != SelectionAvailabilityPackDisabled ||
		selection.RequiredPacks()[0] != "judicial-cases" {
		t.Fatal("SOT-MODEL-023: selection が不変ではありません")
	}

	invalidSelections := []LegalQueryPlanSelectionValues{
		{
			CandidateID:   pack.CandidateID(),
			Availability:  SelectionAvailability("unknown"),
			RequiredPacks: pack.RequiredPacks(),
		},
		{
			CandidateID:  "candidate-core",
			Availability: SelectionAvailabilityPackDisabled,
		},
	}
	for _, values := range invalidSelections {
		if _, err := NewLegalQueryPlanSelection(values); err == nil {
			t.Fatal("SOT-MODEL-023: 不正な availability を受理しました")
		}
	}

	mismatch := planTestValues(
		PlanDecisionCapabilityUnavailable,
		[]LegalQueryCandidate{pack},
		[]LegalQueryPlanSelection{planTestSelectionWithPacks(
			t,
			pack.CandidateID(),
			SelectionAvailabilityPackDisabled,
			[]string{"judicial-cases"},
		)},
		[]ReasonCode{ReasonCodeRequiredPackDisabled},
	)
	if _, err := NewLegalQueryPlan(mismatch); err == nil {
		t.Fatal("SOT-MODEL-023: candidate と異なる requiredPacks を受理しました")
	}
}

func TestLegalQueryPlanEnforcesGlobalRankingAndSelectionInvariants(t *testing.T) {
	t.Parallel()

	first := planTestCandidate(t, "candidate-first", 900, nil,
		planTestStep(t, "法令検索", "step-first"))
	tied := planTestCandidate(t, "candidate-tied", 900, nil,
		planTestStep(t, "法令本文検索", "step-tied"))
	last := planTestCandidate(t, "candidate-last", 700, nil,
		planTestStep(t, "更新一覧", "step-last"))
	values := planTestValues(
		PlanDecisionNeedsClarification,
		[]LegalQueryCandidate{first, tied, last},
		[]LegalQueryPlanSelection{
			planTestSelection(t, first, SelectionAvailabilityAvailable),
			planTestSelection(t, last, SelectionAvailabilityAvailable),
		},
		[]ReasonCode{ReasonCodeAmbiguousCandidates},
	)
	plan, err := NewLegalQueryPlan(values)
	if err != nil {
		t.Fatalf("SOT-MODEL-023: 同点を含む plan を拒否しました: %v", err)
	}
	got := plan.RankedCandidates()
	if got[0].CandidateID() != "candidate-first" ||
		got[1].CandidateID() != "candidate-tied" ||
		got[2].CandidateID() != "candidate-last" {
		t.Fatal("SOT-MODEL-023: 同点候補の入力順を変更しました")
	}

	tests := map[string]LegalQueryPlanValues{
		"score 増加": planTestValues(
			PlanDecisionNeedsClarification,
			[]LegalQueryCandidate{last, first}, nil,
			[]ReasonCode{ReasonCodeBelowExecutionThreshold},
		),
		"candidateId 重複": planTestValues(
			PlanDecisionNeedsClarification,
			[]LegalQueryCandidate{first, first}, nil,
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		),
		"stepId の全体重複": planTestValues(
			PlanDecisionNeedsClarification,
			[]LegalQueryCandidate{
				first,
				planTestCandidate(t, "candidate-duplicate-step", 800, nil,
					planTestStep(t, "法令本文検索", "step-first")),
			},
			nil,
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		),
		"selected の順位逆転": planTestValues(
			PlanDecisionNeedsClarification,
			[]LegalQueryCandidate{first, tied},
			[]LegalQueryPlanSelection{
				planTestSelection(t, tied, SelectionAvailabilityAvailable),
				planTestSelection(t, first, SelectionAvailabilityAvailable),
			},
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		),
		"selected の重複": planTestValues(
			PlanDecisionNeedsClarification,
			[]LegalQueryCandidate{first},
			[]LegalQueryPlanSelection{
				planTestSelection(t, first, SelectionAvailabilityAvailable),
				planTestSelection(t, first, SelectionAvailabilityAvailable),
			},
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		),
		"ranked にない selected": planTestValues(
			PlanDecisionNeedsClarification,
			[]LegalQueryCandidate{first},
			[]LegalQueryPlanSelection{
				planTestSelection(t, last, SelectionAvailabilityAvailable),
			},
			[]ReasonCode{ReasonCodeAmbiguousCandidates},
		),
	}
	for name, invalid := range tests {
		name, invalid := name, invalid
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewLegalQueryPlan(invalid); err == nil {
				t.Fatal("SOT-MODEL-023: plan 全体の不変条件違反を受理しました")
			}
		})
	}
}

func TestLegalQueryPlanEnforcesCandidateAndSelectedStepLimits(t *testing.T) {
	t.Parallel()

	ranked := make([]LegalQueryCandidate, 0, MaxRankedCandidates)
	for index := 0; index < MaxRankedCandidates; index++ {
		ranked = append(ranked, planTestCandidate(
			t,
			"candidate-"+planTestLetter(index),
			1000-index,
			nil,
			planTestStep(t, "法令検索", "step-"+planTestLetter(index)),
		))
	}
	values := planTestValues(
		PlanDecisionNeedsClarification,
		ranked,
		nil,
		[]ReasonCode{ReasonCodeBelowExecutionThreshold},
	)
	if _, err := NewLegalQueryPlan(values); err != nil {
		t.Fatalf("SOT-MODEL-023: 十六候補を拒否しました: %v", err)
	}
	values.RankedCandidates = append(
		values.RankedCandidates,
		planTestCandidate(t, "candidate-over", 1, nil,
			planTestStep(t, "法令検索", "step-over")),
	)
	if _, err := NewLegalQueryPlan(values); err == nil {
		t.Fatal("SOT-MODEL-023: 十七候補を受理しました")
	}

	first := planTestCandidate(t, "candidate-three", 900, nil,
		planTestStep(t, "法令検索", "step-a"),
		planTestStep(t, "法令本文検索", "step-b"),
		planTestStep(t, "法令読取り", "step-c"))
	second := planTestCandidate(t, "candidate-two", 800, nil,
		planTestStep(t, "更新一覧", "step-d"),
		planTestStep(t, "条文読取り", "step-e"))
	tooManySteps := planTestValues(
		PlanDecisionHedged,
		[]LegalQueryCandidate{first, second},
		[]LegalQueryPlanSelection{
			planTestSelection(t, first, SelectionAvailabilityAvailable),
			planTestSelection(t, second, SelectionAvailabilityAvailable),
		},
		[]ReasonCode{ReasonCodeHedgedCloseCandidates},
	)
	if _, err := NewLegalQueryPlan(tooManySteps); err == nil {
		t.Fatal("SOT-MODEL-023: 選択した五 step を受理しました")
	}
}

func TestLegalQueryPlanValidatesProfileVersionAndFixedLanguage(t *testing.T) {
	t.Parallel()

	for _, profileVersion := range []string{"v", strings.Repeat("a", 128)} {
		values := planTestSingleValues(t)
		values.ProfileVersion = profileVersion
		plan, err := NewLegalQueryPlan(values)
		if err != nil {
			t.Fatalf("SOT-MODEL-023: 境界 profileVersion を拒否しました: %v", err)
		}
		if plan.Language() != "ja" {
			t.Fatalf("SOT-MODEL-023: language = %q", plan.Language())
		}
	}

	invalid := []string{
		"",
		strings.Repeat("a", 129),
		" profile",
		"profile\u3000",
		"profile\nv1",
		string([]byte{'v', 0xff}),
	}
	for _, profileVersion := range invalid {
		values := planTestSingleValues(t)
		values.ProfileVersion = profileVersion
		if _, err := NewLegalQueryPlan(values); err == nil {
			t.Fatalf("SOT-MODEL-023: 不正な profileVersion %q を受理しました", profileVersion)
		}
	}
}

func TestLegalQueryPlanIsImmutableAndRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	candidate := planTestCandidate(t, "candidate-pack", 900, []string{"judicial-cases"},
		planTestStep(t, "裁判例検索", "step-pack"))
	ranked := []LegalQueryCandidate{candidate}
	selected := []LegalQueryPlanSelection{
		planTestSelection(t, candidate, SelectionAvailabilityAvailable),
	}
	reasons := []ReasonCode{ReasonCodeSingleClearCandidate}
	plan, err := NewLegalQueryPlan(LegalQueryPlanValues{
		ProfileVersion:   "profile-v1",
		Decision:         PlanDecisionSingle,
		RankedCandidates: ranked,
		Selected:         selected,
		ReasonCodes:      reasons,
		LimitPerAttempt:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	ranked[0].candidateID = "changed"
	ranked[0].requiredPacks[0] = "changed"
	ranked[0].steps[0].stepID = "changed"
	selected[0].requiredPacks[0] = "changed"
	reasons[0] = ReasonCodeAmbiguousCandidates
	gotRanked := plan.RankedCandidates()
	gotSelected := plan.Selected()
	gotReasons := plan.ReasonCodes()
	gotRanked[0].candidateID = "changed-again"
	gotRanked[0].requiredPacks[0] = "changed-again"
	gotRanked[0].steps[0].stepID = "changed-again"
	gotSelected[0].requiredPacks[0] = "changed-again"
	gotReasons[0] = ReasonCodeAmbiguousCandidates
	if plan.RankedCandidates()[0].CandidateID() != "candidate-pack" ||
		plan.RankedCandidates()[0].RequiredPacks()[0] != "judicial-cases" ||
		plan.RankedCandidates()[0].Steps()[0].StepID() != "step-pack" ||
		plan.Selected()[0].RequiredPacks()[0] != "judicial-cases" ||
		plan.ReasonCodes()[0] != ReasonCodeSingleClearCandidate {
		t.Fatal("SOT-MODEL-023: plan を入力または getter から変更できました")
	}

	for _, target := range []any{
		&LegalQueryPlanSelection{},
		&LegalQueryPlan{},
	} {
		if err := json.Unmarshal([]byte(`{}`), target); err == nil {
			t.Fatalf("SOT-MODEL-023: %T を JSON から直接復元できました", target)
		}
	}
}

func planTestSingleValues(t *testing.T) LegalQueryPlanValues {
	t.Helper()
	candidate := planTestCandidate(t, "candidate-single", 900, nil,
		planTestStep(t, "法令検索", "step-single"))
	return planTestValues(
		PlanDecisionSingle,
		[]LegalQueryCandidate{candidate},
		[]LegalQueryPlanSelection{
			planTestSelection(t, candidate, SelectionAvailabilityAvailable),
		},
		[]ReasonCode{ReasonCodeSingleClearCandidate},
	)
}

func planTestValues(
	decision PlanDecision,
	ranked []LegalQueryCandidate,
	selected []LegalQueryPlanSelection,
	reasons []ReasonCode,
) LegalQueryPlanValues {
	if ranked == nil {
		ranked = []LegalQueryCandidate{}
	}
	if selected == nil {
		selected = []LegalQueryPlanSelection{}
	}
	return LegalQueryPlanValues{
		ProfileVersion:   "profile-v1",
		Decision:         decision,
		RankedCandidates: ranked,
		Selected:         selected,
		ReasonCodes:      reasons,
		LimitPerAttempt:  20,
	}
}

func planTestCandidate(
	t *testing.T,
	id string,
	score int,
	requiredPacks []string,
	steps ...LegalQueryCandidateStep,
) LegalQueryCandidate {
	t.Helper()
	candidate, err := NewLegalQueryCandidate(LegalQueryCandidateValues{
		CandidateID:    id,
		SemanticScore:  score,
		Confidence:     ConfidenceHigh,
		EvidenceCodes:  []EvidenceCode{EvidenceExplicitTask},
		ConceptSources: []LegalConceptSource{},
		RequiredPacks:  requiredPacks,
		Steps:          steps,
	})
	if err != nil {
		t.Fatalf("試験用 candidate を作成できません: %v", err)
	}
	return candidate
}

func planTestStep(
	t *testing.T,
	fixtureName string,
	stepID string,
) LegalQueryCandidateStep {
	t.Helper()
	values := validStepFixtures(t)[fixtureName].values
	values.StepID = stepID
	step, err := NewLegalQueryCandidateStep(values)
	if err != nil {
		t.Fatalf("試験用 step を作成できません: %v", err)
	}
	return step
}

func planTestSelection(
	t *testing.T,
	candidate LegalQueryCandidate,
	availability SelectionAvailability,
) LegalQueryPlanSelection {
	t.Helper()
	return planTestSelectionWithPacks(
		t,
		candidate.CandidateID(),
		availability,
		candidate.RequiredPacks(),
	)
}

func planTestSelectionWithPacks(
	t *testing.T,
	candidateID string,
	availability SelectionAvailability,
	requiredPacks []string,
) LegalQueryPlanSelection {
	t.Helper()
	selection, err := NewLegalQueryPlanSelection(LegalQueryPlanSelectionValues{
		CandidateID:   candidateID,
		Availability:  availability,
		RequiredPacks: requiredPacks,
	})
	if err != nil {
		t.Fatalf("試験用 selection を作成できません: %v", err)
	}
	return selection
}

func planTestLetter(index int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	return letters[index : index+1]
}
