package application_test

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type coreFacadeContextKey struct{}

type coreFacadeRefValues struct {
	providerID string
	sourceID   string
}

type coreFacadeLawResourceValues struct {
	lawID      string
	revisionID string
}

type coreFacadeInputs struct {
	lawSearch        legalquery.LawSearchIntentV1
	lawContentSearch legalquery.LawContentSearchIntentV1
	lawDocumentRead  legalquery.LawReadIntentV1
	lawArticleRead   legalquery.LawArticleReadIntentV1
	lawUpdateList    legalquery.LawUpdateListIntentV1
}

type coreFacadeBudgets struct {
	lawSearch        legalquery.LegalQueryStepBudget
	lawContentSearch legalquery.LegalQueryStepBudget
	lawDocumentRead  legalquery.LegalQueryStepBudget
	lawArticleRead   legalquery.LegalQueryStepBudget
	lawUpdateList    legalquery.LegalQueryStepBudget
}

func newCoreFacadePorts(
	t *testing.T,
	providerID string,
	sourceID string,
	date model.Date,
	location model.LawArticleLocation,
) *coreFacadePorts {
	t.Helper()

	fixture := newCoreFacadePayloadFixture(
		t,
		providerID,
		sourceID,
		location,
	)
	return &coreFacadePorts{
		lawSearch: &coreFacadeLawSearchPort{
			result: mustCoreFacadeLawSearchPage(
				t,
				[]model.SourcedResource[model.LawSummary]{fixture.lawSummary},
			),
		},
		lawContentSearch: &coreFacadeLawContentSearchPort{
			result: mustCoreFacadeLawContentPage(
				t,
				[]model.SourcedResource[model.LawContentMatch]{fixture.lawContent},
			),
		},
		lawDocumentRead: &coreFacadeLawDocumentReadPort{
			result: fixture.lawDocument,
		},
		lawArticleRead: &coreFacadeLawArticleReadPort{
			result: fixture.lawArticle,
		},
		lawUpdateList: &coreFacadeLawUpdateListPort{
			result: coreFacadeLawUpdatePage(
				t,
				providerID,
				sourceID,
				date,
				1,
			),
		},
	}
}

func newCoreFacadeBindings(
	t *testing.T,
	providerID string,
	sourceID string,
	ports *coreFacadePorts,
) application.ProviderBindings {
	t.Helper()

	bindings := application.ProviderBindings{
		Descriptor: newBindingDescriptor(
			t,
			providerID,
			lawarticleread.CapabilityID,
			lawcontentsearch.CapabilityID,
			lawdocumentread.CapabilityID,
			lawsearch.CapabilityID,
			lawupdatelist.CapabilityID,
		),
		LawSearch:        ports.lawSearch,
		LawContentSearch: ports.lawContentSearch,
		LawDocumentRead:  ports.lawDocumentRead,
		LawArticleRead:   ports.lawArticleRead,
		LawUpdateList:    ports.lawUpdateList,
	}
	return providerBindingsWithSource(t, bindings, sourceID)
}

func newCoreOnlyFacadeBindings(
	t *testing.T,
	providerID string,
	sourceID string,
	ports *coreFacadePorts,
) application.ProviderBindings {
	t.Helper()
	return newCoreFacadeBindings(t, providerID, sourceID, ports)
}

func newCoreFacadeRoutes(
	t *testing.T,
	bindings []application.ProviderBindings,
	values []application.ProviderRouteValues,
) application.ProviderRoutes {
	t.Helper()

	registry, err := application.NewProviderBindingRegistry(bindings)
	if err != nil {
		t.Fatalf("試験用 ProviderBindingRegistry を作成できません: %v", err)
	}
	routes, err := application.NewProviderRoutes(registry, values)
	if err != nil {
		t.Fatalf("試験用 ProviderRoutes を作成できません: %v", err)
	}
	return routes
}

func mustCoreLegalQueryFacade(
	t *testing.T,
	routes application.ProviderRoutes,
	materializer legalquery.CoreRequestMaterializer,
) application.CoreLegalQueryFacade {
	t.Helper()

	facade, err := application.NewCoreLegalQueryFacade(routes, materializer)
	if err != nil {
		t.Fatalf("試験用 CoreLegalQueryFacade を作成できません: %v", err)
	}
	return facade
}

func newReadyCoreFacade(
	t *testing.T,
	date model.Date,
	location model.LawArticleLocation,
) (*coreFacadePorts, application.CoreLegalQueryFacade) {
	t.Helper()

	ports := newCoreFacadePorts(
		t,
		"core-provider",
		"core-source",
		date,
		location,
	)
	routes := newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newCoreFacadeBindings(
				t,
				"core-provider",
				"core-source",
				ports,
			),
		},
		completeProviderRouteValues("core-provider"),
	)
	return ports, mustCoreLegalQueryFacade(
		t,
		routes,
		legalquery.NewCoreMaterializer(),
	)
}

