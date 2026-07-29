package legalquery

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

func TestCandidateComposerは異なるProfileの必須意図を原文順へ合成する(
	t *testing.T,
) {
	t.Parallel()

	const query = "裁判例を検索し、民法を検索してください。"
	judicialInput, err := NewJudicialDecisionSearchIntentV1(
		JudicialDecisionSearchIntentV1Values{Query: "裁判例"},
	)
	if err != nil {
		t.Fatalf("裁判例検索 input を作成できません: %v", err)
	}
	lawInput, err := NewLawSearchIntentV1(
		LawSearchIntentV1Values{Query: "民法"},
	)
	if err != nil {
		t.Fatalf("法令検索 input を作成できません: %v", err)
	}
	profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
		{
			metadata: mustSelectorTestMetadata(
				t,
				"core",
				"core-composition-v1",
				selectorTestRankingVersion,
			),
			inputs:        []LogicalInput{lawInput},
			sourceStarts:  []int{len("裁判例を検索し、")},
			evidenceCodes: []EvidenceCode{EvidenceExplicitTask},
			composable:    true,
			requiredPacks: nil,
			selectionMode: QuerySelectionModeAutomatic,
		},
		{
			metadata: mustSelectorTestMetadata(
				t,
				"judicial",
				"judicial-composition-v1",
				selectorTestRankingVersion,
			),
			inputs: []LogicalInput{judicialInput},
			sourceStarts: []int{
				0,
			},
			evidenceCodes: []EvidenceCode{
				EvidenceExplicitTask,
				EvidenceExplicitResource,
			},
			composable:    true,
			requiredPacks: []string{"judicial-cases"},
			selectionMode: QuerySelectionModeAutomatic,
		},
	})

	result, err := profileSet.Collect(mustCompositionTestPreprocessResult(t, query))
	if err != nil {
		t.Fatalf("SOT-ARCH-027: composition のエラー = %v", err)
	}
	candidates := result.RankedCandidates()
	if len(candidates) != 1 {
		t.Fatalf("合成後の候補 = %#v", candidates)
	}
	composed := candidates[0]
	if composed.CandidateID() != "candidate-3-1" {
		t.Fatalf("composer candidateId = %q", composed.CandidateID())
	}
	steps := composed.Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() != InputKindJudicialDecisionSearch ||
		steps[1].InputKind() != InputKindLawSearch {
		t.Fatalf("原文順の合成 steps = %#v", steps)
	}
	if steps[0].StepID() != "step-3-1-1" ||
		steps[1].StepID() != "step-3-1-2" {
		t.Fatalf("composer step IDs = %q, %q", steps[0].StepID(), steps[1].StepID())
	}
	if !slices.Equal(
		composed.EvidenceCodes(),
		[]EvidenceCode{EvidenceExplicitTask, EvidenceExplicitResource},
	) {
		t.Fatalf("evidence union = %#v", composed.EvidenceCodes())
	}
	if !slices.Equal(composed.RequiredPacks(), []string{"judicial-cases"}) {
		t.Fatalf("requiredPacks union = %#v", composed.RequiredPacks())
	}
	score := profileSet.metadata[0].Score()
	wantScore, err := score.Score(composed.EvidenceCodes())
	if err != nil {
		t.Fatalf("期待 score を計算できません: %v", err)
	}
	wantConfidence, err := score.ConfidenceFor(wantScore)
	if err != nil {
		t.Fatalf("期待 confidence を計算できません: %v", err)
	}
	if composed.SemanticScore() != wantScore ||
		composed.Confidence() != wantConfidence {
		t.Fatalf(
			"再計算値 = score:%d confidence:%q",
			composed.SemanticScore(),
			composed.Confidence(),
		)
	}
	if result.CompositionConstraint() != QueryCompositionConstraintNone {
		t.Fatalf("composition constraint = %q", result.CompositionConstraint())
	}
}

