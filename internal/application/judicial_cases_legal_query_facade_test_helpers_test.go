package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

var errJudicialFacadeTypedNilMaterializer = errors.New(
	"裁判例 request materializer が typed nil です",
)

type judicialFacadeContextKey struct{}

type judicialFacadeInputs struct {
	search legalquery.JudicialDecisionSearchIntentV1
	read   legalquery.JudicialDecisionReadIntentV1
}

type judicialFacadeBudgets struct {
	search legalquery.LegalQueryStepBudget
	read   legalquery.LegalQueryStepBudget
}

func newJudicialFacadePorts(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceID string,
) *judicialFacadePorts {
	t.Helper()

	payload := newJudicialFacadePayload(t, providerID, sourceID, resourceID)
	return &judicialFacadePorts{
		search: &judicialFacadeSearchPort{result: payload.page},
		read:   &judicialFacadeReadPort{result: payload.details},
	}
}

func newJudicialFacadeBindings(
	t *testing.T,
	providerID string,
	sourceID string,
	ports *judicialFacadePorts,
) application.ProviderBindings {
	t.Helper()

	bindings := newCompleteProviderBindings(t, providerID)
	bindings.JudicialDecisionSearch = ports.search
	bindings.JudicialDecisionRead = ports.read
	return providerBindingsWithSource(t, bindings, sourceID)
}

func judicialFacadeRouteValues(
	providerID string,
) []application.ProviderRouteValues {
	return append(
		completeProviderRouteValues(providerID),
		application.ProviderRouteValues{
			CapabilityID:      judicialdecisionsearch.CapabilityID,
			MajorVersion:      judicialdecisionsearch.MajorVersion,
			Selection:         application.ProviderRouteSelectionPrimary,
			DefaultProviderID: providerID,
		},
		application.ProviderRouteValues{
			CapabilityID:      judicialdecisionread.CapabilityID,
			MajorVersion:      judicialdecisionread.MajorVersion,
			Selection:         application.ProviderRouteSelectionPrimary,
			DefaultProviderID: providerID,
		},
	)
}

func newJudicialFacadeRoutes(
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

func mustJudicialCasesLegalQueryFacade(
	t *testing.T,
	routes application.ProviderRoutes,
	materializer legalquery.JudicialCasesRequestMaterializer,
) application.JudicialCasesLegalQueryFacade {
	t.Helper()

	facade, err := application.NewJudicialCasesLegalQueryFacade(routes, materializer)
	if err != nil {
		t.Fatalf("試験用 JudicialCasesLegalQueryFacade を作成できません: %v", err)
	}
	return facade
}

func newReadyJudicialFacade(
	t *testing.T,
) (*judicialFacadePorts, application.JudicialCasesLegalQueryFacade) {
	t.Helper()
	return newReadyJudicialFacadeWithMaterializer(
		t,
		legalquery.NewJudicialCasesMaterializer(),
	)
}

func newReadyJudicialFacadeWithMaterializer(
	t *testing.T,
	materializer legalquery.JudicialCasesRequestMaterializer,
) (*judicialFacadePorts, application.JudicialCasesLegalQueryFacade) {
	t.Helper()
	ports := newJudicialFacadePorts(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	routes := newJudicialFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newJudicialFacadeBindings(
				t,
				"judicial-provider",
				"judicial-source",
				ports,
			),
		},
		judicialFacadeRouteValues("judicial-provider"),
	)
	return ports, mustJudicialCasesLegalQueryFacade(
		t,
		routes,
		materializer,
	)
}

func newJudicialFacadeInputs(
	t *testing.T,
	ref model.SourceResourceRef,
) judicialFacadeInputs {
	t.Helper()
	return judicialFacadeInputs{
		search: mustJudicialFacadeSearchInput(t, "永住許可"),
		read:   mustJudicialFacadeReadInput(t, ref),
	}
}

func newJudicialFacadeBudgets(
	t *testing.T,
	inputs judicialFacadeInputs,
	limit int,
) judicialFacadeBudgets {
	t.Helper()
	return judicialFacadeBudgets{
		search: judicialFacadeBudgetForInput(
			t,
			"step-judicial-search",
			inputs.search,
			limit,
		),
		read: judicialFacadeBudgetForInput(
			t,
			"step-judicial-read",
			inputs.read,
			limit,
		),
	}
}

