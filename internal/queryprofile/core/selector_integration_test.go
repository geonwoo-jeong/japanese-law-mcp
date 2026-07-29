package core

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func Test製品前処理からCoreProfileとSelectorまで一貫した計画を返す(
	t *testing.T,
) {
	profile := mustProfile(t)
	selection := profile.Metadata().Selection()
	if got, want := selection.SingleThreshold(), 85; got != want {
		t.Fatalf("core profile の singleThreshold = %d, want %d", got, want)
	}
	if got, want := selection.MinimumExecutionThreshold(), 80; got != want {
		t.Fatalf(
			"core profile の minimumExecutionThreshold = %d, want %d",
			got,
			want,
		)
	}
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("製品前処理器を構築できません: %v", err)
	}
	profileSet, err := legalquery.NewQueryProfileSet(
		[]legalquery.QueryProfile{profile},
	)
	if err != nil {
		t.Fatalf("core profile set を構築できません: %v", err)
	}
	packState, err := legalquery.NewStaticPackState(nil, nil)
	if err != nil {
		t.Fatalf("空の pack state を構築できません: %v", err)
	}

	tests := []struct {
		name                  string
		query                 string
		wantDecision          legalquery.PlanDecision
		wantReasons           []legalquery.ReasonCode
		wantRanked            int
		wantSelected          int
		wantFirstMeaningSteps int
		wantFirstScore        int
		wantFirstStepTasks    []legalquery.Task
		wantSignals           []legalquery.CandidateGenerationSignal
		wantReadSteps         int
		wantCollectionSteps   int
		wantStepBudgets       []stepBudgetExpectation
	}{
		{
			name:                  "弱い一般語は候補を実行せず明確化する",
			query:                 "制度について法情報を探したいです。",
			wantDecision:          legalquery.PlanDecisionNeedsClarification,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeBelowExecutionThreshold, legalquery.ReasonCodeAmbiguousCandidates},
			wantRanked:            2,
			wantSelected:          0,
			wantFirstMeaningSteps: 1,
		},
		{
			name:                  "衝突する暫定法略称は二候補を提示して明確化する",
			query:                 "暫定法の候補となる法令を示してください。",
			wantDecision:          legalquery.PlanDecisionNeedsClarification,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeAmbiguousCandidates},
			wantRanked:            2,
			wantSelected:          2,
			wantFirstMeaningSteps: 1,
		},
		{
			name:                  "明示した独立二候補だけをhedgeする",
			query:                 "実行検証用に保証を法令名と条文の二候補で検索してください。",
			wantDecision:          legalquery.PlanDecisionHedged,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeHedgedCloseCandidates},
			wantRanked:            2,
			wantSelected:          2,
			wantFirstMeaningSteps: 1,
			wantCollectionSteps:   2,
		},
		{
			name:                  "二主題は一候補の独立二stepとして実行する",
			query:                 "永住許可と帰化について教えてください。",
			wantDecision:          legalquery.PlanDecisionSingle,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			wantRanked:            1,
			wantSelected:          1,
			wantFirstMeaningSteps: 2,
			wantFirstScore:        85,
			wantFirstStepTasks: []legalquery.Task{
				legalquery.TaskSearch,
				legalquery.TaskSearch,
			},
			wantCollectionSteps: 2,
		},
		{
			name:                  "条文の実照会はread一件を固定予算へ予約する",
			query:                 "民法第709条を取得してください。",
			wantDecision:          legalquery.PlanDecisionSingle,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			wantRanked:            1,
			wantSelected:          1,
			wantFirstMeaningSteps: 1,
			wantFirstStepTasks:    []legalquery.Task{legalquery.TaskRead},
			wantReadSteps:         1,
			wantStepBudgets: []stepBudgetExpectation{
				{reservedItems: 1},
			},
		},
		{
			name:                  "readとcollectionの実照会はreadを先に予約する",
			query:                 "民法第709条を取得して、損害賠償を含む条文を検索してください。",
			wantDecision:          legalquery.PlanDecisionSingle,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			wantRanked:            1,
			wantSelected:          1,
			wantFirstMeaningSteps: 2,
			wantFirstStepTasks: []legalquery.Task{
				legalquery.TaskRead,
				legalquery.TaskSearch,
			},
			wantReadSteps:       1,
			wantCollectionSteps: 1,
			wantStepBudgets: []stepBudgetExpectation{
				{reservedItems: 1},
				{
					reservedItems:   0,
					effectiveLimit:  legalquery.DefaultLimitPerAttempt,
					hasEffectiveLim: true,
				},
			},
		},
		{
			name:                  "四主題は上限内の一候補四stepとして実行する",
			query:                 "商法、会社法、手形法、小切手法を個別に検索してください。",
			wantDecision:          legalquery.PlanDecisionSingle,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			wantRanked:            1,
			wantSelected:          1,
			wantFirstMeaningSteps: 4,
			wantCollectionSteps:   4,
		},
		{
			name:                  "法令名と一般語の四主題も一候補四stepにする",
			query:                 "民法、商法、量子相続、月面抵当について教えてください。",
			wantDecision:          legalquery.PlanDecisionSingle,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
			wantRanked:            1,
			wantSelected:          1,
			wantFirstMeaningSteps: 4,
			wantCollectionSteps:   4,
		},
		{
			name:                  "五主題は切り捨てず空予算で明確化する",
			query:                 "量子相続、月面抵当、火星登記、宇宙帰化、海底供託について教えてください。",
			wantDecision:          legalquery.PlanDecisionNeedsClarification,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeStepLimitExceeded},
			wantRanked:            0,
			wantSelected:          0,
			wantFirstMeaningSteps: -1,
		},
		{
			name:                  "法令名と一般語の五主題も空予算で明確化する",
			query:                 "民法、商法、量子相続、月面抵当、火星登記について教えてください。",
			wantDecision:          legalquery.PlanDecisionNeedsClarification,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeStepLimitExceeded},
			wantRanked:            0,
			wantSelected:          0,
			wantFirstMeaningSteps: -1,
		},
		{
			name:                  "司法pack要求はcoreだけでは候補や実行を捏造しない",
			query:                 "司法パック無効時に医療過誤の裁判例を検索してください。",
			wantDecision:          legalquery.PlanDecisionNeedsClarification,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeBelowExecutionThreshold},
			wantRanked:            0,
			wantSelected:          0,
			wantFirstMeaningSteps: -1,
			wantSignals: []legalquery.CandidateGenerationSignal{
				legalquery.CandidateSignalReservedPackRequest,
			},
		},
		{
			name:                  "取得候補と翻訳助言の混在は候補を保持して実行しない",
			query:                 "民法第709条を英訳して法的に判断してください。",
			wantDecision:          legalquery.PlanDecisionUnsupported,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeMixedUnsupportedIntent},
			wantRanked:            1,
			wantSelected:          0,
			wantFirstMeaningSteps: 1,
		},
		{
			name:                  "対象外資源だけの照会は候補を作らず実行しない",
			query:                 "都道府県の未公開内部文書を横断検索してください。",
			wantDecision:          legalquery.PlanDecisionUnsupported,
			wantReasons:           []legalquery.ReasonCode{legalquery.ReasonCodeUnsupportedTaskOrResource},
			wantRanked:            0,
			wantSelected:          0,
			wantFirstMeaningSteps: -1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			request, err := legalquery.NewRequest(
				legalquery.RequestValues{Query: test.query},
			)
			if err != nil {
				t.Fatalf("製品 request を構築できません: %v", err)
			}
			preprocessed, err := preprocessor.Preprocess(
				context.Background(),
				request,
			)
			if err != nil {
				t.Fatalf("製品前処理のエラー = %v", err)
			}
			profileSetResult, err := profileSet.Collect(preprocessed)
			if err != nil {
				t.Fatalf("core profile contribution の集約エラー = %v", err)
			}
			plan, err := legalquery.SelectLegalQueryPlan(
				legalquery.SelectorInput{
					ProfileSetResult: profileSetResult,
					PackState:        packState,
					LimitPerAttempt:  request.LimitPerAttempt(),
				},
			)
			if err != nil {
				t.Fatalf("selector のエラー = %v", err)
			}
			if err := plan.Validate(); err != nil {
				t.Fatalf("selector が無効な plan を返しました: %v", err)
			}
			ranked := plan.RankedCandidates()
			if plan.Decision() != test.wantDecision {
				t.Fatalf(
					"decision = %q, want %q; selectionMode=%q reasonCodes=%#v rankedCandidates=%#v",
					plan.Decision(),
					test.wantDecision,
					profileSetResult.SelectionMode(),
					plan.ReasonCodes(),
					ranked,
				)
			}
			if got := plan.ReasonCodes(); !slices.Equal(got, test.wantReasons) {
				t.Fatalf("reasonCodes = %#v, want %#v", got, test.wantReasons)
			}
			if test.wantSignals != nil {
				if got := profileSetResult.Signals(); !slices.Equal(
					got,
					test.wantSignals,
				) {
					t.Fatalf("profile signals = %#v, want %#v", got, test.wantSignals)
				}
			}
			if len(ranked) != test.wantRanked {
				t.Fatalf(
					"rankedCandidates = %#v, want %d 件",
					ranked,
					test.wantRanked,
				)
			}
			if len(plan.Selected()) != test.wantSelected {
				t.Fatalf(
					"selected = %#v, want %d 件",
					plan.Selected(),
					test.wantSelected,
				)
			}
			if test.wantFirstMeaningSteps >= 0 &&
				len(ranked[0].Steps()) != test.wantFirstMeaningSteps {
				t.Fatalf(
					"第一候補の steps = %#v, want %d 件",
					ranked[0].Steps(),
					test.wantFirstMeaningSteps,
				)
			}
			if test.wantFirstScore > 0 &&
				ranked[0].SemanticScore() != test.wantFirstScore {
				t.Fatalf(
					"第一候補の semanticScore = %d, want %d",
					ranked[0].SemanticScore(),
					test.wantFirstScore,
				)
			}
			if test.wantFirstStepTasks != nil {
				steps := ranked[0].Steps()
				gotTasks := make([]legalquery.Task, 0, len(steps))
				for _, step := range steps {
					gotTasks = append(gotTasks, step.Task())
				}
				if !slices.Equal(gotTasks, test.wantFirstStepTasks) {
					t.Fatalf(
						"第一候補の step task = %#v, want %#v",
						gotTasks,
						test.wantFirstStepTasks,
					)
				}
			}
			budget := plan.Budget()
			if budget.ReadStepCount() != test.wantReadSteps ||
				budget.CollectionStepCount() !=
					test.wantCollectionSteps {
				t.Fatalf(
					"budget steps = read:%d collection:%d, want read:%d collection:%d",
					budget.ReadStepCount(),
					budget.CollectionStepCount(),
					test.wantReadSteps,
					test.wantCollectionSteps,
				)
			}
			wantBudgetSteps := test.wantReadSteps +
				test.wantCollectionSteps
			if len(budget.StepBudgets()) != wantBudgetSteps {
				t.Fatalf(
					"stepBudgets = %#v, want %d 件",
					budget.StepBudgets(),
					wantBudgetSteps,
				)
			}
			if test.wantStepBudgets != nil {
				assertStepBudgets(t, budget.StepBudgets(), test.wantStepBudgets)
			}
		})
	}
}

type stepBudgetExpectation struct {
	reservedItems   int
	effectiveLimit  int
	hasEffectiveLim bool
}

func assertStepBudgets(
	t *testing.T,
	got []legalquery.LegalQueryStepBudget,
	want []stepBudgetExpectation,
) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("stepBudgets = %#v, want %#v", got, want)
	}
	for index, expected := range want {
		if got[index].ReservedItems() != expected.reservedItems {
			t.Fatalf(
				"stepBudgets[%d].reservedItems = %d, want %d",
				index,
				got[index].ReservedItems(),
				expected.reservedItems,
			)
		}
		effectiveLimit, exists := got[index].EffectiveLimit()
		if exists != expected.hasEffectiveLim ||
			(exists && effectiveLimit != expected.effectiveLimit) {
			t.Fatalf(
				"stepBudgets[%d].effectiveLimit = (%d, %t), want (%d, %t)",
				index,
				effectiveLimit,
				exists,
				expected.effectiveLimit,
				expected.hasEffectiveLim,
			)
		}
	}
}