func TestCandidateComposerは同一位置をProfile順と元Step順で合成する(
	t *testing.T,
) {
	t.Parallel()

	const query = "法情報をそれぞれ検索してください。"
	first := mustCompositionLawSearchIntent(t, "民法")
	second := mustCompositionLawSearchIntent(t, "商法")
	judicial, err := NewJudicialDecisionSearchIntentV1(
		JudicialDecisionSearchIntentV1Values{Query: "損害賠償"},
	)
	if err != nil {
		t.Fatalf("裁判例検索 input を作成できません: %v", err)
	}
	profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
		compositionTestMemberProfile(
			t,
			"core",
			[]LogicalInput{first, second},
			[]int{0, 0},
			nil,
		),
		compositionTestMemberProfile(
			t,
			"judicial",
			[]LogicalInput{judicial},
			[]int{0},
			[]string{"judicial-cases"},
		),
	})
	result, err := profileSet.Collect(mustCompositionTestPreprocessResult(t, query))
	if err != nil {
		t.Fatalf("SOT-ARCH-027: tie-break composition のエラー = %v", err)
	}
	steps := result.RankedCandidates()[0].Steps()
	got := make([]LogicalInputKind, 0, len(steps))
	for _, step := range steps {
		got = append(got, step.InputKind())
	}
	if !slices.Equal(got, []LogicalInputKind{
		InputKindLawSearch,
		InputKindLawSearch,
		InputKindJudicialDecisionSearch,
	}) {
		t.Fatalf("同一位置の完全順 = %#v", got)
	}
	for index, step := range steps[:2] {
		input, ok := step.LogicalInput().(LawSearchIntentV1)
		if !ok || input.Query() != []string{"民法", "商法"}[index] {
			t.Fatalf("元 step 順[%d] = %#v", index, step.LogicalInput())
		}
	}
}

func TestCandidateComposerは一ProfileだけのMemberを変更しない(t *testing.T) {
	t.Parallel()

	const query = "民法を検索してください。"
	profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
		compositionTestMemberProfile(
			t,
			"core",
			[]LogicalInput{mustCompositionLawSearchIntent(t, "民法")},
			[]int{0},
			nil,
		),
	})
	result, err := profileSet.Collect(mustCompositionTestPreprocessResult(t, query))
	if err != nil {
		t.Fatalf("一 profile の Collect() = %v", err)
	}
	candidates := result.RankedCandidates()
	if len(candidates) != 1 ||
		candidates[0].CandidateID() != "candidate-1-1" {
		t.Fatalf("一 profile の member を変更しました: %#v", candidates)
	}
}

func TestCandidateComposerは明確化Profileを除いて残りを合成する(
	t *testing.T,
) {
	t.Parallel()

	const query = "対象不明の情報と、民法と裁判例を検索してください。"
	profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
		{
			metadata: mustSelectorTestMetadata(
				t,
				"ambiguous",
				"ambiguous-composition-v1",
				selectorTestRankingVersion,
			),
			inputs: []LogicalInput{
				mustCompositionLawSearchIntent(t, "対象不明"),
			},
			evidenceCodes: []EvidenceCode{EvidenceGeneralTerm},
			selectionMode: QuerySelectionModeClarificationRequired,
		},
		compositionTestMemberProfile(
			t,
			"core",
			[]LogicalInput{mustCompositionLawSearchIntent(t, "民法")},
			[]int{len("対象不明の情報と、")},
			nil,
		),
		compositionTestMemberProfile(
			t,
			"judicial",
			[]LogicalInput{
				mustCompositionJudicialSearchIntent(t, "裁判例"),
			},
			[]int{len("対象不明の情報と、民法と")},
			[]string{"judicial-cases"},
		),
	})
	result, err := profileSet.Collect(mustCompositionTestPreprocessResult(t, query))
	if err != nil {
		t.Fatalf("明確化 profile を除く composition = %v", err)
	}
	candidates := result.RankedCandidates()
	if len(candidates) != 2 ||
		!hasCompositionCandidateWithStepCount(candidates, 2) {
		t.Fatalf(
			"SOT-ARCH-027: eligible profile 間を合成していません: %#v",
			candidates,
		)
	}
	if result.SelectionMode() != QuerySelectionModeClarificationRequired ||
		result.CompositionConstraint() != QueryCompositionConstraintNone {
		t.Fatalf(
			"SOT-ARCH-027: selection/constraint = %q/%q",
			result.SelectionMode(),
			result.CompositionConstraint(),
		)
	}
}