func coreFacadeRoutesFromReadyFacade(
	t *testing.T,
	date model.Date,
	location model.LawArticleLocation,
) application.ProviderRoutes {
	t.Helper()

	ports := newCoreFacadePorts(
		t,
		"core-provider",
		"core-source",
		date,
		location,
	)
	return newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newCoreFacadeBindings(
				t,
				"core-provider",
				"core-source",
				ports,
			),
		},
		completeProviderRouteValues("core-provider"),
	)
}

func newCoreFacadeInputs(
	t *testing.T,
	asOf model.Date,
	location model.LawArticleLocation,
) coreFacadeInputs {
	t.Helper()

	content, err := legalquery.NewLawContentSearchIntentV1(
		legalquery.LawContentSearchIntentV1Values{
			AllTerms:     []string{"許可"},
			AnyTerms:     []string{"申請"},
			ExcludeTerms: []string{"罰則"},
			AsOf:         &asOf,
		},
	)
	if err != nil {
		t.Fatalf("試験用 law content search input を作成できません: %v", err)
	}
	article, err := legalquery.NewLawArticleReadIntentV1(
		legalquery.LawArticleReadIntentV1Values{
			LawID:    "law-1",
			Location: location,
			AsOf:     &asOf,
		},
	)
	if err != nil {
		t.Fatalf("試験用 law article read input を作成できません: %v", err)
	}
	return coreFacadeInputs{
		lawSearch: mustCoreFacadeLawSearchInput(
			t,
			"行政手続法",
			&asOf,
		),
		lawContentSearch: content,
		lawDocumentRead:  mustCoreFacadeLawReadIDInput(t),
		lawArticleRead:   article,
		lawUpdateList:    mustCoreFacadeLawUpdateInput(t, asOf),
	}
}

func newCoreFacadeBudgets(
	t *testing.T,
	inputs coreFacadeInputs,
	limit int,
) coreFacadeBudgets {
	t.Helper()
	return coreFacadeBudgets{
		lawSearch: coreFacadeBudgetForInput(
			t,
			"step-law-search",
			inputs.lawSearch,
			limit,
		),
		lawContentSearch: coreFacadeBudgetForInput(
			t,
			"step-law-content-search",
			inputs.lawContentSearch,
			limit,
		),
		lawDocumentRead: coreFacadeBudgetForInput(
			t,
			"step-law-document-read",
			inputs.lawDocumentRead,
			limit,
		),
		lawArticleRead: coreFacadeBudgetForInput(
			t,
			"step-law-article-read",
			inputs.lawArticleRead,
			limit,
		),
		lawUpdateList: coreFacadeBudgetForInput(
			t,
			"step-law-update-list",
			inputs.lawUpdateList,
			limit,
		),
	}
}

