package application_test

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type judicialFacadeSearchPort struct {
	calls    int
	contexts []context.Context
	requests []judicialdecisionsearch.Request
	result   judicialdecisionsearch.Page
	err      error
}

func (p *judicialFacadeSearchPort) Search(
	ctx context.Context,
	request judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	p.calls++
	p.contexts = append(p.contexts, ctx)
	p.requests = append(p.requests, request)
	return p.result, p.err
}

type judicialFacadeReadPort struct {
	calls    int
	contexts []context.Context
	requests []judicialdecisionread.Request
	result   model.SourcedResource[model.JudicialDecisionDetails]
	err      error
}

func (p *judicialFacadeReadPort) Read(
	ctx context.Context,
	request judicialdecisionread.Request,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	p.calls++
	p.contexts = append(p.contexts, ctx)
	p.requests = append(p.requests, request)
	return p.result, p.err
}

type judicialFacadePorts struct {
	search *judicialFacadeSearchPort
	read   *judicialFacadeReadPort
}

func (p *judicialFacadePorts) totalCalls() int {
	return p.search.calls + p.read.calls
}

type judicialFacadeMaterializerStub struct {
	validateErr   error
	searchCalls   int
	searchRequest *judicialdecisionsearch.Request
	searchErr     error
	readCalls     int
	readRequest   *judicialdecisionread.Request
	readErr       error
}

func (m *judicialFacadeMaterializerStub) Validate() error {
	if m == nil {
		return errJudicialFacadeTypedNilMaterializer
	}
	return m.validateErr
}

func (m *judicialFacadeMaterializerStub) MaterializeJudicialDecisionSearch(
	input legalquery.JudicialDecisionSearchIntentV1,
	binding legalquery.SelectedCapabilityBinding,
	budget legalquery.LegalQueryStepBudget,
) (judicialdecisionsearch.Request, error) {
	m.searchCalls++
	if m.searchErr != nil {
		return judicialdecisionsearch.Request{}, m.searchErr
	}
	if m.searchRequest != nil {
		return *m.searchRequest, nil
	}
	return legalquery.NewJudicialCasesMaterializer().
		MaterializeJudicialDecisionSearch(input, binding, budget)
}

func (m *judicialFacadeMaterializerStub) MaterializeJudicialDecisionRead(
	input legalquery.JudicialDecisionReadIntentV1,
	binding legalquery.SelectedCapabilityBinding,
	budget legalquery.LegalQueryStepBudget,
) (judicialdecisionread.Request, error) {
	m.readCalls++
	if m.readErr != nil {
		return judicialdecisionread.Request{}, m.readErr
	}
	if m.readRequest != nil {
		return *m.readRequest, nil
	}
	return legalquery.NewJudicialCasesMaterializer().
		MaterializeJudicialDecisionRead(input, binding, budget)
}