func TestCandidateComposerは不正な原文位置では合成せず元候補を保持する(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法と裁判例"
	tests := []struct {
		name  string
		start int
	}{
		{name: "終端", start: len(query)},
		{name: "UTF-8の途中", start: 1},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
				compositionTestMemberProfile(
					t,
					"core",
					[]LogicalInput{mustCompositionLawSearchIntent(t, "民法")},
					[]int{test.start},
					nil,
				),
				compositionTestMemberProfile(
					t,
					"judicial",
					[]LogicalInput{
						mustCompositionJudicialSearchIntent(t, "裁判例"),
					},
					[]int{len("民法と")},
					[]string{"judicial-cases"},
				),
			})
			result, err := profileSet.Collect(
				mustCompositionTestPreprocessResult(t, query),
			)
			if err != nil {
				t.Fatalf(
					"SOT-ARCH-027: 不正位置を合成不適格として扱えません: %v",
					err,
				)
			}
			candidates := result.RankedCandidates()
			if len(candidates) != 2 ||
				candidates[0].CandidateID() != "candidate-1-1" ||
				candidates[1].CandidateID() != "candidate-2-1" {
				t.Fatalf(
					"SOT-ARCH-027: 元候補を保持していません: %#v",
					candidates,
				)
			}
			if result.CompositionConstraint() !=
				QueryCompositionConstraintIneligible {
				t.Fatalf(
					"SOT-ARCH-027: 不正位置の constraint = %q",
					result.CompositionConstraint(),
				)
			}
			plan := mustSelectTestPlan(
				t,
				result,
				mustSelectorTestPackState(
					t,
					[]string{"judicial-cases"},
					[]string{"judicial-cases"},
				),
				DefaultLimitPerAttempt,
			)
			if plan.Decision() != PlanDecisionNeedsClarification ||
				len(plan.Selected()) != 0 ||
				!slices.Equal(
					plan.ReasonCodes(),
					[]ReasonCode{ReasonCodeAmbiguousCandidates},
				) ||
				len(plan.Budget().StepBudgets()) != 0 {
				t.Fatalf(
					"SOT-ARCH-027: 不正位置を部分実行しました: %#v",
					plan,
				)
			}
		})
	}
}

func TestCandidateComposerは不適格Profileを除いて残りを合成し全体を非実行にする(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法と裁判例と商法を検索してください。"
	profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
		compositionTestMemberProfile(
			t,
			"core",
			[]LogicalInput{mustCompositionLawSearchIntent(t, "民法")},
			[]int{0},
			nil,
		),
		compositionTestMemberProfile(
			t,
			"judicial",
			[]LogicalInput{
				mustCompositionJudicialSearchIntent(t, "裁判例"),
			},
			[]int{len("民法と")},
			[]string{"judicial-cases"},
		),
		compositionTestMemberProfile(
			t,
			"invalid-origin",
			[]LogicalInput{mustCompositionLawSearchIntent(t, "商法")},
			[]int{len(query)},
			nil,
		),
	})
	result, err := profileSet.Collect(mustCompositionTestPreprocessResult(t, query))
	if err != nil {
		t.Fatalf("不適格 profile を除く composition = %v", err)
	}
	candidates := result.RankedCandidates()
	if len(candidates) != 2 ||
		!hasCompositionCandidateWithStepCount(candidates, 2) ||
		!hasCompositionCandidateWithStepCount(candidates, 1) {
		t.Fatalf(
			"SOT-ARCH-027: eligible profile 間の合成結果 = %#v",
			candidates,
		)
	}
	if result.CompositionConstraint() !=
		QueryCompositionConstraintIneligible {
		t.Fatalf(
			"SOT-ARCH-027: composition constraint = %q",
			result.CompositionConstraint(),
		)
	}
	plan := mustSelectTestPlan(
		t,
		result,
		mustSelectorTestPackState(
			t,
			[]string{"judicial-cases"},
			[]string{"judicial-cases"},
		),
		DefaultLimitPerAttempt,
	)
	if plan.Decision() != PlanDecisionNeedsClarification ||
		len(plan.Selected()) != 0 ||
		len(plan.Budget().StepBudgets()) != 0 {
		t.Fatalf("SOT-ARCH-027: 不適格 profile の部分実行 plan = %#v", plan)
	}
}

