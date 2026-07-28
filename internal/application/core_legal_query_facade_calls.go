package application

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// SearchLaws は、実効 primary の law.search@1 を一度だけ呼び出す。
func (f CoreLegalQueryFacade) SearchLaws(
	ctx context.Context,
	input legalquery.LawSearchIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (lawsearch.Page, error) {
	if err := f.validateCallContext(ctx); err != nil {
		return lawsearch.Page{}, err
	}
	if err := input.Validate(); err != nil {
		return lawsearch.Page{}, fmt.Errorf(
			"law search logical input が有効ではありません: %w",
			err,
		)
	}
	if err := validateLegalQueryFacadeCollectionBudget(budget); err != nil {
		return lawsearch.Page{}, err
	}
	binding, err := f.primaryBinding(
		lawsearch.CapabilityID,
		lawsearch.MajorVersion,
	)
	if err != nil {
		return lawsearch.Page{}, err
	}
	request, err := f.materializer.MaterializeLawSearch(input, binding, budget)
	if err != nil {
		return lawsearch.Page{}, fmt.Errorf(
			"law.search@1 request を組み立てられません: %w",
			err,
		)
	}
	if err := request.Validate(); err != nil {
		return lawsearch.Page{}, fmt.Errorf(
			"law.search@1 request が有効ではありません: %w",
			err,
		)
	}
	port, exists := f.routes.registry.LawSearch(binding.ProviderID())
	if !exists {
		return lawsearch.Page{}, fmt.Errorf("law.search@1 port が構成されていません")
	}
	result, err := port.Search(ctx, request)
	if err != nil {
		return lawsearch.Page{}, legalQueryFacadeExecutedError(err)
	}
	if err := validateLawSearchFacadeResult(result, request, binding); err != nil {
		return lawsearch.Page{}, err
	}
	return result, nil
}

// SearchLawContent は、実効 primary の law.content.search@1 を一度だけ呼び出す。
func (f CoreLegalQueryFacade) SearchLawContent(
	ctx context.Context,
	input legalquery.LawContentSearchIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (lawcontentsearch.Page, error) {
	if err := f.validateCallContext(ctx); err != nil {
		return lawcontentsearch.Page{}, err
	}
	if err := input.Validate(); err != nil {
		return lawcontentsearch.Page{}, fmt.Errorf(
			"law content search logical input が有効ではありません: %w",
			err,
		)
	}
	if err := validateLegalQueryFacadeCollectionBudget(budget); err != nil {
		return lawcontentsearch.Page{}, err
	}
	binding, err := f.primaryBinding(
		lawcontentsearch.CapabilityID,
		lawcontentsearch.MajorVersion,
	)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}
	request, err := f.materializer.MaterializeLawContentSearch(
		input,
		binding,
		budget,
	)
	if err != nil {
		return lawcontentsearch.Page{}, fmt.Errorf(
			"law.content.search@1 request を組み立てられません: %w",
			err,
		)
	}
	if err := request.Validate(); err != nil {
		return lawcontentsearch.Page{}, fmt.Errorf(
			"law.content.search@1 request が有効ではありません: %w",
			err,
		)
	}
	port, exists := f.routes.registry.LawContentSearch(binding.ProviderID())
	if !exists {
		return lawcontentsearch.Page{},
			fmt.Errorf("law.content.search@1 port が構成されていません")
	}
	result, err := port.Search(ctx, request)
	if err != nil {
		return lawcontentsearch.Page{}, legalQueryFacadeExecutedError(err)
	}
	if err := validateLawContentFacadeResult(result, request, binding); err != nil {
		return lawcontentsearch.Page{}, err
	}
	return result, nil
}

// ReadLawDocument は、ID では primary、ref では明示 binding の read を呼び出す。
func (f CoreLegalQueryFacade) ReadLawDocument(
	ctx context.Context,
	input legalquery.LawReadIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	if err := f.validateCallContext(ctx); err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	if err := input.Validate(); err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{},
			fmt.Errorf("law read logical input が有効ではありません: %w", err)
	}
	if err := validateLegalQueryFacadeReadBudget(budget); err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	binding, err := f.lawBinding(
		input,
		lawdocumentread.CapabilityID,
		lawdocumentread.MajorVersion,
	)
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	request, err := f.materializer.MaterializeLawDocumentRead(
		input,
		binding,
		budget,
	)
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{},
			fmt.Errorf("law.document.read@1 request を組み立てられません: %w", err)
	}
	if err := request.Validate(); err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{},
			fmt.Errorf("law.document.read@1 request が有効ではありません: %w", err)
	}
	port, exists := f.routes.registry.LawDocumentRead(binding.ProviderID())
	if !exists {
		return model.SourcedResource[model.LawDocumentRepresentation]{},
			fmt.Errorf("law.document.read@1 port が構成されていません")
	}
	result, err := port.Read(ctx, request)
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{},
			legalQueryFacadeExecutedError(err)
	}
	if err := validateLawDocumentFacadeResult(
		result,
		request,
		binding,
	); err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	return result, nil
}

