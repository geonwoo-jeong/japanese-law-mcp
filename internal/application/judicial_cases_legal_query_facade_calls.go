package application

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// SearchJudicialDecisions は、実効 primary の judicial-decision.search@1 を一度だけ呼び出す。
func (f JudicialCasesLegalQueryFacade) SearchJudicialDecisions(
	ctx context.Context,
	input legalquery.JudicialDecisionSearchIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (judicialdecisionsearch.Page, error) {
	if err := f.validateCallContext(ctx); err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	if err := input.Validate(); err != nil {
		return judicialdecisionsearch.Page{}, fmt.Errorf(
			"judicial decision search logical input が有効ではありません: %w",
			err,
		)
	}
	if err := validateLegalQueryFacadeCollectionBudget(budget); err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	binding, err := f.primaryBinding(
		judicialdecisionsearch.CapabilityID,
		judicialdecisionsearch.MajorVersion,
	)
	if err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	request, err := f.materializer.MaterializeJudicialDecisionSearch(
		input,
		binding,
		budget,
	)
	if err != nil {
		return judicialdecisionsearch.Page{}, fmt.Errorf(
			"judicial-decision.search@1 request を組み立てられません: %w",
			err,
		)
	}
	if err := request.Validate(); err != nil {
		return judicialdecisionsearch.Page{}, fmt.Errorf(
			"judicial-decision.search@1 request が有効ではありません: %w",
			err,
		)
	}
	port, exists := f.routes.registry.JudicialDecisionSearch(
		binding.ProviderID(),
	)
	if !exists {
		return judicialdecisionsearch.Page{},
			fmt.Errorf("judicial-decision.search@1 port が構成されていません")
	}
	result, err := port.Search(ctx, request)
	if err != nil {
		return judicialdecisionsearch.Page{},
			legalQueryFacadeExecutedError(err)
	}
	if err := validateJudicialDecisionSearchFacadeResult(
		result,
		request,
		binding,
	); err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	return result, nil
}

// ReadJudicialDecision は、入力 ref が明示する provider の read を一度だけ呼び出す。
func (f JudicialCasesLegalQueryFacade) ReadJudicialDecision(
	ctx context.Context,
	input legalquery.JudicialDecisionReadIntentV1,
	budget legalquery.LegalQueryStepBudget,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	if err := f.validateCallContext(ctx); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	if err := input.Validate(); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf(
				"judicial decision read logical input が有効ではありません: %w",
				err,
			)
	}
	if err := validateLegalQueryFacadeReadBudget(budget); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	binding, err := f.judicialDecisionReadBinding(input.Ref())
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	request, err := f.materializer.MaterializeJudicialDecisionRead(
		input,
		binding,
		budget,
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf(
				"judicial-decision.read@1 request を組み立てられません: %w",
				err,
			)
	}
	if err := request.Validate(); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf(
				"judicial-decision.read@1 request が有効ではありません: %w",
				err,
			)
	}
	port, exists := f.routes.registry.JudicialDecisionRead(
		binding.ProviderID(),
	)
	if !exists {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			fmt.Errorf("judicial-decision.read@1 port が構成されていません")
	}
	result, err := port.Read(ctx, request)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{},
			legalQueryFacadeExecutedError(err)
	}
	if err := validateJudicialDecisionReadFacadeResult(
		result,
		request,
		binding,
	); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	return result, nil
}
