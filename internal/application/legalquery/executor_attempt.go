package legalquery

import (
	"context"
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func executorAttemptFromCallError(
	interpretationID string,
	step LegalQueryCandidateStep,
	callError error,
) (LegalQueryAttempt, error) {
	var executed ExecutedStepError
	if !errors.As(callError, &executed) {
		return nil, callError
	}
	if err := executed.Validate(); err != nil {
		return nil, fmt.Errorf("ExecutedStepError が有効ではありません: %w", err)
	}
	publicError, err := executorPublicError(executed.Unwrap())
	if err != nil {
		return nil, err
	}
	attempt, err := NewLegalQueryFailedAttempt(
		LegalQueryFailedAttemptValues{
			InterpretationID: interpretationID,
			Step:             step,
			Error:            publicError,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed attempt を作成できません: %w", err)
	}
	return attempt, nil
}

func executorPublicError(cause error) (model.ErrorResult, error) {
	switch {
	case errors.Is(cause, lawarticleread.ErrAmbiguousLocation):
		return model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeAmbiguousLocation,
		})
	case errors.Is(cause, context.DeadlineExceeded):
		return model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeSourceTimeout,
		})
	case errors.Is(cause, lawdocumentread.ErrNotFound),
		errors.Is(cause, lawarticleread.ErrNotFound),
		errors.Is(cause, judicialdecisionread.ErrNotFound):
		return model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeNotFound,
		})
	}
	var sourceError model.SourceError
	if errors.As(cause, &sourceError) {
		if sourceError.Code() ==
			model.SourceErrorCodeUnsupportedCapability ||
			sourceError.Code() ==
				model.SourceErrorCodeConfigurationRequired {
			return executorInternalErrorResult()
		}
		result, err := model.NewErrorResultFromSourceError(sourceError)
		if err == nil && isAllowedFailedAttemptError(result.Code()) {
			return result, nil
		}
		return executorInternalErrorResult()
	}
	return executorInternalErrorResult()
}

func executorInternalErrorResult() (model.ErrorResult, error) {
	result, err := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeInternalError,
	})
	if err != nil {
		return model.ErrorResult{}, fmt.Errorf(
			"internal_error の公開結果を作成できません: %w",
			err,
		)
	}
	return result, nil
}
