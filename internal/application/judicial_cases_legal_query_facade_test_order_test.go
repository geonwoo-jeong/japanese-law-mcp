package application_test

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestJudicialCasesLegalQueryFacadeValidatesInputBeforeBudget(
	t *testing.T,
) {
	t.Run("裁判例検索", func(t *testing.T) {
		materializer := &judicialFacadeMaterializerStub{}
		ports, facade := newReadyJudicialFacadeWithMaterializer(t, materializer)
		ref := mustJudicialFacadeRef(
			t,
			"judicial-provider",
			"judicial-source",
			"95570/detail2",
		)
		inputs := newJudicialFacadeInputs(t, ref)
		budgets := newJudicialFacadeBudgets(t, inputs, 5)
		invalidInput := legalquery.JudicialDecisionSearchIntentV1{}
		invalidBudget := legalquery.LegalQueryStepBudget{}

		_, inputOnlyErr := facade.SearchJudicialDecisions(
			context.Background(),
			invalidInput,
			budgets.search,
		)
		assertJudicialFacadeFatalError(t, inputOnlyErr)
		assertJudicialFacadeNotExecuted(t, materializer, ports)

		_, budgetOnlyErr := facade.SearchJudicialDecisions(
			context.Background(),
			inputs.search,
			invalidBudget,
		)
		assertJudicialFacadeFatalError(t, budgetOnlyErr)
		assertJudicialFacadeNotExecuted(t, materializer, ports)

		_, bothErr := facade.SearchJudicialDecisions(
			context.Background(),
			invalidInput,
			invalidBudget,
		)
		assertJudicialFacadeFatalError(t, bothErr)
		assertJudicialFacadeNotExecuted(t, materializer, ports)
		assertJudicialFacadeInputBoundaryFirst(
			t,
			bothErr,
			inputOnlyErr,
			budgetOnlyErr,
		)
	})

	t.Run("裁判例読取り", func(t *testing.T) {
		materializer := &judicialFacadeMaterializerStub{}
		ports, facade := newReadyJudicialFacadeWithMaterializer(t, materializer)
		ref := mustJudicialFacadeRef(
			t,
			"judicial-provider",
			"judicial-source",
			"95570/detail2",
		)
		inputs := newJudicialFacadeInputs(t, ref)
		budgets := newJudicialFacadeBudgets(t, inputs, 5)
		invalidInput := legalquery.JudicialDecisionReadIntentV1{}
		invalidBudget := legalquery.LegalQueryStepBudget{}

		_, inputOnlyErr := facade.ReadJudicialDecision(
			context.Background(),
			invalidInput,
			budgets.read,
		)
		assertJudicialFacadeFatalError(t, inputOnlyErr)
		assertJudicialFacadeNotExecuted(t, materializer, ports)

		_, budgetOnlyErr := facade.ReadJudicialDecision(
			context.Background(),
			inputs.read,
			invalidBudget,
		)
		assertJudicialFacadeFatalError(t, budgetOnlyErr)
		assertJudicialFacadeNotExecuted(t, materializer, ports)

		_, bothErr := facade.ReadJudicialDecision(
			context.Background(),
			invalidInput,
			invalidBudget,
		)
		assertJudicialFacadeFatalError(t, bothErr)
		assertJudicialFacadeNotExecuted(t, materializer, ports)
		assertJudicialFacadeInputBoundaryFirst(
			t,
			bothErr,
			inputOnlyErr,
			budgetOnlyErr,
		)
	})
}

func assertJudicialFacadeNotExecuted(
	t *testing.T,
	materializer *judicialFacadeMaterializerStub,
	ports *judicialFacadePorts,
) {
	t.Helper()
	if materializer.searchCalls != 0 || materializer.readCalls != 0 {
		t.Fatalf(
			"事前検証失敗後の materializer 呼出し = (%d, %d)",
			materializer.searchCalls,
			materializer.readCalls,
		)
	}
	if ports.totalCalls() != 0 {
		t.Fatalf("事前検証失敗後の port 呼出し = %d", ports.totalCalls())
	}
}

func assertJudicialFacadeInputBoundaryFirst(
	t *testing.T,
	bothErr error,
	inputOnlyErr error,
	budgetOnlyErr error,
) {
	t.Helper()
	if bothErr.Error() != inputOnlyErr.Error() {
		t.Fatalf(
			"input と budget の同時失敗で input 境界のエラーを返しませんでした: %v",
			bothErr,
		)
	}
	if bothErr.Error() == budgetOnlyErr.Error() {
		t.Fatalf(
			"input と budget の同時失敗で budget 境界を先に検証しました: %v",
			bothErr,
		)
	}
}
