package judicialcases

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
	coreprofile "github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/core"
)

const explicitMixedSearchQuery = "裁判例を「工場騒音」で検索し、民法を検索してください。"

func Test空のCore候補は明示裁判例Resourceの選択を妨げない(t *testing.T) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(
		t,
		"ネット中傷の裁判例を検索してください。",
	)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionSingle {
		t.Fatalf(
			"SOT-ENG-023: explicit judicial resource plan = decision:%q reasons:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
		)
	}
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	steps := candidate.Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("SOT-ENG-023: judicial candidate steps = %#v", steps)
	}
	input, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || input.Query() != "名誉毀損" {
		t.Fatalf("SOT-ENG-023: judicial query = %#v", steps[0].LogicalInput())
	}
}

func Test裁判所の裁判例指定は法令誤記補正を混在させない(t *testing.T) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	const query = "裁判所の裁判例から「解雇権濫用」を探す"
	request := mustMixedProfileRequest(t, query)
	preprocessed := runtime.preprocess(t, request)
	plan := selectMixedProfilePlan(
		t,
		runtime.collectPreprocessed(t, preprocessed),
		request,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionSingle {
		t.Fatalf(
			"SOT-ARCH-021/SOT-ARCH-027: plan = decision:%q reasons:%#v ranked:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
			plan.RankedCandidates(),
		)
	}
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	steps := candidate.Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf(
			"SOT-ARCH-021/SOT-ARCH-027: steps = %#v laws=%#v cues=%#v terms=%#v",
			steps,
			preprocessed.LawNameMentions(),
			preprocessed.CueMentions(),
			preprocessed.QueryTermMentions(),
		)
	}
	input, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || input.Query() != "解雇権濫用" {
		t.Fatalf("SOT-ARCH-021/SOT-ARCH-027: judicial query = %#v", steps[0].LogicalInput())
	}
}

func Test包括的な法情報語は法令条文と裁判例の明確化を要求する(
	t *testing.T,
) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(
		t,
		"永住権について法情報を調べてください。",
	)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	ranked := plan.RankedCandidates()
	selected := plan.Selected()
	if plan.Decision() != legalquery.PlanDecisionNeedsClarification ||
		!slices.Equal(
			plan.ReasonCodes(),
			[]legalquery.ReasonCode{
				legalquery.ReasonCodeAmbiguousCandidates,
			},
		) ||
		len(ranked) != 2 ||
		len(selected) != 2 {
		t.Fatalf(
			"SOT-ENG-023: plan = decision:%q reasons:%#v ranked:%#v selected:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
			ranked,
			selected,
		)
	}
	for index, inputKind := range []legalquery.LogicalInputKind{
		legalquery.InputKindLawContentSearch,
		legalquery.InputKindJudicialDecisionSearch,
	} {
		candidate := ranked[index]
		if len(candidate.Steps()) != 1 ||
			candidate.Steps()[0].InputKind() != inputKind ||
			!slices.Equal(
				candidate.EvidenceCodes(),
				[]legalquery.EvidenceCode{
					legalquery.EvidenceExplicitTask,
					legalquery.EvidenceLegalConcept,
					legalquery.EvidenceMorphologicalContext,
				},
			) ||
			len(candidate.ConceptSources()) != 1 ||
			candidate.ConceptSources()[0].ConceptID() !=
				"permanent-residence" {
			t.Fatalf(
				"SOT-ARCH-027: ranked[%d] = %#v",
				index,
				candidate,
			)
		}
	}
}