func TestCandidateComposerはHedgeに属するMemberを合成不適格にする(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法と商法または裁判例を検索してください。"
	policy := mustSelectorTestMetadata(
		t,
		"core",
		"core-composition-v1",
		selectorTestRankingVersion,
	).Score()
	coreScope, err := NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("core scope を作成できません: %v", err)
	}
	judicialScope, err := NewCandidateIDScope(2)
	if err != nil {
		t.Fatalf("judicial scope を作成できません: %v", err)
	}
	composerScope, err := NewCandidateIDScope(3)
	if err != nil {
		t.Fatalf("composer scope を作成できません: %v", err)
	}
	first := mustDirectCompositionCandidate(
		t,
		policy,
		coreScope,
		1,
		mustCompositionLawSearchIntent(t, "民法"),
		nil,
	)
	alternative := mustDirectCompositionCandidate(
		t,
		policy,
		coreScope,
		2,
		mustCompositionLawSearchIntent(t, "商法"),
		nil,
	)
	judicial := mustDirectCompositionCandidate(
		t,
		policy,
		judicialScope,
		1,
		mustCompositionJudicialSearchIntent(t, "裁判例"),
		[]string{"judicial-cases"},
	)
	hedge, err := NewCandidateHedgePair(CandidateHedgePairValues{
		FirstCandidateID:  first.CandidateID(),
		SecondCandidateID: alternative.CandidateID(),
	})
	if err != nil {
		t.Fatalf("hedge pair を作成できません: %v", err)
	}
	firstMember := mustCandidateCompositionMember(
		t,
		first.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(t, first.Steps()[0].StepID(), 0),
		},
	)
	judicialMember := mustCandidateCompositionMember(
		t,
		judicial.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(
				t,
				judicial.Steps()[0].StepID(),
				len("民法と商法または"),
			),
		},
	)
	composer, err := NewCandidateComposer(defaultCandidateCompositionVersion)
	if err != nil {
		t.Fatalf("composer を作成できません: %v", err)
	}
	candidates := []LegalQueryCandidate{first, alternative, judicial}
	result, err := composer.compose(
		query,
		[]candidateCompositionContribution{
			{
				profileOrdinal: 1,
				candidates:     candidates[:2],
				members: []QueryCandidateCompositionMember{
					firstMember,
				},
				hedgePairs:    []CandidateHedgePair{hedge},
				selectionMode: QuerySelectionModeAutomatic,
			},
			{
				profileOrdinal: 2,
				candidates:     candidates[2:],
				members: []QueryCandidateCompositionMember{
					judicialMember,
				},
				selectionMode: QuerySelectionModeAutomatic,
			},
		},
		candidates,
		[]CandidateHedgePair{hedge},
		policy,
		composerScope,
	)
	if err != nil {
		t.Fatalf("hedge member の compose() = %v", err)
	}
	if result.constraint != QueryCompositionConstraintIneligible ||
		len(result.candidates) != 3 {
		t.Fatalf(
			"SOT-ARCH-027: hedge member result = constraint:%q candidates:%#v",
			result.constraint,
			result.candidates,
		)
	}
}

func TestCandidateComposerは五Stepを切り捨てず制約にする(t *testing.T) {
	t.Parallel()

	const query = "五つの法情報を検索してください。"
	profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
		compositionTestMemberProfile(
			t,
			"core",
			[]LogicalInput{
				mustCompositionLawSearchIntent(t, "民法"),
				mustCompositionLawSearchIntent(t, "商法"),
				mustCompositionLawSearchIntent(t, "会社法"),
			},
			[]int{0, 3, 6},
			nil,
		),
		compositionTestMemberProfile(
			t,
			"judicial",
			[]LogicalInput{
				mustCompositionJudicialSearchIntent(t, "契約"),
				mustCompositionJudicialSearchIntent(t, "不法行為"),
			},
			[]int{9, 12},
			[]string{"judicial-cases"},
		),
	})
	result, err := profileSet.Collect(mustCompositionTestPreprocessResult(t, query))
	if err != nil {
		t.Fatalf("五 step の合成判定で失敗しました: %v", err)
	}
	if result.CompositionConstraint() !=
		QueryCompositionConstraintStepLimitExceeded {
		t.Fatalf(
			"SOT-ARCH-027: composition constraint = %q",
			result.CompositionConstraint(),
		)
	}
	if len(result.RankedCandidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-027: 構成元候補を部分実行用に残しました: %#v",
			result.RankedCandidates(),
		)
	}
}