func judicialFacadeBudgetForInput(
	t *testing.T,
	stepID string,
	input legalquery.LogicalInput,
	limit int,
) legalquery.LegalQueryStepBudget {
	t.Helper()

	task := legalquery.TaskSearch
	capabilityID := judicialdecisionsearch.CapabilityID
	majorVersion := judicialdecisionsearch.MajorVersion
	if input.InputKind() == legalquery.InputKindJudicialDecisionRead {
		task = legalquery.TaskRead
		capabilityID = judicialdecisionread.CapabilityID
		majorVersion = judicialdecisionread.MajorVersion
	} else if input.InputKind() != legalquery.InputKindJudicialDecisionSearch {
		t.Fatalf("裁判例以外の logical input kind = %q", input.InputKind())
	}
	step, err := legalquery.NewLegalQueryCandidateStep(
		legalquery.LegalQueryCandidateStepValues{
			StepID:                 stepID,
			Task:                   task,
			Resource:               legalquery.ResourceJudicialDecision,
			CapabilityID:           capabilityID,
			CapabilityMajorVersion: majorVersion,
			InputKind:              input.InputKind(),
			LogicalInput:           input,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LegalQueryCandidateStep を作成できません: %v", err)
	}
	requiredPacks := []string{"judicial-cases"}
	candidate, err := legalquery.NewLegalQueryCandidate(
		legalquery.LegalQueryCandidateValues{
			CandidateID:    "candidate-judicial",
			SemanticScore:  100,
			Confidence:     legalquery.ConfidenceHigh,
			EvidenceCodes:  []legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			ConceptSources: []legalquery.LegalConceptSource{},
			RequiredPacks:  requiredPacks,
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
			RequiredPacks: requiredPacks,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LegalQueryPlanSelection を作成できません: %v", err)
	}
	plan, err := legalquery.NewLegalQueryPlan(legalquery.LegalQueryPlanValues{
		ProfileVersion:   "judicial-facade-test-v1",
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

func mustJudicialFacadeSearchInput(
	t *testing.T,
	query string,
) legalquery.JudicialDecisionSearchIntentV1 {
	t.Helper()

	input, err := legalquery.NewJudicialDecisionSearchIntentV1(
		legalquery.JudicialDecisionSearchIntentV1Values{Query: query},
	)
	if err != nil {
		t.Fatalf("試験用 judicial decision search input を作成できません: %v", err)
	}
	return input
}

func mustJudicialFacadeReadInput(
	t *testing.T,
	ref model.SourceResourceRef,
) legalquery.JudicialDecisionReadIntentV1 {
	t.Helper()

	input, err := legalquery.NewJudicialDecisionReadIntentV1(
		legalquery.JudicialDecisionReadIntentV1Values{Ref: ref},
	)
	if err != nil {
		t.Fatalf("試験用 judicial decision read input を作成できません: %v", err)
	}
	return input
}

func assertJudicialFacadeCalls(
	t *testing.T,
	ports *judicialFacadePorts,
	ctx context.Context,
) {
	t.Helper()

	if ports.search.calls != 1 ||
		len(ports.search.contexts) != 1 ||
		ports.search.contexts[0] != ctx {
		t.Fatalf(
			"judicial-decision.search の呼出しまたは context = (%d, %#v)",
			ports.search.calls,
			ports.search.contexts,
		)
	}
	if ports.read.calls != 1 ||
		len(ports.read.contexts) != 1 ||
		ports.read.contexts[0] != ctx {
		t.Fatalf(
			"judicial-decision.read の呼出しまたは context = (%d, %#v)",
			ports.read.calls,
			ports.read.contexts,
		)
	}
}

func assertJudicialFacadeExecutedError(
	t *testing.T,
	err error,
	cause error,
) {
	t.Helper()
	if err == nil {
		t.Fatal("実行済み port のエラーが返されませんでした")
	}
	var executed legalquery.ExecutedStepError
	if !errors.As(err, &executed) || !errors.Is(err, cause) {
		t.Fatalf("port のエラーを ExecutedStepError として保持しませんでした: %v", err)
	}
}

func assertJudicialFacadeFatalError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("致命的な契約エラーが返されませんでした")
	}
	var executed legalquery.ExecutedStepError
	if errors.As(err, &executed) {
		t.Fatalf("契約エラーを ExecutedStepError として分類しました: %v", err)
	}
}

func assertJudicialFacadeResultIdentity(
	t *testing.T,
	gotSearch judicialdecisionsearch.Page,
	wantSearch judicialdecisionsearch.Page,
	gotRead model.SourcedResource[model.JudicialDecisionDetails],
	wantRead model.SourcedResource[model.JudicialDecisionDetails],
) {
	t.Helper()
	if !reflect.DeepEqual(gotSearch, wantSearch) ||
		!reflect.DeepEqual(gotRead, wantRead) {
		t.Fatal("型付き裁判例 result を同一値で返しませんでした")
	}
}