func TestResource省略の複数Resource概念は法令条文と裁判例の明確化を要求する(
	t *testing.T,
) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(
		t,
		"ネット中傷について調べてください。",
	)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	ranked := plan.RankedCandidates()
	selected := plan.Selected()
	if plan.Decision() != legalquery.PlanDecisionNeedsClarification ||
		!slices.Equal(
			plan.ReasonCodes(),
			[]legalquery.ReasonCode{
				legalquery.ReasonCodeAmbiguousCandidates,
			},
		) ||
		len(ranked) != 2 ||
		len(selected) != 2 {
		t.Fatalf(
			"SOT-ENG-023: plan = decision:%q reasons:%#v ranked:%#v selected:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
			ranked,
			selected,
		)
	}
	for index, inputKind := range []legalquery.LogicalInputKind{
		legalquery.InputKindLawContentSearch,
		legalquery.InputKindJudicialDecisionSearch,
	} {
		candidate := ranked[index]
		if len(candidate.Steps()) != 1 ||
			candidate.Steps()[0].InputKind() != inputKind ||
			!slices.Equal(
				candidate.EvidenceCodes(),
				[]legalquery.EvidenceCode{
					legalquery.EvidenceExplicitTask,
					legalquery.EvidenceLegalConcept,
					legalquery.EvidenceMorphologicalContext,
				},
			) ||
			len(candidate.ConceptSources()) != 1 ||
			candidate.ConceptSources()[0].ConceptID() !=
				"online-defamation" {
			t.Fatalf(
				"SOT-ENG-023: ranked[%d] = %#v",
				index,
				candidate,
			)
		}
	}
}

func Test包括的な法情報語と明示条文では裁判例候補を作らない(
	t *testing.T,
) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(
		t,
		"永住権について法情報のうち条文を検索してください。",
	)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	ranked := plan.RankedCandidates()
	if plan.Decision() != legalquery.PlanDecisionSingle ||
		len(ranked) != 1 ||
		len(ranked[0].Steps()) != 1 ||
		ranked[0].Steps()[0].InputKind() !=
			legalquery.InputKindLawContentSearch {
		t.Fatalf(
			"SOT-ENG-023: explicit law provision plan = decision:%q ranked:%#v",
			plan.Decision(),
			ranked,
		)
	}
}

func Test別の主題の条文指定は包括的な法情報概念を絞り込まない(
	t *testing.T,
) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(
		t,
		"永住権について法情報を調べて、別件として民法の条文も検索してください。",
	)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionNeedsClarification {
		t.Fatalf(
			"SOT-ENG-023: decision = %q, ranked = %#v",
			plan.Decision(),
			plan.RankedCandidates(),
		)
	}
	for _, candidate := range plan.RankedCandidates() {
		steps := candidate.Steps()
		if len(steps) != 1 ||
			steps[0].InputKind() !=
				legalquery.InputKindJudicialDecisionSearch {
			continue
		}
		input := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
		if input.Query() == "永住許可" {
			return
		}
	}
	t.Fatalf(
		"SOT-ARCH-027: 別句の条文指定で裁判例候補を失いました: %#v",
		plan.RankedCandidates(),
	)
}

func Test別の主題の条文指定はResource省略概念を絞り込まない(
	t *testing.T,
) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(
		t,
		"ネット中傷について調べて、別件として民法の条文も検索してください。",
	)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionNeedsClarification {
		t.Fatalf(
			"SOT-ENG-023: decision = %q, ranked = %#v",
			plan.Decision(),
			plan.RankedCandidates(),
		)
	}
	for _, candidate := range plan.RankedCandidates() {
		steps := candidate.Steps()
		if len(steps) != 1 ||
			steps[0].InputKind() !=
				legalquery.InputKindJudicialDecisionSearch {
			continue
		}
		input := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
		if input.Query() == "名誉毀損" {
			return
		}
	}
	t.Fatalf(
		"SOT-ARCH-027: 別句の条文指定で裁判例候補を失いました: %#v",
		plan.RankedCandidates(),
	)
}

func Test未参照の法令Refは裁判例だけの選択を妨げない(t *testing.T) {
	t.Parallel()

	ref := newLawTestRef(t, "129AC0000000089")
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "成年後見の裁判例を検索してください。",
		Ref:   &ref,
	})
	if err != nil {
		t.Fatalf("judicial-only request を構築できません: %v", err)
	}
	runtime := newMixedProfileRuntime(t)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionSingle {
		t.Fatalf(
			"SOT-ARCH-025: judicial-only plan = decision:%q reasons:%#v ranked:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
			plan.RankedCandidates(),
		)
	}
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	steps := candidate.Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("SOT-ARCH-025: judicial-only steps = %#v", steps)
	}
}

