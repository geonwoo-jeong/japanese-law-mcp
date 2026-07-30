package judicialcases

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func Test日付が介在しても裁判所法の誤記候補を裁判例検索へ混ぜない(
	t *testing.T,
) {
	t.Parallel()

	steps := selectedJudicialTypoBoundarySteps(
		t,
		"裁判所の2025年1月1日の裁判例から「解雇権濫用」を探す",
	)
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("SOT-ARCH-021/SOT-ARCH-027: steps = %#v", steps)
	}
}

func Test別節の法令Resourceがあっても裁判所法の誤記候補を混ぜない(
	t *testing.T,
) {
	t.Parallel()

	steps := selectedJudicialTypoBoundarySteps(
		t,
		"裁判所の裁判例から「解雇権濫用」を探し、民法の法律も検索する",
	)
	assertJudicialAndLawSearchSteps(t, steps, "解雇権濫用", "民法")
}

func Test明示検索に結び付く一意な法令誤記は裁判例との混合でも保持する(
	t *testing.T,
) {
	t.Parallel()

	steps := selectedJudicialTypoBoundarySteps(
		t,
		"建築規準法を検索し、裁判例を「解雇権濫用」で検索する",
	)
	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindLawSearch ||
		steps[1].InputKind() != legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("SOT-ARCH-021/SOT-ARCH-027: steps = %#v", steps)
	}
	law, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || law.Query() != "建築基準法" {
		t.Fatalf("SOT-ARCH-021: law input = %#v", steps[0].LogicalInput())
	}
}

func Test後続法令Resourceがあっても引用内の法令誤記候補を混ぜない(
	t *testing.T,
) {
	t.Parallel()

	steps := selectedJudicialTypoBoundarySteps(
		t,
		"公表裁判例で「著作権 演奏権」を検索し、民法の法律も検索する",
	)
	assertJudicialAndLawSearchSteps(t, steps, "著作権 演奏権", "民法")
}

func selectedJudicialTypoBoundarySteps(
	t *testing.T,
	query string,
) []legalquery.LegalQueryCandidateStep {
	t.Helper()

	runtime := newMixedProfileRuntime(t)
	request := mustMixedProfileRequest(t, query)
	plan := selectMixedProfilePlan(t, runtime.collect(t, request), request, true)
	if plan.Decision() != legalquery.PlanDecisionSingle {
		t.Fatalf(
			"SOT-ARCH-027: plan = decision:%q reasons:%#v ranked:%#v",
			plan.Decision(),
			plan.ReasonCodes(),
			plan.RankedCandidates(),
		)
	}
	return assertSingleMixedSelection(
		t,
		plan,
		legalquery.SelectionAvailabilityAvailable,
	).Steps()
}

func assertJudicialAndLawSearchSteps(
	t *testing.T,
	steps []legalquery.LegalQueryCandidateStep,
	judicialQuery string,
	lawQuery string,
) {
	t.Helper()

	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindJudicialDecisionSearch ||
		steps[1].InputKind() != legalquery.InputKindLawSearch {
		t.Fatalf("SOT-ARCH-021/SOT-ARCH-027: steps = %#v", steps)
	}
	judicial, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if !ok || judicial.Query() != judicialQuery {
		t.Fatalf("SOT-ARCH-027: judicial input = %#v", steps[0].LogicalInput())
	}
	law, ok := steps[1].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || law.Query() != lawQuery {
		t.Fatalf("SOT-ARCH-027: law input = %#v", steps[1].LogicalInput())
	}
}
