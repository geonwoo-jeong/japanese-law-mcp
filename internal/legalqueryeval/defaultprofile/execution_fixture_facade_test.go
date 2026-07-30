package defaultprofile

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestExecutionFixtureFacadeは全Core能力Variantを再生する(t *testing.T) {
	articleInput, updateInput := executionFixtureUnusedCoreInputs(t)
	plan := executionFixtureCorePlan(t, articleInput, updateInput)
	steps := plan.RankedCandidates()[0].Steps()
	budgets := plan.Budget().StepBudgets()
	readSuccess, err := legalquerycorpus.NewReadSuccessOutcome()
	if err != nil {
		t.Fatalf("read_success を作成できません: %v", err)
	}
	collectionSuccess, err := legalquerycorpus.NewCollectionSuccessOutcome(2)
	if err != nil {
		t.Fatalf("collection_success を作成できません: %v", err)
	}
	actions := []resolvedExecutionAction{
		{
			meaningID:    "core",
			stepOrdinal:  1,
			releaseOrder: 1,
			candidateID:  plan.Selected()[0].CandidateID(),
			step:         steps[0],
			budget:       budgets[0],
			outcome:      readSuccess,
		},
		{
			meaningID:    "core",
			stepOrdinal:  2,
			releaseOrder: 2,
			candidateID:  plan.Selected()[0].CandidateID(),
			step:         steps[1],
			budget:       budgets[1],
			outcome:      collectionSuccess,
		},
	}
	fixture := resolvedExecutionFixture{
		actions: actions,
		actionByStepID: map[string]resolvedExecutionAction{
			steps[0].StepID(): actions[0],
			steps[1].StepID(): actions[1],
		},
	}
	facade, err := newExecutionFixtureFacade(fixture)
	if err != nil {
		t.Fatalf("execution fixture facade を作成できません: %v", err)
	}

	article, err := facade.ReadLawArticle(
		context.Background(),
		articleInput,
		budgets[0],
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: law_article_read を再生できません: %v", err)
	}
	if err := article.Validate(); err != nil {
		t.Fatalf("law article payload が有効ではありません: %v", err)
	}
	updates, err := facade.ListLawUpdates(
		context.Background(),
		updateInput,
		budgets[1],
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: law_updates を再生できません: %v", err)
	}
	if err := updates.Validate(); err != nil {
		t.Fatalf("law updates payload が有効ではありません: %v", err)
	}
	if len(updates.Items()) != 2 || updates.Date() != updateInput.Date() {
		t.Fatalf("law updates payload = %#v", updates)
	}
}

func TestExecutionFixtureFacadeは重複StepActionを拒否する(t *testing.T) {
	articleInput, _ := executionFixtureUnusedCoreInputs(t)
	plan := executionFixtureCorePlan(t, articleInput)
	step := plan.RankedCandidates()[0].Steps()[0]
	budget := plan.Budget().StepBudgets()[0]
	readSuccess, err := legalquerycorpus.NewReadSuccessOutcome()
	if err != nil {
		t.Fatalf("read_success を作成できません: %v", err)
	}
	actions := []resolvedExecutionAction{
		{
			releaseOrder: 1,
			step:         step,
			budget:       budget,
			outcome:      readSuccess,
		},
		{
			releaseOrder: 2,
			step:         step,
			budget:       budget,
			outcome:      readSuccess,
		},
	}
	if _, err := newExecutionFixtureFacade(resolvedExecutionFixture{
		actions: actions,
		actionByStepID: map[string]resolvedExecutionAction{
			step.StepID(): actions[1],
		},
	}); err == nil {
		t.Fatal("SOT-ENG-026: 同じ step の action 重複を受理しました")
	}
}

func executionFixtureUnusedCoreInputs(
	t *testing.T,
) (legalquery.LawArticleReadIntentV1, legalquery.LawUpdateListIntentV1) {
	t.Helper()

	location, err := model.NewLawArticleLocation(
		model.LawArticleLocationValues{
			Provision:     model.LawArticleProvisionMain,
			ArticleNumber: "1",
		},
	)
	if err != nil {
		t.Fatalf("条文位置を作成できません: %v", err)
	}
	article, err := legalquery.NewLawArticleReadIntentV1(
		legalquery.LawArticleReadIntentV1Values{
			LawID:    "evaluation-law",
			Location: location,
		},
	)
	if err != nil {
		t.Fatalf("law article input を作成できません: %v", err)
	}
	date, err := model.NewDate("2026-07-28")
	if err != nil {
		t.Fatalf("更新日を作成できません: %v", err)
	}
	updates, err := legalquery.NewLawUpdateListIntentV1(
		legalquery.LawUpdateListIntentV1Values{Date: date},
	)
	if err != nil {
		t.Fatalf("law updates input を作成できません: %v", err)
	}
	return article, updates
}

func executionFixtureCorePlan(
	t *testing.T,
	inputs ...legalquery.LogicalInput,
) legalquery.LegalQueryPlan {
	t.Helper()

	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("candidate scope を作成できません: %v", err)
	}
	candidate, err := legalquery.AssembleLegalQueryCandidate(
		legalquery.CandidateAssemblyValues{
			IDScope:          scope,
			CandidateOrdinal: 1,
			SemanticScore:    100,
			Confidence:       legalquery.ConfidenceHigh,
			EvidenceCodes: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			LogicalInputs: inputs,
		},
	)
	if err != nil {
		t.Fatalf("candidate を作成できません: %v", err)
	}
	selection, err := legalquery.NewLegalQueryPlanSelection(
		legalquery.LegalQueryPlanSelectionValues{
			CandidateID:  candidate.CandidateID(),
			Availability: legalquery.SelectionAvailabilityAvailable,
		},
	)
	if err != nil {
		t.Fatalf("selection を作成できません: %v", err)
	}
	plan, err := legalquery.NewLegalQueryPlan(legalquery.LegalQueryPlanValues{
		ProfileVersion:   "evaluation-fixture-v1",
		Decision:         legalquery.PlanDecisionSingle,
		RankedCandidates: []legalquery.LegalQueryCandidate{candidate},
		Selected:         []legalquery.LegalQueryPlanSelection{selection},
		ReasonCodes: []legalquery.ReasonCode{
			legalquery.ReasonCodeSingleClearCandidate,
		},
		LimitPerAttempt: 20,
	})
	if err != nil {
		t.Fatalf("plan を作成できません: %v", err)
	}
	return plan
}