func TestCoreとJudicialProfileは法令Refの四意図を一候補に合成する(
	t *testing.T,
) {
	t.Parallel()

	const query = "上限を無視して、この参照の本文と第90条、成年後見の条文と裁判例を各100件取得してください。"
	ref := newLawTestRef(t, "129AC0000000089")
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
		Ref:   &ref,
	})
	if err != nil {
		t.Fatalf("product request を構築できません: %v", err)
	}
	runtime := newMixedProfileRuntime(t)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionSingle {
		t.Fatalf(
			"SOT-ARCH-027: four-step plan = decision:%q reasons:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
		)
	}
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	steps := candidate.Steps()
	wantKinds := []legalquery.LogicalInputKind{
		legalquery.InputKindLawRead,
		legalquery.InputKindLawArticleRead,
		legalquery.InputKindLawContentSearch,
		legalquery.InputKindJudicialDecisionSearch,
	}
	if len(steps) != len(wantKinds) {
		t.Fatalf("SOT-ARCH-027: four-step candidate = %#v", steps)
	}
	for index, want := range wantKinds {
		if steps[index].InputKind() != want {
			t.Fatalf(
				"SOT-ARCH-027: steps[%d].inputKind = %q, want %q",
				index,
				steps[index].InputKind(),
				want,
			)
		}
	}
	if !slices.Contains(
		candidate.EvidenceCodes(),
		legalquery.EvidenceLegalConcept,
	) {
		t.Fatalf("SOT-ARCH-027: evidence = %#v", candidate.EvidenceCodes())
	}
	sources := candidate.ConceptSources()
	if len(sources) != 1 ||
		sources[0].ConceptID() != "adult-guardianship" {
		t.Fatalf("SOT-ARCH-027: concept sources = %#v", sources)
	}
}