func TestCandidateComposerはOriginのないNonMemberProperSubsetを保持する(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法と二つの裁判例を検索してください。"
	law := mustCompositionLawSearchIntent(t, "民法")
	firstJudicial := mustCompositionJudicialSearchIntent(t, "契約")
	secondJudicial := mustCompositionJudicialSearchIntent(t, "不法行為")
	profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
		compositionTestMemberProfile(
			t,
			"core",
			[]LogicalInput{law},
			[]int{0},
			nil,
		),
		compositionTestMemberProfile(
			t,
			"judicial",
			[]LogicalInput{firstJudicial, secondJudicial},
			[]int{9, 18},
			[]string{"judicial-cases"},
		),
		{
			metadata: mustSelectorTestMetadata(
				t,
				"subset",
				"subset-composition-v1",
				selectorTestRankingVersion,
			),
			inputs:        []LogicalInput{law, firstJudicial},
			evidenceCodes: []EvidenceCode{EvidenceExplicitTask},
			requiredPacks: []string{"judicial-cases"},
			selectionMode: QuerySelectionModeAutomatic,
		},
	})
	result, err := profileSet.Collect(mustCompositionTestPreprocessResult(t, query))
	if err != nil {
		t.Fatalf("proper subset composition のエラー = %v", err)
	}
	candidates := result.RankedCandidates()
	if len(candidates) != 2 ||
		len(candidates[0].Steps()) != 3 ||
		candidates[1].CandidateID() != "candidate-3-1" ||
		len(candidates[1].Steps()) != 2 {
		t.Fatalf(
			"SOT-ARCH-027: non-member proper subset を保持していません: %#v",
			candidates,
		)
	}
}