func coreFacadeBudgetForInput(
	t *testing.T,
	stepID string,
	input legalquery.LogicalInput,
	limit int,
) legalquery.LegalQueryStepBudget {
	t.Helper()

	specification := coreFacadeStepSpecification(t, input)
	step, err := legalquery.NewLegalQueryCandidateStep(
		legalquery.LegalQueryCandidateStepValues{
			StepID:                 stepID,
			Task:                   specification.task,
			Resource:               specification.resource,
			CapabilityID:           specification.capabilityID,
			CapabilityMajorVersion: specification.majorVersion,
			InputKind:              input.InputKind(),
			LogicalInput:           input,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LegalQueryCandidateStep を作成できません: %v", err)
	}
	candidate, err := legalquery.NewLegalQueryCandidate(
		legalquery.LegalQueryCandidateValues{
			CandidateID:    "candidate-core",
			SemanticScore:  100,
			Confidence:     legalquery.ConfidenceHigh,
			EvidenceCodes:  []legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			ConceptSources: []legalquery.LegalConceptSource{},
			RequiredPacks:  []string{},
			Steps:          []legalquery.LegalQueryCandidateStep{step},
		},
	)
	if err != nil {
		t.Fatalf("試験用 LegalQueryCandidate を作成できません: %v", err)
	}
	selection, err := legalquery.NewLegalQueryPlanSelection(
		legalquery.LegalQueryPlanSelectionValues{
			CandidateID:   candidate.CandidateID(),
			Availability:  legalquery.SelectionAvailabilityAvailable,
			RequiredPacks: []string{},
		},
	)
	if err != nil {
		t.Fatalf("試験用 LegalQueryPlanSelection を作成できません: %v", err)
	}
	plan, err := legalquery.NewLegalQueryPlan(legalquery.LegalQueryPlanValues{
		ProfileVersion:   "core-facade-test-v1",
		Decision:         legalquery.PlanDecisionSingle,
		RankedCandidates: []legalquery.LegalQueryCandidate{candidate},
		Selected:         []legalquery.LegalQueryPlanSelection{selection},
		ReasonCodes: []legalquery.ReasonCode{
			legalquery.ReasonCodeSingleClearCandidate,
		},
		LimitPerAttempt: limit,
	})
	if err != nil {
		t.Fatalf("試験用 LegalQueryPlan を作成できません: %v", err)
	}
	budgets := plan.Budget().StepBudgets()
	if len(budgets) != 1 {
		t.Fatalf("試験用 step budget の件数 = %d、期待値 = 1", len(budgets))
	}
	return budgets[0]
}

type coreFacadeStepValues struct {
	task         legalquery.Task
	resource     legalquery.Resource
	capabilityID string
	majorVersion int
}

func coreFacadeStepSpecification(
	t *testing.T,
	input legalquery.LogicalInput,
) coreFacadeStepValues {
	t.Helper()
	switch input.InputKind() {
	case legalquery.InputKindLawSearch:
		return coreFacadeStepValues{
			task:         legalquery.TaskSearch,
			resource:     legalquery.ResourceLaw,
			capabilityID: lawsearch.CapabilityID,
			majorVersion: lawsearch.MajorVersion,
		}
	case legalquery.InputKindLawContentSearch:
		return coreFacadeStepValues{
			task:         legalquery.TaskSearch,
			resource:     legalquery.ResourceLawProvision,
			capabilityID: lawcontentsearch.CapabilityID,
			majorVersion: lawcontentsearch.MajorVersion,
		}
	case legalquery.InputKindLawRead:
		return coreFacadeStepValues{
			task:         legalquery.TaskRead,
			resource:     legalquery.ResourceLaw,
			capabilityID: lawdocumentread.CapabilityID,
			majorVersion: lawdocumentread.MajorVersion,
		}
	case legalquery.InputKindLawArticleRead:
		return coreFacadeStepValues{
			task:         legalquery.TaskRead,
			resource:     legalquery.ResourceLawProvision,
			capabilityID: lawarticleread.CapabilityID,
			majorVersion: lawarticleread.MajorVersion,
		}
	case legalquery.InputKindLawUpdates:
		return coreFacadeStepValues{
			task:         legalquery.TaskListUpdates,
			resource:     legalquery.ResourceLaw,
			capabilityID: lawupdatelist.CapabilityID,
			majorVersion: lawupdatelist.MajorVersion,
		}
	default:
		t.Fatalf("法令コア以外の logical input kind = %q", input.InputKind())
		return coreFacadeStepValues{}
	}
}

func mustCoreFacadeLawSearchInput(
	t *testing.T,
	query string,
	asOf *model.Date,
) legalquery.LawSearchIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: query, AsOf: asOf},
	)
	if err != nil {
		t.Fatalf("試験用 law search input を作成できません: %v", err)
	}
	return input
}

func mustCoreFacadeLawReadIDInput(
	t *testing.T,
) legalquery.LawReadIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawReadIntentV1(
		legalquery.LawReadIntentV1Values{
			LawID:      "law-1",
			RevisionID: "revision-1",
		},
	)
	if err != nil {
		t.Fatalf("試験用 law read ID input を作成できません: %v", err)
	}
	return input
}

func mustCoreFacadeLawReadRefInput(
	t *testing.T,
	ref model.SourceResourceRef,
) legalquery.LawReadIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawReadIntentV1(
		legalquery.LawReadIntentV1Values{Ref: &ref},
	)
	if err != nil {
		t.Fatalf("試験用 law read ref input を作成できません: %v", err)
	}
	return input
}

func mustCoreFacadeLawUpdateInput(
	t *testing.T,
	date model.Date,
) legalquery.LawUpdateListIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawUpdateListIntentV1(
		legalquery.LawUpdateListIntentV1Values{Date: date},
	)
	if err != nil {
		t.Fatalf("試験用 law update input を作成できません: %v", err)
	}
	return input
}

func mustCoreFacadeDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("試験用 Date を作成できません: %v", err)
	}
	return date
}

func mustCoreFacadeLocation(t *testing.T) model.LawArticleLocation {
	t.Helper()
	return mustCoreFacadeLocationAt(t, "1")
}

func mustCoreFacadeLocationAt(
	t *testing.T,
	articleNumber string,
) model.LawArticleLocation {
	t.Helper()
	location, err := model.NewLawArticleLocation(
		model.LawArticleLocationValues{
			Provision:     model.LawArticleProvisionMain,
			ArticleNumber: articleNumber,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawArticleLocation を作成できません: %v", err)
	}
	return location
}

func mustCoreFacadeLawRef(
	t *testing.T,
	providerID string,
	sourceID string,
	lawID string,
	revisionID string,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "law",
		ResourceID:   lawID,
		VersionID:    revisionID,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を作成できません: %v", err)
	}
	return ref
}