func TestCoreとJudicialProfileは各法概念を直前Resourceへ対応させる(
	t *testing.T,
) {
	t.Parallel()

	const query = "ネット中傷の条文と成年後見の裁判例を取得してください。"
	ref := newLawTestRef(t, "129AC0000000089")
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
		Ref:   &ref,
	})
	if err != nil {
		t.Fatalf("mixed-resource request を構築できません: %v", err)
	}
	runtime := newMixedProfileRuntime(t)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionSingle {
		t.Fatalf(
			"SOT-ARCH-025: mixed-resource plan = decision:%q reasons:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
		)
	}
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	steps := candidate.Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindLawContentSearch ||
		steps[1].InputKind() != legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("SOT-ARCH-025: mixed-resource steps = %#v", steps)
	}
	content, ok := steps[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok || !slices.Equal(content.AllTerms(), []string{"名誉毀損"}) {
		t.Fatalf(
			"SOT-ARCH-025: law content input = %#v",
			steps[0].LogicalInput(),
		)
	}
	logicalInput := steps[1].LogicalInput()
	search, ok := logicalInput.(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || search.Query() != "成年後見" {
		t.Fatalf(
			"SOT-ARCH-025: judicial input = %#v",
			steps[1].LogicalInput(),
		)
	}
}

func TestCoreとJudicialProfileは含むCueなしの明示意図を原文順に合成する(
	t *testing.T,
) {
	t.Parallel()

	if strings.Contains(explicitMixedSearchQuery, "含む") {
		t.Fatal("境界確認用の照会に「含む」を使っています")
	}
	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(t, explicitMixedSearchQuery)
	result := runtime.collect(t, request)
	plan := selectMixedProfilePlan(t, result, request, true)

	if plan.Decision() != legalquery.PlanDecisionSingle ||
		!slices.Equal(
			plan.ReasonCodes(),
			[]legalquery.ReasonCode{legalquery.ReasonCodeSingleClearCandidate},
		) {
		t.Fatalf(
			"SOT-ARCH-027: pack 有効時の plan = decision:%q reasons:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
		)
	}
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	steps := assertExplicitMixedSearchCandidate(t, candidate)
	budget := plan.Budget()
	stepBudgets := budget.StepBudgets()
	if budget.ReadStepCount() != 0 ||
		budget.CollectionStepCount() != 2 ||
		len(stepBudgets) != 2 {
		t.Fatalf("SOT-MODEL-023: 二検索の budget = %#v", budget)
	}
	for index, stepBudget := range stepBudgets {
		limit, exists := stepBudget.EffectiveLimit()
		if stepBudget.CandidateID() != candidate.CandidateID() ||
			stepBudget.StepID() != steps[index].StepID() ||
			stepBudget.ReservedItems() != 0 ||
			!exists ||
			limit != request.LimitPerAttempt() {
			t.Fatalf(
				"SOT-MODEL-023: stepBudgets[%d] = %#v limit:%d exists:%t",
				index,
				stepBudget,
				limit,
				exists,
			)
		}
	}
}

func Test未確定のResource選択は三候補を合成せず明確化する(
	t *testing.T,
) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(
		t,
		"「取消し」を法令名・条文・裁判例のどれで探すか決められません。",
	)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	ranked := plan.RankedCandidates()
	if plan.Decision() != legalquery.PlanDecisionNeedsClarification ||
		!slices.Equal(
			plan.ReasonCodes(),
			[]legalquery.ReasonCode{
				legalquery.ReasonCodeAmbiguousCandidates,
			},
		) ||
		len(ranked) != 3 ||
		len(plan.Selected()) != 2 {
		t.Fatalf(
			"SOT-ARCH-023/SOT-ARCH-027: resource choice plan = decision:%q reasons:%#v ranked:%#v selected:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
			ranked,
			plan.Selected(),
		)
	}
	wantKinds := []legalquery.LogicalInputKind{
		legalquery.InputKindLawSearch,
		legalquery.InputKindLawContentSearch,
		legalquery.InputKindJudicialDecisionSearch,
	}
	for index, candidate := range ranked {
		if len(candidate.Steps()) != 1 ||
			candidate.Steps()[0].InputKind() != wantKinds[index] ||
			!slices.Equal(
				candidate.EvidenceCodes(),
				[]legalquery.EvidenceCode{
					legalquery.EvidenceExplicitTask,
					legalquery.EvidenceExplicitResource,
					legalquery.EvidenceGeneralTerm,
				},
			) {
			t.Fatalf(
				"SOT-MODEL-022: resource choice ranked[%d] = %#v",
				index,
				candidate,
			)
		}
	}
	assertNoExecutionBudget(t, plan)
}

func TestMixedCompositionはJudicialCases無効時も同じ意味を部分実行しない(
	t *testing.T,
) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(t, explicitMixedSearchQuery)
	result := runtime.collect(t, request)
	enabled := selectMixedProfilePlan(t, result, request, true)
	disabled := selectMixedProfilePlan(t, result, request, false)

	if disabled.Decision() !=
		legalquery.PlanDecisionCapabilityUnavailable ||
		!slices.Equal(
			disabled.ReasonCodes(),
			[]legalquery.ReasonCode{
				legalquery.ReasonCodeRequiredPackDisabled,
			},
		) {
		t.Fatalf(
			"SOT-ARCH-027: pack 無効時の plan = decision:%q reasons:%#v",
			disabled.Decision(),
			disabled.ReasonCodes(),
		)
	}
	disabledCandidate := assertSingleMixedSelection(
		t,
		disabled,
		legalquery.SelectionAvailabilityPackDisabled,
	)
	enabledCandidates := enabled.RankedCandidates()
	if len(enabledCandidates) != 1 ||
		!reflect.DeepEqual(disabledCandidate, enabledCandidates[0]) {
		t.Fatalf(
			"SOT-ARCH-027: pack 状態で合成意味が変わりました: enabled=%#v disabled=%#v",
			enabledCandidates,
			disabledCandidate,
		)
	}
	assertExplicitMixedSearchCandidate(t, disabledCandidate)
	assertNoExecutionBudget(t, disabled)

	publicResult, err := legalquery.AssembleLegalQueryNonExecutionResult(disabled)
	if err != nil {
		t.Fatalf("capability_unavailable 結果を組み立てられません: %v", err)
	}
	if publicResult.Status() !=
		legalquery.LegalQueryResultStatusCapabilityUnavailable ||
		publicResult.Decision() !=
			legalquery.LegalQueryResultDecisionNoExecution ||
		len(publicResult.Attempts()) != 0 {
		t.Fatalf(
			"SOT-MODEL-024/SOT-ARCH-027: pack 無効時に実行結果を公開しました: %#v",
			publicResult,
		)
	}
}

