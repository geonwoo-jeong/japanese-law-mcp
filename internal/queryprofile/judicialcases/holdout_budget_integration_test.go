package judicialcases

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestCoreとJudicialProfileは更新日付き複合要求を四Stepに分離する(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法を検索し、成年後見の条文・2026年7月1日の更新・裁判例も各一ページ確認してください。"
	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(t, query)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	if plan.Decision() != legalquery.PlanDecisionSingle {
		t.Fatalf(
			"SOT-ARCH-027: explicit-date mixed plan = decision:%q reasons:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
		)
	}
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	if !slices.Equal(
		candidate.EvidenceCodes(),
		[]legalquery.EvidenceCode{
			legalquery.EvidenceStructuredReference,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceOfficialAlias,
			legalquery.EvidenceLegalConcept,
		},
	) {
		t.Fatalf(
			"SOT-MODEL-022: explicit-date mixed evidence = %#v",
			candidate.EvidenceCodes(),
		)
	}

	steps := candidate.Steps()
	wantKinds := []legalquery.LogicalInputKind{
		legalquery.InputKindLawSearch,
		legalquery.InputKindLawContentSearch,
		legalquery.InputKindLawUpdates,
		legalquery.InputKindJudicialDecisionSearch,
	}
	if len(steps) != len(wantKinds) {
		t.Fatalf("SOT-ARCH-027: explicit-date mixed steps = %#v", steps)
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

	law, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || law.Query() != "民法" {
		t.Fatalf(
			"SOT-ARCH-027: law search input = %#v",
			steps[0].LogicalInput(),
		)
	}
	if _, exists := law.AsOf(); exists {
		t.Fatalf("更新日を法令検索の asOf に流用しました: %#v", law)
	}
	content, ok := steps[1].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok || !slices.Equal(content.AllTerms(), []string{"成年後見"}) {
		t.Fatalf(
			"SOT-ARCH-027: law content input = %#v",
			steps[1].LogicalInput(),
		)
	}
	if _, exists := content.AsOf(); exists {
		t.Fatalf("更新日を条文検索の asOf に流用しました: %#v", content)
	}
	updates, ok := steps[2].LogicalInput().(legalquery.LawUpdateListIntentV1)
	if !ok || updates.Date().String() != "2026-07-01" {
		t.Fatalf(
			"SOT-ARCH-027: law updates input = %#v",
			steps[2].LogicalInput(),
		)
	}
	judicial, ok := steps[3].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || judicial.Query() != "成年後見" {
		t.Fatalf(
			"SOT-ARCH-027: judicial input = %#v",
			steps[3].LogicalInput(),
		)
	}

	budget := plan.Budget()
	if budget.ReadStepCount() != 0 ||
		budget.CollectionStepCount() != 4 ||
		len(budget.StepBudgets()) != 4 {
		t.Fatalf("SOT-MODEL-023: explicit-date mixed budget = %#v", budget)
	}
}

func Test裁判例の共有一般語は近接する更新日より優先する(t *testing.T) {
	t.Parallel()

	const query = "民法を検索し、「営業秘密」の条文・2026年7月1日の更新・裁判例も各一ページ確認してください。"
	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(t, query)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	steps := candidate.Steps()
	if len(steps) != 4 ||
		steps[3].InputKind() !=
			legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("SOT-ARCH-027: shared term steps = %#v", steps)
	}
	judicial, ok := steps[3].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || judicial.Query() != "営業秘密" {
		t.Fatalf(
			"SOT-ARCH-027: shared judicial term = %#v",
			steps[3].LogicalInput(),
		)
	}
}

func Test裁判例の共有一般語は個別指定なしでも更新日より優先する(
	t *testing.T,
) {
	t.Parallel()

	const query = "労働基準法、時間外労働の条文、2026年6月29日の更新一覧、裁判例を検索してください。"
	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(t, query)
	plan := selectMixedProfilePlan(
		t,
		runtime.collect(t, request),
		request,
		true,
	)
	candidate := assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	)
	steps := candidate.Steps()
	if len(steps) != 4 ||
		steps[3].InputKind() !=
			legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("SOT-ARCH-027: shared term steps = %#v", steps)
	}
	judicial, ok := steps[3].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || judicial.Query() != "時間外労働" {
		t.Fatalf(
			"SOT-ARCH-027: shared judicial term = %#v",
			steps[3].LogicalInput(),
		)
	}
}