func TestCandidateComposerは一意でないContributionだけを除外して他Profileを合成する(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法、商法、裁判例と会社法を検索してください。"
	policy := mustSelectorTestMetadata(
		t,
		"core",
		"core-composition-v1",
		selectorTestRankingVersion,
	).Score()
	firstScope, err := NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("第一 profile scope を作成できません: %v", err)
	}
	secondScope, err := NewCandidateIDScope(2)
	if err != nil {
		t.Fatalf("第二 profile scope を作成できません: %v", err)
	}
	thirdScope, err := NewCandidateIDScope(3)
	if err != nil {
		t.Fatalf("第三 profile scope を作成できません: %v", err)
	}
	composerScope, err := NewCandidateIDScope(4)
	if err != nil {
		t.Fatalf("composer scope を作成できません: %v", err)
	}
	first := mustDirectCompositionCandidate(
		t,
		policy,
		firstScope,
		1,
		mustCompositionLawSearchIntent(t, "民法"),
		nil,
	)
	alternative := mustDirectCompositionCandidate(
		t,
		policy,
		firstScope,
		2,
		mustCompositionLawSearchIntent(t, "商法"),
		nil,
	)
	judicial := mustDirectCompositionCandidate(
		t,
		policy,
		secondScope,
		1,
		mustCompositionJudicialSearchIntent(t, "裁判例"),
		[]string{"judicial-cases"},
	)
	statute := mustDirectCompositionCandidate(
		t,
		policy,
		thirdScope,
		1,
		mustCompositionLawSearchIntent(t, "会社法"),
		nil,
	)
	firstMember := mustCandidateCompositionMember(
		t,
		first.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(t, first.Steps()[0].StepID(), 0),
		},
	)
	alternativeMember := mustCandidateCompositionMember(
		t,
		alternative.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(
				t,
				alternative.Steps()[0].StepID(),
				len("民法、"),
			),
		},
	)
	judicialMember := mustCandidateCompositionMember(
		t,
		judicial.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(
				t,
				judicial.Steps()[0].StepID(),
				len("民法、商法、"),
			),
		},
	)
	statuteMember := mustCandidateCompositionMember(
		t,
		statute.CandidateID(),
		[]QueryCandidateStepOrigin{
			mustCandidateStepOrigin(
				t,
				statute.Steps()[0].StepID(),
				len("民法、商法、裁判例と"),
			),
		},
	)
	tests := map[string][]QueryCandidateCompositionMember{
		"member一件と非member候補": {firstMember},
		"複数member":           {firstMember, alternativeMember},
	}
	for name, firstMembers := range tests {
		name, firstMembers := name, firstMembers
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			composer, composerErr := NewCandidateComposer(
				defaultCandidateCompositionVersion,
			)
			if composerErr != nil {
				t.Fatalf("composer を作成できません: %v", composerErr)
			}
			candidates := []LegalQueryCandidate{
				first,
				alternative,
				judicial,
				statute,
			}
			result, composeErr := composer.compose(
				query,
				[]candidateCompositionContribution{
					{
						profileOrdinal: 1,
						candidates:     candidates[:2],
						members:        firstMembers,
						selectionMode:  QuerySelectionModeAutomatic,
					},
					{
						profileOrdinal: 2,
						candidates:     candidates[2:3],
						members: []QueryCandidateCompositionMember{
							judicialMember,
						},
						selectionMode: QuerySelectionModeAutomatic,
					},
					{
						profileOrdinal: 3,
						candidates:     candidates[3:],
						members: []QueryCandidateCompositionMember{
							statuteMember,
						},
						selectionMode: QuerySelectionModeAutomatic,
					},
				},
				candidates,
				nil,
				policy,
				composerScope,
			)
			if composeErr != nil {
				t.Fatalf(
					"SOT-ARCH-027: 一意でない contribution の fallback = %v",
					composeErr,
				)
			}
			got := result.candidates
			if len(got) != 3 ||
				got[0].CandidateID() != first.CandidateID() ||
				got[1].CandidateID() != alternative.CandidateID() ||
				got[2].CandidateID() != "candidate-4-1" {
				t.Fatalf(
					"SOT-ARCH-027: 不適格候補の保持または他 profile の合成が不正です: %#v",
					got,
				)
			}
			steps := got[2].Steps()
			if len(steps) != 2 ||
				steps[0].InputKind() !=
					InputKindJudicialDecisionSearch ||
				steps[1].InputKind() != InputKindLawSearch {
				t.Fatalf(
					"SOT-ARCH-027: eligible profile の合成 steps = %#v",
					steps,
				)
			}
			if result.constraint != QueryCompositionConstraintIneligible {
				t.Fatalf(
					"SOT-ARCH-027: 一意でない contribution の constraint = %q",
					result.constraint,
				)
			}
		})
	}
}

func TestCandidateComposerはConceptSource衝突を拒否する(t *testing.T) {
	t.Parallel()

	const query = "二つの法情報を検索してください。"
	firstSource := mustConceptSource(t)
	secondSource, err := NewLegalConceptSource(LegalConceptSourceValues{
		ConceptID:   firstSource.ConceptID(),
		Title:       "異なる公的資料",
		URL:         firstSource.URL(),
		ConfirmedOn: firstSource.ConfirmedOn(),
	})
	if err != nil {
		t.Fatalf("衝突試験用 concept source を作成できません: %v", err)
	}
	profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
		{
			metadata: mustSelectorTestMetadata(
				t,
				"core",
				"core-composition-v1",
				selectorTestRankingVersion,
			),
			inputs: []LogicalInput{
				mustCompositionLawSearchIntent(t, "永住許可"),
			},
			sourceStarts: []int{0},
			evidenceCodes: []EvidenceCode{
				EvidenceExplicitTask,
				EvidenceLegalConcept,
			},
			conceptSources: []LegalConceptSource{firstSource},
			composable:     true,
			selectionMode:  QuerySelectionModeAutomatic,
		},
		{
			metadata: mustSelectorTestMetadata(
				t,
				"judicial",
				"judicial-composition-v1",
				selectorTestRankingVersion,
			),
			inputs: []LogicalInput{
				mustCompositionJudicialSearchIntent(t, "永住許可"),
			},
			sourceStarts: []int{9},
			evidenceCodes: []EvidenceCode{
				EvidenceExplicitTask,
				EvidenceLegalConcept,
			},
			conceptSources: []LegalConceptSource{secondSource},
			requiredPacks:  []string{"judicial-cases"},
			composable:     true,
			selectionMode:  QuerySelectionModeAutomatic,
		},
	})
	if _, err := profileSet.Collect(
		mustCompositionTestPreprocessResult(t, query),
	); err == nil {
		t.Fatal("SOT-ARCH-027: 同じ conceptId の異なる資料を受理しました")
	}
}