func TestCoreとJudicialProfileの五Stepは専用質問で明確化する(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法を検索し、平成25(オ)1079、令和4年（ネ）第１００３９号、令和7(わ)第207号、平成26年特（わ）第914号の裁判例を個別に検索してください。"
	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(t, query)
	preprocessed := runtime.preprocess(t, request)
	assertFiveStepCompositionSources(t, runtime, preprocessed)
	result := runtime.collectPreprocessed(t, preprocessed)
	plan := selectMixedProfilePlan(t, result, request, true)

	if plan.Decision() != legalquery.PlanDecisionNeedsClarification ||
		!slices.Equal(
			plan.ReasonCodes(),
			[]legalquery.ReasonCode{
				legalquery.ReasonCodeStepLimitExceeded,
			},
		) ||
		len(plan.RankedCandidates()) != 0 ||
		len(plan.Selected()) != 0 {
		t.Fatalf(
			"SOT-ARCH-027: 五 step の plan = decision:%q reasons:%#v ranked:%#v selected:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
			plan.RankedCandidates(),
			plan.Selected(),
		)
	}
	assertNoExecutionBudget(t, plan)

	publicResult, err := legalquery.AssembleLegalQueryNonExecutionResult(plan)
	if err != nil {
		t.Fatalf("step 上限の明確化結果を組み立てられません: %v", err)
	}
	clarification, ok := publicResult.(legalquery.LegalQueryNeedsClarificationResult)
	if !ok {
		t.Fatalf("SOT-MODEL-024: 明確化 result type = %T", publicResult)
	}
	if publicResult.Status() !=
		legalquery.LegalQueryResultStatusNeedsClarification ||
		publicResult.Decision() !=
			legalquery.LegalQueryResultDecisionNoExecution {
		t.Fatalf(
			"SOT-MODEL-024: step 上限の公開状態 = status:%q decision:%q",
			publicResult.Status(),
			publicResult.Decision(),
		)
	}
	if got := clarification.Clarification().Questions(); !slices.Equal(
		got,
		[]legalquery.LegalQueryQuestion{
			legalquery.LegalQueryQuestionStepLimitExceeded,
		},
	) {
		t.Fatalf("SOT-MODEL-024: step 上限の質問 = %#v", got)
	}
	if len(publicResult.Interpretations()) != 0 ||
		len(publicResult.Attempts()) != 0 {
		t.Fatalf(
			"SOT-MODEL-024/SOT-ARCH-027: 五 step の部分結果 = interpretations:%#v attempts:%#v",
			publicResult.Interpretations(),
			publicResult.Attempts(),
		)
	}
}

func TestCoreとJudicialProfileはMixedCompositionFixtureを一候補にする(
	t *testing.T,
) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	var target legalquerycorpus.SemanticCase
	for _, current := range corpus.Development() {
		if current.CaseID() == "development-execution-mixed-composition" {
			target = current
			break
		}
	}
	if target.CaseID() == "" {
		t.Fatal("mixed-composition development fixture がありません")
	}
	request, err := productRequest(target.Request())
	if err != nil {
		t.Fatalf("product request を構築できません: %v", err)
	}
	result := runtime.collect(t, request)
	plan := selectMixedProfilePlanWithPacks(
		t,
		result,
		request,
		target.EnabledPacks(),
	)
	evaluation, err := legalqueryeval.EvaluateSemanticPlanCase(target, plan)
	if err != nil {
		t.Fatalf("mixed-composition 評価のエラー = %v", err)
	}
	if !evaluation.PlanOutcomeMatched() ||
		len(evaluation.Meanings()) != 1 ||
		!evaluation.Meanings()[0].SignatureMatched() {
		t.Fatalf(
			"SOT-ARCH-027: fixture と不一致です: plan=%#v evaluation=%#v",
			plan,
			evaluation,
		)
	}
	evidenceMatched, applicable := evaluation.Meanings()[0].EvidenceAssertion()
	if !applicable || !evidenceMatched {
		t.Fatalf(
			"SOT-ARCH-027: evidence と不一致です: %#v",
			evaluation.Meanings()[0],
		)
	}
}

