package legalquery

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// Port は、transport adapter が統合法情報照会を一回実行するための入口である。
type Port interface {
	Query(context.Context, Request) (LegalQueryResult, error)
}

// CoreCapabilityFacade は、法令コアの logical input を既存の型付き能力へ接続する。
type CoreCapabilityFacade interface {
	Validate() error
	SearchLaws(
		context.Context,
		LawSearchIntentV1,
		LegalQueryStepBudget,
	) (lawsearch.Page, error)
	SearchLawContent(
		context.Context,
		LawContentSearchIntentV1,
		LegalQueryStepBudget,
	) (lawcontentsearch.Page, error)
	ReadLawDocument(
		context.Context,
		LawReadIntentV1,
		LegalQueryStepBudget,
	) (model.SourcedResource[model.LawDocumentRepresentation], error)
	ReadLawArticle(
		context.Context,
		LawArticleReadIntentV1,
		LegalQueryStepBudget,
	) (model.SourcedResource[model.LawArticleFragment], error)
	ListLawUpdates(
		context.Context,
		LawUpdateListIntentV1,
		LegalQueryStepBudget,
	) (lawupdatelist.Page, error)
}

// JudicialCasesCapabilityFacade は、裁判例 pack の logical input を型付き能力へ接続する。
type JudicialCasesCapabilityFacade interface {
	Validate() error
	SearchJudicialDecisions(
		context.Context,
		JudicialDecisionSearchIntentV1,
		LegalQueryStepBudget,
	) (judicialdecisionsearch.Page, error)
	ReadJudicialDecision(
		context.Context,
		JudicialDecisionReadIntentV1,
		LegalQueryStepBudget,
	) (model.SourcedResource[model.JudicialDecisionDetails], error)
}
