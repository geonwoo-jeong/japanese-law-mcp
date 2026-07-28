package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type legalQueryTestPortCalls struct {
	mu     sync.Mutex
	values []string
}

func (c *legalQueryTestPortCalls) record(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values = append(c.values, value)
}

func (c *legalQueryTestPortCalls) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.values...)
}

func replaceLegalQueryTestPorts(
	bindings []application.ProviderBindings,
	calls *legalQueryTestPortCalls,
) {
	for index := range bindings {
		switch bindings[index].Descriptor.ProviderID() {
		case "e-gov-law-api-v2":
			bindings[index].LawSearch = legalQueryTestLawSearchPort{calls: calls}
			bindings[index].LawContentSearch = legalQueryTestLawContentSearchPort{
				calls: calls,
			}
			bindings[index].LawDocumentRead = legalQueryTestLawDocumentReadPort{
				calls: calls,
			}
			bindings[index].LawArticleRead = legalQueryTestLawArticleReadPort{
				calls: calls,
			}
		case "e-gov-law-api-v1":
			bindings[index].LawUpdateList = legalQueryTestLawUpdateListPort{
				calls: calls,
			}
		case "courts-hanrei-html":
			bindings[index].JudicialDecisionSearch =
				legalQueryTestJudicialSearchPort{calls: calls}
			bindings[index].JudicialDecisionRead =
				legalQueryTestJudicialReadPort{calls: calls}
		}
	}
}

type legalQueryTestLawSearchPort struct {
	calls *legalQueryTestPortCalls
}

func (p legalQueryTestLawSearchPort) Search(
	_ context.Context,
	request lawsearch.Request,
) (lawsearch.Page, error) {
	p.calls.record("law.search:" + request.Query())
	page, err := emptyLegalQueryTestSourcePage()
	if err != nil {
		return lawsearch.Page{}, err
	}
	return lawsearch.NewPage(lawsearch.PageValues{Page: page})
}

type legalQueryTestLawContentSearchPort struct {
	calls *legalQueryTestPortCalls
}

func (p legalQueryTestLawContentSearchPort) Search(
	_ context.Context,
	_ lawcontentsearch.Request,
) (lawcontentsearch.Page, error) {
	p.calls.record("law.content.search")
	return lawcontentsearch.Page{},
		errors.New("試験で想定していない law.content.search 呼出しです")
}

type legalQueryTestLawDocumentReadPort struct {
	calls *legalQueryTestPortCalls
}

func (p legalQueryTestLawDocumentReadPort) Read(
	_ context.Context,
	_ lawdocumentread.Request,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	p.calls.record("law.document.read")
	return model.SourcedResource[model.LawDocumentRepresentation]{},
		errors.New("試験で想定していない law.document.read 呼出しです")
}

type legalQueryTestLawArticleReadPort struct {
	calls *legalQueryTestPortCalls
}

func (p legalQueryTestLawArticleReadPort) Read(
	_ context.Context,
	_ lawarticleread.Request,
) (model.SourcedResource[model.LawArticleFragment], error) {
	p.calls.record("law.article.read")
	return model.SourcedResource[model.LawArticleFragment]{},
		errors.New("試験で想定していない law.article.read 呼出しです")
}

type legalQueryTestLawUpdateListPort struct {
	calls *legalQueryTestPortCalls
}

func (p legalQueryTestLawUpdateListPort) List(
	_ context.Context,
	_ lawupdatelist.Request,
) (lawupdatelist.Page, error) {
	p.calls.record("law.update.list")
	return lawupdatelist.Page{},
		errors.New("試験で想定していない law.update.list 呼出しです")
}

type legalQueryTestJudicialSearchPort struct {
	calls *legalQueryTestPortCalls
}

func (p legalQueryTestJudicialSearchPort) Search(
	_ context.Context,
	request judicialdecisionsearch.Request,
) (judicialdecisionsearch.Page, error) {
	p.calls.record("judicial-decision.search:" + request.Query())
	page, err := emptyLegalQueryTestSourcePage()
	if err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	return judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{Page: page},
	)
}

type legalQueryTestJudicialReadPort struct {
	calls *legalQueryTestPortCalls
}

func (p legalQueryTestJudicialReadPort) Read(
	_ context.Context,
	request judicialdecisionread.Request,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	ref := request.Ref()
	p.calls.record("judicial-decision.read:" + ref.Key().ResourceID())
	return newLegalQueryTestJudicialDecisionResource(ref)
}

func emptyLegalQueryTestSourcePage() (model.SourcePage, error) {
	return model.NewSourcePage(model.SourcePageValues{ReturnedCount: 0})
}

func newLegalQueryTestJudicialDecisionResource(
	ref model.SourceResourceRef,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	summary, source, err := newLegalQueryTestJudicialDecisionSummary(ref)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	details, err := model.NewJudicialDecisionDetails(
		model.JudicialDecisionDetailsValues{Summary: summary},
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf("試験用裁判例詳細を構築できません: %w", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    ref.Key(),
		URL:            summary.DetailURL(),
		RetrievedAt:    time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-045",
	})
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf("試験用裁判例 provenance を構築できません: %w", err)
	}
	return model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionDetails]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       details,
		},
	)
}

func newLegalQueryTestJudicialDecisionSummary(
	ref model.SourceResourceRef,
) (model.JudicialDecisionSummary, model.InformationSource, error) {
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         ref.Key().SourceID(),
		Name:       "裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		return model.JudicialDecisionSummary{}, model.InformationSource{},
			fmt.Errorf("試験用裁判例情報源を構築できません: %w", err)
	}
	decisionDate, err := model.NewDate("2025-03-03")
	if err != nil {
		return model.JudicialDecisionSummary{}, model.InformationSource{},
			fmt.Errorf("試験用裁判年月日を構築できません: %w", err)
	}
	summary, err := model.NewJudicialDecisionSummary(
		model.JudicialDecisionSummaryValues{
			DecisionID:          ref.Key().ResourceID(),
			PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
			SourceCategoryLabel: "最高裁判例",
			CaseNumber:          "令和6年（受）第1号",
			DecisionDate:        decisionDate,
			CourtName:           "最高裁判所",
			DetailURL: "https://www.courts.go.jp/hanrei/" +
				ref.Key().ResourceID() +
				"/index.html",
			Documents: []model.JudicialDocumentLink{},
			Source:    source,
		},
	)
	if err != nil {
		return model.JudicialDecisionSummary{}, model.InformationSource{},
			fmt.Errorf("試験用裁判例概要を構築できません: %w", err)
	}
	return summary, source, nil
}