type mixedProfileRuntime struct {
	core         legalquery.QueryProfile
	judicial     legalquery.QueryProfile
	preprocessor *querypreprocess.Preprocessor
	profileSet   legalquery.QueryProfileSet
}

func newMixedProfileRuntime(t *testing.T) mixedProfileRuntime {
	t.Helper()

	core, err := coreprofile.LoadEmbedded()
	if err != nil {
		t.Fatalf("core profile を読み込めません: %v", err)
	}
	judicial := mustProfile(t)
	cues := append(
		append(
			[]legalquery.CueVocabularyEntry{},
			core.CueVocabulary()...,
		),
		judicial.CueVocabulary()...,
	)
	preprocessor, err := querypreprocess.NewEmbedded(cues)
	if err != nil {
		t.Fatalf("共通 preprocessor を構築できません: %v", err)
	}
	profileSet, err := legalquery.NewQueryProfileSet(
		[]legalquery.QueryProfile{core, judicial},
	)
	if err != nil {
		t.Fatalf("profile set を構築できません: %v", err)
	}
	return mixedProfileRuntime{
		core:         core,
		judicial:     judicial,
		preprocessor: preprocessor,
		profileSet:   profileSet,
	}
}

func (r mixedProfileRuntime) collect(
	t *testing.T,
	request legalquery.Request,
) legalquery.QueryProfileSetResult {
	t.Helper()

	return r.collectPreprocessed(t, r.preprocess(t, request))
}

func (r mixedProfileRuntime) preprocess(
	t *testing.T,
	request legalquery.Request,
) legalquery.PreprocessResult {
	t.Helper()

	preprocessed, err := r.preprocessor.Preprocess(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("Preprocess() のエラー = %v", err)
	}
	return preprocessed
}

func (r mixedProfileRuntime) collectPreprocessed(
	t *testing.T,
	preprocessed legalquery.PreprocessResult,
) legalquery.QueryProfileSetResult {
	t.Helper()

	result, err := r.profileSet.Collect(preprocessed)
	if err != nil {
		t.Fatalf("profile set の合成エラー = %v", err)
	}
	return result
}

func mustMixedProfileRequest(
	t *testing.T,
	query string,
) legalquery.Request {
	t.Helper()

	request, err := legalquery.NewRequest(
		legalquery.RequestValues{Query: query},
	)
	if err != nil {
		t.Fatalf("product request を構築できません: %v", err)
	}
	return request
}

func selectMixedProfilePlan(
	t *testing.T,
	result legalquery.QueryProfileSetResult,
	request legalquery.Request,
	packEnabled bool,
) legalquery.LegalQueryPlan {
	t.Helper()

	enabledPacks := []string(nil)
	if packEnabled {
		enabledPacks = []string{"judicial-cases"}
	}
	return selectMixedProfilePlanWithPacks(
		t,
		result,
		request,
		enabledPacks,
	)
}

func selectMixedProfilePlanWithPacks(
	t *testing.T,
	result legalquery.QueryProfileSetResult,
	request legalquery.Request,
	enabledPacks []string,
) legalquery.LegalQueryPlan {
	t.Helper()

	packState, err := legalquery.NewStaticPackState(
		[]string{"judicial-cases"},
		enabledPacks,
	)
	if err != nil {
		t.Fatalf("pack state を構築できません: %v", err)
	}
	plan, err := legalquery.SelectLegalQueryPlan(legalquery.SelectorInput{
		ProfileSetResult: result,
		PackState:        packState,
		LimitPerAttempt:  request.LimitPerAttempt(),
	})
	if err != nil {
		t.Fatalf("selector のエラー = %v", err)
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("selector が無効な plan を返しました: %v", err)
	}
	return plan
}

