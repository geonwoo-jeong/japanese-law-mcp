package mcp

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func queryLegalInformationResultFixture(
	t *testing.T,
	fixture querySchemaModelFixture,
) map[string]legalquery.LegalQueryResult {
	t.Helper()

	completed, err := legalquery.NewLegalQueryCompletedResult(
		fixture.availablePlan,
		[]legalquery.LegalQueryAttempt{fixture.nonemptyAttempt},
	)
	if err != nil {
		t.Fatalf("completed result を作成できません: %v", err)
	}
	empty, err := legalquery.NewLegalQueryEmptyResult(
		fixture.availablePlan,
		[]legalquery.LegalQueryAttempt{fixture.emptyAttempt},
	)
	if err != nil {
		t.Fatalf("empty result を作成できません: %v", err)
	}
	partial, err := legalquery.NewLegalQueryPartialResult(
		fixture.availablePlan,
		[]legalquery.LegalQueryAttempt{
			fixture.nonemptyAttempt,
			fixture.failedAttempt,
		},
	)
	if err != nil {
		t.Fatalf("partial result を作成できません: %v", err)
	}
	clarification, err := legalquery.NewLegalQueryNeedsClarificationResult(
		fixture.clarificationPlan,
		[]legalquery.LegalQueryQuestion{legalquery.LegalQueryQuestionTask},
	)
	if err != nil {
		t.Fatalf("needs_clarification result を作成できません: %v", err)
	}
	unavailable, err := legalquery.NewLegalQueryCapabilityUnavailableResult(
		fixture.unavailablePlan,
	)
	if err != nil {
		t.Fatalf("capability_unavailable result を作成できません: %v", err)
	}
	unsupported, err := legalquery.NewLegalQueryUnsupportedResult(
		fixture.unsupportedPlan,
	)
	if err != nil {
		t.Fatalf("unsupported result を作成できません: %v", err)
	}
	return map[string]legalquery.LegalQueryResult{
		"completed":              completed,
		"empty":                  empty,
		"partial":                partial,
		"needs_clarification":    clarification,
		"capability_unavailable": unavailable,
		"unsupported":            unsupported,
	}
}

func queryLegalInformationAllFailedError(
	t *testing.T,
	fixture querySchemaModelFixture,
) error {
	t.Helper()

	executor, err := legalquery.NewExecutor(
		queryLegalInformationFailingCoreFacade{},
		nil,
	)
	if err != nil {
		t.Fatalf("executor を作成できません: %v", err)
	}
	_, err = executor.Execute(context.Background(), fixture.availablePlan)
	if err == nil {
		t.Fatal("全 step 失敗が error になりませんでした")
	}
	var allFailed legalquery.LegalQueryAllFailedError
	if !errors.As(err, &allFailed) {
		t.Fatalf("LegalQueryAllFailedError ではありません: %T %v", err, err)
	}
	return err
}

type queryLegalInformationFailingCoreFacade struct{}

func (queryLegalInformationFailingCoreFacade) Validate() error {
	return nil
}

func (queryLegalInformationFailingCoreFacade) SearchLaws(
	context.Context,
	legalquery.LawSearchIntentV1,
	legalquery.LegalQueryStepBudget,
) (lawsearch.Page, error) {
	provider := mustSearchLawsTestProviderDescriptor()
	sourceError := mustSourceError(model.SourceErrorValues{
		Code:       model.SourceErrorCodeRateLimited,
		Provider:   provider,
		Capability: provider.Capabilities()[0],
		Operation:  testOperation("search_laws"),
		RetryAfter: "120",
	})
	executed, err := legalquery.NewExecutedStepError(sourceError)
	if err != nil {
		return lawsearch.Page{}, err
	}
	return lawsearch.Page{}, executed
}

func (queryLegalInformationFailingCoreFacade) SearchLawContent(
	context.Context,
	legalquery.LawContentSearchIntentV1,
	legalquery.LegalQueryStepBudget,
) (lawcontentsearch.Page, error) {
	return lawcontentsearch.Page{}, fmt.Errorf("固定した内部失敗")
}

func (queryLegalInformationFailingCoreFacade) ReadLawDocument(
	context.Context,
	legalquery.LawReadIntentV1,
	legalquery.LegalQueryStepBudget,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	return model.SourcedResource[model.LawDocumentRepresentation]{},
		fmt.Errorf("固定した内部失敗")
}

func (queryLegalInformationFailingCoreFacade) ReadLawArticle(
	context.Context,
	legalquery.LawArticleReadIntentV1,
	legalquery.LegalQueryStepBudget,
) (model.SourcedResource[model.LawArticleFragment], error) {
	return model.SourcedResource[model.LawArticleFragment]{},
		fmt.Errorf("固定した内部失敗")
}

func (queryLegalInformationFailingCoreFacade) ListLawUpdates(
	context.Context,
	legalquery.LawUpdateListIntentV1,
	legalquery.LegalQueryStepBudget,
) (lawupdatelist.Page, error) {
	return lawupdatelist.Page{}, fmt.Errorf("固定した内部失敗")
}

var _ legalquery.CoreCapabilityFacade = queryLegalInformationFailingCoreFacade{}
