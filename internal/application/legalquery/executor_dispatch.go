package legalquery

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func (e Executor) executeStep(
	ctx context.Context,
	interpretationID string,
	step LegalQueryCandidateStep,
	budget LegalQueryStepBudget,
) (LegalQueryAttempt, error) {
	switch step.InputKind() {
	case InputKindLawSearch:
		return e.executeLawSearch(ctx, interpretationID, step, budget)
	case InputKindLawContentSearch:
		return e.executeLawContentSearch(ctx, interpretationID, step, budget)
	case InputKindLawRead:
		return e.executeLawDocumentRead(ctx, interpretationID, step, budget)
	case InputKindLawArticleRead:
		return e.executeLawArticleRead(ctx, interpretationID, step, budget)
	case InputKindLawUpdates:
		return e.executeLawUpdates(ctx, interpretationID, step, budget)
	case InputKindJudicialDecisionSearch:
		return e.executeJudicialSearch(ctx, interpretationID, step, budget)
	case InputKindJudicialDecisionRead:
		return e.executeJudicialRead(ctx, interpretationID, step, budget)
	default:
		return nil, fmt.Errorf(
			"step=%s の inputKind が定義されていません",
			step.StepID(),
		)
	}
}

func (e Executor) executeLawSearch(
	ctx context.Context,
	interpretationID string,
	step LegalQueryCandidateStep,
	budget LegalQueryStepBudget,
) (LegalQueryAttempt, error) {
	input, ok := step.LogicalInput().(LawSearchIntentV1)
	if !ok {
		return nil, executorInputVariantError(step)
	}
	page, err := e.core.SearchLaws(ctx, input, budget)
	if err != nil {
		return executorAttemptFromCallError(interpretationID, step, err)
	}
	if err := page.Validate(); err != nil {
		return nil, fmt.Errorf("law.search@1 の結果が有効ではありません: %w", err)
	}
	items, truncated, err := executorCollectionItems(page.Items(), budget)
	if err != nil {
		return nil, err
	}
	preview, err := newExecutorPagePreview(page.Page(), len(items), truncated)
	if err != nil {
		return nil, err
	}
	attempt, err := NewLegalQueryLawSearchAttempt(
		LegalQueryLawSearchAttemptValues{
			InterpretationID: interpretationID,
			Step:             step,
			Page:             preview,
			Items:            items,
		},
	)
	return attempt, err
}

func (e Executor) executeLawContentSearch(
	ctx context.Context,
	interpretationID string,
	step LegalQueryCandidateStep,
	budget LegalQueryStepBudget,
) (LegalQueryAttempt, error) {
	input, ok := step.LogicalInput().(LawContentSearchIntentV1)
	if !ok {
		return nil, executorInputVariantError(step)
	}
	page, err := e.core.SearchLawContent(ctx, input, budget)
	if err != nil {
		return executorAttemptFromCallError(interpretationID, step, err)
	}
	if err := page.Validate(); err != nil {
		return nil, fmt.Errorf(
			"law.content.search@1 の結果が有効ではありません: %w",
			err,
		)
	}
	items, truncated, err := executorCollectionItems(page.Items(), budget)
	if err != nil {
		return nil, err
	}
	preview, err := newExecutorPagePreview(page.Page(), len(items), truncated)
	if err != nil {
		return nil, err
	}
	attempt, err := NewLegalQueryLawContentSearchAttempt(
		LegalQueryLawContentSearchAttemptValues{
			InterpretationID: interpretationID,
			Step:             step,
			Page:             preview,
			Items:            items,
		},
	)
	return attempt, err
}

func (e Executor) executeLawDocumentRead(
	ctx context.Context,
	interpretationID string,
	step LegalQueryCandidateStep,
	budget LegalQueryStepBudget,
) (LegalQueryAttempt, error) {
	input, ok := step.LogicalInput().(LawReadIntentV1)
	if !ok {
		return nil, executorInputVariantError(step)
	}
	item, err := e.core.ReadLawDocument(ctx, input, budget)
	if err != nil {
		return executorAttemptFromCallError(interpretationID, step, err)
	}
	attempt, err := NewLegalQueryLawDocumentAttempt(
		LegalQueryLawDocumentAttemptValues{
			InterpretationID: interpretationID,
			Step:             step,
			Item:             item,
		},
	)
	return attempt, err
}

func (e Executor) executeLawArticleRead(
	ctx context.Context,
	interpretationID string,
	step LegalQueryCandidateStep,
	budget LegalQueryStepBudget,
) (LegalQueryAttempt, error) {
	input, ok := step.LogicalInput().(LawArticleReadIntentV1)
	if !ok {
		return nil, executorInputVariantError(step)
	}
	item, err := e.core.ReadLawArticle(ctx, input, budget)
	if err != nil {
		return executorAttemptFromCallError(interpretationID, step, err)
	}
	attempt, err := NewLegalQueryLawArticleAttempt(
		LegalQueryLawArticleAttemptValues{
			InterpretationID: interpretationID,
			Step:             step,
			Item:             item,
		},
	)
	return attempt, err
}

