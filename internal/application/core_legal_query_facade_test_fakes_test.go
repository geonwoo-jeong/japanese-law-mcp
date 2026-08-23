package application_test

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawrevisionlist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawtarget"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type coreFacadeLawSearchPort struct {
	calls    int
	contexts []context.Context
	requests []lawsearch.Request
	result   lawsearch.Page
	err      error
}

type coreFacadeLawTargetResolver struct {
	queries  []string
	target   lawtarget.ResolvedLawTarget
	resolved bool
	err      error
}

func (r *coreFacadeLawTargetResolver) ResolveLogicalInput(
	_ context.Context,
	query string,
) (lawtarget.ResolvedLawTarget, bool, error) {
	r.queries = append(r.queries, query)
	return r.target, r.resolved, r.err
}

func (p *coreFacadeLawSearchPort) Search(
	ctx context.Context,
	request lawsearch.Request,
) (lawsearch.Page, error) {
	p.calls++
	p.contexts = append(p.contexts, ctx)
	p.requests = append(p.requests, request)
	return p.result, p.err
}

type coreFacadeLawContentSearchPort struct {
	calls    int
	contexts []context.Context
	requests []lawcontentsearch.Request
	result   lawcontentsearch.Page
	err      error
}

func (p *coreFacadeLawContentSearchPort) Search(
	ctx context.Context,
	request lawcontentsearch.Request,
) (lawcontentsearch.Page, error) {
	p.calls++
	p.contexts = append(p.contexts, ctx)
	p.requests = append(p.requests, request)
	return p.result, p.err
}

type coreFacadeLawDocumentReadPort struct {
	calls    int
	contexts []context.Context
	requests []lawdocumentread.Request
	result   model.SourcedResource[model.LawDocumentRepresentation]
	err      error
}

func (p *coreFacadeLawDocumentReadPort) Read(
	ctx context.Context,
	request lawdocumentread.Request,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	p.calls++
	p.contexts = append(p.contexts, ctx)
	p.requests = append(p.requests, request)
	return p.result, p.err
}

type coreFacadeLawArticleReadPort struct {
	calls    int
	contexts []context.Context
	requests []lawarticleread.Request
	result   model.SourcedResource[model.LawArticleFragment]
	err      error
}

func (p *coreFacadeLawArticleReadPort) Read(
	ctx context.Context,
	request lawarticleread.Request,
) (model.SourcedResource[model.LawArticleFragment], error) {
	p.calls++
	p.contexts = append(p.contexts, ctx)
	p.requests = append(p.requests, request)
	return p.result, p.err
}

type coreFacadeLawUpdateListPort struct {
	calls    int
	contexts []context.Context
	requests []lawupdatelist.Request
	result   lawupdatelist.Page
	err      error
}

type coreFacadeLawRevisionListPort struct {
	calls int
}

func (p *coreFacadeLawRevisionListPort) List(
	context.Context,
	lawrevisionlist.Request,
) (lawrevisionlist.Page, error) {
	p.calls++
	return lawrevisionlist.Page{}, nil
}

func (p *coreFacadeLawUpdateListPort) List(
	ctx context.Context,
	request lawupdatelist.Request,
) (lawupdatelist.Page, error) {
	p.calls++
	p.contexts = append(p.contexts, ctx)
	p.requests = append(p.requests, request)
	return p.result, p.err
}

type coreFacadePorts struct {
	lawSearch        *coreFacadeLawSearchPort
	lawContentSearch *coreFacadeLawContentSearchPort
	lawDocumentRead  *coreFacadeLawDocumentReadPort
	lawArticleRead   *coreFacadeLawArticleReadPort
	lawRevisionList  *coreFacadeLawRevisionListPort
	lawUpdateList    *coreFacadeLawUpdateListPort
}

func (p *coreFacadePorts) totalCalls() int {
	return p.lawSearch.calls +
		p.lawContentSearch.calls +
		p.lawDocumentRead.calls +
		p.lawArticleRead.calls +
		p.lawRevisionList.calls +
		p.lawUpdateList.calls
}

type coreFacadeMaterializerStub struct {
	lawSearchRequest *lawsearch.Request
	lawSearchErr     error
	validateErr      error
}

func (m *coreFacadeMaterializerStub) Validate() error {
	if m.validateErr != nil {
		return m.validateErr
	}
	return nil
}

func (m *coreFacadeMaterializerStub) MaterializeLawSearch(
	input legalquery.LawSearchIntentV1,
	binding legalquery.SelectedCapabilityBinding,
	budget legalquery.LegalQueryStepBudget,
) (lawsearch.Request, error) {
	if m.lawSearchErr != nil {
		return lawsearch.Request{}, m.lawSearchErr
	}
	if m.lawSearchRequest != nil {
		return *m.lawSearchRequest, nil
	}
	return legalquery.NewCoreMaterializer().MaterializeLawSearch(
		input,
		binding,
		budget,
	)
}

func (*coreFacadeMaterializerStub) MaterializeLawContentSearch(
	input legalquery.LawContentSearchIntentV1,
	binding legalquery.SelectedCapabilityBinding,
	budget legalquery.LegalQueryStepBudget,
) (lawcontentsearch.Request, error) {
	return legalquery.NewCoreMaterializer().MaterializeLawContentSearch(
		input,
		binding,
		budget,
	)
}

func (*coreFacadeMaterializerStub) MaterializeLawDocumentRead(
	input legalquery.LawReadIntentV1,
	binding legalquery.SelectedCapabilityBinding,
	budget legalquery.LegalQueryStepBudget,
) (lawdocumentread.Request, error) {
	return legalquery.NewCoreMaterializer().MaterializeLawDocumentRead(
		input,
		binding,
		budget,
	)
}

func (*coreFacadeMaterializerStub) MaterializeLawArticleRead(
	input legalquery.LawArticleReadIntentV1,
	binding legalquery.SelectedCapabilityBinding,
	budget legalquery.LegalQueryStepBudget,
) (lawarticleread.Request, error) {
	return legalquery.NewCoreMaterializer().MaterializeLawArticleRead(
		input,
		binding,
		budget,
	)
}

func (*coreFacadeMaterializerStub) MaterializeLawUpdateList(
	input legalquery.LawUpdateListIntentV1,
	binding legalquery.SelectedCapabilityBinding,
	budget legalquery.LegalQueryStepBudget,
) (lawupdatelist.Request, error) {
	return legalquery.NewCoreMaterializer().MaterializeLawUpdateList(
		input,
		binding,
		budget,
	)
}