// ReadLawArticle は、ID では primary、ref では明示 binding の read を呼び出す。
func (f CoreLegalQueryFacade) ReadLawArticle(
	ctx context.Context,
	input legalquery.LawArticleReadIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (model.SourcedResource[model.LawArticleFragment], error) {
	if err := f.validateCallContext(ctx); err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	if err := input.Validate(); err != nil {
		return model.SourcedResource[model.LawArticleFragment]{},
			fmt.Errorf("law article read logical input が有効ではありません: %w", err)
	}
	if err := validateLegalQueryFacadeReadBudget(budget); err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	binding, err := f.lawBinding(
		input,
		lawarticleread.CapabilityID,
		lawarticleread.MajorVersion,
	)
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	request, err := f.materializer.MaterializeLawArticleRead(
		input,
		binding,
		budget,
	)
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{},
			fmt.Errorf("law.article.read@1 request を組み立てられません: %w", err)
	}
	if err := request.Validate(); err != nil {
		return model.SourcedResource[model.LawArticleFragment]{},
			fmt.Errorf("law.article.read@1 request が有効ではありません: %w", err)
	}
	port, exists := f.routes.registry.LawArticleRead(binding.ProviderID())
	if !exists {
		return model.SourcedResource[model.LawArticleFragment]{},
			fmt.Errorf("law.article.read@1 port が構成されていません")
	}
	result, err := port.Read(ctx, request)
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{},
			legalQueryFacadeExecutedError(err)
	}
	if err := validateLawArticleFacadeResult(
		result,
		request,
		binding,
	); err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	return result, nil
}

// ListLawUpdates は、実効 primary の完全な law.update.list@1 を一度だけ呼び出す。
func (f CoreLegalQueryFacade) ListLawUpdates(
	ctx context.Context,
	input legalquery.LawUpdateListIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (lawupdatelist.Page, error) {
	if err := f.validateCallContext(ctx); err != nil {
		return lawupdatelist.Page{}, err
	}
	if err := input.Validate(); err != nil {
		return lawupdatelist.Page{}, fmt.Errorf(
			"law update list logical input が有効ではありません: %w",
			err,
		)
	}
	if err := validateLegalQueryFacadeCollectionBudget(budget); err != nil {
		return lawupdatelist.Page{}, err
	}
	binding, err := f.primaryBinding(
		lawupdatelist.CapabilityID,
		lawupdatelist.MajorVersion,
	)
	if err != nil {
		return lawupdatelist.Page{}, err
	}
	request, err := f.materializer.MaterializeLawUpdateList(
		input,
		binding,
		budget,
	)
	if err != nil {
		return lawupdatelist.Page{}, fmt.Errorf(
			"law.update.list@1 request を組み立てられません: %w",
			err,
		)
	}
	if err := request.Validate(); err != nil {
		return lawupdatelist.Page{}, fmt.Errorf(
			"law.update.list@1 request が有効ではありません: %w",
			err,
		)
	}
	port, exists := f.routes.registry.LawUpdateList(binding.ProviderID())
	if !exists {
		return lawupdatelist.Page{},
			fmt.Errorf("law.update.list@1 port が構成されていません")
	}
	result, err := port.List(ctx, request)
	if err != nil {
		return lawupdatelist.Page{}, legalQueryFacadeExecutedError(err)
	}
	if err := validateLawUpdateFacadeResult(result, request, binding); err != nil {
		return lawupdatelist.Page{}, err
	}
	return result, nil
}

func legalQueryFacadeExecutedError(cause error) error {
	result, err := legalquery.NewExecutedStepError(cause)
	if err != nil {
		return fmt.Errorf("provider port error を分類できません: %w", err)
	}
	return result
}
