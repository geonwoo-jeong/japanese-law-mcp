package judicialcases

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func Test公表裁判例検索は引用内の法令誤記補正を混在させない(t *testing.T) {
	t.Parallel()

	runtime := newMixedProfileRuntime(t)
	const query = "公表裁判例で「著作権 演奏権」を検索する"
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
	if !ok || input.Query() != "著作権 演奏権" {
		t.Fatalf(
			"SOT-ARCH-021/SOT-ARCH-027: judicial query = %#v",
			steps[0].LogicalInput(),
		)
	}
}