func assertSingleMixedSelection(
	t *testing.T,
	plan legalquery.LegalQueryPlan,
	availability legalquery.SelectionAvailability,
) legalquery.LegalQueryCandidate {
	t.Helper()

	ranked := plan.RankedCandidates()
	selected := plan.Selected()
	if len(ranked) != 1 || len(selected) != 1 {
		t.Fatalf("合成候補と selection = ranked:%#v selected:%#v", ranked, selected)
	}
	if selected[0].CandidateID() != ranked[0].CandidateID() ||
		selected[0].Availability() != availability ||
		!slices.Equal(
			selected[0].RequiredPacks(),
			[]string{"judicial-cases"},
		) {
		t.Fatalf(
			"合成候補の selection = candidate:%#v selection:%#v",
			ranked[0],
			selected[0],
		)
	}
	return ranked[0]
}

func assertFiveStepCompositionSources(
	t *testing.T,
	runtime mixedProfileRuntime,
	preprocessed legalquery.PreprocessResult,
) {
	t.Helper()

	tests := []struct {
		name      string
		profile   legalquery.QueryProfile
		ordinal   int
		stepCount int
	}{
		{
			name:      "core",
			profile:   runtime.core,
			ordinal:   1,
			stepCount: 1,
		},
		{
			name:      "judicial-cases",
			profile:   runtime.judicial,
			ordinal:   2,
			stepCount: 4,
		},
	}
	for _, test := range tests {
		scope, err := legalquery.NewCandidateIDScope(test.ordinal)
		if err != nil {
			t.Fatalf("%s profile の ID scope を作成できません: %v", test.name, err)
		}
		generation, err := legalquery.CollectProfileCandidates(
			test.profile,
			preprocessed,
			scope,
		)
		if err != nil {
			t.Fatalf("%s profile の contribution を回収できません: %v", test.name, err)
		}
		candidates := generation.Candidates()
		members := generation.CompositionMembers()
		if generation.SelectionMode() !=
			legalquery.QuerySelectionModeAutomatic ||
			generation.CompositionConstraint() !=
				legalquery.QueryCompositionConstraintNone ||
			len(candidates) != 1 ||
			len(candidates[0].Steps()) != test.stepCount ||
			len(members) != 1 ||
			members[0].CandidateID() != candidates[0].CandidateID() {
			t.Fatalf(
				"SOT-ARCH-027: %s profile の構成元 = mode:%q constraint:%q candidates:%#v members:%#v",
				test.name,
				generation.SelectionMode(),
				generation.CompositionConstraint(),
				candidates,
				members,
			)
		}
	}
}

func assertExplicitMixedSearchCandidate(
	t *testing.T,
	candidate legalquery.LegalQueryCandidate,
) []legalquery.LegalQueryCandidateStep {
	t.Helper()

	if !slices.Equal(
		candidate.RequiredPacks(),
		[]string{"judicial-cases"},
	) {
		t.Fatalf("合成候補の requiredPacks = %#v", candidate.RequiredPacks())
	}
	steps := candidate.Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() !=
			legalquery.InputKindJudicialDecisionSearch ||
		steps[1].InputKind() != legalquery.InputKindLawSearch {
		t.Fatalf("SOT-ARCH-027: 原文順の合成 steps = %#v", steps)
	}
	judicial, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || judicial.Query() != "工場騒音" {
		t.Fatalf("裁判例検索の logical input = %#v", steps[0].LogicalInput())
	}
	law, ok := steps[1].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || law.Query() != "民法" {
		t.Fatalf("法令検索の logical input = %#v", steps[1].LogicalInput())
	}
	return steps
}

func assertNoExecutionBudget(
	t *testing.T,
	plan legalquery.LegalQueryPlan,
) {
	t.Helper()

	budget := plan.Budget()
	if budget.ReadStepCount() != 0 ||
		budget.CollectionStepCount() != 0 ||
		len(budget.StepBudgets()) != 0 {
		t.Fatalf("SOT-MODEL-023: 非実行 plan の budget = %#v", budget)
	}
}