func (e Executor) executeLawUpdates(
	ctx context.Context,
	interpretationID string,
	step LegalQueryCandidateStep,
	budget LegalQueryStepBudget,
) (LegalQueryAttempt, error) {
	input, ok := step.LogicalInput().(LawUpdateListIntentV1)
	if !ok {
		return nil, executorInputVariantError(step)
	}
	page, err := e.core.ListLawUpdates(ctx, input, budget)
	if err != nil {
		return executorAttemptFromCallError(interpretationID, step, err)
	}
	if err := page.Validate(); err != nil {
		return nil, fmt.Errorf(
			"law.update.list@1 の結果が有効ではありません: %w",
			err,
		)
	}
	items, truncated, err := executorCollectionItems(page.Items(), budget)
	if err != nil {
		return nil, err
	}
	preview, err := newExecutorPagePreview(page.Page(), len(items), truncated)
	if err != nil {
		return nil, err
	}
	attempt, err := NewLegalQueryLawUpdatesAttempt(
		LegalQueryLawUpdatesAttemptValues{
			InterpretationID: interpretationID,
			Step:             step,
			Page:             preview,
			Items:            items,
		},
	)
	return attempt, err
}

func (e Executor) executeJudicialSearch(
	ctx context.Context,
	interpretationID string,
	step LegalQueryCandidateStep,
	budget LegalQueryStepBudget,
) (LegalQueryAttempt, error) {
	if e.judicial == nil {
		return nil, fmt.Errorf("judicial capability facade がありません")
	}
	input, ok := step.LogicalInput().(JudicialDecisionSearchIntentV1)
	if !ok {
		return nil, executorInputVariantError(step)
	}
	page, err := e.judicial.SearchJudicialDecisions(ctx, input, budget)
	if err != nil {
		return executorAttemptFromCallError(interpretationID, step, err)
	}
	if err := page.Validate(); err != nil {
		return nil, fmt.Errorf(
			"judicial-decision.search@1 の結果が有効ではありません: %w",
			err,
		)
	}
	items, truncated, err := executorCollectionItems(page.Items(), budget)
	if err != nil {
		return nil, err
	}
	preview, err := newExecutorPagePreview(page.Page(), len(items), truncated)
	if err != nil {
		return nil, err
	}
	attempt, err := NewLegalQueryJudicialSearchAttempt(
		LegalQueryJudicialSearchAttemptValues{
			InterpretationID: interpretationID,
			Step:             step,
			Page:             preview,
			Items:            items,
		},
	)
	return attempt, err
}

func (e Executor) executeJudicialRead(
	ctx context.Context,
	interpretationID string,
	step LegalQueryCandidateStep,
	budget LegalQueryStepBudget,
) (LegalQueryAttempt, error) {
	if e.judicial == nil {
		return nil, fmt.Errorf("judicial capability facade がありません")
	}
	input, ok := step.LogicalInput().(JudicialDecisionReadIntentV1)
	if !ok {
		return nil, executorInputVariantError(step)
	}
	item, err := e.judicial.ReadJudicialDecision(ctx, input, budget)
	if err != nil {
		return executorAttemptFromCallError(interpretationID, step, err)
	}
	attempt, err := NewLegalQueryJudicialDecisionAttempt(
		LegalQueryJudicialDecisionAttemptValues{
			InterpretationID: interpretationID,
			Step:             step,
			Item:             item,
		},
	)
	return attempt, err
}

func executorInputVariantError(step LegalQueryCandidateStep) error {
	return fmt.Errorf(
		"step=%s の inputKind と logical input が一致しません",
		step.StepID(),
	)
}

func executorCollectionItems[T any](
	items []T,
	budget LegalQueryStepBudget,
) ([]T, bool, error) {
	limit, exists := budget.EffectiveLimit()
	if !exists {
		return nil, false, fmt.Errorf(
			"collection step=%s に effectiveLimit がありません",
			budget.StepID(),
		)
	}
	visible := len(items)
	if visible > limit {
		visible = limit
	}
	cloned := make([]T, visible)
	copy(cloned, items[:visible])
	return cloned, visible < len(items), nil
}

func newExecutorPagePreview(
	source model.SourcePage,
	returnedCount int,
	truncated bool,
) (LegalQueryPagePreview, error) {
	if err := source.Validate(); err != nil {
		return LegalQueryPagePreview{}, fmt.Errorf(
			"source page が有効ではありません: %w",
			err,
		)
	}
	totalCount, hasTotal := source.TotalCount()
	totalRelation, hasRelation := source.TotalRelation()
	var total *int
	if hasTotal && hasRelation {
		if totalRelation == model.TotalRelationLowerBound &&
			totalCount < returnedCount {
			totalCount = returnedCount
		}
		total = &totalCount
	}
	var hasMore *bool
	_, hasNext := source.NextToken()
	switch {
	case hasTotal && totalRelation == model.TotalRelationExact:
		value := returnedCount < totalCount
		hasMore = &value
	case truncated || hasNext:
		value := true
		hasMore = &value
	case hasTotal &&
		totalRelation == model.TotalRelationLowerBound &&
		returnedCount < totalCount:
		value := true
		hasMore = &value
	}
	return NewLegalQueryPagePreview(LegalQueryPagePreviewValues{
		ReturnedCount: returnedCount,
		HasMore:       hasMore,
		TotalCount:    total,
		TotalRelation: totalRelation,
	})
}
