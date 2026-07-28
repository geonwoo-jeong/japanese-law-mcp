package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func addQueryLegalInformationTool(
	server *sdk.Server,
	querier legalquery.Port,
) {
	server.AddTool(
		&sdk.Tool{
			Name: "query_legal_information",
			Description: "日本語の法情報照会文から、利用可能な法令情報の取得方法を選び、" +
				"解釈ごとの型付き結果を返します。",
			InputSchema:  newQueryLegalInformationInputSchema(),
			OutputSchema: newQueryLegalInformationOutputSchema(),
		},
		func(
			ctx context.Context,
			request *sdk.CallToolRequest,
		) (*sdk.CallToolResult, error) {
			return callQueryLegalInformation(
				ctx,
				querier,
				request.Params.Arguments,
			)
		},
	)
}

func callQueryLegalInformation(
	ctx context.Context,
	querier legalquery.Port,
	arguments json.RawMessage,
) (*sdk.CallToolResult, error) {
	if isNilQueryLegalInformationPort(querier) {
		return errorToolResult(newInternalErrorResult())
	}
	request, err := decodeQueryLegalInformationInput(arguments)
	if err != nil {
		return errorToolResult(
			mapQueryLegalInformationInvalidArgument(err),
		)
	}
	result, err := querier.Query(ctx, request)
	if err != nil {
		return errorToolResult(mapQueryLegalInformationError(err))
	}
	if isNilLegalQueryResult(result) || result.Validate() != nil {
		return errorToolResult(newInternalErrorResult())
	}
	return successQueryLegalInformationToolResult(result)
}

func mapQueryLegalInformationInvalidArgument(
	err error,
) model.ErrorResult {
	var inputError queryLegalInformationInputError
	if errors.As(err, &inputError) {
		if result, buildErr := model.NewErrorResult(model.ErrorResultValues{
			Code: model.ErrorCodeInvalidArgument,
			Details: map[string]any{
				"field":  inputError.field,
				"reason": inputError.reason,
			},
		}); buildErr == nil {
			return result
		}
		return newInternalErrorResult()
	}
	if result, exists := legalQueryArgumentErrorResult(err); exists {
		return result
	}
	return newInternalErrorResult()
}

func mapQueryLegalInformationError(err error) model.ErrorResult {
	if result, exists := legalQueryArgumentErrorResult(err); exists {
		return result
	}
	var allFailed legalquery.LegalQueryAllFailedError
	if errors.As(err, &allFailed) && allFailed.Validate() == nil {
		result := allFailed.ErrorResult()
		if result.Validate() == nil {
			return result
		}
	}
	return newInternalErrorResult()
}

func legalQueryArgumentErrorResult(
	err error,
) (model.ErrorResult, bool) {
	var argumentError legalquery.ArgumentError
	if !errors.As(err, &argumentError) ||
		argumentError.Validate() != nil {
		return model.ErrorResult{}, false
	}
	result, buildErr := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeInvalidArgument,
		Details: map[string]any{
			"field":  argumentError.Field(),
			"reason": argumentError.Reason(),
		},
	})
	if buildErr != nil {
		return model.ErrorResult{}, false
	}
	return result, true
}

func successQueryLegalInformationToolResult(
	result legalquery.LegalQueryResult,
) (*sdk.CallToolResult, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return errorToolResult(newInternalErrorResult())
	}
	return &sdk.CallToolResult{
		Content: []sdk.Content{
			&sdk.TextContent{Text: string(payload)},
		},
		StructuredContent: json.RawMessage(payload),
		IsError:           false,
	}, nil
}

func isNilQueryLegalInformationPort(querier legalquery.Port) bool {
	return isNilQueryLegalInformationValue(querier)
}

func isNilLegalQueryResult(result legalquery.LegalQueryResult) bool {
	return isNilQueryLegalInformationValue(result)
}

func isNilQueryLegalInformationValue(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