func TestCandidateComposerはPack無効でも意味を変えず全体を非実行にする(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法と裁判例を検索してください。"
	profileSet := mustCompositionTestProfileSet(t, []compositionTestProfile{
		compositionTestMemberProfile(
			t,
			"core",
			[]LogicalInput{mustCompositionLawSearchIntent(t, "民法")},
			[]int{0},
			nil,
		),
		compositionTestMemberProfile(
			t,
			"judicial",
			[]LogicalInput{
				mustCompositionJudicialSearchIntent(t, "損害賠償"),
			},
			[]int{9},
			[]string{"judicial-cases"},
		),
	})
	result, err := profileSet.Collect(mustCompositionTestPreprocessResult(t, query))
	if err != nil {
		t.Fatalf("pack 無効試験の composition = %v", err)
	}
	packState := mustSelectorTestPackState(
		t,
		[]string{"judicial-cases"},
		nil,
	)
	plan := mustSelectTestPlan(t, result, packState, DefaultLimitPerAttempt)
	if plan.Decision() != PlanDecisionCapabilityUnavailable ||
		!slices.Equal(
			plan.ReasonCodes(),
			[]ReasonCode{ReasonCodeRequiredPackDisabled},
		) ||
		len(plan.Selected()) != 1 ||
		len(plan.RankedCandidates()) != 1 ||
		len(plan.RankedCandidates()[0].Steps()) != 2 {
		t.Fatalf(
			"SOT-ARCH-027: pack 無効 plan = decision:%q reasons:%#v selected:%#v ranked:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
			plan.Selected(),
			plan.RankedCandidates(),
		)
	}
}

type compositionTestProfile struct {
	metadata       QueryProfileMetadata
	inputs         []LogicalInput
	sourceStarts   []int
	evidenceCodes  []EvidenceCode
	conceptSources []LegalConceptSource
	requiredPacks  []string
	composable     bool
	selectionMode  QuerySelectionMode
}

func (p compositionTestProfile) Metadata() QueryProfileMetadata {
	return p.metadata
}

func (compositionTestProfile) CueVocabulary() []CueVocabularyEntry {
	return nil
}

func (p compositionTestProfile) Generate(
	_ CandidateGenerationInput,
	scope CandidateIDScope,
) (CandidateGeneration, error) {
	score, err := p.metadata.Score().Score(p.evidenceCodes)
	if err != nil {
		return CandidateGeneration{}, err
	}
	confidence, err := p.metadata.Score().ConfidenceFor(score)
	if err != nil {
		return CandidateGeneration{}, err
	}
	candidate, err := AssembleLegalQueryCandidate(CandidateAssemblyValues{
		IDScope:          scope,
		CandidateOrdinal: 1,
		SemanticScore:    score,
		Confidence:       confidence,
		EvidenceCodes:    p.evidenceCodes,
		ConceptSources:   p.conceptSources,
		RequiredPacks:    p.requiredPacks,
		LogicalInputs:    p.inputs,
	})
	if err != nil {
		return CandidateGeneration{}, err
	}
	values := CandidateGenerationValues{
		ProfileID:          p.metadata.ProfileID(),
		ProfileVersion:     p.metadata.ProfileVersion(),
		RankingVersion:     p.metadata.RankingVersion(),
		Candidates:         []LegalQueryCandidate{candidate},
		SelectionMode:      p.selectionMode,
		CompositionMembers: nil,
	}
	if p.composable {
		origins := make([]QueryCandidateStepOrigin, 0, len(p.sourceStarts))
		for index, start := range p.sourceStarts {
			origin, originErr := NewQueryCandidateStepOrigin(
				QueryCandidateStepOriginValues{
					StepID:          candidate.Steps()[index].StepID(),
					SourceStartByte: start,
				},
			)
			if originErr != nil {
				return CandidateGeneration{}, originErr
			}
			origins = append(origins, origin)
		}
		member, memberErr := NewQueryCandidateCompositionMember(
			QueryCandidateCompositionMemberValues{
				CandidateID: candidate.CandidateID(),
				Role:        QueryCandidateCompositionRoleRequiredMember,
				StepOrigins: origins,
			},
		)
		if memberErr != nil {
			return CandidateGeneration{}, memberErr
		}
		values.CompositionMembers = []QueryCandidateCompositionMember{member}
	}
	return NewCandidateGeneration(values)
}

