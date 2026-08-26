package judicialcitationtrace

import (
	"errors"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// OperationError は、原文を含まない公開可能な引用追跡エラーである。
type OperationError struct {
	result model.ErrorResult
}

func (e OperationError) Error() string             { return e.result.Message() }
func (e OperationError) Code() model.ErrorCode     { return e.result.Code() }
func (e OperationError) Retryable() bool           { return e.result.Retryable() }
func (e OperationError) Result() model.ErrorResult { return e.result }

func newOperationError(cause error) OperationError {
	return OperationError{result: safeErrorResult(cause)}
}

func safeErrorResult(cause error) model.ErrorResult {
	var sourceError model.SourceError
	if errors.As(cause, &sourceError) {
		if result, err := model.NewErrorResultFromSourceError(sourceError); err == nil {
			return result
		}
	}
	code := model.ErrorCodeInternalError
	if errors.Is(cause, judicialdecisionread.ErrNotFound) ||
		errors.Is(cause, judicialcasecitationextract.ErrNotFound) {
		code = model.ErrorCodeNotFound
	}
	result, err := model.NewErrorResult(model.ErrorResultValues{Code: code})
	if err != nil {
		panic(fmt.Sprintf("固定された公開エラーを構成できません: %v", err))
	}
	return result
}

func newIssueFromError(
	direction model.JudicialCitationIssueDirection,
	stage model.JudicialCitationIssueStage,
	cause error,
) (model.JudicialCitationIssue, error) {
	result := safeErrorResult(cause)
	return model.NewJudicialCitationIssue(model.JudicialCitationIssueValues{
		Direction: direction,
		Stage:     stage,
		Code:      string(result.Code()),
		Message:   result.Message(),
		Retryable: result.Retryable(),
	})
}

func newFixedIssue(
	direction model.JudicialCitationIssueDirection,
	stage model.JudicialCitationIssueStage,
	code, message string,
) (model.JudicialCitationIssue, error) {
	return model.NewJudicialCitationIssue(model.JudicialCitationIssueValues{
		Direction: direction,
		Stage:     stage,
		Code:      code,
		Message:   message,
		Retryable: false,
	})
}