func compositionTestMemberProfile(
	t *testing.T,
	profileID string,
	inputs []LogicalInput,
	sourceStarts []int,
	requiredPacks []string,
) compositionTestProfile {
	t.Helper()
	return compositionTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			profileID,
			profileID+"-composition-v1",
			selectorTestRankingVersion,
		),
		inputs:       inputs,
		sourceStarts: sourceStarts,
		evidenceCodes: []EvidenceCode{
			EvidenceExplicitTask,
			EvidenceExplicitResource,
		},
		requiredPacks: requiredPacks,
		composable:    true,
		selectionMode: QuerySelectionModeAutomatic,
	}
}

func hasCompositionCandidateWithStepCount(
	candidates []LegalQueryCandidate,
	count int,
) bool {
	for _, candidate := range candidates {
		if len(candidate.Steps()) == count {
			return true
		}
	}
	return false
}

func mustCompositionTestProfileSet(
	t *testing.T,
	profiles []compositionTestProfile,
) QueryProfileSet {
	t.Helper()
	values := make([]QueryProfile, 0, len(profiles))
	for index := range profiles {
		values = append(values, profiles[index])
	}
	profileSet, err := NewQueryProfileSet(values)
	if err != nil {
		t.Fatalf("composition 試験用 profile set を作成できません: %v", err)
	}
	return profileSet
}

func mustCompositionTestPreprocessResult(
	t *testing.T,
	query string,
) PreprocessResult {
	t.Helper()
	result, err := NewPreprocessResult(PreprocessResultValues{
		Query:         query,
		ComparisonKey: querynormalization.ComparisonKey(query),
	})
	if err != nil {
		t.Fatalf("composition 試験用前処理結果を作成できません: %v", err)
	}
	return result
}

func mustCompositionLawSearchIntent(
	t *testing.T,
	query string,
) LawSearchIntentV1 {
	t.Helper()
	input, err := NewLawSearchIntentV1(LawSearchIntentV1Values{Query: query})
	if err != nil {
		t.Fatalf("法令検索 input を作成できません: %v", err)
	}
	return input
}

func mustCompositionJudicialSearchIntent(
	t *testing.T,
	query string,
) JudicialDecisionSearchIntentV1 {
	t.Helper()
	input, err := NewJudicialDecisionSearchIntentV1(
		JudicialDecisionSearchIntentV1Values{Query: query},
	)
	if err != nil {
		t.Fatalf("裁判例検索 input を作成できません: %v", err)
	}
	return input
}

func mustDirectCompositionCandidate(
	t *testing.T,
	policy QueryScorePolicy,
	scope CandidateIDScope,
	ordinal int,
	input LogicalInput,
	requiredPacks []string,
) LegalQueryCandidate {
	t.Helper()
	evidence := []EvidenceCode{
		EvidenceExplicitTask,
		EvidenceExplicitResource,
	}
	score, err := policy.Score(evidence)
	if err != nil {
		t.Fatalf("composition 試験用 score を計算できません: %v", err)
	}
	confidence, err := policy.ConfidenceFor(score)
	if err != nil {
		t.Fatalf("composition 試験用 confidence を計算できません: %v", err)
	}
	candidate, err := AssembleLegalQueryCandidate(CandidateAssemblyValues{
		IDScope:          scope,
		CandidateOrdinal: ordinal,
		SemanticScore:    score,
		Confidence:       confidence,
		EvidenceCodes:    evidence,
		RequiredPacks:    requiredPacks,
		LogicalInputs:    []LogicalInput{input},
	})
	if err != nil {
		t.Fatalf("composition 試験用 candidate を作成できません: %v", err)
	}
	return candidate
}
